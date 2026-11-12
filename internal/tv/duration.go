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
	"context"
	"fmt"
	"net/http"

	"takeoutfm.dev/takeout/lib/duration"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

// unscannedTVEpisodes returns up to limit episodes whose duration hasn't been read.
func (tv *TV) unscannedTVEpisodes(limit int) []TVEpisode {
	var list []TVEpisode
	tv.db.Where("(duration = 0 or duration is null) and key <> ''").Order("id").Limit(limit).Find(&list)
	return list
}

func (tv *TV) updateTVEpisodeDuration(x TVEpisode, seconds int) error {
	return tv.db.Model(&TVEpisode{}).Where("id = ?", x.ID).Update("duration", seconds).Error
}

// SyncDurations reads the duration of up to limit episodes that haven't been
// scanned, from the file headers, and stores it in seconds, or -1 if it can't
// be read. Files that fail to download are left for the next run.
func (tv *TV) SyncDurations(ctx context.Context, client *http.Client, limit int) []duration.Result {
	var results []duration.Result
	for _, x := range tv.unscannedTVEpisodes(limit) {
		if ctx.Err() != nil {
			break
		}
		name := fmt.Sprintf("%d S%02dE%02d %s", x.TVID, x.Season, x.Episode, x.Name)
		secs, err := duration.Seconds(ctx, client, tv.EpisodeURL(ctx, x), x.Size)
		if err != nil {
			log.Printf("duration %s: %s\n", name, err)
			continue
		}
		if secs < 0 {
			log.Printf("duration %s: unreadable\n", name)
		}
		if err := tv.updateTVEpisodeDuration(x, secs); err != nil {
			log.Println(err)
			continue
		}
		results = append(results, duration.Result{Name: name, ETag: x.ETag, Seconds: secs})
	}
	return results
}
