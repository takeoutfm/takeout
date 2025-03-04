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

// Package client provides an http client used extensively by Takeout for
// syncing media. There's support for local file-based caching and cache-only
// usage.
package client // import "takeoutfm.dev/takeout/lib/client"

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"takeoutfm.dev/takeout/lib/header"
	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/lib/pls"
)

const (
	DirectiveMaxAge       = "max-age"
	DirectiveOnlyIfCached = "only-if-cached"
)

var (
	ErrCacheMiss          = errors.New("cache miss")
	ErrSchemeNotSupported = errors.New("scheme not supported")
)

type RateLimiter interface {
	RateLimit(host string)
}

type Config struct {
	UserAgent string
	CacheDir  string
	MaxAge    time.Duration
}

func (c *Config) Merge(o Config) {
	if o.CacheDir != "" {
		c.CacheDir = o.CacheDir
	}
	c.MaxAge = o.MaxAge
	if o.UserAgent != "" {
		c.UserAgent = o.UserAgent
	}
}

type Getter interface {
	Get(url string) (http.Header, []byte, error)
	GetBody(url string) ([]byte, error)
	GetJson(url string, result interface{}) error
	GetJsonWith(headers map[string]string, url string, result interface{}) error
	GetXML(url string, result interface{}) error
	GetPLS(url string) (pls.Playlist, error)
}

type client struct {
	client      *http.Client
	useCache    bool
	userAgent   string
	cache       httpcache.Cache
	maxAge      time.Duration
	onlyCached  bool
	rateLimiter RateLimiter
}

func NewDefaultGetter() Getter {
	c := &client{}
	c.client = &http.Client{}
	c.rateLimiter = DefaultLimiter
	return c
}

func NewCacheOnlyGetter(config Config) Getter {
	c := &client{}
	c.onlyCached = true
	c.useCache = true
	c.userAgent = config.UserAgent
	c.maxAge = config.MaxAge
	c.cache = diskcache.New(config.CacheDir)
	transport := httpcache.NewTransport(c.cache)
	c.client = transport.Client()
	c.rateLimiter = DefaultLimiter
	return c
}

func NewGetter(config Config) Getter {
	c := &client{}
	c.userAgent = config.UserAgent
	c.useCache = len(config.CacheDir) > 0
	if c.useCache {
		c.maxAge = config.MaxAge
		c.cache = diskcache.New(config.CacheDir)
		transport := httpcache.NewTransport(c.cache)
		c.client = transport.Client()
	} else {
		c.client = &http.Client{}
	}
	c.rateLimiter = DefaultLimiter
	return c
}

func NewTransportGetter(config Config, transport http.RoundTripper) Getter {
	c := &client{}
	c.userAgent = config.UserAgent
	c.useCache = false
	c.client = &http.Client{Transport: transport}
	c.rateLimiter = UnlimitedLimiter
	return c
}

var DefaultLimiter RateLimiter = &timeLimiter{}
var UnlimitedLimiter RateLimiter = unlimitedLimiter(0)

type timeLimiter struct {
	lastRequest sync.Map
}

func (l *timeLimiter) RateLimit(host string) {
	t := time.Now()
	time.Sleep(time.Second)
	l.lastRequest.Store(host, t)
}

type unlimitedLimiter int

func (unlimitedLimiter) RateLimit(host string) {
}

func (c *client) doGet(headers map[string]string, urlStr string) (*http.Response, error) {
	//log.Printf("doGet %s\n", urlStr)
	url, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(header.UserAgent, c.userAgent)
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	throttle := true
	if c.useCache {
		maxAge := int(c.maxAge.Seconds())
		if c.onlyCached {
			req.Header.Set(header.CacheControl, DirectiveOnlyIfCached)
		} else if maxAge > 0 {
			req.Header.Set(header.CacheControl, fmt.Sprintf("%s=%d", DirectiveMaxAge, maxAge))
		}
		// peek into the cache, if there's something there don't slow down
		cachedResp, err := httpcache.CachedResponse(c.cache, req)
		if err != nil {
			log.Printf("cache error %s\n", err)
		}
		if cachedResp != nil {
			throttle = false
			//log.Printf("is cached\n")
		}
	}
	if throttle {
		c.rateLimiter.RateLimit(url.Hostname())
	}

	//log.Printf("get %s\n", req.URL.String())
	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("client.Do err %s\n", err)
		return nil, err
	}

	if c.onlyCached && resp.StatusCode == 504 {
		// the cache returns 504 for cache only miss
		return nil, ErrCacheMiss
	}

	if resp.StatusCode != 200 {
		return resp, errors.New(fmt.Sprintf("http error %d: %s",
			resp.StatusCode, url.String()))
	}

	return resp, err
}

const (
	maxAttempts = 5
	backoff     = time.Second * 3
)

func (c *client) doGetWithRetry(headers map[string]string, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err = c.doGet(headers, url)
		if err == nil || resp == nil {
			// success
			// or error with no response
			break
		}
		if resp.StatusCode < http.StatusInternalServerError &&
			resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		// server error, try again with backoff
		if attempt+1 < maxAttempts {
			log.Printf("got err %d: retry backoff attempt %d of %d\n",
				resp.StatusCode,
				attempt+1,
				maxAttempts)
			time.Sleep(backoff)
		}
	}

	return resp, err
}

func (c *client) GetWith(headers map[string]string, url string) (http.Header, []byte, error) {
	resp, err := c.doGetWithRetry(headers, url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.Header, body, err
}

func (c *client) Get(url string) (http.Header, []byte, error) {
	return c.GetWith(nil, url)
}

func (c *client) GetBody(url string) (body []byte, err error) {
	_, body, err = c.GetWith(nil, url)
	return
}

func (c *client) GetJson(url string, result interface{}) error {
	return c.GetJsonWith(nil, url, result)
}

func (c *client) GetJsonWith(headers map[string]string, url string, result interface{}) error {
	resp, err := c.doGetWithRetry(headers, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(result); err != nil {
		return err
	}
	return nil
}

func (c *client) GetXML(urlString string, result interface{}) error {
	// TODO use only for testing
	// if strings.HasPrefix(urlString, "file:") {
	// 	u, err := url.Parse(urlString)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	data, err := os.ReadFile(u.Path[1:])
	// 	if err != nil {
	// 		return err
	// 	}
	// 	reader := bytes.NewReader(data)
	// 	decoder := xml.NewDecoder(reader)
	// 	if err = decoder.Decode(result); err != nil {
	// 		return err
	// 	}
	// } else {
	resp, err := c.doGet(nil, urlString)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := xml.NewDecoder(resp.Body)
	if err = decoder.Decode(result); err != nil {
		return err
	}
	return nil
}

func (c *client) GetPLS(urlString string) (pls.Playlist, error) {
	resp, err := c.doGet(nil, urlString)
	if err != nil {
		return pls.Playlist{}, err
	}
	defer resp.Body.Close()
	return pls.Parse(resp.Body)

}

func Get(path string) ([]byte, error) {
	if strings.HasPrefix(path, "http") {
		c := NewDefaultGetter()
		return c.GetBody(path)
	} else if strings.HasPrefix(path, "file") {
		u, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(u.Path)
	} else {
		return os.ReadFile(path)
	}
	// return nil, ErrSchemeNotSupported
}
