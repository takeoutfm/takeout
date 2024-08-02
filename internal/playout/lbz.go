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

package playout

import (
	"time"

	lbz "github.com/kori/go-listenbrainz"
	"github.com/takeoutfm/takeout/player"
)

func lbzTrack(p *player.Player) lbz.Track {
	return lbz.Track{
		Artist: p.Artist(),
		Album:  p.Album(),
		Title:  p.Title(),
	}
}

func (playout *Playout) lbzNowPlaying(p *player.Player) {
	if p.IsMusic() {
		if lbzToken := playout.ListenBrainzToken(); len(lbzToken) > 0 {
			go func() {
				lbz.SubmitPlayingNow(lbzTrack(p), lbzToken)
			}()
		}
	}
}

func (playout *Playout) lbzListened(p *player.Player) {
	if p.IsMusic() {
		if lbzToken := playout.ListenBrainzToken(); len(lbzToken) > 0 {
			go func() {
				lbz.SubmitSingle(lbzTrack(p), lbzToken, time.Now().Unix())
			}()
		}
	}
}
