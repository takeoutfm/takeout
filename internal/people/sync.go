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
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/tmdb"
	"takeoutfm.dev/takeout/model"
)

func PersonDetail(client *tmdb.TMDB, peid int) (model.Person, error) {
	detail, err := client.PersonDetail(peid)
	if err != nil {
		return model.Person{}, err
	}
	p := model.Person{
		PEID:        int64(peid),
		IMID:        detail.IMDB_ID,
		Name:        detail.Name,
		ProfilePath: detail.ProfilePath,
		Bio:         detail.Biography,
		Birthplace:  detail.Birthplace,
		Birthday:    date.ParseDate(detail.Birthday),
		Deathday:    date.ParseDate(detail.Deathday),
	}
	return p, nil
}

func EnsurePerson(peid int, client *tmdb.TMDB, db *gorm.DB) (model.Person, error) {
	p, err := Person(db, peid)
	if err != nil {
		p, err = PersonDetail(client, peid)
		if err != nil {
			return p, err
		}
		err = CreatePerson(db, &p)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}
