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
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/music"
	. "takeoutfm.dev/takeout/model"
)

type fakeLibrary struct {
	pools    map[string][]Title
	stations map[uint]Station
	nextID   uint
}

func newLibrary() *fakeLibrary {
	return &fakeLibrary{
		pools: map[string][]Title{
			"big":              pool(400, 80),
			"tiny":             pool(10, 5), // about 35 minutes
			`+series:"Charts"`: pool(200, 40),
		},
		stations: make(map[uint]Station),
	}
}

func (l *fakeLibrary) Tracks(query string, limit int) []Title { return l.pools[query] }

func (l *fakeLibrary) TakeoutStations() []Station {
	var list []Station
	for _, s := range l.stations {
		list = append(list, s)
	}
	return list
}

func (l *fakeLibrary) CreateStation(s *Station) error {
	for _, x := range l.stations {
		if x.Name == s.Name {
			return errors.New("duplicate")
		}
	}
	l.nextID++
	s.ID = l.nextID
	l.stations[s.ID] = *s
	return nil
}

// add puts an existing station row in the library.
func (l *fakeLibrary) add(id uint, s Station) {
	s.ID = id
	l.stations[id] = s
	l.nextID = max(l.nextID, id)
}

func (l *fakeLibrary) UpdateStation(s *Station) error { l.stations[s.ID] = *s; return nil }
func (l *fakeLibrary) DeleteStation(s *Station) error { delete(l.stations, s.ID); return nil }

func (l *fakeLibrary) byName() map[string]Station {
	m := make(map[string]Station)
	for _, s := range l.stations {
		m[s.Name] = s
	}
	return m
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
		{Name: "Big", Music: "big", Image: "big.png", Description: "Lots"},
		{Name: "Tiny", Music: "tiny"},
		{Name: "Spooky", Music: "big", Layout: "Jan", Match: "Oct"},
	})
	lib := newLibrary()
	now := t0
	if err := r.Sync(now, lib); err != nil {
		t.Fatal(err)
	}

	stations := lib.byName()
	if len(stations) != 2 {
		t.Fatalf("stations %v", stations)
	}
	big := stations["Big"]
	if big.Type != music.TypeLiveStation || big.Rank != 0 || big.Image != "big.png" ||
		big.Ref != "Big" || !big.Shared {
		t.Errorf("big %+v", big)
	}
	if stations["Charts"].Rank != 1 {
		t.Errorf("charts rank %d", stations["Charts"].Rank)
	}
	if _, ok := stations["Tiny"]; ok {
		t.Error("tiny station has too little music")
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
	if lib.byName()["Big"].ID != big.ID {
		t.Error("station ID changed")
	}

	// in season: the seasonal station appears, in lineup order
	oct := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	if err := r.Sync(oct, lib); err != nil {
		t.Fatal(err)
	}
	if s, ok := lib.byName()["Spooky"]; !ok || s.Rank != 1 {
		t.Fatalf("spooky %+v", s)
	}
	if lib.byName()["Charts"].Rank != 2 {
		t.Error("charts not moved down")
	}
}

func TestSyncRemoved(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: "big"}})
	lib := newLibrary()
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	r.config.Radio.Lineup = []config.RadioEntry{{Name: "Other", Music: "big"}}
	if err := r.Sync(t0.Add(time.Hour), lib); err != nil {
		t.Fatal(err)
	}
	if _, ok := lib.byName()["Big"]; ok {
		t.Error("removed station kept")
	}
	if _, ok := lib.byName()["Other"]; !ok {
		t.Error("new station missing")
	}
	if len(r.history("Big")) != 0 {
		t.Error("removed station's schedule kept")
	}
}

func TestExtendDue(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: "big"}})
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
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: "big"}})
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
	cfg.Radio.Lineup = []config.RadioEntry{{Name: "Rock", Music: "my rock"}, {Name: "Empty"}}
	cfg.Radio.Genres = []string{"rock", "", "jazz"}
	cfg.Radio.Decades = []int{1980}
	cfg.Radio.Series = []string{"Hot 100"}
	list := lineup(cfg)
	var names []string
	for _, s := range list {
		names = append(names, s.Name)
	}
	want := []string{"Rock", "Jazz", "1980s Top Tracks", "Hot 100"}
	if fmt.Sprint(names) != fmt.Sprint(want) || list[0].Query != "my rock" {
		t.Fatalf("got %+v", list)
	}
}

func TestSyncDeletesGenerated(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: "big"}})
	lib := newLibrary()
	lib.add(5, Station{Name: "Big", Type: music.TypeGenre, Ref: "/music/search?q=x&radio=1"})
	lib.add(6, Station{Name: "Gone", Type: music.TypePeriod})
	lib.add(7, Station{Name: "KEXP", Type: music.TypeStream})
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	if _, ok := lib.stations[5]; ok {
		t.Error("old generated station kept")
	}
	if _, ok := lib.stations[6]; ok {
		t.Error("old generated station kept")
	}
	if s := lib.stations[7]; s.Type != music.TypeStream {
		t.Error("stream station changed")
	}
	big, ok := lib.byName()["Big"]
	if !ok || big.Type != music.TypeLiveStation || big.Ref != "Big" {
		t.Errorf("live station not created %+v", big)
	}
	if len(r.history("Big")) == 0 {
		t.Error("live station not scheduled")
	}
}

func TestScheduleLocalTime(t *testing.T) {
	r := makeRadio(t, []config.RadioEntry{{Name: "Big", Music: "big"}})
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
	r := makeRadio(t, []config.RadioEntry{{Name: "KEXP", Music: "big"}, {Name: "Big", Music: "big"}})
	lib := newLibrary()
	lib.add(7, Station{Name: "KEXP", Type: music.TypeStream})
	if err := r.Sync(t0, lib); err != nil {
		t.Fatal(err)
	}
	if s := lib.stations[7]; s.Type != music.TypeStream {
		t.Error("stream station changed")
	}
	if len(r.history("KEXP")) != 0 {
		t.Error("scheduled a station without a row")
	}
	if len(r.history("Big")) == 0 {
		t.Error("other stations not scheduled")
	}
}
