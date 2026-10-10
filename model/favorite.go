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
	User string `gorm:"uniqueIndex:idx_movie_favorites_user_im_id,priority:1" json:"-"`
	Date time.Time
	TMID int64
	IMID string `gorm:"uniqueIndex:idx_movie_favorites_user_im_id,priority:2"`
	ETag string `gorm:"-"`
}

func (f *MovieFavorite) IsFavorite(m Movie) bool {
	return f.IMID != "" && f.IMID == m.IMID
}

func (f *MovieFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && f.IMID != ""
}

type TVSeriesFavorite struct {
	gorm.Model
	User string `gorm:"uniqueIndex:idx_tv_series_favorites_user_tv_id,priority:1" json:"-"`
	Date time.Time
	TVID int64 `gorm:"uniqueIndex:idx_tv_series_favorites_user_tv_id,priority:2"`
}

func (f *TVSeriesFavorite) IsFavorite(s TVSeries) bool {
	return f.TVID > 0 && f.TVID == s.TVID
}

func (f *TVSeriesFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && f.TVID > 0
}

type TrackFavorite struct {
	gorm.Model
	User string `gorm:"uniqueIndex:idx_track_favorites_user_r_id,priority:1" json:"-"`
	Date time.Time
	RID  string `gorm:"uniqueIndex:idx_track_favorites_user_r_id,priority:2"`
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
	User string `gorm:"uniqueIndex:idx_artist_favorites_user_ar_id,priority:1" json:"-"`
	Date time.Time
	ARID string `gorm:"uniqueIndex:idx_artist_favorites_user_ar_id,priority:2"`
}

func (f *ArtistFavorite) IsFavorite(a Artist) bool {
	return f.ARID != "" && f.ARID == a.ARID
}

func (f *ArtistFavorite) IsValid() bool {
	return f.User != "" && f.Date.IsZero() == false && f.ARID != ""
}
