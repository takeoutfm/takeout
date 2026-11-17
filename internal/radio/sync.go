// Copyright 2026 defsub
//
// This file is part of TakeoutFM.
//
// TakeoutFM is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// TakeoutFM is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with TakeoutFM.  If not, see <https://www.gnu.org/licenses/>.

package radio

import (
	"hash/fnv"
	"sync"
	"time"

	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

const (
	// how long ended tracks are kept
	keepEnded = time.Hour
	// a schedule request extends schedules ending sooner than this
	extendLead = 2 * time.Hour
)

// Library is the music library: station pools and the live station rows
// that /api/radio lists.
type Library interface {
	// Tracks is the pool for a query: tracks with a recording ID and a
	// scanned duration.
	Tracks(query string, limit int) []Title
	// TakeoutStations is the stations Takeout creates: live, stream and
	// older generated ones.
	TakeoutStations() []Station
	CreateStation(s *Station) error
	UpdateStation(s *Station) error
	DeleteStation(s *Station) error
}

// MusicLibrary is the Library for a media collection.
type MusicLibrary struct {
	*music.Music
}

func NewLibrary(m *music.Music) MusicLibrary {
	return MusicLibrary{m}
}

func (l MusicLibrary) Tracks(query string, limit int) []Title {
	var titles []Title
	for _, t := range l.Search(query, limit) {
		if t.RID == "" || t.Duration <= 0 {
			continue
		}
		release := t.RGID
		if release == "" {
			release = t.Release
		}
		titles = append(titles, Title{RID: t.RID, Artist: t.Artist, Release: release, Seconds: t.Duration})
	}
	return titles
}

func seed(name string, t time.Time) uint64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	return h.Sum64() ^ uint64(t.Unix())
}

// syncMu keeps the radio job and a schedule request from extending
// schedules at once in this process.
var syncMu sync.Mutex

func (r *Radio) searchLimit() int {
	return r.config.Radio.SearchLimit
}

func (r *Radio) minContent(s station) time.Duration {
	if s.MinContent > 0 {
		return s.MinContent
	}
	return r.config.Radio.MinContent
}

// Sync updates the live station rows to match the config and extends every
// station's schedule to now + Horizon. Scheduled tracks never change, so
// it's safe to run at any time; after downtime a schedule restarts at now.
func (r *Radio) Sync(now time.Time, lib Library) error {
	syncMu.Lock()
	defer syncMu.Unlock()
	return r.sync(now, lib)
}

// ExtendDue extends the schedules if any live station's ends within
// extendLead, so listeners never run out when the job is late. The schedule
// API calls it before answering.
func (r *Radio) ExtendDue(now time.Time, lib Library) error {
	due := false
	for _, s := range lib.TakeoutStations() {
		if s.Type != music.TypeLiveStation {
			continue
		}
		if end, ok := r.lastEnd(s.Name); !ok || end.Before(now.Add(extendLead)) {
			due = true
			break
		}
	}
	if !due {
		return nil
	}
	return r.Sync(now, lib)
}

func (r *Radio) sync(now time.Time, lib Library) error {
	cfg := r.config.Radio
	loc := cfg.Location()
	if cfg.Timezone != "" && loc == time.Local {
		log.Printf("radio: unknown time zone %q, using local time\n", cfg.Timezone)
	}

	type fill struct {
		station
		pool []Title
	}
	var fills []fill
	for _, s := range lineup(r.config) {
		if !s.onAir(now.In(loc)) {
			continue
		}
		pool := usable(lib.Tracks(s.Query, r.searchLimit()))
		if d := poolDuration(pool); d == 0 || d < r.minContent(s) {
			log.Printf("radio: skipping %s, only %s of music\n", s.Name, d.Round(time.Minute))
			continue
		}
		fills = append(fills, fill{s, pool})
	}

	var active []station
	for _, f := range fills {
		active = append(active, f.station)
	}
	listed, err := syncStations(active, lib)
	if err != nil {
		return err
	}

	var keep []string
	for name := range listed {
		keep = append(keep, name)
	}
	if err := r.deleteOthers(keep); err != nil {
		return err
	}
	if err := r.deleteEnded(now.Add(-keepEnded)); err != nil {
		return err
	}

	until := now.Add(cfg.Horizon)
	for _, f := range fills {
		if !listed[f.Name] {
			// no station row, so nobody could tune in
			continue
		}
		history := r.history(f.Name)
		start := now
		if n := len(history); n > 0 && history[n-1].End.After(start) {
			start = history[n-1].End
		}
		if !start.Before(until) {
			continue
		}
		byRID := make(map[string]Title)
		for _, t := range f.pool {
			byRID[t.RID] = t
		}
		var past []played
		for _, t := range history {
			title, ok := byRID[t.RID]
			if !ok {
				title = Title{RID: t.RID}
			}
			past = append(past, played{Title: title, Start: t.Start})
		}
		tracks := pick(f.pool, past, start, until, seed(f.Name, start))
		for i := range tracks {
			tracks[i].Station = f.Name
		}
		if err := r.createTracks(tracks); err != nil {
			return err
		}
		if len(tracks) > 0 {
			log.Printf("radio: %s scheduled %d tracks to %s\n", f.Name, len(tracks),
				tracks[len(tracks)-1].End.In(loc).Format(time.RFC3339))
		}
	}
	return nil
}

// syncStations makes the live station rows match active, in order: new
// stations are created, existing ones updated in place so their IDs stay
// the same, and ones no longer active deleted. An older generated station
// (genre, decade, series or other) with a live station's name becomes that
// live station, keeping its ID; the rest are deleted. Stream stations are
// left alone. It returns the names of the active stations that have a row.
func syncStations(active []station, lib Library) (map[string]bool, error) {
	listed := make(map[string]bool)
	existing := make(map[string]Station)
	for _, s := range lib.TakeoutStations() {
		switch s.Type {
		case music.TypeStream:
		case music.TypeLiveStation:
			existing[s.Name] = s
		default:
			// generated stations from older releases
			if err := lib.DeleteStation(&s); err != nil {
				return nil, err
			}
		}
	}
	for i, a := range active {
		s, ok := existing[a.Name]
		delete(existing, a.Name)
		if ok {
			listed[a.Name] = true
			if s.Rank == i && s.Image == a.Image && s.Description == a.Description {
				continue
			}
			s.Rank, s.Image, s.Description = i, a.Image, a.Description
			if err := lib.UpdateStation(&s); err != nil {
				return nil, err
			}
			continue
		}
		s = Station{
			User:        music.TakeoutUser,
			Shared:      true,
			Type:        music.TypeLiveStation,
			Name:        a.Name,
			Creator:     "Takeout",
			Ref:         a.Name,
			Image:       a.Image,
			Description: a.Description,
			Rank:        i,
		}
		if err := lib.CreateStation(&s); err != nil {
			// for example a stream station with the same name
			log.Printf("radio: can't create station %s: %s\n", a.Name, err)
			continue
		}
		listed[a.Name] = true
	}
	for _, s := range existing {
		if err := lib.DeleteStation(&s); err != nil {
			return nil, err
		}
	}
	return listed, nil
}
