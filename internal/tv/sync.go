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
	"fmt"
	"regexp"
	"strings"
	"time"

	"takeoutfm.dev/takeout/internal/people"
	"takeoutfm.dev/takeout/lib/bucket"
	"takeoutfm.dev/takeout/lib/client"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/lib/search"
	"takeoutfm.dev/takeout/lib/str"
	"takeoutfm.dev/takeout/lib/tmdb"
	. "takeoutfm.dev/takeout/model"
)

const (
	FieldCast      = "cast"
	FieldCharacter = "character"
	FieldCrew      = "crew"
	FieldDate      = "date"
	FieldEpisode   = "episode"
	FieldGenre     = "genre"
	FieldKeyword   = "keyword"
	FieldName      = "name"
	FieldRating    = "rating"
	FieldRevenue   = "revenue"
	FieldRuntime   = "runtime"
	FieldSeason    = "season"
	FieldSeries    = "series"
	FieldTagline   = "tagline"
	FieldTitle     = "title"
	FieldVote      = "vote"
	FieldVoteCount = "vote_count"
)

type syncContext struct {
	series map[string]int
}

func (tv *TV) Sync() error {
	return tv.SyncSince(time.Time{})
}

func (tv *TV) SyncSince(lastSync time.Time) error {
	for _, bucket := range tv.buckets {
		err := tv.syncBucket(bucket, lastSync)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	fuzzyNameRegexp = regexp.MustCompile(`[^a-zA-Z0-9]`)

	// s##e##
	// S##E##
	episodeRegexp = regexp.MustCompile(`(?i)S(\d\d)E(\d\d)`)

	// Doctor Who (1963) - S01E01 - An Unearthly Child.mkv
	// Sopranos (1999) - S06E21.mkv
	// Sopranos (2007) - S06E21 - Made in America.mkv
	// Name (Date) - SXXEYY[ - Optional].mkv
	tvRegexp = regexp.MustCompile(`.*/(.+?)\s*\(([\d]+)\)\s+[^\d]*(S\d\dE\d\d)[^\d]*?(?:\s-\s(.+))?\.(mkv|mp4)$`)
)

func (tv *TV) syncBucket(bucket bucket.Bucket, lastSync time.Time) error {
	objectCh, err := bucket.List(lastSync)
	if err != nil {
		return err
	}

	s, err := tv.newSearch()
	if err != nil {
		return err
	}
	defer s.Close()

	context := syncContext{}
	context.series = make(map[string]int)

	var matches []string
	for o := range objectCh {
		matches = tvRegexp.FindStringSubmatch(o.Path)
		if matches != nil && len(matches) >= 4 {
			series := matches[1]
			year := matches[2]
			detail := matches[3]
			err = tv.doEpisode(&context, o, s, series, year, detail)
			if err != nil {
				log.Println(err)
			}
		}
	}
	return nil
}

func fuzzyName(name string) string {
	return fuzzyNameRegexp.ReplaceAllString(name, "")
}

func (tv *TV) doEpisode(context *syncContext, o *bucket.Object, s search.Searcher, series, year, detail string) error {
	season, episode, err := parseEpisode(detail)

	results, err := tv.tmdb.TVSearch(series)
	if err != nil {
		return err
	}

	index := make(search.IndexMap)

	for _, r := range results {
		fmt.Printf("result %s %s\n", r.Name, r.FirstAirDate)
		if fuzzyName(series) == fuzzyName(r.Name) &&
			strings.Contains(r.FirstAirDate, year) {
			log.Println("matched", r.Name, r.FirstAirDate)

			_, ok := context.series[r.Name]
			if !ok {
				_, err := tv.syncSeries(r.ID)
				if err == nil {
					context.series[r.Name] = r.ID
				} else {
					log.Println(err)
				}
			}

			fields, err := tv.syncEpisode(o, r.ID, season, episode)
			if err != nil {
				log.Println(err)
				continue
			}

			// only episode fields are stored in the index
			index[o.Key] = fields
			break
		}
	}

	s.Index(index)

	return nil
}

func parseEpisode(s string) (int, int, error) {
	matches := episodeRegexp.FindStringSubmatch(s)
	if len(matches) != 3 {
		return -1, -1, ErrInvalidEpisode
	}

	season := str.Atoi(matches[1])
	episode := str.Atoi(matches[2])

	if season == 0 || episode == 0 {
		return -1, -1, ErrInvalidEpisode
	}

	return season, episode, nil
}

// sync the series with genries & keywords
func (tv *TV) syncSeries(tvid int) (search.FieldMap, error) {
	fields := make(search.FieldMap)

	detail, err := tv.tmdb.TVDetail(tvid)
	if err != nil {
		return fields, err
	}

	series, err := tv.LookupSeries(tvid)
	if err != nil {
		if err != ErrSeriesNotFound {
			return fields, err
		}
		// series doesn't exist, create it with tvid
		series = TVSeries{TVID: int64(detail.ID)}
		err = tv.createSeries(&series)
		if err != nil {
			return fields, err
		}
	}
	series.Name = detail.Name
	series.SortName = str.SortTitle(detail.Name)
	series.OriginalName = detail.OriginalName
	series.OriginalLanguage = detail.OriginalLanguage
	series.BackdropPath = detail.BackdropPath
	series.PosterPath = detail.PosterPath
	series.Overview = detail.Overview
	series.Tagline = detail.Tagline
	series.SeasonCount = detail.NumberOfSeasons
	series.EpisodeCount = detail.NumberOfEpisodes
	series.VoteAverage = detail.VoteAverage
	series.VoteCount = detail.VoteCount
	series.Date = date.ParseDate(detail.FirstAirDate)   // 2013-02-06
	series.EndDate = date.ParseDate(detail.LastAirDate) // 2013-02-06

	// get rating
	rating, err := tv.rating(tvid)
	if err != nil {
		return fields, err
	}
	series.Rating = rating

	// update existing series detail
	err = tv.updateSeries(&series)
	if err != nil {
		return fields, err
	}

	// TODO additional season metadata is ignored but things like name, air
	// date, overview, and poster could be useful later.

	fields.AddField(FieldDate, series.Date)
	fields.AddField(FieldName, series.Name)
	fields.AddField(FieldTagline, series.Tagline)
	fields.AddField(FieldVote, int(series.VoteAverage*10))
	fields.AddField(FieldVoteCount, series.VoteCount)
	fields.AddField(FieldRating, series.Rating)

	// delete and repopulate all other related detail
	tv.deleteSeriesCast(tvid)
	tv.deleteSeriesCrew(tvid)
	tv.deleteGenres(tvid)
	tv.deleteKeywords(tvid)

	credits, err := tv.tmdb.SeriesCredits(tvid)
	if err != nil {
		return fields, err
	}
	err = tv.processSeriesCredits(series, credits, fields)
	if err != nil {
		return fields, err
	}
	err = tv.syncProcessGenres(series.TVID, detail.Genres, fields)
	if err != nil {
		return fields, err
	}
	keywords, err := tv.tmdb.TVKeywordNames(tvid)
	if err != nil {
		return fields, err
	}
	err = tv.syncProcessKeywords(series.TVID, keywords, fields)
	if err != nil {
		return fields, err
	}

	return fields, nil
}

func (tv *TV) rating(tvid int) (string, error) {
	ratings, err := tv.tmdb.TVContentRatings(tvid)
	if err != nil {
		return "", err
	}
	ratingMap := make(map[string]tmdb.ContentRating)
	for _, r := range ratings.Results {
		ratingMap[r.Country] = r
	}
	for _, country := range tv.config.TV.ReleaseCountries {
		r, ok := ratingMap[country]
		if ok {
			return r.Rating, nil
		}
	}
	return "", ErrRatingNotFound
}

func (tv *TV) syncEpisode(o *bucket.Object, tvid, season, episode int) (search.FieldMap, error) {
	fields := make(search.FieldMap)

	series, err := tv.LookupTVID(tvid)
	if err != nil {
		// series must exist
		return fields, err
	}

	ep, err := tv.FindSeasonEpisode(series, season, episode)
	if err != nil {
		if err != ErrEpisodeNotFound {
			return fields, err
		}
		ep = TVEpisode{TVID: int64(tvid), Season: season, Episode: episode}
		err = tv.createEpisode(&ep)
		if err != nil {
			return fields, err
		}
	}

	detail, err := tv.tmdb.EpisodeDetail(tvid, season, episode)
	if err != nil {
		return fields, err
	}

	ep.Name = detail.Name
	ep.Overview = detail.Overview
	ep.Date = date.ParseDate(detail.AirDate) // 2013-02-0
	ep.StillPath = detail.StillPath
	ep.VoteAverage = detail.VoteAverage
	ep.VoteCount = detail.VoteCount
	ep.Runtime = detail.Runtime
	ep.Key = o.Key
	ep.Size = o.Size
	ep.ETag = o.ETag
	ep.LastModified = o.LastModified

	// updating existing episode detail
	err = tv.updateEpisode(&ep)
	if err != nil {
		return fields, err
	}

	fields.AddField(FieldSeries, series.Name)
	fields.AddField(FieldSeason, ep.Season)
	fields.AddField(FieldEpisode, ep.Episode)
	fields.AddField(FieldName, ep.Name)
	fields.AddField(FieldTitle, ep.Name)
	fields.AddField(FieldDate, ep.Date)
	fields.AddField(FieldRuntime, ep.Runtime)
	fields.AddField(FieldVote, int(ep.VoteAverage*10))
	fields.AddField(FieldVoteCount, ep.VoteCount)

	// delete and repopulate all other stuff
	tv.deleteEpisodeCast(ep)
	tv.deleteEpisodeCrew(ep)

	credits, err := tv.tmdb.EpisodeCredits(tvid, season, episode)
	if err != nil {
		return fields, err
	}
	err = tv.processEpisodeCredits(ep, credits, fields)
	if err != nil {
		return fields, err
	}

	return fields, err
}

func (tv *TV) syncProcessGenres(tvid int64, genres []tmdb.Genre, fields search.FieldMap) error {
	for _, o := range genres {
		g := TVGenre{
			Name: o.Name,
			TVID: tvid,
		}
		err := tv.createGenre(&g)
		if err != nil {
			return err
		}
		fields.AddField(FieldGenre, g.Name)
	}
	return nil
}

func (tv *TV) syncProcessKeywords(tvid int64, keywords []string, fields search.FieldMap) error {
	for _, keyword := range keywords {
		k := TVKeyword{
			Name: keyword,
			TVID: tvid,
		}
		err := tv.createKeyword(&k)
		if err != nil {
			return err
		}
		fields.AddField(FieldKeyword, k.Name)
	}
	return nil
}

func (tv *TV) processSeriesCredits(s TVSeries, credits tmdb.Credits, fields search.FieldMap) error {
	for _, c := range tv.config.TV.SortedCast(credits) {
		p, err := people.EnsurePerson(c.ID, tv.tmdb, tv.db)
		if err != nil {
			return err
		}
		cast := newSeriesCast(s, p, c)
		err = tv.createSeriesCast(&cast)
		if err != nil {
			return err
		}
		fields.AddField(FieldCast, p.Name)
		fields.AddField(FieldCharacter, cast.Character)
	}
	for _, c := range tv.config.TV.RelevantCrew(credits) {
		p, err := people.EnsurePerson(c.ID, tv.tmdb, tv.db)
		if err != nil {
			return err
		}
		crew := newSeriesCrew(s, p, c)
		err = tv.createSeriesCrew(&crew)
		if err != nil {
			return err
		}
		fields.AddField(FieldCrew, p.Name)
		fields.AddField(crew.Department, p.Name)
		fields.AddField(crew.Job, p.Name)
	}
	return nil
}

func (tv *TV) processEpisodeCredits(e TVEpisode, credits tmdb.Credits, fields search.FieldMap) error {
	for _, c := range tv.config.TV.SortedCast(credits) {
		p, err := people.EnsurePerson(c.ID, tv.tmdb, tv.db)
		if err != nil {
			return err
		}
		cast := newEpisodeCast(e, p, c)
		err = tv.createEpisodeCast(&cast)
		if err != nil {
			return err
		}
		fields.AddField(FieldCast, p.Name)
		fields.AddField(FieldCharacter, cast.Character)
	}
	for _, c := range tv.config.TV.RelevantCrew(credits) {
		p, err := people.EnsurePerson(c.ID, tv.tmdb, tv.db)
		if err != nil {
			return err
		}
		crew := newEpisodeCrew(e, p, c)
		err = tv.createEpisodeCrew(&crew)
		if err != nil {
			return err
		}
		fields.AddField(FieldCrew, p.Name)
		fields.AddField(crew.Department, p.Name)
		fields.AddField(crew.Job, p.Name)
	}
	return nil
}

func newEpisodeCast(e TVEpisode, p Person, cast tmdb.Cast) TVEpisodeCast {
	return TVEpisodeCast{
		EID:       e.ID,
		PEID:      p.PEID,
		Character: cast.Character,
		Rank:      cast.Order,
	}
}

func newEpisodeCrew(e TVEpisode, p Person, crew tmdb.Crew) TVEpisodeCrew {
	return TVEpisodeCrew{
		EID:        e.ID,
		PEID:       p.PEID,
		Department: crew.Department,
		Job:        crew.Job,
	}
}

func newSeriesCast(s TVSeries, p Person, cast tmdb.Cast) TVSeriesCast {
	return TVSeriesCast{
		TVID:      s.TVID,
		PEID:      p.PEID,
		Character: cast.Character,
		Rank:      cast.Order,
	}
}

func newSeriesCrew(s TVSeries, p Person, crew tmdb.Crew) TVSeriesCrew {
	return TVSeriesCrew{
		TVID:       s.TVID,
		PEID:       p.PEID,
		Department: crew.Department,
		Job:        crew.Job,
	}
}

func (tv *TV) SyncPosters(client client.Getter) {
	for _, s := range tv.Series() {
		// sync poster
		img := tv.TMDBSeriesPoster(s)
		if img != "" {
			log.Printf("sync %s poster %s\n", s.Name, img)
			client.Get(img)
		}

		// sync small poster
		img = tv.TMDBSeriesPosterSmall(s)
		if img != "" {
			log.Printf("sync %s small poster %s\n", s.Name, img)
			client.Get(img)
		}
	}
}

func (tv *TV) SyncBackdrops(client client.Getter) {
	for _, s := range tv.Series() {
		// sync backdrop
		img := tv.TMDBSeriesBackdrop(s)
		if img != "" {
			log.Printf("sync %s backdrop %s\n", s.Name, img)
			client.Get(img)
		}
	}
}

func (tv *TV) SyncStills(client client.Getter) {
	for _, s := range tv.Series() {
		for _, e := range tv.SeriesEpisodes(s) {
			// sync still
			img := tv.TMDBEpisodeStill(e)
			if img != "" {
				log.Printf("sync %s s%de%d still %s\n", s.Name, e.Season, e.Episode, img)
				client.Get(img)
			}

			// sync still small
			img = tv.TMDBEpisodeStillSmall(e)
			if img != "" {
				log.Printf("sync %s s%de%d still small %s\n", s.Name, e.Season, e.Episode, img)
				client.Get(img)
			}
		}
	}
}

func (tv *TV) SyncProfileImages(client client.Getter) {
	for _, s := range tv.Series() {
		// cast images
		cast := tv.SeriesCast(s)
		for _, p := range cast {
			img := tv.TMDBPersonProfile(p.Person)
			if img != "" {
				log.Printf("sync %s cast profile %s\n", p.Person.Name, img)
				client.Get(img)
			}
		}

		// crew images
		crew := tv.SeriesCrew(s)
		for _, p := range crew {
			img := tv.TMDBPersonProfile(p.Person)
			if img != "" {
				log.Printf("sync %s crew profile %s\n", p.Person.Name, img)
				client.Get(img)
			}
		}
	}
}
