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
	"strings"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

func tracks(prefix string, n int) []Title {
	var list []Title
	for i := 0; i < n; i++ {
		list = append(list, Title{Kind: ProgramTrack, RID: fmt.Sprintf("%s-%d", prefix, i), Seconds: 180 + i*7})
	}
	return list
}

func TestBreaks(t *testing.T) {
	c := makeGuide(t, []config.LineupEntry{
		{Name: "Jazz Action", Movies: "genre:action", Align: 15 * time.Minute, BreakMusic: "jazz"},
		{Name: "Sitcoms", Shows: "genre:comedy", Align: 30 * time.Minute},
	})
	lib := testLibrary()
	lib["music:jazz"] = tracks("jazz", 40)
	lib["music:"] = tracks("any", 40)
	now := at("2026-11-11 19:07:00")
	if err := c.Sync(now, lib); err != nil {
		t.Fatal(err)
	}

	for name, prefix := range map[string]string{"Jazz Action": "jazz-", "Sitcoms": "any-"} {
		ch, err := c.findChannel(name)
		if err != nil {
			t.Fatal(err)
		}
		var shows, music []Program
		for _, p := range c.history(ch.ID) {
			if p.Kind == ProgramTrack {
				music = append(music, p)
			} else {
				shows = append(shows, p)
			}
		}
		if len(music) == 0 {
			t.Fatalf("%s: no music", name)
		}
		gaps := 0
		for i := 1; i < len(shows); i++ {
			start, end := shows[i-1].End, shows[i].Start
			if end.Sub(start) < minMusicBreak {
				continue
			}
			gaps++
			var in []Program
			for _, m := range music {
				if !m.Start.Before(start) && m.Start.Before(end) {
					in = append(in, m)
				}
			}
			if len(in) == 0 {
				t.Errorf("%s: silent %s gap at %s", name, end.Sub(start), start)
				continue
			}
			// back to back from the end of the program until the next starts
			at := start
			for _, m := range in {
				if !m.Start.Equal(at) {
					t.Errorf("%s: track at %s, expected %s", name, m.Start, at)
				}
				if !strings.HasPrefix(m.RID, prefix) {
					t.Errorf("%s: track %s from the wrong music", name, m.RID)
				}
				at = m.End
			}
			if !at.Equal(end) {
				t.Errorf("%s: music ends at %s, next program at %s", name, at, end)
			}
		}
		if gaps == 0 {
			t.Errorf("%s: no gaps to fill", name)
		}
		// no track again until the pool has gone round
		for i, m := range music {
			for _, o := range music[i+1 : min(i+40, len(music))] {
				if m.RID == o.RID {
					t.Errorf("%s: %s repeated too soon", name, m.RID)
				}
			}
		}
	}

	// running again adds nothing to gaps that have music
	var before, after int64
	c.db.Model(&Program{}).Where("kind = ?", ProgramTrack).Count(&before)
	if err := c.Sync(now, lib); err != nil {
		t.Fatal(err)
	}
	c.db.Model(&Program{}).Where("kind = ?", ProgramTrack).Count(&after)
	if before != after {
		t.Errorf("tracks %d, then %d", before, after)
	}

	// tracks resolve for the guide
	for _, l := range c.Listings(now, now.Add(3*time.Hour), fakeResolver{}) {
		for _, e := range l.Programs {
			if e.Kind == ProgramTrack && (e.Track == nil || e.OffAir()) {
				t.Errorf("%s: unresolved track %s", l.Channel.Name, e.RID)
			}
		}
	}
}

// A guide request fills a slot the job didn't get to, once.
func TestFillDue(t *testing.T) {
	c := makeGuide(t, featureLineup())
	lib := featureLibrary()
	if err := c.Sync(at("2026-11-11 19:07:00"), lib); err != nil {
		t.Fatal(err)
	}
	friday := at("2026-11-13 20:00:00")
	due := at("2026-11-13 08:00:00")
	if !c.hasReservedBefore(due.Add(featureLead)) {
		t.Fatal("no reserved slot")
	}
	for i := 0; i < 2; i++ {
		if err := c.FillDue(due, lib); err != nil {
			t.Fatal(err)
		}
	}
	var feature []Program
	c.db.Where("block = ? and start_at >= ? and start_at < ?", "New on Friday", friday, friday.Add(4*time.Hour)).
		Order("start_at").Find(&feature)
	if len(feature) != 2 || feature[0].Kind != ProgramMovie {
		t.Fatalf("Friday has %d feature programs: %+v", len(feature), feature)
	}
	if c.hasReservedBefore(friday.Add(time.Minute)) {
		t.Error("Friday is still reserved")
	}
}
