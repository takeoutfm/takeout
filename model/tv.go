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

package model // import "takeoutfm.dev/takeout/model"

import (
	"strings"
	"time"

	"github.com/google/uuid"
	g "gorm.io/gorm"
	"takeoutfm.dev/takeout/lib/gorm"
)

type TVSeries struct {
	gorm.Model
	TVID             int64 `gorm:"uniqueIndex:idx_series_tv_id"`
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
	UUID         string `gorm:"index:idx_episodes_uuid" json:"-"`
	TVID         int64  `gorm:"uniqueIndex:idx_episodes_tv_id_season_episode,priority:1"`
	Name         string
	Overview     string
	Date         time.Time
	StillPath    string
	Season       int `gorm:"uniqueIndex:idx_episodes_tv_id_season_episode,priority:2"`
	Episode      int `gorm:"uniqueIndex:idx_episodes_tv_id_season_episode,priority:3"`
	VoteAverage  float32
	VoteCount    int
	Runtime      int
	Key          string
	Size         int64
	ETag         string `gorm:"index:idx_episodes_e_tag"`
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
	TVID int64 `gorm:"index:idx_genres_tv_id"`
	Name string
}

func (TVGenre) TableName() string {
	return "genres"
}

type TVKeyword struct {
	gorm.Model
	TVID int64 `gorm:"index:idx_keywords_tv_id"`
	Name string
}

func (TVKeyword) TableName() string {
	return "keywords"
}

type TVSeriesCast struct {
	gorm.Model
	TVID      int64 `gorm:"index:idx_series_cast_tv_id"`
	PEID      int64 `gorm:"index:idx_series_cast_pe_id"`
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
	TVID       int64 `gorm:"index:idx_series_crew_tv_id"`
	PEID       int64 `gorm:"index:idx_series_crew_pe_id"`
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
	EID       uint  `gorm:"index:idx_episode_cast_e_id"`
	PEID      int64 `gorm:"index:idx_episode_cast_pe_id"`
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
	EID        uint  `gorm:"index:idx_episode_crew_e_id"`
	PEID       int64 `gorm:"index:idx_episode_crew_pe_id"`
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
