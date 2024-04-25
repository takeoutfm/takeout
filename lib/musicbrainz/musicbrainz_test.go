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

package musicbrainz

import (
	"bytes"
	"embed"
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

type mbzServer struct {
	t *testing.T
}

func (s mbzServer) RoundTrip(r *http.Request) (*http.Response, error) {
	var body string
	//s.t.Logf("got %s\n", r.URL.String())
	if strings.HasPrefix(r.URL.Path, "/ws/2/artist/ba0d6274-db14-4ef5-b28d-657ebde1a396") {
		body = jsonFile("test/artist_ba0d6274-db14-4ef5-b28d-657ebde1a396.json")
	} else if strings.HasPrefix(r.URL.Path, "/ws/2/artist/5b11f4ce-a62d-471e-81fc-a69a8278c7da") {
		body = jsonFile("test/artist_5b11f4ce-a62d-471e-81fc-a69a8278c7da.json")
	} else if strings.HasPrefix(r.URL.Path, "/ws/2/artist") {
		if r.URL.Query().Get("query") == "arid:a74b1b7f-71a5-4011-9441-d0b5e4122711" {
			body = jsonFile("test/artist_search_a74b1b7f-71a5-4011-9441-d0b5e4122711.json")
		} else if r.URL.Query().Get("query") == `artist:"The Black Angels"` {
			body = jsonFile("test/artist_search_49814f71-8fef-41ec-9af8-b6995c0bd601.json")
		}
	} else if strings.HasPrefix(r.URL.Path, "/ws/2/release-group/b067cde8-f3a7-394a-b8e7-640ca744f2e4") {
		body = jsonFile("test/releasegroup_b067cde8-f3a7-394a-b8e7-640ca744f2e4.json")
	} else if strings.HasPrefix(r.URL.Path, "/ws/2/release-group") {
		if r.URL.Query().Get("query") == `arid:66fc5bf8-daa4-4241-b378-9bc9077939d2 AND release:"Undertow"` {
			body = jsonFile("test/releasegroup_search_66fc5bf8-daa4-4241-b378-9bc9077939d2.json")
		}
	} else if strings.HasPrefix(r.URL.Path, "/ws/2/release") {
		if r.URL.Query().Get("artist") == "66fc5bf8-daa4-4241-b378-9bc9077939d2" {
			if r.URL.Query().Get("offset") == "0" {
				body = jsonFile("test/release_search_66fc5bf8-daa4-4241-b378-9bc9077939d2_offset_0.json")
			} else if r.URL.Query().Get("offset") == "100" {
				body = jsonFile("test/release_search_66fc5bf8-daa4-4241-b378-9bc9077939d2_offset_100.json")
			} else if r.URL.Query().Get("offset") == "200" {
				body = jsonFile("test/release_search_66fc5bf8-daa4-4241-b378-9bc9077939d2_offset_200.json")
			}
		}
	} else if strings.HasPrefix(r.URL.Path, "/release/df28cef9-7e83-481a-8c6d-61c45d0f3cd8") {
		body = jsonFile("test/cover_release_df28cef9-7e83-481a-8c6d-61c45d0f3cd8.json")
	} else if strings.HasPrefix(r.URL.Path, "/release-group/b067cde8-f3a7-394a-b8e7-640ca744f2e4") {
		body = jsonFile("test/cover_releasegroup_b067cde8-f3a7-394a-b8e7-640ca744f2e4.json")
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

func TestArtistDetail1(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	a, err := m.ArtistDetail("ba0d6274-db14-4ef5-b28d-657ebde1a396")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("expect artist")
	}
	if a.Name != "The Smashing Pumpkins" {
		t.Error("expect smashing pumpkins")
	}
	if a.BeginArea.Name != "Chicago" {
		t.Error("expect chicago")
	}
	if a.SortName != "Smashing Pumpkins, The" {
		t.Error("expect sort name")
	}
}

func TestArtistDetail2(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	a, err := m.ArtistDetail("5b11f4ce-a62d-471e-81fc-a69a8278c7da")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil {
		t.Fatal("expect artist")
	}
	if a.Name != "Nirvana" {
		t.Error("")
	}
	if a.BeginArea.Name != "Aberdeen" {
		t.Error("expect aberdeen")
	}
	if a.LifeSpan.Ended != true {
		t.Error("expect ended")
	}
	if a.LifeSpan.End != "1994-04-05" {
		t.Error("expect end date")
	}
	if a.Country != "US" {
		t.Error("expect us")
	}
}

func TestSearchArtist(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	a := m.SearchArtist("The Black Angels")
	if a == nil {
		t.Fatal("expect artist")
	}
	if a.Name != "The Black Angels" {
		t.Error("expect black angels")
	}
	if a.SortName != "Black Angels, The" {
		t.Error("expect sort name")
	}
	if a.Country != "US" {
		t.Error("expect us")
	}
	if a.BeginArea.Name != "Austin" {
		t.Error("expect austin")
	}
}

func TestSearchArtistID(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	a := m.SearchArtistID("a74b1b7f-71a5-4011-9441-d0b5e4122711")
	if a == nil {
		t.Fatal("expect artist")
	}
	if a.Name != "Radiohead" {
		t.Error("expect radiohead")
	}
	if a.Country != "GB" {
		t.Error("expect gb")
	}
}

func TestArtistReleases(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	releases, err := m.ArtistReleases("", "66fc5bf8-daa4-4241-b378-9bc9077939d2")
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) == 0 {
		t.Fatal("expect releases")
	}
	if len(releases) < 200 || len(releases) > 300 {
		t.Error("expect release count around 287")
	}
	rmap := make(map[string]string)
	for _, r := range releases {
		if r.Status == "Official" {
			rmap[r.Title] = r.Date
		}
	}
	titles := []string{"Undertow", "Ænima", "Lateralus", "10,000 Days", "Fear Inoculum"}
	for _, title := range titles {
		_, ok := rmap[title]
		if !ok {
			t.Errorf("expect release '%s'", title)
		}
	}
}

func TestSearchReleaseGroup(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	result, err := m.SearchReleaseGroup("66fc5bf8-daa4-4241-b378-9bc9077939d2", "Undertow")
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expect result")
	}
	if len(result.ReleaseGroups) == 0 {
		t.Error("expect release groups")
	}
	release := result.ReleaseGroups[0]
	if release.Title != "Undertow" {
		t.Error("expect undertow")
	}
	if release.PrimaryType != "Album" {
		t.Error("expect undertow")
	}
}

func TestReleases(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	releases, err := m.Releases("b067cde8-f3a7-394a-b8e7-640ca744f2e4") // undertow rgid
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) == 0 {
		t.Error("expect releases")
	}
	release := releases[0]
	if release.ReleaseGroup.ID != "b067cde8-f3a7-394a-b8e7-640ca744f2e4" {
		t.Error("expect rgid")
	}
	if release.Title != "Undertow" {
		t.Error("expect undertow")
	}
	if release.Status != "Official" {
		t.Error("expect official")
	}
}

func TestCoverArtArchive(t *testing.T) {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, mbzServer{t: t})
	m := NewMusicBrainz(c)
	art, err := m.CoverArtArchive(
		"df28cef9-7e83-481a-8c6d-61c45d0f3cd8", "b067cde8-f3a7-394a-b8e7-640ca744f2e4") // undertow rgid
	if err != nil {
		t.Fatal(err)
	}
	if art == nil {
		t.Error("expect art")
	}
	if len(art.Images) == 0 {
		t.Error("expect images")
	}
	for _, img := range art.Images {
		if img.Image == "" || !strings.HasPrefix(img.Image, "http") {
			t.Errorf("expect image got '%s'", img.Image)
		}
	}

}
