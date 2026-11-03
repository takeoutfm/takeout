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
	"net/url"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/people"
	"takeoutfm.dev/takeout/internal/podcast"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/tmdb"
	"takeoutfm.dev/takeout/model"
	"takeoutfm.dev/takeout/view"
)

//go:embed res/static
var resStatic embed.FS

//go:embed res/template
var resTemplates embed.FS

func getTemplateFS(config *config.Config) fs.FS {
	return resTemplates
}

func getTemplates(config *config.Config) *template.Template {
	return template.Must(template.New("").Funcs(doFuncMap()).ParseFS(getTemplateFS(config),
		"res/template/*.html",
		"res/template/activity/*.html",
		"res/template/music/*.html",
		"res/template/people/*.html",
		"res/template/podcast/*.html",
		"res/template/tv/*.html",
		"res/template/film/*.html"))
}

// tmdbImage is the server's image proxy path for a TMDB image, or ""
// without one.
func tmdbImage(size, path string) string {
	if path == "" {
		return ""
	}
	return "/img/tm/" + size + path
}

func doFuncMap() template.FuncMap {
	return template.FuncMap{
		"ymd":   date.YMD,
		"asset": staticAssets.URL,
		// plainText returns a plain string (not template.HTML) so the
		// template engine escapes it. Never return template.HTML for
		// content from feeds or other third parties.
		"plainText": plainText,
		"chart": func(o any) string {
			var link string
			switch o.(type) {
			case *view.TrackStats:
				link = fmt.Sprintf("/api/activity/tracks/%s/chart", o.(*view.TrackStats).Interval)
			}
			return link
		},
		"link": func(o any) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("/releases/%d", o.(model.Release).ID)
			case model.Artist:
				link = fmt.Sprintf("/artists/%d", o.(model.Artist).ID)
			case model.Movie:
				link = fmt.Sprintf("/movies/%d", o.(model.Movie).ID)
			case model.Series:
				link = fmt.Sprintf("/series/%d", o.(model.Series).ID)
			case model.Episode:
				link = fmt.Sprintf("/episodes/%d", o.(model.Episode).ID)
			case model.TVSeries:
				link = fmt.Sprintf("/tv/series/%d", o.(model.TVSeries).ID)
			case model.TVEpisode:
				link = fmt.Sprintf("/tv/episodes/%d", o.(model.TVEpisode).ID)
			}
			return link
		},
		"link_amz": func(o any) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://www.amazon.com/dp/%s", o.(model.Release).Asin)
			}
			return link
		},
		"link_mbz": func(o any) string {
			var link string
			switch o.(type) {
			case model.Release:
				link = fmt.Sprintf("https://musicbrainz.org/release-group/%s", o.(model.Release).RGID)
			case model.Artist:
				link = fmt.Sprintf("https://musicbrainz.org/artist/%s", o.(model.Artist).ARID)
			}
			return link
		},
		"popular": func(o any) string {
			var link string
			switch o.(type) {
			case model.Artist:
				link = fmt.Sprintf("/artists/%d/popular", o.(model.Artist).ID)
			}
			return link
		},
		"singles": func(o any) string {
			var link string
			switch o.(type) {
			case model.Artist:
				link = fmt.Sprintf("/artists/%d/singles", o.(model.Artist).ID)
			}
			return link
		},
		"want": func(o any) string {
			var link string
			switch o.(type) {
			case model.Artist:
				link = fmt.Sprintf("/artists/%d/want", o.(model.Artist).ID)
			}
			return link
		},
		"ref": func(o any, args ...string) string {
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
		"runtime": func(m model.Movie) string {
			hours := m.Runtime / 60
			mins := m.Runtime % 60
			return fmt.Sprintf("%dh %dm", hours, mins)
		},
		"letter": func(a model.Artist) string {
			return a.SortName[0:1]
		},
		"cover": func(o any) string {
			return music.CoverSmall(o)
		},
		"image": func(o any) string {
			var img string
			switch o.(type) {
			case model.Series:
				img = podcast.SeriesImage(o.(model.Series))
			case model.Episode:
				img = podcast.EpisodeImage(o.(model.Episode))
			}
			return img
		},
		"profile": func(o any) string {
			var img string
			switch o.(type) {
			case model.Person:
				img = people.PersonProfile(o.(model.Person))
			}
			return img
		},
		"poster": func(o any) string {
			var img string
			switch o.(type) {
			case model.Movie:
				img = film.MoviePoster(o.(model.Movie))
			case model.TVSeries:
				img = tv.SeriesPoster(o.(model.TVSeries))
			}
			return img
		},
		"backdrop": func(o any) string {
			var img string
			switch o.(type) {
			case model.Movie:
				img = film.MovieBackdrop(o.(model.Movie))
			case model.TVSeries:
				img = tv.SeriesBackdrop(o.(model.TVSeries))
			}
			return img
		},
		"still": func(o any) string {
			var img string
			switch o.(type) {
			case model.TVEpisode:
				img = tv.EpisodeStillImage(o.(model.TVEpisode))
			}
			return img
		},
		// header images, sharp on high-density screens
		"poster_large": func(o any) string {
			var path string
			switch o.(type) {
			case model.Movie:
				path = o.(model.Movie).PosterPath
			case model.TVSeries:
				path = o.(model.TVSeries).PosterPath
			}
			return tmdbImage(tmdb.Poster780, path)
		},
		"profile_large": func(o any) string {
			var path string
			switch o.(type) {
			case model.Person:
				path = o.(model.Person).ProfilePath
			}
			return tmdbImage(tmdb.Profile632, path)
		},
		"still_large": func(o any) string {
			var img string
			switch o.(type) {
			case model.TVEpisode:
				img = tv.EpisodeStillLarge(o.(model.TVEpisode))
			}
			return img
		},
	}
}
