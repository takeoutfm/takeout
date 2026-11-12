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
	"reflect"
	"testing"
	"time"

	"takeoutfm.dev/takeout/lib/date"
	. "takeoutfm.dev/takeout/model"
)

var la, _ = time.LoadLocation("America/Los_Angeles")

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, la)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func movies(n int, seconds int) []Title {
	var pool []Title
	for i := 1; i <= n; i++ {
		pool = append(pool, Title{Kind: ProgramMovie, TMID: int64(i), Seconds: seconds + i*37, Vote: float32(i % 10)})
	}
	return pool
}

func series(tvid int64, seasons, episodes, seconds int) []Title {
	var pool []Title
	// out of order on purpose
	for s := seasons; s >= 0; s-- {
		for e := episodes; e >= 1; e-- {
			pool = append(pool, Title{Kind: ProgramEpisode, TVID: tvid, Season: s, Episode: e, Seconds: seconds})
		}
	}
	return pool
}

func TestNextStart(t *testing.T) {
	end := at("2026-11-11 09:37:12")
	cases := []struct {
		align, minBreak time.Duration
		want            string
	}{
		{5 * time.Minute, 0, "2026-11-11 09:40:00"},
		{15 * time.Minute, 0, "2026-11-11 09:45:00"},
		{5 * time.Minute, 3 * time.Minute, "2026-11-11 09:45:00"},
		{5 * time.Minute, 2 * time.Minute, "2026-11-11 09:40:00"},
	}
	for _, c := range cases {
		got := nextStart(end, c.align, c.minBreak, la)
		if !got.Equal(at(c.want)) {
			t.Errorf("%s %s: got %s want %s", c.align, c.minBreak, got.In(la), c.want)
		}
	}
	// already on a mark
	if got := nextStart(at("2026-11-11 10:00:00"), 5*time.Minute, 0, la); !got.Equal(at("2026-11-11 10:00:00")) {
		t.Errorf("got %s", got.In(la))
	}
}

func TestAlignLocal(t *testing.T) {
	// India is +5:30, so hourly marks are at :00 local, not UTC
	kolkata, _ := time.LoadLocation("Asia/Kolkata")
	tm := time.Date(2026, 1, 1, 10, 10, 0, 0, kolkata)
	got := ceilAlign(tm, time.Hour, kolkata).In(kolkata)
	if got.Hour() != 11 || got.Minute() != 0 {
		t.Errorf("got %s", got)
	}
}

func checkSchedule(t *testing.T, plan fillPlan, out []Program) {
	t.Helper()
	if len(out) == 0 {
		t.Fatal("no programs")
	}
	if !out[0].Start.Equal(plan.Start) {
		t.Errorf("first start %s", out[0].Start)
	}
	for i, p := range out {
		if !p.Start.Equal(ceilAlign(p.Start, plan.Align, plan.loc())) {
			t.Errorf("%d not aligned %s", i, p.Start.In(la))
		}
		if !p.Start.Before(plan.Until) {
			t.Errorf("%d starts after until", i)
		}
		if i > 0 {
			prev := out[i-1]
			if p.Start.Before(prev.End.Add(plan.MinBreak)) {
				t.Errorf("%d overlaps: %s < %s", i, p.Start, prev.End)
			}
		}
	}
}

func TestFillMovies(t *testing.T) {
	plan := fillPlan{
		Start:    at("2026-11-11 08:00:00"),
		Until:    at("2026-11-18 08:00:00"),
		Align:    5 * time.Minute,
		Location: la,
		Seed:     42,
	}
	pool := movies(200, 90*60)
	out := fill(plan, pool, nil, busyTimes{})
	checkSchedule(t, plan, out)

	// deterministic
	again := fill(plan, pool, nil, busyTimes{})
	if !reflect.DeepEqual(out, again) {
		t.Error("not deterministic")
	}

	// 200 movies is ~300 hours, so nothing repeats within a week
	seen := make(map[int64]bool)
	for _, p := range out {
		if seen[p.TMID] {
			t.Errorf("repeat %d at %s", p.TMID, p.Start)
		}
		seen[p.TMID] = true
	}
}

func TestFillSmallPoolRepeats(t *testing.T) {
	plan := fillPlan{
		Start:    at("2026-11-11 00:00:00"),
		Until:    at("2026-11-18 00:00:00"),
		Align:    5 * time.Minute,
		Location: la,
	}
	// 30 two hour movies, 60 hours; cooldown is 30 hours
	pool := movies(30, 2*3600)
	out := fill(plan, pool, nil, busyTimes{})
	checkSchedule(t, plan, out)
	last := make(map[int64]time.Time)
	for _, p := range out {
		if l, ok := last[p.TMID]; ok && p.Start.Sub(l) < 24*time.Hour {
			t.Errorf("%d repeated after %s", p.TMID, p.Start.Sub(l))
		}
		last[p.TMID] = p.Start
	}
}

func TestFillSeriesInOrder(t *testing.T) {
	plan := fillPlan{
		Start:    at("2026-11-11 18:00:00"),
		Until:    at("2026-11-13 18:00:00"),
		Align:    30 * time.Minute,
		Location: la,
	}
	var pool []Title
	pool = append(pool, series(1, 3, 10, 22*60)...)
	pool = append(pool, series(2, 2, 8, 23*60)...)
	pool = append(pool, series(3, 5, 20, 44*60)...)

	history := []Program{
		{Kind: ProgramEpisode, TVID: 1, Season: 1, Episode: 10, Start: at("2026-11-11 17:00:00")},
		{Kind: ProgramEpisode, TVID: 2, Season: 2, Episode: 8, Start: at("2026-11-11 17:30:00")},
	}
	out := fill(plan, pool, history, busyTimes{})
	checkSchedule(t, plan, out)

	next := map[int64][2]int{1: {2, 1}, 2: {1, 1}, 3: {1, 1}}
	prev := int64(2)
	for _, p := range out {
		if p.Season == 0 {
			t.Error("special scheduled")
		}
		want := next[p.TVID]
		if p.Season != want[0] || p.Episode != want[1] {
			t.Errorf("series %d got S%dE%d want S%dE%d", p.TVID, p.Season, p.Episode, want[0], want[1])
		}
		if p.TVID == prev {
			t.Errorf("series %d twice in a row at %s", p.TVID, p.Start.In(la))
		}
		prev = p.TVID
		// advance, wrapping
		max := map[int64][2]int{1: {3, 10}, 2: {2, 8}, 3: {5, 20}}[p.TVID]
		s, e := p.Season, p.Episode+1
		if e > max[1] {
			s, e = s+1, 1
		}
		if s > max[0] {
			s = 1
		}
		next[p.TVID] = [2]int{s, e}
	}
}

func TestFillBusy(t *testing.T) {
	plan := fillPlan{
		Start:    at("2026-11-11 00:00:00"),
		Until:    at("2026-11-14 00:00:00"),
		Align:    5 * time.Minute,
		Location: la,
	}
	pool := movies(30, 100*60)
	busy := busyTimes{}
	a := fill(plan, pool, nil, busy)
	plan.Seed = 1
	b := fill(plan, pool, nil, busy)
	for _, p := range a {
		for _, q := range b {
			if p.TMID == q.TMID && p.Start.Before(q.End) && q.Start.Before(p.End) {
				t.Errorf("%d on both channels at %s", p.TMID, p.Start)
			}
		}
	}
}

func TestFillOnAir(t *testing.T) {
	plan := fillPlan{
		Start:    at("2026-10-30 00:00:00"),
		Until:    at("2026-11-03 00:00:00"),
		Align:    5 * time.Minute,
		Location: la,
		OnAir: func(day time.Time) bool {
			return date.MatchTime("Jan", "Oct", day)
		},
	}
	out := fill(plan, movies(50, 100*60), nil, busyTimes{})
	if len(out) == 0 {
		t.Fatal("no programs")
	}
	for _, p := range out {
		if p.Start.In(la).Month() != time.October {
			t.Errorf("program in %s", p.Start.In(la))
		}
	}
	if last := out[len(out)-1]; last.Start.In(la).Day() != 31 {
		t.Errorf("last program %s", last.Start.In(la))
	}
}

func TestFillDST(t *testing.T) {
	// clocks go back on 2026-11-01 in Los Angeles
	plan := fillPlan{
		Start:    at("2026-10-31 20:00:00"),
		Until:    at("2026-11-02 08:00:00"),
		Align:    15 * time.Minute,
		Location: la,
	}
	out := fill(plan, movies(40, 95*60), nil, busyTimes{})
	checkSchedule(t, plan, out)
	for _, p := range out {
		if m := p.Start.In(la).Minute(); m%15 != 0 {
			t.Errorf("start %s", p.Start.In(la))
		}
	}
}

func TestFillEmpty(t *testing.T) {
	if out := fill(fillPlan{Start: at("2026-11-11 00:00:00"), Until: at("2026-11-12 00:00:00")}, nil, nil, busyTimes{}); out != nil {
		t.Error("expected nothing")
	}
}

// A movie channel fills up to a special without leaving a break no movie fits.
func TestFillCloseGapBeforeBlock(t *testing.T) {
	special := at("2026-11-12 20:00:00")
	sp := Program{Start: special, End: special.Add(2 * time.Hour), Kind: ProgramMovie, TMID: 9999, Block: "Marathon"}
	var gaps []time.Duration
	for seed := uint64(1); seed <= 50; seed++ {
		plan := fillPlan{
			Start:    at("2026-11-12 08:00:00"),
			Until:    at("2026-11-13 08:00:00"),
			Align:    30 * time.Minute,
			MinBreak: time.Minute,
			Location: la,
			Seed:     seed,
			Blocks:   []block{{Start: special, End: sp.End.Add(30 * time.Minute), Programs: []Program{sp}}},
		}
		// 75 minutes to a little over 3 hours
		out := fill(plan, movies(200, 75*60), nil, busyTimes{})
		checkSchedule(t, plan, out)
		for i, p := range out {
			if p.Block != "" {
				gap := p.Start.Sub(nextStart(out[i-1].End, plan.Align, plan.MinBreak, la))
				gaps = append(gaps, gap)
				if gap >= plan.Align {
					t.Errorf("seed %d: %s before the special", seed, gap)
				}
				break
			}
		}
	}
	if len(gaps) != 50 {
		t.Fatalf("special missing: %d", len(gaps))
	}
}
