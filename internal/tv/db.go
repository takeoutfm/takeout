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

package tv

import (
	"errors"
	"time"

	"github.com/takeoutfm/takeout/internal/people"
	. "github.com/takeoutfm/takeout/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func (tv *TV) openDB() (err error) {
	cfg := tv.config.TV.DB.GormConfig()

	switch tv.config.TV.DB.Driver {
	case "sqlite3":
		tv.db, err = gorm.Open(sqlite.Open(tv.config.TV.DB.Source), cfg)
	case "mysql":
		tv.db, err = gorm.Open(mysql.Open(tv.config.TV.DB.Source), cfg)
	case "postgres":
		// postgres untested
		tv.db, err = gorm.Open(postgres.Open(tv.config.TV.DB.Source), cfg)
	default:
		err = errors.New("driver not supported")
	}

	if err != nil {
		return
	}

	tv.db.AutoMigrate(
		&TVSeriesCast{}, &TVSeriesCrew{},
		&TVEpisodeCast{}, &TVEpisodeCrew{},
		&TVGenre{}, &TVKeyword{}, &TVSeries{}, &TVEpisode{}, &Person{})
	return
}

func (tv *TV) closeDB() {
	conn, err := tv.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

func (tv *TV) Series() []TVSeries {
	var series []TVSeries
	tv.db.Order("sort_name").Find(&series)
	return series
}

func (tv *TV) Episodes(series TVSeries) []TVEpisode {
	var episodes []TVEpisode
	tv.db.Where(`episodes.tv_id = ?`, series.TVID).
		Order("season asc, episode asc").Find(&episodes)
	return episodes
}

func (tv *TV) FindSeasonEpisode(series TVSeries, season, episode int) (TVEpisode, error) {
	var ep TVEpisode
	err := tv.db.First(&ep, `episodes.tv_id = ? and episodes.season = ? and episodes.episode = ?`,
		series.TVID, season, episode).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return TVEpisode{}, ErrEpisodeNotFound
	}
	return ep, err

}

func (tv *TV) Genre(name string) []TVSeries {
	var series []TVSeries
	tv.db.Where("series.tv_id in (select tv_id from genres where name = ?)", name).
		Order("series.date").Find(&series)
	return series
}

func (tv *TV) Genres(series TVSeries) []string {
	var genres []Genre
	var list []string
	tv.db.Where("tv_id = ?", series.TVID).Order("name").Find(&genres)
	for _, g := range genres {
		list = append(list, g.Name)
	}
	return list
}

func (tv *TV) Keyword(name string) []TVSeries {
	var series []TVSeries
	tv.db.Where("series.tv_id in (select tv_id from keywords where name = ?)", name).
		Order("series.date").Find(&series)
	return series
}

func (tv *TV) Keywords(series TVSeries) []string {
	var keywords []Keyword
	var list []string
	tv.db.Where("tv_id = ?", series.TVID).Order("name").Find(&keywords)
	for _, g := range keywords {
		list = append(list, g.Name)
	}
	return list
}

func (tv *TV) SeriesCast(series TVSeries) []TVSeriesCast {
	var cast []TVSeriesCast
	var people []Person
	tv.db.Order("rank asc").
		Joins(`inner join series on "series_cast".tv_id = series.tv_id`).
		Where("series.tv_id = ?", series.TVID).Find(&cast)
	tv.db.Joins(`inner join "series_cast" on people.pe_id = "series_cast".pe_id`).
		Joins(`inner join series on series.tv_id = "series_cast".tv_id`).
		Where("series.tv_id = ?", series.TVID).Find(&people)
	pmap := make(map[int64]Person)
	for _, p := range people {
		pmap[p.PEID] = p
	}
	for i := range cast {
		cast[i].Person = pmap[cast[i].PEID]
	}
	return cast
}

func (tv *TV) SeriesCrew(series TVSeries) []TVSeriesCrew {
	var crew []TVSeriesCrew
	var people []Person
	tv.db.Joins(`inner join series on "series_crew".tv_id = series.tv_id`).
		Where("series.tv_id = ?", series.TVID).Find(&crew)
	tv.db.Joins(`inner join "series_crew" on people.pe_id = "series_crew".pe_id`).
		Joins(`inner join series on series.tv_id = "series_crew".tv_id`).
		Where("series.tv_id = ?", series.TVID).Find(&people)
	pmap := make(map[int64]Person)
	for _, p := range people {
		pmap[p.PEID] = p
	}
	for i := range crew {
		crew[i].Person = pmap[crew[i].PEID]
	}
	return crew
}

func (tv *TV) EpisodeCast(episode TVEpisode) []TVEpisodeCast {
	var cast []TVEpisodeCast
	var people []Person
	tv.db.Order("rank asc").
		Joins(`inner join episodes on episode_cast.e_id = episodes.id`).
		Where("episodes.id = ?", episode.ID).Find(&cast)
	tv.db.Joins(`inner join episode_cast on people.pe_id = episode_cast.pe_id`).
		Joins(`inner join episodes on episodes.id = episode_cast.e_id`).
		Where("episodes.id = ?", episode.ID).Find(&people)
	pmap := make(map[int64]Person)
	for _, p := range people {
		pmap[p.PEID] = p
	}
	for i := range cast {
		cast[i].Person = pmap[cast[i].PEID]
	}
	return cast
}

func (tv *TV) EpisodeCrew(episode TVEpisode) []TVEpisodeCrew {
	var crew []TVEpisodeCrew
	var people []Person
	tv.db.Joins(`inner join episodes on "episode_crew".e_id = episodes.id`).
		Where("episodes.id = ?", episode.ID).Find(&crew)
	tv.db.Joins(`inner join "episode_crew" on people.pe_id = "episode_crew".pe_id`).
		Joins(`inner join episodes on episodes.id = "episode_crew".e_id`).
		Where("episodes.id = ?", episode.ID).Find(&people)
	pmap := make(map[int64]Person)
	for _, p := range people {
		pmap[p.PEID] = p
	}
	for i := range crew {
		crew[i].Person = pmap[crew[i].PEID]
	}
	return crew
}

func (tv *TV) SeriesStarring(p Person) []TVSeries {
	var series []TVSeries
	tv.db.Where(`series.tv_id in (select tv_id from series_cast where pe_id = ?)`, p.PEID).
		Order("series.date").Find(&series)
	return series
}

func (tv *TV) department(dept string, p Person) []TVSeries {
	var series []TVSeries
	tv.db.Where(`series.tv_id in (select tv_id from series_crew where department = ? and pe_id = ?)`,
		dept, p.PEID).
		Order("series.date").Find(&series)
	return series
}

func (tv *TV) SeriesDirecting(p Person) []TVSeries {
	return tv.department("Directing", p)
}

func (tv *TV) SeriesProducing(p Person) []TVSeries {
	return tv.department("Production", p)
}

func (tv *TV) SeriesWriting(p Person) []TVSeries {
	return tv.department("Writing", p)
}

func (tv *TV) deleteSeries(tvid int) {
	var list []TVSeries
	tv.db.Where("tv_id = ?", tvid).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteSeriesCast(tvid int) {
	var list []TVSeriesCast
	tv.db.Where("tv_id = ?", tvid).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteSeriesCrew(tvid int) {
	var list []TVSeriesCrew
	tv.db.Where("tv_id = ?", tvid).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteEpisode(tvid, season, episode int) {
	var list []TVEpisode
	tv.db.Where("tv_id = ? and season = ? and episode = ?", tvid, season, episode).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteEpisodeCast(e TVEpisode) {
	var list []TVEpisodeCast
	tv.db.Where("e_id = ?", e.ID).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteEpisodeCrew(e TVEpisode) {
	var list []TVEpisodeCrew
	tv.db.Where("e_id = ?", e.ID).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteGenres(tvid int) {
	var list []TVGenre
	tv.db.Where("tv_id = ?", tvid).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) deleteKeywords(tvid int) {
	var list []TVKeyword
	tv.db.Where("tv_id = ?", tvid).Find(&list)
	for _, o := range list {
		tv.db.Unscoped().Delete(o)
	}
}

func (tv *TV) Person(peid int) (Person, error) {
	return people.Person(tv.db, peid)
}

func (tv *TV) LookupSeries(id int) (TVSeries, error) {
	var series TVSeries
	err := tv.db.First(&series, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return TVSeries{}, ErrSeriesNotFound
	}
	return series, err
}

func (tv *TV) LookupEpisode(id int) (TVEpisode, error) {
	var episode TVEpisode
	err := tv.db.First(&episode, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return TVEpisode{}, ErrEpisodeNotFound
	}
	return episode, err
}

func (tv *TV) LookupTVID(tvid int) (TVSeries, error) {
	var series TVSeries
	err := tv.db.First(&series, "tv_id = ?", tvid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return TVSeries{}, ErrSeriesNotFound
	}
	return series, err
}

func (tv *TV) LookupUUID(uuid string) (TVEpisode, error) {
	var episode TVEpisode
	err := tv.db.First(&episode, "uuid = ?", uuid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return TVEpisode{}, ErrEpisodeNotFound
	}
	return episode, err
}

func (tv *TV) EpisodeStarring(p Person) []TVEpisode {
	var episodes []TVEpisode
	tv.db.Where(`episodes.e_id in (select e_id from episode_cast where pe_id = ?)`, p.PEID).
		Order("episodes.date").Find(&episodes)
	return episodes
}

func (tv *TV) episodeDepartment(dept string, p Person) []TVEpisode {
	var episodes []TVEpisode
	tv.db.Where(`episodes.e_id in (select e_id from episode_crew where department = ? and pe_id = ?)`,
		dept, p.PEID).
		Order("episodes.date").Find(&episodes)
	return episodes
}

func (tv *TV) EpisodeDirecting(p Person) []TVEpisode {
	return tv.episodeDepartment("Directing", p)
}

func (tv *TV) EpisodeProducing(p Person) []TVEpisode {
	return tv.episodeDepartment("Production", p)
}

func (tv *TV) EpisodeWriting(p Person) []TVEpisode {
	return tv.episodeDepartment("Writing", p)
}

func (tv *TV) AddedTVEpisodes() []TVEpisode {
	var episodes []TVEpisode
	tv.db.Where("episodes.last_modified >= ?", time.Now().Add(tv.config.TV.Recent*-1)).
		Order("episodes.last_modified desc").
		Limit(tv.config.TV.RecentLimit).
		Find(&episodes)
	return episodes
}

func (tv *TV) LookupETag(etag string) (TVEpisode, error) {
	episode := TVEpisode{ETag: etag}
	err := tv.db.First(&episode, &episode).Error
	return episode, err
}

func (tv *TV) SeriesCount() int64 {
	var count int64
	tv.db.Model(&TVSeries{}).Count(&count)
	return count
}

func (tv *TV) LastModified() time.Time {
	var episodes []TVEpisode
	tv.db.Order("last_modified desc").Limit(1).Find(&episodes)
	if len(episodes) == 1 {
		return episodes[0].LastModified
	} else {
		return time.Time{}
	}
}

func (tv *TV) episodesFor(keys []string) []TVEpisode {
	var episodes []TVEpisode
	tv.db.Where("key in (?)", keys).Find(&episodes)
	return episodes
}

func (tv *TV) createGenre(g *TVGenre) error {
	return tv.db.Create(g).Error
}

func (tv *TV) createKeyword(k *TVKeyword) error {
	return tv.db.Create(k).Error
}

func (tv *TV) createSeriesCast(c *TVSeriesCast) error {
	return tv.db.Create(c).Error
}

func (tv *TV) createSeriesCrew(c *TVSeriesCrew) error {
	return tv.db.Create(c).Error
}

func (tv *TV) createEpisodeCast(c *TVEpisodeCast) error {
	return tv.db.Create(c).Error
}

func (tv *TV) createEpisodeCrew(c *TVEpisodeCrew) error {
	return tv.db.Create(c).Error
}

// TODO
// func (tv *TV) createTVGuest(c *TVCast) error {
// 	return tv.db.Create(c).Error
// }

func (tv *TV) createPerson(p *Person) error {
	return people.CreatePerson(tv.db, p)
}

func (tv *TV) createSeries(s *TVSeries) error {
	return tv.db.Create(s).Error
}

func (tv *TV) updateSeries(s *TVSeries) error {
	return tv.db.Save(s).Error
}

func (tv *TV) createEpisode(e *TVEpisode) error {
	return tv.db.Create(e).Error
}

func (tv *TV) updateEpisode(e *TVEpisode) error {
	return tv.db.Save(e).Error
}
