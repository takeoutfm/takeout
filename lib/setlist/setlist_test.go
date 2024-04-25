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

package setlist

import (
	"bytes"
	"embed"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/takeoutfm/takeout/lib/client"
)

//go:embed test/*.json
var jsonFiles embed.FS

func jsonFile(name string) string {
	d, err := jsonFiles.ReadFile(name)
	if err != nil {
		return ""
	}
	return string(d)
}

type setlistServer struct {
	t *testing.T
}

func (s setlistServer) RoundTrip(r *http.Request) (*http.Response, error) {
	var body string
	s.t.Logf("got %s\n", r.URL.String())
	if strings.HasPrefix(r.URL.Path, "/rest/1.0/search/setlists") {
		if r.URL.Query().Get("artistMbid") == "ca891d65-d9b0-4258-89f7-e6ba29d83767" &&
			r.URL.Query().Get("year") == "2022" {
			page := r.URL.Query().Get("p")
			file := fmt.Sprintf("test/artist_ca891d65-d9b0-4258-89f7-e6ba29d83767_2022_%s.json", page)
			body = jsonFile(file)
		}
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

const apiKey = ""

func makeClient(t *testing.T) *Client {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, setlistServer{t: t})
	return NewClient(Config{ApiKey: apiKey}, c)
}

func TestSetlist(t *testing.T) {
	s := makeClient(t)
	arid := "ca891d65-d9b0-4258-89f7-e6ba29d83767" // iron maiden
	result := s.ArtistYear(arid, 2022)
	for _, sl := range result {
		t.Logf("%s %s @ %s, %s, %s\n", sl.Tour.Name, sl.EventTime().Format("Mon, Jan 2, 2006"),
			sl.Venue.Name, sl.Venue.City.Name, sl.Venue.City.Country.Name)
		for _, v := range sl.Sets.Set {
			if v.Encore == 0 {
				t.Logf("set %s - %d\n", v.Name, len(v.Songs))
				for i, t := range v.Songs {
					fmt.Printf("%d. %s (%s)\n", i, t.Name, t.Info)
				}
			} else {
				t.Logf("encore %s - %d\n", v.Name, len(v.Songs))
				for i, t := range v.Songs {
					fmt.Printf("%d. %s (%s)\n", i, t.Name, t.Info)
				}
			}
		}
	}
}
