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
	"encoding/json"
	"hash/fnv"
	"slices"
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

// Library is the music library that live stations are scheduled from.
type Library interface {
	// Tracks is the pool for a query: tracks with a recording ID and a
	// scanned duration.
	Tracks(query string, limit int) []Title
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

// Sync updates the station rows to match the config and extends every live
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
	for _, s := range r.Live() {
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
		pool      []Title
		shows     []show
		showPools [][]Title
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
		f := fill{station: s, pool: pool}
		for _, c := range s.Shows {
			sh, err := newShow(c)
			if err != nil {
				log.Printf("radio: %s: %s\n", s.Name, err)
				continue
			}
			showPool := usable(lib.Tracks(sh.query(now), r.searchLimit()))
			need := min(sh.Length, r.minContent(s))
			if d := poolDuration(showPool); d == 0 || d < need {
				log.Printf("radio: skipping show %s on %s, only %s of music\n",
					sh.Name, s.Name, d.Round(time.Minute))
				continue
			}
			f.shows = append(f.shows, sh)
			f.showPools = append(f.showPools, showPool)
		}
		fills = append(fills, f)
	}

	onAir := make(map[string]bool)
	for _, f := range fills {
		onAir[f.Name] = true
	}
	changed, err := r.syncStations(r.wanted(onAir))
	if err != nil {
		return err
	}

	var keep []string
	for _, s := range r.Live() {
		keep = append(keep, s.Name)
	}
	if err := r.deleteOthers(keep); err != nil {
		return err
	}
	if err := r.deleteEnded(now.Add(-keepEnded)); err != nil {
		return err
	}
	for _, name := range changed {
		// new music or shows: reschedule after the track on now
		if err := r.deleteAfter(name, now); err != nil {
			return err
		}
	}

	until := now.Add(cfg.Horizon)
	for _, f := range fills {
		if !slices.Contains(keep, f.Name) {
			// the name is taken by another station
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
		for _, p := range append([][]Title{f.pool}, f.showPools...) {
			for _, t := range p {
				byRID[t.RID] = t
			}
		}
		var past []played
		for _, t := range history {
			title, ok := byRID[t.RID]
			if !ok {
				title = Title{RID: t.RID}
			}
			past = append(past, played{Title: title, Start: t.Start})
		}

		// the station's music, switching to a show's for its slot; the
		// track on when a slot starts or ends plays to its end
		list := slots(f.shows, start, until, loc)
		var tracks []RadioTrack
		for t := start; t.Before(until); {
			pool, name, end := f.pool, "", until
			if sl, on, next := at(list, t); on {
				pool, name = f.showPools[sl.Show], f.shows[sl.Show].Name
				if sl.End.Before(until) {
					end = sl.End
				}
			} else if !next.IsZero() && next.Before(until) {
				end = next
			}
			part := pick(pool, past, t, end, seed(f.Name+"/"+name, t))
			if len(part) == 0 {
				break
			}
			for i := range part {
				part[i].Station, part[i].Show = f.Name, name
				past = append(past, played{Title: byRID[part[i].RID], Start: part[i].Start})
			}
			tracks = append(tracks, part...)
			t = part[len(part)-1].End
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

// wanted returns the station rows the config describes, in order: every
// live station, on the air or not, then the streams. A name used twice
// keeps its first station.
func (r *Radio) wanted(onAir map[string]bool) []Station {
	var list []Station
	seen := make(map[string]bool)
	add := func(s Station) {
		if s.Name == "" {
			return
		}
		if seen[s.Name] {
			log.Printf("radio: station name %s is used twice\n", s.Name)
			return
		}
		seen[s.Name] = true
		list = append(list, s)
	}
	for i, s := range lineup(r.config) {
		add(Station{
			Name:        s.Name,
			Type:        TypeLive,
			Creator:     "Takeout",
			Image:       s.Image,
			Description: s.Description,
			Rank:        i,
			OnAir:       onAir[s.Name],
			Plan:        s.plan(),
		})
	}
	for i, v := range r.config.Radio.Streams {
		src, err := json.Marshal(v.Source)
		if err != nil {
			log.Println(err)
			continue
		}
		add(Station{
			Name:        v.Title,
			Type:        TypeStream,
			Creator:     v.Creator,
			Image:       v.Image,
			Description: v.Description,
			Source:      string(src),
			Rank:        i,
		})
	}
	return list
}

// syncStations makes the station rows match want, matched by name: new
// stations are created, changed ones updated in place so their IDs stay the
// same, and the rest deleted. It returns the live stations whose plan (music
// or shows) changed.
func (r *Radio) syncStations(want []Station) ([]string, error) {
	var changed []string
	existing := make(map[string]Station)
	for _, s := range r.stations() {
		existing[s.Name] = s
	}
	for _, w := range want {
		s, ok := existing[w.Name]
		delete(existing, w.Name)
		if !ok {
			if err := r.db.Create(&w).Error; err != nil {
				return nil, err
			}
			continue
		}
		if w.Type == TypeLive && s.Plan != w.Plan {
			changed = append(changed, w.Name)
		}
		if s.Type == w.Type && s.Creator == w.Creator && s.Image == w.Image &&
			s.Description == w.Description && s.Source == w.Source &&
			s.Rank == w.Rank && s.OnAir == w.OnAir && s.Plan == w.Plan {
			continue
		}
		w.Model = s.Model
		if err := r.db.Save(&w).Error; err != nil {
			return nil, err
		}
	}
	for _, s := range existing {
		if err := r.db.Unscoped().Delete(&s).Error; err != nil {
			return nil, err
		}
	}
	return changed, nil
}
