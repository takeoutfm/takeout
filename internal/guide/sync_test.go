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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

type fakeLibrary map[string][]Title

func (l fakeLibrary) Movies(query string, limit int) []Title { return l[query] }
func (l fakeLibrary) Shows(query string, limit int) []Title  { return l[query] }

// Recent movies are the keys "recent:added" and "recent:released".
func (l fakeLibrary) RecentMovies(source string) []Title { return l["recent:"+source] }

// Collections are the keys "collection:<name>".
func (l fakeLibrary) Collection(name string) []Title { return l["collection:"+name] }

func (l fakeLibrary) CollectionCounts(tmids []int64) map[string]int {
	in := make(map[int64]bool)
	for _, id := range tmids {
		in[id] = true
	}
	counts := make(map[string]int)
	for k, titles := range l {
		name, ok := strings.CutPrefix(k, "collection:")
		if !ok {
			continue
		}
		for _, t := range titles {
			if in[t.TMID] {
				counts[name]++
			}
		}
	}
	return counts
}

func makeGuide(t *testing.T, lineup []config.LineupEntry) *Guide {
	cfg, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Guide.DB.Source = filepath.Join(t.TempDir(), "guide.db")
	cfg.Guide.Timezone = "America/Los_Angeles"
	cfg.Guide.Lineup = lineup
	c := NewGuide(cfg)
	if err := c.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func testLibrary() fakeLibrary {
	var shows []Title
	shows = append(shows, series(10, 4, 12, 22*60)...)
	shows = append(shows, series(11, 6, 22, 44*60)...)
	return fakeLibrary{
		"genre:action": movies(80, 100*60),
		"genre:comedy": shows,
		"tiny":         movies(5, 100*60),
	}
}

func lineup() []config.LineupEntry {
	return []config.LineupEntry{
		{Name: "Spooky", Layout: "Jan", Match: "Oct", Movies: "genre:action"},
		{Name: "Action", Movies: "genre:action"},
		{Name: "Tiny", Movies: "tiny"},
		{Name: "Sitcoms", Shows: "genre:comedy", Align: 30 * time.Minute},
		{Name: "More Action", Movies: "genre:action", MinBreak: 3 * time.Minute},
	}
}

func TestSync(t *testing.T) {
	c := makeGuide(t, lineup())
	lib := testLibrary()
	now := at("2026-11-11 19:07:00")

	if c.HasGuide() {
		t.Error("channels before sync")
	}
	if err := c.Sync(now, lib); err != nil {
		t.Fatal(err)
	}

	guide := c.Listings(now, now.Add(3*time.Hour), nil)
	var names []string
	for _, g := range guide {
		names = append(names, g.Channel.Name)
	}
	if want := []string{"Action", "Sitcoms", "More Action"}; !equal(names, want) {
		t.Fatalf("guide channels %v want %v", names, want)
	}
	for _, g := range guide {
		first := g.Programs[0]
		if first.Start.After(now) || !first.End.After(now) {
			t.Errorf("%s: nothing on now, first %s-%s", g.Channel.Name, first.Start, first.End)
		}
	}

	// the schedule reaches the horizon
	horizon := now.Add(c.Horizon())
	for _, g := range c.Listings(now, horizon, nil) {
		last := g.Programs[len(g.Programs)-1]
		if last.Start.Before(horizon.Add(-3 * time.Hour)) {
			t.Errorf("%s ends early at %s", g.Channel.Name, last.Start)
		}
	}

	// no movie on two channels at once
	all := c.airing(now)
	for i, p := range all {
		for _, q := range all[i+1:] {
			if p.ChannelID != q.ChannelID && p.Kind == ProgramMovie && p.TMID == q.TMID &&
				p.Start.Before(q.End) && q.Start.Before(p.End) {
				t.Errorf("movie %d on two channels at %s", p.TMID, p.Start)
			}
		}
	}

	// re-running changes nothing already scheduled and continues seamlessly
	before := c.Listings(now, horizon, nil)
	later := now.Add(6 * time.Hour)
	if err := c.Sync(later, lib); err != nil {
		t.Fatal(err)
	}
	after := c.Listings(now, later.Add(c.Horizon()), nil)
	for i, g := range before {
		a := after[i].Programs
		for j, p := range g.Programs {
			if a[j].ID != p.ID || !a[j].Start.Equal(p.Start) {
				t.Fatalf("%s program %d changed", g.Channel.Name, j)
			}
		}
		if len(a) <= len(g.Programs) {
			t.Errorf("%s not extended", g.Channel.Name)
		}
		for j := 1; j < len(a); j++ {
			if a[j].Start.Before(a[j-1].End) {
				t.Errorf("%s overlap at %s", g.Channel.Name, a[j].Start)
			}
		}
	}

	if !c.HasGuide() {
		// HasGuide uses the real clock; the test schedule is in 2026-11
		if time.Now().Before(horizon) {
			t.Error("expected channels")
		}
	}

	// old programs are removed after a day
	future := now.Add(3 * 24 * time.Hour)
	if err := c.Sync(future, lib); err != nil {
		t.Fatal(err)
	}
	if old := c.Listings(now, now.Add(time.Hour), nil); len(old) != 0 {
		t.Errorf("old programs kept: %d channels", len(old))
	}
}

func TestSyncSeasonal(t *testing.T) {
	c := makeGuide(t, lineup())
	now := at("2026-10-27 12:00:00")
	if err := c.Sync(now, testLibrary()); err != nil {
		t.Fatal(err)
	}
	// Spooky is on until Halloween
	g := c.Listings(at("2026-10-31 20:00:00"), at("2026-10-31 21:00:00"), nil)
	if len(g) == 0 || g[0].Channel.Name != "Spooky" {
		t.Fatalf("Spooky not first on Halloween")
	}
	g = c.Listings(at("2026-11-01 12:00:00"), at("2026-11-01 13:00:00"), nil)
	if len(g) > 0 && g[0].Channel.Name == "Spooky" {
		t.Error("Spooky on in November")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type fakeResolver struct{}

func (fakeResolver) Movie(tmid int64) (Movie, bool) {
	return Movie{TMID: tmid, Title: "Movie"}, tmid == 1
}

func (fakeResolver) Episode(tvid int64, season, episode int) (TVSeries, TVEpisode, bool) {
	return TVSeries{TVID: tvid, Name: "Show"},
		TVEpisode{TVID: tvid, Season: season, Episode: episode, Name: "Pilot"}, tvid == 7
}

func TestResolve(t *testing.T) {
	r := fakeResolver{}
	e := resolve(Program{Kind: ProgramMovie, TMID: 1}, r)
	if e.OffAir() || e.Movie.Title != "Movie" {
		t.Errorf("movie %+v", e)
	}
	e = resolve(Program{Kind: ProgramEpisode, TVID: 7, Season: 1, Episode: 1}, r)
	if e.OffAir() || e.Series.Name != "Show" || e.TVEpisode.Name != "Pilot" || e.Episode != 1 {
		t.Errorf("episode %+v", e)
	}
	if e := resolve(Program{Kind: ProgramMovie, TMID: 2}, r); !e.OffAir() {
		t.Error("missing movie on air")
	}
	if e := resolve(Program{Kind: ProgramEpisode, TVID: 8}, r); !e.OffAir() {
		t.Error("missing episode on air")
	}
	if e := resolve(Program{Kind: ProgramMovie, TMID: 1}, nil); !e.OffAir() {
		t.Error("nil resolver resolved")
	}
}
