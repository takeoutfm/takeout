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
	"fmt"
	"math/rand/v2"
	"sort"
	"time"

	. "takeoutfm.dev/takeout/model"
)

// Title is a movie or episode that can be scheduled.
type Title struct {
	Kind    string // ProgramMovie or ProgramEpisode
	TMID    int64
	TVID    int64
	Season  int
	Episode int
	Seconds int     // playing time
	Vote    float32 // 0-10
}

func (t Title) duration() time.Duration {
	return time.Duration(t.Seconds) * time.Second
}

// key identifies the title itself.
func (t Title) key() string {
	if t.Kind == ProgramEpisode {
		return fmt.Sprintf("e:%d:%d:%d", t.TVID, t.Season, t.Episode)
	}
	return fmt.Sprintf("m:%d", t.TMID)
}

// unit is what the scheduler picks between: a movie or a whole series.
func (t Title) unit() string {
	if t.Kind == ProgramEpisode {
		return fmt.Sprintf("s:%d", t.TVID)
	}
	return fmt.Sprintf("m:%d", t.TMID)
}

func programTitle(p Program) Title {
	return Title{Kind: p.Kind, TMID: p.TMID, TVID: p.TVID, Season: p.Season, Episode: p.Episode}
}

func (t Title) program(start time.Time) Program {
	return Program{
		Start:   start.UTC(),
		End:     start.Add(t.duration()).UTC(),
		Kind:    t.Kind,
		TMID:    t.TMID,
		TVID:    t.TVID,
		Season:  t.Season,
		Episode: t.Episode,
	}
}

type interval struct {
	start, end time.Time
}

// busyTimes records when titles are on the air, so the same title isn't scheduled
// on two channels at once.
type busyTimes map[string][]interval

// add marks a program's title busy until next, when its slot ends.
func (b busyTimes) add(p Program, next time.Time) {
	k := programTitle(p).key()
	b[k] = append(b[k], interval{p.Start, next})
}

func (b busyTimes) overlaps(key string, start, end time.Time) bool {
	for _, i := range b[key] {
		if start.Before(i.end) && i.start.Before(end) {
			return true
		}
	}
	return false
}

// fillPlan describes the part of a channel's schedule to fill.
type fillPlan struct {
	Start    time.Time // the first program starts here; should be aligned
	Until    time.Time // no programs start at or after this
	Align    time.Duration
	MinBreak time.Duration
	Location *time.Location
	OnAir    func(day time.Time) bool // seasonal window; nil is always
	Seed     uint64
}

func (p fillPlan) loc() *time.Location {
	if p.Location == nil {
		return time.UTC
	}
	return p.Location
}

func midnight(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
}

// ceilAlign returns the first multiple of align since local midnight at or
// after t.
func ceilAlign(t time.Time, align time.Duration, loc *time.Location) time.Time {
	if align <= 0 {
		return t.UTC()
	}
	m := midnight(t, loc)
	k := (t.Sub(m) + align - 1) / align
	return m.Add(k * align).UTC()
}

// floorAlign returns the last multiple of align since local midnight at or
// before t.
func floorAlign(t time.Time, align time.Duration, loc *time.Location) time.Time {
	if align <= 0 {
		return t.UTC()
	}
	m := midnight(t, loc)
	k := t.Sub(m) / align
	return m.Add(k * align).UTC()
}

// nextStart is when the program after one ending at end starts: the next
// align mark that leaves at least minBreak.
func nextStart(end time.Time, align, minBreak time.Duration, loc *time.Location) time.Time {
	return ceilAlign(end.Add(minBreak), align, loc)
}

func nextMidnight(t time.Time, loc *time.Location) time.Time {
	m := midnight(t, loc)
	l := m.AddDate(0, 0, 1)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc).UTC()
}

// poolDuration is the total playing time of all titles in the pool.
func poolDuration(pool []Title) time.Duration {
	var d time.Duration
	for _, t := range pool {
		d += t.duration()
	}
	return d
}

type unit struct {
	key      string
	movie    bool
	titles   []Title // one movie, or a series' episodes in order
	vote     float32
	next     int // index of the next title (series)
	lastAir  time.Time
	hasAired bool
}

func (u *unit) title() Title {
	return u.titles[u.next]
}

// buildUnits groups the pool into movies and series, sorted for determinism.
func buildUnits(pool []Title) []*unit {
	byKey := make(map[string]*unit)
	var units []*unit
	seen := make(map[string]bool)
	for _, t := range pool {
		if t.Seconds <= 0 || seen[t.key()] {
			continue
		}
		if t.Kind == ProgramEpisode && t.Season == 0 {
			// specials are out of order
			continue
		}
		seen[t.key()] = true
		k := t.unit()
		u, ok := byKey[k]
		if !ok {
			u = &unit{key: k, movie: t.Kind != ProgramEpisode}
			byKey[k] = u
			units = append(units, u)
		}
		u.titles = append(u.titles, t)
	}
	for _, u := range units {
		sort.Slice(u.titles, func(i, j int) bool {
			a, b := u.titles[i], u.titles[j]
			if a.Season != b.Season {
				return a.Season < b.Season
			}
			return a.Episode < b.Episode
		})
		var sum float32
		for _, t := range u.titles {
			sum += t.Vote
		}
		u.vote = sum / float32(len(u.titles))
	}
	sort.Slice(units, func(i, j int) bool { return units[i].key < units[j].key })
	return units
}

// aired records that title t of u started at start.
func (u *unit) aired(t Title, start time.Time) {
	if !u.hasAired || start.After(u.lastAir) {
		u.lastAir = start
		u.hasAired = true
	}
	if u.movie {
		return
	}
	// continue after this episode, or after where it would be
	u.next = len(u.titles)
	for i, e := range u.titles {
		if e.Season > t.Season || (e.Season == t.Season && e.Episode > t.Episode) {
			u.next = i
			break
		}
	}
	if u.next >= len(u.titles) {
		u.next = 0
	}
}

const (
	minMovieCooldown = 24 * time.Hour
	maxMovieCooldown = 7 * 24 * time.Hour
)

// fill schedules programs from plan.Start until plan.Until. History is the
// channel's earlier programs, in start order, used to continue series and
// avoid recent repeats. busy holds what other channels are airing and is
// updated with the new programs. The result depends only on the arguments.
func fill(plan fillPlan, pool []Title, history []Program, busy busyTimes) []Program {
	units := buildUnits(pool)
	if len(units) == 0 {
		return nil
	}
	loc := plan.loc()
	align := plan.Align
	if align <= 0 {
		align = time.Minute
	}

	byKey := make(map[string]*unit)
	var movieTime time.Duration
	var series int
	var episodeTime time.Duration
	var episodes int
	for _, u := range units {
		byKey[u.key] = u
		if u.movie {
			movieTime += u.titles[0].duration()
		} else {
			series++
			for _, t := range u.titles {
				episodeTime += t.duration()
				episodes++
			}
		}
	}

	// a movie doesn't repeat for about half the time it takes to show them all
	cooldown := min(max(movieTime/2, minMovieCooldown), maxMovieCooldown)
	// series take turns
	spacing := time.Hour
	if episodes > 0 {
		spacing = max(time.Duration(series)*(episodeTime/time.Duration(episodes)), time.Hour)
	}

	prev := ""
	for _, p := range history {
		t := programTitle(p)
		if u, ok := byKey[t.unit()]; ok {
			u.aired(t, p.Start)
		}
		prev = t.unit()
	}

	rng := rand.New(rand.NewPCG(plan.Seed, 0x6368616e6e656c))

	var out []Program
	t := plan.Start
	for i := 0; t.Before(plan.Until) && i < 100000; i++ {
		if plan.OnAir != nil && !plan.OnAir(t.In(loc)) {
			t = nextMidnight(t, loc)
			continue
		}
		u := pick(units, t, prev, busy, rng, cooldown, spacing, align, plan.MinBreak, loc)
		if u == nil {
			// nothing can air now; try the next mark
			t = t.Add(align)
			continue
		}
		title := u.title()
		p := title.program(t)
		next := nextStart(p.End, align, plan.MinBreak, loc)
		out = append(out, p)
		busy.add(p, next)
		u.aired(title, t)
		prev = u.key
		t = next
	}
	return out
}

type candidate struct {
	u      *unit
	weight float64
}

func pick(units []*unit, t time.Time, prev string, busy busyTimes, rng *rand.Rand,
	cooldown, spacing, align, minBreak time.Duration, loc *time.Location) *unit {

	var candidates []candidate
	var total float64
	var fallback *unit
	for _, u := range units {
		title := u.title()
		end := t.Add(title.duration())
		next := nextStart(end, align, minBreak, loc)
		if busy.overlaps(title.key(), t, next) {
			continue
		}

		since := time.Duration(1 << 62)
		if u.hasAired {
			since = t.Sub(u.lastAir)
		}
		if fallback == nil || (fallback.hasAired && (!u.hasAired || u.lastAir.Before(fallback.lastAir))) {
			fallback = u
		}

		// better rated titles more often
		w := 0.5 + float64(u.vote)/10
		if u.movie {
			if since < cooldown {
				continue
			}
		} else {
			if u.key == prev && len(units) > 1 {
				continue
			}
			w *= min(max(float64(since)/float64(spacing), 0.05), 1)
		}

		// mildly prefer small gaps and returning to a round time
		gap := next.Sub(end) - minBreak
		w *= 1 + 0.25*(1-min(float64(gap)/float64(align), 1))
		if next.In(loc).Minute()%30 == 0 {
			w *= 1.25
		}

		candidates = append(candidates, candidate{u, w})
		total += w
	}
	if len(candidates) == 0 {
		return fallback
	}
	r := rng.Float64() * total
	for _, c := range candidates {
		r -= c.weight
		if r < 0 {
			return c.u
		}
	}
	return candidates[len(candidates)-1].u
}
