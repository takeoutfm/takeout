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
	"time"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

const (
	// a feature's programs are chosen this long before it airs
	featureLead = 24 * time.Hour
	// leftover slot time shorter than this is a break, longer is regular programming
	featureLeftover = 30 * time.Minute
)

// slot is when a feature airs.
type slot struct {
	feature    config.FeatureConfig
	start, end time.Time
}

// featureSlots returns the channel's feature slots that start in [from, until),
// in start order. When features overlap, the first listed wins.
func featureSlots(cc config.LineupEntry, from, until time.Time, loc *time.Location) []slot {
	var slots []slot
	if len(cc.Features) == 0 {
		return nil
	}
	for day := midnight(from, loc); day.Before(until); day = nextMidnight(day, loc).In(loc) {
		var today []slot
		for _, f := range cc.Features {
			if f.Length <= 0 {
				continue
			}
			if f.Layout != "" && !date.MatchTime(f.Layout, f.Match, day) {
				continue
			}
			start := day.Add(specialStart(f.At)).UTC()
			s := slot{feature: f, start: start, end: start.Add(f.Length)}
			clash := false
			for _, o := range today {
				if s.start.Before(o.end) && o.start.Before(s.end) {
					clash = true
				}
			}
			if !clash {
				today = append(today, s)
			}
		}
		// in start order
		for i := 1; i < len(today); i++ {
			for j := i; j > 0 && today[j].start.Before(today[j-1].start); j-- {
				today[j], today[j-1] = today[j-1], today[j]
			}
		}
		// overlaps are settled over the whole day, then the range applies
		for _, s := range today {
			if !s.start.Before(from) && s.start.Before(until) {
				slots = append(slots, s)
			}
		}
	}
	return slots
}

// overlapsFeature reports whether [start, end) overlaps one of the channel's
// feature slots. Features are fixed, so specials move out of their way.
func overlapsFeature(cc config.LineupEntry, start, end time.Time, loc *time.Location) bool {
	for _, s := range featureSlots(cc, start.Add(-48*time.Hour), end, loc) {
		if start.Before(s.end) && s.start.Before(end) {
			return true
		}
	}
	return false
}

// featureBlocks reserves the feature slots from start on, as single reserved
// programs that regular programming fills around.
func featureBlocks(cc config.LineupEntry, start, until time.Time, align time.Duration,
	loc *time.Location) []block {

	var blocks []block
	for _, s := range featureSlots(cc, start, until, loc) {
		p := Program{
			Start: s.start,
			End:   s.end,
			Kind:  ProgramReserved,
			Block: s.feature.Name,
		}
		blocks = append(blocks, block{Start: s.start, End: ceilAlign(s.end, align, loc), Programs: []Program{p}})
	}
	return blocks
}

// featureTitles is what a feature can show: its source or its queries.
func featureTitles(f config.FeatureConfig, lib Library, limit int) []Title {
	if f.Source != "" {
		return lib.RecentMovies(f.Source)
	}
	var titles []Title
	if f.Movies != "" {
		titles = append(titles, lib.Movies(f.Movies, limit)...)
	}
	if f.Shows != "" {
		titles = append(titles, lib.Shows(f.Shows, limit)...)
	}
	return titles
}

// fillFeatures chooses the programs for the channel's reserved slots that air
// within featureLead of now: titles the feature hasn't shown that fit, then
// regular programming for a long leftover. With nothing new, the slot is all
// regular programming, without the feature's name.
func (g *Guide) fillFeatures(now time.Time, ch Channel, cc config.LineupEntry, pool []Title,
	lib Library, busy busyTimes, align, minBreak time.Duration, loc *time.Location) error {

	for _, r := range g.reservedBefore(ch.ID, now.Add(featureLead)) {
		var f config.FeatureConfig
		for _, x := range cc.Features {
			if x.Name == r.Block {
				f = x
			}
		}

		var programs []Program
		var shown []FeatureTitle
		t := r.Start
		if f.Name != "" {
			aired := g.featureKeys(ch.ID, f.Name)
			candidates := featureTitles(f, lib, g.config.Guide.SearchLimit)
			if f.Source == "" {
				// a query has no newest first; vary it
				rng := rand.New(rand.NewPCG(seed(f.Name, r.Start), 0x66656174))
				rng.Shuffle(len(candidates), func(i, j int) {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				})
			}
			for _, c := range candidates {
				if c.Seconds <= 0 || aired[c.key()] {
					continue
				}
				p := c.program(t)
				next := nextStart(p.End, align, minBreak, loc)
				if p.End.After(r.End) || busy.overlaps(c.key(), t, next) {
					// too long for what's left, or on elsewhere; try the next
					continue
				}
				p.Block = f.Name
				p.ChannelID = ch.ID
				programs = append(programs, p)
				shown = append(shown, FeatureTitle{ChannelID: ch.ID, Feature: f.Name, Key: c.key(), Start: p.Start})
				aired[c.key()] = true
				busy.add(p, next)
				t = next
			}
		}
		if len(programs) == 0 {
			log.Printf("channels: %s has nothing new for %q at %s\n", cc.Name, r.Block,
				r.Start.In(loc).Format(time.RFC3339))
		}

		// the rest of the slot
		if r.End.Sub(t) >= featureLeftover && len(pool) > 0 {
			plan := fillPlan{
				Start:    t,
				Until:    r.End,
				Stop:     r.End,
				Align:    align,
				MinBreak: minBreak,
				Location: loc,
				Seed:     seed(cc.Name, t),
			}
			for _, p := range fill(plan, pool, g.history(ch.ID), busy) {
				p.ChannelID = ch.ID
				programs = append(programs, p)
			}
		}
		if err := g.replaceReserved(r, programs, shown); err != nil {
			return err
		}
		if len(shown) > 0 {
			log.Printf("channels: %s filled %q at %s with %d titles\n", cc.Name, f.Name,
				r.Start.In(loc).Format(time.RFC3339), len(shown))
		}
	}
	return nil
}
