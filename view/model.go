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
	"time"

	"github.com/takeoutfm/takeout/model"
)

type TracksFunc func() []model.Track

type TrackList struct {
	Title  string
	Tracks TracksFunc
}

type Index struct {
	Time         int64
	HasMusic     bool
	HasMovies    bool
	HasShows     bool
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
	Artist  model.Artist
	Release model.Release
	Image   string
	Tracks  []model.Track
	Singles []model.Track
	Popular []model.Track
	Similar []model.Release
}

type Search struct {
	Artists  []model.Artist
	Releases []model.Release
	Tracks   []model.Track
	Stations []model.Station
	Movies   []model.Movie
	Series   []model.Series
	Episodes []model.Episode
	Query    string
	Hits     int
}

type Radio struct {
	Artist  []model.Station
	Genre   []model.Station
	Similar []model.Station
	Period  []model.Station
	Series  []model.Station
	Other   []model.Station
	Stream  []model.Station
}

type Movies struct {
	Movies []model.Movie
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

type Watch struct {
	Movie    model.Movie
	Location string
}

type TVShows struct {
	Series []model.TVSeries
}

type TVSeries struct {
	Series    model.TVSeries
	Episodes  []model.TVEpisode
	Genres    []string
	Keywords  []string
	Cast      []model.TVSeriesCast
	Crew      []model.TVSeriesCrew
	Directing []model.Person
	Starring  []model.Person
	Writing   []model.Person
	Vote      int
	VoteCount int
}

type TVEpisode struct {
	Series    model.TVSeries
	Episode   model.TVEpisode
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
