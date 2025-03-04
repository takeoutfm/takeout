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

package rss // import "takeoutfm.dev/takeout/lib/pls"

import (
	"bytes"
	"embed"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"takeoutfm.dev/takeout/lib/client"
)

//go:embed test/*.xml
var jsonFiles embed.FS

func jsonFile(name string) string {
	d, err := jsonFiles.ReadFile(name)
	if err != nil {
		return ""
	}
	return string(d)
}

type rssServer struct {
	t *testing.T
}

func (s rssServer) RoundTrip(r *http.Request) (*http.Response, error) {
	var body string
	//s.t.Logf("got %s\n", r.URL.String())
	if strings.HasPrefix(r.URL.Path, "/twit.xml") {
		body = jsonFile("test/twit.xml")
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

func makeClient(t *testing.T) *RSS {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, rssServer{t: t})
	return NewRSS(c)
}

func TestFetch(t *testing.T) {
	r := makeClient(t)
	c, err := r.Fetch("https://feeds.twit.tv/twit.xml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "This Week in Tech (Audio)" {
		t.Error("expect title")
	}
	if c.Author != "TWiT" {
		t.Error("expect author")
	}
	if c.Link() != "https://twit.tv/shows/this-week-in-tech" {
		t.Error("expect link")
	}
	if len(c.Image.Link) == 0 {
		t.Error("expect image link")
	}
	if len(c.Items) == 0 {
		t.Error("expect items")
	}
	for _, i := range c.Items {
		if len(i.Title) == 0 {
			t.Error("expect item title")
		}
		if len(i.Author) == 0 {
			t.Error("expect item author")
		}
		if len(i.Description) == 0 {
			t.Error("expect item description")
		}
		if strings.HasPrefix(i.URL(), "http") == false {
			t.Error("expect item link")
		}
		if len(i.GUID.Value) == 0 {
			t.Error("expect guid value")
		}
		if strings.Contains(i.ItemGUID(), "://") {
			t.Error("expect guid is not url")
		}
		if len(i.PubDate) == 0 {
			t.Error("expect pubdate")
		}
		if i.PublishTime().IsZero() {
			t.Error("expect valid pubdate")
		}
		if len(i.ItemImage()) == 0 {
			t.Error("expect item image")
		}
	}
}
