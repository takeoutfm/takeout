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
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/guide"
	"takeoutfm.dev/takeout/model"
	"takeoutfm.dev/takeout/view"
)

type guideLibrary struct{}

func (guideLibrary) Movies(query string, limit int) []guide.Title {
	var titles []guide.Title
	for i := 1; i <= 30; i++ {
		titles = append(titles, guide.Title{Kind: model.ProgramMovie, TMID: int64(9100 + i), Seconds: 2 * 3600})
	}
	return titles
}

func (guideLibrary) Shows(query string, limit int) []guide.Title { return nil }

func (guideLibrary) CollectionCounts(tmids []int64) map[string]int     { return nil }
func (guideLibrary) Collection(name string) []guide.Title              { return nil }
func (guideLibrary) RecentMovies(source string) []guide.Title          { return nil }
func (guideLibrary) BreakTracks(query string, limit int) []guide.Title { return nil }

type guideContext struct {
	*TestContext
	ch *guide.Guide
}

func (c guideContext) Guide() *guide.Guide { return c.ch }

func guideRequest(t *testing.T, ctx Context, query string) (int, view.Guide) {
	r := httptest.NewRequest("GET", "https://takeout/api/guide"+query, nil)
	r = withContext(r, ctx)
	w := httptest.NewRecorder()
	apiGuide(w, r)
	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	var g view.Guide
	json.Unmarshal(body, &g)
	return resp.StatusCode, g
}

func TestApiGuide(t *testing.T) {
	tc := NewTestContext(t)
	tc.Film() // creates the movie tables

	cfg, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Guide.DB.Source = filepath.Join(t.TempDir(), "guide.db")
	cfg.Guide.Lineup = []config.LineupEntry{{Name: "Movies", Movies: "all"}}
	ch := guide.NewGuide(cfg)
	if err := ch.Open(); err != nil {
		t.Fatal(err)
	}
	defer ch.Close()
	if err := ch.Sync(time.Now(), guideLibrary{}); err != nil {
		t.Fatal(err)
	}

	// half the scheduled movies are in the library, the rest are off air
	db, err := gorm.Open(sqlite.Open(tc.Config().Film.DB.Source), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 15; i++ {
		db.Create(&model.Movie{TMID: int64(9100 + i), Title: "Movie", Key: "m.mkv", ETag: "x"})
	}

	ctx := guideContext{tc, ch}

	code, g := guideRequest(t, ctx, "")
	if code != 200 {
		t.Fatalf("got %d", code)
	}
	if len(g.Channels) != 1 || g.Channels[0].Name != "Movies" {
		t.Fatalf("channels %+v", g.Channels)
	}
	if d := g.End.Sub(g.Start); d != 3*time.Hour {
		t.Errorf("default range %s", d)
	}
	now := time.UnixMilli(g.Time)
	programs := g.Channels[0].Programs
	if programs[0].Start.After(now) || !programs[0].End.After(now) {
		t.Errorf("first program %s-%s not on now", programs[0].Start, programs[0].End)
	}

	// a whole day has both resolved and off air programs
	start := now.Truncate(time.Hour)
	q := "?start=" + start.Format(time.RFC3339) + "&end=" + start.Add(24*time.Hour).Format(time.RFC3339)
	_, g = guideRequest(t, ctx, q)
	var found, offair int
	for _, p := range g.Channels[0].Programs {
		switch p.Kind {
		case model.ProgramMovie:
			found++
			if p.Location == "" || p.Title != "Movie" || p.TargetID == 0 {
				t.Errorf("unresolved movie %+v", p)
			}
		case view.ProgramOffAir:
			offair++
			if p.Location != "" {
				t.Errorf("off air with location")
			}
		}
	}
	if found == 0 || offair == 0 {
		t.Errorf("found %d off air %d", found, offair)
	}

	// end is capped at the horizon
	_, g = guideRequest(t, ctx, "?end="+time.Now().Add(30*24*time.Hour).Format(time.RFC3339))
	if g.End.After(time.Now().Add(cfg.Guide.Horizon + time.Minute)) {
		t.Errorf("end not capped: %s", g.End)
	}

	for _, bad := range []string{"?start=yesterday", "?end=2020-01-01T00:00:00Z"} {
		if code, _ := guideRequest(t, ctx, bad); code != 400 {
			t.Errorf("%s: got %d", bad, code)
		}
	}
}

func TestApiGuideNoChannels(t *testing.T) {
	code, g := guideRequest(t, NewTestContext(t), "")
	if code != 200 || len(g.Channels) != 0 {
		t.Errorf("got %d %+v", code, g)
	}
}
