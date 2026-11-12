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
	"math/rand/v2"
	"slices"
	"sort"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

const (
	defaultSpecialAt        = "20:00"
	defaultSpecialMaxLength = 12 * time.Hour
	defaultMinFilms         = 3
	// the same special doesn't run on two channels this close together
	specialApart = 7 * 24 * time.Hour
	// a special starts on an evening up to this many days after it's due
	specialJitterDays = 3
	// fewer programs than this isn't much of an event
	minSpecialPrograms = 2
)

// special is a themed set of titles that can air as one block.
type special struct {
	name   string
	titles []Title
	byDate bool // keep release order; otherwise shuffled when laid out
}

// specials returns a channel's possible specials: its themes, then the film
// collections with at least MinFilms movies in the channel's pool.
func specials(cc config.LineupEntry, pool []Title, lib Library, limit int) []special {
	sc := cc.Specials
	var list []special
	for _, th := range sc.Themes {
		var titles []Title
		if th.Movies != "" {
			titles = append(titles, lib.Movies(th.Movies, limit)...)
		}
		if th.Shows != "" {
			titles = append(titles, lib.Shows(th.Shows, limit)...)
		}
		list = append(list, special{name: th.Name, titles: titles, byDate: th.Order == "date"})
	}
	if sc.Collections {
		minFilms := sc.MinFilms
		if minFilms <= 0 {
			minFilms = defaultMinFilms
		}
		var tmids []int64
		for _, t := range pool {
			if t.Kind == ProgramMovie {
				tmids = append(tmids, t.TMID)
			}
		}
		counts := lib.CollectionCounts(tmids)
		var names []string
		for name, n := range counts {
			if n >= minFilms && !slices.Contains(sc.Skip, name) {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			// every film owned from the collection, even ones the channel's
			// query misses: a marathon missing one would feel broken
			list = append(list, special{name: name, titles: lib.Collection(name), byDate: true})
		}
	}
	return list
}

// order is the special's titles in airing order: release order, or units
// (movies and whole series) shuffled with episodes kept in order.
func (s special) order(seed uint64) []Title {
	titles := slices.Clone(s.titles)
	if s.byDate {
		sort.SliceStable(titles, func(i, j int) bool { return titles[i].Date.Before(titles[j].Date) })
		return titles
	}
	units := buildUnits(titles)
	rng := rand.New(rand.NewPCG(seed, 0x7370656369616c))
	rng.Shuffle(len(units), func(i, j int) { units[i], units[j] = units[j], units[i] })
	var out []Title
	for _, u := range units {
		out = append(out, u.titles...)
	}
	return out
}

// layout places the special's titles back to back from start, stopping after
// the last whole program that ends within maxLength. end is when the program
// after the special can start.
func (s special) layout(start time.Time, maxLength, align, minBreak time.Duration,
	loc *time.Location) (programs []Program, end time.Time) {

	limit := start.Add(maxLength)
	t := start
	seen := make(map[string]bool)
	for _, title := range s.order(seed(s.name, start)) {
		if title.Seconds <= 0 || seen[title.key()] {
			continue
		}
		seen[title.key()] = true
		p := title.program(t)
		if p.End.After(limit) {
			break
		}
		p.Block = s.name
		programs = append(programs, p)
		t = nextStart(p.End, align, minBreak, loc)
	}
	return programs, t
}

// specialStart parses "15:04", falling back to 20:00.
func specialStart(at string) time.Duration {
	if at == "" {
		at = defaultSpecialAt
	}
	t, err := time.Parse("15:04", at)
	if err != nil {
		log.Printf("channels: bad special time %q, using %s\n", at, defaultSpecialAt)
		t, _ = time.Parse("15:04", defaultSpecialAt)
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
}

// specialBlocks returns the specials to fill around from start on: ones
// already planned, and a new one if the channel is due. A newly planned
// special is recorded, so later runs keep it where it is.
func (g *Guide) specialBlocks(ch Channel, cc config.LineupEntry, pool []Title, lib Library,
	start, until time.Time, align, minBreak time.Duration, loc *time.Location) ([]block, error) {

	sc := cc.Specials
	if sc.Every <= 0 && len(sc.Themes) == 0 && !sc.Collections {
		return nil, nil
	}
	maxLength := sc.MaxLength
	if maxLength <= 0 {
		maxLength = defaultSpecialMaxLength
	}
	list := specials(cc, pool, lib, g.config.Guide.SearchLimit)
	byName := make(map[string]special)
	for _, s := range list {
		byName[s.name] = s
	}

	airings := g.specialAirings(ch.ID)
	var blocks []block
	planned := false
	for _, a := range airings {
		if !a.End.After(start) {
			continue
		}
		if !a.Start.Before(start) {
			planned = true
		}
		s, ok := byName[a.Name]
		if !ok {
			// gone from the config or the library: regular programming instead
			continue
		}
		programs, end := s.layout(a.Start, maxLength, align, minBreak, loc)
		if len(programs) > 0 {
			blocks = append(blocks, block{Start: a.Start, End: end, Programs: programs})
		}
	}
	if sc.Every <= 0 || planned || len(list) == 0 {
		return blocks, nil
	}

	// due once Every has passed since the last one; the first comes right away
	due := start
	lastAired := make(map[string]time.Time)
	for _, a := range airings {
		if next := a.Start.Add(sc.Every); next.After(due) {
			due = next
		}
		lastAired[a.Name] = a.Start
	}
	if !due.Before(until) {
		return blocks, nil
	}

	// an evening up to a few days after it's due, so it isn't predictable
	rng := rand.New(rand.NewPCG(seed(ch.Name, due), 0x65766572))
	at := midnight(due, loc).AddDate(0, 0, rng.IntN(specialJitterDays+1)).Add(specialStart(sc.At)).UTC()
	for at.Before(start) {
		at = at.AddDate(0, 0, 1)
	}
	// a feature's evening is taken; try the next
	for i := 0; i < 7 && overlapsFeature(cc, at, at.Add(maxLength), loc); i++ {
		at = at.AddDate(0, 0, 1)
	}

	// the special that has gone longest without airing here
	candidates := slices.Clone(list)
	rng.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
	sort.SliceStable(candidates, func(i, j int) bool {
		return lastAired[candidates[i].name].Before(lastAired[candidates[j].name])
	})
	for _, s := range candidates {
		if g.specialNear(s.name, ch.ID, at, specialApart) {
			continue
		}
		programs, end := s.layout(at, maxLength, align, minBreak, loc)
		if len(programs) < minSpecialPrograms {
			continue
		}
		a := SpecialAiring{ChannelID: ch.ID, Name: s.name, Start: at, End: end}
		if err := g.createSpecialAiring(&a); err != nil {
			return blocks, err
		}
		log.Printf("channels: %s special %q at %s\n", cc.Name, s.name, at.In(loc).Format(time.RFC3339))
		return append(blocks, block{Start: at, End: end, Programs: programs}), nil
	}
	return blocks, nil
}
