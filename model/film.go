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
	"github.com/google/uuid"
	g "gorm.io/gorm"
	"strings"
	"takeoutfm.dev/takeout/lib/gorm"
	"time"
)

type Movie struct {
	gorm.Model
	UUID             string `gorm:"index:idx_movies_uuid" json:"-"`
	TMID             int64  `gorm:"uniqueIndex:idx_movies_tm_id"`
	IMID             string `gorm:"index:idx_movies_im_id"`
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
	Key              string `gorm:"index:idx_movies_key"`
	Size             int64
	ETag             string `gorm:"index:idx_movies_e_tag"`
	LastModified     time.Time
}

func (m *Movie) BeforeCreate(tx *g.DB) (err error) {
	m.UUID = uuid.NewString()
	return
}

type Collection struct {
	gorm.Model
	Name     string `gorm:"index:idx_collections_name"`
	SortName string
	TMID     int64 `gorm:"index:idx_collections_tm_id"`
}

type Genre struct {
	gorm.Model
	TMID int64 `gorm:"index:idx_genres_tm_id"`
	Name string
}

type Keyword struct {
	gorm.Model
	TMID int64 `gorm:"index:idx_keywords_tm_id"`
	Name string
}

type Cast struct {
	gorm.Model
	TMID      int64 `gorm:"index:idx_casts_tm_id"`
	PEID      int64 `gorm:"index:idx_casts_pe_id"`
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
	TMID       int64 `gorm:"index:idx_crews_tm_id"`
	PEID       int64 `gorm:"index:idx_crews_pe_id"`
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

type Trailer struct {
	gorm.Model
	TMID     int64 `gorm:"index:idx_trailers_tm_id"`
	Name     string
	Site     string
	Key      string
	Size     int
	Date     time.Time
	Official bool
	URL      string
}
