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

package model // import "takeoutfm.dev/takeout/model"

import (
	"time"

	"takeoutfm.dev/takeout/lib/gorm"
)

// Favorite data should be long-lived and w/o internal sync identifiers.  Use
// external globally unique IDs and stable media metadata. It should all be
// meaningful even after a full re-sync.

type Favorites struct {
	Artists []ArtistFavorite
	Movies  []MovieFavorite
	Shows   []TVSeriesFavorite
	Tracks  []TrackFavorite
}

type MovieFavorite struct {
	gorm.Model
	User string    `gorm:"index:idx_movie_fav_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_movie_fav_date"`
	TMID int64
	IMID string
	ETag string `gorm:"-"`
}

func (f *MovieFavorite) IsFavorite(m Movie) bool {
	return (f.IMID != "" && f.IMID == m.IMID) || (f.TMID > 0 && f.TMID == m.TMID)
}

func (f *MovieFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && (f.TMID > 0 || f.IMID != "")
}

type TVSeriesFavorite struct {
	gorm.Model
	User string    `gorm:"index:idx_tvseries_fav_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_tvseries_fav_date"`
	TVID int64
}

func (f *TVSeriesFavorite) IsFavorite(s TVSeries) bool {
	return f.TVID > 0 && f.TVID == s.TVID
}

func (f *TVSeriesFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && f.TVID > 0
}

type TrackFavorite struct {
	gorm.Model
	User string    `gorm:"index:idx_track_fav_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_track_fav_date"`
	RID  string
	RGID string
	ETag string `gorm:"-"`
}

func (f *TrackFavorite) IsFavorite(t Track) bool {
	return f.RID != "" && f.RID == t.RID
}

func (f *TrackFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && f.RID != "" /*&& f.RGID != ""*/
}

type ArtistFavorite struct {
	gorm.Model
	User string    `gorm:"index:idx_artist_fav_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_artist_fav_date"`
	ARID string
}

func (f *ArtistFavorite) IsFavorite(a Artist) bool {
	return f.ARID != "" && f.ARID == a.ARID
}

func (f *ArtistFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && f.ARID != ""
}
