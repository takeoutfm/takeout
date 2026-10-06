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
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/model"
)

func makeFavorite(t *testing.T) *Favorite {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	f := NewFavorite(config)
	err = f.Open()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestArtistFavoriteAdd(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	favorite := model.ArtistFavorite{
		User: user,
		Date: time.Now(),
		ARID: "3b3827b3-2f07-47ee-ab4b-3f82512b701d",
	}

	err := fav.createArtistFavorite(&favorite)
	if err != nil {
		t.Fatal(err)
	}
	if favorite.ID == 0 {
		t.Error("expect favorite ID")
	}

	list := fav.artistFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	if list[0].ARID != "3b3827b3-2f07-47ee-ab4b-3f82512b701d" {
		t.Fatal("expected to find favorite ARID")
	}

}

func TestArtistFavoriteDelete(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	list := fav.artistFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	err := fav.deleteArtistFavorite(&list[0])
	if err != nil {
		t.Fatal(err)
	}
}

func TestMovieFavoriteAdd(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	favorite := model.MovieFavorite{
		User: user,
		Date: time.Now(),
		IMID: "tt0076759",
	}

	err := fav.createMovieFavorite(&favorite)
	if err != nil {
		t.Fatal(err)
	}
	if favorite.ID == 0 {
		t.Error("expect favorite ID")
	}

	list := fav.movieFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	if list[0].IMID != "tt0076759" {
		t.Fatal("expected to find favorite IMID")
	}
}

func TestMovieFavoriteDelete(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	list := fav.movieFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	err := fav.deleteMovieFavorite(&list[0])
	if err != nil {
		t.Fatal(err)
	}
}

func TestTrackFavoriteAdd(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	favorite := model.TrackFavorite{
		User: user,
		Date: time.Now(),
		RID:  "475e80fa-ef76-4706-a384-5764f4a860e1",
		RGID: "3b6d0276-cd42-452f-b6cf-740072d83ffc",
		ETag: "test etag",
	}

	err := fav.createTrackFavorite(&favorite)
	if err != nil {
		t.Fatal(err)
	}
	if favorite.ID == 0 {
		t.Error("expect favorite ID")
	}

	list := fav.trackFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	if list[0].RID != "475e80fa-ef76-4706-a384-5764f4a860e1" {
		t.Fatal("expected to find favorite RID")
	}
}

func TestTrackFavoriteDelete(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	list := fav.trackFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	err := fav.deleteTrackFavorite(&list[0])
	if err != nil {
		t.Fatal(err)
	}
}

func TestTVSeriesFavoriteAdd(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	favorite := model.TVSeriesFavorite{
		User: user,
		Date: time.Now(),
		TVID: 1398,
	}

	err := fav.createTVSeriesFavorite(&favorite)
	if err != nil {
		t.Fatal(err)
	}
	if favorite.ID == 0 {
		t.Error("expect favorite ID")
	}

	list := fav.tvSeriesFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	if list[0].TVID != 1398 {
		t.Fatal("expected to find favorite TVID")
	}
}

func TestTVSeriesFavoriteDelete(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	list := fav.tvSeriesFavorites(user, 1)
	if len(list) == 0 {
		t.Fatal("expected to find favorite")
	}

	err := fav.deleteTVSeriesFavorite(&list[0])
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteUserFavorites(t *testing.T) {
	user := "testuser"
	fav := makeFavorite(t)

	if err := fav.createArtistFavorite(&model.ArtistFavorite{
		User: user,
		Date: time.Now(),
		ARID: "3b3827b3-2f07-47ee-ab4b-3f82512b701d",
	}); err != nil {
		t.Fatal(err)
	}

	if err := fav.createMovieFavorite(&model.MovieFavorite{
		User: user,
		Date: time.Now(),
		IMID: "tt0076759",
	}); err != nil {
		t.Fatal(err)
	}

	if err := fav.createTrackFavorite(&model.TrackFavorite{
		User: user,
		Date: time.Now(),
		RID:  "475e80fa-ef76-4706-a384-5764f4a860e1",
		RGID: "3b6d0276-cd42-452f-b6cf-740072d83ffc",
		ETag: "test etag",
	}); err != nil {
		t.Fatal(err)
	}

	if err := fav.createTVSeriesFavorite(&model.TVSeriesFavorite{
		User: user,
		Date: time.Now(),
		TVID: 1398,
	}); err != nil {
		t.Fatal(err)
	}

	if err := fav.deleteArtistFavorites(user); err != nil {
		t.Fatal(err)
	}
	if err := fav.deleteMovieFavorites(user); err != nil {
		t.Fatal(err)
	}
	if err := fav.deleteTrackFavorites(user); err != nil {
		t.Fatal(err)
	}
	if err := fav.deleteTVSeriesFavorites(user); err != nil {
		t.Fatal(err)
	}

	if len(fav.artistFavorites(user, 999)) != 0 {
		t.Fatal("expected zero favorites")
	}
	if len(fav.movieFavorites(user, 999)) != 0 {
		t.Fatal("expected zero favorites")
	}
	if len(fav.trackFavorites(user, 999)) != 0 {
		t.Fatal("expected zero favorites")
	}
	if len(fav.tvSeriesFavorites(user, 999)) != 0 {
		t.Fatal("expected zero favorites")
	}
}
