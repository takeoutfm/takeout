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

package model

import (
	"time"

	"github.com/takeoutfm/takeout/lib/gorm"
)

// Activity data should be long-lived and w/o internal sync identifiers.  Use
// external globally unique IDs and stable media metadata. It should all be
// meaningful even after a full re-sync.

// Scrobble is a track listening event.
//
// Spec location: https://www.last.fm/api/show/track.scrobble
// not using: context, streamId, chosenByUser
type Scrobble struct {
	Artist      string    `json:""` // Required
	Track       string    `json:""` // Required
	Timestamp   time.Time `json:""` // Required
	Album       string    `json:""` // Optional
	AlbumArtist string    `json:""` // Optional
	TrackNumber int       `json:""` // Optional
	Duration    int       `json:""` // Optional - Length of track in seconds
	MBID        string    `json:""` // Optional - recording (or track?) MBID
}

func (s Scrobble) PreferredArtist() string {
	if s.AlbumArtist != "" {
		return s.AlbumArtist
	}
	return s.Artist
}

type Events struct {
	MovieEvents   []MovieEvent
	EpisodeEvents []EpisodeEvent
	TrackEvents   []TrackEvent
}

type MovieEvent struct {
	gorm.Model
	User string    `gorm:"index:idx_movie_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_movie_date"`
	TMID string
	IMID string
	ETag string `gorm:"-"`
}

func (m *MovieEvent) IsValid() bool {
	return m.User != "" && m.Date.IsZero() == false && (m.TMID != "" || m.IMID != "")
}

type TrackEvent struct {
	gorm.Model
	User string    `gorm:"index:idx_track_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_track_date"` // TODO dup index name w/ music
	RID  string
	RGID string
	ETag string `gorm:"-"`
}

func (t *TrackEvent) IsValid() bool {
	return t.User != "" && t.Date.IsZero() == false && t.RID != "" && t.RGID != ""
}

type EpisodeEvent struct {
	gorm.Model
	User string    `gorm:"index:idx_episode_user" json:"-"`
	Date time.Time `gorm:"uniqueIndex:idx_episode_date"`
	EID  string
}

func (e *EpisodeEvent) IsValid() bool {
	return e.User != "" && e.Date.IsZero() == false && e.EID != ""
}

type ActivityArtist struct {
	Artist Artist
	Count  int
}

type ActivityRelease struct {
	Release Release
	Count   int
}

type ActivityTrack struct {
	Track Track
	Count int
}

type ActivityMovie struct {
	Movie Movie
	Count int
}

type ActivityEpisode struct {
	Episode Episode
	Count   int
}
