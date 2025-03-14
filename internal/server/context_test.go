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

package server

import (
	"errors"
	"html/template"
	"net/http"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/activity"
	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/podcast"
	"takeoutfm.dev/takeout/internal/progress"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/client"
	"takeoutfm.dev/takeout/lib/gorm"
	"takeoutfm.dev/takeout/lib/str"
	"takeoutfm.dev/takeout/model"
)

const TestUserID = "takeout"
const TestArtistID = "100"
const TestReleaseID = "101"

const TestTrackID = "102"
const TestTrackUUID = "65de7d6e-faae-4592-a3b8-81eabd18f212"
const TestTrackETag = "fecca96e-dcd5-454a-82c4-17204818f7ef"
const TestTrackREID = "ecba0a2c-1585-4e90-980d-f839e1d603c5"
const TestTrackRGID = "299a873c-f3b0-4d5f-8589-668d815e1241"
const TestTrackRID = "dd99fb43-3a89-470c-a6ac-db03cd5a79dd"

const TestStationID = "103"
const TestMovieID = "104"
const TestSeriesID = "105"
const TestEpisodeID = "106"
const TestPlaylistID = "107"

type TestContext struct {
	t    *testing.T
	a    *activity.Activity
	auth *auth.Auth
	m    *music.Music
	pod  *podcast.Podcast
	p    *progress.Progress
	f    *film.Film
	tv   *tv.TV
}

func NewTestContext(t *testing.T) *TestContext {
	return &TestContext{t: t}
}

func (c *TestContext) Activity() *activity.Activity {
	if c.a == nil {
		c.a = activity.NewActivity(c.Config())
		err := c.a.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.a
}

func (c *TestContext) Auth() *auth.Auth {
	if c.auth == nil {
		c.auth = auth.NewAuth(c.Config())
		err := c.auth.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.auth
}

func (c *TestContext) Config() *config.Config {
	config, err := config.TestingConfig()
	if err != nil {
		c.t.Fatal(err)
	}
	return config
}

func (c *TestContext) Music() *music.Music {
	if c.m == nil {
		c.m = music.NewMusic(c.Config())
		err := c.m.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.m
}

func (c *TestContext) Podcast() *podcast.Podcast {
	if c.pod == nil {
		c.pod = podcast.NewPodcast(c.Config())
		err := c.pod.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.pod
}

func (c *TestContext) Progress() *progress.Progress {
	if c.p == nil {
		c.p = progress.NewProgress(c.Config())
		err := c.p.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.p
}

func (c *TestContext) Template() *template.Template {
	return &template.Template{}
}

func (c *TestContext) User() auth.User {
	return auth.User{
		Name:  TestUserID,
		Media: "test",
	}
}

func (c *TestContext) Session() auth.Session {
	return auth.Session{
		User:    TestUserID,
		Token:   "2c52d8aa-e37e-4ed6-884b-1a565f18bbfc",
		Expires: time.Now().Add(24 * time.Hour),
	}
}

func (c *TestContext) Film() *film.Film {
	if c.f == nil {
		c.f = film.NewFilm(c.Config())
		err := c.f.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.f
}

func (c *TestContext) TV() *tv.TV {
	if c.tv == nil {
		c.tv = tv.NewTV(c.Config())
		err := c.tv.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.tv
}

func (c *TestContext) ImageClient() client.Getter {
	return nil
}

func (c *TestContext) LocateTrack(t model.Track) string {
	return "/api/tracks/4e3f3533-5f1a-4899-b44b-83268e0b2b39/location"
}

func (c *TestContext) LocateMovie(model.Movie) string {
	return "/api/movies/77d9513d-33d0-47ba-ab16-f19fc5e5200b/location"
}

func (c *TestContext) LocateEpisode(model.Episode) string {
	return "/api/episodes/2dc5f8f66003e208da5b801e38e27818/location"
}

func (c *TestContext) LocateTVEpisode(model.TVEpisode) string {
	return "/api/tv/episodes/24eb51db-a4b3-4b23-9d8b-6ccb97150f2c/location"
}


func (c *TestContext) FindArtist(id string) (model.Artist, error) {
	if id == TestArtistID {
		return model.Artist{Name: "test artist"}, nil
	}
	return model.Artist{}, errors.New("artist not found")
}

func (c *TestContext) FindRelease(id string) (model.Release, error) {
	if id == TestReleaseID {
		return model.Release{
			Model: gorm.Model{ID: uint(str.Atoi(TestReleaseID))},
			Name:  "test release",
		}, nil
	}
	return model.Release{}, errors.New("release not found")
}

func (c *TestContext) FindReleaseTracks(release model.Release) []model.Track {
	t, _ := c.FindTrack(TestTrackID)
	return []model.Track{t}
}

func (c *TestContext) FindTrack(id string) (model.Track, error) {
	if id == TestTrackID {
		return model.Track{
			Model: gorm.Model{ID: uint(str.Atoi(TestTrackID))},
			Title: "test title",
			UUID:  TestTrackUUID,
			RID:   TestTrackRID,
			REID:  TestTrackREID,
			RGID:  TestTrackRGID,
		}, nil
	}
	return model.Track{}, errors.New("track not found")
}

func (c *TestContext) FindStation(id string) (model.Station, error) {
	if id == TestStationID {
		return model.Station{Name: "test station", Shared: true}, nil
	}
	return model.Station{}, errors.New("station not found")
}

func (c *TestContext) FindPlaylist(id string) (model.Playlist, error) {
	if id == TestPlaylistID {
		return model.Playlist{Name: "test playlist", User: TestUserID}, nil
	}
	p, err := c.Music().FindPlaylist(auth.User{Name: TestUserID}, id)
	if err == nil {
		return p, nil
	}
	return model.Playlist{}, errors.New("playlist not found")
}

func (c *TestContext) FindMovie(id string) (model.Movie, error) {
	if id == TestMovieID {
		return model.Movie{Title: "test movie"}, nil
	}
	return model.Movie{}, errors.New("movie not found")
}

func (c *TestContext) FindSeries(id string) (model.Series, error) {
	if id == TestSeriesID {
		return model.Series{Title: "test series"}, nil
	}
	return model.Series{}, errors.New("series not found")
}

func (c *TestContext) FindSeriesEpisodes(series model.Series) []model.Episode {
	e, _ := c.FindEpisode(TestEpisodeID)
	return []model.Episode{e}
}

func (c *TestContext) FindEpisode(id string) (model.Episode, error) {
	if id == TestEpisodeID {
		return model.Episode{Title: "test episode", SID: TestSeriesID}, nil
	}
	return model.Episode{}, errors.New("episode not found")
}

func (c *TestContext) TrackImage(model.Track) string {
	return ""
}

func (c *TestContext) ArtistImage(model.Artist) string {
	return ""
}

func (c *TestContext) ArtistBackground(model.Artist) string {
	return ""
}

func (c *TestContext) MovieImage(model.Movie) string {
	return ""
}

func (c *TestContext) EpisodeImage(model.Episode) string {
	return ""
}

func (c *TestContext) TVEpisodeImage(model.TVEpisode) string {
	return ""
}

func (c *TestContext) TVSeriesImage(model.TVSeries) string {
	return ""
}

func (c *TestContext) FindTVSeries(string) (model.TVSeries, error) {
	return model.TVSeries{}, nil
}

func (c *TestContext) FindTVEpisode(string) (model.TVEpisode, error) {
	return model.TVEpisode{}, nil
}

func (c *TestContext) FindPerson(string) (model.Person, error) {
	return model.Person{}, nil
}

//

func TestWithContext(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	rr := withContext(r, NewTestContext(t))
	if contextValue(rr) == nil {
		t.Fatal("expect context")
	}
}

func TestMakeContext(t *testing.T) {
	ctx := NewTestContext(t)
	u := auth.User{Name: "test user"}
	m := makeMedia("test media", ctx.Config())
	makeContext(ctx, u, &config.Config{}, m)
}
