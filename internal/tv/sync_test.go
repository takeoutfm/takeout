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

package tv

import (
	"testing"
)

func TestTVRegexp1(t *testing.T) {
	matches := tvRegexp.FindStringSubmatch("/bucket/path/Sopranos (1999) - S05E21 - Made in America.mkv")
	if len(matches) == 0 {
		t.Fatal("expect matches")
	}

	show := matches[1]
	year := matches[2]
	episode := matches[3]
	title := matches[4]

	if show != "Sopranos" {
		t.Error("exepct sopranos")
	}
	if year != "1999" {
		t.Error("expect 1999")
	}
	if episode != "S05E21" {
		t.Error("expect S05E21")
	}
	if title != "Made in America" {
		t.Error("expect Made in America")
	}
}

func TestTVRegexp2(t *testing.T) {
	matches := tvRegexp.FindStringSubmatch("/bucket/path/Sopranos (1999) - S05E21.mkv")
	if len(matches) == 0 {
		t.Fatal("expect matches")
	}

	show := matches[1]
	year := matches[2]
	episode := matches[3]

	if show != "Sopranos" {
		t.Error("exepct sopranos")
	}
	if year != "1999" {
		t.Error("expect 1999")
	}
	if episode != "S05E21" {
		t.Error("expect S05E21")
	}
}

func TestEpisodeRegexp1(t *testing.T) {
	matches := episodeRegexp.FindStringSubmatch("S05E21")
	if len(matches) == 0 {
		t.Fatal("expect matches")
	}

	season := matches[1]
	episode := matches[2]

	if season != "05" {
		t.Error("05")
	}
	if episode != "21" {
		t.Error("expect 21")
	}
}

func TestEpisodeRegexp2(t *testing.T) {
	matches := episodeRegexp.FindStringSubmatch("s99e34")
	if len(matches) == 0 {
		t.Fatal("expect matches")
	}

	season := matches[1]
	episode := matches[2]

	if season != "99" {
		t.Error("99")
	}
	if episode != "34" {
		t.Error("expect 34")
	}
}

func TestEpisodeRegexp3(t *testing.T) {
	s, e, err := parseEpisode("S03E04")
	if err != nil {
		t.Error("expect no error")
	}
	if s != 3 {
		t.Error("expect season 3")
	}
	if e != 4 {
		t.Error("expect episode 4")
	}
}
