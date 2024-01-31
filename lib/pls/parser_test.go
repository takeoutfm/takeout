// Copyright 2023 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package pls

import (
	"testing"
)

func TestParse(t *testing.T) {
	data := `
[playlist]
numberofentries=3
File1=https://ice2.somafm.com/dronezone-128-aac
Title1=SomaFM: Drone Zone (#1): Served best chilled, safe with most medications. Atmospheric textures with minimal beats.
Length1=-1
File2=https://ice1.somafm.com/dronezone-128-aac
Title2=SomaFM: Drone Zone (#2): Served best chilled, safe with most medications. Atmospheric textures with minimal beats.
Length2=-1
File3=https://ice4.somafm.com/dronezone-128-aac
Title3=SomaFM: Drone Zone (#3): Served best chilled, safe with most medications. Atmospheric textures with minimal beats.
Length3=-1
Version=2

`
	playlist, err := parse(data)
	if err != nil {
		t.Error(err)
	}

	if len(playlist.Entries) != 3 {
		t.Error("expect 3")
	}

	// 1
	if playlist.Entries[0].Index != 1 {
		t.Error("expect index = 1")
	}
	if playlist.Entries[0].File != "https://ice2.somafm.com/dronezone-128-aac" {
		t.Error("expect ice2")
	}
	if playlist.Entries[0].Title !=
		"SomaFM: Drone Zone (#1): Served best chilled, safe with most medications. Atmospheric textures with minimal beats." {
		t.Error("expect drone zone #1")
	}
	if playlist.Entries[0].Length != -1 {
		t.Error("expect length -1")
	}

	// 2
	if playlist.Entries[1].Index != 2 {
		t.Error("expect index = 2")
	}
	if playlist.Entries[1].File != "https://ice1.somafm.com/dronezone-128-aac" {
		t.Error("expect ice1")
	}
	if playlist.Entries[1].Title !=
		"SomaFM: Drone Zone (#2): Served best chilled, safe with most medications. Atmospheric textures with minimal beats." {
		t.Error("expect drone zone #1")
	}
	if playlist.Entries[1].Length != -1 {
		t.Error("expect length -1")
	}

	// 3
	if playlist.Entries[2].Index != 3 {
		t.Error("expect index = 3")
	}
	if playlist.Entries[2].File != "https://ice4.somafm.com/dronezone-128-aac" {
		t.Error("expect ice4")
	}
	if playlist.Entries[2].Title !=
		"SomaFM: Drone Zone (#3): Served best chilled, safe with most medications. Atmospheric textures with minimal beats." {
		t.Error("expect drone zone #1")
	}
	if playlist.Entries[2].Length != -1 {
		t.Error("expect length -1")
	}
}
