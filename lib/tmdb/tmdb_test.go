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

package tmdb // import "takeoutfm.dev/takeout/lib/tmdb"

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

type tmdbServer struct {
	t *testing.T
}

func (s tmdbServer) RoundTrip(r *http.Request) (*http.Response, error) {
	var body string
	//s.t.Logf("got %s\n", r.URL.String())
	if strings.HasPrefix(r.URL.Path, "/3/configuration") {
		body = jsonFile("test/configuration.json")
	} else if strings.HasPrefix(r.URL.Path, "/3/search/movie") {
		body = jsonFile("test/search_movie.json")
	} else if strings.HasPrefix(r.URL.Path, "/3/movie/550/release_dates") {
		body = jsonFile("test/movie_550_releasedates.json")
	} else if strings.HasPrefix(r.URL.Path, "/3/movie/580/keywords") {
		body = jsonFile("test/movie_580_keywords.json")
	} else if strings.HasPrefix(r.URL.Path, "/3/movie/49849/credits") {
		body = jsonFile("test/movie_49849_credits.json")
	} else if strings.HasPrefix(r.URL.Path, "/3/movie/49849") {
		body = jsonFile("test/movie_49849.json")
	} else if strings.HasPrefix(r.URL.Path, "/3/person/11357") {
		body = jsonFile("test/person_11357.json")
	}
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}, nil
}

func makeClient(t *testing.T) *TMDB {
	c := client.NewTransportGetter(client.Config{UserAgent: "test/1.0"}, tmdbServer{t: t})
	return NewTMDB(Config{Language: "en-US", Key: "903a776b0638da68e9ade38ff538e1d3"}, c)
}

func TestConfiguration(t *testing.T) {
	tmdb := makeClient(t)
	config, err := tmdb.configuration()
	if err != nil {
		t.Fatal(err)
	}
	if config.Images.BaseURL == "" {
		t.Error("expect baseurl")
	}
	if config.Images.SecureBaseURL == "" {
		t.Error("expect secure baseurl")
	}
}

func TestMovieSearch(t *testing.T) {
	tmdb := makeClient(t)
	result, err := tmdb.MovieSearch("cowboys and aliens")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Error("expect results")
	}
	movie := result[0]
	if movie.ID != 49849 {
		t.Error("expect id")
	}
	if movie.Title != "Cowboys & Aliens" {
		t.Error("expect title")
	}
	if movie.OriginalTitle != "Cowboys & Aliens" {
		t.Error("expect original title")
	}
	if movie.ReleaseDate != "2011-07-29" {
		t.Error("expect date")
	}
}

func TestMovieDetail(t *testing.T) {
	tmdb := makeClient(t)
	movie, err := tmdb.MovieDetail(49849)
	if err != nil {
		t.Fatal(err)
	}
	if movie.ID != 49849 {
		t.Error("expect id")
	}
	if movie.Title != "Cowboys & Aliens" {
		t.Error("expect title")
	}
	if movie.OriginalTitle != "Cowboys & Aliens" {
		t.Error("expect original title")
	}
	if movie.ReleaseDate != "2011-07-29" {
		t.Error("expect date")
	}
	if len(movie.Genres) == 0 {
		t.Error("expect genres")
	}
	for _, g := range movie.Genres {
		if g.Name == "" {
			t.Error("expect genre")
		}
	}
}

func TestMovieCredits(t *testing.T) {
	tmdb := makeClient(t)
	credits, err := tmdb.MovieCredits(49849)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, c := range credits.Cast {
		if c.Name == "" {
			t.Error("expect cast name")
		}
		if c.Character == "" {
			t.Error("expect cast character")
		}
		if c.Name == "Daniel Craig" && c.Character == "Jake Lonergan" {
			found++
		}
	}
	if found != 1 {
		t.Error("expect actor")
	}
	for _, c := range credits.Crew {
		if c.Department == "" {
			t.Error("expect dept")
		}
		if c.Job == "" {
			t.Error("expect job")
		}
		if c.Name == "" {
			t.Error("expect name")
		}
	}
}

func TestMovieReleaseType(t *testing.T) {
	tmdb := makeClient(t)
	release, err := tmdb.MovieReleaseType(550, "US", TypeTheatrical) // fight club
	if err != nil {
		t.Fatal(err)
	}
	if release.Certification != "R" {
		t.Error("expect R rating")
	}
}

func TestMovieKeywordNames(t *testing.T) {
	tmdb := makeClient(t)
	keywords, err := tmdb.MovieKeywordNames(580) // jaws the revenge
	if err != nil {
		t.Fatal(err)
	}
	if len(keywords) == 0 {
		t.Error("expect keywords")
	}
	found := 0
	for _, v := range keywords {
		if v == "shark" {
			found++
		}
	}
	if found != 1 {
		t.Error("expect shark")
	}
}

func TestPersonDetail(t *testing.T) {
	tmdb := makeClient(t)
	person, err := tmdb.PersonDetail(11357) // bruce campbell
	if err != nil {
		t.Fatal(err)
	}
	if person.Name != "Bruce Campbell" {
		t.Error("expect bruce")
	}
	if person.Birthplace != "Birmingham, Michigan, USA" {
		t.Error("expect birmingham")
	}
}

// func TestTVSearch(t *testing.T) {
// 	config, err := config.TestConfig()
// 	if err != nil {
// 		t.Errorf("GetConfig %s\n", err)
// 	}

// 	if config.TMDB.Key == "" {
// 		t.Errorf("no key\n")
// 	}
// 	m := NewTMDB(config)
// 	results, err := m.TVSearch("the shining")
// 	if err != nil {
// 		t.Errorf("%s\n", err)
// 	}
// 	for _, r := range results {
// 		d := date.ParseDate(r.FirstAirDate)
// 		fmt.Printf("%d %s (%d)\n", r.ID, r.Name, d.Year())
// 		fmt.Printf("  %s\n", m.OriginalPoster(r.PosterPath))
// 		for _, g := range r.GenreIDs {
// 			fmt.Printf("  %s\n", m.TVGenre(g))
// 		}
// 	}
// }

// func TestTVDetail(t *testing.T) {
// 	config, err := config.TestConfig()
// 	if err != nil {
// 		t.Errorf("GetConfig %s\n", err)
// 	}

// 	if config.TMDB.Key == "" {
// 		t.Errorf("no key\n")
// 	}
// 	m := NewTMDB(config)
// 	tv, err := m.TVDetail(1867) // game of thrones
// 	if err != nil {
// 		t.Errorf("%s\n", err)
// 	}
// 	fmt.Printf("%s (%s)\n", tv.Name, tv.FirstAirDate)
// 	fmt.Printf("%+v\n", tv)
// }

// func TestEpisodeDetail(t *testing.T) {
// 	config, err := config.TestConfig()
// 	if err != nil {
// 		t.Errorf("GetConfig %s\n", err)
// 	}
// 	if config.TMDB.Key == "" {
// 		t.Errorf("no key\n")
// 	}
// 	m := NewTMDB(config)
// 	episode, err := m.EpisodeDetail(1399, 1, 1) // game of thrones
// 	if err != nil {
// 		t.Errorf("%s\n", err)
// 	}
// 	fmt.Printf("%d %s (%s)\n", episode.ID, episode.Name, episode.AirDate)
// 	fmt.Printf("%+v\n", episode)
// }

// func TestEpisodeCredits(t *testing.T) {
// 	config, err := config.TestConfig()
// 	if err != nil {
// 		t.Errorf("GetConfig %s\n", err)
// 	}
// 	if config.TMDB.Key == "" {
// 		t.Errorf("no key\n")
// 	}
// 	m := NewTMDB(config)
// 	credits, err := m.EpisodeCredits(1399, 1, 1) // game of thrones
// 	if err != nil {
// 		t.Errorf("%s\n", err)
// 	}
// 	for _, c := range credits.Cast {
// 		fmt.Printf("cast: %s - %s\n", c.Name, c.Character)
// 	}
// 	for _, c := range credits.Crew {
// 		fmt.Printf("crew: %s - %s\n", c.Name, c.Job)
// 	}
// 	for _, c := range credits.Guests {
// 		fmt.Printf("guest: %s - %s\n", c.Name, c.Character)
// 	}
// }
