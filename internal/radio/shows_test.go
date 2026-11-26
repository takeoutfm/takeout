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
	"fmt"
	"strings"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

// showPool is n tracks by one artist, with RIDs starting with prefix.
func showPool(prefix string, n int) []Title {
	var list []Title
	for i := 0; i < n; i++ {
		list = append(list, Title{
			RID:     fmt.Sprintf("%s%d", prefix, i),
			Artist:  "Gary Numan",
			Release: fmt.Sprintf("rel%d", i/10),
			Seconds: 200 + i%40,
		})
	}
	return list
}

func la(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestNewShow(t *testing.T) {
	s, err := newShow(config.RadioShow{Name: "All Day", Music: config.Queries{"x"}})
	if err != nil || s.Length != 24*time.Hour || s.hour != 0 || s.min != 0 {
		t.Errorf("defaults %+v %v", s, err)
	}
	s, err = newShow(config.RadioShow{Name: "Late", Music: config.Queries{"x"}, At: "23:30", Length: time.Hour})
	if err != nil || s.hour != 23 || s.min != 30 {
		t.Errorf("at %+v %v", s, err)
	}
	for _, bad := range []config.RadioShow{
		{Music: config.Queries{"x"}},
		{Name: "No music"},
		{Name: "Bad time", Music: config.Queries{"x"}, At: "8pm"},
	} {
		if _, err := newShow(bad); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}

func TestShowQueries(t *testing.T) {
	now := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	month := 30 * 24 * time.Hour
	for _, c := range []struct {
		show config.RadioShow
		want []string
	}{
		{config.RadioShow{Name: "New", Released: month},
			[]string{`+first_date:>="2026-10-22"`}},
		{config.RadioShow{Name: "New Rock", Music: config.Queries{"+genre:rock"}, Released: month},
			[]string{`+genre:rock +first_date:>="2026-10-22"`}},
		{config.RadioShow{Name: "Gary", Music: config.Queries{`+artist:"Gary Numan"`}},
			[]string{`+artist:"Gary Numan"`}},
		// several queries, each limited to new releases; empty ones dropped
		{config.RadioShow{Name: "New Shoegaze", Released: month,
			Music: config.Queries{`+artist:"DIIV"`, " ", `+artist:"Slowdive"`}},
			[]string{`+artist:"DIIV" +first_date:>="2026-10-22"`,
				`+artist:"Slowdive" +first_date:>="2026-10-22"`}},
	} {
		s, err := newShow(c.show)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.queries(now); fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%s: %q", c.show.Name, got)
		}
	}
	if _, err := newShow(config.RadioShow{Name: "Blank", Music: config.Queries{" "}}); err == nil {
		t.Error("accepted a show with only blank queries")
	}
}

func TestPoolQueries(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{
		// two queries combined; the overlap isn't repeated
		{Name: "Both", Music: config.Queries{"tiny", "half"}},
	})
	r.config.Radio.Series = nil
	lib := newLibrary()
	lib.pools["half"] = append(pool(10, 5), showPool("h", 30)...)
	got := r.pool(lib, []string{"tiny", "half"})
	if len(got) != 40 {
		t.Errorf("pool has %d tracks, want 40", len(got))
	}
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	if names(r.Live()) != "[Both]" {
		t.Errorf("live %s", names(r.Live()))
	}
}

func TestSlots(t *testing.T) {
	loc := la(t)
	fri, _ := newShow(config.RadioShow{Name: "Late Friday", Music: config.Queries{"x"},
		Layout: "Mon", Match: "Fri", At: "23:00", Length: 3 * time.Hour})
	sat, _ := newShow(config.RadioShow{Name: "Saturday", Music: config.Queries{"x"},
		Layout: "Mon", Match: "Sat", At: "09:00", Length: 2 * time.Hour})
	shows := []show{fri, sat}

	// early Saturday: Friday's late show is still on, then Saturday's
	from := time.Date(2026, 11, 21, 1, 0, 0, 0, loc)
	list := slots(shows, from, from.Add(24*time.Hour), loc)
	if len(list) != 2 {
		t.Fatalf("slots %+v", list)
	}
	if list[0].Show != 0 || !list[0].Start.Equal(time.Date(2026, 11, 20, 23, 0, 0, 0, loc)) {
		t.Errorf("friday %+v", list[0])
	}
	if list[1].Show != 1 || !list[1].Start.Equal(time.Date(2026, 11, 21, 9, 0, 0, 0, loc)) ||
		list[1].End.Sub(list[1].Start) != 2*time.Hour {
		t.Errorf("saturday %+v", list[1])
	}

	sl, on, _ := at(list, from)
	if !on || sl.Show != 0 {
		t.Errorf("on at 1am %+v %v", sl, on)
	}
	_, on, next := at(list, from.Add(3*time.Hour))
	if on || !next.Equal(list[1].Start) {
		t.Errorf("next after 4am %v %v", on, next)
	}

	// every day when there's no Layout
	daily, _ := newShow(config.RadioShow{Name: "Daily", Music: config.Queries{"x"}, At: "06:00", Length: time.Hour})
	if n := len(slots([]show{daily}, from, from.Add(7*24*time.Hour), loc)); n != 7 {
		t.Errorf("daily slots %d", n)
	}
}

func TestSyncShow(t *testing.T) {
	gary := config.RadioShow{Name: "Gary Numan Mondays", Music: config.Queries{`+artist:"Gary Numan"`},
		Layout: "Mon", Match: "Mon", At: "06:00", Length: 2 * time.Hour, Image: "gary.png"}
	r := makeRadio(t, []config.RadioEntry{
		{Name: "Big", Music: config.Queries{"big"}, Shows: []config.RadioShow{gary}},
	})
	lib := newLibrary()
	loc := la(t)
	// t0 is Monday 04:00 in Los Angeles
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 16, 6, 0, 0, 0, loc)
	end := start.Add(2 * time.Hour)
	checkShow(t, r.history("Big"), "Gary Numan Mondays", start, end)

	if s, ok := r.Show("Big", "Gary Numan Mondays"); !ok || s.Image != "gary.png" {
		t.Errorf("show %+v", s)
	}

	// a changed show reschedules after the track on now
	now := t0.Add(time.Hour)
	before := r.StationSchedule("Big", now, now.Add(time.Second))
	r.config.Radio.Lineup[0].Shows[0].At = "08:00"
	if err := r.Sync(now, lib); err != nil {
		t.Fatal(err)
	}
	after := r.StationSchedule("Big", now, now.Add(time.Second))
	if len(before) != 1 || len(after) != 1 || after[0].RID != before[0].RID ||
		!after[0].Start.Equal(before[0].Start) {
		t.Errorf("track on now changed: %+v %+v", before, after)
	}
	start = time.Date(2026, 11, 16, 8, 0, 0, 0, loc)
	checkShow(t, r.history("Big"), "Gary Numan Mondays", start, start.Add(2*time.Hour))
}

func TestSyncShowResume(t *testing.T) {
	gary := config.RadioShow{Name: "Gary", Music: config.Queries{`+artist:"Gary Numan"`},
		At: "06:00", Length: 2 * time.Hour}
	r := makeRadio(t, []config.RadioEntry{
		{Name: "Big", Music: config.Queries{"big"}, Shows: []config.RadioShow{gary}},
	})
	lib := newLibrary()
	loc := la(t)
	// the first run ends mid-show, the next carries it on
	r.config.Radio.Horizon = 3 * time.Hour
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	if err := r.Sync(t0.Add(2*time.Hour), lib); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 16, 6, 0, 0, 0, loc)
	checkShow(t, r.history("Big"), "Gary", start, start.Add(2*time.Hour))
}

func TestSyncShowTooSmall(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: config.Queries{"big"}, Shows: []config.RadioShow{
		{Name: "Tiny Show", Music: config.Queries{"tiny"}, At: "06:00", Length: 3 * time.Hour},
		{Name: "Nothing", Music: config.Queries{"nothing"}, At: "10:00", Length: time.Hour},
	}}})
	if err := r.Sync(t0, newLibrary()); err != nil {
		t.Fatal(err)
	}
	for _, x := range r.history("Big") {
		if x.Show != "" {
			t.Fatalf("scheduled a show without enough music: %+v", x)
		}
	}
}

// checkShow checks the show's tracks: all from its pool, back to back, the
// first starting when the track on at start ends, and the last starting
// before end.
func checkShow(t *testing.T, h []RadioTrack, name string, start, end time.Time) {
	t.Helper()
	var first, last = -1, -1
	for i, x := range h {
		if i > 0 && !x.Start.Equal(h[i-1].End) {
			t.Fatalf("gap at %d", i)
		}
		if x.Show == "" {
			if strings.HasPrefix(x.RID, "g") {
				t.Fatalf("show track outside the show: %+v", x)
			}
			continue
		}
		if x.Show != name || !strings.HasPrefix(x.RID, "g") {
			t.Fatalf("wrong show track %+v", x)
		}
		if first < 0 {
			first = i
		}
		last = i
	}
	if first < 1 {
		t.Fatalf("no show, or no track before it")
	}
	for i := first; i <= last; i++ {
		if h[i].Show != name {
			t.Fatalf("show interrupted at %d", i)
		}
	}
	if h[first].Start.Before(start) || !h[first-1].Start.Before(start) {
		t.Errorf("show starts %s, slot %s", h[first].Start, start)
	}
	if !h[last].Start.Before(end) || (last+1 < len(h) && h[last+1].Start.Before(end)) {
		t.Errorf("show ends %s, slot ends %s", h[last].End, end)
	}
}
