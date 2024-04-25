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

// Package spiff provides the model for all playlists used within TakeoutFM. A
// spiff is a container for one or more media tracks with basic media metadata.
// TakeoutFM also uses json patch to manipulate spiffs for remote playlist
// management.
package spiff

import (
	"encoding/json"
)

// See the following specifications:
//  https://www.xspf.org/spec
//  https://www.xspf.org/jspf

type Header struct {
	Title    string  `json:"title"`
	Creator  string  `json:"creator,omitempty"`
	Image    string  `json:"image,omitempty"`
	Location string  `json:"location,omitempty"`
	Date     string  `json:"date,omitempty"` // "2005-01-08T17:10:47-05:00",
}

type Playlist struct {
	Spiff    Spiff   `json:"playlist"`
	Index    int     `json:"index"`
	Position float64 `json:"position"`
	Type     string  `json:"type"`
}

type Spiff struct {
	Header
	Entries  []Entry `json:"track"`
}

type Entry struct {
	Ref        string   `json:"$ref,omitempty"`
	Creator    string   `json:"creator,omitempty" spiff:"creator"`
	Album      string   `json:"album,omitempty" spiff:"album"`
	Title      string   `json:"title,omitempty" spiff:"title"`
	Image      string   `json:"image,omitempty" spiff:"image"`
	Location   []string `json:"location,omitempty" spiff:"location"`
	Identifier []string `json:"identifier,omitempty" spiff:"identifier"`
	Size       []int64  `json:"size,omitempty"`
	Date       string   `json:"date,omitempty" spiff:"date"` // "2005-01-08T17:10:47-05:00",
}

const (
	TypeMusic   = "music"
	TypeVideo   = "video"
	TypePodcast = "podcast"
	TypeStream  = "stream"
)

func NewPlaylist(listType string) *Playlist {
	return &Playlist{Spiff{Header{}, []Entry{}}, -1, 0, listType}
}

func Unmarshal(data []byte) (*Playlist, error) {
	var playlist Playlist
	err := json.Unmarshal(data, &playlist)
	return &playlist, err
}

func (playlist *Playlist) Marshal() ([]byte, error) {
	data, err := json.Marshal(playlist)
	return data, err
}
