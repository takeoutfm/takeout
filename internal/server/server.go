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
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"

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
func upgradeContext(ctx Context, user auth.User) (RequestContext, error) {
	mediaName, userConfig, err := mediaConfigFor(ctx.Config(), user)
	if err != nil {
		return RequestContext{}, err
	}
	media := makeMedia(mediaName, userConfig)
	return makeContext(ctx, user, userConfig, media), nil
}

// sessionContext creates a minimal context with the provided session.
func sessionContext(ctx Context, session auth.Session) RequestContext {
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
	// mux.Handle("POST /api/login", requestHandler(ctx, apiLogin)) -- not used anymore?
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
	mux.Handle("GET /api/playlist", accessTokenAuthHandler(ctx, apiPlaylist))
	mux.Handle("PATCH /api/playlist", accessTokenAuthHandler(ctx, apiPlaylistPatch))

	// saved playlists
	mux.Handle("GET /api/playlists", accessTokenAuthHandler(ctx, apiPlaylists))
	mux.Handle("POST /api/playlists", accessTokenAuthHandler(ctx, apiPlaylistsCreate))
	mux.Handle("GET /api/playlists/{id}", accessTokenAuthHandler(ctx, apiPlaylistsGet))
	mux.Handle("GET /api/playlists/{id}/playlist", accessTokenAuthHandler(ctx, apiPlaylistsGetPlaylist))
	mux.Handle("PATCH /api/playlists/{id}/playlist", accessTokenAuthHandler(ctx, apiPlaylistsPatch))
	mux.Handle("DELETE /api/playlists/{id}", accessTokenAuthHandler(ctx, apiPlaylistsDelete))

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
	mux.Handle("GET /api/tracks/{id}/playlist", accessTokenAuthHandler(ctx, apiTrackPlaylist))

	// people
	mux.Handle("GET /api/profiles/{peid}", accessTokenAuthHandler(ctx, apiProfileGet))

	// movies
	mux.Handle("GET /api/movies", accessTokenAuthHandler(ctx, apiMovies))
	mux.Handle("GET /api/movies/{id}", accessTokenAuthHandler(ctx, apiMovieGet))
	mux.Handle("GET /api/movies/{id}/playlist", accessTokenAuthHandler(ctx, apiMovieGetPlaylist))
	mux.Handle("GET /api/movie-genres/{name}", accessTokenAuthHandler(ctx, apiMovieGenreGet))
	mux.Handle("GET /api/movie-keywords/{name}", accessTokenAuthHandler(ctx, apiMovieKeywordGet))

	// tv shows
	mux.Handle("GET /api/tv", accessTokenAuthHandler(ctx, apiTV))
	mux.Handle("GET /api/tv/series/{id}", accessTokenAuthHandler(ctx, apiTVSeriesGet))
	mux.Handle("GET /api/tv/episodes/{id}", accessTokenAuthHandler(ctx, apiTVEpisodeGet))
	// //api/tv/genres/{name}
	// //api/tv/keywords/{name}

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
	mux.Handle("GET /api/tv/episodes/{uuid}/location", mediaTokenAuthHandler(ctx, apiTVEpisodeLocation))

	// download
	mux.Handle("GET /d/", fileAuthHandler(ctx, apiDownload, "/d"))

	// progress
	mux.Handle("GET /api/progress", accessTokenAuthHandler(ctx, apiProgressGet))
	mux.Handle("POST /api/progress", accessTokenAuthHandler(ctx, apiProgressPost))

	// activity
	// /activity/tracks/yesterday
	// /activity/tracks/lastweek/stats
	// /activity/tracks/lastweek/chart
	mux.Handle("POST /api/activity", accessTokenAuthHandler(ctx, apiActivityPost))
	mux.Handle("GET /api/activity/tracks/{res}", accessTokenAuthHandler(ctx, apiActivityTrackHistory))
	mux.Handle("GET /api/activity/tracks/{res}/stats", accessTokenAuthHandler(ctx, apiActivityTrackStats))
	mux.Handle("GET /api/activity/tracks/{res}/counts", accessTokenAuthHandler(ctx, apiActivityTrackCounts))
	mux.Handle("GET /api/activity/tracks/{res}/chart", accessTokenAuthHandler(ctx, apiActivityTrackChart))

	// TODO - disable for now, work in progress
	// settings
	// mux.Handle("PUT /api/objects/{uuid}", accessTokenAuthHandler(ctx, apiObjectPut));
	// mux.Handle("GET /api/objects", accessTokenAuthHandler(ctx, apiObjectsList));
	// mux.Handle("GET /api/objects/{uuid}", accessTokenAuthHandler(ctx, apiObjectGet));

	// Images
	client := client.NewCacheOnlyGetter(config.Server.ImageClient)
	mux.Handle("GET /img/mb/rg/{rgid}", imageHandler(ctx, imgReleaseGroupFront, client))
	mux.Handle("GET /img/mb/rg/{rgid}/{side}", imageHandler(ctx, imgReleaseGroup, client))
	mux.Handle("GET /img/mb/re/{reid}", imageHandler(ctx, imgReleaseFront, client))
	mux.Handle("GET /img/mb/re/{reid}/{side}", imageHandler(ctx, imgRelease, client))
	mux.Handle("GET /img/tm/{size}/{path}", imageHandler(ctx, imgTMDB, client))
	mux.Handle("GET /img/fa/{arid}/t/{path}", imageHandler(ctx, imgArtistThumb, client))
	mux.Handle("GET /img/fa/{arid}/b/{path}", imageHandler(ctx, imgArtistBackground, client))

	// // swaggerHandler := func(w http.ResponseWriter, r *http.Request) {
	// // 	http.Redirect(w, r, "/static/swagger.json", 302)
	// // }
	// http.HandleFunc("/swagger.json", swaggerHandler)

	go func() {
		ctrl := http.NewServeMux()
		ctrl.Handle("GET /jobs/{name}", requestHandler(ctx, jobsHandler))
		ctrl.Handle("GET /config", requestHandler(ctx,
			func(w http.ResponseWriter, r *http.Request) {
				ctx := contextValue(r)
				ctx.Config().Write(w)
			}))
		ctrl.Handle("GET /config/{media}", requestHandler(ctx,
			func(w http.ResponseWriter, r *http.Request) {
				ctx := contextValue(r)
				config, err := mediaConfig(ctx.Config(), r.PathValue("media"))
				if err != nil {
					serverErr(w, err)
				} else {
					config.Write(w)
				}
			}))

		ctrl.Handle("GET /debug/pprof", http.HandlerFunc(pprof.Index))
		ctrl.Handle("GET /debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		ctrl.Handle("GET /debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		ctrl.Handle("GET /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		ctrl.Handle("GET /debug/pprof/heap", pprof.Handler("heap"))
		ctrl.Handle("GET /debug/pprof/block", pprof.Handler("block"))
		ctrl.Handle("GET /debug/pprof/goroutine", pprof.Handler("goroutine"))
		ctrl.Handle("GET /debug/pprof/threadcreate", pprof.Handler("threadcreate"))

		socketPath := "/tmp/takeout.sock"
		sock, err := net.Listen("unix", socketPath)
		log.CheckError(err)
		log.CheckError(os.Chmod(socketPath, 0600))
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			os.Remove(socketPath)
			os.Exit(1)
		}()
		err = http.Serve(sock, ctrl)
		log.CheckError(err)
	}()

	log.Println("listening on", config.Server.Listen)
	return http.ListenAndServe(config.Server.Listen, mux)
}
