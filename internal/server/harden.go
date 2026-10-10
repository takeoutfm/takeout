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
	"net/http"
	"time"
)

const (
	// Largest request body accepted by any endpoint. API bodies are small
	// JSON documents (credentials, playlist patches, activity events).
	// Login endpoints apply the much smaller maxLoginBodyBytes on top.
	maxBodyBytes = 4 << 20

	maxHeaderBytes = 64 << 10
)

// limitBody caps the size of every request body so handlers that read the
// whole body can't be made to buffer an unbounded amount of memory.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// securityHeaders adds response headers that don't change how the UI works.
//
// The Content-Security-Policy deliberately only uses directives that don't
// restrict scripts or styles: the bundled UI builds inline onclick handlers in
// music.js, so a script-src policy would break it. Once those are moved to
// addEventListener a "script-src 'self'" can be added as a second layer.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		// session/media tokens can appear in query strings; don't leak
		// them to other sites through the Referer header
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy",
			"base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

// hardened wraps a handler with the body limit and security headers.
func hardened(next http.Handler) http.Handler {
	return securityHeaders(limitBody(next))
}

// newHTTPServer returns a server with timeouts so slow or stalled clients
// can't hold connections open forever.
//
// WriteTimeout is intentionally not set: it would cut off long media
// downloads/streams served by this process.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}
