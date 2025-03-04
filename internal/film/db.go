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

package film

import (
	"errors"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/people"
	. "takeoutfm.dev/takeout/model"
)

func (f *Film) openDB() (err error) {
	cfg := f.config.Film.DB.GormConfig()

	switch f.config.Film.DB.Driver {
	case "sqlite3":
		f.db, err = gorm.Open(sqlite.Open(f.config.Film.DB.Source), cfg)
	case "mysql":
		f.db, err = gorm.Open(mysql.Open(f.config.Film.DB.Source), cfg)
	case "postgres":
		// postgres untested
		f.db, err = gorm.Open(postgres.Open(f.config.Film.DB.Source), cfg)
	default:
		err = errors.New("driver not supported")
	}

	if err != nil {
		return
	}

	f.db.AutoMigrate(&Cast{}, &Collection{}, &Crew{}, &Genre{}, &Keyword{}, &Movie{}, &Person{}, &Trailer{})
	return
}

func (f *Film) closeDB() {
	conn, err := f.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

func (f *Film) Movies() []Movie {
	var movies []Movie
	f.db.Order("sort_title").Find(&movies)
	return movies
}

func (f *Film) Genre(name string) []Movie {
	var movies []Movie
	f.db.Where("movies.tm_id in (select tm_id from genres where name = ?)", name).
		Order("movies.date").Find(&movies)
	return movies
}

func (f *Film) Genres(m Movie) []string {
	var genres []Genre
	var list []string
	f.db.Where("tm_id = ?", m.TMID).Order("name").Find(&genres)
	for _, g := range genres {
		list = append(list, g.Name)
	}
	return list
}

func (f *Film) Keyword(name string) []Movie {
	var movies []Movie
	f.db.Where("movies.tm_id in (select tm_id from keywords where name = ?)", name).
		Order("movies.date").Find(&movies)
	return movies
}

func (f *Film) Keywords(m Movie) []string {
	var keywords []Keyword
	var list []string
	f.db.Where("tm_id = ?", m.TMID).Order("name").Find(&keywords)
	for _, g := range keywords {
		list = append(list, g.Name)
	}
	return list
}

func (f *Film) Collections() []Collection {
	var collections []Collection
	f.db.Group("name").Order("sort_name").Find(&collections)
	return collections
}

func (f *Film) MovieCollections(m Movie) []Collection {
	var collections []Collection
	f.db.Where("tm_id = ?", m.TMID).Find(&collections)
	return collections
}

func (f *Film) CollectionMovies(c Collection) []Movie {
	var movies []Movie
	f.db.Where("movies.tm_id in (select tm_id from collections where name = ?)", c.Name).
		Order("movies.date").Find(&movies)
	return movies
}

func (f *Film) MovieTrailers(m Movie) []Trailer {
	var trailers []Trailer
	f.db.Where("tm_id = ?", m.TMID).Find(&trailers)
	return trailers
}

func (f *Film) Cast(m Movie) []Cast {
	var cast []Cast
	var people []Person
	f.db.Order("rank asc").
		Joins(`inner join movies on "cast".tm_id = movies.tm_id`).
		Where("movies.tm_id = ?", m.TMID).Find(&cast)
	f.db.Joins(`inner join "cast" on people.pe_id = "cast".pe_id`).
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

func (f *Film) Crew(m Movie) []Crew {
	var crew []Crew
	var people []Person
	f.db.Joins(`inner join movies on "crew".tm_id = movies.tm_id`).
		Where("movies.tm_id = ?", m.TMID).Find(&crew)
	f.db.Joins(`inner join "crew" on people.pe_id = "crew".pe_id`).
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

func (f *Film) deleteMovie(tmid int) {
	var list []Movie
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) deleteCast(tmid int) {
	var list []Cast
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) deleteCollections(tmid int) {
	var list []Collection
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) deleteTrailers(tmid int) {
	var list []Trailer
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) deleteCrew(tmid int) {
	var list []Crew
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) deleteGenres(tmid int) {
	var list []Genre
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) deleteKeywords(tmid int) {
	var list []Keyword
	f.db.Where("tm_id = ?", tmid).Find(&list)
	for _, o := range list {
		f.db.Unscoped().Delete(o)
	}
}

func (f *Film) Person(peid int) (Person, error) {
	return people.Person(f.db, peid)
}

func (f *Film) UpdateMovie(m *Movie) error {
	return f.db.Save(m).Error
}

func (f *Film) LookupCollectionName(name string) (Collection, error) {
	var collection Collection
	err := f.db.First(&collection, "name = ?", name).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Collection{}, errors.New("collection not found")
	}
	return collection, err
}

func (f *Film) LookupMovie(id int) (Movie, error) {
	var movie Movie
	err := f.db.First(&movie, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (f *Film) LookupTMID(tmid int) (Movie, error) {
	var movie Movie
	err := f.db.First(&movie, "tm_id = ?", tmid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (f *Film) LookupIMID(imid string) (Movie, error) {
	var movie Movie
	err := f.db.First(&movie, "im_id = ?", imid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (f *Film) LookupUUID(uuid string) (Movie, error) {
	var movie Movie
	err := f.db.First(&movie, "uuid = ?", uuid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Movie{}, errors.New("movie not found")
	}
	return movie, err
}

func (f *Film) lookupIMIDs(imids []string) []Movie {
	var movies []Movie
	f.db.Where("im_id in (?)", imids).Find(&movies)
	return movies
}

func (f *Film) Starring(p Person) []Movie {
	var movies []Movie
	f.db.Where(`movies.tm_id in (select tm_id from "cast" where pe_id = ?)`, p.PEID).
		Order("movies.date").Find(&movies)
	return movies
}

func (f *Film) department(dept string, p Person) []Movie {
	var movies []Movie
	f.db.Where(`movies.tm_id in (select tm_id from "crew" where department = ? and pe_id = ?)`,
		dept, p.PEID).
		Order("movies.date").Find(&movies)
	return movies
}

func (f *Film) Directing(p Person) []Movie {
	return f.department("Directing", p)
}

func (f *Film) Producing(p Person) []Movie {
	return f.department("Production", p)
}

func (f *Film) Writing(p Person) []Movie {
	return f.department("Writing", p)
}

func (f *Film) moviesFor(keys []string) []Movie {
	var movies []Movie
	f.db.Where("key in (?)", keys).Find(&movies)
	return movies
}

func (f *Film) RecentlyAdded() []Movie {
	var movies []Movie
	f.db.Where("movies.last_modified >= ?", time.Now().Add(f.config.Film.Recent*-1)).
		Order("movies.last_modified desc, sort_title").
		Limit(f.config.Film.RecentLimit).
		Find(&movies)
	return movies
}

func (f *Film) RecentlyReleased() []Movie {
	var movies []Movie
	f.db.Where("movies.date >= ?", time.Now().Add(f.config.Film.Recent*-1)).
		Order("movies.date desc, sort_title").
		Limit(f.config.Film.RecentLimit).
		Find(&movies)
	return movies
}

func (f *Film) LookupETag(etag string) (Movie, error) {
	movie := Movie{ETag: etag}
	err := f.db.First(&movie, &movie).Error
	return movie, err
}

func (f *Film) MovieCount() int64 {
	var count int64
	f.db.Model(&Movie{}).Count(&count)
	return count
}

func (f *Film) LastModified() time.Time {
	var movies []Movie
	f.db.Order("last_modified desc").Limit(1).Find(&movies)
	if len(movies) == 1 {
		return movies[0].LastModified
	} else {
		return time.Time{}
	}
}

func (f *Film) createCast(c *Cast) error {
	return f.db.Create(c).Error
}

func (f *Film) createCollection(c *Collection) error {
	return f.db.Create(c).Error
}

func (f *Film) createCrew(c *Crew) error {
	return f.db.Create(c).Error
}

func (f *Film) createGenre(g *Genre) error {
	return f.db.Create(g).Error
}

func (f *Film) createKeyword(k *Keyword) error {
	return f.db.Create(k).Error
}

func (f *Film) createMovie(m *Movie) error {
	return f.db.Create(m).Error
}

func (f *Film) createPerson(p *Person) error {
	return people.CreatePerson(f.db, p)
}

func (f *Film) createTrailer(t *Trailer) error {
	return f.db.Create(t).Error
}
