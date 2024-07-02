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
	"embed"
	_ "embed"
	"fmt"
	"html/template"

	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/lib/log"
	"github.com/takeoutfm/takeout/model"
	"github.com/takeoutfm/takeout/view"
)

//go:embed res/static
var resStatic embed.FS

func mountResFS(resFS embed.FS) http.FileSystem {
	fsys, err := fs.Sub(resFS, "res")
	if err != nil {
		log.Panicln(err)
	}
	return http.FS(fsys)
}

//go:embed res/template
var resTemplates embed.FS

func getTemplateFS(config *config.Config) fs.FS {
	return resTemplates
}

func getTemplates(config *config.Config) *template.Template {
	return template.Must(template.New("").Funcs(doFuncMap()).ParseFS(getTemplateFS(config),
		"res/template/*.html",
		"res/template/music/*.html",
		"res/template/video/*.html",
		"res/template/podcast/*.html"))
}

func doFuncMap() template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"ymd":  date.YMD,
		"unescapeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"link": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("/v?release=%d", o.(model.Release).ID)
			case model.Artist:
				link = fmt.Sprintf("/v?artist=%d", o.(model.Artist).ID)
			case model.Track:
				link = locateTrack(o.(model.Track))
			case model.Movie:
				link = fmt.Sprintf("/v?movie=%d", o.(model.Movie).ID)
			case model.Series:
				link = fmt.Sprintf("/v?series=%d", o.(model.Series).ID)
			case model.Episode:
				link = fmt.Sprintf("/v?episode=%d", o.(model.Episode).ID)
			case model.Station:
				link = fmt.Sprintf("/v?station=%d", o.(model.Station).ID)
			}
			return link
		},
		"link_amz": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://www.amazon.com/dp/%s", o.(model.Release).Asin)
			}
			return link
		},
		"link_camel": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://camelcamelcamel.com/product/%s", o.(model.Release).Asin)
			}
			return link
		},
		"link_mbz": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://musicbrainz.org/release-group/%s", o.(model.Release).RGID)
			case model.Artist:
				link = fmt.Sprintf("https://musicbrainz.org/artist/%s", o.(model.Artist).ARID)
			}
			return link
		},
		"link_google": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://www.google.com/search?q=%s",
					url.QueryEscape(
						strings.Join([]string{o.(model.Release).Name, "by", o.(model.Release).Artist}, " ")))
			}
			return link
		},
		"link_wiki": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://en.wikipedia.org/w/index.php?title=Special:Search&search=%s",
					url.QueryEscape(
						strings.Join([]string{o.(model.Release).Name, "by", o.(model.Release).Artist}, " ")))
			}
			return link
		},
		"url": func(o interface{}) string {
			var loc string
			switch o.(type) {
			case model.Track:
				loc = locateTrack(o.(model.Track))
			case model.Movie:
				loc = locateMovie(o.(model.Movie))
			case model.Episode:
				loc = locateEpisode(o.(model.Episode))
			}
			return loc
		},
		"popular": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Artist:
				link = fmt.Sprintf("/v?popular=%d", o.(model.Artist).ID)
			}
			return link
		},
		"singles": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Artist:
				link = fmt.Sprintf("/v?singles=%d", o.(model.Artist).ID)
			}
			return link
		},
		"want": func(o interface{}) string {
			var link string
			switch o.(type) {
			case model.Artist:
				link = fmt.Sprintf("/v?want=%d", o.(model.Artist).ID)
			}
			return link
		},
		"ref": func(o interface{}, args ...string) string {
			var ref string
			switch o.(type) {
			case model.Release:
				ref = fmt.Sprintf("/music/releases/%d/tracks", o.(model.Release).ID)
			case model.Artist:
				ref = fmt.Sprintf("/music/artists/%d/%s", o.(model.Artist).ID, args[0])
			case model.Track:
				ref = fmt.Sprintf("/music/tracks/%d", o.(model.Track).ID)
			case string:
				ref = fmt.Sprintf("/music/search?q=%s", url.QueryEscape(o.(string)))
			case model.Station:
				ref = fmt.Sprintf("/music/stations/%d", o.(model.Station).ID)
			case view.Playlist:
				ref = fmt.Sprintf("/music/playlists/%d", o.(view.Playlist).ID)
			}
			return ref
		},
		"home": func() string {
			return "/v?home=1"
		},
		"runtime": func(m model.Movie) string {
			hours := m.Runtime / 60
			mins := m.Runtime % 60
			return fmt.Sprintf("%dh %dm", hours, mins)
		},
		"letter": func(a model.Artist) string {
			return a.SortName[0:1]
		},
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	var result interface{}
	var temp string

	if v := r.URL.Query().Get("release"); v != "" {
		// /v?release={release-id}
		m := ctx.Music()
		id, _ := strconv.Atoi(v)
		release, _ := m.LookupRelease(id)
		result = ReleaseView(ctx, release)
		temp = "release.html"
	} else if v := r.URL.Query().Get("artist"); v != "" {
		// /v?artist={artist-id}
		m := ctx.Music()
		id, _ := strconv.Atoi(v)
		artist, _ := m.LookupArtist(id)
		result = ArtistView(ctx, artist)
		temp = "artist.html"
	} else if v := r.URL.Query().Get("artists"); v != "" {
		// /v?artists=x
		result = ArtistsView(ctx)
		temp = "artists.html"
	} else if v := r.URL.Query().Get("popular"); v != "" {
		// /v?popular={artist-id}
		m := ctx.Music()
		id, _ := strconv.Atoi(v)
		artist, _ := m.LookupArtist(id)
		result = PopularView(ctx, artist)
		temp = "popular.html"
	} else if v := r.URL.Query().Get("singles"); v != "" {
		// /v?singles={artist-id}
		m := ctx.Music()
		id, _ := strconv.Atoi(v)
		artist, _ := m.LookupArtist(id)
		result = SinglesView(ctx, artist)
		temp = "singles.html"
	} else if v := r.URL.Query().Get("want"); v != "" {
		// /v?want={artist-id}
		m := ctx.Music()
		id, _ := strconv.Atoi(v)
		artist, _ := m.LookupArtist(id)
		result = WantListView(ctx, artist)
		temp = "want.html"
	} else if v := r.URL.Query().Get("home"); v != "" {
		// /v?home=x
		result = HomeView(ctx)
		temp = "home.html"
	} else if v := r.URL.Query().Get("q"); v != "" {
		// /v?q={pattern}
		result = SearchView(ctx, strings.TrimSpace(v))
		temp = "search.html"
	} else if v := r.URL.Query().Get("radio"); v != "" {
		// /v?radio=x
		result = RadioView(ctx)
		temp = "radio.html"
	} else if v := r.URL.Query().Get("playlists"); v != "" {
		// /v?playlist=x
		playlists := ctx.Music().UserPlaylists(ctx.User())
		result = PlaylistsView(ctx, playlists)
		temp = "playlists.html"
	} else if v := r.URL.Query().Get("movies"); v != "" {
		// /v?movies=x
		result = MoviesView(ctx)
		temp = "movies.html"
	} else if v := r.URL.Query().Get("movie"); v != "" {
		// /v?movie={movie-id}
		vid := ctx.Video()
		id, _ := strconv.Atoi(v)
		movie, _ := vid.LookupMovie(id)
		result = MovieView(ctx, movie)
		temp = "movie.html"
	} else if v := r.URL.Query().Get("profile"); v != "" {
		// /v?profile={person-id}
		vid := ctx.Video()
		id, _ := strconv.Atoi(v)
		person, _ := vid.LookupPerson(id)
		result = ProfileView(ctx, person)
		temp = "profile.html"
	} else if v := r.URL.Query().Get("genre"); v != "" {
		// /v?genre={genre-name}
		name := strings.TrimSpace(v)
		result = GenreView(ctx, name)
		temp = "genre.html"
	} else if v := r.URL.Query().Get("keyword"); v != "" {
		// /v?keyword={keyword-name}
		name := strings.TrimSpace(v)
		result = KeywordView(ctx, name)
		temp = "keyword.html"
	} else if v := r.URL.Query().Get("watch"); v != "" {
		// /v?watch={movie-id}
		vid := ctx.Video()
		id, _ := strconv.Atoi(v)
		movie, _ := vid.LookupMovie(id)
		result = WatchView(ctx, movie)
		temp = "watch.html"
	} else if v := r.URL.Query().Get("podcasts"); v != "" {
		// /v?podcasts=x
		result = PodcastsView(ctx)
		temp = "podcasts.html"
	} else if v := r.URL.Query().Get("series"); v != "" {
		// /v?series={series-id}
		p := ctx.Podcast()
		id, _ := strconv.Atoi(v)
		series, _ := p.LookupSeries(id)
		result = SeriesView(ctx, series)
		temp = "series.html"
	} else if v := r.URL.Query().Get("episode"); v != "" {
		// /v?episode={episode-id}
		p := ctx.Podcast()
		id, _ := strconv.Atoi(v)
		episode, _ := p.LookupEpisode(id)
		result = EpisodeView(ctx, episode)
		temp = "episode.html"
	} else {
		result = IndexView(ctx)
		temp = "index.html"
	}

	render(ctx, temp, result, w, r)
}

func render(ctx Context, temp string, view interface{},
	w http.ResponseWriter, r *http.Request) {
	err := ctx.Template().ExecuteTemplate(w, temp, view)
	if err != nil {
		serverErr(w, err)
	}
}
