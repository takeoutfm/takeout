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
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"takeoutfm.dev/takeout/lib/log"
)

const (
	staticPrefix = "/static/"

	// immutableAge is used when the request has the asset's current hash.
	immutableAge = "public, max-age=31536000, immutable"
)

// asset is one embedded static file, held in memory.
type asset struct {
	body  []byte
	gz    []byte // nil when gzip doesn't make it smaller
	hash  string
	ctype string
}

// assets serves the embedded static files from memory. Each file is read,
// hashed and (when it helps) gzipped once at startup. Pages link to them
// with URL(), which adds the hash so browsers can cache them for good.
type assets struct {
	files map[string]*asset
}

var staticAssets = mustAssets(resStatic, "res/static")

func mustAssets(fsys fs.FS, dir string) *assets {
	a, err := newAssets(fsys, dir)
	if err != nil {
		log.Panicln(err)
	}
	return a
}

func newAssets(fsys fs.FS, dir string) (*assets, error) {
	a := &assets{files: make(map[string]*asset)}
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(p, dir+"/")
		sum := sha256.Sum256(body)
		f := &asset{
			body:  body,
			hash:  hex.EncodeToString(sum[:])[:12],
			ctype: assetType(name, body),
		}
		if compressible(f.ctype) {
			gz, err := gzipBytes(body)
			if err != nil {
				return err
			}
			if len(gz) < len(body) {
				f.gz = gz
			}
		}
		a.files[name] = f
		return nil
	})
	return a, err
}

// URL returns the path for the named asset with its content hash.
func (a *assets) URL(name string) string {
	f, ok := a.files[name]
	if !ok {
		log.Printf("unknown asset %s\n", name)
		return staticPrefix + name
	}
	return staticPrefix + name + "?v=" + f.hash
}

func (a *assets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := a.files[strings.TrimPrefix(r.URL.Path, staticPrefix)]
	if !ok {
		notFoundErr(w)
		return
	}

	h := w.Header()
	h.Set("Content-Type", f.ctype)
	h.Add("Vary", "Accept-Encoding")
	if r.URL.Query().Get("v") == f.hash {
		h.Set("Cache-Control", immutableAge)
	} else {
		h.Set("Cache-Control", "no-cache")
	}

	body, etag := f.body, `"`+f.hash+`"`
	if f.gz != nil && acceptsGzip(r) {
		body, etag = f.gz, `"`+f.hash+`-gz"`
		h.Set("Content-Encoding", "gzip")
	}
	h.Set("ETag", etag)

	// both encodings are the same content, so either tag matches
	if strings.Contains(r.Header.Get("If-None-Match"), f.hash) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Length", strconv.Itoa(len(body)))
	w.Write(body)
}

func assetType(name string, body []byte) string {
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		return t
	}
	return http.DetectContentType(body)
}

func compressible(ctype string) bool {
	return strings.HasPrefix(ctype, "text/") ||
		strings.HasPrefix(ctype, "image/svg+xml") ||
		strings.Contains(ctype, "javascript") ||
		strings.Contains(ctype, "json")
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// acceptsGzip reports whether the request's Accept-Encoding allows gzip.
func acceptsGzip(r *http.Request) bool {
	for _, e := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		coding, params, _ := strings.Cut(strings.TrimSpace(e), ";")
		if coding != "gzip" && coding != "*" {
			continue
		}
		q := strings.ReplaceAll(strings.TrimSpace(params), " ", "")
		return q != "q=0" && q != "q=0.0" && q != "q=0.00" && q != "q=0.000"
	}
	return false
}
