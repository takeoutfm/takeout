// Copyright 2025 defsub
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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/takeoutfm/takeout/lib/gorm"
	g "gorm.io/gorm"
)

type TVSeries struct {
	gorm.Model
	TVID             int64 `gorm:"uniqueIndex:idx_series_tvid"`
	Name             string
	SortName         string
	Date             time.Time
	EndDate          time.Time
	Tagline          string
	OriginalName     string
	OriginalLanguage string
	Overview         string
	BackdropPath     string
	PosterPath       string
	SeasonCount      int
	EpisodeCount     int
	VoteAverage      float32
	VoteCount        int
	Rating           string
}

func (TVSeries) TableName() string {
	return "series"
}

// unique key is TVID, season, episode
type TVEpisode struct {
	gorm.Model
	UUID         string `gorm:"index:idx_episode_uuid" json:"-"`
	TVID         int64  `gorm:"index:idx_episode_tvid"`
	Name         string
	Overview     string
	Date         time.Time
	StillPath    string
	Season       int `gorm:"index:idx_episode_season"`
	Episode      int `gorm:"index:idx_episode_episode"`
	VoteAverage  float32
	VoteCount    int
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
}

func (TVEpisode) TableName() string {
	return "episodes"
}

func (e *TVEpisode) BeforeCreate(tx *g.DB) (err error) {
	e.UUID = uuid.NewString()
	return
}

type TVGenre struct {
	gorm.Model
	TVID int64 `gorm:"index:idx_genre_tvid"`
	Name string
}

func (TVGenre) TableName() string {
	return "genres"
}

type TVKeyword struct {
	gorm.Model
	TVID int64 `gorm:"index:idx_keyword_tvid"`
	Name string
}

func (TVKeyword) TableName() string {
	return "keywords"
}

type TVSeriesCast struct {
	gorm.Model
	TVID      int64 `gorm:"index:idx_cast_tvid"`
	PEID      int64 `gorm:"index:idx_cast_peid"`
	Character string
	Rank      int
	Person    Person `gorm:"-"`
}

func (TVSeriesCast) TableName() string {
	return "series_cast"
}

func (c TVSeriesCast) HasJob(name string) bool {
	return false
}

func (c TVSeriesCast) GetPerson() Person {
	return c.Person
}

type TVSeriesCrew struct {
	gorm.Model
	TVID       int64 `gorm:"index:idx_crew_tvid"`
	PEID       int64 `gorm:"index:idx_crew_peid"`
	Department string
	Job        string
	Person     Person `gorm:"-"`
}

func (TVSeriesCrew) TableName() string {
	return "series_crew"
}

func (c TVSeriesCrew) HasJob(name string) bool {
	return strings.EqualFold(c.Job, name)
}

func (c TVSeriesCrew) GetPerson() Person {
	return c.Person
}

type TVEpisodeCast struct {
	gorm.Model
	EID       uint  `gorm:"index:idx_episode_cast_eid"`
	PEID      int64 `gorm:"index:idx_episode_cast_peid"`
	Character string
	Rank      int
	Person    Person `gorm:"-"`
}

func (TVEpisodeCast) TableName() string {
	return "episode_cast"
}

func (c TVEpisodeCast) HasJob(name string) bool {
	return false
}

func (c TVEpisodeCast) GetPerson() Person {
	return c.Person
}

type TVEpisodeCrew struct {
	gorm.Model
	EID        uint  `gorm:"index:idx_episode_crew_eid"`
	PEID       int64 `gorm:"index:idx_episode_crew_peid"`
	Department string
	Job        string
	Person     Person `gorm:"-"`
}

func (TVEpisodeCrew) TableName() string {
	return "episode_crew"
}

func (c TVEpisodeCrew) HasJob(name string) bool {
	return strings.EqualFold(c.Job, name)
}

func (c TVEpisodeCrew) GetPerson() Person {
	return c.Person
}
