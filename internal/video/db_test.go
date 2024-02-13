// Copyright 2024 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package video

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/lib/str"
	"github.com/takeoutfm/takeout/model"
)

func makeVideo(t *testing.T) *Video {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	v := NewVideo(config)
	err = v.Open()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCast(t *testing.T) {
	v := makeVideo(t)

	c := model.Cast{
		TMID:      1,
		PEID:      1,
		Character: "test character",
		Rank:      1,
	}

	err := v.createCast(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Error("expect ID")
	}

	v.deleteCast(int(c.TMID))
}

func TestCrew(t *testing.T) {
	v := makeVideo(t)

	c := model.Crew{
		TMID:       100,
		PEID:       100,
		Department: "test department",
		Job:        "test job",
	}

	err := v.createCrew(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Error("expect ID")
	}

	v.deleteCrew(int(c.TMID))
}

func TestCollection(t *testing.T) {
	v := makeVideo(t)

	c := model.Collection{
		TMID:     100,
		Name:     "test name",
		SortName: "test name, the",
	}

	err := v.createCollection(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Error("expect ID")
	}

	_, err = v.LookupCollectionName("test name")
	if err != nil {
		t.Error("expect lookup collection name")
	}

	if len(v.Collections()) != 1 {
		t.Error("expect collections")
	}

	v.deleteCollections(int(c.TMID))

	if len(v.Collections()) != 0 {
		t.Error("expect no collections")
	}
}

func TestGenre(t *testing.T) {
	v := makeVideo(t)

	g := model.Genre{
		TMID: 100,
		Name: "test name",
	}

	err := v.createGenre(&g)
	if err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 {
		t.Error("expect ID")
	}

	v.deleteGenres(100)
}

func TestKeyword(t *testing.T) {
	v := makeVideo(t)

	k := model.Keyword{
		TMID: 100,
		Name: "test name",
	}

	err := v.createKeyword(&k)
	if err != nil {
		t.Fatal(err)
	}
	if k.ID == 0 {
		t.Error("expect ID")
	}

	v.deleteKeywords(100)
}

func TestMovie(t *testing.T) {
	v := makeVideo(t)

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

	err := v.createMovie(&m)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == 0 {
		t.Error("expect ID")
	}
	if m.UUID == "" {
		t.Error("expect UUID")
	}

	_, err = v.LookupMovie(int(m.ID))
	if err != nil {
		t.Error("expect to find movie by id")
	}
	_, err = v.LookupETag(m.ETag)
	if err != nil {
		t.Error("expect to find movie by etag")
	}
	_, err = v.FindMovie("uuid:" + m.UUID)
	if err != nil {
		t.Error("expect to find movie by uuid")
	}
	_, err = v.FindMovie("imid:" + m.IMID)
	if err != nil {
		t.Error("expect to find movie by imid")
	}
	_, err = v.LookupIMID(m.IMID)
	if err != nil {
		t.Error("expect to lookup movie by imid")
	}
	_, err = v.FindMovie("tmid:" + str.Itoa(int(m.TMID)))
	if err != nil {
		t.Error("expect to find movie by tmid")
	}
	_, err = v.LookupTMID(int(m.TMID))
	if err != nil {
		t.Error("expect to lookup movie by tmid")
	}

	m.Title = "new movie title"
	v.UpdateMovie(&m)

	mm, err := v.LookupMovie(int(m.ID))
	if err != nil {
		t.Error("expect to find movie by id")
	}
	if mm.Title != "new movie title" {
		t.Error("expect updated title")
	}

	v.deleteMovie(100)

	_, err = v.LookupMovie(int(m.ID))
	if err == nil {
		t.Error("expect not to find movie by id")
	}
}

func TestPerson(t *testing.T) {
	v := makeVideo(t)

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

	err := v.createPerson(&p)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == 0 {
		t.Error("expect ID")
	}

	_, err = v.LookupPerson(int(p.ID))
	if err != nil {
		t.Error("expect to find person by id")
	}

	_, err = v.Person(int(peid))
	if err != nil {
		t.Error("expect to find person by peid")
	}
}
