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

package tv

import (
	"testing"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/model"
)

// Episodes from before the duration column existed have NULL there, not 0,
// and still need scanning.
func TestUnscannedTVEpisodes(t *testing.T) {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	tv := NewTV(config)
	if err := tv.Open(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer tv.Close()

	episodes := []model.TVEpisode{
		{TVID: 77, Season: 1, Episode: 1, Key: "a.mkv"},
		{TVID: 77, Season: 1, Episode: 2, Key: "b.mkv"},
		{TVID: 77, Season: 1, Episode: 3, Key: "c.mkv", Duration: 42},
		{TVID: 77, Season: 1, Episode: 4},
	}
	for i := range episodes {
		if err := tv.createEpisode(&episodes[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := tv.db.Exec("update episodes set duration = null where tv_id = 77 and episode = 2").Error; err != nil {
		t.Fatal(err)
	}

	var got []int
	for _, e := range tv.unscannedTVEpisodes(100) {
		if e.TVID == 77 {
			got = append(got, e.Episode)
		}
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("got episodes %v, want [1 2]", got)
	}
}
