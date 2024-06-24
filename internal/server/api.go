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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/lib/encoding/xspf"
	"github.com/takeoutfm/takeout/lib/header"
	"github.com/takeoutfm/takeout/lib/log"
	"github.com/takeoutfm/takeout/lib/str"
	"github.com/takeoutfm/takeout/model"
	"github.com/takeoutfm/takeout/spiff"
	"github.com/takeoutfm/takeout/view"
)

const (
	ApplicationJson = "application/json"

	ParamID   = "id"
	ParamRes  = "res"
	ParamName = "name"
	ParamEID  = "eid"
	ParamUUID = "uuid"

	QuerySearch = "q"
	QueryStart  = "start"
	QueryEnd    = "end"
	QueryToken  = "token"
)

type credentials struct {
	User     string
	Pass     string
	Passcode string
}

// type status struct {
// 	Status  int
// 	Message string `json:,omitempty`
// 	Cookie  string `json:,omitempty`
// }

// apiLogin handles login requests and returns a cookie.
// func apiLogin(w http.ResponseWriter, r *http.Request) {
// 	ctx := contextValue(r)
// 	w.Header().Set(header.ContentType, ApplicationJson)

// 	var creds credentials
// 	body, _ := ioutil.ReadAll(r.Body)
// 	err := json.Unmarshal(body, &creds)
// 	if err != nil {
// 		serverErr(w, err)
// 		return
// 	}

// 	var result status
// 	var session auth.Session
// 	if creds.Passcode == "" {
// 		session, err = doLogin(ctx, creds.User, creds.Pass)
// 	} else {
// 		session, err = doPasscodeLogin(ctx, creds.User, creds.Pass, creds.Passcode)
// 	}
// 	if err != nil {
// 		authErr(w, err)
// 		result = status{
// 			Status:  http.StatusUnauthorized,
// 			Message: "error",
// 		}
// 	} else {
// 		cookie := ctx.Auth().NewCookie(&session)
// 		http.SetCookie(w, &cookie)
// 		result = status{
// 			Status:  http.StatusOK,
// 			Message: "ok",
// 			Cookie:  cookie.Value,
// 		}
// 	}

// 	enc := json.NewEncoder(w)
// 	enc.Encode(result)
// }

type tokenResponse struct {
	AccessToken  string
	RefreshToken string
	MediaToken   string `json:",omitempty"`
}

// apiTokenLogin handles login requests and returns tokens.
func apiTokenLogin(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)

	var creds credentials
	body, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(body, &creds)
	if err != nil {
		authErr(w, err)
		return
	}

	var session auth.Session
	if creds.Passcode == "" {
		session, err = doLogin(ctx, creds.User, creds.Pass)
	} else {
		session, err = doPasscodeLogin(ctx, creds.User, creds.Pass, creds.Passcode)
	}
	if err != nil {
		if auth.CredentialsError(err) {
			authErr(w, err)
		} else {
			serverErr(w, err)
		}
		return
	}

	authorizeNew(session, w, r)
}

type linkCredentials struct {
	Code     string
	User     string
	Pass     string
	Passcode string
}

// apiLink links a code to valid login credentials
func apiLink(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)

	var creds linkCredentials
	body, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(body, &creds)
	if err != nil {
		authErr(w, err)
		return
	}

	err = doCodeAuth(ctx, creds.User, creds.Pass, creds.Passcode, creds.Code)
	if err != nil {
		if auth.CredentialsError(err) {
			authErr(w, err)
		} else {
			serverErr(w, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// authorizeNew creates and sends new tokens for the provided session.
func authorizeNew(session auth.Session, w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	w.Header().Set(header.ContentType, ApplicationJson)

	var resp tokenResponse
	var err error
	resp.RefreshToken = session.Token
	resp.AccessToken, err = ctx.Auth().NewAccessToken(session)
	if err != nil {
		serverErr(w, err)
		return
	}
	resp.MediaToken, err = ctx.Auth().NewMediaToken(session)
	if err != nil {
		serverErr(w, err)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(resp)
}

// authorizeRefresh refreshes and sends new access token for the provided session.
// MediaToken is unchanged.
func authorizeRefresh(session auth.Session, w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	w.Header().Set(header.ContentType, ApplicationJson)

	var resp tokenResponse
	var err error
	resp.RefreshToken = session.Token
	resp.AccessToken, err = ctx.Auth().NewAccessToken(session)
	if err != nil {
		serverErr(w, err)
		return
	}

	// extend the session lifetime
	err = ctx.Auth().Refresh(&session)
	if err != nil {
		serverErr(w, err)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(resp)
}

// apiTokenRefresh uses refresh token to create a new access token.
func apiTokenRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	session := ctx.Session()
	authorizeRefresh(*session, w, r)
}

type codeResponse struct {
	AccessToken string
	Code        string
}

// apiCodeGet begins a code-based authorization phase.
// The code is used to separately link with a new or existing login.
// The token is used to check if the code has been linked.
func apiCodeGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	w.Header().Set(header.ContentType, ApplicationJson)

	var resp codeResponse
	var err error
	ctx.Auth().DeleteExpiredCodes()
	code := ctx.Auth().GenerateCode()
	resp.Code = code.Value
	resp.AccessToken, err = ctx.Auth().NewCodeToken(code.Value)
	if err != nil {
		serverErr(w, err)
		return
	}

	enc := json.NewEncoder(w)
	enc.Encode(resp)
}

type codeCheck struct {
	Code string
}

// apiCodeCheck
func apiCodeCheck(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)

	var check codeCheck
	body, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(body, &check)
	if err != nil {
		authErr(w, err)
		return
	}

	code := ctx.Auth().LookupCode(check.Code)
	if code == nil {
		authErr(w, ErrInvalidCode)
		return
	}
	if code.Linked() == false {
		// valid but not linked yet
		accessDenied(w)
		return
	}

	session := ctx.Auth().TokenSession(code.Token)
	if session == nil {
		serverErr(w, ErrInvalidSession)
		return
	}

	authorizeNew(*session, w, r)
}

var locationRegexp = regexp.MustCompile(`/api/(tracks)/([0-9a-zA-Z-]+)/location`)

// writePlaylist will write a playlist to the response and optionally fully
// resolve tracks for external app (vlc) playback.
func writePlaylist(w http.ResponseWriter, r *http.Request, plist *spiff.Playlist) {
	if strings.HasSuffix(r.URL.Path, ".xspf") {
		// create XML spiff with tracks fully resolved
		ctx := contextValue(r)
		w.Header().Set(header.ContentType, xspf.XMLContentType)
		encoder := xspf.NewXMLEncoder(w)
		encoder.Header(plist.Spiff.Title)
		for i := range plist.Spiff.Entries {
			matches := locationRegexp.FindStringSubmatch(plist.Spiff.Entries[i].Location[0])
			if matches != nil {
				src := matches[1]
				if src == "tracks" {
					m := ctx.Music()
					uuid := matches[2]
					track, err := m.FindTrack("uuid:" + uuid)
					if err != nil {
						continue
					}
					// TODO need to extent bucket URLExpiration for these tracks
					url := m.TrackURL(&track)
					plist.Spiff.Entries[i].Location = []string{url.String()}
				}
			}
			encoder.Encode(plist.Spiff.Entries[i])
		}
		encoder.Footer()

	} else {
		// use json spiff with track location
		w.Header().Set(header.ContentType, ApplicationJson)
		result, _ := plist.Marshal()
		w.Write(result)
	}
}

// TODO check
func recvStation(w http.ResponseWriter, r *http.Request,
	s *model.Station) error {
	ctx := contextValue(r)
	body, _ := io.ReadAll(r.Body)
	err := json.Unmarshal(body, s)
	if err != nil {
		serverErr(w, err)
		return err
	}
	if s.Name == "" || s.Ref == "" {
		http.Error(w, "bummer", http.StatusBadRequest)
		return err
	}
	s.User = ctx.User().Name
	if s.Ref == "/api/playlist" {
		// copy playlist
		p := ctx.Music().UserPlaylist(ctx.User())
		if p != nil {
			s.Playlist = p.Playlist
		}
	}
	return nil
}

func makeEmptyPlaylist(w http.ResponseWriter, r *http.Request) (*model.Playlist, error) {
	ctx := contextValue(r)
	plist := spiff.NewPlaylist(spiff.TypeMusic)
	plist.Spiff.Location = r.URL.Path
	plist.Spiff.Entries = []spiff.Entry{} // so json track isn't null
	data, _ := plist.Marshal()
	p := model.Playlist{User: ctx.User().Name, Playlist: data}
	err := ctx.Music().CreatePlaylist(&p)
	return &p, err
}

func apiPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	p := ctx.Music().UserPlaylist(ctx.User())
	if p == nil {
		var err error
		p, err = makeEmptyPlaylist(w, r)
		if err != nil {
			serverErr(w, err)
			return
		}
	}
	w.Header().Set(header.ContentType, ApplicationJson)
	w.WriteHeader(http.StatusOK)
	w.Write(p.Playlist)
}

func apiPlaylistPatch(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx := contextValue(r)
	user := ctx.User()
	m := ctx.Music()
	p := m.UserPlaylist(user)
	if p == nil {
		p, err = makeEmptyPlaylist(w, r)
		if err != nil {
			serverErr(w, err)
			return
		}
	}
	doPlaylistPatch(ctx, p, w, r)
}

func apiPlaylists(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	playlists := ctx.Music().UserPlaylists(ctx.User())
	view := PlaylistsView(ctx, playlists)
	apiView(w, r, view)
}

func apiPlaylistsCreate(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		serverErr(w, err)
		return
	}

	// unmarshal to obtain playlist title
	plist, err := spiff.Unmarshal(data)
	if err != nil {
		serverErr(w, err)
		return
	}
	if plist.Spiff.Title == "" {
		badRequest(w, ErrMissingTitle)
		return
	}

	// save user playlist with name from title
	p := model.Playlist{User: ctx.User().Name, Name: plist.Spiff.Title, Playlist: data}
	err = ctx.Music().CreatePlaylist(&p)
	if err != nil {
		serverErr(w, err)
		return
	}

	// update location from saved playlist ID (/api/playlists/id/playlist)
	plist.Spiff.Location = fmt.Sprintf("%s/%d/playlist", r.URL.Path, p.ID)

	// resolve refs
	err = Resolve(ctx, plist)
	if err != nil {
		serverErr(w, err)
		return
	}

	// user name is creator
	plist.Spiff.Creator = ctx.User().Name

	// save updated playlist
	data, err = plist.Marshal()
	if err != nil {
		serverErr(w, err)
		return
	}
	p.Playlist = data
	err = ctx.Music().UpdatePlaylist(&p)
	if err != nil {
		serverErr(w, err)
		return
	}

	apiView(w, r, PlaylistView(ctx, p))
}

func apiPlaylistsGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	playlist, err := ctx.FindPlaylist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, PlaylistView(ctx, playlist))
	}
}

func apiPlaylistsGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	playlist, err := ctx.FindPlaylist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		w.Header().Set(header.ContentType, ApplicationJson)
		w.WriteHeader(http.StatusOK)
		w.Write(playlist.Playlist)
	}
}

func apiPlaylistsPatch(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	playlist, err := ctx.FindPlaylist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		doPlaylistPatch(ctx, &playlist, w, r)
	}
}

func apiPlaylistsDelete(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	playlist, err := ctx.FindPlaylist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		err = ctx.Music().DeletePlaylist(ctx.User(), int(playlist.ID))
		if err != nil {
			serverErr(w, err)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func doPlaylistPatch(ctx Context, p *model.Playlist, w http.ResponseWriter, r *http.Request) {
	var err error

	before := p.Playlist

	// apply patch
	patch, _ := io.ReadAll(r.Body)
	p.Playlist, err = spiff.Patch(p.Playlist, patch)
	if err != nil {
		serverErr(w, err)
		return
	}
	plist, _ := spiff.Unmarshal(p.Playlist)
	err = Resolve(ctx, plist)
	if err != nil {
		serverErr(w, err)
		return
	}

	if plist.Spiff.Entries == nil {
		plist.Spiff.Entries = []spiff.Entry{}
	}

	if plist.Type != spiff.TypeStream {
		// TODO check if spiff entries have changed to stream; need
		// better way to handle this but for now, any entry without
		// identifiers or sizes is a radio stream so fix spiff
		// accordingly
		change := false
		for _, e := range plist.Spiff.Entries {
			if len(e.Identifier) == 0 || len(e.Size) == 0 {
				change = true
				break
			}
			if len(e.Size) == 1 && e.Size[0] == -1 {
				change = true
				break
			}
		}
		if change {
			plist.Type = spiff.TypeStream
		}
	}

	p.Playlist, _ = plist.Marshal()
	ctx.Music().UpdatePlaylist(p)

	v, _ := spiff.Compare(before, p.Playlist)
	if v {
		// entries didn't change, only metadata
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.Header().Set(header.ContentType, ApplicationJson)
		w.WriteHeader(http.StatusOK)
		w.Write(p.Playlist)
	}
}

func apiProgressGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	view := ProgressView(ctx)
	apiView(w, r, view)
}

func apiProgressPost(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	user := ctx.User()
	var offsets model.Offsets
	body, err := io.ReadAll(r.Body)
	if err != nil {
		badRequest(w, err)
		return
	}
	err = json.Unmarshal(body, &offsets)
	if err != nil {
		badRequest(w, err)
		return
	}
	for i := range offsets.Offsets {
		// will update array inplace
		o := &offsets.Offsets[i]
		if len(o.User) != 0 {
			// post must not have a user
			badRequest(w, err)
			return
		}
		// use authenticated user
		o.User = user.Name
		if !o.Valid() {
			badRequest(w, ErrInvalidOffset)
			return
		}
	}
	for _, o := range offsets.Offsets {
		// update each offset as needed
		log.Printf("update progress %s %d/%d\n", o.ETag, o.Offset, o.Duration)
		err = ctx.Progress().Update(user, o)
	}
	w.WriteHeader(http.StatusNoContent)
}

func apiView(w http.ResponseWriter, r *http.Request, view interface{}) {
	w.Header().Set(header.ContentType, ApplicationJson)
	json.NewEncoder(w).Encode(view)
}

func apiHome(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	view := HomeView(ctx)
	apiView(w, r, view)
}

func apiIndex(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	view := IndexView(ctx)
	apiView(w, r, view)
}

func apiSearch(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	if v := r.URL.Query().Get(QuerySearch); v != "" {
		// /api/search?q={pattern}
		view := SearchView(ctx, strings.TrimSpace(v))
		apiView(w, r, view)
	} else {
		notFoundErr(w)
	}
}

func apiArtists(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	apiView(w, r, ArtistsView(ctx))
}

func apiArtistGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	artist, err := ctx.FindArtist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, ArtistView(ctx, artist))
	}
}

func apiArtistGetResource(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	res := r.PathValue(ParamRes)
	artist, err := ctx.FindArtist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		switch res {
		case "popular":
			apiView(w, r, PopularView(ctx, artist))
		case "singles":
			apiView(w, r, SinglesView(ctx, artist))
		case "playlist":
			apiArtistGetPlaylist(w, r)
		case "wantlist":
			apiView(w, r, WantListView(ctx, artist))
		default:
			notFoundErr(w)
		}
	}
}

func apiArtistGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	res := r.PathValue(ParamRes)
	artist, err := ctx.FindArtist(id)
	if err != nil {
		notFoundErr(w)
	} else {
		// /api/artists/:id/:res/playlist -> /music/artists/:id/:res
		nref := fmt.Sprintf("/music/artists/%s/%s", id, res)
		plist := ResolveArtistPlaylist(ctx,
			ArtistView(ctx, artist), r.URL.Path, nref)
		writePlaylist(w, r, plist)
	}
}

func apiRadioGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	apiView(w, r, RadioView(ctx))
}

func apiRadioPost(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	var s model.Station
	err := recvStation(w, r, &s)
	if err != nil {
		return
	}
	err = ctx.Music().CreateStation(&s)
	if err != nil {
		serverErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	enc.Encode(s)
}

func apiRadioStationGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	station, err := ctx.FindStation(id)
	if err != nil {
		notFoundErr(w)
		return
	}
	if !station.Visible(ctx.User().Name) {
		notFoundErr(w)
		return
	}
	plist := RefreshStation(ctx, &station)
	writePlaylist(w, r, plist)
}

func apiMovies(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	apiView(w, r, MoviesView(ctx))
}

func apiMovieGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	movie, err := ctx.FindMovie(id)
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, MovieView(ctx, movie))
	}
}

func apiMovieGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	movie, err := ctx.FindMovie(id)
	if err != nil {
		notFoundErr(w)
	} else {
		view := MovieView(ctx, movie)
		plist := ResolveMoviePlaylist(ctx, view, r.URL.Path)
		writePlaylist(w, r, plist)
	}
}

func apiMovieProfileGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	person, err := ctx.Video().LookupPerson(str.Atoi(id))
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, ProfileView(ctx, person))
	}
}

func apiMovieGenreGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	name := r.PathValue(ParamName)
	// TODO sanitize
	apiView(w, r, GenreView(ctx, name))
}

func apiMovieKeywordGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	name := r.PathValue(ParamName)
	// TODO sanitize
	apiView(w, r, KeywordView(ctx, name))
}

func apiPodcasts(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	apiView(w, r, PodcastsView(ctx))
}

func apiPodcastsSubscribed(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	apiView(w, r, PodcastsSubscribedView(ctx))
}

func apiPodcastSeriesGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	series, err := ctx.Podcast().FindSeries(id)
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, SeriesView(ctx, series))
	}
}

func apiPodcastSeriesSubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	series, err := ctx.Podcast().FindSeries(id)
	if err != nil {
		notFoundErr(w)
	} else {
		err := ctx.Podcast().Subscribe(series.SID, ctx.User().Name)
		if err != nil {
			serverErr(w, err)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func apiPodcastSeriesUnsubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	series, err := ctx.Podcast().FindSeries(id)
	if err != nil {
		notFoundErr(w)
	} else {
		err := ctx.Podcast().Unsubscribe(series.SID, ctx.User().Name)
		if err != nil {
			serverErr(w, err)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func apiPodcastSeriesGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	series, err := ctx.Podcast().FindSeries(id)
	if err != nil {
		notFoundErr(w)
	} else {
		view := SeriesView(ctx, series)
		plist := ResolveSeriesPlaylist(ctx, view, r.URL.Path)
		writePlaylist(w, r, plist)
	}
}

func apiPodcastEpisodeGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	episode, err := ctx.Podcast().FindEpisode(id)
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, EpisodeView(ctx, episode))
	}
}

func apiPodcastEpisodeGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	episode, err := ctx.Podcast().FindEpisode(id)
	if err != nil {
		notFoundErr(w)
	} else {
		series, err := ctx.Podcast().FindSeries(episode.SID)
		if err != nil {
			notFoundErr(w)
			return
		}
		plist := ResolveSeriesEpisodePlaylist(ctx,
			SeriesView(ctx, series),
			EpisodeView(ctx, episode),
			r.URL.Path)
		writePlaylist(w, r, plist)
	}
}

// TODO check
//
// PUT /api/radio/1 < Station{}
// 204: no content
// 404: not found
// 500: error
//
// PATCH /api/radio/1 < json+patch > 204
// 204: no content
// 404: not found
// 500: error
//
// DELETE /api/radio/1
// 204: success, no content
// 404: not found
// 500: error
func apiStation(w http.ResponseWriter, r *http.Request, id int) {
	ctx := contextValue(r)
	s, err := ctx.Music().LookupStation(id)
	if err != nil {
		notFoundErr(w)
		return
	}
	if !s.Visible(ctx.User().Name) {
		notFoundErr(w)
		return
	}

	switch r.Method {
	case http.MethodGet:
		RefreshStation(ctx, &s)
		w.WriteHeader(http.StatusOK)
		w.Write(s.Playlist)
	case http.MethodPut:
		var up model.Station
		err := recvStation(w, r, &up)
		if err != nil {
			return
		}
		s.Name = up.Name
		s.Ref = up.Ref
		s.Playlist = up.Playlist
		err = ctx.Music().UpdateStation(&s)
		if err != nil {
			serverErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPatch:
		patch, _ := io.ReadAll(r.Body)
		s.Playlist, err = spiff.Patch(s.Playlist, patch)
		if err != nil {
			serverErr(w, err)
			return
		}
		// unmarshal & resovle
		plist, _ := spiff.Unmarshal(s.Playlist)
		Resolve(ctx, plist)
		if plist.Spiff.Entries == nil {
			plist.Spiff.Entries = []spiff.Entry{}
		}
		// marshal & persist
		s.Playlist, _ = plist.Marshal()
		ctx.Music().UpdateStation(&s)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		err = ctx.Music().DeleteStation(&s)
		if err != nil {
			serverErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "bummer", http.StatusBadRequest)
	}
}

func apiReleaseGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	release, err := ctx.FindRelease(id)
	if err != nil {
		notFoundErr(w)
	} else {
		apiView(w, r, ReleaseView(ctx, release))
	}
}

func apiReleaseGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	release, err := ctx.FindRelease(id)
	if err != nil {
		notFoundErr(w)
	} else {
		view := ReleaseView(ctx, release)
		plist := ResolveReleasePlaylist(ctx, view, r.URL.Path)
		writePlaylist(w, r, plist)
	}
}

func apiTrackLocation(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	uuid := r.PathValue(ParamUUID)
	track, err := ctx.FindTrack("uuid:" + uuid)
	if err != nil {
		notFoundErr(w)
		return
	}
	if track.UUID != uuid {
		accessDenied(w)
		return
	}

	url := ctx.Music().TrackURL(&track)
	doRedirect(w, r, url, http.StatusTemporaryRedirect)
}

func apiMovieLocation(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	uuid := r.PathValue(ParamUUID)
	movie, err := ctx.FindMovie("uuid:" + uuid)
	if err != nil {
		notFoundErr(w)
		return
	}
	if movie.UUID != uuid {
		accessDenied(w)
		return
	}

	url := ctx.Video().MovieURL(movie)
	doRedirect(w, r, url, http.StatusTemporaryRedirect)
}

func apiEpisodeLocation(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	id := r.PathValue(ParamID)
	episode, err := ctx.Podcast().FindEpisode(id)
	if err != nil {
		notFoundErr(w)
	} else {
		url := ctx.Podcast().EpisodeURL(episode)
		doRedirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func apiActivityGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	apiView(w, r, ActivityView(ctx))
}

func apiActivityPost(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)

	var events model.Events
	body, err := io.ReadAll(r.Body)
	if err != nil {
		badRequest(w, err)
		return
	}
	err = json.Unmarshal(body, &events)
	if err != nil {
		badRequest(w, err)
		return
	}

	err = ctx.Activity().CreateEvents(ctx, events)
	if err != nil {
		serverErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func startEnd(r *http.Request) (time.Time, time.Time) {
	// now until 1 year back, limits will apply
	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	s := r.URL.Query().Get(QueryStart)
	if s != "" {
		start = date.ParseDate(s)
	}
	e := r.URL.Query().Get(QueryEnd)
	if e != "" {
		end = date.ParseDate(e)
	}

	return date.StartOfDay(start), date.EndOfDay(end)
}

func apiActivityTracksGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	start, end := startEnd(r)
	apiView(w, r, ActivityTracksView(ctx, start, end))
}

func apiActivityTracksGetResource(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	start, end := startEnd(r)
	res := r.PathValue(ParamRes)

	switch res {
	case "popular":
		apiView(w, r, ActivityPopularTracksView(ctx, start, end))
	case "recent":
		apiView(w, r, ActivityTracksView(ctx, start, end))
	case "playlist":
		apiActivityTracksGetPlaylist(w, r)
	default:
		notFoundErr(w)
	}
}

func apiActivityTracksGetPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	start, end := startEnd(r)
	res := r.PathValue(ParamRes)
	if res == "playlist" {
		res = "recent"
	}

	var tracks *view.ActivityTracks
	switch res {
	case "popular":
		tracks = ActivityPopularTracksView(ctx, start, end)
	case "recent":
		tracks = ActivityTracksView(ctx, start, end)
	default:
		notFoundErr(w)
	}

	plist := ResolveActivityTracksPlaylist(ctx, tracks, res, r.URL.Path)
	writePlaylist(w, r, plist)
}

func apiActivityMoviesGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	start, end := startEnd(r)
	apiView(w, r, ActivityMoviesView(ctx, start, end))
}

func apiActivityReleasesGet(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	start, end := startEnd(r)
	apiView(w, r, ActivityReleasesView(ctx, start, end))
}

func doRedirect(w http.ResponseWriter, r *http.Request, u *url.URL, code int) {
	if u.Scheme == "file" {
		ctx := contextValue(r)
		path := u.Path

		// file URL from bucket is file://{/path/to some/file.ext}
		// with path "/path/to some/file.ext"
		// token signs local file path unescaped
		token, err := ctx.Auth().NewFileToken(path)
		if err != nil {
			serverErr(w, err)
			return
		}

		// /d/path/to%20some/file.ext?token=xyz
		url := strings.Join([]string{"/d", u.EscapedPath(), "?", QueryToken, "=", url.QueryEscape(token)}, "")
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, u.String(), http.StatusTemporaryRedirect)
	}
}

func apiDownload(w http.ResponseWriter, r *http.Request) {
	prefix := "/d"
	path := strings.TrimPrefix(r.URL.Path, prefix)
	http.ServeFile(w, r, path)
}
