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

package model // import "takeoutfm.dev/takeout/model"

import (
	"time"

	"takeoutfm.dev/takeout/lib/gorm"
)

type Offset struct {
	gorm.Model
	User     string    `gorm:"index:idx_offset_user" json:"-"`
	ETag     string    `gorm:"uniqueIndex:idx_offset_etag;uniqueIndex:idx_offset_date"`
	Offset   int       `gorm:"default:0"`
	Duration int       `gorm:"default:0"`
	Date     time.Time `gorm:"uniqueIndex:idx_offset_date"`
}

type Offsets struct {
	Offsets []Offset
}

func (o Offset) Valid() bool {
	if len(o.User) == 0 || len(o.ETag) == 0 || o.Offset < 0 || o.Date.IsZero() {
		return false
	}
	// duration can be unknown (0) but if known, offset must be within
	// duration
	if o.Duration > 0 && o.Offset > o.Duration {
		return false
	}
	return true
}
