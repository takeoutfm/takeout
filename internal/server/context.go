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

package server

import (
	"context"
	"fmt"
	"html/template"
	"net/http"

	"takeoutfm.dev/takeout/internal/activity"
	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/podcast"
	"takeoutfm.dev/takeout/internal/progress"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/client"
	"takeoutfm.dev/takeout/lib/str"
	"takeoutfm.dev/takeout/model"
)

type contextKey string

var (
	contextKeyContext = contextKey("context")
)

func withContext(r *http.Request, ctx Context) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKeyContext, ctx))
}

func contextValue(r *http.Request) Context {
	return r.Context().Value(contextKeyContext).(Context)
}

type Context interface {
	Activity() *activity.Activity
	Auth() *auth.Auth
	Config() *config.Config
	Music() *music.Music
	Podcast() *podcast.Podcast
	Progress() *progress.Progress
	Template() *template.Template
	User() auth.User
	Session() auth.Session
	Film() *film.Film
	TV() *tv.TV
	ImageClient() client.Getter

	LocateTrack(model.Track) string
	LocateMovie(model.Movie) string
	LocateEpisode(model.Episode) string
	LocateTVEpisode(model.TVEpisode) string

	FindArtist(string) (model.Artist, error)
	FindRelease(string) (model.Release, error)
	FindReleaseTracks(model.Release) []model.Track
	FindTrack(string) (model.Track, error)
	FindStation(string) (model.Station, error)
	FindPlaylist(string) (model.Playlist, error)
	FindMovie(string) (model.Movie, error)
	FindTVSeries(string) (model.TVSeries, error)
	FindTVEpisode(string) (model.TVEpisode, error)
	FindSeries(string) (model.Series, error)
	FindSeriesEpisodes(model.Series) []model.Episode
	FindEpisode(string) (model.Episode, error)
	FindPerson(string) (model.Person, error)

	TrackImage(model.Track) string
	ArtistImage(model.Artist) string
	ArtistBackground(model.Artist) string
	MovieImage(model.Movie) string
	EpisodeImage(model.Episode) string
	TVSeriesImage(model.TVSeries) string
	TVEpisodeImage(model.TVEpisode) string
}

type RequestContext struct {
	activity    *activity.Activity
	auth        *auth.Auth
	config      *config.Config
	user        auth.User
	media       *Media
	progress    *progress.Progress
	session     auth.Session
	template    *template.Template
	imageClient client.Getter
}

func makeContext(ctx Context, u auth.User, c *config.Config, m *Media) RequestContext {
	return RequestContext{
		activity: ctx.Activity(),
		auth:     ctx.Auth(),
		config:   c,
		media:    m,
		progress: ctx.Progress(),
		template: ctx.Template(),
		user:     u,
	}
}

func makeAuthOnlyContext(ctx Context, session auth.Session) RequestContext {
	return RequestContext{
		auth:    ctx.Auth(),
		session: session,
	}
}

func makeImageContext(ctx Context, client client.Getter) RequestContext {
	return RequestContext{
		imageClient: client,
	}
}

func (ctx RequestContext) Activity() *activity.Activity {
	return ctx.activity
}

func (ctx RequestContext) Auth() *auth.Auth {
	return ctx.auth
}

func (ctx RequestContext) Config() *config.Config {
	return ctx.config
}

func (ctx RequestContext) Music() *music.Music {
	return ctx.media.music
}

func (ctx RequestContext) Podcast() *podcast.Podcast {
	return ctx.media.podcast
}

func (ctx RequestContext) Progress() *progress.Progress {
	return ctx.progress
}

func (ctx RequestContext) Template() *template.Template {
	return ctx.template
}

func (ctx RequestContext) User() auth.User {
	return ctx.user
}

func (ctx RequestContext) Session() auth.Session {
	return ctx.session
}

func (ctx RequestContext) Film() *film.Film {
	return ctx.media.film
}

func (ctx RequestContext) TV() *tv.TV {
	return ctx.media.tv
}

func (RequestContext) LocateTrack(t model.Track) string {
	return locateTrack(t)
}

func (RequestContext) LocateMovie(v model.Movie) string {
	return locateMovie(v)
}

func (RequestContext) LocateEpisode(e model.Episode) string {
	return locateEpisode(e)
}

func (RequestContext) LocateTVEpisode(e model.TVEpisode) string {
	return locateTVEpisode(e)
}

func (ctx RequestContext) FindArtist(id string) (model.Artist, error) {
	return ctx.Music().FindArtist(id)
}

func (ctx RequestContext) FindRelease(id string) (model.Release, error) {
	return ctx.Music().FindRelease(id)
}

func (ctx RequestContext) FindReleaseTracks(release model.Release) []model.Track {
	return ctx.Music().ReleaseTracks(release)
}

func (ctx RequestContext) FindTrack(id string) (model.Track, error) {
	return ctx.Music().FindTrack(id)
}

func (ctx RequestContext) FindStation(id string) (model.Station, error) {
	return ctx.Music().FindStation(id)
}

func (ctx RequestContext) FindPlaylist(id string) (model.Playlist, error) {
	return ctx.Music().FindPlaylist(ctx.User(), id)
}

func (ctx RequestContext) FindMovie(id string) (model.Movie, error) {
	return ctx.Film().FindMovie(id)
}

func (ctx RequestContext) FindTVSeries(id string) (model.TVSeries, error) {
	return ctx.TV().FindSeries(id)
}

func (ctx RequestContext) FindTVEpisode(id string) (model.TVEpisode, error) {
	return ctx.TV().FindEpisode(id)
}

func (ctx RequestContext) FindSeries(id string) (model.Series, error) {
	return ctx.Podcast().FindSeries(id)
}

func (ctx RequestContext) FindSeriesEpisodes(series model.Series) []model.Episode {
	return ctx.Podcast().Episodes(series)
}

func (ctx RequestContext) FindEpisode(id string) (model.Episode, error) {
	return ctx.Podcast().FindEpisode(id)
}

func (ctx RequestContext) FindPerson(id string) (model.Person, error) {
	peid := str.Atoi(id)
	person, err := ctx.Film().Person(peid)
	if err != nil {
		person, err = ctx.TV().Person(peid)
		if err != nil {
			return model.Person{}, err
		}
	}
	return person, nil
}

func (ctx RequestContext) TrackImage(t model.Track) string {
	return ctx.Music().TrackImage(t).String()
}

func (ctx RequestContext) ArtistImage(a model.Artist) string {
	return ctx.Music().ArtistImage(a)
}

func (ctx RequestContext) ArtistBackground(a model.Artist) string {
	return ctx.Music().ArtistBackground(a)
}

func (ctx RequestContext) MovieImage(m model.Movie) string {
	return film.MoviePoster(m)
}

func (ctx RequestContext) TVSeriesImage(s model.TVSeries) string {
	return tv.SeriesPoster(s)
}

func (ctx RequestContext) TVEpisodeImage(e model.TVEpisode) string {
	return tv.EpisodeStillImage(e)
}

func (ctx RequestContext) EpisodeImage(e model.Episode) string {
	return podcast.EpisodeImage(e)
}

func (ctx RequestContext) ImageClient() client.Getter {
	return ctx.imageClient
}

func locateTrack(t model.Track) string {
	return fmt.Sprintf("/api/tracks/%s/location", t.UUID)
}

func locateMovie(v model.Movie) string {
	return fmt.Sprintf("/api/movies/%s/location", v.UUID)
}

func locateEpisode(e model.Episode) string {
	return fmt.Sprintf("/api/episodes/%d/location", e.ID)
}

func locateTVEpisode(e model.TVEpisode) string {
	return fmt.Sprintf("/api/tv/episodes/%s/location", e.UUID)
}
