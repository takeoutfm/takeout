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

// Package film provides support for all movie media.
package film

import (
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/bucket"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/search"
	"takeoutfm.dev/takeout/lib/tmdb"
	. "takeoutfm.dev/takeout/model"
)

type Film struct {
	config  *config.Config
	db      *gorm.DB
	tmdb    *tmdb.TMDB
	buckets []bucket.Bucket
}

func NewFilm(config *config.Config) *Film {
	return &Film{
		config: config,
		tmdb:   tmdb.NewTMDB(config.TMDB.Config, config.NewGetter()),
	}
}

func (f *Film) Open() (err error) {
	err = f.openDB()
	if err == nil {
		f.buckets, err = bucket.OpenMedia(f.config.Buckets, config.MediaFilm)
	}
	return
}

func (f *Film) Close() {
	f.closeDB()
}

func (f *Film) FindMovie(identifier string) (Movie, error) {
	id, err := strconv.Atoi(identifier)
	if err != nil {
		if strings.HasPrefix(identifier, "uuid:") {
			return f.LookupUUID(identifier[5:])
		} else if strings.HasPrefix(identifier, "imid:") {
			return f.LookupIMID(identifier[5:])
		} else if strings.HasPrefix(identifier, "tmid:") {
			id, err := strconv.Atoi(identifier[5:])
			if err != nil {
				return Movie{}, err
			}
			return f.LookupTMID(id)
		} else {
			return f.LookupIMID(identifier)
		}
	} else {
		return f.LookupMovie(id)
	}
}

func (f *Film) FindMovies(identifiers []string) []Movie {
	return f.lookupIMIDs(identifiers)
}

func (f *Film) newSearch() (search.Searcher, error) {
	keywords := []string{
		FieldGenre,
		FieldKeyword,
	}
	s := f.config.NewSearcher()
	err := s.Open(f.config.Film.SearchIndexName, keywords)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (f *Film) Search(q string, limit ...int) []Movie {
	s, err := f.newSearch()
	if err != nil {
		return []Movie{}
	}
	defer s.Close()

	l := f.config.Film.SearchLimit
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
		movies = append(movies, f.moviesFor(chunk)...)
	}

	return movies
}

func (f *Film) MovieURL(m Movie) *url.URL {
	// FIXME assume first bucket!!!
	return f.buckets[0].ObjectURL(m.Key)
}

func MoviePoster(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Poster342, m.PosterPath}, "")
}

func (f *Film) TMDBMoviePoster(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	url := f.tmdb.Poster(m.PosterPath, tmdb.Poster342)
	if url == nil {
		return ""
	}
	return url.String()
}

func MoviePosterSmall(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Poster154, m.PosterPath}, "")
}

func (f *Film) TMDBMoviePosterSmall(m Movie) string {
	if m.PosterPath == "" {
		return ""
	}
	url := f.tmdb.Poster(m.PosterPath, tmdb.Poster154)
	if url == nil {
		return ""
	}
	return url.String()
}

func MovieBackdrop(m Movie) string {
	if m.BackdropPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Backdrop1280, m.BackdropPath}, "")
}

func (f *Film) TMDBMovieBackdrop(m Movie) string {
	if m.BackdropPath == "" {
		return ""
	}
	url := f.tmdb.Backdrop(m.BackdropPath, tmdb.Backdrop1280)
	if url == nil {
		return ""
	}
	return url.String()
}

// func (f *Film) PersonProfile(p Person) string {
// 	if p.ProfilePath == "" {
// 		return ""
// 	}
// 	url := fmt.Sprintf("/img/tm/%s%s", tmdb.Profile185, p.ProfilePath)
// 	return url
// }

func (f *Film) TMDBPersonProfile(p Person) string {
	if p.ProfilePath == "" {
		return ""
	}
	url := f.tmdb.PersonProfile(p.ProfilePath, tmdb.Profile185)
	if url == nil {
		return ""
	}
	return url.String()
}

func (f *Film) HasMovies() bool {
	return f.MovieCount() > 0
}

func (f *Film) Recommend() []Recommend {
	var recommend []Recommend
	for _, r := range f.config.Film.Recommend.When {
		if date.Match(r.Layout, r.Match) {
			movies := f.Search(r.Query)
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
