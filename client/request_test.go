// Copyright 2024 defsub
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

package client

import (
	"bytes"
	"net/http"
	"io"
	"testing"
)

func TestAuthorization(t *testing.T) {
	h := NewHeaders()
	h.Authorization("test-token")
	if h[HeaderAuthorization] != "Bearer test-token" {
		t.Error("expect bearer test-token")
	}
}

func TestUserAgent(t *testing.T) {
	h := NewHeaders()
	h.UserAgent("test/1.0")
	if h[HeaderUserAgent] != "test/1.0" {
		t.Error("expect user-agent")
	}
}

type testContext struct {
	t *testing.T
	h Headers
}

func newTestContext(t *testing.T) *testContext {
	return &testContext{
		t: t,
		h: NewHeaders(),
	}
}

func (t testContext) Headers() Headers {
	return t.h
}

func (t testContext) Endpoint() string {
	return "https://test.com"
}

func (t testContext) Transport() http.RoundTripper {
	return testServer{t: t.t}
}

type testServer struct {
	t *testing.T
}

type simpleJson struct {
	A string `json:"a"`
}

func (s testServer) RoundTrip(r *http.Request) (*http.Response, error) {
	headers := make(http.Header)
	headers.Add("X-UserAgent", r.Header.Get(HeaderUserAgent))
	if r.URL.Path == "/good" {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"a":"b"}`)),
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/simple-json" {
		headers.Add("Content-type", "application/json")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"a":"b"}`)),
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/location" {
		headers.Add("Location", "https://redirect.com/location")
		return &http.Response{
			StatusCode: 307,
			Body:       io.NopCloser(bytes.NewBufferString(``)),
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/post-echo" {
		headers.Add("Content-type", r.Header.Get(HeaderContentType))
		return &http.Response{
			StatusCode: 200,
			Body:       r.Body,
			Header:     headers,
		}, nil
	} else if r.URL.Path == "/simple-patch" {
		headers.Add("Content-type", "application/json")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"a":"b"}`)),
			Header:     headers,
		}, nil
	} else {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString(`{"a":"b"}`)),
			Header:     headers,
		}, nil
	}
}

func TestDo(t *testing.T) {
	userAgent := "test/1.0"

	c := newTestContext(t)
	c.Headers().UserAgent(userAgent)

	methods := []string{"GET", "POST", "PATCH"}
	for _, m := range methods {
		req, err := http.NewRequest(m, "/good", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := do(c, req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Header.Get("X-UserAgent") != userAgent {
			t.Error("expect user-agent")
		}
	}

	for _, m := range methods {
		req, err := http.NewRequest(m, "/bad", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = do(c, req)
		if err == nil {
			t.Error("expect err")
		}
	}
}

func TestGet(t *testing.T) {
	c := newTestContext(t)
	c.Headers().UserAgent("test/1.0")
	var result simpleJson
	Get(c, "/simple-json", &result)
	if result.A != "b" {
		t.Error("expect json b")
	}
}

func TestGetLocation(t *testing.T) {
	c := newTestContext(t)
	c.Headers().UserAgent("test/1.0")
	u, err := GetLocation(c, "/location")
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "https://redirect.com/location" {
		t.Errorf("expect redirect got %s", u.String())
	}
}

func TestPost(t *testing.T) {
	c := newTestContext(t)
	c.Headers().UserAgent("test/1.0")
	var data simpleJson
	data.A = "x"
	var result simpleJson
	Post(c, "/post-echo", &data, &result)
	if result.A != "x" {
		t.Error("expect json x")
	}
}

func TestPatch(t *testing.T) {
	c := newTestContext(t)
	c.Headers().UserAgent("test/1.0")
	patch := patchClear()
	var result simpleJson
	Patch(c, "/simple-patch", patch, &result)
	if result.A != "b" {
		t.Error("expect json b")
	}
}
