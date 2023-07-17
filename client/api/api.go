// Copyright 2023 defsub
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

package api

import (
	"net/url"
	"strings"

	"github.com/takeoutfm/takeout/client/api/request"
	"github.com/takeoutfm/takeout/lib/spiff"
	"github.com/takeoutfm/takeout/view"
)

const (
	BearerNone = iota
	BearerCode
	BearerAccess
	BearerRefresh
	BearerMedia
)

type ApiContext interface {
	Endpoint() string
	UserAgent() string
	Code() string
	CodeToken() string
	AccessToken() string
	RefreshToken() string
	MediaToken() string
	UpdateAccessToken(string)
}

type context struct {
	api    ApiContext
	bearer int
}

func with(api ApiContext, bearer int) request.Context {
	return context{api: api, bearer: bearer}
}

func (c context) Endpoint() string {
	return c.api.Endpoint()
}

func (c context) Headers() request.Headers {
	headers := request.NewHeaders()
	headers.UserAgent(c.api.UserAgent())
	switch c.bearer {
	case BearerCode:
		headers.Authorization(c.api.CodeToken())
	case BearerAccess:
		headers.Authorization(c.api.AccessToken())
	case BearerRefresh:
		headers.Authorization(c.api.RefreshToken())
	case BearerMedia:
		headers.Authorization(c.api.MediaToken())
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

func Code(context ApiContext) (*AccessCode, error) {
	var result AccessCode
	if err := request.Get(with(context, BearerNone), "/api/code", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func CheckCode(context ApiContext) (*Tokens, error) {
	var tokens Tokens
	check := codeCheck{Code: context.Code()}
	err := request.Post(with(context, BearerCode), "/api/code", &check, &tokens)
	if err != nil {
		return nil, err
	}
	return &tokens, nil
}

func Home(context ApiContext) (*view.Home, error) {
	var result view.Home
	err := get(context, "/api/home", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Playlist(context ApiContext) (*spiff.Playlist, error) {
	var result spiff.Playlist
	err := get(context, "/api/playlist", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Locate(context ApiContext, uri string) (*url.URL, error) {
	return request.GetLocation(with(context, BearerMedia), uri)
}

func Progress(context ApiContext) (*view.Progress, error) {
	var result view.Progress
	err := get(context, "/api/progress", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Replace(context ApiContext, query string, shuffle bool) (*spiff.Playlist, error) {
	var result spiff.Playlist
	var radio string
	if shuffle {
		radio = "&radio=1"
	}
	data := patchReplace(
		strings.Join([]string{"/music/search?q=", url.QueryEscape(query), radio}, ""),
		"music", "", "")
	err := patch(context, "/api/playlist", data, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func Position(context ApiContext, index int, position float64) error {
	var result spiff.Playlist
	data := patchPosition(index, position)
	err := patch(context, "/api/playlist", data, &result)
	return err
}

func get(context ApiContext, uri string, result interface{}) error {
	call := func() error {
		return request.Get(with(context, BearerAccess), uri, result)
	}
	err := call()
	if err == request.ErrUnauthorized {
		err = refresh(context)
		if err == nil {
			err = call()
		}
	}
	return err
}

func patch(context ApiContext, uri string, data, result interface{}) error {
	call := func() error {
		return request.Patch(with(context, BearerAccess), uri, data, result)
	}
	err := call()
	if err == request.ErrUnauthorized {
		err = refresh(context)
		if err == nil {
			err = call()
		}
	}
	return err
}

func refresh(context ApiContext) error {
	var tokens Tokens
	err := request.Get(with(context, BearerRefresh), "/api/token", &tokens)
	if err != nil {
		return err
	}
	context.UpdateAccessToken(tokens.AccessToken)
	return nil
}
