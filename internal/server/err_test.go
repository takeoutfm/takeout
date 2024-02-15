// Copyright 2024 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type errorHandler bool

func (errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/err/server" {
		serverErr(w, errors.New("server error"))
	} else if r.URL.Path == "/err/bad" {
		badRequest(w, errors.New("bad request"))
	} else if r.URL.Path == "/err/auth" {
		authErr(w, errors.New("auth error"))
	} else if r.URL.Path == "/err/deny" {
		accessDenied(w)
	} else if r.URL.Path == "/err/notfound" {
		notFoundErr(w)
	}
}

func testError(t *testing.T, path string, code int) {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := httptest.NewRecorder()
	handler := http.Handler(errorHandler(true))
	handler.ServeHTTP(result, req)
	if result.Code != code {
		t.Errorf("expect %d got %d", code, result.Code)
	}
}

func TestServerError(t *testing.T) {
	testError(t, "/err/server", 500)
}

func TestBadRequest(t *testing.T) {
	testError(t, "/err/bad", 400)
}

func TestAuthError(t *testing.T) {
	testError(t, "/err/auth", 401)
}

func TestDenyError(t *testing.T) {
	testError(t, "/err/deny", 403)
}

func TestNotFound(t *testing.T) {
	testError(t, "/err/notfound", 404)
}
