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

// Package client provides an (partial) implementation of Takeout API with
// support for authentication and tokens.
package client

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/takeoutfm/takeout/model"
	"github.com/takeoutfm/takeout/spiff"
	"github.com/takeoutfm/takeout/view"
)

const (
	bearerNone = iota
	bearerCode
	bearerAccess
	bearerRefresh
	bearerMedia
)

type Context interface {
	Endpoint() string
	UserAgent() string
	Transport() http.RoundTripper
	Code() string
	CodeToken() string
	AccessToken() string
	RefreshToken() string
	MediaToken() string
	UpdateAccessToken(string)
}

type request struct {
	context Context
	bearer  int
}

func with(context Context, bearer int) requestContext {
	return request{context: context, bearer: bearer}
}

func (r request) Endpoint() string {
	return r.context.Endpoint()
}

func (r request) Transport() http.RoundTripper {
	return r.context.Transport()
}

func (r request) Headers() Headers {
	headers := NewHeaders()
	headers.UserAgent(r.context.UserAgent())
	switch r.bearer {
	case bearerCode:
		headers.Authorization(r.context.CodeToken())
	case bearerAccess:
		headers.Authorization(r.context.AccessToken())
	case bearerRefresh:
		headers.Authorization(r.context.RefreshToken())
	case bearerMedia:
		headers.Authorization(r.context.MediaToken())
	}
	return headers
}

type AccessCode struct {
	AccessToken string
	Code        string
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
	MediaToken   string `json:",omitempty"`
}

type codeCheck struct {
	Code string
}

func Code(context Context) (*AccessCode, error) {
	var result AccessCode
	if err := Get(with(context, bearerNone), "/api/code", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func CheckCode(context Context) (*Tokens, error) {
	var tokens Tokens
	check := codeCheck{Code: context.Code()}
	err := Post(with(context, bearerCode), "/api/code", &check, &tokens)
	if err != nil {
		return nil, err
	}
	return &tokens, nil
}

func Home(context Context) (*view.Home, error) {
	var result view.Home
	err := get(context, "/api/home", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Radio(context Context) (*view.Radio, error) {
	var result view.Radio
	err := get(context, "/api/radio", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Playlist(context Context) (*spiff.Playlist, error) {
	var result spiff.Playlist
	err := get(context, "/api/playlist", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Locate(context Context, uri string) (*url.URL, error) {
	return GetLocation(with(context, bearerMedia), uri)
}

func Progress(context Context) (*view.Progress, error) {
	var result view.Progress
	err := get(context, "/api/progress", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func SearchReplace(context Context, query string, shuffle, best bool) (*spiff.Playlist, error) {
	var result spiff.Playlist
	var radio string
	var match string
	if shuffle {
		radio = "&radio=1"
	}
	if best {
		match = "&m=1"
	}
	data := patchReplace(
		strings.Join([]string{"/music/search?q=", url.QueryEscape(query), radio, match}, ""),
		spiff.TypeMusic, "", "")
	err := patch(context, "/api/playlist", data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Replace(context Context, ref, spiffType, creator, title string) (*spiff.Playlist, error) {
	var result spiff.Playlist
	data := patchReplace(ref, spiffType, creator, title)
	err := patch(context, "/api/playlist", data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Position(context Context, index int, position float64) error {
	var result spiff.Playlist
	data := patchPosition(index, position)
	err := patch(context, "/api/playlist", data, &result)
	return err
}

func Activity(context Context, activity model.Events) error {
	// TODO need to check/use result
	var result map[string]string
	err := post(context, "/api/activity", activity, &result)
	return err
}

func get(context Context, uri string, result interface{}) error {
	call := func() error {
		return Get(with(context, bearerAccess), uri, result)
	}
	err := call()
	if err == ErrUnauthorized {
		err = refresh(context)
		if err == nil {
			err = call()
		}
	}
	return err
}

func post(context Context, uri string, data, result interface{}) error {
	call := func() error {
		return Post(with(context, bearerAccess), uri, data, result)
	}
	err := call()
	if err == ErrUnauthorized {
		err = refresh(context)
		if err == nil {
			err = call()
		}
	}
	return err
}

func patch(context Context, uri string, data, result interface{}) error {
	call := func() error {
		return Patch(with(context, bearerAccess), uri, data, result)
	}
	err := call()
	if err == ErrUnauthorized {
		err = refresh(context)
		if err == nil {
			err = call()
		}
	}
	return err
}

func refresh(context Context) error {
	var tokens Tokens
	err := Get(with(context, bearerRefresh), "/api/token", &tokens)
	if err != nil {
		return err
	}
	context.UpdateAccessToken(tokens.AccessToken)
	return nil
}
