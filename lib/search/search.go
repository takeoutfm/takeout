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

// Package search provides a wrapper for bleve search, building a search
// database for an index of fields.
package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"path/filepath"
	"strings"
)

type FieldMap map[string]interface{}
type IndexMap map[string]FieldMap

type Config struct {
	IndexDir string
}

type Searcher interface {
	Open(name string, keywords []string) error
	Index(m IndexMap)
	Search(q string, limit int) ([]string, error)
	Delete(keys []string) error
	Close()
}

type search struct {
	config Config
	index  bleve.Index
}

func NewSearcher(config Config) Searcher {
	return &search{config: config}
}

func (s *search) Open(name string, keywords []string) error {
	mapping := bleve.NewIndexMapping()
	// Note that keywords are fields where we want only exact matches.
	// see https://blevesearch.com/docs/Analyzers/
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name
	keywordMapping := bleve.NewDocumentMapping()
	for _, v := range keywords {
		keywordMapping.AddFieldMappingsAt(v, keywordFieldMapping)
	}
	mapping.AddDocumentMapping("_default", keywordMapping)

	path := "" // in memory
	if s.config.IndexDir != "" && name != "" {
		path = filepath.Join(s.config.IndexDir, name+".bleve")
	} else if name != "" {
		path = name
	}

	index, err := bleve.New(path, mapping)
	if err == bleve.ErrorIndexPathExists {
		index, err = bleve.Open(path)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	s.index = index

	return nil
}

func (s *search) Close() {
	if s.index != nil {
		s.index.Close()
		s.index = nil
	}
}

// see https://blevesearch.com/docs/Query-String-Query/
func (s *search) Search(q string, limit int) ([]string, error) {
	query := bleve.NewQueryStringQuery(q)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = limit
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, hit := range searchResult.Hits {
		keys = append(keys, hit.ID)
	}
	return keys, nil
}

func (s *search) Index(m IndexMap) {
	for k, v := range m {
		s.index.Index(k, v)
	}
}

func (s *search) Delete(keys []string) error {
	b := s.index.NewBatch()
	for _, key := range keys {
		b.Delete(key)
	}
	return s.index.Batch(b)
}

func CloneFields(fields FieldMap) FieldMap {
	target := make(FieldMap)
	for k, v := range fields {
		target[k] = v
	}
	return target
}

func AddField(fields FieldMap, key string, value interface{}) FieldMap {
	key = strings.ToLower(key)
	keys := []string{key}
	for _, k := range keys {
		k := strings.Replace(k, " ", "_", -1)
		switch value.(type) {
		case string:
			svalue := value.(string)
			//svalue = fixName(svalue)
			if v, ok := fields[k]; ok {
				switch v.(type) {
				case string:
					// string becomes array of 2 strings
					fields[k] = []string{v.(string), svalue}
				case []string:
					// array of 3+ strings
					s := v.([]string)
					s = append(s, svalue)
					fields[k] = s
				default:
					panic("bad field types")
				}
			} else {
				// single string
				fields[k] = svalue
			}
		default:
			// numeric, date, etc.
			fields[k] = value
		}
	}
	return fields
}
