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

package server

import (
	"fmt"
	"time"

	"github.com/takeoutfm/takeout/internal/music"
	"github.com/takeoutfm/takeout/internal/video"
	"github.com/takeoutfm/takeout/model"

	. "github.com/takeoutfm/takeout/view"
)

func IndexView(ctx Context) *Index {
	view := &Index{}
	view.Time = time.Now().UnixMilli()
	view.HasMusic = ctx.Music().HasMusic()
	view.HasMovies = ctx.Video().HasMovies()
	view.HasPodcasts = ctx.Podcast().HasPodcasts()
	return view
}

func HomeView(ctx Context) *Home {
	view := &Home{}
	m := ctx.Music()
	v := ctx.Video()
	p := ctx.Podcast()

	view.AddedReleases = m.RecentlyAdded()
	view.NewReleases = m.RecentlyReleased()
	view.AddedMovies = v.RecentlyAdded()
	view.NewMovies = v.RecentlyReleased()
	view.RecommendMovies = v.Recommend()
	view.NewEpisodes = p.RecentEpisodes()
	view.NewSeries = p.RecentSeries()

	view.CoverSmall = m.CoverSmall
	view.PosterSmall = v.MoviePosterSmall
	view.EpisodeImage = p.EpisodeImage
	view.SeriesImage = p.SeriesImage
	return view
}

func ArtistsView(ctx Context) *Artists {
	view := &Artists{}
	view.Artists = ctx.Music().Artists()
	view.CoverSmall = ctx.Music().CoverSmall
	return view
}

func ArtistView(ctx Context, artist model.Artist) *Artist {
	m := ctx.Music()
	view := &Artist{}
	view.Artist = artist
	view.Releases = m.ArtistReleases(&artist)
	view.Similar = m.SimilarArtists(&artist)
	view.Image = m.ArtistImage(&artist)
	view.Background = m.ArtistBackground(&artist)
	view.CoverSmall = m.CoverSmall
	view.Popular = TrackList{
		Title: fmt.Sprintf("%s \u2013 Popular", artist.Name),
		Tracks: func() []model.Track {
			tracks := m.ArtistPopularTracks(artist, ctx.Config().Music.PopularLimit)
			return tracks
		},
	}
	view.Singles = TrackList{
		Title: fmt.Sprintf("%s \u2013 Singles", artist.Name),
		Tracks: func() []model.Track {
			tracks := m.ArtistSingleTracks(artist, ctx.Config().Music.SinglesLimit)
			return tracks
		},
	}
	view.Deep = TrackList{
		Title: fmt.Sprintf("%s \u2013 Deep Tracks", artist.Name),
		Tracks: func() []model.Track {
			return m.ArtistDeep(artist, ctx.Config().Music.RadioLimit)
		},
	}
	view.Radio = TrackList{
		Title: fmt.Sprintf("%s \u2013 Radio", artist.Name),
		Tracks: func() []model.Track {
			return m.ArtistRadio(artist)
		},
	}
	view.Shuffle = TrackList{
		Title: fmt.Sprintf("%s \u2013 Shuffle", artist.Name),
		Tracks: func() []model.Track {
			return m.ArtistShuffle(artist, ctx.Config().Music.RadioLimit)
		},
	}
	view.Tracks = TrackList{
		Title: fmt.Sprintf("%s \u2013 Tracks", artist.Name),
		Tracks: func() []model.Track {
			return m.ArtistTracks(artist)
		},
	}
	return view
}

func PopularView(ctx Context, artist model.Artist) *Popular {
	m := ctx.Music()
	view := &Popular{}
	view.Artist = artist
	view.Popular = m.ArtistPopularTracks(artist)
	limit := ctx.Config().Music.PopularLimit
	if len(view.Popular) > limit {
		view.Popular = view.Popular[:limit]
	}
	view.CoverSmall = m.CoverSmall
	return view
}

func SinglesView(ctx Context, artist model.Artist) *Singles {
	m := ctx.Music()
	view := &Singles{}
	view.Artist = artist
	view.Singles = m.ArtistSingleTracks(artist)
	limit := ctx.Config().Music.SinglesLimit
	if len(view.Singles) > limit {
		view.Singles = view.Singles[:limit]
	}
	view.CoverSmall = m.CoverSmall
	return view
}

func WantListView(ctx Context, artist model.Artist) *WantList {
	m := ctx.Music()
	view := &WantList{}
	view.Artist = artist
	view.Releases = m.WantArtistReleases(artist)
	view.CoverSmall = m.CoverSmall
	return view
}

func ReleaseView(ctx Context, release model.Release) *Release {
	m := ctx.Music()
	view := &Release{}
	view.Release = release
	artist := m.Artist(release.Artist)
	if artist != nil {
		view.Artist = *artist
	}
	view.Tracks = ctx.FindReleaseTracks(release)
	view.Singles = m.ReleaseSingles(release)
	view.Popular = m.ReleasePopular(release)
	view.Similar = m.SimilarReleases(&view.Artist, release)
	view.Image = m.CoverSmall(release)
	view.CoverSmall = m.CoverSmall
	return view
}

func SearchView(ctx Context, query string) *Search {
	m := ctx.Music()
	v := ctx.Video()
	p := ctx.Podcast()
	view := &Search{}
	artists, releases, _, stations := m.Query(query)
	view.Artists = artists
	view.Releases = releases
	for _, s := range stations {
		if s.Visible(ctx.User().Name) {
			view.Stations = append(view.Stations, s)
		}
	}
	view.Query = query
	view.Tracks = m.Search(query)
	view.Movies = v.Search(query)
	view.Series, view.Episodes = p.Search(query)
	view.Hits = len(view.Artists) +
		len(view.Releases) +
		len(view.Stations) +
		len(view.Tracks) +
		len(view.Movies) +
		len(view.Series) +
		len(view.Episodes)
	view.CoverSmall = m.CoverSmall
	view.PosterSmall = v.MoviePosterSmall
	return view
}

func RadioView(ctx Context) *Radio {
	m := ctx.Music()
	view := &Radio{}
	for _, s := range m.Stations(ctx.User()) {
		switch s.Type {
		case music.TypeArtist:
			view.Artist = append(view.Artist, s)
		case music.TypeGenre:
			view.Genre = append(view.Genre, s)
		case music.TypeSimilar:
			view.Similar = append(view.Similar, s)
		case music.TypePeriod:
			view.Period = append(view.Period, s)
		case music.TypeSeries:
			view.Series = append(view.Series, s)
		case music.TypeStream:
			view.Stream = append(view.Stream, s)
		default:
			view.Other = append(view.Other, s)
		}
	}
	return view
}

func MoviesView(ctx Context) *Movies {
	v := ctx.Video()
	view := &Movies{}
	view.Movies = v.Movies()
	view.PosterSmall = v.MoviePosterSmall
	view.Backdrop = v.MovieBackdrop
	return view
}

func MovieView(ctx Context, m model.Movie) *Movie {
	v := ctx.Video()
	view := &Movie{}
	view.Movie = m
	view.Location = ctx.LocateMovie(m)
	collection := v.MovieCollection(m)
	if collection != nil {
		view.Collection = *collection
		view.Other = v.CollectionMovies(collection)
		if len(view.Other) == 1 && view.Other[0].ID == m.ID {
			// collection is just this movie so remove
			view.Other = view.Other[1:]
		}
	}
	view.Cast = v.Cast(m)
	view.Crew = v.Crew(m)
	for _, c := range view.Crew {
		switch c.Job {
		case video.JobDirector:
			view.Directing = append(view.Directing, c.Person)
		case video.JobNovel, video.JobScreenplay, video.JobStory:
			view.Writing = append(view.Writing, c.Person)
		}
	}
	for i, c := range view.Cast {
		if i == 3 {
			break
		}
		view.Starring = append(view.Starring, c.Person)
	}
	view.Genres = v.Genres(m)
	view.Keywords = v.Keywords(m)
	view.Vote = int(m.VoteAverage * 10)
	view.VoteCount = m.VoteCount
	view.Poster = v.MoviePoster
	view.PosterSmall = v.MoviePosterSmall
	view.Backdrop = v.MovieBackdrop
	view.Profile = v.PersonProfile
	return view
}

func ProfileView(ctx Context, p model.Person) *Profile {
	v := ctx.Video()
	view := &Profile{}
	view.Person = p
	view.Starring = v.Starring(p)
	view.Writing = v.Writing(p)
	view.Directing = v.Directing(p)
	view.PosterSmall = v.MoviePosterSmall
	view.Backdrop = v.MovieBackdrop
	view.Profile = v.PersonProfile
	return view
}

func GenreView(ctx Context, name string) *Genre {
	v := ctx.Video()
	view := &Genre{}
	view.Name = name
	view.Movies = v.Genre(name)
	view.PosterSmall = v.MoviePosterSmall
	view.Backdrop = v.MovieBackdrop
	return view
}

func KeywordView(ctx Context, name string) *Keyword {
	v := ctx.Video()
	view := &Keyword{}
	view.Name = name
	view.Movies = v.Keyword(name)
	view.PosterSmall = v.MoviePosterSmall
	view.Backdrop = v.MovieBackdrop
	return view
}

func WatchView(ctx Context, m model.Movie) *Watch {
	v := ctx.Video()
	view := &Watch{}
	view.Movie = m
	view.Location = ctx.LocateMovie(m)
	view.PosterSmall = v.MoviePosterSmall
	view.Backdrop = v.MovieBackdrop
	return view
}

func PodcastsView(ctx Context) *Podcasts {
	p := ctx.Podcast()
	view := &Podcasts{}
	view.Series = p.Series()
	view.SeriesImage = p.SeriesImage
	return view
}

func PodcastsSubscribedView(ctx Context) *Podcasts {
	p := ctx.Podcast()
	view := &Podcasts{}
	view.Series = p.SeriesFor(ctx.User().Name)
	view.SeriesImage = p.SeriesImage
	return view
}

func SeriesView(ctx Context, s model.Series) *Series {
	p := ctx.Podcast()
	view := &Series{}
	view.Series = s
	view.Episodes = ctx.FindSeriesEpisodes(s)
	limit := ctx.Config().Podcast.EpisodeLimit
	if len(view.Episodes) > limit {
		view.Episodes = view.Episodes[:limit]
	}
	view.SeriesImage = p.SeriesImage
	view.EpisodeImage = p.EpisodeImage
	return view
}

func EpisodeView(ctx Context, e model.Episode) *Episode {
	view := &Episode{}
	view.Episode = e
	view.EpisodeImage = ctx.Podcast().EpisodeImage
	return view
}

func ProgressView(ctx Context) *Progress {
	view := &Progress{}
	view.Offsets = ctx.Progress().Offsets(ctx.User())
	return view
}

func OffsetView(ctx Context, offset model.Offset) *Offset {
	view := &Offset{}
	view.Offset = offset
	return view
}

func ActivityView(ctx Context) *Activity {
	view := &Activity{}
	view.RecentTracks = ctx.Activity().RecentTracks(ctx)
	view.RecentMovies = ctx.Activity().RecentMovies(ctx)
	view.RecentReleases = ctx.Activity().RecentReleases(ctx)
	return view
}

func ActivityTracksView(ctx Context, start, end time.Time) *ActivityTracks {
	view := &ActivityTracks{}
	view.Tracks = ctx.Activity().Tracks(ctx, start, end)
	return view
}

func ActivityPopularTracksView(ctx Context, start, end time.Time) *ActivityTracks {
	view := &ActivityTracks{}
	view.Tracks = ctx.Activity().PopularTracks(ctx, start, end)
	return view
}

func ActivityMoviesView(ctx Context, start, end time.Time) *ActivityMovies {
	view := &ActivityMovies{}
	view.Movies = ctx.Activity().Movies(ctx, start, end)
	return view
}

func ActivityReleasesView(ctx Context, start, end time.Time) *ActivityReleases {
	view := &ActivityReleases{}
	view.Releases = ctx.Activity().Releases(ctx, start, end)
	return view
}

func PlaylistView(ctx Context, playlist model.Playlist) *Playlist {
	return NewPlaylist(playlist)
}

func PlaylistsView(ctx Context, playlists []*model.Playlist) *Playlists {
	view := &Playlists{}
	list := make([]Playlist, len(playlists))
	for i := range playlists {
		list[i] = *NewPlaylist(*playlists[i])
	}
	view.Playlists = list
	return view
}
