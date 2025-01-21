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

package people

import (
	"errors"

	"github.com/takeoutfm/takeout/model"
	"gorm.io/gorm"
)

func Person(db *gorm.DB, peid int) (model.Person, error) {
	var person model.Person
	// TODO fix this logs an error every time and it's not an error
	err := db.Where("pe_id = ?", peid).First(&person).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Person{}, ErrPersonNotFound
	}
	return person, err
}

// func LookupPerson(db *gorm.DB, id int) (model.Person, error) {
// 	var person model.Person
// 	err := db.First(&person, id).Error
// 	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
// 		return model.Person{}, ErrPersonNotFound
// 	}
// 	return person, err
// }

func CreatePerson(db *gorm.DB, p *model.Person) error {
	return db.Create(p).Error
}
