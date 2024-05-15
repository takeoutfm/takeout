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

// Package server is the Takeout server.
package server

import (
	"net/http"

	"github.com/takeoutfm/takeout/internal/activity"
	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/internal/progress"
	"github.com/takeoutfm/takeout/lib/client"
	"github.com/takeoutfm/takeout/lib/log"
)

const (
	LoginPage = "/static/login.html"
	LinkPage  = "/static/link.html"

	SuccessRedirect = "/"
	LinkRedirect    = LinkPage
	LoginRedirect   = LoginPage

	FormUser     = "user"
	FormPass     = "pass"
	FormCode     = "code"
	FormPassCode = "passcode"
)

// doLogin creates a login session for the provided user or returns an error
func doLogin(ctx Context, user, pass string) (auth.Session, error) {
	return ctx.Auth().Login(user, pass)
}

// doPasscodeLogin creates a login session for the provided user or returns an error
func doPasscodeLogin(ctx Context, user, pass, passcode string) (auth.Session, error) {
	return ctx.Auth().PasscodeLogin(user, pass, passcode)
}

// upgradeContext creates a full context based on user and media configuration.
// This is used for most requests after the user has been authorized.
func upgradeContext(ctx Context, user *auth.User) (RequestContext, error) {
	mediaName, userConfig, err := mediaConfigFor(ctx.Config(), user)
	if err != nil {
		return RequestContext{}, err
	}
	media := makeMedia(mediaName, userConfig)
	return makeContext(ctx, user, userConfig, media), nil
}

// sessionContext creates a minimal context with the provided session.
func sessionContext(ctx Context, session *auth.Session) RequestContext {
	return makeAuthOnlyContext(ctx, session)
}

// imageContext creates a minimal context with the provided client.
func imageContext(ctx Context, client client.Getter) RequestContext {
	return makeImageContext(ctx, client)
}

// loginHandler performs a web based login session and sends back a cookie.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	r.ParseForm()
	user := r.Form.Get(FormUser)
	pass := r.Form.Get(FormPass)
	passcode := r.Form.Get(FormPassCode)

	var err error
	var session auth.Session
	if passcode == "" {
		session, err = doLogin(ctx, user, pass)
	} else {
		session, err = doPasscodeLogin(ctx, user, pass, passcode)
	}
	if err != nil {
		authErr(w, ErrUnauthorized)
		return
	}

	cookie := ctx.Auth().NewCookie(&session)
	http.SetCookie(w, &cookie)

	// Use 303 for PRG
	// https://en.wikipedia.org/wiki/Post/Redirect/Get
	http.Redirect(w, r, SuccessRedirect, http.StatusSeeOther)
}

// linkHandler performs a web based login and links to the provided code.
func linkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	r.ParseForm()
	user := r.Form.Get(FormUser)
	pass := r.Form.Get(FormPass)
	passcode := r.Form.Get(FormPassCode)
	value := r.Form.Get(FormCode)
	err := doCodeAuth(ctx, user, pass, passcode, value)
	if err == nil {
		// success
		// Use 303 for PRG
		http.Redirect(w, r, SuccessRedirect, http.StatusSeeOther)
		return
	}
	// Use 303 for PRG
	http.Redirect(w, r, LinkRedirect, http.StatusSeeOther)
}

// imageHandler handles unauthenticated image requests.
func imageHandler(ctx RequestContext, handler http.HandlerFunc, client client.Getter) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := imageContext(ctx, client)
		handler.ServeHTTP(w, withContext(r, ctx))
	}
	return http.HandlerFunc(fn)
}

// requestHandler handles unauthenticated requests.
func requestHandler(ctx RequestContext, handler http.HandlerFunc) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, withContext(r, ctx))
	}
	return http.HandlerFunc(fn)
}

func makeAuth(config *config.Config) (*auth.Auth, error) {
	a := auth.NewAuth(config)
	err := a.Open()
	return a, err
}

func makeActivity(config *config.Config) (*activity.Activity, error) {
	a := activity.NewActivity(config)
	err := a.Open()
	return a, err
}

func makeProgress(config *config.Config) (*progress.Progress, error) {
	p := progress.NewProgress(config)
	err := p.Open()
	return p, err
}

// Serve configures and starts the Takeout web, websocket, and API services.
func Serve(config *config.Config) error {
	auth, err := makeAuth(config)
	log.CheckError(err)

	activity, err := makeActivity(config)
	log.CheckError(err)

	progress, err := makeProgress(config)
	log.CheckError(err)

	schedule(config)

	// base context for all requests
	ctx := RequestContext{
		activity: activity,
		auth:     auth,
		config:   config,
		progress: progress,
		template: getTemplates(config),
	}

	resFileServer := http.FileServer(mountResFS(resStatic))
	staticHandler := func(w http.ResponseWriter, r *http.Request) {
		resFileServer.ServeHTTP(w, r)
	}

	mux := http.NewServeMux()

	aliasHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			switch r.URL.Path {
			case "/link", "/link.htm", "/link.html":
				r.URL.Path = LinkPage
				mux.ServeHTTP(w, r)
			case "/login", "/login.htm", "/login.html":
				r.URL.Path = LoginPage
				mux.ServeHTTP(w, r)
			default:
				serverErr(w, ErrNotFound)
			}
		} else {
			serverErr(w, ErrNotFound)
		}
	}

	mux.Handle("GET /static/", http.HandlerFunc(staticHandler))
	mux.Handle("GET /", accessTokenAuthHandler(ctx, viewHandler))
	mux.Handle("GET /v", accessTokenAuthHandler(ctx, viewHandler))

	// cookie auth
	mux.Handle("POST /api/login", requestHandler(ctx, apiLogin))
	mux.Handle("POST /login", requestHandler(ctx, loginHandler))
	mux.Handle("GET /login", http.HandlerFunc(aliasHandler))
	mux.Handle("GET /login.htm", http.HandlerFunc(aliasHandler))
	mux.Handle("GET /login.html", http.HandlerFunc(aliasHandler))
	mux.Handle("POST /link", requestHandler(ctx, linkHandler))
	mux.Handle("GET /link", http.HandlerFunc(aliasHandler))
	mux.Handle("GET /link.htm", http.HandlerFunc(aliasHandler))
	mux.Handle("GET /link.html", http.HandlerFunc(aliasHandler))

	// token auth
	mux.Handle("POST /api/token", requestHandler(ctx, apiTokenLogin))
	mux.Handle("GET /api/token", refreshTokenAuthHandler(ctx, apiTokenRefresh))

	// code auth
	mux.Handle("GET /api/code", requestHandler(ctx, apiCodeGet))
	mux.Handle("POST /api/code", codeTokenAuthHandler(ctx, apiCodeCheck))
	mux.Handle("POST /api/link", requestHandler(ctx, apiLink))

	// misc
	mux.Handle("GET /api/home", accessTokenAuthHandler(ctx, apiHome))
	mux.Handle("GET /api/index", accessTokenAuthHandler(ctx, apiIndex))
	mux.Handle("GET /api/search", accessTokenAuthHandler(ctx, apiSearch))

	// playlist
	mux.Handle("GET /api/playlist", accessTokenAuthHandler(ctx, apiPlaylistGet))
	mux.Handle("PATCH /api/playlist", accessTokenAuthHandler(ctx, apiPlaylistPatch))

	// music
	mux.Handle("GET /api/artists", accessTokenAuthHandler(ctx, apiArtists))
	mux.Handle("GET /api/artists/{id}", accessTokenAuthHandler(ctx, apiArtistGet))
	mux.Handle("GET /api/artists/{id}/{res}", accessTokenAuthHandler(ctx, apiArtistGetResource))
	mux.Handle("GET /api/artists/{id}/{res}/playlist", accessTokenAuthHandler(ctx, apiArtistGetPlaylist))
	mux.Handle("GET /api/artists/{id}/{res}/playlist.xspf", accessTokenAuthHandler(ctx, apiArtistGetPlaylist))
	mux.Handle("GET /api/radio", accessTokenAuthHandler(ctx, apiRadioGet))
	mux.Handle("GET /api/radio/{id}", accessTokenAuthHandler(ctx, apiRadioStationGetPlaylist))
	mux.Handle("GET /api/radio/{id}/playlist", accessTokenAuthHandler(ctx, apiRadioStationGetPlaylist))
	mux.Handle("GET /api/radio/{id}/playlist.xspf", accessTokenAuthHandler(ctx, apiRadioStationGetPlaylist))
	mux.Handle("GET /api/stations/{id}", accessTokenAuthHandler(ctx, apiRadioStationGetPlaylist))
	mux.Handle("GET /api/stations/{id}/playlist", accessTokenAuthHandler(ctx, apiRadioStationGetPlaylist))
	mux.Handle("GET /api/stations/{id}/playlist.xspf", accessTokenAuthHandler(ctx, apiRadioStationGetPlaylist))
	mux.Handle("GET /api/releases/{id}", accessTokenAuthHandler(ctx, apiReleaseGet))
	mux.Handle("GET /api/releases/{id}/playlist", accessTokenAuthHandler(ctx, apiReleaseGetPlaylist))
	mux.Handle("GET /api/releases/{id}/playlist.xspf", accessTokenAuthHandler(ctx, apiReleaseGetPlaylist))

	// video
	mux.Handle("GET /api/movies", accessTokenAuthHandler(ctx, apiMovies))
	mux.Handle("GET /api/movies/{id}", accessTokenAuthHandler(ctx, apiMovieGet))
	mux.Handle("GET /api/movies/{id}/playlist", accessTokenAuthHandler(ctx, apiMovieGetPlaylist))
	mux.Handle("GET /api/movie-genres/{name}", accessTokenAuthHandler(ctx, apiMovieGenreGet))
	mux.Handle("GET /api/movie-keywords/{name}", accessTokenAuthHandler(ctx, apiMovieKeywordGet))
	mux.Handle("GET /api/profiles/{id}", accessTokenAuthHandler(ctx, apiMovieProfileGet))
	// mux.Handle("GET /api/tv", apiTVShows)
	// mux.Handle("GET /api/tv/{id}", apiTVShowGet)
	// mux.Handle("GET /api/tv/{id}/episodes/{eid}", apiTVShowEpisodeGet)

	// podcast
	mux.Handle("GET /api/podcasts", accessTokenAuthHandler(ctx, apiPodcasts))
	mux.Handle("GET /api/podcasts/subscribed", accessTokenAuthHandler(ctx, apiPodcastsSubscribed))
	mux.Handle("GET /api/series/{id}", accessTokenAuthHandler(ctx, apiPodcastSeriesGet))
	mux.Handle("PUT /api/series/{id}/subscribed", accessTokenAuthHandler(ctx, apiPodcastSeriesSubscribe))
	mux.Handle("DELETE /api/series/{id}/subscribed", accessTokenAuthHandler(ctx, apiPodcastSeriesUnsubscribe))
	mux.Handle("GET /api/series/{id}/playlist", accessTokenAuthHandler(ctx, apiPodcastSeriesGetPlaylist))
	mux.Handle("GET /api/series/{id}/playlist.xspf", accessTokenAuthHandler(ctx, apiPodcastSeriesGetPlaylist))
	mux.Handle("GET /api/episodes/{id}", accessTokenAuthHandler(ctx, apiPodcastEpisodeGet))
	mux.Handle("GET /api/episodes/{id}/playlist", accessTokenAuthHandler(ctx, apiPodcastEpisodeGetPlaylist))
	mux.Handle("GET /api/episodes/{id}/playlist.xspf", accessTokenAuthHandler(ctx, apiPodcastEpisodeGetPlaylist))

	// location
	mux.Handle("GET /api/tracks/{uuid}/location", mediaTokenAuthHandler(ctx, apiTrackLocation))
	mux.Handle("GET /api/movies/{uuid}/location", mediaTokenAuthHandler(ctx, apiMovieLocation))
	mux.Handle("GET /api/episodes/{id}/location", mediaTokenAuthHandler(ctx, apiEpisodeLocation))

	// download
	mux.Handle("GET /d/", fileAuthHandler(ctx, apiDownload, "/d"))

	// progress
	mux.Handle("GET /api/progress", accessTokenAuthHandler(ctx, apiProgressGet))
	mux.Handle("POST /api/progress", accessTokenAuthHandler(ctx, apiProgressPost))

	// activity
	mux.Handle("GET /api/activity", accessTokenAuthHandler(ctx, apiActivityGet))
	mux.Handle("POST /api/activity", accessTokenAuthHandler(ctx, apiActivityPost))
	mux.Handle("GET /api/activity/tracks", accessTokenAuthHandler(ctx, apiActivityTracksGet))
	mux.Handle("GET /api/activity/tracks/{res}", accessTokenAuthHandler(ctx, apiActivityTracksGetResource))
	mux.Handle("GET /api/activity/tracks/{res}/playlist", accessTokenAuthHandler(ctx, apiActivityTracksGetPlaylist))
	mux.Handle("GET /api/activity/movies", accessTokenAuthHandler(ctx, apiActivityMoviesGet))
	// /activity/radio - ?
	mux.Handle("GET /api/activity/releases", accessTokenAuthHandler(ctx, apiActivityReleasesGet))

	// TODO - disable for now, work in progress
	// settings
	// mux.Handle("PUT /api/objects/{uuid}", accessTokenAuthHandler(ctx, apiObjectPut));
	// mux.Handle("GET /api/objects", accessTokenAuthHandler(ctx, apiObjectsList));
	// mux.Handle("GET /api/objects/{uuid}", accessTokenAuthHandler(ctx, apiObjectGet));

	// Hook
	//mux.Post("/hook/", requestHandler(ctx, hookHandler))

	// Images
	client := client.NewCacheOnlyGetter(config.Server.ImageClient)
	mux.Handle("GET /img/mb/rg/{rgid}", imageHandler(ctx, imgReleaseGroupFront, client))
	mux.Handle("GET /img/mb/rg/{rgid}/{side}", imageHandler(ctx, imgReleaseGroup, client))
	mux.Handle("GET /img/mb/re/{reid}", imageHandler(ctx, imgReleaseFront, client))
	mux.Handle("GET /img/mb/re/{reid}/{side}", imageHandler(ctx, imgRelease, client))
	mux.Handle("GET /img/tm/{size}/{path}", imageHandler(ctx, imgVideo, client))
	mux.Handle("GET /img/fa/{arid}/t/{path}", imageHandler(ctx, imgArtistThumb, client))
	mux.Handle("GET /img/fa/{arid}/b/{path}", imageHandler(ctx, imgArtistBackground, client))

	// pprof
	// mux.Handle("GET /debug/pprof", http.HandlerFunc(pprof.Index))
	// mux.Handle("GET /debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	// mux.Handle("GET /debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	// mux.Handle("GET /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	// mux.Handle("GET /debug/pprof/heap", pprof.Handler("heap"))
	// mux.Handle("GET /debug/pprof/block", pprof.Handler("block"))
	// mux.Handle("GET /debug/pprof/goroutine", pprof.Handler("goroutine"))
	// mux.Handle("GET /debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	// // swaggerHandler := func(w http.ResponseWriter, r *http.Request) {
	// // 	http.Redirect(w, r, "/static/swagger.json", 302)
	// // }
	// http.HandleFunc("/swagger.json", swaggerHandler)

	log.Printf("listening on %s\n", config.Server.Listen)

	return http.ListenAndServe(config.Server.Listen, mux)
}
