// Copyright 2024 defsub
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

package server

import (
	"html/template"
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/activity"
	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/internal/music"
	"github.com/takeoutfm/takeout/internal/podcast"
	"github.com/takeoutfm/takeout/internal/progress"
	"github.com/takeoutfm/takeout/internal/video"
	"github.com/takeoutfm/takeout/lib/client"
	"github.com/takeoutfm/takeout/model"
)

type TextContext struct {
	t *testing.T

	a    *activity.Activity
	auth *auth.Auth
	m    *music.Music
	pod  *podcast.Podcast
	p    *progress.Progress
	v    *video.Video
}

func NewTestContext(t *testing.T) *TextContext {
	return &TextContext{t: t}
}

func (c *TextContext) Activity() *activity.Activity {
	if c.a == nil {
		c.a = activity.NewActivity(c.Config())
		err := c.a.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.a
}

func (c *TextContext) Auth() *auth.Auth {
	if c.auth == nil {
		c.auth = auth.NewAuth(c.Config())
		err := c.auth.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.auth
}

func (c *TextContext) Config() *config.Config {
	config, err := config.TestingConfig()
	if err != nil {
		c.t.Fatal(err)
	}
	return config
}

func (c *TextContext) Music() *music.Music {
	if c.m == nil {
		c.m = music.NewMusic(c.Config())
		err := c.m.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.m
}

func (c *TextContext) Podcast() *podcast.Podcast {
	if c.pod == nil {
		c.pod = podcast.NewPodcast(c.Config())
		err := c.pod.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.pod
}

func (c *TextContext) Progress() *progress.Progress {
	if c.p == nil {
		c.p = progress.NewProgress(c.Config())
		err := c.p.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.p
}

func (c *TextContext) Template() *template.Template {
	return &template.Template{}
}

func (c *TextContext) User() *auth.User {
	return &auth.User{
		Name:  "takeout",
		Media: "test",
	}
}

func (c *TextContext) Session() *auth.Session {
	return &auth.Session{
		User:    "takeout",
		Token:   "2c52d8aa-e37e-4ed6-884b-1a565f18bbfc",
		Expires: time.Now().Add(24 * time.Hour),
	}
}

func (c *TextContext) Video() *video.Video {
	if c.v == nil {
		c.v = video.NewVideo(c.Config())
		err := c.v.Open()
		if err != nil {
			c.t.Fatal(err)
		}
	}
	return c.v
}

func (c *TextContext) ImageClient() client.Getter {
	return nil
}

func (c *TextContext) LocateTrack(model.Track) string {
	return ""
}

func (c *TextContext) LocateMovie(model.Movie) string {
	return ""
}

func (c *TextContext) LocateEpisode(model.Episode) string {
	return ""
}

func (c *TextContext) FindArtist(string) (model.Artist, error) {
	return model.Artist{}, nil
}

func (c *TextContext) FindRelease(string) (model.Release, error) {
	return model.Release{}, nil
}

func (c *TextContext) FindTrack(string) (model.Track, error) {
	return model.Track{}, nil
}

func (c *TextContext) FindStation(string) (model.Station, error) {
	return model.Station{}, nil
}

func (c *TextContext) FindMovie(string) (model.Movie, error) {
	return model.Movie{}, nil
}

func (c *TextContext) FindSeries(string) (model.Series, error) {
	return model.Series{}, nil
}

func (c *TextContext) FindEpisode(string) (model.Episode, error) {
	return model.Episode{}, nil
}

func (c *TextContext) TrackImage(model.Track) string {
	return ""
}

func (c *TextContext) ArtistImage(model.Artist) string {
	return ""
}

func (c *TextContext) ArtistBackground(model.Artist) string {
	return ""
}

func (c *TextContext) MovieImage(model.Movie) string {
	return ""
}

func (c *TextContext) EpisodeImage(model.Episode) string {
	return ""
}
