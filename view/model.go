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

// Package view is the TakeoutFM API viewmodel.
package view // import "takeoutfm.dev/takeout/view"

import (
	"time"

	"takeoutfm.dev/takeout/model"
)

type TracksFunc func() []model.Track

type TrackList struct {
	Title  string
	Tracks TracksFunc
}

type Index struct {
	Time               int64
	HasMusic           bool
	HasMovies          bool
	HasShows           bool
	HasPodcasts        bool
	HasPlaylists       bool
	HasRecommendMovies bool
	HasGuide           bool
	MovieGenres        []string
}

// Guide is what's on the channels between Start and End.
type Guide struct {
	Time     int64 // server time in milliseconds; use it, not the device clock
	Start    time.Time
	End      time.Time
	Channels []Channel
}

// Channel is a channel and its programs in the requested range.
type Channel struct {
	Name     string // unique; a channel's identity
	Programs []Program
}

const ProgramOffAir = "offair"

// Program is one program on a channel. The gap between its End and the next program's
// Start is a break. Kind is movie, episode or offair, when the title has left
// the library.
type Program struct {
	Kind       string
	Block      string `json:",omitempty"`
	Start      time.Time
	End        time.Time
	Title      string
	Subtitle   string `json:",omitempty"`
	Rating     string `json:",omitempty"`
	Image      string `json:",omitempty"`
	Background string `json:",omitempty"`
	ETag       string `json:",omitempty"`
	Size       int64  `json:",omitempty"`
	Location   string `json:",omitempty"`
	TargetID   uint   `json:",omitempty"` // row for Kind: /api/movies/{id} or /api/tv/episodes/{id}
	TMID       int64  `json:",omitempty"`
	TVID       int64  `json:",omitempty"`
	Season     int    `json:",omitempty"`
	Episode    int    `json:",omitempty"`
}

type Home struct {
	AddedReleases   []model.Release
	NewReleases     []model.Release
	AddedMovies     []model.Movie
	NewMovies       []model.Movie
	RecommendMovies []model.Recommend
	NewEpisodes     []model.Episode
	NewSeries       []model.Series
	AddedTVEpisodes []model.TVEpisode
}

type Artists struct {
	Artists []model.Artist
}

type Artist struct {
	Artist     model.Artist
	Image      string
	Background string
	Releases   []model.Release
	Similar    []model.Artist
	IsFavorite bool
	Deep       TrackList `json:"-"`
	Popular    TrackList `json:"-"`
	Radio      TrackList `json:"-"`
	Shuffle    TrackList `json:"-"`
	Singles    TrackList `json:"-"`
	Tracks     TrackList `json:"-"`
}

type Popular struct {
	Artist  model.Artist
	Popular []model.Track
}

type Singles struct {
	Artist  model.Artist
	Singles []model.Track
}

type WantList struct {
	Artist   model.Artist
	Releases []model.Release
}

type Release struct {
	Artist     model.Artist
	Release    model.Release
	Image      string
	Tracks     []model.Track
	Singles    []model.Track
	Popular    []model.Track
	Similar    []model.Release
	Background string
}

type Search struct {
	Artists    []model.Artist
	Releases   []model.Release
	Tracks     []model.Track
	Stations   []model.Station
	Movies     []model.Movie
	Series     []model.Series
	Episodes   []model.Episode
	TVEpisodes []model.TVEpisode
	Query      string
	Hits       int
}

type Radio struct {
	Live   []model.Station // live stations on the air, in config order
	Stream []model.Station // internet radio, in config order
}

type Movies struct {
	Movies []model.Movie
}

// RadioSchedule is what's on each live station between Start and End.
type RadioSchedule struct {
	Time     int64 // server time in milliseconds; use it, not the device clock
	Start    time.Time
	End      time.Time
	Stations []RadioStation
}

// RadioStation is a live station's tracks in the requested range. ID is the
// station's ID in /api/radio.
type RadioStation struct {
	ID     uint
	Name   string
	Image  string `json:",omitempty"`
	Tracks []RadioTrack
	Shows  []RadioShow `json:",omitempty"` // the shows named by Tracks
}

// RadioShow is a regular show on a live station.
type RadioShow struct {
	Name        string
	Image       string `json:",omitempty"`
	Description string `json:",omitempty"`
}

// RadioTrack is a scheduled track. ID is the track row.
type RadioTrack struct {
	Start    time.Time
	End      time.Time
	Title    string
	Artist   string
	Release  string `json:",omitempty"`
	Image    string `json:",omitempty"`
	Location string
	ETag     string `json:",omitempty"`
	Size     int64  `json:",omitempty"`
	ID       uint
	Show     string `json:",omitempty"` // the show it's part of, if any
}

type Movie struct {
	Movie      model.Movie
	Location   string
	Collection model.Collection
	Other      []model.Movie
	Cast       []model.Cast
	Crew       []model.Crew
	Starring   []model.Person
	Directing  []model.Person
	Writing    []model.Person
	Genres     []string
	Keywords   []string
	Vote       int
	VoteCount  int
	Trailers   []model.Trailer
	IsFavorite bool
}

type Profile struct {
	Person model.Person
	Movies MovieCredits
	Shows  TVCredits
}

type MovieCredits struct {
	Starring  []model.Movie
	Directing []model.Movie
	Writing   []model.Movie
}

type TVCredits struct {
	Starring  []model.TVSeries
	Directing []model.TVSeries
	Writing   []model.TVSeries
}

type Genre struct {
	Name   string
	Movies []model.Movie
}

type Keyword struct {
	Name   string
	Movies []model.Movie
}

type TVList struct {
	Series   []model.TVSeries
	Episodes []model.TVEpisode
}

type TVShows struct {
	Series []model.TVSeries
}

type TVSeries struct {
	Series     model.TVSeries
	Episodes   []model.TVEpisode
	Genres     []string
	Keywords   []string
	Cast       []model.TVSeriesCast
	Crew       []model.TVSeriesCrew
	Directing  []model.Person
	Starring   []model.Person
	Writing    []model.Person
	Vote       int
	VoteCount  int
	IsFavorite bool
}

type TVEpisode struct {
	Series    model.TVSeries
	Episode   model.TVEpisode
	Location  string
	Cast      []model.TVEpisodeCast
	Crew      []model.TVEpisodeCrew
	Directing []model.Person
	Starring  []model.Person
	Writing   []model.Person
	Vote      int
	VoteCount int
}

type Podcasts struct {
	Series []model.Series
}

type Series struct {
	Series   model.Series
	Episodes []model.Episode
}

type Episode struct {
	Episode model.Episode
}

type Progress struct {
	Offsets []model.Offset
}

type Offset struct {
	Offset model.Offset
}

type TrackStats struct {
	Interval     string
	Artists      []model.ActivityArtist
	Releases     []model.ActivityRelease
	Tracks       []model.ActivityTrack
	ArtistCount  int
	ReleaseCount int
	TrackCount   int
	ListenCount  int
}

type TrackHistory struct {
	Tracks []model.ActivityTrack
}

type TrackCounts struct {
	Counts []model.ActivityCount
}

func (c *TrackCounts) Total() int {
	total := 0
	for _, v := range c.Counts {
		total += v.Count
	}
	return total
}

func (c *TrackCounts) Values() []int {
	result := make([]int, len(c.Counts))
	for i, v := range c.Counts {
		result[i] = v.Count
	}
	return result
}

type TrackCharts struct {
	Labels []string
	Charts []TrackChart
}

func (c *TrackCharts) AddCounts(label string, counts *TrackCounts) {
	c.Charts = append(c.Charts, TrackChart{
		Label:       label,
		ListenCount: counts.Total(),
		Counts:      counts.Values(),
	})
}

type TrackChart struct {
	Start       time.Time
	ListenCount int
	Label       string
	Counts      []int
}

type Playlist struct {
	ID         int
	Name       string
	TrackCount int
}

type Playlists struct {
	Playlists []Playlist
}

func NewPlaylist(p model.Playlist) *Playlist {
	return &Playlist{ID: int(p.ID), Name: p.Name, TrackCount: p.TrackCount}
}

type Favorite struct {
	Artists []model.Artist
	Movies  []model.Movie
	Shows   []model.TVSeries
	Tracks  []model.Track
}

type FavoriteArtists struct {
	Artists []model.Artist
}

type FavoriteTracks struct {
	Tracks []model.Track
}
