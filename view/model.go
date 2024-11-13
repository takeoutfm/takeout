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
package view

import (
	"github.com/takeoutfm/takeout/model"
)

type CoverFunc func(interface{}) string
type PosterFunc func(model.Movie) string
type BackdropFunc func(model.Movie) string
type ProfileFunc func(model.Person) string
type SeriesImageFunc func(model.Series) string
type EpisodeImageFunc func(model.Episode) string
type TracksFunc func() []model.Track

type TrackList struct {
	Title  string
	Tracks TracksFunc
}

type Index struct {
	Time         int64
	HasMusic     bool
	HasMovies    bool
	HasPodcasts  bool
	HasPlaylists bool
}

type Home struct {
	AddedReleases   []model.Release
	NewReleases     []model.Release
	AddedMovies     []model.Movie
	NewMovies       []model.Movie
	RecommendMovies []model.Recommend
	NewEpisodes     []model.Episode
	NewSeries       []model.Series
	CoverSmall      CoverFunc        `json:"-"`
	PosterSmall     PosterFunc       `json:"-"`
	SeriesImage     SeriesImageFunc  `json:"-"`
	EpisodeImage    EpisodeImageFunc `json:"-"`
}

type Artists struct {
	Artists    []model.Artist
	CoverSmall CoverFunc `json:"-"`
}

type Artist struct {
	Artist     model.Artist
	Image      string
	Background string
	Releases   []model.Release
	Similar    []model.Artist
	CoverSmall CoverFunc `json:"-"`
	Deep       TrackList `json:"-"`
	Popular    TrackList `json:"-"`
	Radio      TrackList `json:"-"`
	Shuffle    TrackList `json:"-"`
	Singles    TrackList `json:"-"`
	Tracks     TrackList `json:"-"`
}

type Popular struct {
	Artist     model.Artist
	Popular    []model.Track
	CoverSmall CoverFunc `json:"-"`
}

type Singles struct {
	Artist     model.Artist
	Singles    []model.Track
	CoverSmall CoverFunc `json:"-"`
}

type WantList struct {
	Artist     model.Artist
	Releases   []model.Release
	CoverSmall CoverFunc `json:"-"`
}

type Release struct {
	Artist     model.Artist
	Release    model.Release
	Image      string
	Tracks     []model.Track
	Singles    []model.Track
	Popular    []model.Track
	Similar    []model.Release
	CoverSmall CoverFunc `json:"-"`
}

type Search struct {
	Artists     []model.Artist
	Releases    []model.Release
	Tracks      []model.Track
	Stations    []model.Station
	Movies      []model.Movie
	Series      []model.Series
	Episodes    []model.Episode
	Query       string
	Hits        int
	CoverSmall  CoverFunc  `json:"-"`
	PosterSmall PosterFunc `json:"-"`
}

type Radio struct {
	Artist     []model.Station
	Genre      []model.Station
	Similar    []model.Station
	Period     []model.Station
	Series     []model.Station
	Other      []model.Station
	Stream     []model.Station
	CoverSmall CoverFunc `json:"-"`
}

type Movies struct {
	Movies      []model.Movie
	PosterSmall PosterFunc   `json:"-"`
	Backdrop    BackdropFunc `json:"-"`
}

type Movie struct {
	Movie       model.Movie
	Location    string
	Collection  model.Collection
	Other       []model.Movie
	Cast        []model.Cast
	Crew        []model.Crew
	Starring    []model.Person
	Directing   []model.Person
	Writing     []model.Person
	Genres      []string
	Keywords    []string
	Vote        int
	VoteCount   int
	Poster      PosterFunc   `json:"-"`
	PosterSmall PosterFunc   `json:"-"`
	Backdrop    BackdropFunc `json:"-"`
	Profile     ProfileFunc  `json:"-"`
}

type Profile struct {
	Person      model.Person
	Starring    []model.Movie
	Directing   []model.Movie
	Writing     []model.Movie
	PosterSmall PosterFunc   `json:"-"`
	Backdrop    BackdropFunc `json:"-"`
	Profile     ProfileFunc  `json:"-"`
}

type Genre struct {
	Name        string
	Movies      []model.Movie
	PosterSmall PosterFunc   `json:"-"`
	Backdrop    BackdropFunc `json:"-"`
}

type Keyword struct {
	Name        string
	Movies      []model.Movie
	PosterSmall PosterFunc   `json:"-"`
	Backdrop    BackdropFunc `json:"-"`
}

type Watch struct {
	Movie       model.Movie
	Location    string
	PosterSmall PosterFunc   `json:"-"`
	Backdrop    BackdropFunc `json:"-"`
}

type Podcasts struct {
	Series      []model.Series
	SeriesImage SeriesImageFunc `json:"-"`
}

type Series struct {
	Series       model.Series
	Episodes     []model.Episode
	SeriesImage  SeriesImageFunc  `json:"-"`
	EpisodeImage EpisodeImageFunc `json:"-"`
}

type Episode struct {
	Episode      model.Episode
	EpisodeImage EpisodeImageFunc `json:"-"`
}

type Progress struct {
	Offsets []model.Offset
}

type Offset struct {
	Offset model.Offset
}

type TrackStats struct {
	Artists       []model.ActivityArtist
	Releases      []model.ActivityRelease
	Tracks        []model.ActivityTrack
	TotalTracks   int
	TotalArtists  int
	TotalReleases int
	ArtistCount   int
	ReleaseCount  int
	TrackCount    int
	CoverSmall    CoverFunc `json:"-"`
}

type TrackHistory struct {
	Tracks []model.ActivityTrack
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
