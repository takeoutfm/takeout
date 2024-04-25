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

// Package video provides support for all movie media.
package video

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/lib/bucket"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/lib/search"
	"github.com/takeoutfm/takeout/lib/tmdb"
	. "github.com/takeoutfm/takeout/model"
	"gorm.io/gorm"
)

type Video struct {
	config  *config.Config
	db      *gorm.DB
	tmdb    *tmdb.TMDB
	buckets []bucket.Bucket
}

func NewVideo(config *config.Config) *Video {
	return &Video{
		config: config,
		tmdb:   tmdb.NewTMDB(config.TMDB.Config, config.NewGetter()),
	}
}

func (v *Video) Open() (err error) {
	err = v.openDB()
	if err == nil {
		v.buckets, err = bucket.OpenMedia(v.config.Buckets, config.MediaVideo)
	}
	return
}

func (v *Video) Close() {
	v.closeDB()
}

func (v *Video) FindMovie(identifier string) (Movie, error) {
	id, err := strconv.Atoi(identifier)
	if err != nil {
		if strings.HasPrefix(identifier, "uuid:") {
			return v.LookupUUID(identifier[5:])
		} else if strings.HasPrefix(identifier, "imid:") {
			return v.LookupIMID(identifier[5:])
		} else if strings.HasPrefix(identifier, "tmid:") {
			id, err := strconv.Atoi(identifier[5:])
			if err != nil {
				return Movie{}, err
			}
			return v.LookupTMID(id)
		} else {
			return v.LookupIMID(identifier)
		}
	} else {
		return v.LookupMovie(id)
	}
}

func (v *Video) FindMovies(identifiers []string) []Movie {
	return v.lookupIMIDs(identifiers)
}

func (v *Video) newSearch() (search.Searcher, error) {
	keywords := []string{
		FieldGenre,
		FieldKeyword,
	}
	s := v.config.NewSearcher()
	err := s.Open(v.config.Video.SearchIndexName, keywords)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (v *Video) Search(q string, limit ...int) []Movie {
	s, err := v.newSearch()
	if err != nil {
		return []Movie{}
	}
	defer s.Close()

	l := v.config.Video.SearchLimit
	if len(limit) == 1 {
		l = limit[0]
	}

	keys, err := s.Search(q, l)
	if err != nil {
		return nil
	}

	// split potentially large # of result keys into chunks to query
	chunkSize := 100
	var movies []Movie
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[i:end]
		movies = append(movies, v.moviesFor(chunk)...)
	}

	return movies
}

func (v *Video) MovieURL(m Movie) *url.URL {
	// FIXME assume first bucket!!!
	return v.buckets[0].ObjectURL(m.Key)
}

func (v *Video) MoviePoster(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	url := fmt.Sprintf("/img/tm/%s%s", tmdb.Poster342, m.PosterPath)
	return url
}

func (v *Video) TMDBMoviePoster(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	url := v.tmdb.Poster(m.PosterPath, tmdb.Poster342)
	if url == nil {
		return ""
	}
	return url.String()
}

func (v *Video) MoviePosterSmall(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	url := fmt.Sprintf("/img/tm/%s%s", tmdb.Poster154, m.PosterPath)
	return url
}

func (v *Video) TMDBMoviePosterSmall(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	url := v.tmdb.Poster(m.PosterPath, tmdb.Poster154)
	if url == nil {
		return ""
	}
	return url.String()
}

func (v *Video) MovieBackdrop(m Movie) string {
	if m.BackdropPath == "" {
		return ""
	}
	url := fmt.Sprintf("/img/tm/%s%s", tmdb.Backdrop1280, m.BackdropPath)
	return url
}

func (v *Video) TMDBMovieBackdrop(m Movie) string {
	if m.BackdropPath == "" {
		return ""
	}
	url := v.tmdb.Backdrop(m.BackdropPath, tmdb.Backdrop1280)
	if url == nil {
		return ""
	}
	return url.String()
}

func (v *Video) PersonProfile(p Person) string {
	if p.ProfilePath == "" {
		return ""
	}
	url := fmt.Sprintf("/img/tm/%s%s", tmdb.Profile185, p.ProfilePath)
	return url
}

func (v *Video) TMDBPersonProfile(p Person) string {
	if p.ProfilePath == "" {
		return ""
	}
	url := v.tmdb.PersonProfile(p.ProfilePath, tmdb.Profile185)
	if url == nil {
		return ""
	}
	return url.String()
}

func (v *Video) HasMovies() bool {
	return v.MovieCount() > 0
}

func (v *Video) Recommend() []Recommend {
	var recommend []Recommend
	for _, r := range v.config.Video.Recommend.When {
		if date.Match(r.Layout, r.Match) {
			movies := v.Search(r.Query)
			if len(movies) > 0 {
				recommend = append(recommend, Recommend{
					Name:   r.Name,
					Movies: movies,
				})
			}
		}
	}
	return recommend
}
