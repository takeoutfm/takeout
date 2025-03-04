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

package client // import "takeoutfm.dev/takeout/client"

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

type requestContext interface {
	Endpoint() string
	Headers() Headers
	Transport() http.RoundTripper
}

var (
	HeaderAuthorization = http.CanonicalHeaderKey("Authorization")
	HeaderContentType   = http.CanonicalHeaderKey("Content-Type")
	HeaderLocation      = http.CanonicalHeaderKey("Location")
	HeaderUserAgent     = http.CanonicalHeaderKey("User-Agent")

	ErrClientError   = errors.New("client error")
	ErrForbidden     = errors.New("forbidden")
	ErrNoRedirection = errors.New("no redirection")
	ErrServerError   = errors.New("server error")
	ErrUnauthorized  = errors.New("unauthorized")
)

func Get(context requestContext, uri string, result interface{}) error {
	req, err := http.NewRequest(http.MethodGet, endpoint(context, uri), nil)
	if err != nil {
		return err
	}
	return doJson(context, req, result)
}

func GetLocation(context requestContext, uri string) (*url.URL, error) {
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

func Post(context requestContext, uri string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint(context, uri), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	return doJson(context, req, result)
}

func Patch(context requestContext, uri string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, endpoint(context, uri), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	return doJson(context, req, result)
}

func endpoint(context requestContext, uri string) string {
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

func doJson(context requestContext, req *http.Request, result interface{}) error {
	resp, err := do(context, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(result); err != nil {
			return err
		}
	}
	return nil
}

func do(context requestContext, req *http.Request) (*http.Response, error) {
	client := http.Client{Transport: context.Transport()}
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
