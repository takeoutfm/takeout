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

// Package activity manages user activity data.
package activity

import (
	"sort"

	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/internal/music"
	"github.com/takeoutfm/takeout/internal/podcast"
	"github.com/takeoutfm/takeout/internal/video"
	"github.com/takeoutfm/takeout/lib/log"
	. "github.com/takeoutfm/takeout/model"
	"gorm.io/gorm"

	"errors"
	"strconv"
	"time"
)

var (
	ErrInvalidUser     = errors.New("invalid user")
	ErrTrackNotFound   = errors.New("track not found")
	ErrMovieNotFound   = errors.New("movie not found")
	ErrEpisodeNotFound = errors.New("episode not found")
	ErrReleaseNotFound = errors.New("release not found")
)

type Context interface {
	Music() *music.Music
	Podcast() *podcast.Podcast
	User() auth.User
	Video() *video.Video
}

type Activity struct {
	config *config.Config
	db     *gorm.DB
}

func NewActivity(config *config.Config) *Activity {
	return &Activity{
		config: config,
	}
}

func (a *Activity) Open() error {
	return a.openDB()
}

func (a *Activity) Close() {
	a.closeDB()
}

func (a *Activity) DeleteUserEvents(ctx Context) error {
	user := ctx.User()
	err := a.deleteMovieEvents(user.Name)
	if err != nil {
		log.Println("movie delete error: ", err)
		return err
	}
	err = a.deleteEpisodeEvents(user.Name)
	if err != nil {
		log.Println("series delete error: ", err)
		return err
	}
	err = a.deleteTrackEvents(user.Name)
	if err != nil {
		log.Println("track delete error: ", err)
		return err
	}
	return nil
}

func (a *Activity) resolveMovieEvent(e MovieEvent, ctx Context) (ActivityMovie, error) {
	v := ctx.Video()
	if e.IMID == "" {
		return ActivityMovie{}, ErrMovieNotFound
	}
	movie, err := v.FindMovie(e.IMID)
	if err != nil {
		return ActivityMovie{}, err
	}
	result := ActivityMovie{}
	//result.Count = e.Count
	result.Movie = movie
	return result, nil
}

func (a *Activity) resolveEpisodeEvent(e EpisodeEvent, ctx Context) (ActivityEpisode, error) {
	p := ctx.Podcast()
	if e.EID == "" {
		return ActivityEpisode{}, ErrEpisodeNotFound
	}
	episode, err := p.FindEpisode(e.EID)
	if err != nil {
		return ActivityEpisode{}, err
	}
	result := ActivityEpisode{}
	//result.Count = e.Count
	result.Episode = episode
	return result, nil
}

func (a *Activity) resolveTrackEvent(e trackEvent, ctx Context) (ActivityTrack, error) {
	m := ctx.Music()

	var err error
	var track Track
	if e.RID != "" {
		track, err = m.FindTrack(e.RID)
		if err != nil {
			log.Printf("track event %d, RID %s not found\n", e.ID, e.RID)
			return ActivityTrack{}, err
		}
	} else if e.ETag != "" {
		track, err = m.LookupETag(e.ETag)
		if err != nil {
			log.Printf("track event %d, etag %s not found\n", e.ID, e.ETag)
			return ActivityTrack{}, err
		}
	}
	result := ActivityTrack{}
	result.Count = e.Count
	result.Track = track
	return result, nil
}

func (a *Activity) resolveMovieEvents(events []MovieEvent, ctx Context) []ActivityMovie {
	movies := []ActivityMovie{}
	for _, e := range events {
		movie, err := a.resolveMovieEvent(e, ctx)
		if err == nil {
			movies = append(movies, movie)
		}
	}
	return movies
}

func (a *Activity) resolveEpisodeEvents(events []EpisodeEvent, ctx Context) []ActivityEpisode {
	episodes := []ActivityEpisode{}
	for _, e := range events {
		episode, err := a.resolveEpisodeEvent(e, ctx)
		if err == nil {
			episodes = append(episodes, episode)
		}
	}
	return episodes
}

func (a *Activity) resolveTrackEvents(events []trackEvent, ctx Context) []ActivityTrack {
	tracks := []ActivityTrack{}
	for _, e := range events {
		track, err := a.resolveTrackEvent(e, ctx)
		if err == nil {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

func (a *Activity) Movies(ctx Context, start, end time.Time) []ActivityMovie {
	user := ctx.User()
	events := a.movieEventsFrom(user.Name, start, end, a.config.Activity.EventLimit)
	return a.resolveMovieEvents(events, ctx)
}

func (a *Activity) Tracks(ctx Context, start, end time.Time) []ActivityTrack {
	user := ctx.User()
	events := a.trackEventsFrom(user.Name, start, end, a.config.Activity.EventLimit)
	return a.resolveTrackEvents(events, ctx)
}

func (a *Activity) TopTracks(ctx Context, start, end time.Time) []ActivityTrack {
	user := ctx.User()
	events := a.topTrackEventsFrom(user.Name, start, end, a.config.Activity.TopTracksLimit)
	return a.resolveTrackEvents(events, ctx)
}

func (a *Activity) TopArtists(ctx Context, start, end time.Time) []ActivityArtist {
	user := ctx.User()
	events := a.trackEventsFrom(user.Name, start, end, a.config.Activity.TopTracksLimit)
	tracks := a.resolveTrackEvents(events, ctx)
	result := a.groupByArtist(ctx, tracks)
	if len(result) > a.config.Activity.TopArtistsLimit {
		result = result[:a.config.Activity.TopArtistsLimit]
	}
	return result
}

func (a *Activity) TopReleases(ctx Context, start, end time.Time) []ActivityRelease {
	user := ctx.User()
	events := a.trackEventsFrom(user.Name, start, end, a.config.Activity.TopTracksLimit)
	tracks := a.resolveTrackEvents(events, ctx)
	result := a.groupByRelease(ctx, tracks)
	if len(result) > a.config.Activity.TopReleasesLimit {
		result = result[:a.config.Activity.TopReleasesLimit]
	}
	return result
}

func (a *Activity) groupByArtist(ctx Context, tracks []ActivityTrack) []ActivityArtist {
	// count tracks by artist (ARID)
	counts := make(map[string]int)
	for _, t := range tracks {
		counts[t.Track.Artist]++
	}

	keys := sortByCount(counts)

	list := ctx.Music().Artists()
	artists := make(map[string]Artist)
	for _, v := range list {
		artists[v.Name] = v
	}

	result := make([]ActivityArtist, 0, len(keys))
	for _, key := range keys {
		artist := artists[key]
		count := counts[key]
		result = append(result, ActivityArtist{Artist: artist, Count: count})
	}

	return result
}

func (a *Activity) groupByRelease(ctx Context, tracks []ActivityTrack) []ActivityRelease {
	// count tracks by release (REID)
	counts := make(map[string]int)
	for _, t := range tracks {
		counts[t.Track.REID]++
	}

	keys := sortByCount(counts)

	// build release map with REID as key
	list := ctx.Music().ReleasesForREIDs(keys)
	releases := make(map[string]Release)
	for _, v := range list {
		releases[v.REID] = v
	}

	result := make([]ActivityRelease, 0, len(keys))
	for _, key := range keys {
		release := releases[key]
		count := counts[key]
		result = append(result, ActivityRelease{Release: release, Count: count})
	}

	return result
}

func sortByCount(counts map[string]int) []string {
	// sort keys by count
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return counts[keys[i]] > counts[keys[j]]
	})
	return keys
}

// Add a scrobble with an MBID that should match a track we have
func (a *Activity) UserScrobble(user auth.User, s Scrobble, music *music.Music) error {
	// ensure there's a valid user
	// if s.User == "" {
	// 	s.User = user.Name
	// } else if s.User != user.Name {
	// 	return ErrInvalidUser
	// }

	// if s.MBID != "" {
	// 	_, err := music.FindTrack(s.MBID)
	// 	if err != nil {
	// 		// no track with that MBID (RID)
	// 		// code below will hopefully find a new one
	// 		s.MBID = ""
	// 	}
	// }
	// if s.MBID == "" {
	// 	tracks := music.SearchTracks(s.Track, s.PreferredArtist(), s.Album)
	// 	if len(tracks) > 0 {
	// 		// use first matching track MBZ recording ID
	// 		s.MBID = tracks[0].RID
	// 	}
	// }

	// // MBID may still be empty but allow anyway for now
	// return a.createScrobble(&s)
	return nil
}

func (a *Activity) CreateEvents(ctx Context, events Events) error {
	user := ctx.User()
	for _, e := range events.MovieEvents {
		e.User = user.Name
		e.Date = e.Date.Local()
		if e.ETag != "" {
			// resolve using ETag
			video, err := ctx.Video().LookupETag(e.ETag)
			if err != nil {
				return err
			}
			e.IMID = video.IMID
			e.TMID = strconv.FormatInt(video.TMID, 10)
		}
		if e.IsValid() {
			err := a.createMovieEvent(&e)
			if err != nil {
				return err
			}
		}
		// TODO ignore invalid events
	}

	// for _, e := range events.ReleaseEvents {
	// 	e.User = user.Name
	// 	e.Date = e.Date.Local()
	// 	if e.IsValid() {
	// 		err := a.createReleaseEvent(&e)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	// TODO ignore invalid events
	// }

	for _, e := range events.EpisodeEvents {
		e.User = user.Name
		e.Date = e.Date.Local()
		if e.IsValid() {
			err := a.createEpisodeEvent(&e)
			if err != nil {
				return err
			}
		}
		// TODO ignore invalid events
	}

	for _, e := range events.TrackEvents {
		e.User = user.Name
		e.Date = e.Date.Local()
		if e.ETag != "" {
			// resolve using ETag
			track, err := ctx.Music().LookupETag(e.ETag)
			if err != nil {
				return err
			}
			e.RID = track.RID
			e.RGID = track.RGID
		}
		if e.IsValid() {
			err := a.createTrackEvent(&e)
			if err != nil {
				return err
			}
		}
		// TODO ignore invalid events
	}

	return nil
}
