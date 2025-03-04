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

package listenbrainz // import "takeoutfm.dev/takeout/lib/listenbrainz"

import (
	"testing"

	"takeoutfm.dev/takeout/lib/client"
)

func TestArtistTopTracks(t *testing.T) {
	c := client.NewDefaultGetter()
	l := NewListenBrainz(c)
	tracks, err := l.ArtistTopTracks("6cb79cb2-9087-44d4-828b-5c6fdff2c957")
	if err != nil {
		t.Fatal(err)
	}
	for _, track := range tracks {
		t.Log(track.Rank(), track.Track())
	}
}
