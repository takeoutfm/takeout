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

package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/takeoutfm/takeout/model"
	"github.com/takeoutfm/takeout/spiff"
	"github.com/takeoutfm/takeout/view"
)

type testApiContext struct {
	t            *testing.T
	h            Headers
	code         string
	codeToken    string
	accessToken  string
	refreshToken string
	mediaToken   string
}

func newTestApiContext(t *testing.T) *testApiContext {
	return &testApiContext{
		t: t,
		h: NewHeaders(),
	}
}

func (t *testApiContext) Headers() Headers {
	return t.h
}

func (t *testApiContext) UserAgent() string {
	return "test-agent/1.0"
}

func (t *testApiContext) Endpoint() string {
	return "https://localhost:8888"
}

func (t *testApiContext) Transport() http.RoundTripper {
	return testApiServer{t: t.t}
}

func (t *testApiContext) Code() string {
	return t.code
}

func (t *testApiContext) CodeToken() string {
	return t.codeToken
}

func (t *testApiContext) AccessToken() string {
	return t.accessToken
}

func (t *testApiContext) RefreshToken() string {
	return t.refreshToken
}

func (t *testApiContext) MediaToken() string {
	return t.mediaToken
}

func (t *testApiContext) UpdateAccessToken(accessToken string) {
	t.accessToken = accessToken
}

type testApiServer struct {
	t *testing.T
}

func bearer(r *http.Request) string {
	s := strings.Split(r.Header.Get("Authorization"), " ")
	if len(s) != 2 {
		return ""
	}
	if s[0] != "Bearer" {
		return ""
	}
	return s[1]
}

func (s testApiServer) RoundTrip(r *http.Request) (*http.Response, error) {
	headers := make(http.Header)
	if r.URL.Path == "/api/code" {
		headers.Add("Content-type", "application/json")
		if r.Method == "GET" {
			result := AccessCode{
				AccessToken: "6c1c74b8-ff55-4b2c-a9a7-e370d10dcffa",
				Code:        "A1B2C3",
			}
			data, _ := json.Marshal(result)
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
				Header:     headers,
			}, nil
		} else if r.Method == "POST" {
			if bearer(r) == "" {
				return &http.Response{
					StatusCode: 401,
					Header:     headers,
				}, nil
			}
			result := Tokens{
				AccessToken:  "235f9fc2-279c-4ba8-a533-61dd771f4257",
				RefreshToken: "e8950bb9-b87a-474c-9761-06d283c6a8fc",
				MediaToken:   "14b61c79-786f-4d71-a93b-c4c6216b16f1",
			}
			data, _ := json.Marshal(result)
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
				Header:     headers,
			}, nil
		}
	} else if r.URL.Path == "/api/token" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 500,
				Header:     headers,
			}, nil
		}
		if bearer(r) == "test-refresh-token" {
			result := Tokens{
				AccessToken:  "235f9fc2-279c-4ba8-a533-61dd771f4257",
				RefreshToken: "e8950bb9-b87a-474c-9761-06d283c6a8fc",
				MediaToken:   "14b61c79-786f-4d71-a93b-c4c6216b16f1",
			}
			data, _ := json.Marshal(result)
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
				Header:     headers,
			}, nil
		}
	} else if r.URL.Path == "/api/home" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		if bearer(r) == "test-expired-token" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		headers.Add("Content-type", "application/json")
		result := view.Home{
		}
		data, _ := json.Marshal(result)
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/api/radio" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		headers.Add("Content-type", "application/json")
		result := view.Radio{
		}
		data, _ := json.Marshal(result)
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/api/playlist" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		headers.Add("Content-type", "application/json")
		if r.Method == "GET" {
			result := spiff.Playlist{
				Index: 3,
				Position: 10.5,
			}
			data, _ := json.Marshal(result)
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
				Header:     headers,
			}, nil
		} else if r.Method == "PATCH" {
			result := spiff.Playlist{
				Index: 33,
				Position: 123.5,
			}
			data, _ := json.Marshal(result)
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
				Header:     headers,
			}, nil
		}
	} else if r.URL.Path == "/api/tracks/test-track-uuid/location" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		headers.Add("Location", "https://test-redirect-to-location/")
		return &http.Response{
			StatusCode: 307,
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/api/progress" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		headers.Add("Content-type", "application/json")
		result := view.Progress{
		}
		data, _ := json.Marshal(result)
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBuffer(data)),
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/api/activity" {
		if bearer(r) == "" {
			return &http.Response{
				StatusCode: 401,
				Header:     headers,
			}, nil
		}
		if r.Method == "POST" {
			var events model.Events
			body, _ := ioutil.ReadAll(r.Body)
			err := json.Unmarshal(body, &events)
			if err != nil {
				return &http.Response{
					StatusCode: 400,
					Header:     headers,
				}, nil
			}
			return &http.Response{
				StatusCode: 204,
				Header:     headers,
			}, nil
		}
	}
	return &http.Response{
		StatusCode: 500,
		Header:     headers,
	}, nil
}

func TestCode(t *testing.T) {
	c := newTestApiContext(t)
	code, err := Code(c)
	if err != nil {
		t.Fatal(err)
	}
	if code.AccessToken == "" {
		t.Error("exepct code token")
	}
	if code.Code == "" {
		t.Error("exepct code")
	}

	c.code = code.Code
	c.codeToken = code.AccessToken
	tokens, err := CheckCode(c)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken == "" {
		t.Error("expect accessToken")
	}
	if tokens.MediaToken == "" {
		t.Error("expect mediaToken")
	}
	if tokens.RefreshToken == "" {
		t.Error("expect refreshToken")
	}
}

func TestHome(t *testing.T) {
	c := newTestApiContext(t)
	_, err := Home(c)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	_, err = Home(c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRadio(t *testing.T) {
	c := newTestApiContext(t)
	_, err := Radio(c)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	_, err = Radio(c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPlaylist(t *testing.T) {
	c := newTestApiContext(t)
	_, err := Playlist(c)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	playlist, err := Playlist(c)
	if err != nil {
		t.Fatal(err)
	}
	if playlist.Index != 3 {
		t.Error("expect index")
	}
	if playlist.Position != 10.5 {
		t.Error("expect position")
	}
}

func TestLocate(t *testing.T) {
	c := newTestApiContext(t)
	_, err := Locate(c, "/api/tracks/test-track-uuid/location")
	if err == nil {
		t.Error("expect error")
	}

	c.mediaToken = "test-media-token"

	u, err := Locate(c, "/api/tracks/test-track-uuid/location")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(u.String(), "https://") == false {
		t.Error("expect redirect location")
	}
}

func TestProgress(t *testing.T) {
	c := newTestApiContext(t)
	_, err := Progress(c)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	_, err = Progress(c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchReplace(t *testing.T) {
	c := newTestApiContext(t)
	_, err := SearchReplace(c, "test-query", true, true)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	playlist, err := SearchReplace(c, "test-query", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if playlist.Index != 33 {
		t.Error("expect index")
	}
	if playlist.Position != 123.5 {
		t.Error("expect position")
	}
}

func TestReplace(t *testing.T) {
	c := newTestApiContext(t)
	_, err := Replace(c, "test-ref", "music", "test-creator", "test-title")
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	playlist, err := Replace(c, "test-ref", "music", "test-creator", "test-title")
	if err != nil {
		t.Fatal(err)
	}
	if playlist.Index != 33 {
		t.Error("expect index")
	}
	if playlist.Position != 123.5 {
		t.Error("expect position")
	}
}

func TestPosition(t *testing.T) {
	c := newTestApiContext(t)
	err := Position(c, 3, 33.3)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	err = Position(c, 3, 33.3)
	if err != nil {
		t.Fatal(err)
	}
}

func TestActivity(t *testing.T) {
	c := newTestApiContext(t)

	var events model.Events

	err := Activity(c, events)
	if err == nil {
		t.Error("expect error")
	}

	c.accessToken = "test-access-token"
	c.refreshToken = "test-refresh-token"
	err = Activity(c, events)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRefresh(t *testing.T) {
	c := newTestApiContext(t)

	c.accessToken = "test-expired-token"
	c.refreshToken = "test-refresh-token"
	_, err := Home(c)
	if err != nil {
		t.Fatal(err)
	}

	if c.accessToken == "test-expired-token" {
		t.Error("expired new access token")
	}
}
