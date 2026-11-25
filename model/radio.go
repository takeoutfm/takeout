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

// Station is a radio station: live, scheduled from the library, or an
// internet radio stream. The radio job keeps the rows in step with the
// Radio config, matched by name, so a station keeps its ID while its name
// stays the same.
type Station struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex"`
	Type        string // live or stream
	Creator     string
	Image       string
	Description string
	Source      string `json:"-"` // a stream's sources, as JSON
	Rank        int    `json:"-"` // order in the config
	OnAir       bool   `json:"-"` // a live station that's scheduled now
	Plan        string `json:"-"` // a live station's music and shows, to see changes
}

// RadioTrack is one scheduled track on a live radio station. Stations are
// keyed by name, so renaming one starts a fresh schedule, and tracks by
// recording ID, since music syncs recreate track rows.
type RadioTrack struct {
	gorm.Model
	Station string    `gorm:"index:idx_radio_tracks_station_start,priority:1"`
	Start   time.Time `gorm:"column:start_at;index:idx_radio_tracks_station_start,priority:2"`
	End     time.Time `gorm:"column:end_at;index:idx_radio_tracks_end"`
	RID     string
	Show    string // the show it's part of, if any
}
