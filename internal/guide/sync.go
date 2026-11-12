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

package guide

import (
	"hash/fnv"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

const (
	// a channel needs this much distinct content to be scheduled
	minPool = 48 * time.Hour
	// how long ended programs are kept, for the day's guide
	keepEnded = 24 * time.Hour
	// added to a TMDB runtime when the file duration isn't known
	runtimeMargin = 2 * time.Minute
)

// Library finds the titles for a channel's queries.
type Library interface {
	Movies(query string, limit int) []Title
	Shows(query string, limit int) []Title
	// CollectionCounts counts, for each film collection, how many of the
	// movies in tmids belong to it.
	CollectionCounts(tmids []int64) map[string]int
	// Collection is every movie in the library from the named collection,
	// in release order.
	Collection(name string) []Title
}

// MediaLibrary is the Library and Resolver for a media collection.
type MediaLibrary struct {
	film   *film.Film
	tv     *tv.TV
	series map[int64]*TVSeries
}

func NewLibrary(f *film.Film, t *tv.TV) *MediaLibrary {
	return &MediaLibrary{film: f, tv: t, series: make(map[int64]*TVSeries)}
}

func (l *MediaLibrary) Movie(tmid int64) (Movie, bool) {
	m, err := l.film.LookupTMID(int(tmid))
	return m, err == nil
}

func (l *MediaLibrary) Episode(tvid int64, season, episode int) (TVSeries, TVEpisode, bool) {
	s, ok := l.series[tvid]
	if !ok {
		if v, err := l.tv.LookupTVID(int(tvid)); err == nil {
			s = &v
		}
		l.series[tvid] = s
	}
	if s == nil {
		return TVSeries{}, TVEpisode{}, false
	}
	e, err := l.tv.FindSeasonEpisode(*s, season, episode)
	if err != nil || e.Key == "" {
		return TVSeries{}, TVEpisode{}, false
	}
	return *s, e, true
}

// seconds is the scanned file duration, or the TMDB runtime with a margin.
func seconds(duration, runtime int) int {
	if duration > 0 {
		return duration
	}
	if runtime > 0 {
		return runtime*60 + int(runtimeMargin.Seconds())
	}
	return 0
}

func movieTitle(m Movie) Title {
	return Title{
		Kind:    ProgramMovie,
		TMID:    m.TMID,
		Seconds: seconds(m.Duration, m.Runtime),
		Vote:    m.VoteAverage,
		Date:    m.Date,
	}
}

func (l *MediaLibrary) Movies(query string, limit int) []Title {
	var titles []Title
	for _, m := range l.film.Search(query, limit) {
		titles = append(titles, movieTitle(m))
	}
	return titles
}

func (l *MediaLibrary) CollectionCounts(tmids []int64) map[string]int {
	return l.film.CollectionCounts(tmids)
}

func (l *MediaLibrary) Collection(name string) []Title {
	var titles []Title
	for _, m := range l.film.CollectionMovies(Collection{Name: name}) {
		titles = append(titles, movieTitle(m))
	}
	return titles
}

func (l *MediaLibrary) Shows(query string, limit int) []Title {
	var titles []Title
	for _, e := range l.tv.Search(query, limit) {
		titles = append(titles, Title{
			Kind:    ProgramEpisode,
			TVID:    e.TVID,
			Season:  e.Season,
			Episode: e.Episode,
			Seconds: seconds(e.Duration, e.Runtime),
			Vote:    e.VoteAverage,
			Date:    e.Date,
		})
	}
	return titles
}

func channelSettings(gc config.GuideConfig, cc config.LineupEntry) (align, minBreak time.Duration) {
	align, minBreak = gc.Align, gc.MinBreak
	if cc.Align > 0 {
		align = cc.Align
	}
	if cc.MinBreak > 0 {
		minBreak = cc.MinBreak
	}
	if align <= 0 {
		align = 5 * time.Minute
	}
	return
}

func seed(name string, t time.Time) uint64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	return h.Sum64() ^ uint64(t.Unix())
}

// Sync extends every lineup channel's schedule to now + Horizon and removes
// programs that ended more than a day ago. Programs already scheduled never
// change, so it's safe to run at any time and catches up after downtime.
func (g *Guide) Sync(now time.Time, lib Library) error {
	cfg := g.config.Guide
	if len(cfg.Lineup) == 0 {
		return nil
	}
	loc := cfg.Location()
	if cfg.Timezone != "" && loc == time.Local {
		log.Printf("channels: unknown time zone %q, using local time\n", cfg.Timezone)
	}

	if err := g.deleteEnded(now.Add(-keepEnded)); err != nil {
		return err
	}

	busy := busyTimes{}
	for _, p := range g.airing(now) {
		busy.add(p, p.End)
	}

	until := now.Add(cfg.Horizon)
	type channelFill struct {
		cc      config.LineupEntry
		ch      Channel
		plan    fillPlan
		pool    []Title
		history []Program
	}
	var fills []channelFill
	for _, cc := range cfg.Lineup {
		ch, err := g.ensureChannel(cc.Name)
		if err != nil {
			return err
		}
		align, minBreak := channelSettings(cfg, cc)
		history := g.history(ch.ID)

		start := floorAlign(now, align, loc)
		if n := len(history); n > 0 {
			if next := nextStart(history[n-1].End, align, minBreak, loc); next.After(start) {
				start = next
			}
		}
		if !start.Before(until) {
			continue
		}

		var pool []Title
		if cc.Movies != "" {
			pool = append(pool, lib.Movies(cc.Movies, cfg.SearchLimit)...)
		}
		if cc.Shows != "" {
			pool = append(pool, lib.Shows(cc.Shows, cfg.SearchLimit)...)
		}
		if d := poolDuration(pool); d < minPool {
			log.Printf("channels: skipping %s, only %s of content\n", cc.Name, d.Round(time.Minute))
			continue
		}

		plan := fillPlan{
			Start:    start,
			Until:    until,
			Align:    align,
			MinBreak: minBreak,
			Location: loc,
			Seed:     seed(cc.Name, start),
		}
		if cc.Layout != "" {
			layout, match := cc.Layout, cc.Match
			plan.OnAir = func(day time.Time) bool {
				return date.MatchTime(layout, match, day)
			}
		}
		fills = append(fills, channelFill{cc, ch, plan, pool, history})
	}

	// Specials first, on every channel, so regular programming everywhere
	// steers clear of their titles.
	for i := range fills {
		f := &fills[i]
		blocks, err := g.specialBlocks(f.ch, f.cc, f.pool, lib, f.plan.Start, until,
			f.plan.Align, f.plan.MinBreak, loc)
		if err != nil {
			return err
		}
		f.plan.Blocks = blocks
		for _, b := range blocks {
			for _, p := range b.Programs {
				if !p.Start.Before(f.plan.Start) {
					busy.add(p, nextStart(p.End, f.plan.Align, f.plan.MinBreak, loc))
				}
			}
		}
	}

	for _, f := range fills {
		programs := fill(f.plan, f.pool, f.history, busy)
		for i := range programs {
			programs[i].ChannelID = f.ch.ID
		}
		if err := g.createPrograms(programs); err != nil {
			return err
		}
		if len(programs) > 0 {
			log.Printf("channels: %s scheduled %d programs to %s\n", f.cc.Name, len(programs),
				programs[len(programs)-1].Start.In(loc).Format(time.RFC3339))
		}
	}
	return nil
}
