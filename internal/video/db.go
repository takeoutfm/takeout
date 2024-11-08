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

package video

import (
	"errors"
	"time"

	. "github.com/takeoutfm/takeout/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func (v *Video) openDB() (err error) {
	cfg := v.config.Music.DB.GormConfig()

	switch v.config.Music.DB.Driver {
	case "sqlite3":
		v.db, err = gorm.Open(sqlite.Open(v.config.Video.DB.Source), cfg)
	case "mysql":
		v.db, err = gorm.Open(mysql.Open(v.config.Video.DB.Source), cfg)
	case "postgres":
		// postgres untested
		v.db, err = gorm.Open(postgres.Open(v.config.Video.DB.Source), cfg)
	default:
		err = errors.New("driver not supported")
	}

	if err != nil {
		return
	}

	v.db.AutoMigrate(&Cast{}, &Collection{}, &Crew{}, &Genre{}, &Keyword{}, &Movie{}, &Person{})
	return
}

func (v *Video) closeDB() {
	conn, err := v.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

func (v *Video) Movies() []Movie {
	var movies []Movie
	v.db.Order("sort_title").Find(&movies)
	return movies
}

func (v *Video) Genre(name string) []Movie {
	var movies []Movie
	v.db.Where("movies.tm_id in (select tm_id from genres where name = ?)", name).
		Order("movies.date").Find(&movies)
	return movies
}

func (v *Video) Genres(m Movie) []string {
	var genres []Genre
	var list []string
	v.db.Where("tm_id = ?", m.TMID).Order("name").Find(&genres)
	for _, g := range genres {
		list = append(list, g.Name)
	}
	return list
}

func (v *Video) Keyword(name string) []Movie {
	var movies []Movie
	v.db.Where("movies.tm_id in (select tm_id from keywords where name = ?)", name).
		Order("movies.date").Find(&movies)
	return movies
}

func (v *Video) Keywords(m Movie) []string {
	var keywords []Keyword
	var list []string
	v.db.Where("tm_id = ?", m.TMID).Order("name").Find(&keywords)
	for _, g := range keywords {
		list = append(list, g.Name)
	}
	return list
}

func (v *Video) Collections() []Collection {
	var collections []Collection
	v.db.Group("name").Order("sort_name").Find(&collections)
	return collections
}

func (v *Video) MovieCollections(m Movie) []Collection {
	var collections []Collection
	v.db.Where("tm_id = ?", m.TMID).Find(&collections)
	return collections
}

func (v *Video) CollectionMovies(c Collection) []Movie {
	var movies []Movie
	v.db.Where("movies.tm_id in (select tm_id from collections where name = ?)", c.Name).
		Order("movies.date").Find(&movies)
	return movies
}

func (v *Video) Cast(m Movie) []Cast {
	var cast []Cast
	var people []Person
	v.db.Order("rank asc").
		Joins(`inner join movies on "cast".tm_id = movies.tm_id`).
		Where("movies.tm_id = ?", m.TMID).Find(&cast)
	v.db.Joins(`inner join "cast" on people.pe_id = "cast".pe_id`).
		Joins(`inner join movies on movies.tm_id = "cast".tm_id`).
		Where("movies.tm_id = ?", m.TMID).Find(&people)
	pmap := make(map[int64]Person)
	for _, p := range people {
		pmap[p.PEID] = p
	}
	for i := range cast {
		cast[i].Person = pmap[cast[i].PEID]
	}
	return cast
}

func (v *Video) Crew(m Movie) []Crew {
	var crew []Crew
	var people []Person
	v.db.Joins(`inner join movies on "crew".tm_id = movies.tm_id`).
		Where("movies.tm_id = ?", m.TMID).Find(&crew)
	v.db.Joins(`inner join "crew" on people.pe_id = "crew".pe_id`).
		Joins(`inner join movies on movies.tm_id = "crew".tm_id`).
		Where("movies.tm_id = ?", m.TMID).Find(&people)
	pmap := make(map[int64]Person)
	for _, p := range people {
		pmap[p.PEID] = p
	}
	for i := range crew {
		crew[i].Person = pmap[crew[i].PEID]
	}
	return crew
}

func (v *Video) deleteMovie(tmid int) {
	var list []Movie
	v.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteTVShow(tvid int) {
	var list []TVShow
	v.db.Where("tv_id = ?", tvid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteEpisode(epid int) {
	var list []Episode
	v.db.Where("ep_id = ?", epid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteCast(tmid int) {
	var list []Cast
	v.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteCollections(tmid int) {
	var list []Collection
	v.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteCrew(tmid int) {
	var list []Crew
	v.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteGenres(tmid int) {
	var list []Genre
	v.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) deleteKeywords(tmid int) {
	var list []Keyword
	v.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		v.db.Unscoped().Delete(o)
	}
}

func (v *Video) Person(peid int) (Person, error) {
	var person Person
	// TODO fix this logs an error every time and it's not an error
	err := v.db.Where("pe_id = ?", peid).First(&person).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Person{}, errors.New("person not found")
	}
	return person, err
}

func (v *Video) UpdateMovie(m *Movie) error {
	return v.db.Save(m).Error
}

func (v *Video) LookupCollectionName(name string) (Collection, error) {
	var collection Collection
	err := v.db.First(&collection, "name = ?", name).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Collection{}, errors.New("collection not found")
	}
	return collection, err
}

func (v *Video) LookupMovie(id int) (Movie, error) {
	var movie Movie
	err := v.db.First(&movie, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (v *Video) LookupTMID(tmid int) (Movie, error) {
	var movie Movie
	err := v.db.First(&movie, "tm_id = ?", tmid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (v *Video) LookupIMID(imid string) (Movie, error) {
	var movie Movie
	err := v.db.First(&movie, "im_id = ?", imid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (v *Video) LookupUUID(uuid string) (Movie, error) {
	var movie Movie
	err := v.db.First(&movie, "uuid = ?", uuid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (v *Video) lookupIMIDs(imids []string) []Movie {
	var movies []Movie
	v.db.Where("im_id in (?)", imids).Find(&movies)
	return movies
}

func (v *Video) LookupPerson(id int) (Person, error) {
	var person Person
	err := v.db.First(&person, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Person{}, errors.New("person not found")
	}
	return person, err
}

func (v *Video) Starring(p Person) []Movie {
	var movies []Movie
	v.db.Where(`movies.tm_id in (select tm_id from "cast" where pe_id = ?)`, p.PEID).
		Order("movies.date").Find(&movies)
	return movies
}

func (v *Video) department(dept string, p Person) []Movie {
	var movies []Movie
	v.db.Where(`movies.tm_id in (select tm_id from "crew" where department = ? and pe_id = ?)`,
		dept, p.PEID).
		Order("movies.date").Find(&movies)
	return movies
}

func (v *Video) Directing(p Person) []Movie {
	return v.department("Directing", p)
}

func (v *Video) Producing(p Person) []Movie {
	return v.department("Production", p)
}

func (v *Video) Writing(p Person) []Movie {
	return v.department("Writing", p)
}

func (v *Video) moviesFor(keys []string) []Movie {
	var movies []Movie
	v.db.Where("key in (?)", keys).Find(&movies)
	return movies
}

func (v *Video) RecentlyAdded() []Movie {
	var movies []Movie
	v.db.Where("movies.last_modified >= ?", time.Now().Add(v.config.Video.Recent*-1)).
		Order("movies.last_modified desc, sort_title").
		Limit(v.config.Video.RecentLimit).
		Find(&movies)
	return movies
}

func (v *Video) RecentlyReleased() []Movie {
	var movies []Movie
	v.db.Where("movies.date >= ?", time.Now().Add(v.config.Video.Recent*-1)).
		Order("movies.date desc, sort_title").
		Limit(v.config.Music.RecentLimit).
		Find(&movies)
	return movies
}

func (v *Video) LookupETag(etag string) (Movie, error) {
	movie := Movie{ETag: etag}
	err := v.db.First(&movie, &movie).Error
	return movie, err
}

func (v *Video) MovieCount() int64 {
	var count int64
	v.db.Model(&Movie{}).Count(&count)
	return count
}

func (v *Video) LastModified() time.Time {
	var movies []Movie
	v.db.Order("last_modified desc").Limit(1).Find(&movies)
	if len(movies) == 1 {
		return movies[0].LastModified
	} else {
		return time.Time{}
	}
}

func (v *Video) createCast(c *Cast) error {
	return v.db.Create(c).Error
}

func (v *Video) createCollection(c *Collection) error {
	return v.db.Create(c).Error
}

func (v *Video) createCrew(c *Crew) error {
	return v.db.Create(c).Error
}

func (v *Video) createGenre(g *Genre) error {
	return v.db.Create(g).Error
}

func (v *Video) createKeyword(k *Keyword) error {
	return v.db.Create(k).Error
}

func (v *Video) createMovie(m *Movie) error {
	return v.db.Create(m).Error
}

func (v *Video) createPerson(p *Person) error {
	return v.db.Create(p).Error
}

func (v *Video) createTVShow(tv *TVShow) error {
	return v.db.Create(tv).Error
}

func (v *Video) createEpisode(episode *Episode) error {
	return v.db.Create(episode).Error
}
