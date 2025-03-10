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
	"takeoutfm.dev/takeout/lib/gorm"
	"time"
)

type Series struct {
	gorm.Model
	SID         string `gorm:"uniqueIndex:idx_series"` // hash of link
	Title       string
	Description string
	Author      string
	Link        string
	Image       string
	Copyright   string
	Date        time.Time // last build date
	TTL         int
}

func (Series) TableName() string {
	return "series" // series is zero plural
}

type Episode struct {
	gorm.Model
	SID         string // series ID
	EID         string `gorm:"uniqueIndex:idx_episode"` // GUID
	Title       string
	Author      string
	Link        string
	Description string
	ContentType string
	Size        int64
	URL         string
	Date        time.Time // publish time
	Image       string
}

type Subscription struct {
	gorm.Model
	SID  string `gorm:"primaryKey"`
	User string `gorm:"primaryKey"`
}
