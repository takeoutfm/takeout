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
	"compress/gzip"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"takeoutfm.dev/takeout/lib/gorm"
	"takeoutfm.dev/takeout/model"
)

// pageMux serves all pages with the test context, without auth.
func pageMux(t *testing.T) http.Handler {
	ctx := NewTestContext(t)
	mux := http.NewServeMux()
	for pattern, p := range pages {
		h := pageHandler(p)
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			h(w, withContext(r, ctx))
		})
	}
	return mux
}

func getPage(h http.Handler, url string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", url, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func vary(w *httptest.ResponseRecorder) string {
	return strings.Join(w.Header().Values("Vary"), ", ")
}

var listPages = []string{
	"/", "/search?q=jazz", "/artists", "/radio", "/playlists",
	"/recently-added", "/new-releases", "/favorites",
	"/movies", "/movie-genres/Drama", "/movie-keywords/heist", "/profiles/1",
	"/tv", "/podcasts", "/activity", "/activity/lastweek",
}

func TestPagesFull(t *testing.T) {
	h := pageMux(t)
	for _, url := range listPages {
		w := getPage(h, url, nil)
		body := w.Body.String()
		if w.Code != http.StatusOK {
			t.Errorf("%s: code %d", url, w.Code)
			continue
		}
		if !strings.HasPrefix(body, "<!DOCTYPE html>") || !strings.Contains(body, `<div class="page" data-title=`) {
			t.Errorf("%s: not a full page", url)
		}
		if !strings.Contains(vary(w), HeaderFragment) ||
			w.Header().Get("Cache-Control") != "private, no-cache" {
			t.Errorf("%s: headers %v", url, w.Header())
		}
	}
}

func TestPagesFragment(t *testing.T) {
	h := pageMux(t)
	for _, url := range listPages {
		w := getPage(h, url, map[string]string{HeaderFragment: "1"})
		body := w.Body.String()
		if w.Code != http.StatusOK {
			t.Errorf("%s: code %d", url, w.Code)
			continue
		}
		if !strings.HasPrefix(body, `<div class="page" data-title=`) || strings.Contains(body, "<html") {
			t.Errorf("%s: not a fragment: %.80q", url, body)
		}
	}
}

func TestPageTitle(t *testing.T) {
	h := pageMux(t)
	w := getPage(h, "/artists", nil)
	if !strings.Contains(w.Body.String(), "<title>Artists - TakeoutFM</title>") {
		t.Error("artists title")
	}
	w = getPage(h, "/", map[string]string{HeaderFragment: "1"})
	if !strings.Contains(w.Body.String(), `data-title="TakeoutFM"`) {
		t.Error("home title")
	}
}

func TestPageSection(t *testing.T) {
	h := pageMux(t)
	// the test library is empty, so only the sections every user has show
	w := getPage(h, "/activity/lastweek", nil)
	body := w.Body.String()
	if !strings.Contains(body, `data-section="activity" aria-current="page"`) {
		t.Error("activity not current in the sidebar")
	}
	if n := len(regexp.MustCompile(`<a class="side"[^>]*aria-current="page"`).FindAllString(body, -1)); n != 1 {
		t.Error("expected exactly one current sidebar item")
	}
	w = getPage(h, "/favorites", map[string]string{HeaderFragment: "1"})
	if !strings.Contains(w.Body.String(), `data-section="favorites"`) {
		t.Error("fragment without its section")
	}
}

func TestPageEscaping(t *testing.T) {
	h := pageMux(t)
	w := getPage(h, "/search?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E", nil)
	body := w.Body.String()
	if strings.Contains(body, "<script>alert") {
		t.Fatal("query not escaped")
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt; - TakeoutFM") {
		t.Fatal("expected escaped title")
	}
}

func TestPagesNotFound(t *testing.T) {
	h := pageMux(t)
	for _, url := range []string{
		"/artists/999999", "/artists/x", "/artists/0", "/artists/999999/popular",
		"/artists/999999/singles", "/artists/999999/want", "/releases/999999",
		"/movies/999999", "/tv/series/999999", "/tv/episodes/999999",
		"/series/999999", "/episodes/999999",
		"/v", "/v?release=1", "/nope", "/movies/1/watch",
	} {
		if w := getPage(h, url, nil); w.Code != http.StatusNotFound {
			t.Errorf("%s: code %d", url, w.Code)
		}
	}
}

// TestPageTemplates renders the pages that need an id, which the empty
// test library can't serve.
func TestPageTemplates(t *testing.T) {
	ctx := NewTestContext(t)
	artist := model.Artist{Name: "test artist"}
	views := map[string]any{
		"artist.html":    ArtistView(ctx, artist),
		"popular.html":   PopularView(ctx, artist),
		"singles.html":   SinglesView(ctx, artist),
		"want.html":      WantListView(ctx, artist),
		"release.html":   ReleaseView(ctx, model.Release{Name: "test release"}),
		"movie.html":     MovieView(ctx, model.Movie{Title: "test movie"}),
		"tvseries.html":  TVSeriesView(ctx, model.TVSeries{Name: "test series"}),
		"tvepisode.html": TVEpisodeView(ctx, model.TVEpisode{Name: "test episode"}),
		"series.html":    SeriesView(ctx, model.Series{Title: "test series"}),
		"episode.html":   EpisodeView(ctx, model.Episode{Title: "test episode"}),
	}
	tmpl := ctx.Template()
	for name, v := range views {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, name, v); err != nil {
			t.Errorf("%s: %s", name, err)
		}
	}
	for _, p := range pages {
		if tmpl.Lookup(p.temp) == nil {
			t.Errorf("missing template %s", p.temp)
		}
	}
}

func TestPageGzip(t *testing.T) {
	h := pageMux(t)
	w := getPage(h, "/", map[string]string{"Accept-Encoding": "gzip"})
	if w.Header().Get("Content-Encoding") != "gzip" || !strings.Contains(vary(w), "Accept-Encoding") {
		t.Fatalf("expected gzip: %v", w.Header())
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.HasPrefix(string(body), "<!DOCTYPE html>") {
		t.Fatal("gzip body")
	}
}

func TestPageLinks(t *testing.T) {
	funcs := doFuncMap()
	link := funcs["link"].(func(any) string)
	cases := map[string]any{
		"/artists/1":     model.Artist{Model: gorm.Model{ID: 1}},
		"/releases/2":    model.Release{Model: gorm.Model{ID: 2}},
		"/movies/3":      model.Movie{Model: gorm.Model{ID: 3}},
		"/tv/series/4":   model.TVSeries{Model: gorm.Model{ID: 4}},
		"/tv/episodes/5": model.TVEpisode{Model: gorm.Model{ID: 5}},
		"/series/6":      model.Series{Model: gorm.Model{ID: 6}},
		"/episodes/7":    model.Episode{Model: gorm.Model{ID: 7}},
	}
	for want, o := range cases {
		if got := link(o); got != want {
			t.Errorf("link %T = %s, want %s", o, got, want)
		}
	}
}

// TestTemplatesNoSelfClosing catches <div/> and the like: HTML has no
// self-closing tags, so the browser leaves the element open and the rest of
// the layout (the player too) ends up inside the page.
func TestTemplatesNoSelfClosing(t *testing.T) {
	selfClosing := regexp.MustCompile(`<(a|div|p|span|section|main|table|tr|td|h[1-6]|ul|li|canvas|audio|video|form|label)\b[^>]*/>`)
	err := fs.WalkDir(resTemplates, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := fs.ReadFile(resTemplates, p)
		if err != nil {
			return err
		}
		for _, m := range selfClosing.FindAll(b, -1) {
			t.Errorf("%s: %s", p, m)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
