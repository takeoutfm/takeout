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
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

func specialLibrary() fakeLibrary {
	lib := testLibrary()
	pool := lib["genre:action"]
	// Die Hard: three action movies, released out of TMID order
	var dieHard []Title
	for i, tmid := range []int64{7, 3, 5} {
		m := pool[tmid-1]
		m.Date = time.Date(1988+i*2, 7, 1, 0, 0, 0, 0, time.UTC)
		dieHard = append(dieHard, m)
	}
	lib["collection:Die Hard Collection"] = dieHard
	// two of three in the pool: not enough for a special
	lib["collection:Alien Collection"] = []Title{pool[0], pool[1], {Kind: ProgramMovie, TMID: 900, Seconds: 6000}}
	lib["keyword:heist"] = pool[20:26]
	return lib
}

func specialLineup() []config.LineupEntry {
	specials := config.SpecialsConfig{
		Every:       14 * 24 * time.Hour,
		MaxLength:   8 * time.Hour,
		Collections: true,
		Themes:      []config.ThemeConfig{{Name: "Heist Night", Movies: "keyword:heist"}},
	}
	return []config.LineupEntry{
		{Name: "Action", Movies: "genre:action", Specials: specials},
		{Name: "More Action", Movies: "genre:action", Specials: specials},
	}
}

func TestSpecials(t *testing.T) {
	c := makeGuide(t, specialLineup())
	lib := specialLibrary()
	now := at("2026-11-11 19:07:00")
	loc := c.Location()

	// keep a copy of every program, since old ones are cleaned up
	seen := make(map[uint]Program)
	end := now.Add(70 * 24 * time.Hour)
	for day := now; day.Before(end); day = day.Add(24 * time.Hour) {
		if err := c.Sync(day, lib); err != nil {
			t.Fatal(err)
		}
		for _, p := range c.airing(day.Add(-time.Hour)) {
			seen[p.ID] = p
		}
	}

	for _, name := range []string{"Action", "More Action"} {
		ch, err := c.findChannel(name)
		if err != nil {
			t.Fatal(err)
		}
		airings := c.specialAirings(ch.ID)
		if len(airings) < 4 {
			t.Fatalf("%s: %d specials in 10 weeks", name, len(airings))
		}
		for i, a := range airings {
			if a.Name == "Alien Collection" {
				t.Errorf("%s: a collection without enough films", name)
			}
			if h, m := a.Start.In(loc).Hour(), a.Start.In(loc).Minute(); h != 20 || m != 0 {
				t.Errorf("%s: %s starts at %d:%02d", name, a.Name, h, m)
			}
			if i > 0 {
				prev := airings[i-1]
				if prev.Name == a.Name {
					t.Errorf("%s: %s twice in a row", name, a.Name)
				}
				if gap := a.Start.Sub(prev.Start); gap < 13*24*time.Hour {
					t.Errorf("%s: specials %s apart", name, gap)
				}
			}

			// the special's programs, back to back from its start
			var block []Program
			for _, p := range seen {
				if p.ChannelID == ch.ID && !p.Start.Before(a.Start) && p.Start.Before(a.End) {
					block = append(block, p)
				}
			}
			sortPrograms(block)
			if len(block) < minSpecialPrograms || !block[0].Start.Equal(a.Start) {
				t.Errorf("%s: %s at %s has %d programs", name, a.Name, a.Start, len(block))
				continue
			}
			for _, p := range block {
				if p.Block != a.Name {
					t.Errorf("%s: %s has a program from %q", name, a.Name, p.Block)
				}
			}
			if a.Name == "Die Hard Collection" {
				var order []int64
				for _, p := range block {
					order = append(order, p.TMID)
				}
				if len(order) != 3 || order[0] != 7 || order[1] != 3 || order[2] != 5 {
					t.Errorf("%s: Die Hard in order %v, want release order [7 3 5]", name, order)
				}
			}
			if last := block[len(block)-1]; last.End.After(a.Start.Add(8 * time.Hour)) {
				t.Errorf("%s: %s runs past its max length to %s", name, a.Name, last.End)
			}
		}

		// specials and regular programming never overlap
		var all []Program
		for _, p := range seen {
			if p.ChannelID == ch.ID {
				all = append(all, p)
			}
		}
		sortPrograms(all)
		for i := 1; i < len(all); i++ {
			if all[i].Start.Before(all[i-1].End) {
				t.Errorf("%s: %s overlaps the program before it", name, all[i].Start)
			}
		}
	}

	// no movie on two channels at once
	var everything []Program
	for _, p := range seen {
		everything = append(everything, p)
	}
	sortPrograms(everything)
	for i, p := range everything {
		for _, q := range everything[i+1:] {
			if !q.Start.Before(p.End) {
				break
			}
			if p.ChannelID != q.ChannelID && p.TMID == q.TMID {
				t.Errorf("movie %d on two channels at %s", p.TMID, q.Start)
			}
		}
	}

	// the same special isn't on both channels within a week
	var both []SpecialAiring
	c.db.Order("start_at").Find(&both)
	for i, a := range both {
		for _, b := range both[i+1:] {
			if a.ChannelID != b.ChannelID && a.Name == b.Name && b.Start.Sub(a.Start) < specialApart {
				t.Errorf("%s on both channels %s apart", a.Name, b.Start.Sub(a.Start))
			}
		}
	}
}

// A channel without Specials config has none, and its programs have no block.
func TestNoSpecials(t *testing.T) {
	c := makeGuide(t, lineup())
	now := at("2026-11-11 19:07:00")
	if err := c.Sync(now, specialLibrary()); err != nil {
		t.Fatal(err)
	}
	var count int64
	c.db.Model(&SpecialAiring{}).Count(&count)
	if count != 0 {
		t.Errorf("%d specials", count)
	}
	for _, p := range c.airing(now) {
		if p.Block != "" {
			t.Fatalf("program in block %q", p.Block)
		}
	}
}

func sortPrograms(list []Program) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j].Start.Before(list[j-1].Start); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}
