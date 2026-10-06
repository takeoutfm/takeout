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
	ErrInvalidUser             = errors.New("invalid user")
	ErrInvalidTrackFavorite    = errors.New("invalid track favorite")
	ErrInvalidArtistFavorite   = errors.New("invalid artist favorite")
	ErrInvalidMovieFavorite    = errors.New("invalid movie favorite")
	ErrInvalidTVSeriesFavorite = errors.New("invalid tvseries favorite")
	ErrTrackNotFound           = errors.New("track not found")
	ErrMovieNotFound           = errors.New("movie not found")
	ErrTVSeriesNotFound        = errors.New("tvseries not found")
	ErrArtistNotFound          = errors.New("artist not found")
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
	user := ctx.User()
	for _, f := range favorites.Movies {
		f.User = user.Name
		if f.ETag != "" {
			// resolve using ETag
			movie, err := ctx.Film().LookupETag(f.ETag)
			if err != nil {
				return err
			}
			f.IMID = movie.IMID
			f.TMID = movie.TMID
		}
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidMovieFavorite)
		}
		err := fav.createMovieFavorite(&f)
		if err != nil {
			return err
		}
	}

	for _, f := range favorites.Artists {
		f.User = user.Name
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidArtistFavorite)
		}
		err := fav.createArtistFavorite(&f)
		if err != nil {
			return err
		}
	}

	for _, f := range favorites.Shows {
		f.User = user.Name
		if f.IsValid() == false {
			return fmt.Errorf("favorite is %#v: %w", f, ErrInvalidTVSeriesFavorite)
		}
		err := fav.createTVSeriesFavorite(&f)
		if err != nil {
			return err
		}
	}

	for _, f := range favorites.Tracks {
		f.User = user.Name
		if f.ETag != "" {
			// resolve using ETag
			track, err := ctx.Music().LookupETag(f.ETag)
			if err != nil {
				return err
			}
			f.RID = track.RID
			f.RGID = track.RGID
		}
		if f.IsValid() == false {
			return fmt.Errorf("event is %#v: %w", f, ErrInvalidTrackFavorite)
		}
		err := fav.createTrackFavorite(&f)
		if err != nil {
			return err
		}
	}

	return nil
}
