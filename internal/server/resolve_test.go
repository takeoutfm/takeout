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

	"github.com/takeoutfm/takeout/model"
	"github.com/takeoutfm/takeout/spiff"
)

func TestTrackEntry(t *testing.T) {
	ctx := NewTestContext(t)
	track := model.Track{
		Artist:       "test artist",
		ReleaseTitle: "test release",
		Title:        "test title",
	}
	entry := trackEntry(ctx, track)
	if entry.Creator != track.Artist {
		t.Error("expect artist")
	}
	if entry.Title != track.Title {
		t.Error("expect title")
	}
}

func TestMovieEntry(t *testing.T) {
	ctx := NewTestContext(t)
	movie := model.Movie{
		Title: "test title",
	}
	entry := movieEntry(ctx, movie)
	if entry.Title != movie.Title {
		t.Error("expect title")
	}
}

func TestEpisodeEntry(t *testing.T) {
	ctx := NewTestContext(t)
	series := model.Series{
		Title: "test series title",
	}
	episode := model.Episode{
		Title: "test episode title",
	}
	entry := episodeEntry(ctx, series, episode)
	if entry.Title != episode.Title {
		t.Error("expect title")
	}
}

func TestAddTrackEntries(t *testing.T) {
	ctx := NewTestContext(t)
	track := model.Track{
		Artist:       "test artist",
		ReleaseTitle: "test release",
		Title:        "test title",
	}

	var entries []spiff.Entry
	entries = addTrackEntries(ctx, []model.Track{track}, entries)

	if len(entries) != 1 {
		t.Error("expect entries")
	}
}

func TestAddMovieEntries(t *testing.T) {
	ctx := NewTestContext(t)
	movie := model.Movie{
		Title: "test title",
	}

	var entries []spiff.Entry
	entries = addMovieEntries(ctx, []model.Movie{movie}, entries)

	if len(entries) != 1 {
		t.Error("expect entries")
	}
}

func TestAddEpisodeEntries(t *testing.T) {
	ctx := NewTestContext(t)
	series := model.Series{
		Title: "test series title",
	}
	episode := model.Episode{
		Title: "test episode title",
	}

	var entries []spiff.Entry
	entries = addEpisodeEntries(ctx, series, []model.Episode{episode}, entries)

	if len(entries) != 1 {
		t.Error("expect entries")
	}
}

func TestAddStationEntries(t *testing.T) {
	ctx := NewTestContext(t)
	station := model.Station{
		Name: "test station",
	}

	var entries []spiff.Entry
	entries = addStationEntries(ctx, station, entries)

	if len(entries) != 1 {
		t.Error("expect entries")
	}
}

func TestResolveArtistRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveArtistRef(ctx, TestArtistID, "popular", entries)
	if err != nil {
		t.Error("expect artist resolved")
	}

	entries, err = resolveArtistRef(ctx, "foo", "popular", entries)
	if err == nil {
		t.Error("expect resolve error")
	}
}

func TestResolveReleaseRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveReleaseRef(ctx, TestReleaseID, entries)
	if err != nil {
		t.Error("expect release resolved")
	}

	entries, err = resolveReleaseRef(ctx, "foo", entries)
	if err == nil {
		t.Error("expect release error")
	}
}

func TestResolveTrackRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveTrackRef(ctx, TestTrackID, entries)
	if err != nil {
		t.Error("expect track resolved")
	}

	entries, err = resolveTrackRef(ctx, "foo", entries)
	if err == nil {
		t.Error("expect track error")
	}
}

func TestResolveMovieRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveMovieRef(ctx, TestMovieID, entries)
	if err != nil {
		t.Error("expect movie resolved")
	}

	entries, err = resolveMovieRef(ctx, "foo", entries)
	if err == nil {
		t.Error("expect movie error")
	}
}

func TestResolveSeriesRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveSeriesRef(ctx, TestSeriesID, entries)
	if err != nil {
		t.Error("expect series resolved")
	}

	entries, err = resolveSeriesRef(ctx, "foo", entries)
	if err == nil {
		t.Error("expect series error")
	}
}

func TestResolveEpisodeRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveEpisodeRef(ctx, TestEpisodeID, entries)
	if err != nil {
		t.Error("expect episode resolved")
	}

	entries, err = resolveEpisodeRef(ctx, "foo", entries)
	if err == nil {
		t.Error("expect episode error")
	}
}

func TestResolveSearchRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveSearchRef(ctx, "/music/search?q=test+search", entries)
	if err != nil {
		t.Error(err)
	}
}

func TestResolveStationRef(t *testing.T) {
	ctx := NewTestContext(t)

	var entries []spiff.Entry
	entries, err := resolveStationRef(ctx, TestStationID, entries)
	if err != nil {
		t.Error(err)
	}
}

func TestResolvePlaylistWithTrackRef(t *testing.T) {
	ctx := NewTestContext(t)
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{{Ref: "/music/tracks/" + TestTrackID}}
	err := Resolve(ctx, &p)
	if err != nil {
		t.Error(err)
	}
	if len(p.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

func TestResolvePlaylistWithReleaseRef(t *testing.T) {
	ctx := NewTestContext(t)
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{{Ref: "/music/releases/" + TestReleaseID + "/tracks"}}
	err := Resolve(ctx, &p)
	if err != nil {
		t.Error(err)
	}
	if len(p.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

func TestResolvePlaylistWithMovieRef(t *testing.T) {
	ctx := NewTestContext(t)
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{{Ref: "/movies/" + TestMovieID}}
	err := Resolve(ctx, &p)
	if err != nil {
		t.Error(err)
	}
	if len(p.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

func TestResolvePlaylistWithSeriesRef(t *testing.T) {
	ctx := NewTestContext(t)
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{{Ref: "/podcasts/series/" + TestSeriesID}}
	err := Resolve(ctx, &p)
	if err != nil {
		t.Error(err)
	}
	if len(p.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

func TestResolvePlaylistWithEpisodeRef(t *testing.T) {
	ctx := NewTestContext(t)
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{{Ref: "/podcasts/episodes/" + TestEpisodeID}}
	err := Resolve(ctx, &p)
	if err != nil {
		t.Error(err)
	}
	if len(p.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

func TestDedup(t *testing.T) {
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{
		{Identifier: []string{"bbb"}},
		{Identifier: []string{"aaa"}},
		{Identifier: []string{"aaa"}},
		{Identifier: []string{""}},
		{Identifier: []string{"aaa"}},
	}
	dedup(&p)
	if len(p.Spiff.Entries) != 3 {
		t.Error("expect 3 entries")
	}
}

func TestResolvePlaylistWithTrackRadioRef(t *testing.T) {
	ctx := NewTestContext(t)
	var p spiff.Playlist
	p.Spiff.Entries = []spiff.Entry{{Ref: "/music/tracks/" + TestTrackID + "/radio"}}
	err := Resolve(ctx, &p)
	if err != nil {
		t.Error(err)
	}
	if len(p.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

func TestResolveTrackPlaylist(t *testing.T) {
	ctx := NewTestContext(t)
	track, err := ctx.FindTrack(TestTrackID)
	if err != nil {
		t.Error(err)
	}
	plist := ResolveTrackPlaylist(ctx, track, "/api/track/"+TestTrackID+"/playlist")
	if len(plist.Spiff.Entries) == 0 {
		t.Error("expect entries")
	}
}

// TODO incomplete
