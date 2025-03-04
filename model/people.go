// Copyright 2024 defsub
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

type Person struct {
	gorm.Model
	PEID        int64 `gorm:"uniqueIndex:idx_person_peid"`
	IMID        string
	Name        string
	ProfilePath string
	Bio         string
	Birthplace  string
	Birthday    time.Time
	Deathday    time.Time
}

func (Person) TableName() string {
	return "people" // not peoples
}

type Billing struct {
	Actors    []Person
	Directors []Person
	Producers []Person
	Writers   []Person
}

type Role interface {
	GetPerson() Person
	HasJob(string) bool
}
