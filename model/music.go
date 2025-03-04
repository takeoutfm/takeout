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

package model // import "takeoutfm.dev/takeout/model"

import (
	"github.com/google/uuid"
	g "gorm.io/gorm"
	"takeoutfm.dev/takeout/lib/gorm"
	"time"
)

// Artist info from MusicBrainz.
type Artist struct {
	gorm.Model
	Name           string `gorm:"uniqueIndex:idx_artist_name"`
	SortName       string
	ARID           string `gorm:"uniqueIndex:idx_artist_arid"`
	Disambiguation string
	Country        string
	Area           string
	Date           time.Time
	EndDate        time.Time
	Genre          string
}

type CoverArt interface {
	HasArtwork() bool
	HasFrontArtwork() bool
	HasBackArtwork() bool
	HasOtherArtwork() bool
	HasGroupArtwork() bool
	ArtworkMBIDs() (string, string)
}

// Release info from MusicBrainz.
type Release struct {
	gorm.Model
	Artist         string `gorm:"uniqueIndex:idx_release;index:idx_release_artist"`
	Name           string `gorm:"uniqueIndex:idx_release;index:idx_release_name" sql:"collate:nocase"`
	RGID           string `gorm:"index:idx_release_rgid"`
	REID           string `gorm:"uniqueIndex:idx_release;index:idx_release_reid"`
	Disambiguation string
	Asin           string
	Country        string
	Type           string `gorm:"index:idx_release_type"`
	SecondaryType  string
	Date           time.Time `gorm:"index:idx_release_rgdate"` // rg first release
	ReleaseDate    time.Time `gorm:"index:idx_release_redate"` // re release date
	Status         string
	TrackCount     int
	DiscCount      int
	Artwork        bool
	FrontArtwork   bool
	BackArtwork    bool
	OtherArtwork   string
	GroupArtwork   bool
	Media          []Media `gorm:"-"`
	SingleName     string  `gorm:"index:idx_release_single_name"`
	GroupName      string  `gorm:"index:idx_release_group_name"`
}

func (r Release) HasArtwork() bool {
	return r.Artwork
}

func (r Release) HasFrontArtwork() bool {
	return r.FrontArtwork
}

func (r Release) HasBackArtwork() bool {
	return r.BackArtwork
}

func (r Release) HasOtherArtwork() bool {
	return r.OtherArtwork != ""
}

func (r Release) HasGroupArtwork() bool {
	return r.GroupArtwork
}

func (r Release) ArtworkMBIDs() (string, string) {
	return r.REID, r.RGID
}

func (r Release) Official() bool {
	return r.Status == "Official"
}

// Release Media from MusicBrainz.
type Media struct {
	gorm.Model
	REID       string `gorm:"uniqueIndex:idx_media"`
	Name       string `gorm:"uniqueIndex:idx_media"`
	Position   int    `gorm:"uniqueIndex:idx_media"`
	Format     string
	TrackCount int
}

// Popular tracks for an artist from Last.fm.
type Popular struct {
	gorm.Model
	Artist string `gorm:"uniqueIndex:idx_popular"`
	Title  string `gorm:"uniqueIndex:idx_popular"`
	Rank   int
}

func (Popular) TableName() string {
	return "popular" // not populars
}

// Similar artist info from Last.fm
type Similar struct {
	gorm.Model
	Artist string `gorm:"uniqueIndex:idx_similar"`
	ARID   string `gorm:"uniqueIndex:idx_similar"`
	Rank   int
}

func (Similar) TableName() string {
	return "similar" // not similars
}

// Artist tags from MusicBrainz.
type ArtistTag struct {
	gorm.Model
	Artist string `gorm:"uniqueIndex:idx_tag"`
	Tag    string `gorm:"uniqueIndex:idx_tag"`
	Count  int
}

// Tracks from S3 bucket object names. Naming is adjusted based on
// data from MusicBrainz.
type Track struct {
	gorm.Model
	UUID         string `gorm:"index:idx_track_uuid"`
	Artist       string `spiff:"creator" gorm:"index:idx_track_artist"`
	Release      string `gorm:"index:idx_track_release"`
	Date         string `gorm:"index:idx_track_date"`
	TrackNum     int    `spiff:"tracknum"`
	DiscNum      int
	Title        string `spiff:"title" gorm:"index:idx_track_title"`
	Key          string // TODO - unique constraint
	Size         int64
	ETag         string
	LastModified time.Time
	TrackCount   int
	DiscCount    int
	REID         string `gorm:"index:idx_track_reid"`
	RGID         string `gorm:"index:idx_track_rgid"`
	RID          string `gorm:"index:idx_track_rid"`  // recording id
	ARID         string `gorm:"index:idx_track_arid"` // TODO only for local right now
	MediaTitle   string
	ReleaseTitle string `spiff:"album"`
	TrackArtist  string // artist with featured artists
	ReleaseDate  time.Time
	Artwork      bool
	FrontArtwork bool
	BackArtwork  bool
	OtherArtwork string
	GroupArtwork bool
}

func (t *Track) BeforeCreate(tx *g.DB) (err error) {
	t.UUID = uuid.NewString()
	return
}

// Prefer A feat. B instead of just A.
func (t Track) PreferredArtist() string {
	if t.TrackArtist != "" && t.TrackArtist != t.Artist {
		return t.TrackArtist
	}
	return t.Artist
}

func (t Track) HasArtwork() bool {
	return t.Artwork
}

func (t Track) HasFrontArtwork() bool {
	return t.FrontArtwork
}

func (t Track) HasBackArtwork() bool {
	return t.BackArtwork
}

func (t Track) HasOtherArtwork() bool {
	return t.OtherArtwork != ""
}

func (t Track) HasGroupArtwork() bool {
	return t.GroupArtwork
}

func (t Track) ArtworkMBIDs() (string, string) {
	return t.REID, t.RGID
}

type Playlist struct {
	gorm.Model
	User       string `gorm:"uniqueIndex:idx_playlist"`
	Name       string `gorm:"uniqueIndex:idx_playlist"`
	Playlist   []byte
	TrackCount int
}

type Station struct {
	gorm.Model
	User        string `gorm:"uniqueIndex:idx_station" json:"-"`
	Name        string `gorm:"uniqueIndex:idx_station"`
	Creator     string
	Ref         string `json:"-"`
	Shared      bool   `json:"-"`
	Type        string
	Image       string
	Description string
	Playlist    []byte `json:"-"`
}

func (s *Station) Visible(user string) bool {
	return s.User == user || s.Shared
}

type ArtistImage struct {
	gorm.Model
	Artist string `gorm:"uniqueIndex:idx_artist_img"`
	URL    string `gorm:"uniqueIndex:idx_artist_img"`
	Source string `gorm:"uniqueIndex:idx_artist_img"`
	Rank   int
}

type ArtistBackground struct {
	gorm.Model
	Artist string `gorm:"uniqueIndex:idx_artist_bg"`
	URL    string `gorm:"uniqueIndex:idx_artist_bg"`
	Source string `gorm:"uniqueIndex:idx_artist_bg"`
	Rank   int
}

// type Scrobble struct {
// 	gorm.Model
// 	User    string
// 	Artist  string
// 	Release string
// 	Title   string
// 	Date    time.Time
// }
