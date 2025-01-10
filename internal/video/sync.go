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
	"regexp"
	"strings"
	"time"

	"github.com/takeoutfm/takeout/lib/bucket"
	"github.com/takeoutfm/takeout/lib/client"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/lib/log"
	"github.com/takeoutfm/takeout/lib/search"
	"github.com/takeoutfm/takeout/lib/str"
	"github.com/takeoutfm/takeout/lib/tmdb"
	"github.com/takeoutfm/takeout/internal/people"
	. "github.com/takeoutfm/takeout/model"
)

const (
	FieldBudget     = "budget"
	FieldCast       = "cast"
	FieldCharacter  = "character"
	FieldCollection = "collection"
	FieldCrew       = "crew"
	FieldDate       = "date"
	FieldGenre      = "genre"
	FieldKeyword    = "keyword"
	FieldName       = "name"
	FieldRating     = "rating"
	FieldRevenue    = "revenue"
	FieldRuntime    = "runtime"
	FieldTagline    = "tagline"
	FieldTitle      = "title"
	FieldVote       = "vote"
	FieldVoteCount  = "vote_count"

	// JobDirector   = "Director"
	// JobNovel      = "Novel"
	// JobScreenplay = "Screenplay"
	// JobStory      = "Story"

	PreferLargest  = "largest"
	PreferSmallest = "smallest"
)

var (
	ErrDuplicateFound = errors.New("duplicate found")
	ErrInvalidEpisode = errors.New("invalid episode pattern")
)

type SyncContext interface {
	Video() *Video
	Object() *bucket.Object
	Client() *tmdb.TMDB
	Searcher() search.Searcher
}

type syncContext struct {
	video    *Video
	object   *bucket.Object
	client   *tmdb.TMDB
	searcher search.Searcher
}

func (c *syncContext) Video() *Video {
	return c.video
}

func (c *syncContext) Object() *bucket.Object {
	return c.object
}

func (c *syncContext) Client() *tmdb.TMDB {
	return c.client
}

func (c *syncContext) Searcher() search.Searcher {
	return c.searcher
}

func newSyncContext(v *Video, o *bucket.Object, client *tmdb.TMDB, searcher search.Searcher) SyncContext {
	return &syncContext{video: v, object: o, client: client, searcher: searcher}
}

func (v *Video) Sync() error {
	return v.SyncSince(time.Time{})
}

func (v *Video) SyncSince(lastSync time.Time) error {
	for _, bucket := range v.buckets {
		err := v.syncBucket(bucket, lastSync)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	fuzzyNameRegexp = regexp.MustCompile(`[^a-zA-Z0-9]`)

	// Movies/Thriller/Zero Dark Thirty (2012).mkv
	// Movies/Thriller/Zero Dark Thirty (2012) - HD.mkv
	movieRegexp = regexp.MustCompile(`.*/(.+?)\s*\(([\d]+)\)(\s-\s(.+))?\.(mkv|mp4)$`)

	// The Shining
	// Doctor Who (1963) - S01E01 - An Unearthly Child.mkv
	// Sopranos - S05E21.mkv
	// Sopranos - S05E21 - Made in America.mkv
	// Sopranos (1999) - S05E21 - Made in America.mkv
	// Sopranos (2007) - S05E21 - Made in America.mkv
	// Name (Date) - SXXEYY[ - Optional].mkv
	tvRegexp = regexp.MustCompile(`.*/(.+?)\s*\(([\d]+)\)\s+[^\d]*(S\d\dE\d\d)[^\d]*?(?:\s-\s(.+))?\.(mkv|mp4)$`)
)

func (v *Video) syncBucket(bucket bucket.Bucket, lastSync time.Time) error {
	objectCh, err := bucket.List(lastSync)
	if err != nil {
		return err
	}

	client := tmdb.NewTMDB(v.config.TMDB.Config, v.config.NewGetter())

	s, err := v.newSearch()
	if err != nil {
		return err
	}
	defer s.Close()

	var matches []string
	for o := range objectCh {
		matches = movieRegexp.FindStringSubmatch(o.Path)
		if matches != nil {
			title := matches[1]
			year := matches[2]
			err := v.doMovie(o, client, s, title, year)
			if err != nil {
				log.Println(err)
			}
			continue
		}
	}
	return nil
}

func fuzzyName(name string) string {
	return fuzzyNameRegexp.ReplaceAllString(name, "")
}

func (v *Video) doMovie(o *bucket.Object, client *tmdb.TMDB, s search.Searcher, title, year string) error {
	results, err := client.MovieSearch(title)
	if err != nil {
		return err
	}

	index := make(search.IndexMap)

	for _, r := range results {
		//fmt.Printf("result %s %s\n", r.Title, r.ReleaseDate)
		if fuzzyName(title) == fuzzyName(r.Title) &&
			strings.Contains(r.ReleaseDate, year) {
			log.Println("matched", r.Title, r.ReleaseDate)
			fields, err := v.syncMovie(client, r.ID,
				o.Key, o.Size, o.ETag, o.LastModified)
			if err != nil {
				if err != ErrDuplicateFound {
					log.Println(err)
				}
				continue
			}
			index[o.Key] = fields
			break
		}
	}

	s.Index(index)

	return nil
}

func (v *Video) syncMovie(client *tmdb.TMDB, tmid int,
	key string, size int64, etag string, lastModified time.Time) (search.FieldMap, error) {

	// check for duplicates and resolve
	m, err := v.LookupTMID(tmid)
	if err == nil {
		switch v.config.Video.DuplicateResolution {
		case PreferLargest:
			if m.Size >= size {
				// ignore the smaller movie
				return nil, ErrDuplicateFound
			}
		case PreferSmallest:
			if m.Size <= size {
				// ignore the larger movie
				return nil, ErrDuplicateFound
			}
		default:
			log.Panicf("unsupported DuplicateResolution '%s'",
				v.config.Video.DuplicateResolution)
		}
	}

	v.deleteMovie(tmid)
	v.deleteCast(tmid)
	v.deleteCollections(tmid)
	v.deleteCrew(tmid)
	v.deleteGenres(tmid)
	v.deleteKeywords(tmid)

	fields := make(search.FieldMap)

	detail, err := client.MovieDetail(tmid)
	if err != nil {
		return fields, err
	}

	m = Movie{
		TMID:             int64(detail.ID),
		IMID:             detail.IMDB_ID,
		Title:            detail.Title,
		SortTitle:        str.SortTitle(detail.Title),
		OriginalTitle:    detail.OriginalTitle,
		OriginalLanguage: detail.OriginalLanguage,
		BackdropPath:     detail.BackdropPath,
		PosterPath:       detail.PosterPath,
		Budget:           detail.Budget,
		Revenue:          detail.Revenue,
		Overview:         detail.Overview,
		Tagline:          detail.Tagline,
		Runtime:          detail.Runtime,
		VoteAverage:      detail.VoteAverage,
		VoteCount:        detail.VoteCount,
		Date:             date.ParseDate(detail.ReleaseDate), // 2013-02-06
		Key:              key,
		Size:             size,
		ETag:             etag,
		LastModified:     lastModified,
	}

	// rating / certification
	for _, country := range v.config.Video.ReleaseCountries {
		release, err := v.certification(client, tmid, country)
		if err == tmdb.ErrReleaseTypeNotFound {
			continue
		} else if err != nil {
			return fields, err
		}
		m.Rating = release.Certification
		break
	}

	fields.AddField(FieldBudget, m.Budget)
	fields.AddField(FieldDate, m.Date)
	fields.AddField(FieldRating, m.Rating)
	fields.AddField(FieldRevenue, m.Revenue)
	fields.AddField(FieldRuntime, m.Runtime)
	fields.AddField(FieldTitle, m.Title)
	fields.AddField(FieldTagline, m.Tagline)
	fields.AddField(FieldVote, int(m.VoteAverage*10))
	fields.AddField(FieldVoteCount, m.VoteCount)

	err = v.createMovie(&m)
	if err != nil {
		return fields, err
	}

	// collections
	if detail.Collection.Name != "" {
		c := Collection{
			TMID:     m.TMID,
			Name:     detail.Collection.Name,
			SortName: str.SortTitle(detail.Collection.Name),
		}
		err = v.createCollection(&c)
		if err != nil {
			return fields, err
		}
		fields.AddField(FieldCollection, c.Name)
	}

	// genres
	err = v.processGenres(m.TMID, detail.Genres, fields)
	if err != nil {
		return fields, err
	}

	// keywords
	keywords, err := client.MovieKeywordNames(tmid)
	err = v.processKeywords(m.TMID, keywords, fields)
	if err != nil {
		return fields, err
	}

	// credits
	credits, err := client.MovieCredits(tmid)
	if err != nil {
		return fields, err
	}
	err = v.processCredits(m, client, credits, fields)

	return fields, err
}

func (v *Video) certification(client *tmdb.TMDB, tmid int, country string) (tmdb.Release, error) {
	types := []int{tmdb.TypeTheatrical, tmdb.TypeDigital}
	for _, t := range types {
		release, err := client.MovieReleaseType(tmid, country, t)
		if err == tmdb.ErrReleaseTypeNotFound {
			continue
		}
		return release, err
	}
	return tmdb.Release{}, tmdb.ErrReleaseTypeNotFound
}

func (v *Video) processGenres(tmid int64, genres []tmdb.Genre, fields search.FieldMap) error {
	for _, o := range genres {
		g := Genre{
			Name: o.Name,
			TMID: tmid, // as TMID
		}
		err := v.createGenre(&g)
		if err != nil {
			return err
		}
		fields.AddField(FieldGenre, g.Name)
	}
	return nil
}

func (v *Video) processKeywords(tmid int64, keywords []string, fields search.FieldMap) error {
	for _, keyword := range keywords {
		k := Keyword{
			Name: keyword,
			TMID: tmid,
		}
		err := v.createKeyword(&k)
		if err != nil {
			return err
		}
		fields.AddField(FieldKeyword, k.Name)
	}
	return nil
}

func (v *Video) sortedCast(credits tmdb.Credits) []tmdb.Cast {
	limit := v.config.Video.CastLimit
	cast := credits.SortedCast()
	if len(cast) > limit {
		cast = cast[:limit]
	}
	return cast
}

func (v *Video) relevantCrew(credits tmdb.Credits) []tmdb.Crew {
	return credits.CrewWithJobs(v.config.Video.CrewJobs)
}

func (v *Video) processCredits(m Movie, client *tmdb.TMDB, credits tmdb.Credits, fields search.FieldMap) error {
	for _, c := range v.sortedCast(credits) {
		p, err := people.EnsurePerson(c.ID, client, v.db)
		if err != nil {
			return err
		}
		cast, err := v.createCastMember(m, p, c)
		if err != nil {
			return err
		}
		fields.AddField(FieldCast, p.Name)
		fields.AddField(FieldCharacter, cast.Character)
	}
	for _, c := range v.relevantCrew(credits) {
		p, err := people.EnsurePerson(c.ID, client, v.db)
		if err != nil {
			return err
		}
		crew, err := v.createCrewMember(m, p, c)
		if err != nil {
			return err
		}
		fields.AddField(FieldCrew, p.Name)
		fields.AddField(crew.Department, p.Name)
		fields.AddField(crew.Job, p.Name)
	}
	return nil
}

func (v *Video) createCastMember(m Movie, p Person, cast tmdb.Cast) (Cast, error) {
	c := Cast{
		TMID:      m.TMID,
		PEID:      p.PEID,
		Character: cast.Character,
		Rank:      cast.Order,
	}
	err := v.createCast(&c)
	if err != nil {
		return Cast{}, err
	}
	return c, err
}

func (v *Video) createCrewMember(m Movie, p Person, crew tmdb.Crew) (Crew, error) {
	c := Crew{
		TMID:       m.TMID,
		PEID:       p.PEID,
		Department: crew.Department,
		Job:        crew.Job,
	}
	err := v.createCrew(&c)
	if err != nil {
		return Crew{}, err
	}
	return c, nil
}

func (v *Video) SyncPosters(client client.Getter) {
	for _, m := range v.Movies() {
		// sync poster
		img := v.TMDBMoviePoster(m)
		if img != "" {
			log.Printf("sync %s poster %s\n", m.Title, img)
			client.Get(img)
		}

		// sync small poster
		img = v.TMDBMoviePosterSmall(m)
		if img != "" {
			log.Printf("sync %s small poster %s\n", m.Title, img)
			client.Get(img)
		}
	}
}

func (v *Video) SyncBackdrops(client client.Getter) {
	for _, m := range v.Movies() {
		// sync backdrop
		img := v.TMDBMovieBackdrop(m)
		if img != "" {
			log.Printf("sync %s backdrop %s\n", m.Title, img)
			client.Get(img)
		}
	}
}

func (v *Video) SyncProfileImages(client client.Getter) {
	for _, m := range v.Movies() {
		// cast images
		cast := v.Cast(m)
		for _, p := range cast {
			img := v.TMDBPersonProfile(p.Person)
			if img != "" {
				log.Printf("sync %s cast profile %s\n", p.Person.Name, img)
				client.Get(img)
			}
		}

		// crew images
		crew := v.Crew(m)
		for _, p := range crew {
			img := v.TMDBPersonProfile(p.Person)
			if img != "" {
				log.Printf("sync %s crew profile %s\n", p.Person.Name, img)
				client.Get(img)
			}
		}
	}
}
