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

package request

import (
	"bytes"
	"errors"

	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const (
	BearerAuthorization = "Bearer"
)

type Headers map[string]string

func NewHeaders() Headers {
	return Headers{}
}

func (h Headers) Authorization(bearer string) Headers {
	value := strings.Join([]string{BearerAuthorization, bearer}, " ")
	h[HeaderAuthorization] = value
	return h
}

func (h Headers) UserAgent(value string) Headers {
	h[HeaderUserAgent] = value
	return h
}

type Context interface {
	Endpoint() string
	Headers() Headers
}

var (
	HeaderUserAgent     = http.CanonicalHeaderKey("User-Agent")
	HeaderAuthorization = http.CanonicalHeaderKey("Authorization")
	HeaderLocation      = http.CanonicalHeaderKey("Location")
)

var (
	ErrUnauthorized  = errors.New("Unauthorized")
	ErrForbidden     = errors.New("Forbidden")
	ErrClientError   = errors.New("Client Error")
	ErrServerError   = errors.New("Server Error")
	ErrNoRedirection = errors.New("No Redirection")
)

func Get(context Context, uri string, result interface{}) error {
	req, err := http.NewRequest(http.MethodGet, endpoint(context, uri), nil)
	if err != nil {
		return err
	}
	return doJson(context, req, result)
}

func GetLocation(context Context, uri string) (*url.URL, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint(context, uri), nil)
	if err != nil {
		return nil, err
	}
	resp, err := do(context, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	url, err := resp.Location()
	if err != nil {
		return nil, err
	}
	return url, err
}

func Post(context Context, uri string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	//fmt.Printf("post data is %s\n", string(data))

	req, err := http.NewRequest(http.MethodPost, endpoint(context, uri), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	return doJson(context, req, result)
}

func Patch(context Context, uri string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	//fmt.Printf("patch data is %s\n", string(data))

	req, err := http.NewRequest(http.MethodPatch, endpoint(context, uri), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	return doJson(context, req, result)
}

func endpoint(context Context, uri string) string {
	url := strings.Join([]string{context.Endpoint(), uri}, "")
	return url
}

func applyHeaders(req *http.Request, headers Headers) {
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}
}

func doJson(context Context, req *http.Request, result interface{}) error {
	resp, err := do(context, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(result); err != nil {
		return err
	}

	return nil
}

func do(context Context, req *http.Request) (*http.Response, error) {
	client := http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	applyHeaders(req, context.Headers())
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	err = errorCheck(resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func errorCheck(resp *http.Response) error {
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	} else if resp.StatusCode == 403 {
		return ErrForbidden
	} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return ErrClientError
	} else if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return ErrServerError
	}
	return nil

}
