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

// Package favorite manages user favorite data
package favorite

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"

	"errors"
)

var (
	ErrInvalidUser               = errors.New("invalid user")
	ErrInvalidTrackFavorite      = errors.New("invalid track favorite")
	ErrInvalidArtistFavorite     = errors.New("invalid artist favorite")
	ErrInvalidMovieFavorite      = errors.New("invalid movie favorite")
	ErrInvalidTVSeriesFavorite   = errors.New("invalid tvseries favorite")
	ErrTrackNotFound             = errors.New("track not found")
	ErrMovieNotFound             = errors.New("movie not found")
	ErrTVSeriesNotFound          = errors.New("tvseries not found")
	ErrArtistNotFound            = errors.New("artist not found")
	ErrTrackFavoriteExists       = errors.New("track favorite already exists")
	ErrMovieFavoriteExists       = errors.New("movie favorite already exists")
	ErrArtistFavoriteExists      = errors.New("artist favorite already exists")
	ErrTVSeriesFavoriteExists    = errors.New("tvseries favorite already exists")
	ErrArtistFavoriteDuplicate   = errors.New("artist favorite duplicate")
	ErrMovieFavoriteDuplicate    = errors.New("movie favorite duplicate")
	ErrTrackFavoriteDuplicate    = errors.New("track favorite duplicate")
	ErrTVSeriesFavoriteDuplicate = errors.New("tvseries favorite duplicate")
)

type Context interface {
	User() auth.User
	Music() *music.Music
	Film() *film.Film
	TV() *tv.TV
}

type Favorite struct {
	config *config.Config
	db     *gorm.DB
}

func NewFavorite(config *config.Config) *Favorite {
	return &Favorite{
		config: config,
	}
}

func (fav *Favorite) Open() error {
	return fav.openDB()
}

func (fav *Favorite) Close() {
	fav.closeDB()
}

func (fav *Favorite) DeleteArtistFavorite(ctx Context, artist Artist) error {
	user := ctx.User()
	artists := fav.artistFavorites(user.Name, fav.config.Favorite.ArtistLimit)
	for _, a := range artists {
		if a.IsFavorite(artist) == false {
			continue
		}
		if err := fav.deleteArtistFavorite(&a); err != nil {
			return err
		}
	}
	return nil
}

func (fav *Favorite) DeleteMovieFavorite(ctx Context, movie Movie) error {
	user := ctx.User()
	movies := fav.movieFavorites(user.Name, fav.config.Favorite.MovieLimit)
	for _, m := range movies {
		if m.IsFavorite(movie) == false {
			continue
		}
		if err := fav.deleteMovieFavorite(&m); err != nil {
			return err
		}
	}
	return nil
}

func (fav *Favorite) DeleteTrackFavorite(ctx Context, track Track) error {
	user := ctx.User()
	tracks := fav.trackFavorites(user.Name, fav.config.Favorite.TrackLimit)
	for _, t := range tracks {
		if t.IsFavorite(track) == false {
			continue
		}
		if err := fav.deleteTrackFavorite(&t); err != nil {
			return err
		}
	}
	return nil
}

func (fav *Favorite) DeleteTVSeriesFavorite(ctx Context, series TVSeries) error {
	user := ctx.User()
	list := fav.tvSeriesFavorites(user.Name, fav.config.Favorite.TVSeriesLimit)
	for _, s := range list {
		if s.IsFavorite(series) == false {
			continue
		}
		if err := fav.deleteTVSeriesFavorite(&s); err != nil {
			return err
		}
	}
	return nil
}

func (fav *Favorite) DeleteUserFavorites(ctx Context) error {
	user := ctx.User()
	err := fav.deleteMovieFavorites(user.Name)
	if err != nil {
		log.Println("movie favorite delete error: ", err)
		return err
	}
	err = fav.deleteTrackFavorites(user.Name)
	if err != nil {
		log.Println("track favorite error: ", err)
		return err
	}
	err = fav.deleteArtistFavorites(user.Name)
	if err != nil {
		log.Println("artist favorite error: ", err)
		return err
	}
	err = fav.deleteTVSeriesFavorites(user.Name)
	if err != nil {
		log.Println("tvseries delete error: ", err)
		return err
	}
	return nil
}

func (fav *Favorite) resolveMovieFavorite(ctx Context, m MovieFavorite) (Movie, error) {
	f := ctx.Film()
	if m.IMID == "" {
		return Movie{}, ErrMovieNotFound
	}
	movie, err := f.FindMovie(m.IMID)
	if err != nil {
		return Movie{}, err
	}
	return movie, nil
}

func (fav *Favorite) resolveTVSeriesFavorite(ctx Context, s TVSeriesFavorite) (TVSeries, error) {
	tv := ctx.TV()
	if s.TVID <= 0 {
		return TVSeries{}, ErrTVSeriesNotFound
	}
	tvSeries, err := tv.FindSeries(fmt.Sprintf("tvid:%d", s.TVID))
	if err != nil {
		return TVSeries{}, err
	}
	return tvSeries, nil
}

func (fav *Favorite) resolveArtistFavorite(ctx Context, a ArtistFavorite) (Artist, error) {
	m := ctx.Music()
	if a.ARID == "" {
		return Artist{}, ErrArtistNotFound
	}
	artist, err := m.FindArtist(a.ARID)
	if err != nil {
		return Artist{}, err
	}
	return artist, nil
}

func (fav *Favorite) resolveTrackFavorite(ctx Context, t TrackFavorite) (Track, error) {
	m := ctx.Music()

	var err error
	var track Track
	if t.RID != "" {
		track, err = m.FindTrack(t.RID)
		if err != nil {
			log.Printf("track favorite %d, RID %s not found\n", t.ID, t.RID)
			return Track{}, err
		}
	} else if t.ETag != "" {
		track, err = m.LookupETag(t.ETag)
		if err != nil {
			log.Printf("track favorite %d, etag %s not found\n", t.ID, t.ETag)
			return Track{}, err
		}
	}
	return track, nil
}

func (fav *Favorite) resolveMovieFavorites(ctx Context, list []MovieFavorite) []Movie {
	movies := []Movie{}
	for _, m := range list {
		movie, err := fav.resolveMovieFavorite(ctx, m)
		if err == nil {
			movies = append(movies, movie)
		}
	}
	return movies
}

func (fav *Favorite) resolveTVSeriesFavorites(ctx Context, list []TVSeriesFavorite) []TVSeries {
	tvSeries := []TVSeries{}
	for _, s := range list {
		series, err := fav.resolveTVSeriesFavorite(ctx, s)
		if err == nil {
			tvSeries = append(tvSeries, series)
		}
	}
	return tvSeries
}

func (fav *Favorite) resolveTrackFavorites(ctx Context, list []TrackFavorite) []Track {
	tracks := []Track{}
	for _, t := range list {
		track, err := fav.resolveTrackFavorite(ctx, t)
		if err == nil {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

func (fav *Favorite) resolveArtistFavorites(ctx Context, list []ArtistFavorite) []Artist {
	artists := []Artist{}
	for _, a := range list {
		artist, err := fav.resolveArtistFavorite(ctx, a)
		if err == nil {
			artists = append(artists, artist)
		}
	}
	return artists
}

func (fav *Favorite) Movies(ctx Context) []Movie {
	user := ctx.User()
	list := fav.movieFavorites(user.Name, fav.config.Favorite.MovieLimit)
	return fav.resolveMovieFavorites(ctx, list)
}

func (fav *Favorite) Shows(ctx Context) []TVSeries {
	user := ctx.User()
	list := fav.tvSeriesFavorites(user.Name, fav.config.Favorite.TVSeriesLimit)
	return fav.resolveTVSeriesFavorites(ctx, list)
}

func (fav *Favorite) Artists(ctx Context) []Artist {
	user := ctx.User()
	list := fav.artistFavorites(user.Name, fav.config.Favorite.ArtistLimit)
	return fav.resolveArtistFavorites(ctx, list)
}

func (fav *Favorite) Tracks(ctx Context) []Track {
	user := ctx.User()
	list := fav.trackFavorites(user.Name, fav.config.Favorite.TrackLimit)
	return fav.resolveTrackFavorites(ctx, list)
}

func (fav *Favorite) IsFavoriteArtist(ctx Context, artist Artist) bool {
	user := ctx.User()
	artists := fav.artistFavorites(user.Name, fav.config.Favorite.ArtistLimit)
	for _, a := range artists {
		if a.IsFavorite(artist) {
			return true
		}
	}
	return false
}

func (fav *Favorite) IsFavoriteMovie(ctx Context, movie Movie) bool {
	user := ctx.User()
	movies := fav.movieFavorites(user.Name, fav.config.Favorite.MovieLimit)
	for _, m := range movies {
		if m.IsFavorite(movie) {
			return true
		}
	}
	return false
}

func (fav *Favorite) IsFavoriteTrack(ctx Context, track Track) bool {
	user := ctx.User()
	tracks := fav.trackFavorites(user.Name, fav.config.Favorite.TrackLimit)
	for _, t := range tracks {
		if t.IsFavorite(track) {
			return true
		}
	}
	return false
}

func (fav *Favorite) IsFavoriteTVSeries(ctx Context, series TVSeries) bool {
	user := ctx.User()
	shows := fav.tvSeriesFavorites(user.Name, fav.config.Favorite.TVSeriesLimit)
	for _, s := range shows {
		if s.IsFavorite(series) {
			return true
		}
	}
	return false
}

func (fav *Favorite) CreateFavorites(ctx Context, favorites Favorites) error {
	if err := fav.prepareFavorites(ctx, favorites); err != nil {
		return err
	}
	return fav.InTx(func(tx *Favorite) error {
		return tx.insertFavorites(favorites)
	})
}

func (fav *Favorite) prepareFavorites(ctx Context, favorites Favorites) error {
	user := ctx.User()
	if user.Name == "" {
		return ErrInvalidUser
	}

	now := time.Now()

	var movieMap = make(map[string]bool)

	for i := range favorites.Movies {
		f := &favorites.Movies[i]
		f.User = user.Name
		if f.Date.IsZero() {
			f.Date = now
		}
		if f.ETag != "" {
			// resolve using ETag
			movie, err := ctx.Film().LookupETag(f.ETag)
			if err != nil {
				return fmt.Errorf("movie etag %s: %w", f.ETag, err)
			}
			f.IMID = movie.IMID
			f.TMID = movie.TMID
			if fav.IsFavoriteMovie(ctx, movie) {
				return ErrMovieFavoriteExists
			}
		} else if f.IMID != "" {
			movie, err := ctx.Film().LookupIMID(f.IMID)
			if err != nil {
				return fmt.Errorf("movie imid %s: %w", f.IMID, err)
			}
			f.TMID = movie.TMID
			if fav.IsFavoriteMovie(ctx, movie) {
				return ErrMovieFavoriteExists
			}
		}

		if f.IMID != "" {
			if movieMap[f.IMID] {
				return ErrMovieFavoriteDuplicate
			}
			movieMap[f.IMID] = true
		}

		// check valid after since having an ETag only is not valid,
		// need to resolve first and call valid after.
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidMovieFavorite)
		}
	}

	var artistMap = make(map[string]bool)

	for i := range favorites.Artists {
		f := &favorites.Artists[i]
		f.User = user.Name
		if f.Date.IsZero() {
			f.Date = now
		}
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidArtistFavorite)
		}
		artist, err := ctx.Music().LookupARID(f.ARID)
		if err != nil {
			return fmt.Errorf("artist arid %s: %w", f.ARID, err)
		}
		if fav.IsFavoriteArtist(ctx, artist) {
			return ErrArtistFavoriteExists
		}
		if artistMap[f.ARID] {
			return ErrArtistFavoriteDuplicate
		}
		artistMap[f.ARID] = true
	}

	var showMap = make(map[int64]bool)

	for i := range favorites.Shows {
		f := &favorites.Shows[i]
		f.User = user.Name
		if f.Date.IsZero() {
			f.Date = now
		}
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidTVSeriesFavorite)
		}
		series, err := ctx.TV().LookupTVID(int(f.TVID))
		if err != nil {
			return fmt.Errorf("tvseries tvid %d: %w", f.TVID, err)
		}
		if fav.IsFavoriteTVSeries(ctx, series) {
			return ErrTVSeriesFavoriteExists
		}
		if showMap[f.TVID] {
			return ErrTVSeriesFavoriteDuplicate
		}
		showMap[f.TVID] = true
	}

	var trackMap = make(map[string]bool)

	for i := range favorites.Tracks {
		f := &favorites.Tracks[i]
		f.User = user.Name
		if f.Date.IsZero() {
			f.Date = now
		}
		if f.ETag != "" {
			// resolve using ETag
			track, err := ctx.Music().LookupETag(f.ETag)
			if err != nil {
				return fmt.Errorf("track etag %s: %w", f.ETag, err)
			}
			f.RID = track.RID
			f.RGID = track.RGID
			if fav.IsFavoriteTrack(ctx, track) {
				return ErrTrackFavoriteExists
			}
		} else if f.RID != "" {
			track, err := ctx.Music().LookupRID(f.RID)
			if err != nil {
				return fmt.Errorf("track rid %s: %w", f.RID, err)
			}
			f.RGID = track.RGID
			if fav.IsFavoriteTrack(ctx, track) {
				return ErrTrackFavoriteExists
			}
		}

		if f.RID != "" {
			if trackMap[f.RID] {
				return ErrTrackFavoriteDuplicate
			}
			trackMap[f.RID] = true
		}

		// check valid after since having an ETag only is not valid,
		// need to resolve first and call valid after.
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidTrackFavorite)
		}
	}

	return nil
}

func (fav *Favorite) insertFavorites(favorites Favorites) error {
	for i := range favorites.Movies {
		f := &favorites.Movies[i]
		if err := fav.createMovieFavorite(f); err != nil {
			return err
		}
	}

	for i := range favorites.Artists {
		f := &favorites.Artists[i]
		if err := fav.createArtistFavorite(f); err != nil {
			return err
		}
	}

	for i := range favorites.Shows {
		f := &favorites.Shows[i]
		if err := fav.createTVSeriesFavorite(f); err != nil {
			return err
		}
	}

	for i := range favorites.Tracks {
		f := &favorites.Tracks[i]
		if err := fav.createTrackFavorite(f); err != nil {
			return err
		}
	}

	return nil
}
