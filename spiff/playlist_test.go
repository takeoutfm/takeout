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

package spiff // import "takeoutfm.dev/takeout/spiff"

import (
	"testing"
)

func TestPlaylist(t *testing.T) {
	p := Playlist{
		Spiff: Spiff{
			Header: Header{
				Title:    "test playlist",
				Creator:  "test creator",
				Image:    "https://img.com/img.png",
				Location: "https://t.com/t.spiff",
				Date:     "2024-1-1",
			},
			Entries: []Entry{
				{
					Creator:  "Gary Numan",
					Album:    "Live",
					Title:    "Films",
					Location: []string{"https:/t./com/films.flac"},
				},
			},
		},
	}

	data, err := p.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	plist, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}

	if p.Spiff.Creator != plist.Spiff.Creator {
		t.Error("expect same creator")
	}

	if p.Length() != plist.Length() {
		t.Error("expect same entries")
	}

	if len(p.Spiff.Entries[0].Location) != len(plist.Spiff.Entries[0].Location) {
		t.Error("expect same locations")
	}
}

func TestNewPlaylist(t *testing.T) {
	p := NewPlaylist(TypeMusic)
	if p.Type != TypeMusic {
		t.Error("expect type music")
	}
}
