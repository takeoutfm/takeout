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

package film

import (
	"context"
	"net/http"

	"takeoutfm.dev/takeout/lib/duration"
	"takeoutfm.dev/takeout/lib/log"
	. "takeoutfm.dev/takeout/model"
)

// unscannedMovies returns up to limit movies whose duration hasn't been read.
func (f *Film) unscannedMovies(limit int) []Movie {
	var list []Movie
	f.db.Where("duration = 0 and key <> ''").Order("id").Limit(limit).Find(&list)
	return list
}

func (f *Film) updateMovieDuration(x Movie, seconds int) error {
	return f.db.Model(&Movie{}).Where("id = ?", x.ID).Update("duration", seconds).Error
}

// SyncDurations reads the duration of up to limit movies that haven't been
// scanned, from the file headers, and stores it in seconds, or -1 if it can't
// be read. Files that fail to download are left for the next run.
func (f *Film) SyncDurations(ctx context.Context, client *http.Client, limit int) []duration.Result {
	var results []duration.Result
	for _, x := range f.unscannedMovies(limit) {
		if ctx.Err() != nil {
			break
		}
		name := x.Title
		secs, err := duration.Seconds(ctx, client, f.MovieURL(ctx, x), x.Size)
		if err != nil {
			log.Printf("duration %s: %s\n", name, err)
			continue
		}
		if secs < 0 {
			log.Printf("duration %s: unreadable\n", name)
		}
		if err := f.updateMovieDuration(x, secs); err != nil {
			log.Println(err)
			continue
		}
		results = append(results, duration.Result{Name: name, ETag: x.ETag, Seconds: secs})
	}
	return results
}
