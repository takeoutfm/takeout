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
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/radio"
	"takeoutfm.dev/takeout/model"
	"takeoutfm.dev/takeout/view"
)

// radioLibrary is the real music library with a pool for query "x" only, so
// the default Music.Radio* stations find nothing.
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
		for _, s := range m.TakeoutStations() {
			if s.Type == music.TypeLiveStation {
				m.DeleteStation(&s)
			}
		}
	})

	cfg, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Radio.DB.Source = filepath.Join(t.TempDir(), "radio.db")
	cfg.Radio.Lineup = []config.RadioEntry{{Name: "Test FM", Music: "x", Description: "Testing"}}
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
	if len(rv.Live) != 1 || rv.Live[0].Name != "Test FM" || rv.Live[0].Description != "Testing" {
		t.Fatalf("live %+v", rv.Live)
	}
	station := rv.Live[0]

	code, s := radioScheduleRequest(t, ctx, "")
	if code != 200 {
		t.Fatalf("got %d", code)
	}
	if d := s.End.Sub(s.Start); d != 30*time.Minute {
		t.Errorf("default range %s", d)
	}
	if len(s.Stations) != 1 || s.Stations[0].ID != station.ID || s.Stations[0].Name != "Test FM" {
		t.Fatalf("stations %+v", s.Stations)
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

	// the playlist starts with the current track
	plist := RefreshStation(ctx, &station)
	if len(plist.Spiff.Entries) == 0 || plist.Spiff.Entries[0].Location[0] != tracks[0].Location {
		t.Errorf("playlist %+v", plist.Spiff.Entries)
	}
}

func TestApiRadioScheduleEmpty(t *testing.T) {
	code, s := radioScheduleRequest(t, NewTestContext(t), "")
	if code != 200 || len(s.Stations) != 0 {
		t.Errorf("got %d %+v", code, s)
	}
}
