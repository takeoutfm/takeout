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
	"path/filepath"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

type fakeLibrary struct {
	pools map[string][]Title
}

func newLibrary() *fakeLibrary {
	return &fakeLibrary{
		pools: map[string][]Title{
			"big":                  pool(400, 80),
			"tiny":                 pool(10, 5), // about 35 minutes
			`+series:"Charts"`:     pool(200, 40),
			`+artist:"Gary Numan"`: showPool("g", 60),
		},
	}
}

func (l *fakeLibrary) Tracks(query string, limit int) []Title { return l.pools[query] }

// byName is every station row, on the air or not.
func byName(r *Radio) map[string]Station {
	m := make(map[string]Station)
	for _, s := range r.stations() {
		m[s.Name] = s
	}
	return m
}

func names(list []Station) string {
	var n []string
	for _, s := range list {
		n = append(n, s.Name)
	}
	return fmt.Sprint(n)
}

func makeRadio(t *testing.T, lineup []config.RadioEntry) *Radio {
	cfg, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Radio.DB.Source = filepath.Join(t.TempDir(), "radio.db")
	cfg.Radio.Timezone = "America/Los_Angeles"
	cfg.Radio.Lineup = lineup
	cfg.Radio.Decades = nil
	cfg.Radio.Series = []string{"Charts"}
	r := NewRadio(cfg)
	if err := r.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	return r
}

func TestSync(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{
		{Name: "Big", Music: config.Queries{"big"}, Image: "big.png", Description: "Lots"},
		{Name: "Tiny", Music: config.Queries{"tiny"}},
		{Name: "Spooky", Music: config.Queries{"big"}, Layout: "Jan", Match: "Oct"},
	})
	lib := newLibrary()
	now := t0
	if err := r.Sync(now, lib); err != nil {
		t.Fatal(err)
	}

	// every configured station has a row; only those with music in season
	// are on the air
	stations := byName(r)
	if len(stations) != 4 {
		t.Fatalf("stations %v", stations)
	}
	big := stations["Big"]
	if big.Type != TypeLive || big.Rank != 0 || big.Image != "big.png" ||
		big.Description != "Lots" || big.Creator != "Takeout" || !big.OnAir {
		t.Errorf("big %+v", big)
	}
	if stations["Tiny"].OnAir {
		t.Error("tiny station has too little music")
	}
	if stations["Spooky"].OnAir {
		t.Error("spooky station out of season")
	}
	if got := names(r.Live()); got != "[Big Charts]" {
		t.Errorf("live %s", got)
	}

	for _, name := range []string{"Big", "Charts"} {
		h := r.history(name)
		if len(h) == 0 || !h[0].Start.Equal(now) {
			t.Fatalf("%s starts %v", name, h)
		}
		if end := h[len(h)-1].End; end.Before(now.Add(24 * time.Hour)) {
			t.Errorf("%s ends %v", name, end)
		}
	}

	// a later run extends, keeps what's scheduled and the station IDs
	before := len(r.history("Big"))
	later := now.Add(3 * time.Hour)
	if err := r.Sync(later, lib); err != nil {
		t.Fatal(err)
	}
	h := r.history("Big")
	if len(h) <= before {
		t.Errorf("not extended: %d to %d", before, len(h))
	}
	for i := 1; i < len(h); i++ {
		if !h[i].Start.Equal(h[i-1].End) {
			t.Fatalf("gap at %d", i)
		}
	}
	if h[0].End.Before(later.Add(-keepEnded)) {
		t.Errorf("old tracks kept: %v", h[0].End)
	}
	if byName(r)["Big"].ID != big.ID {
		t.Error("station ID changed")
	}

	// in season: the seasonal station comes on the air in lineup order,
	// with the ID it had
	spooky := byName(r)["Spooky"]
	oct := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	if err := r.Sync(oct, lib); err != nil {
		t.Fatal(err)
	}
	if got := names(r.Live()); got != "[Big Spooky Charts]" {
		t.Errorf("live %s", got)
	}
	if byName(r)["Spooky"].ID != spooky.ID {
		t.Error("seasonal station ID changed")
	}
	if len(r.history("Spooky")) == 0 {
		t.Error("seasonal station not scheduled")
	}
}

func TestSyncRemoved(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: config.Queries{"big"}}})
	lib := newLibrary()
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	r.config.Radio.Lineup = []config.RadioEntry{{Name: "Other", Music: config.Queries{"big"}}}
	if err := r.Sync(t0.Add(time.Hour), lib); err != nil {
		t.Fatal(err)
	}
	if _, ok := byName(r)["Big"]; ok {
		t.Error("removed station kept")
	}
	if _, ok := byName(r)["Other"]; !ok {
		t.Error("new station missing")
	}
	if len(r.history("Big")) != 0 {
		t.Error("removed station's schedule kept")
	}
}

func TestExtendDue(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: config.Queries{"big"}}})
	lib := newLibrary()
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	end, _ := r.lastEnd("Big")

	// plenty left: nothing changes
	if err := r.ExtendDue(t0.Add(time.Hour), lib); err != nil {
		t.Fatal(err)
	}
	if e, _ := r.lastEnd("Big"); !e.Equal(end) {
		t.Error("extended early")
	}
	// the job was late: the request extends
	if err := r.ExtendDue(end.Add(-time.Hour), lib); err != nil {
		t.Fatal(err)
	}
	if e, _ := r.lastEnd("Big"); !e.After(end) {
		t.Error("not extended")
	}
}

func TestSchedule(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: config.Queries{"big"}}})
	lib := newLibrary()
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	from := t0.Add(10 * time.Minute)
	list := r.Schedule(from, from.Add(30*time.Minute))
	stations := make(map[string]int)
	for _, x := range list {
		stations[x.Station]++
		if !x.Start.Before(from.Add(30*time.Minute)) || !x.End.After(from) {
			t.Fatalf("outside range %v %v", x.Start, x.End)
		}
	}
	if stations["Big"] < 8 || stations["Charts"] < 8 {
		t.Errorf("tracks %v", stations)
	}
}

func TestLineup(t *testing.T) {
	cfg, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	// no Radio settings, no stations
	cfg.Radio.Lineup = nil
	cfg.Radio.Genres = nil
	cfg.Radio.Decades = nil
	cfg.Radio.Series = nil
	if list := lineup(cfg); len(list) != 0 {
		t.Fatalf("got %+v", list)
	}

	// lineup, genres, decades, series; a repeated name keeps the first
	cfg.Radio.Lineup = []config.RadioEntry{{Name: "Rock", Music: config.Queries{"my rock"}}, {Name: "Empty"}}
	cfg.Radio.Genres = []string{"rock", "", "jazz"}
	cfg.Radio.Decades = []int{1980}
	cfg.Radio.Series = []string{"Hot 100"}
	list := lineup(cfg)
	var names []string
	for _, s := range list {
		names = append(names, s.Name)
	}
	want := []string{"Rock", "Jazz", "1980s Top Tracks", "Hot 100"}
	if fmt.Sprint(names) != fmt.Sprint(want) || fmt.Sprint(list[0].Queries) != "[my rock]" {
		t.Fatalf("got %+v", list)
	}
}

func TestStreams(t *testing.T) {
	r := makeRadio(t, nil)
	r.config.Radio.Series = nil
	r.config.Radio.Streams = []config.RadioStream{
		{Creator: "KEXP", Title: "KEXP", Source: []config.ContentDescription{
			{ContentType: "audio/aac", URL: "https://kexp.org/kexp.aac"}}},
		{Creator: "SomaFM", Title: "Groove Salad", Image: "gs.png"},
	}
	lib := newLibrary()
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	list := r.Streams()
	if got := names(list); got != "[KEXP Groove Salad]" {
		t.Fatalf("streams %s", got)
	}
	if list[0].Type != TypeStream || list[0].Source != `[{"contentType":"audio/aac","url":"https://kexp.org/kexp.aac"}]` {
		t.Errorf("kexp %+v", list[0])
	}
	if len(r.Live()) != 0 {
		t.Error("streams listed as live")
	}

	// changed and removed streams; IDs stay with names
	kexp := list[0]
	r.config.Radio.Streams = []config.RadioStream{{Creator: "KEXP", Title: "KEXP", Image: "new.png"}}
	if err := r.Sync(t0.Add(time.Hour), lib); err != nil {
		t.Fatal(err)
	}
	list = r.Streams()
	if names(list) != "[KEXP]" || list[0].ID != kexp.ID || list[0].Image != "new.png" {
		t.Errorf("streams %+v", list)
	}
}

func TestFindStation(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{
		{Name: "Big", Music: config.Queries{"big"}},
		{Name: "Bigger", Music: config.Queries{"big"}},
		{Name: "Tiny", Music: config.Queries{"tiny"}},
	})
	r.config.Radio.Streams = []config.RadioStream{{Creator: "SomaFM", Title: "Groove Salad"}}
	if err := r.Sync(t0, newLibrary()); err != nil {
		t.Fatal(err)
	}
	big := byName(r)["Big"]
	if s, err := r.FindStation(fmt.Sprint(big.ID)); err != nil || s.Name != "Big" {
		t.Errorf("by id %+v %v", s, err)
	}
	if s, err := r.FindStation("name:big"); err != nil || s.ID != big.ID {
		t.Errorf("by name %+v %v", s, err)
	}
	if s, err := r.FindStation("name:soma"); err != nil || s.Name != "Groove Salad" {
		t.Errorf("by creator %+v %v", s, err)
	}
	// off the air: not offered
	tiny := byName(r)["Tiny"]
	if _, err := r.FindStation(fmt.Sprint(tiny.ID)); err != ErrStationNotFound {
		t.Errorf("found off-air station: %v", err)
	}
	if _, err := r.FindStation("name:tiny"); err != ErrStationNotFound {
		t.Errorf("found off-air station by name: %v", err)
	}
	if got := names(r.StationsLike("r")); got != "[Bigger Charts Groove Salad]" {
		t.Errorf("like %s", got)
	}
}

func TestScheduleLocalTime(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: config.Queries{"big"}}})
	lib := newLibrary()
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	now := t0.In(la)
	if err := r.Sync(now, lib); err != nil {
		t.Fatal(err)
	}
	list := r.StationSchedule("Big", now, now.Add(10*time.Minute))
	if len(list) == 0 || list[0].Start.After(now) || !list[0].End.After(now) {
		t.Fatalf("not on now: %+v", list)
	}
}

func TestSyncNameTaken(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "KEXP", Music: config.Queries{"big"}}, {Name: "Big", Music: config.Queries{"big"}}})
	r.config.Radio.Streams = []config.RadioStream{{Creator: "KEXP", Title: "KEXP"}}
	if err := r.Sync(t0, newLibrary()); err != nil {
		t.Fatal(err)
	}
	// the live station comes first, so the stream with its name is dropped
	if s := byName(r)["KEXP"]; s.Type != TypeLive {
		t.Errorf("kexp %+v", s)
	}
	if len(r.Streams()) != 0 {
		t.Error("stream with a taken name kept")
	}
	if len(r.history("KEXP")) == 0 || len(r.history("Big")) == 0 {
		t.Error("live stations not scheduled")
	}
}
