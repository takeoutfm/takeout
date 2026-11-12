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

package model // import "takeoutfm.dev/takeout/model"

import (
	"time"

	"takeoutfm.dev/takeout/lib/gorm"
)

// Channel is a named channel from the Guide lineup. Rows are matched by
// name, so their IDs are stable across media syncs.
type Channel struct {
	gorm.Model
	Name string `gorm:"uniqueIndex:idx_channels_name"`
}

const (
	ProgramMovie   = "movie"
	ProgramEpisode = "episode"
	ProgramTrack   = "track"
)

// Program is one scheduled entry on a channel. Titles are keyed by external
// IDs, not media rows, since media syncs recreate rows.
type Program struct {
	gorm.Model
	ChannelID uint      `gorm:"index:idx_programs_channel_start,priority:1"`
	Start     time.Time `gorm:"column:start_at;index:idx_programs_channel_start,priority:2"`
	End       time.Time `gorm:"column:end_at;index:idx_programs_end"` // when the file ends
	Kind      string
	Block     string // special or feature name, empty for regular programming
	TMID      int64
	TVID      int64
	Season    int
	Episode   int
	RID       string
}
