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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"takeoutfm.dev/takeout/view"
)

var testCSS = strings.Repeat("body { color: white; }\n", 50)

func testAssets(t *testing.T) *assets {
	fsys := fstest.MapFS{
		"res/static/a.css":    {Data: []byte(testCSS)},
		"res/static/tiny.js":  {Data: []byte("x")},
		"res/static/icon.png": {Data: []byte("\x89PNG\r\n\x1a\n")},
	}
	a, err := newAssets(fsys, "res/static")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func get(a *assets, url string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", url, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	return w
}

func TestAssetURL(t *testing.T) {
	a := testAssets(t)
	u := a.URL("a.css")
	if !strings.HasPrefix(u, "/static/a.css?v=") || len(u) != len("/static/a.css?v=")+12 {
		t.Fatalf("url %s", u)
	}
	if u != testAssets(t).URL("a.css") {
		t.Fatal("hash not stable")
	}
	if a.URL("missing.css") != "/static/missing.css" {
		t.Fatal("unknown asset url")
	}
}

func TestAssetServe(t *testing.T) {
	a := testAssets(t)
	w := get(a, "/static/a.css", nil)
	if w.Code != http.StatusOK || w.Body.String() != testCSS {
		t.Fatalf("code %d body %q", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Fatalf("content type %s", ct)
	}
	if w.Header().Get("Content-Encoding") != "" {
		t.Fatal("gzip without Accept-Encoding")
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("cache control %s", w.Header().Get("Cache-Control"))
	}
}

func TestAssetImmutable(t *testing.T) {
	a := testAssets(t)
	w := get(a, a.URL("a.css"), nil)
	if w.Header().Get("Cache-Control") != immutableAge {
		t.Fatalf("cache control %s", w.Header().Get("Cache-Control"))
	}
	w = get(a, "/static/a.css?v=000000000000", nil)
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Fatal("old hash cached as immutable")
	}
}

func TestAssetGzip(t *testing.T) {
	a := testAssets(t)
	w := get(a, "/static/a.css", map[string]string{"Accept-Encoding": "br, gzip"})
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip")
	}
	if !strings.Contains(w.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatal("expected Vary")
	}
	zr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if string(body) != testCSS {
		t.Fatal("gzip body differs")
	}

	// refused, too small to help, or not text
	cases := []struct{ url, ae string }{
		{"/static/a.css", "gzip;q=0"},
		{"/static/a.css", "identity"},
		{"/static/tiny.js", "gzip"},
		{"/static/icon.png", "gzip"},
	}
	for _, c := range cases {
		w := get(a, c.url, map[string]string{"Accept-Encoding": c.ae})
		if w.Header().Get("Content-Encoding") != "" {
			t.Errorf("%s with %q: unexpected gzip", c.url, c.ae)
		}
	}
}

func TestAssetNotModified(t *testing.T) {
	a := testAssets(t)
	for _, ae := range []string{"", "gzip"} {
		w := get(a, "/static/a.css", map[string]string{"Accept-Encoding": ae})
		etag := w.Header().Get("ETag")
		w = get(a, "/static/a.css", map[string]string{"If-None-Match": etag})
		if w.Code != http.StatusNotModified || w.Body.Len() != 0 {
			t.Fatalf("etag %s: code %d", etag, w.Code)
		}
	}
	w := get(a, "/static/a.css", map[string]string{"If-None-Match": `"000000000000"`})
	if w.Code != http.StatusOK {
		t.Fatalf("stale etag: code %d", w.Code)
	}
}

func TestAssetNotFound(t *testing.T) {
	a := testAssets(t)
	for _, url := range []string{"/static/", "/static/nope.css", "/static/../template/index.html"} {
		if w := get(a, url, nil); w.Code != http.StatusNotFound {
			t.Errorf("%s: code %d", url, w.Code)
		}
	}
}

func TestStaticAssets(t *testing.T) {
	for _, name := range []string{"music.js", "takeout.css", "login.html", "favicon.svg"} {
		if _, ok := staticAssets.files[name]; !ok {
			t.Errorf("missing %s", name)
		}
	}
	var buf bytes.Buffer
	err := getTemplates(nil).ExecuteTemplate(&buf, "index.html", &view.Index{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), staticAssets.URL("music.js")) {
		t.Fatal("index.html doesn't use the hashed music.js url")
	}
}
