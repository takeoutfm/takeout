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
	"strconv"
	"strings"

	"gorm.io/gorm"
	"takeoutfm.dev/takeout/lib/tmdb"
	"takeoutfm.dev/takeout/model"
)

var (
	JobDirector          = "Director"
	JobNovel             = "Novel"
	JobScreenplay        = "Screenplay"
	JobStory             = "Story"
	JobProducer          = "Producer"
	JobExecutiveProducer = "Executive Producer"
)

func FindPerson(db *gorm.DB, identifier string) (model.Person, error) {
	id, err := strconv.Atoi(identifier)
	if err != nil {
		return model.Person{}, ErrPersonNotFound
	} else {
		return Person(db, id)
	}
}

func PersonProfile(p model.Person) string {
	if p.ProfilePath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Profile185, p.ProfilePath}, "")
}

func PersonProfileSmall(p model.Person) string {
	if p.ProfilePath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Profile45, p.ProfilePath}, "")
}

func NewBilling[A model.Role, B model.Role](cast []A, crew []B) (billing model.Billing) {
	billing.Actors = featured(cast)
	billing.Directors = jobs(crew, []string{JobDirector})
	billing.Producers = jobs(crew, []string{JobExecutiveProducer, JobProducer})
	billing.Writers = jobs(crew, []string{JobNovel, JobScreenplay, JobStory})
	return
}

func featured[T model.Role](cast []T) []model.Person {
	var result []model.Person
	for i, c := range cast {
		if i == 3 {
			break
		}
		result = append(result, c.GetPerson())
	}
	return result
}

func jobs[T model.Role](crew []T, jobs []string) []model.Person {
	var result []model.Person
	for _, c := range crew {
		for _, j := range jobs {
			if c.HasJob(j) {
				result = append(result, c.GetPerson())
			}
		}
	}
	return result
}
