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
	"net/url"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/people"
	"takeoutfm.dev/takeout/lib/bucket"
	"takeoutfm.dev/takeout/lib/search"
	"takeoutfm.dev/takeout/lib/tmdb"
	. "takeoutfm.dev/takeout/model"
)

var (
	ErrEpisodeNotFound = errors.New("episode not found")
	ErrInvalidEpisode  = errors.New("invalid episode")
	ErrSeriesNotFound  = errors.New("series not found")
	ErrRatingNotFound  = errors.New("rating not found")
)

type TV struct {
	config  *config.Config
	db      *gorm.DB
	tmdb    *tmdb.TMDB
	buckets []bucket.Bucket
}

func NewTV(config *config.Config) *TV {
	return &TV{
		config: config,
		tmdb:   tmdb.NewTMDB(config.TMDB.Config, config.NewGetter()),
	}
}

func (tv *TV) Open() (err error) {
	err = tv.openDB()
	if err == nil {
		tv.buckets, err = bucket.OpenMedia(tv.config.Buckets, config.MediaTV)
	}
	return
}

func (tv *TV) Close() {
	tv.closeDB()
}

func (tv *TV) FindSeries(identifier string) (TVSeries, error) {
	id, err := strconv.Atoi(identifier)
	if err != nil {
		if strings.HasPrefix(identifier, "tvid:") {
			id, err := strconv.Atoi(identifier[5:])
			if err != nil {
				return TVSeries{}, err
			}
			return tv.LookupTVID(id)
		}
		return TVSeries{}, ErrSeriesNotFound
	} else {
		return tv.LookupSeries(id)
	}
}

func (tv *TV) FindEpisode(identifier string) (TVEpisode, error) {
	id, err := strconv.Atoi(identifier)
	if err != nil {
		// season, episode, err := parseEpisode(identifier)
		// if err == nil {
		// }
		if strings.HasPrefix(identifier, "uuid:") {
			return tv.LookupUUID(identifier[5:])
		}
		return TVEpisode{}, ErrEpisodeNotFound
	} else {
		return tv.LookupEpisode(id)
	}
}

func (tv *TV) FindPerson(identifier string) (Person, error) {
	return people.FindPerson(tv.db, identifier)
}

func (tv *TV) newSearch() (search.Searcher, error) {
	keywords := []string{
		FieldGenre,
		FieldKeyword,
	}
	s := tv.config.NewSearcher()
	err := s.Open(tv.config.TV.SearchIndexName, keywords)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (tv *TV) Search(q string, limit ...int) []TVEpisode {
	s, err := tv.newSearch()
	if err != nil {
		return []TVEpisode{}
	}
	defer s.Close()

	l := tv.config.TV.SearchLimit
	if len(limit) == 1 {
		l = limit[0]
	}

	keys, err := s.Search(q, l)
	if err != nil {
		return nil
	}

	// split potentially large # of result keys into chunks to query
	chunkSize := 100
	var episodes []TVEpisode
	for i := 0; i < len(keys); i += chunkSize {
		end := i + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[i:end]
		episodes = append(episodes, tv.episodesFor(chunk)...)
	}

	return episodes
}

func (tv *TV) EpisodeURL(e TVEpisode) *url.URL {
	// FIXME assume first bucket!!!
	return tv.buckets[0].ObjectURL(e.Key)
}

func SeriesPoster(s TVSeries) string {
	if s.PosterPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Poster342, s.PosterPath}, "")
}

func SeriesPosterSmall(s TVSeries) string {
	if s.PosterPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Poster154, s.PosterPath}, "")
}

func SeriesBackdrop(s TVSeries) string {
	if s.BackdropPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Backdrop1280, s.BackdropPath}, "")
}

func EpisodeStillImage(e TVEpisode) string {
	if e.StillPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Still300, e.StillPath}, "")
}

func EpisodeStillSmall(e TVEpisode) string {
	if e.StillPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.Still185, e.StillPath}, "")
}

func EpisodeStillLarge(e TVEpisode) string {
	if e.StillPath == "" {
		return ""
	}
	return strings.Join([]string{"/img/tm/", tmdb.StillOriginal, e.StillPath}, "")
}

func (tv *TV) HasShows() bool {
	return tv.SeriesCount() > 0
}

func (tv *TV) TMDBSeriesPoster(s TVSeries) string {
	if s.PosterPath == "" {
		return ""
	}
	url := tv.tmdb.Poster(s.PosterPath, tmdb.Poster342)
	if url == nil {
		return ""
	}
	return url.String()
}

func (tv *TV) TMDBSeriesPosterSmall(s TVSeries) string {
	if s.PosterPath == "" {
		return ""
	}
	url := tv.tmdb.Poster(s.PosterPath, tmdb.Poster154)
	if url == nil {
		return ""
	}
	return url.String()
}

func (tv *TV) TMDBSeriesBackdrop(s TVSeries) string {
	if s.BackdropPath == "" {
		return ""
	}
	url := tv.tmdb.Backdrop(s.BackdropPath, tmdb.Backdrop1280)
	if url == nil {
		return ""
	}
	return url.String()
}

func (tv *TV) TMDBEpisodeStill(e TVEpisode) string {
	if e.StillPath == "" {
		return ""
	}
	url := tv.tmdb.Still(e.StillPath, tmdb.Still300)
	if url == nil {
		return ""
	}
	return url.String()
}

func (tv *TV) TMDBEpisodeStillSmall(e TVEpisode) string {
	if e.StillPath == "" {
		return ""
	}
	url := tv.tmdb.Still(e.StillPath, tmdb.Still185)
	if url == nil {
		return ""
	}
	return url.String()
}

func (tv *TV) TMDBPersonProfile(p Person) string {
	if p.ProfilePath == "" {
		return ""
	}
	url := tv.tmdb.PersonProfile(p.ProfilePath, tmdb.Profile185)
	if url == nil {
		return ""
	}
	return url.String()
}
