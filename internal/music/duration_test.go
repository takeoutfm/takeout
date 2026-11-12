// Copyright 2026 defsub
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

// Tracks from before the duration column existed have NULL there, not 0, and
// still need scanning.
func TestUnscannedTracks(t *testing.T) {
	m := makeMusic(t)
	tracks := []model.Track{
		{Artist: "dur", Release: "r", TrackNum: 1, Title: "zero", Key: "a.flac"},
		{Artist: "dur", Release: "r", TrackNum: 2, Title: "null", Key: "b.flac"},
		{Artist: "dur", Release: "r", TrackNum: 3, Title: "done", Key: "c.flac", Duration: 42},
		{Artist: "dur", Release: "r", TrackNum: 4, Title: "no key"},
	}
	for i := range tracks {
		if err := m.createTrack(&tracks[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.db.Exec("update tracks set duration = null where artist = 'dur' and title = 'null'").Error; err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, x := range m.unscannedTracks(100) {
		if x.Artist == "dur" {
			got = append(got, x.Title)
		}
	}
	if len(got) != 2 || got[0] != "zero" || got[1] != "null" {
		t.Errorf("got %v, want [zero null]", got)
	}
}
