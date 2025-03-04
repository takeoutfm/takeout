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

package fanart // import "takeoutfm.dev/takeout/lib/fanart"

import (
	"bytes"
	"embed"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"takeoutfm.dev/takeout/lib/client"
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

type fanartServer struct {
	t *testing.T
}

func (s fanartServer) RoundTrip(r *http.Request) (*http.Response, error) {
	var body string
	//s.t.Logf("got %s\n", r.URL.String())
	if strings.HasPrefix(r.URL.Path, "/v3/music/a74b1b7f-71a5-4011-9441-d0b5e4122711") {
		body = jsonFile("test/music_a74b1b7f-71a5-4011-9441-d0b5e4122711.json")
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

func TestFanart(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, fanartServer{t: t})
	f := NewFanart(Config{ProjectKey: "93ede276ba6208318031727060b697c8"}, c)
	a := f.ArtistArt("a74b1b7f-71a5-4011-9441-d0b5e4122711")
	if a == nil {
		t.Fatal("expect art")
	}
	if len(a.ArtistBackgrounds) == 0 {
		t.Error("expect backgrounds")
	}
	if len(a.ArtistThumbs) == 0 {
		t.Error("expect thumbs")
	}
	if a.MBID != "a74b1b7f-71a5-4011-9441-d0b5e4122711" {
		t.Errorf("expect mbid got %s", a.MBID)
	}
	for _, img := range a.ArtistBackgrounds {
		if strings.HasPrefix(img.URL, "http") == false {
			t.Error("expect bg url")
		}
	}
	for _, img := range a.ArtistThumbs {
		if strings.HasPrefix(img.URL, "http") == false {
			t.Error("expect thumb url")
		}
	}
}
