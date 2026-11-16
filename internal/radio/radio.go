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

// Package radio schedules live radio stations from the music library. A
// station is only a schedule of tracks: tuning in plays the track on now at
// an offset, so nothing is broadcast and a station nobody hears costs
// nothing. It's separate from the TV guide (internal/guide) and shares no
// code with it.
package radio

import (
	"errors"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

type Radio struct {
	config *config.Config
	db     *gorm.DB
}

func NewRadio(config *config.Config) *Radio {
	return &Radio{config: config}
}

func (r *Radio) Open() error {
	cfg := r.config.Radio.DB.GormConfig()
	var err error
	switch r.config.Radio.DB.Driver {
	case "sqlite3":
		r.db, err = gorm.Open(sqlite.Open(r.config.Radio.DB.Source), cfg)
	default:
		err = errors.New("driver not supported")
	}
	if err != nil {
		return err
	}
	return r.db.AutoMigrate(&RadioTrack{})
}

func (r *Radio) Close() {
	if r.db == nil {
		return
	}
	conn, err := r.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

// Horizon is how far ahead schedules are built.
func (r *Radio) Horizon() time.Duration {
	return r.config.Radio.Horizon
}

// Schedule returns the tracks overlapping [start, end) on every station,
// by station then start time.
func (r *Radio) Schedule(start, end time.Time) []RadioTrack {
	var list []RadioTrack
	r.db.Where("start_at < ? and end_at > ?", end.UTC(), start.UTC()).
		Order("station, start_at").Find(&list)
	return list
}

// StationSchedule returns one station's tracks overlapping [start, end), in
// order.
func (r *Radio) StationSchedule(station string, start, end time.Time) []RadioTrack {
	var list []RadioTrack
	r.db.Where("station = ? and start_at < ? and end_at > ?", station, end.UTC(), start.UTC()).
		Order("start_at").Find(&list)
	return list
}

// history returns a station's scheduled tracks, in order.
func (r *Radio) history(station string) []RadioTrack {
	var list []RadioTrack
	r.db.Where("station = ?", station).Order("start_at").Find(&list)
	return list
}

// lastEnd returns when a station's schedule ends, or false with none.
func (r *Radio) lastEnd(station string) (time.Time, bool) {
	var list []RadioTrack
	r.db.Where("station = ?", station).Order("end_at desc").Limit(1).Find(&list)
	if len(list) == 0 {
		return time.Time{}, false
	}
	return list[0].End, true
}

func (r *Radio) createTracks(list []RadioTrack) error {
	if len(list) == 0 {
		return nil
	}
	// SQLite compares times as text, so store them all in UTC
	for i := range list {
		list[i].Start, list[i].End = list[i].Start.UTC(), list[i].End.UTC()
	}
	return r.db.CreateInBatches(list, 200).Error
}

// deleteEnded removes tracks that ended before t.
func (r *Radio) deleteEnded(t time.Time) error {
	return r.db.Unscoped().Where("end_at < ?", t.UTC()).Delete(&RadioTrack{}).Error
}
