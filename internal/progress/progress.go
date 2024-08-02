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

// Package progress manages user progress data which contains media offset and
// duration to allow incremental watch/listen progress to be saved and
// retrieved to/from the server based on etag.
package progress

import (
	"errors"
	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	. "github.com/takeoutfm/takeout/model"
	"gorm.io/gorm"
)

var (
	ErrOffsetTooOld = errors.New("offset is old")
	ErrOffsetSame   = errors.New("offset is the same")
	ErrAccessDenied = errors.New("access denied")
)

type Progress struct {
	config *config.Config
	db     *gorm.DB
}

func NewProgress(config *config.Config) *Progress {
	return &Progress{
		config: config,
	}
}

func (p *Progress) Open() (err error) {
	err = p.openDB()
	return
}

func (p *Progress) Close() {
	p.closeDB()
}

// Offsets gets all the offets for the user.
func (p *Progress) Offsets(user auth.User) []Offset {
	offsets := p.userOffsets(user.Name)
	return offsets
}

// Offset gets the user offset based on the internal id.
func (p *Progress) Offset(user auth.User, id int) (Offset, error) {
	return p.lookupUserOffset(user.Name, id)
}

// Update will create or update an offset for the provided user using the etag
// as the primary key.
func (p *Progress) Update(user auth.User, newOffset Offset) error {
	offset, err := p.lookupUserOffsetEtag(user.Name, newOffset.ETag)
	if err != nil {
		return p.createOffset(&newOffset)
	}

	if newOffset.Date.Before(offset.Date) {
		return ErrOffsetTooOld
	} else if newOffset.Date == offset.Date {
		return ErrOffsetSame
	}
	offset.Offset = newOffset.Offset
	offset.Date = newOffset.Date
	if newOffset.Duration > 0 {
		offset.Duration = newOffset.Duration
	}

	return p.updateOffset(&offset)
}

// Delete will delete the provided user & offset, ensuring the offset belongs
// to the user.
func (p *Progress) Delete(user auth.User, offset Offset) error {
	if user.Name != offset.User {
		return ErrAccessDenied
	}
	return p.deleteOffset(&offset)
}
