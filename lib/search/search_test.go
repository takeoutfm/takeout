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

package search

import (
	"testing"
)

type TestSearch struct {
	t *testing.T
}

func (t TestSearch) Open(name string, keywords []string) error {
	return nil
}

func (t TestSearch) Index(m IndexMap) {
}

func (t TestSearch) Search(q string, limit int) ([]string, error) {
	return []string{}, nil
}

func (t TestSearch) Delete(keys []string) error {
	return nil
}

func (t TestSearch) Close() {
}

func NewTestSearch(t *testing.T) Searcher {
	return TestSearch{t: t}
}

func TestSearchIndex(t *testing.T) {
	c := Config{IndexDir: ""}
	s := NewSearcher(c)
	err := s.Open("", []string{})
	if err != nil {
		t.Fatal(err)
	}

	m := make(FieldMap)
	m["artist"] = "Gary Numan"
	m["release"] = "The Pleasure Principle"
	m["title"] = "Films"
	m["tags"] = []string{"pop rock", "new wave", "indie", "electronic"}
	m["instruments"] = []string{"guitar", "drums", "piano"}
	m["keyboard"] = "Gary Numan"
	m["piano"] = "Gary Numan"
	m["bass guitar"] = "Ade"
	m["drums/drum set"] = "Bill Smith"
	m["mix"] = "jim smith, joe blow"
	m["type"] = "music"

	index := make(IndexMap)
	index["1234"] = m

	s.Index(index)

	results, err := s.Search("numan", 100)
	if err != nil {
		t.Error(err)
	}
	if results[0] != "1234" {
		t.Error("expect result")
	}

	results, err = s.Search("radiohead", 100)
	if err != nil {
		t.Error(err)
	}
	if len(results) != 0 {
		t.Error("expect no results")
	}

	results, err = s.Search(`+tags:"pop rock" +tags:"indie"`, 100)
	if results[0] != "1234" {
		t.Error("expect tags result")
	}

	results, err = s.Search(`+title:"films"`, 100)
	if results[0] != "1234" {
		t.Error("expect title result")
	}

	results, err = s.Search(`+artist:"numan"`, 100)
	if results[0] != "1234" {
		t.Error("expect artist result")
	}

	results, err = s.Search(`+title:"films" +artist:numan +tags:"new wave" +piano:numan`, 100)
	if results[0] != "1234" {
		t.Error("expect mixed result")
	}
}
