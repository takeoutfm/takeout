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

func featureLineup() []config.LineupEntry {
	return []config.LineupEntry{{
		Name:   "Good Movies",
		Movies: "genre:action",
		Specials: config.SpecialsConfig{
			Every:       7 * 24 * time.Hour,
			Collections: true,
		},
		Features: []config.FeatureConfig{
			{Name: "New on Friday", Layout: "Mon", Match: "Fri", At: "20:00", Length: 4 * time.Hour,
				Source: "added", Description: "The latest movies"},
			{Name: "Saturday Westerns", Layout: "Mon", Match: "Sat", At: "20:00", Length: 4 * time.Hour,
				Movies: "genre:western"},
			// overlaps New on Friday, which is listed first
			{Name: "Late Friday", Layout: "Mon", Match: "Fri", At: "21:00", Length: 2 * time.Hour,
				Movies: "genre:western"},
		},
	}}
}

func featureLibrary() fakeLibrary {
	lib := specialLibrary()
	var added, westerns []Title
	for i := int64(0); i < 3; i++ {
		added = append(added, Title{Kind: ProgramMovie, TMID: 101 + i, Seconds: 100 * 60})
	}
	for i := int64(0); i < 4; i++ {
		westerns = append(westerns, Title{Kind: ProgramMovie, TMID: 201 + i, Seconds: 110 * 60})
	}
	lib["recent:added"] = added
	lib["genre:western"] = westerns
	return lib
}

func TestFeatures(t *testing.T) {
	c := makeGuide(t, featureLineup())
	lib := featureLibrary()
	loc := c.Location()
	now := at("2026-11-11 19:07:00") // a Wednesday

	if err := c.Sync(now, lib); err != nil {
		t.Fatal(err)
	}
	friday := at("2026-11-13 20:00:00")
	saturday := at("2026-11-14 20:00:00")

	// a week out, the slots are reserved and shown with the feature's name
	listing := c.Listings(friday.Add(-time.Minute), friday.Add(time.Minute), nil)
	if len(listing) != 1 || len(listing[0].Programs) == 0 {
		t.Fatal("nothing on Friday")
	}
	var reserved *Entry
	for i, e := range listing[0].Programs {
		if e.Kind == ProgramReserved {
			reserved = &listing[0].Programs[i]
		}
	}
	if reserved == nil || reserved.Block != "New on Friday" || !reserved.Start.Equal(friday) ||
		!reserved.End.Equal(friday.Add(4*time.Hour)) || reserved.Description != "The latest movies" {
		t.Fatalf("Friday slot %+v", reserved)
	}
	if reserved.OffAir() {
		t.Error("a reserved slot is off air")
	}

	seen := make(map[uint]Program)
	for day := now.Add(24 * time.Hour); day.Before(at("2026-12-01 00:00:00")); day = day.Add(24 * time.Hour) {
		if err := c.Sync(day, lib); err != nil {
			t.Fatal(err)
		}
		// what's still to come replaces what was seen, since filling a
		// slot replaces its reserved program
		current := make(map[uint]bool)
		for _, p := range c.airing(day.Add(-time.Hour)) {
			seen[p.ID] = p
			current[p.ID] = true
		}
		for id, p := range seen {
			if p.End.After(day.Add(-time.Hour)) && !current[id] {
				delete(seen, id)
			}
		}
	}
	var all []Program
	for _, p := range seen {
		all = append(all, p)
	}
	sortPrograms(all)

	slotPrograms := func(start time.Time) []Program {
		var list []Program
		for _, p := range all {
			if !p.Start.Before(start) && p.Start.Before(start.Add(4*time.Hour)) {
				list = append(list, p)
			}
		}
		return list
	}
	tmids := func(list []Program, block string) []int64 {
		var ids []int64
		for _, p := range list {
			if p.Block == block {
				ids = append(ids, p.TMID)
			}
		}
		return ids
	}

	for _, p := range all {
		if p.Kind == ProgramReserved && p.Start.Before(at("2026-11-29 00:00:00")) {
			t.Errorf("slot at %s never filled", p.Start.In(loc))
		}
		if p.Block == "Late Friday" {
			t.Error("the second of two overlapping features aired")
		}
		if p.Block != "" && p.Block != "New on Friday" && p.Block != "Saturday Westerns" {
			// a special: never during a feature
			for _, f := range []time.Time{friday, saturday} {
				for w := 0; w < 3; w++ {
					s := f.AddDate(0, 0, 7*w)
					if p.Start.Before(s.Add(4*time.Hour)) && s.Before(p.End) {
						t.Errorf("special %q at %s during a feature", p.Block, p.Start.In(loc))
					}
				}
			}
		}
	}
	for i := 1; i < len(all); i++ {
		if all[i].Start.Before(all[i-1].End) {
			t.Errorf("%s overlaps the program before it", all[i].Start.In(loc))
		}
	}

	// first Friday: the two new movies that fit, from 8:00, inside the slot
	first := slotPrograms(friday)
	if got := tmids(first, "New on Friday"); len(got) != 2 || got[0] != 101 || got[1] != 102 {
		t.Errorf("first Friday showed %v", got)
	}
	if !first[0].Start.Equal(friday) {
		t.Errorf("first Friday starts at %s", first[0].Start.In(loc))
	}
	for _, p := range first {
		if p.End.After(friday.Add(4 * time.Hour)) {
			t.Errorf("%d runs past the slot", p.TMID)
		}
	}
	// next Friday: only the one not shown yet, then regular programming
	second := slotPrograms(friday.AddDate(0, 0, 7))
	if got := tmids(second, "New on Friday"); len(got) != 1 || got[0] != 103 {
		t.Errorf("second Friday showed %v", got)
	}
	if len(tmids(second, "")) == 0 {
		t.Error("second Friday's leftover isn't regular programming")
	}
	// third Friday: nothing new, so all regular programming
	third := slotPrograms(friday.AddDate(0, 0, 14))
	if got := tmids(third, "New on Friday"); len(got) != 0 {
		t.Errorf("third Friday repeated %v", got)
	}
	if len(tmids(third, "")) == 0 {
		t.Error("third Friday is empty")
	}
	// Saturdays: westerns not shown before
	shownWesterns := make(map[int64]bool)
	for w := 0; w < 2; w++ {
		for _, id := range tmids(slotPrograms(saturday.AddDate(0, 0, 7*w)), "Saturday Westerns") {
			if shownWesterns[id] {
				t.Errorf("western %d twice", id)
			}
			shownWesterns[id] = true
		}
	}
	if len(shownWesterns) != 4 {
		t.Errorf("%d westerns in two Saturdays", len(shownWesterns))
	}
}
