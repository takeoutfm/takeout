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
	"github.com/takeoutfm/takeout/internal/people"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/model"

	. "github.com/takeoutfm/takeout/view"
)

func IndexView(ctx Context) *Index {
	view := &Index{}
	view.Time = time.Now().UnixMilli()
	view.HasMusic = ctx.Music().HasMusic()
	view.HasMovies = ctx.Film().HasMovies()
	view.HasShows = ctx.TV().HasShows()
	view.HasPodcasts = ctx.Podcast().HasPodcasts()
	view.HasPlaylists = ctx.Music().HasPlaylists(ctx.User())
	return view
}

func HomeView(ctx Context) *Home {
	view := &Home{}
	m := ctx.Music()
	f := ctx.Film()
	p := ctx.Podcast()
	tv := ctx.TV()

	view.AddedReleases = m.RecentlyAdded()
	view.NewReleases = m.RecentlyReleased()
	view.AddedMovies = f.RecentlyAdded()
	view.NewMovies = f.RecentlyReleased()
	view.RecommendMovies = f.Recommend()
	view.NewEpisodes = p.RecentEpisodes()
	view.AddedTVEpisodes = tv.AddedTVEpisodes()

	return view
}

func ArtistsView(ctx Context) *Artists {
	view := &Artists{}
	view.Artists = ctx.Music().Artists()
	return view
}

func ArtistView(ctx Context, artist model.Artist) *Artist {
	m := ctx.Music()
	view := &Artist{}
	view.Artist = artist
	view.Releases = m.ArtistReleases(artist)
	view.Similar = m.SimilarArtists(artist)
	view.Image = m.ArtistImage(artist)
	view.Background = m.ArtistBackground(artist)
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
	return view
}

func WantListView(ctx Context, artist model.Artist) *WantList {
	m := ctx.Music()
	view := &WantList{}
	view.Artist = artist
	view.Releases = m.WantArtistReleases(artist)
	return view
}

func ReleaseView(ctx Context, release model.Release) *Release {
	m := ctx.Music()
	view := &Release{}
	view.Release = release
	artist, err := m.Artist(release.Artist)
	if err == nil {
		view.Artist = artist
	}
	view.Tracks = ctx.FindReleaseTracks(release)
	view.Singles = m.ReleaseSingles(release)
	view.Popular = m.ReleasePopular(release)
	view.Similar = m.SimilarReleases(view.Artist, release)
	view.Image = music.CoverSmall(release)
	return view
}

func SearchView(ctx Context, query string) *Search {
	m := ctx.Music()
	f := ctx.Film()
	p := ctx.Podcast()
	tv := ctx.TV()
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
	view.Movies = f.Search(query)
	view.Series, view.Episodes = p.Search(query)
	view.TVEpisodes = tv.Search(query)
	view.Hits = len(view.Artists) +
		len(view.Releases) +
		len(view.Stations) +
		len(view.Tracks) +
		len(view.Movies) +
		len(view.Series) +
		len(view.Episodes) +
		len(view.TVEpisodes)
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
	f := ctx.Film()
	view := &Movies{}
	view.Movies = f.Movies()
	return view
}

func MovieView(ctx Context, m model.Movie) *Movie {
	f := ctx.Film()
	view := &Movie{}
	view.Movie = m
	view.Location = ctx.LocateMovie(m)
	collections := f.MovieCollections(m)
	if len(collections) > 0 {
		view.Collection = collections[0]
		view.Other = f.CollectionMovies(collections[0])
		if len(view.Other) == 1 && view.Other[0].ID == m.ID {
			// collection is just this movie so remove
			view.Other = view.Other[1:]
		}
	}
	view.Cast = f.Cast(m)
	view.Crew = f.Crew(m)

	billing := people.NewBilling(view.Cast, view.Crew)
	view.Directing = billing.Directors
	view.Starring = billing.Actors

	view.Genres = f.Genres(m)
	view.Keywords = f.Keywords(m)
	view.Vote = int(m.VoteAverage * 10)
	view.VoteCount = m.VoteCount
	view.Trailers = f.MovieTrailers(m)
	return view
}

func ProfileView(ctx Context, p model.Person) *Profile {
	f := ctx.Film()
	tv := ctx.TV()
	view := &Profile{}
	view.Person = p
	view.Movies.Directing = f.Directing(p)
	view.Movies.Starring = f.Starring(p)
	view.Movies.Writing = f.Writing(p)
	view.Shows.Directing = tv.SeriesDirecting(p)
	view.Shows.Starring = tv.SeriesStarring(p)
	view.Shows.Writing = tv.SeriesWriting(p)
	fmt.Printf("%+v\n", view)
	return view
}

func GenreView(ctx Context, name string) *Genre {
	f := ctx.Film()
	view := &Genre{}
	view.Name = name
	view.Movies = f.Genre(name)
	return view
}

func KeywordView(ctx Context, name string) *Keyword {
	f := ctx.Film()
	view := &Keyword{}
	view.Name = name
	view.Movies = f.Keyword(name)
	return view
}

func WatchView(ctx Context, m model.Movie) *Watch {
	view := &Watch{}
	view.Movie = m
	view.Location = ctx.LocateMovie(m)
	return view
}

func TVShowsView(ctx Context) *TVShows {
	tv := ctx.TV()
	view := &TVShows{}
	view.Series = tv.Series()
	return view
}

func TVSeriesView(ctx Context, s model.TVSeries) *TVSeries {
	tv := ctx.TV()
	view := &TVSeries{}
	view.Series = s
	view.Episodes = tv.Episodes(s)
	view.Cast = tv.SeriesCast(s)
	view.Crew = tv.SeriesCrew(s)

	billing := people.NewBilling(view.Cast, view.Crew)
	view.Directing = billing.Directors
	view.Starring = billing.Actors
	view.Writing = billing.Writers

	view.Keywords = tv.Keywords(s)
	view.Genres = tv.Genres(s)
	view.Vote = int(s.VoteAverage * 10)
	view.VoteCount = s.VoteCount
	return view
}

func TVEpisodeView(ctx Context, e model.TVEpisode) *TVEpisode {
	tv := ctx.TV()
	view := &TVEpisode{}
	view.Episode = e
	view.Location = ctx.LocateTVEpisode(e)
	series, _ := tv.LookupTVID(int(e.TVID))
	view.Series = series
	view.Cast = tv.EpisodeCast(e)
	view.Crew = tv.EpisodeCrew(e)

	billing := people.NewBilling(view.Cast, view.Crew)
	view.Directing = billing.Directors
	view.Starring = billing.Actors
	view.Writing = billing.Writers

	view.Vote = int(e.VoteAverage * 10)
	view.VoteCount = e.VoteCount
	return view
}

func PodcastsView(ctx Context) *Podcasts {
	p := ctx.Podcast()
	view := &Podcasts{}
	view.Series = p.Series()
	return view
}

func PodcastsSubscribedView(ctx Context) *Podcasts {
	p := ctx.Podcast()
	view := &Podcasts{}
	view.Series = p.SeriesFor(ctx.User().Name)
	return view
}

func SeriesView(ctx Context, s model.Series) *Series {
	view := &Series{}
	view.Series = s
	view.Episodes = ctx.FindSeriesEpisodes(s)
	limit := ctx.Config().Podcast.EpisodeLimit
	if len(view.Episodes) > limit {
		view.Episodes = view.Episodes[:limit]
	}
	return view
}

func EpisodeView(ctx Context, e model.Episode) *Episode {
	view := &Episode{}
	view.Episode = e
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

func TrackStatsView(ctx Context, interval string, d date.DateRange) *TrackStats {
	view := &TrackStats{}
	tracks := ctx.Activity().TopTracks(ctx, d.Start, d.End)
	view.Interval = interval
	view.Tracks = tracks
	view.Artists = ctx.Activity().TopArtists(ctx, tracks)
	view.Releases = ctx.Activity().TopReleases(ctx, tracks)
	view.ArtistCount = len(view.Artists)
	view.ReleaseCount = len(view.Releases)
	view.TrackCount = len(tracks)
	for _, t := range tracks {
		view.ListenCount += t.Count
	}
	return view
}

func TrackHistoryView(ctx Context, d date.DateRange) *TrackHistory {
	view := &TrackHistory{}
	view.Tracks = ctx.Activity().Tracks(ctx, d.Start, d.End)
	return view
}

func TrackDayCountsView(ctx Context, d date.DateRange) *TrackCounts {
	return ctx.Activity().TrackDayCounts(ctx, d)
}

func TrackMonthCountsView(ctx Context, d date.DateRange) *TrackCounts {
	return ctx.Activity().TrackMonthCounts(ctx, d)
}

// func ActivityMoviesView(ctx Context, start, end time.Time) *ActivityMovies {
// 	view := &ActivityMovies{}
// 	view.Movies = ctx.Activity().Movies(ctx, start, end)
// 	return view
// }

// func ActivityReleasesView(ctx Context, start, end time.Time) *ActivityReleases {
// 	view := &ActivityReleases{}
// 	view.Releases = ctx.Activity().Releases(ctx, start, end)
// 	return view
// }

func PlaylistView(ctx Context, playlist model.Playlist) *Playlist {
	return NewPlaylist(playlist)
}

func PlaylistsView(ctx Context, playlists []model.Playlist) *Playlists {
	view := &Playlists{}
	list := make([]Playlist, len(playlists))
	for i := range playlists {
		list[i] = *NewPlaylist(playlists[i])
	}
	view.Playlists = list
	return view
}
