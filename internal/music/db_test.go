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

package music

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/model"
)

func makeMusic(t *testing.T) *Music {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	m := NewMusic(config)
	err = m.Open()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestTrack(t *testing.T) {
	m := makeMusic(t)

	track := model.Track{
		Artist:       "test artist",
		Release:      "test release",
		Date:         "2023",
		TrackNum:     1,
		DiscNum:      1,
		Title:        "test title",
		Key:          "test key",
		Size:         99999999,
		ETag:         "test etag",
		LastModified: time.Now(),
		TrackCount:   10,
		DiscCount:    1,
		REID:         "36bfc8bc-679a-49ef-a075-37023f6cea82",
		RGID:         "3b6d0276-cd42-452f-b6cf-740072d83ffc",
		RID:          "475e80fa-ef76-4706-a384-5764f4a860e1",
		MediaTitle:   "test title",
		ReleaseDate:  time.Now().Add(-9999 * time.Hour),
		Artwork:      true,
		FrontArtwork: true,
		BackArtwork:  false,
		OtherArtwork: "",
		GroupArtwork: true,
	}

	err := m.createTrack(&track)
	if err != nil {
		t.Fatal(err)
	}
	if track.ID == 0 {
		t.Error("expect track ID")
	}
	if track.UUID == "" {
		t.Error("expect uuid")
	}

	_, err = m.LookupTrack(int(track.ID))
	if err != nil {
		t.Error("expect to find track by id")
	}

	_, err = m.LookupUUID(track.UUID)
	if err != nil {
		t.Error("expect to find track by uuid")
	}
}

func TestArtist(t *testing.T) {
	arid := "a0962d3b-eaaa-4663-96ed-5951836828eb"

	m := makeMusic(t)

	a := model.Artist{
		Name:           "the test artist",
		SortName:       "test artist, the",
		ARID:           arid,
		Disambiguation: "none",
		Country:        "test country",
		Area:           "test area",
		Date:           time.Now(),
		// no EndDate
		Genre: "test genre",
	}

	err := m.createArtist(&a)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Error("expect ID")
	}

	_, err = m.LookupArtist(int(a.ID))
	if err != nil {
		t.Error("expect to find arist by id")
	}

	_, err = m.LookupARID(arid)
	if err != nil {
		t.Error("expect to find arist by arid")
	}
}

func TestRelease(t *testing.T) {
	m := makeMusic(t)

	reid := "91ee703e-0ab1-40e9-bd12-c999399387d2"
	r := model.Release{
		Artist:         "test artist",
		Name:           "test name",
		RGID:           "2d02e8a2-0c97-44b8-9b6f-9f81fd527198",
		REID:           reid,
		Disambiguation: "none",
		Asin:           "test asin",
		Country:        "test country",
		Type:           "Album", // Single, EP
		SecondaryType:  "",      // Live, Compilation
		Date:           time.Now(),
		ReleaseDate:    time.Now(),
		Status:         "Official",
		TrackCount:     13,
		DiscCount:      1,
		Artwork:        true,
		FrontArtwork:   true,
		BackArtwork:    false,
		OtherArtwork:   "",
		GroupArtwork:   true,
	}

	err := m.createRelease(&r)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID == 0 {
		t.Error("expect ID")
	}

	_, err = m.LookupRelease(int(r.ID))
	if err != nil {
		t.Error("expect to find release by id")
	}
	_, err = m.LookupREID(reid)
	if err != nil {
		t.Error("expect to find release by reid")
	}
}

func TestMedia(t *testing.T) {
	m := makeMusic(t)

	reid := "8b229ba4-8927-453a-a5fb-3011b72ec276"
	media := model.Media{
		REID:       reid,
		Name:       "test media name",
		Position:   1,
		Format:     "CD",
		TrackCount: 13,
	}

	err := m.createMedia(&media)
	if err != nil {
		t.Fatal(err)
	}
	if media.ID == 0 {
		t.Error("expect ID")
	}

	r := model.Release{
		REID: reid,
	}

	list := m.releaseMedia(r)
	if len(list) == 0 {
		t.Error("expect media")
	}

	m.deleteReleaseMedia(reid)

	list = m.releaseMedia(r)
	if len(list) != 0 {
		t.Error("expect no media")
	}
}

func TestPopular(t *testing.T) {
	m := makeMusic(t)

	p := model.Popular{
		Artist: "test artist",
		Title:  "test title",
		Rank:   1,
	}

	err := m.createPopular(&p)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Error("expect ID")
	}

	err = m.deletePopularFor("test artist")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSimilar(t *testing.T) {
	m := makeMusic(t)

	s := model.Similar{
		Artist: "test artist",
		Rank:   1,
	}

	err := m.createSimilar(&s)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == 0 {
		t.Error("expect ID")
	}

	err = m.deleteSimilarFor("test artist")
	if err != nil {
		t.Error(err)
	}
}

func TestArtistTag(t *testing.T) {
	m := makeMusic(t)

	tag := model.ArtistTag{
		Artist: "test artist",
		Tag:    "test tag",
		Count:  10,
	}

	err := m.createArtistTag(&tag)
	if err != nil {
		t.Fatal(err)
	}
	if tag.ID == 0 {
		t.Error("expect ID")
	}
}

func TestArtistBackground(t *testing.T) {
	m := makeMusic(t)

	bg := model.ArtistBackground{
		Artist: "test artist",
		URL:    "https://i.co/bg.png",
		Source: "test source",
		Rank:   10,
	}

	err := m.createArtistBackground(&bg)
	if err != nil {
		t.Fatal(err)
	}

	a := model.Artist{
		Name: "test artist",
	}
	list := m.artistBackgrounds(&a)
	if len(list) == 0 {
		t.Error("expect backgrounds")
	}
}

func TestArtistImage(t *testing.T) {
	m := makeMusic(t)

	img := model.ArtistImage{
		Artist: "test artist",
		URL:    "https://i.co/bg.png",
		Source: "test source",
		Rank:   10,
	}

	err := m.createArtistImage(&img)
	if err != nil {
		t.Fatal(err)
	}

	a := model.Artist{
		Name: "test artist",
	}
	list := m.artistImages(&a)
	if len(list) == 0 {
		t.Error("expect images")
	}
}

func TestStation(t *testing.T) {
	m := makeMusic(t)

	user := "takeout"
	s := model.Station{
		User:        user,
		Name:        "test station",
		Creator:     "test creator",
		Ref:         "/music/search?q=artist:test",
		Shared:      true,
		Type:        TypeArtist,
		Image:       "https://image.png",
		Description: "test description",
	}

	err := m.CreateStation(&s)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == 0 {
		t.Error("expect id")
	}
	if s.CreatedAt.IsZero() {
		t.Error("expect CreatedAt")
	}
	if s.UpdatedAt.IsZero() {
		t.Error("expect UpdatedAt")
	}

	s2, err := m.LookupStation(int(s.ID))
	if err != nil {
		t.Fatal(err)
	}
	if s2.ID != s.ID {
		t.Error("expect same id")
	}
	if s2.Name != "test station" {
		t.Error("expect name")
	}

	err = m.DeleteStation(&s)
	if err != nil {
		t.Fatal(err)
	}
	s2, err = m.LookupStation(int(s.ID))
	if err == nil {
		t.Error("expect deleted")
	}
}

func TestPlaylist(t *testing.T) {
	user := "takeout"

	m := makeMusic(t)

	p := model.Playlist{
		User: user,
		Playlist: []byte(`{"playlist":{}}`),
	}

	err := m.CreatePlaylist(&p)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Error("expect id")
	}

	u := auth.User{
		Name: user,
	}

	pp := m.UserPlaylist(&u)
	if pp == nil {
		t.Error("expect playlist")
	}

	pp.Playlist = []byte(`{"playlist":{title:"xyz"}}`)
	err = m.UpdatePlaylist(pp)
	if err != nil {
		t.Error("expect updated")
	}
}

func TestPlaylistID(t *testing.T) {
	user := "takeout"
	m := makeMusic(t)
	u := auth.User{Name: user}

	p := model.Playlist{
		User: user,
		Name: "my playlist",
		Playlist: []byte(`{"playlist":{}}`),
	}

	err := m.CreatePlaylist(&p)
	if err != nil {
		t.Fatal(err)
	}

	playlists := m.UserPlaylists(&u)
	if len(playlists) == 0 {
		t.Error("expect playlists")
	}

	found := false
	id := 0
	for _, p := range playlists {
		if p.Name == "my playlist" {
			found = true
			id = int(p.ID)
			break
		}
	}
	if !found {
		t.Error("playlist not found")
	}

	pp := m.LookupPlaylist(&u, id)
	if pp == nil {
		t.Error("playlist id not found")
	}

	if pp.Name != "my playlist" {
		t.Error("wrong playlist name")
	}
}

func TestPlaylistDelete(t *testing.T) {
	user := "takeout"
	m := makeMusic(t)
	u := auth.User{Name: user}

	id := -1
	playlists := m.UserPlaylists(&u)
	for _, p := range playlists {
		if p.Name == "my playlist" {
			id = int(p.ID)
			break
		}
	}
	if id == -1 {
		t.Error("playlist not found")
	}

	err := m.DeletePlaylist(&u, id)
	if err != nil {
		t.Error(err)
	}

	id = -1
	playlists = m.UserPlaylists(&u)
	for _, p := range playlists {
		if p.Name == "my playlist" {
			id = int(p.ID)
			break
		}
	}
	if id != -1 {
		t.Error("playlist not deleted")
	}
}
