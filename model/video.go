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

package model

import (
	"github.com/google/uuid"
	"github.com/takeoutfm/takeout/lib/gorm"
	g "gorm.io/gorm"
	"strings"
	"time"
)

type Movie struct {
	gorm.Model
	UUID             string `gorm:"index:idx_movie_uuid" json:"-"`
	TMID             int64  `gorm:"uniqueIndex:idx_movie_tmid"`
	IMID             string
	Title            string
	Date             time.Time
	Rating           string
	Tagline          string
	OriginalTitle    string
	OriginalLanguage string
	Overview         string
	Budget           int64
	Revenue          int64
	Runtime          int
	VoteAverage      float32
	VoteCount        int
	BackdropPath     string
	PosterPath       string
	SortTitle        string
	Key              string
	Size             int64
	ETag             string
	LastModified     time.Time
}

func (m *Movie) BeforeCreate(tx *g.DB) (err error) {
	m.UUID = uuid.NewString()
	return
}

type Collection struct {
	gorm.Model
	Name     string
	SortName string
	TMID     int64
}

type Genre struct {
	gorm.Model
	TMID int64
	Name string
}

type Keyword struct {
	gorm.Model
	TMID int64 `gorm:"index:idx_keyword_tmid"`
	Name string
}

type Cast struct {
	gorm.Model
	TMID      int64 `gorm:"index:idx_cast_tmid"`
	PEID      int64 `gorm:"index:idx_cast_peid"`
	Character string
	Rank      int
	Person    Person `gorm:"-"`
}

func (Cast) TableName() string {
	return "cast" // not casts
}

func (c Cast) HasJob(name string) bool {
	return false
}

func (c Cast) GetPerson() Person {
	return c.Person
}

type Crew struct {
	gorm.Model
	TMID       int64 `gorm:"index:idx_crew_tmid"`
	PEID       int64 `gorm:"index:idx_crew_peid"`
	Department string
	Job        string
	Person     Person `gorm:"-"`
}

func (Crew) TableName() string {
	return "crew" // not crews
}

func (c Crew) HasJob(name string) bool {
	return strings.EqualFold(c.Job, name)
}

func (c Crew) GetPerson() Person {
	return c.Person
}

type Recommend struct {
	Name   string
	Movies []Movie
}
