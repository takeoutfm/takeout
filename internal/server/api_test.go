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
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/takeoutfm/takeout/spiff"
	"github.com/takeoutfm/takeout/view"
)

func TestApiPlaylistsCreate(t *testing.T) {
	plist := spiff.Playlist{}
	plist.Type = "music"
	plist.Spiff.Title = "my test playlist"
	plist.Spiff.Entries = []spiff.Entry{{
		Identifier: []string{"abc"},
		Size:       []int64{123},
		Creator:    "test creator",
		Album:      "test album",
		Title:      "test title"},
	}
	data, err := plist.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("POST", "https://takeout/api/playlists", bytes.NewReader(data))
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsCreate(w, r)

	resp := w.Result()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 204 {
		t.Error("expected 204")
	}
}

func TestApiPlaylists(t *testing.T) {
	r := httptest.NewRequest("GET", "https://takeout/api/playlists", nil)
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylists(w, r)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Error("expected 200")
	}

	var view view.Playlists
	json.Unmarshal(body, &view)
	if len(view.Playlists) != 1 {
		t.Error("expected 1 entry")
	}
	if view.Playlists[0].ID != 1 {
		t.Error("expected first id 1")
	}
	if view.Playlists[0].Name != "my test playlist" {
		t.Error("expected playlist name")
	}
}

func TestApiPlaylistsGet(t *testing.T) {
	r := httptest.NewRequest("GET", "https://takeout/api/playlists/1", nil)
	r.SetPathValue("id", "1")
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsGet(w, r)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Error("expected 200")
	}

	var view view.Playlist
	json.Unmarshal(body, &view)
	if view.ID != 1 {
		t.Error("expected view 1")
	}
	if view.Name != "my test playlist" {
		t.Error("expected view name")
	}
}

func TestApiPlaylistsGetPlaylist(t *testing.T) {
	r := httptest.NewRequest("GET", "https://takeout/api/playlists/1/playlist", nil)
	r.SetPathValue("id", "1")
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsGetPlaylist(w, r)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Error("expected 200")
	}

	var plist spiff.Playlist
	json.Unmarshal(body, &plist)
	if len(plist.Spiff.Entries) != 1 {
		t.Errorf("expected 1 entries got %d", len(plist.Spiff.Entries))
	}
}

func TestApiPlaylistsPatch(t *testing.T) {
	patch := `[{"op":"add","path":"/playlist/track/-","value":{"identifier":["cba"],"size":[456],"title":"test title two"}}]`
	data := []byte(patch)

	r := httptest.NewRequest("PATCH", "https://takeout/api/playlists/1/playlist", bytes.NewReader(data))
	r.SetPathValue("id", "1")
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsPatch(w, r)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Error("expected 200")
	}

	var plist spiff.Playlist
	json.Unmarshal(body, &plist)
	if len(plist.Spiff.Entries) != 2 {
		t.Errorf("expected 2 entries got %d", len(plist.Spiff.Entries))
	}

}

func TestApiPlaylistsGetPlaylistPatched(t *testing.T) {
	r := httptest.NewRequest("GET", "https://takeout/api/playlists/1/playlist", nil)
	r.SetPathValue("id", "1")
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsGetPlaylist(w, r)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Error("expected 200")
	}

	var plist spiff.Playlist
	json.Unmarshal(body, &plist)
	if len(plist.Spiff.Entries) != 2 {
		t.Errorf("expected 2 entries got %d", len(plist.Spiff.Entries))
	}
}

func TestApiPlaylistsDelete(t *testing.T) {
	r := httptest.NewRequest("DELETE", "https://takeout/api/playlists/1", nil)
	r.SetPathValue("id", "1")
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsDelete(w, r)

	resp := w.Result()
	if resp.StatusCode != 204 {
		t.Errorf("expected 204 got %d", resp.StatusCode)
	}
}

func TestApiPlaylistsGetPlaylistNotFound(t *testing.T) {
	r := httptest.NewRequest("GET", "https://takeout/api/playlists/1", nil)
	r.SetPathValue("id", "1")
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylistsGetPlaylist(w, r)

	resp := w.Result()
	if resp.StatusCode != 404 {
		t.Error("expected 404")
	}
}

func TestApiPlaylistsGetPlaylistsEmpty(t *testing.T) {
	r := httptest.NewRequest("GET", "https://takeout/api/playlists", nil)
	r = withContext(r, NewTestContext(t))

	w := httptest.NewRecorder()
	apiPlaylists(w, r)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Error("expected 200")
	}

	var view view.Playlists
	json.Unmarshal(body, &view)
	if len(view.Playlists) != 0 {
		t.Error("expected 0 entries")
	}
}
