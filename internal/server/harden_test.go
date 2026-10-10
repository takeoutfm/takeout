// Copyright 2023 defsub
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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimitBody(t *testing.T) {
	var readErr error
	var got int
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		got, readErr = len(b), err
	}))

	// under the limit
	req := httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("a", 1024)))
	h.ServeHTTP(httptest.NewRecorder(), req)
	if readErr != nil || got != 1024 {
		t.Fatalf("small body: got %d bytes, err %v", got, readErr)
	}

	// over the limit
	req = httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("a", maxBodyBytes+10)))
	h.ServeHTTP(httptest.NewRecorder(), req)
	if readErr == nil {
		t.Fatal("expected error reading oversized body")
	}
	if got > maxBodyBytes {
		t.Fatalf("read %d bytes, limit is %d", got, maxBodyBytes)
	}

	// GET with no body must still work
	req = httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("GET: got %d", rec.Code)
	}
}

func TestLoginBodyLimitIsSmaller(t *testing.T) {
	h := limitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxLoginBodyBytes)
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, "too big", http.StatusRequestEntityTooLarge)
		}
	}))
	req := httptest.NewRequest("POST", "/login", strings.NewReader(strings.Repeat("a", maxLoginBodyBytes+1)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	hardened(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	for k, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	csp := rec.Header().Get("Content-Security-Policy")
	for _, d := range []string{"frame-ancestors 'none'", "object-src 'none'", "base-uri 'none'"} {
		if !strings.Contains(csp, d) {
			t.Errorf("CSP %q missing %q", csp, d)
		}
	}
	// must not restrict scripts: the bundled UI uses inline handlers
	if strings.Contains(csp, "script-src") || strings.Contains(csp, "default-src") {
		t.Errorf("CSP unexpectedly restricts scripts: %q", csp)
	}
}

func TestNewHTTPServerTimeouts(t *testing.T) {
	s := newHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if s.ReadHeaderTimeout <= 0 || s.ReadTimeout <= 0 || s.IdleTimeout <= 0 {
		t.Fatalf("missing timeouts: %+v", s)
	}
	if s.WriteTimeout != 0 {
		t.Fatal("WriteTimeout must stay unset so media streaming isn't cut off")
	}
}
