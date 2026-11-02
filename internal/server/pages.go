// Copyright 2026 defsub
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
	"bytes"
	"compress/gzip"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/model"
	"takeoutfm.dev/takeout/view"
)

const (
	// HeaderFragment is sent by nav.js to get only a page's body, which it
	// puts in <main> so the player keeps playing.
	HeaderFragment = "X-Takeout-Fragment"

	ParamInterval = "interval"

	appTitle = "TakeoutFM"

	// pages smaller than this aren't worth gzipping
	minGzipSize = 1024
)

// page renders one web UI page from its view. section is the sidebar item
// it belongs to, if any. build returns ErrNotFound for a bad or unknown id.
type page struct {
	temp    string
	section string
	build   func(ctx Context, r *http.Request) (view any, title string, err error)
}

// releaseList is a titled grid of releases (releases.html).
type releaseList struct {
	Title    string
	Releases []model.Release
}

// pageData is what layout.html and fragment.html render.
type pageData struct {
	*view.Index // only for the full layout
	Title       string
	Section     string
	Content     template.HTML // output of html/template, already escaped
}

// pages maps each web UI path to its page. A page's path is its API path
// without /api.
var pages = map[string]page{
	"GET /{$}": {"home.html", "home", func(ctx Context, r *http.Request) (any, string, error) {
		return HomeView(ctx), "", nil
	}},
	"GET /search": {"search.html", "", func(ctx Context, r *http.Request) (any, string, error) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		return SearchView(ctx, q), q, nil
	}},
	"GET /recently-added": {"releases.html", "added", func(ctx Context, r *http.Request) (any, string, error) {
		return &releaseList{"Recently Added", ctx.Music().RecentlyAdded()}, "Recently Added", nil
	}},
	"GET /new-releases": {"releases.html", "new", func(ctx Context, r *http.Request) (any, string, error) {
		return &releaseList{"New Releases", ctx.Music().RecentlyReleased()}, "New Releases", nil
	}},
	"GET /favorites": {"favorites.html", "favorites", func(ctx Context, r *http.Request) (any, string, error) {
		return FavoriteView(ctx), "Favorites", nil
	}},
	"GET /artists": {"artists.html", "artists", func(ctx Context, r *http.Request) (any, string, error) {
		return ArtistsView(ctx), "Artists", nil
	}},
	"GET /artists/{id}": {"artist.html", "artists", func(ctx Context, r *http.Request) (any, string, error) {
		artist, err := lookupArtist(ctx, r)
		if err != nil {
			return nil, "", err
		}
		return ArtistView(ctx, artist), artist.Name, nil
	}},
	"GET /artists/{id}/popular": {"popular.html", "artists", func(ctx Context, r *http.Request) (any, string, error) {
		artist, err := lookupArtist(ctx, r)
		if err != nil {
			return nil, "", err
		}
		return PopularView(ctx, artist), artist.Name, nil
	}},
	"GET /artists/{id}/singles": {"singles.html", "artists", func(ctx Context, r *http.Request) (any, string, error) {
		artist, err := lookupArtist(ctx, r)
		if err != nil {
			return nil, "", err
		}
		return SinglesView(ctx, artist), artist.Name, nil
	}},
	"GET /artists/{id}/want": {"want.html", "artists", func(ctx Context, r *http.Request) (any, string, error) {
		artist, err := lookupArtist(ctx, r)
		if err != nil {
			return nil, "", err
		}
		return WantListView(ctx, artist), artist.Name, nil
	}},
	"GET /releases/{id}": {"release.html", "artists", func(ctx Context, r *http.Request) (any, string, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, "", err
		}
		release, err := ctx.Music().LookupRelease(id)
		if err != nil {
			return nil, "", ErrNotFound
		}
		return ReleaseView(ctx, release), release.Name, nil
	}},
	"GET /radio": {"radio.html", "radio", func(ctx Context, r *http.Request) (any, string, error) {
		return RadioView(ctx), "Radio", nil
	}},
	"GET /playlists": {"playlists.html", "playlists", func(ctx Context, r *http.Request) (any, string, error) {
		return PlaylistsView(ctx, ctx.Music().UserPlaylists(ctx.User())), "Playlists", nil
	}},
	"GET /movies": {"movies.html", "movies", func(ctx Context, r *http.Request) (any, string, error) {
		return MoviesView(ctx), "Movies", nil
	}},
	"GET /movies/{id}": {"movie.html", "movies", func(ctx Context, r *http.Request) (any, string, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, "", err
		}
		movie, err := ctx.Film().LookupMovie(id)
		if err != nil {
			return nil, "", ErrNotFound
		}
		return MovieView(ctx, movie), movie.Title, nil
	}},
	"GET /movie-genres/{name}": {"genre.html", "movies", func(ctx Context, r *http.Request) (any, string, error) {
		name := strings.TrimSpace(r.PathValue(ParamName))
		return GenreView(ctx, name), name, nil
	}},
	"GET /movie-keywords/{name}": {"keyword.html", "movies", func(ctx Context, r *http.Request) (any, string, error) {
		name := strings.TrimSpace(r.PathValue(ParamName))
		return KeywordView(ctx, name), name, nil
	}},
	"GET /profiles/{peid}": {"profile.html", "", func(ctx Context, r *http.Request) (any, string, error) {
		person, err := ctx.FindPerson(r.PathValue(ParamPEID))
		if err != nil {
			return nil, "", ErrNotFound
		}
		return ProfileView(ctx, person), person.Name, nil
	}},
	"GET /tv": {"shows.html", "tv", func(ctx Context, r *http.Request) (any, string, error) {
		return TVShowsView(ctx), "TV Shows", nil
	}},
	"GET /tv/series/{id}": {"tvseries.html", "tv", func(ctx Context, r *http.Request) (any, string, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, "", err
		}
		series, err := ctx.TV().LookupSeries(id)
		if err != nil {
			return nil, "", ErrNotFound
		}
		return TVSeriesView(ctx, series), series.Name, nil
	}},
	"GET /tv/episodes/{id}": {"tvepisode.html", "tv", func(ctx Context, r *http.Request) (any, string, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, "", err
		}
		episode, err := ctx.TV().LookupEpisode(id)
		if err != nil {
			return nil, "", ErrNotFound
		}
		return TVEpisodeView(ctx, episode), episode.Name, nil
	}},
	"GET /podcasts": {"podcasts.html", "podcasts", func(ctx Context, r *http.Request) (any, string, error) {
		return PodcastsView(ctx), "Podcasts", nil
	}},
	"GET /series/{id}": {"series.html", "podcasts", func(ctx Context, r *http.Request) (any, string, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, "", err
		}
		series, err := ctx.Podcast().LookupSeries(id)
		if err != nil {
			return nil, "", ErrNotFound
		}
		return SeriesView(ctx, series), series.Title, nil
	}},
	"GET /episodes/{id}": {"episode.html", "podcasts", func(ctx Context, r *http.Request) (any, string, error) {
		id, err := pathID(r)
		if err != nil {
			return nil, "", err
		}
		episode, err := ctx.Podcast().LookupEpisode(id)
		if err != nil {
			return nil, "", ErrNotFound
		}
		return EpisodeView(ctx, episode), episode.Title, nil
	}},
	"GET /activity": {"activity.html", "activity", func(ctx Context, r *http.Request) (any, string, error) {
		return activityPage(ctx, "recent")
	}},
	"GET /activity/{interval}": {"activity.html", "activity", func(ctx Context, r *http.Request) (any, string, error) {
		return activityPage(ctx, r.PathValue(ParamInterval))
	}},
}

func activityPage(ctx Context, interval string) (any, string, error) {
	d := date.NewInterval(time.Now(), interval)
	return TrackStatsView(ctx, interval, d), "Activity", nil
}

func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue(ParamID))
	if err != nil || id <= 0 {
		return 0, ErrNotFound
	}
	return id, nil
}

func lookupArtist(ctx Context, r *http.Request) (model.Artist, error) {
	id, err := pathID(r)
	if err != nil {
		return model.Artist{}, err
	}
	artist, err := ctx.Music().LookupArtist(id)
	if err != nil {
		return model.Artist{}, ErrNotFound
	}
	return artist, nil
}

func pageTitle(title string) string {
	if title == "" {
		return appTitle
	}
	return title + " - " + appTitle
}

// pageHandler renders a page as the full layout, or only its body when
// nav.js asks for a fragment. The page is rendered into a buffer first so
// a template error is a clean 500 and the result can be gzipped.
func pageHandler(p page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := contextValue(r)
		v, title, err := p.build(ctx, r)
		if errors.Is(err, ErrNotFound) {
			notFoundErr(w)
			return
		} else if err != nil {
			serverErr(w, err)
			return
		}

		var body bytes.Buffer
		if err := ctx.Template().ExecuteTemplate(&body, p.temp, v); err != nil {
			serverErr(w, err)
			return
		}
		data := pageData{
			Title:   pageTitle(title),
			Section: p.section,
			Content: template.HTML(body.String()),
		}
		name := "fragment.html"
		if r.Header.Get(HeaderFragment) == "" {
			name = "layout.html"
			data.Index = IndexView(ctx)
		}
		var out bytes.Buffer
		if err := ctx.Template().ExecuteTemplate(&out, name, data); err != nil {
			serverErr(w, err)
			return
		}

		h := w.Header()
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Set("Cache-Control", "private, no-cache")
		h.Add("Vary", HeaderFragment)
		writeGzip(w, r, out.Bytes())
	}
}

// writeGzip writes b, gzipped when the client accepts it and it's worth it.
func writeGzip(w http.ResponseWriter, r *http.Request, b []byte) {
	h := w.Header()
	h.Add("Vary", "Accept-Encoding")
	if len(b) < minGzipSize || !acceptsGzip(r) {
		h.Set("Content-Length", strconv.Itoa(len(b)))
		w.Write(b)
		return
	}
	h.Set("Content-Encoding", "gzip")
	zw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if _, err := zw.Write(b); err != nil {
		log.Printf("page write: %s\n", err)
	}
	zw.Close()
}
