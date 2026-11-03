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

func TestCover(t *testing.T) {
	cases := []struct {
		release model.Release
		expect  string
	}{
		{model.Release{REID: "re1", RGID: "rg1", Artwork: true, FrontArtwork: true},
			"/img/mb/re/re1/front"},
		{model.Release{REID: "re1", RGID: "rg1", Artwork: true, FrontArtwork: true, GroupArtwork: true},
			"/img/mb/rg/rg1/front"},
		// a release with no front image uses its other image, which the
		// covers job caches as /release/re1/123-250
		{model.Release{REID: "re1", RGID: "rg1", Artwork: true, OtherArtwork: "123"},
			"/img/mb/re/re1/123"},
		{model.Release{REID: "re1", RGID: "rg1"},
			"/static/album-white-36dp.svg"},
	}
	for _, c := range cases {
		if v := Cover(c.release, "250"); v != c.expect {
			t.Errorf("expect %s got %s", c.expect, v)
		}
	}
}
