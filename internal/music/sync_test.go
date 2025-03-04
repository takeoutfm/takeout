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

	"takeoutfm.dev/takeout/model"
)

func TestPickDisamiguation(t *testing.T) {
	m := makeMusic(t)

	releases := []model.Release{
		{
			Name:           "Weezer",
			Disambiguation: "Blue Album",
			Status:         "Official",
			Country:        "XE",
			FrontArtwork:   true,
		},
		{
			Name:           "Weezer",
			Disambiguation: "Blue Album",
			Status:         "Bootleg",
			Country:        "US",
			FrontArtwork:   true,
		},
		{
			Name:           "Weezer",
			Disambiguation: "Blue Album",
			Status:         "Official",
			Country:        "US",
			FrontArtwork:   false,
		},
		{
			Name:           "Weezer",
			Disambiguation: "Blue Album",
			Status:         "Official",
			Country:        "US",
			FrontArtwork:   true,
		},
		{
			Name:           "Weezer",
			Disambiguation: "Blue Album, Deluxe Edition",
			Status:         "Official",
			Country:        "US",
			FrontArtwork:   true,
		},
	}

	tracks := []model.Track{
		{Release: "Weezer (Blue Album)"},
		{Release: "Weezer - Blue Album"},
		{Release: "Weezer Blue Album"},
		{Release: "Weezer [Blue Album]"},
		{Release: "Blue Album"},
	}

	for _, track := range tracks {
		pick := m.pickDisambiguation(track, releases)
		if pick == -1 {
			t.Fatal("expect release")
		}
		r := releases[pick]
		if r.Country != "US" {
			t.Error("expect US release" + r.Country)
		}
		if r.Status != "Official" {
			t.Error("expect official release")
		}
		if r.FrontArtwork != true {
			t.Error("expect front art")
		}
	}

}

func TestPickRelease(t *testing.T) {
	m := makeMusic(t)

	releases := []model.Release{
		{
			Name:           "Master of Reality",
			Disambiguation: "",
			Status:         "Official",
			Country:        "XE",
			FrontArtwork:   true,
		},
		{
			Name:           "Master of Reality",
			Disambiguation: "",
			Status:         "Official",
			Country:        "US",
			FrontArtwork:   false,
		},
		{
			Name:           "Master of Reality",
			Disambiguation: "Deluxe Edition",
			Status:         "Official",
			Country:        "US",
			FrontArtwork:   true,
		},
		{
			Name:           "Master of Reality",
			Disambiguation: "",
			Status:         "Bootleg",
			Country:        "US",
			FrontArtwork:   true,
		},
		{
			Name:           "Master of Reality",
			Disambiguation: "",
			Status:         "Official",
			Country:        "US",
			FrontArtwork:   true,
		},
	}

	pick := m.pickRelease(releases)
	if pick == -1 {
		t.Fatal("expect release")
	}
	r := releases[pick]
	if r.Country != "US" {
		t.Error("expect US release")
	}
	if r.FrontArtwork != true {
		t.Error("expect release with front art")
	}
	if r.Disambiguation != "" {
		t.Error("expect no disambiguation")
	}
	if r.Status != "Official" {
		t.Error("expect official")
	}

}
