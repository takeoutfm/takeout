// Copyright 2024 defsub
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
	"testing"
	"time"

	"github.com/takeoutfm/takeout/model"
)

func TestIndexView(t *testing.T) {
	ctx := NewTestContext(t)
	view := IndexView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestHomeView(t *testing.T) {
	ctx := NewTestContext(t)
	view := HomeView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestArtistsView(t *testing.T) {
	ctx := NewTestContext(t)
	view := ArtistsView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestArtistView(t *testing.T) {
	ctx := NewTestContext(t)
	view := ArtistView(ctx, model.Artist{Name: "test artist"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestPopularView(t *testing.T) {
	ctx := NewTestContext(t)
	view := PopularView(ctx, model.Artist{Name: "test artist"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestSinglesView(t *testing.T) {
	ctx := NewTestContext(t)
	view := SinglesView(ctx, model.Artist{Name: "test artist"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestWantListView(t *testing.T) {
	ctx := NewTestContext(t)
	view := WantListView(ctx, model.Artist{Name: "test artist"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestReleaseView(t *testing.T) {
	ctx := NewTestContext(t)
	view := ReleaseView(ctx, model.Release{Name: "test name", Artist: "test artist"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestSearchView(t *testing.T) {
	ctx := NewTestContext(t)
	query := "test query"
	view := SearchView(ctx, query)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestRadioView(t *testing.T) {
	ctx := NewTestContext(t)
	view := RadioView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestMoviesView(t *testing.T) {
	ctx := NewTestContext(t)
	view := MoviesView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestMovieView(t *testing.T) {
	ctx := NewTestContext(t)
	view := MovieView(ctx, model.Movie{Title: "test title"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestWatchView(t *testing.T) {
	ctx := NewTestContext(t)
	view := WatchView(ctx, model.Movie{Title: "test title"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestProfileView(t *testing.T) {
	ctx := NewTestContext(t)
	view := ProfileView(ctx, model.Person{Name: "test name"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestGenreView(t *testing.T) {
	ctx := NewTestContext(t)
	view := GenreView(ctx, "test genre")
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestKeywordView(t *testing.T) {
	ctx := NewTestContext(t)
	view := KeywordView(ctx, "test keyword")
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestPodcastsView(t *testing.T) {
	ctx := NewTestContext(t)
	view := PodcastsView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestPodcastsSubscribedView(t *testing.T) {
	ctx := NewTestContext(t)
	view := PodcastsSubscribedView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestSeriesView(t *testing.T) {
	ctx := NewTestContext(t)
	view := SeriesView(ctx, model.Series{Title: "test title"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestEpisodeView(t *testing.T) {
	ctx := NewTestContext(t)
	view := EpisodeView(ctx, model.Episode{Title: "test title"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestProgressView(t *testing.T) {
	ctx := NewTestContext(t)
	view := ProgressView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestOffsetView(t *testing.T) {
	ctx := NewTestContext(t)
	view := OffsetView(ctx, model.Offset{User: "takeout", ETag: "1234"})
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestActivityView(t *testing.T) {
	ctx := NewTestContext(t)
	view := ActivityView(ctx)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestActivityTracksView(t *testing.T) {
	ctx := NewTestContext(t)
	start := time.Now().Add(-time.Hour * 24)
	end := time.Now()
	view := ActivityTracksView(ctx, start, end)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestActivityPopularTracksView(t *testing.T) {
	ctx := NewTestContext(t)
	start := time.Now().Add(-time.Hour * 24)
	end := time.Now()
	view := ActivityPopularTracksView(ctx, start, end)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestActivityMoviesView(t *testing.T) {
	ctx := NewTestContext(t)
	start := time.Now().Add(-time.Hour * 24)
	end := time.Now()
	view := ActivityMoviesView(ctx, start, end)
	if view == nil {
		t.Fatal("expect view")
	}
}

func TestActivityReleasesView(t *testing.T) {
	ctx := NewTestContext(t)
	start := time.Now().Add(-time.Hour * 24)
	end := time.Now()
	view := ActivityReleasesView(ctx, start, end)
	if view == nil {
		t.Fatal("expect view")
	}
}
