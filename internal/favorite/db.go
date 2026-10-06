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

package favorite

import (
	"errors"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	. "takeoutfm.dev/takeout/model"
)

func (fav *Favorite) openDB() (err error) {
	cfg := fav.config.Favorite.DB.GormConfig()

	if fav.config.Activity.DB.Driver == "sqlite3" {
		fav.db, err = gorm.Open(sqlite.Open(fav.config.Favorite.DB.Source), cfg)
	} else {
		err = errors.New("driver not supported")
	}

	if err != nil {
		return
	}

	fav.db.AutoMigrate(&ArtistFavorite{}, &MovieFavorite{},  &TrackFavorite{}, &TVSeriesFavorite{})
	return
}

func (fav *Favorite) closeDB() {
	conn, err := fav.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

func favoritesOf[T any](db *gorm.DB, user string, limit int) []T {
	var favorites []T
	db.Where("user = ?", user).
		Order("date desc").Limit(limit).Find(&favorites)
	return favorites
}

func deleteFavoritesByUser[T any](db *gorm.DB, user string) error {
	return db.Unscoped().Where("user = ?", user).Delete(new(T)).Error
}

func createFavorite[T any](db *gorm.DB, f *T) error {
	return db.Create(f).Error
}

func deleteFavorite[T any](db *gorm.DB, f *T) error {
	return db.Unscoped().Delete(f).Error
}

func (fav *Favorite) trackFavorites(user string, limit int) []TrackFavorite {
	return favoritesOf[TrackFavorite](fav.db, user, limit)
}

func (fav *Favorite) artistFavorites(user string, limit int) []ArtistFavorite {
	return favoritesOf[ArtistFavorite](fav.db, user, limit)
}

func (fav *Favorite) movieFavorites(user string, limit int) []MovieFavorite {
	return favoritesOf[MovieFavorite](fav.db, user, limit)
}

func (fav *Favorite) tvSeriesFavorites(user string, limit int) []TVSeriesFavorite {
	return favoritesOf[TVSeriesFavorite](fav.db, user, limit)
}

func (fav *Favorite) deleteTrackFavorites(user string) error {
	return deleteFavoritesByUser[TrackFavorite](fav.db, user)
}

func (fav *Favorite) deleteArtistFavorites(user string) error {
	return deleteFavoritesByUser[ArtistFavorite](fav.db, user)
}

func (fav *Favorite) deleteMovieFavorites(user string) error {
	return deleteFavoritesByUser[MovieFavorite](fav.db, user)
}

func (fav *Favorite) deleteTVSeriesFavorites(user string) error {
	return deleteFavoritesByUser[TVSeriesFavorite](fav.db, user)
}

func (fav *Favorite) createTrackFavorite(f *TrackFavorite) error {
	return createFavorite(fav.db, f)
}

func (fav *Favorite) createArtistFavorite(f *ArtistFavorite) error {
	return createFavorite(fav.db, f)
}

func (fav *Favorite) createMovieFavorite(f *MovieFavorite) error {
	return createFavorite(fav.db, f)
}

func (fav *Favorite) createTVSeriesFavorite(f *TVSeriesFavorite) error {
	return createFavorite(fav.db, f)
}

func (fav *Favorite) deleteTrackFavorite(f *TrackFavorite) error {
	return deleteFavorite(fav.db, f)
}

func (fav *Favorite) deleteArtistFavorite(f *ArtistFavorite) error {
	return deleteFavorite(fav.db, f)
}

func (fav *Favorite) deleteMovieFavorite(f *MovieFavorite) error {
	return deleteFavorite(fav.db, f)
}

func (fav *Favorite) deleteTVSeriesFavorite(f *TVSeriesFavorite) error {
	return deleteFavorite(fav.db, f)
}
