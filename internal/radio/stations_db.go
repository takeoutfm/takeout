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

package radio

import (
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
	. "takeoutfm.dev/takeout/model"
)

const (
	TypeLive   = "live"   // scheduled from the library
	TypeStream = "stream" // internet radio
)

var ErrStationNotFound = errors.New("station not found")

// Live returns the live stations on the air, in config order.
func (r *Radio) Live() []Station {
	var list []Station
	r.db.Where("type = ? and on_air", TypeLive).Order("rank, id").Find(&list)
	return list
}

// Streams returns the stream stations, in config order.
func (r *Radio) Streams() []Station {
	var list []Station
	r.db.Where("type = ?", TypeStream).Order("rank, id").Find(&list)
	return list
}

// listed reports whether a station is one that's offered: a stream, or a
// live station on the air.
func listed(s Station) bool {
	return s.Type == TypeStream || (s.Type == TypeLive && s.OnAir)
}

// LookupStation returns an offered station by ID.
func (r *Radio) LookupStation(id int) (Station, error) {
	var s Station
	err := r.db.First(&s, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !listed(s)) {
		return Station{}, ErrStationNotFound
	}
	return s, err
}

// FindStation finds an offered station by ID, or by name with a "name:"
// prefix (case-insensitive, exact first, then partial).
func (r *Radio) FindStation(identifier string) (Station, error) {
	if name, ok := strings.CutPrefix(identifier, "name:"); ok {
		list := r.StationsLike(name)
		for _, s := range list {
			if strings.EqualFold(s.Name, name) {
				return s, nil
			}
		}
		if len(list) > 0 {
			return list[0], nil
		}
		return Station{}, ErrStationNotFound
	}
	id, err := strconv.Atoi(identifier)
	if err != nil {
		return Station{}, ErrStationNotFound
	}
	return r.LookupStation(id)
}

// StationsLike returns the offered stations whose name or creator contains
// q, live ones first, in config order.
func (r *Radio) StationsLike(q string) []Station {
	var list []Station
	like := "%" + q + "%"
	r.db.Where("(name like ? or creator like ?) and (type = ? or on_air)", like, like, TypeStream).
		Order("type, rank, id").Find(&list)
	return list
}

func (r *Radio) stations() []Station {
	var list []Station
	r.db.Find(&list)
	return list
}
