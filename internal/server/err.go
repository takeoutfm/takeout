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
	"errors"
	"net/http"
)

var (
	ErrNoMedia              = errors.New("media not available")
	ErrInvalidMethod        = errors.New("invalid request method")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrInvalidCode          = errors.New("invalid code")
	ErrNotFound             = errors.New("not found")
	ErrInvalidOffset        = errors.New("invalid offset")
	ErrAccessDenied         = errors.New("access denied")
	ErrAccessDeniedRedirect = errors.New("access denied with redirect")
	ErrMissingToken         = errors.New("missing token")
	ErrMissingAccessToken   = errors.New("missing access token")
	ErrMissingMediaToken    = errors.New("missing media token")
	ErrMissingFileToken     = errors.New("missing file token")
	ErrMissingCodeToken     = errors.New("missing code token")
	ErrMissingCookie        = errors.New("missing cookie")
	ErrInvalidSession       = errors.New("invalid session")
	ErrInvalidUUID          = errors.New("invalid uuid")
	ErrInvalidContent       = errors.New("invalid content")
	ErrMissingTitle         = errors.New("missing title")
	ErrMissingParameter     = errors.New("missing required parameter")
	ErrInvalidParameter     = errors.New("invalid parameter")
)

func serverErr(w http.ResponseWriter, err error) {
	if err != nil {
		//log.Printf("got err %s\n", err)
		handleErr(w, "bummer", http.StatusInternalServerError)
	}
}

func badRequest(w http.ResponseWriter, err error) {
	if err != nil {
		handleErr(w, err.Error(), http.StatusBadRequest)
	}
}

// client provided no credentials or invalid credentials.
func authErr(w http.ResponseWriter, err error) {
	if err != nil {
		handleErr(w, err.Error(), http.StatusUnauthorized)
	}
}

// client provided credentials but access is not allowed.
func accessDenied(w http.ResponseWriter) {
	handleErr(w, ErrAccessDenied.Error(), http.StatusForbidden)
}

func notFoundErr(w http.ResponseWriter) {
	handleErr(w, ErrNotFound.Error(), http.StatusNotFound)
}

func handleErr(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
}
