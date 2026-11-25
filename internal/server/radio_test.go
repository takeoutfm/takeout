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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/radio"
	"takeoutfm.dev/takeout/model"
	"takeoutfm.dev/takeout/view"
)

// radioLibrary is the real music library with a pool for query "x" only, so
// the default stations find nothing.
type radioLibrary struct {
	radio.MusicLibrary
	pool []radio.Title
}

func (l radioLibrary) Tracks(query string, limit int) []radio.Title {
	if query == "x" {
		return l.pool
	}
	return nil
}

type radioContext struct {
	*TestContext
	r *radio.Radio
}

func (c radioContext) Radio() *radio.Radio { return c.r }

func (c radioContext) FindStation(id string) (model.Station, error) { return c.r.FindStation(id) }

func radioScheduleRequest(t *testing.T, ctx Context, query string) (int, view.RadioSchedule) {
	r := httptest.NewRequest("GET", "https://takeout/api/radio/schedule"+query, nil)
	r = withContext(r, ctx)
	w := httptest.NewRecorder()
	apiRadioSchedule(w, r)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	var s view.RadioSchedule
	json.Unmarshal(body, &s)
	return resp.StatusCode, s
}

func TestApiRadioSchedule(t *testing.T) {
	tc := NewTestContext(t)
	m := tc.Music()

	db, err := gorm.Open(sqlite.Open(tc.Config().Music.DB.Source), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var pool []radio.Title
	for i := 0; i < 60; i++ {
		rid := fmt.Sprintf("radio-rid-%d", i)
		db.Create(&model.Track{UUID: fmt.Sprintf("radio-uuid-%d", i), RID: rid,
			Key: fmt.Sprintf("radio/%d.flac", i), Title: fmt.Sprintf("Song %d", i),
			Artist: fmt.Sprintf("Artist %d", i%20), ETag: "e", Duration: 200})
		pool = append(pool, radio.Title{RID: rid, Artist: fmt.Sprintf("Artist %d", i%20), Seconds: 200})
	}
	t.Cleanup(func() {
		db.Unscoped().Where("r_id like ?", "radio-rid-%").Delete(&model.Track{})
	})

	cfg, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Radio.DB.Source = filepath.Join(t.TempDir(), "radio.db")
	cfg.Radio.Lineup = []config.RadioEntry{
		{Name: "Test FM", Music: "x", Description: "Testing"},
		{Name: "Show FM", Music: "x", Shows: []config.RadioShow{
			{Name: "All Day", Music: "x", Image: "all-day.png", Description: "Every day"}}},
	}
	cfg.Radio.Streams = []config.RadioStream{{Creator: "Test", Title: "Test Stream"}}
	r := radio.NewRadio(cfg)
	if err := r.Open(); err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.Sync(time.Now(), radioLibrary{radio.NewLibrary(m), pool}); err != nil {
		t.Fatal(err)
	}
	ctx := radioContext{tc, r}

	// the station is listed as live
	rv := RadioView(ctx)
	if len(rv.Live) != 2 || rv.Live[0].Name != "Test FM" || rv.Live[0].Description != "Testing" {
		t.Fatalf("live %+v", rv.Live)
	}
	station := rv.Live[0]
	if len(rv.Stream) != 1 || rv.Stream[0].Name != "Test Stream" {
		t.Fatalf("stream %+v", rv.Stream)
	}

	// search finds stations by name, or with a radio: query
	if got := searchRadio(ctx, "test fm"); len(got) != 1 || got[0].ID != station.ID {
		t.Errorf("search %+v", got)
	}
	if got := searchRadio(ctx, "radio:test"); len(got) != 2 {
		t.Errorf("radio search %+v", got)
	}
	if got := searchRadio(ctx, "artist:test"); len(got) != 0 {
		t.Errorf("artist search %+v", got)
	}

	code, s := radioScheduleRequest(t, ctx, "")
	if code != 200 {
		t.Fatalf("got %d", code)
	}
	if d := s.End.Sub(s.Start); d != 30*time.Minute {
		t.Errorf("default range %s", d)
	}
	if len(s.Stations) != 2 || s.Stations[0].ID != station.ID || s.Stations[0].Name != "Test FM" {
		t.Fatalf("stations %+v", s.Stations)
	}
	if len(s.Stations[0].Shows) != 0 || s.Stations[0].Tracks[0].Show != "" {
		t.Errorf("show on Test FM %+v", s.Stations[0])
	}
	// an all-day show: every track is in it, and the station names it once
	shows := s.Stations[1]
	if len(shows.Shows) != 1 || shows.Shows[0] != (view.RadioShow{Name: "All Day",
		Image: "all-day.png", Description: "Every day"}) {
		t.Errorf("shows %+v", shows.Shows)
	}
	for _, x := range shows.Tracks {
		if x.Show != "All Day" {
			t.Fatalf("track not in the show %+v", x)
		}
	}
	tracks := s.Stations[0].Tracks
	now := time.UnixMilli(s.Time)
	if len(tracks) < 9 || tracks[0].Start.After(now) || !tracks[0].End.After(now) {
		t.Fatalf("tracks %+v", tracks)
	}
	for _, x := range tracks {
		if x.Location == "" || x.Title == "" || x.Artist == "" || x.ID == 0 {
			t.Fatalf("unresolved %+v", x)
		}
	}

	// end is capped at the horizon
	_, s = radioScheduleRequest(t, ctx, "?end="+time.Now().Add(72*time.Hour).Format(time.RFC3339))
	if s.End.After(time.Now().Add(cfg.Radio.Horizon + time.Minute)) {
		t.Errorf("end not capped: %s", s.End)
	}

	for _, bad := range []string{"?start=now", "?end=2020-01-01T00:00:00Z"} {
		if code, _ := radioScheduleRequest(t, ctx, bad); code != 400 {
			t.Errorf("%s: got %d", bad, code)
		}
	}

	// one station's schedule: the same shape, only that station
	code, one := radioStationScheduleRequest(t, ctx, fmt.Sprint(station.ID))
	if code != 200 || len(one.Stations) != 1 || one.Stations[0].ID != station.ID ||
		len(one.Stations[0].Tracks) != len(tracks) || one.Time == 0 {
		t.Errorf("station schedule %d %+v", code, one)
	}
	if code, _ := radioStationScheduleRequest(t, ctx, fmt.Sprint(rv.Stream[0].ID)); code != 404 {
		t.Errorf("stream schedule: got %d", code)
	}
	if code, _ := radioStationScheduleRequest(t, ctx, "9999"); code != 404 {
		t.Errorf("missing station schedule: got %d", code)
	}

	// live tiles tune in; stream tiles play through the queue
	page := radioPage(t, ctx)
	for _, want := range []string{
		fmt.Sprintf(`data-action="live" data-live="%d" data-name="Test FM"`, station.ID),
		fmt.Sprintf(`data-radio-now="%d"`, station.ID),
		fmt.Sprintf(`data-ref="/music/stations/%d"`, rv.Stream[0].ID),
	} {
		if !strings.Contains(page, want) {
			t.Errorf("radio page lacks %s", want)
		}
	}
	if strings.Contains(page, fmt.Sprintf(`data-ref="/music/stations/%d"`, station.ID)) {
		t.Error("live station plays through the queue")
	}

	// the playlist starts with the current track
	plist := RefreshStation(ctx, &station)
	if len(plist.Spiff.Entries) == 0 || plist.Spiff.Entries[0].Location[0] != tracks[0].Location {
		t.Errorf("playlist %+v", plist.Spiff.Entries)
	}
}

func radioStationScheduleRequest(t *testing.T, ctx Context, id string) (int, view.RadioSchedule) {
	r := httptest.NewRequest("GET", "https://takeout/api/radio/"+id+"/schedule", nil)
	r.SetPathValue(ParamID, id)
	r = withContext(r, ctx)
	w := httptest.NewRecorder()
	apiRadioStationSchedule(w, r)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	var s view.RadioSchedule
	json.Unmarshal(body, &s)
	return resp.StatusCode, s
}

// radioPage renders the Radio page's tiles.
func radioPage(t *testing.T, ctx Context) string {
	var buf bytes.Buffer
	if err := ctx.Template().ExecuteTemplate(&buf, "radio.html", RadioView(ctx)); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestApiRadioScheduleEmpty(t *testing.T) {
	code, s := radioScheduleRequest(t, NewTestContext(t), "")
	if code != 200 || len(s.Stations) != 0 {
		t.Errorf("got %d %+v", code, s)
	}
}
