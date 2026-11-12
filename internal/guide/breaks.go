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
	. "takeoutfm.dev/takeout/model"
)

const (
	// shorter gaps stay silent
	minMusicBreak = 30 * time.Second
	// a track isn't played again on a channel within this many breaks' worth of tracks
	recentTracks = 200
)

// breakQuery is the channel's break music query, else the collection's.
func breakQuery(gc config.GuideConfig, cc config.LineupEntry) string {
	if cc.BreakMusic != "" {
		return cc.BreakMusic
	}
	return gc.BreakMusic
}

// fillBreaks puts music in the channel's upcoming gaps between programs that
// don't have any yet. Tracks play back to back; the last is cut off at the
// next program's start, where the apps fade it out. It's safe to run any
// time, so gaps made by specials and features get music too.
func (g *Guide) fillBreaks(now time.Time, ch Channel, cc config.LineupEntry, lib Library) error {
	history := g.history(ch.ID)
	var shows, tracks []Program
	for _, p := range history {
		if p.Kind == ProgramTrack {
			tracks = append(tracks, p)
		} else {
			shows = append(shows, p)
		}
	}

	type gap struct{ start, end time.Time }
	var gaps []gap
	for i := 1; i < len(shows); i++ {
		start, end := shows[i-1].End, shows[i].Start
		if !end.After(now) || end.Sub(start) < minMusicBreak {
			continue
		}
		filled := false
		for _, t := range tracks {
			if !t.Start.Before(start) && t.Start.Before(end) {
				filled = true
				break
			}
		}
		if !filled {
			gaps = append(gaps, gap{start, end})
		}
	}
	if len(gaps) == 0 {
		return nil
	}

	pool := lib.BreakTracks(breakQuery(g.config.Guide, cc), g.config.Guide.SearchLimit)
	if len(pool) == 0 {
		return nil
	}
	// what the channel played lately, oldest first
	var recent []string
	for _, t := range tracks[max(0, len(tracks)-recentTracks):] {
		recent = append(recent, t.RID)
	}
	played := make(map[string]int)
	for _, rid := range recent {
		played[rid]++
	}
	rng := rand.New(rand.NewPCG(seed(ch.Name, gaps[0].start), 0x6d75736963))
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	next := 0
	pick := func() Title {
		// the next track not played lately, or the least recent if all were
		for i := 0; i < len(pool); i++ {
			t := pool[(next+i)%len(pool)]
			if played[t.RID] == 0 {
				next = (next + i + 1) % len(pool)
				return t
			}
		}
		for len(recent) > 0 {
			rid := recent[0]
			recent = recent[1:]
			played[rid]--
			if played[rid] == 0 {
				for _, t := range pool {
					if t.RID == rid {
						return t
					}
				}
			}
		}
		t := pool[next]
		next = (next + 1) % len(pool)
		return t
	}

	var out []Program
	for _, gp := range gaps {
		for t := gp.start; t.Before(gp.end); {
			title := pick()
			p := title.program(t)
			p.ChannelID = ch.ID
			t = p.End
			if p.End.After(gp.end) {
				p.End = gp.end
			}
			out = append(out, p)
			recent = append(recent, title.RID)
			played[title.RID]++
		}
	}
	return g.createPrograms(out)
}
