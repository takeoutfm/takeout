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
	"context"
	"net/http"

	"takeoutfm.dev/takeout/lib/duration"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

// unscannedTracks returns up to limit tracks whose duration hasn't been read.
func (m *Music) unscannedTracks(limit int) []Track {
	var list []Track
	m.db.Where("duration = 0 and key <> ''").Order("id").Limit(limit).Find(&list)
	return list
}

func (m *Music) updateTrackDuration(x Track, seconds int) error {
	return m.db.Model(&Track{}).Where("id = ?", x.ID).Update("duration", seconds).Error
}

// SyncDurations reads the duration of up to limit tracks that haven't been
// scanned, from the file headers, and stores it in seconds, or -1 if it can't
// be read. Files that fail to download are left for the next run.
func (m *Music) SyncDurations(ctx context.Context, client *http.Client, limit int) []duration.Result {
	var results []duration.Result
	for _, x := range m.unscannedTracks(limit) {
		if ctx.Err() != nil {
			break
		}
		name := x.Artist + " / " + x.Title
		secs, err := duration.Seconds(ctx, client, m.TrackURL(ctx, x), x.Size)
		if err != nil {
			log.Printf("duration %s: %s\n", name, err)
			continue
		}
		if secs < 0 {
			log.Printf("duration %s: unreadable\n", name)
		}
		if err := m.updateTrackDuration(x, secs); err != nil {
			log.Println(err)
			continue
		}
		results = append(results, duration.Result{Name: name, ETag: x.ETag, Seconds: secs})
	}
	return results
}
