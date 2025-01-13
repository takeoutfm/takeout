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

package film

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/lib/str"
	"github.com/takeoutfm/takeout/model"
)

func makeFilm(t *testing.T) *Film {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	f := NewFilm(config)
	err = f.Open()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCast(t *testing.T) {
	f := makeFilm(t)

	c := model.Cast{
		TMID:      1,
		PEID:      1,
		Character: "test character",
		Rank:      1,
	}

	err := f.createCast(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Error("expect ID")
	}

	f.deleteCast(int(c.TMID))
}

func TestCrew(t *testing.T) {
	f := makeFilm(t)

	c := model.Crew{
		TMID:       100,
		PEID:       100,
		Department: "test department",
		Job:        "test job",
	}

	err := f.createCrew(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Error("expect ID")
	}

	f.deleteCrew(int(c.TMID))
}

func TestCollection(t *testing.T) {
	f := makeFilm(t)

	c := model.Collection{
		TMID:     100,
		Name:     "test name",
		SortName: "test name, the",
	}

	err := f.createCollection(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Error("expect ID")
	}

	_, err = f.LookupCollectionName("test name")
	if err != nil {
		t.Error("expect lookup collection name")
	}

	if len(f.Collections()) != 1 {
		t.Error("expect collections")
	}

	f.deleteCollections(int(c.TMID))

	if len(f.Collections()) != 0 {
		t.Error("expect no collections")
	}
}

func TestGenre(t *testing.T) {
	f := makeFilm(t)

	g := model.Genre{
		TMID: 100,
		Name: "test name",
	}

	err := f.createGenre(&g)
	if err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 {
		t.Error("expect ID")
	}

	f.deleteGenres(100)
}

func TestKeyword(t *testing.T) {
	f := makeFilm(t)

	k := model.Keyword{
		TMID: 100,
		Name: "test name",
	}

	err := f.createKeyword(&k)
	if err != nil {
		t.Fatal(err)
	}
	if k.ID == 0 {
		t.Error("expect ID")
	}

	f.deleteKeywords(100)
}

func TestMovie(t *testing.T) {
	f := makeFilm(t)

	m := model.Movie{
		TMID:             100,
		IMID:             "IM200",
		Title:            "test title",
		Date:             time.Now(),
		Rating:           "NR",
		Tagline:          "test tagline",
		OriginalTitle:    "test orig title",
		OriginalLanguage: "en",
		Overview:         "test overview",
		Budget:           999999,
		Revenue:          9999999,
		Runtime:          999,
		VoteAverage:      0.7,
		VoteCount:        999,
		BackdropPath:     "",
		PosterPath:       "",
		SortTitle:        "test sort title",
		Key:              "test key",
		Size:             99999999,
		ETag:             "test etag",
		LastModified:     time.Now(),
	}

	err := f.createMovie(&m)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == 0 {
		t.Error("expect ID")
	}
	if m.UUID == "" {
		t.Error("expect UUID")
	}

	_, err = f.LookupMovie(int(m.ID))
	if err != nil {
		t.Error("expect to find movie by id")
	}
	_, err = f.LookupETag(m.ETag)
	if err != nil {
		t.Error("expect to find movie by etag")
	}
	_, err = f.FindMovie("uuid:" + m.UUID)
	if err != nil {
		t.Error("expect to find movie by uuid")
	}
	_, err = f.FindMovie("imid:" + m.IMID)
	if err != nil {
		t.Error("expect to find movie by imid")
	}
	_, err = f.LookupIMID(m.IMID)
	if err != nil {
		t.Error("expect to lookup movie by imid")
	}
	_, err = f.FindMovie("tmid:" + str.Itoa(int(m.TMID)))
	if err != nil {
		t.Error("expect to find movie by tmid")
	}
	_, err = f.LookupTMID(int(m.TMID))
	if err != nil {
		t.Error("expect to lookup movie by tmid")
	}

	m.Title = "new movie title"
	f.UpdateMovie(&m)

	mm, err := f.LookupMovie(int(m.ID))
	if err != nil {
		t.Error("expect to find movie by id")
	}
	if mm.Title != "new movie title" {
		t.Error("expect updated title")
	}

	f.deleteMovie(100)

	_, err = f.LookupMovie(int(m.ID))
	if err == nil {
		t.Error("expect not to find movie by id")
	}
}

func TestPerson(t *testing.T) {
	f := makeFilm(t)

	peid := int64(9999)
	p := model.Person{
		PEID:        peid,
		IMID:        "IM1234",
		Name:        "test person",
		ProfilePath: "test path",
		Bio:         "test bio",
		Birthplace:  "test birthplace",
		Birthday:    time.Now().Add(-99999 * time.Hour),
		Deathday:    time.Time{},
	}

	err := f.createPerson(&p)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Error("expect ID")
	}

	// _, err = f.LookupPerson(int(p.ID))
	// if err != nil {
	// 	t.Error("expect to find person by id")
	// }

	_, err = f.Person(int(peid))
	if err != nil {
		t.Error("expect to find person by peid")
	}
}
