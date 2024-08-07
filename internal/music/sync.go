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

package music

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/takeoutfm/takeout/lib/client"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/lib/log"
	"github.com/takeoutfm/takeout/lib/musicbrainz"
	"github.com/takeoutfm/takeout/lib/search"
	"github.com/takeoutfm/takeout/lib/str"
	. "github.com/takeoutfm/takeout/model"
)

type SyncOptions struct {
	Since    time.Time
	Tracks   bool
	Releases bool
	Popular  bool
	Similar  bool
	Index    bool
	Artwork  bool
	Artist   string
	Resolve  bool
}

func NewSyncOptions() SyncOptions {
	return SyncOptions{
		Since:    time.Time{},
		Tracks:   true,
		Releases: true,
		Popular:  true,
		Similar:  true,
		Artwork:  true,
		Index:    true,
	}
}

func NewSyncPopular() SyncOptions {
	return SyncOptions{
		Since:   time.Time{},
		Popular: true,
	}
}

func NewSyncSimilar() SyncOptions {
	return SyncOptions{
		Since:   time.Time{},
		Similar: true,
	}
}

func (m *Music) LastModified() time.Time {
	return m.lastModified()
}

func (m *Music) Sync(options SyncOptions) {
	if options.Since.IsZero() {
		if options.Tracks {
			log.Printf("sync tracks\n")
			log.CheckError(m.syncBucketTracks())
			log.Printf("sync artists\n")
			log.CheckError(m.syncArtists())
		}
		if options.Releases {
			log.Printf("sync releases\n")
			log.CheckError(m.syncReleases())
			log.Printf("fix track releases\n")
			_, err := m.fixTrackReleases()
			log.CheckError(err)
			log.Printf("assign track releases\n")
			_, err = m.assignTrackReleases()
			log.CheckError(err)
			_, err = m.assignTrackReleaseDates()
			// XXX this crashed with release group not found
			log.CheckError(err)
			log.Printf("fix track release titles\n")
			log.CheckError(m.fixTrackReleaseTitles())
		}
		if options.Popular {
			log.Printf("sync popular\n")
			log.CheckError(m.syncPopular())
		}
		if options.Similar {
			log.Printf("sync similar\n")
			log.CheckError(m.syncSimilar())
		}
		if options.Artwork {
			log.Printf("sync artwork\n")
			log.CheckError(m.syncArtwork())
		}
		if options.Index {
			log.Printf("sync index\n")
			log.CheckError(m.syncIndex())
		}
	} else {
		if options.Resolve {
			log.Printf("resolving")
			err := m.resolve()
			log.CheckError(err)
		}
		if options.Tracks {
			modified, err := m.syncBucketTracksSince(options.Since)
			log.CheckError(err)
			if modified {
				log.CheckError(m.syncArtists())
			}
		}
		var artists []Artist
		if options.Artist != "" {
			a, err := m.Artist(options.Artist)
			if err == nil {
				artists = []Artist{a}
			} else {
				a, err := m.syncArtist(options.Artist, "")
				log.CheckError(err)
				artists = []Artist{a}
			}
		} else {
			artists = m.trackArtistsSince(options.Since)
		}
		if options.Releases {
			log.CheckError(m.syncReleasesFor(artists))
			_, err := m.fixTrackReleases()
			log.CheckError(err)
			modified, err := m.assignTrackReleases()
			log.CheckError(err)
			if modified {
				_, err = m.assignTrackReleaseDates()
				log.CheckError(err)
				log.CheckError(m.fixTrackReleaseTitles())
			}
		}
		if options.Popular {
			log.CheckError(m.syncPopularFor(artists))
		}
		if options.Similar {
			log.CheckError(m.syncSimilarFor(artists))
		}
		if options.Artwork {
			log.CheckError(m.syncArtworkFor(artists))
			if len(artists) > 0 {
				log.CheckError(m.syncMissingArtwork())
			}
		}
		if options.Index {
			log.CheckError(m.syncIndexFor(artists))
		}
	}
}

// TODO update steps
//
// sync steps:
// 1. Sync tracks from bucket based on path name
//    -> Table: tracks
// 2. Sync artists from MusicBrainz (arid)
//    a. Obtain arid for each artist using MusicBrainz
//    b. If none, try last.fm to get arid and use MusicBrainz
//    c. Update track artist name from MusicBrainz
//    -> Table: artists, artist_tags, tracks
// 3. Sync releases for artist from MusicBrainz
//    a. Obtain and store each release group from MusicBrainz (rgid)
//    b. Match each track release with release group
//    c. For tracks w/o matches, search MusicBrainz a release (reid)
//    -> Table: releases, tracks
// 4. Sync top tracks from last.fm for each artist using arid
//    -> Table: popular
// 5. Sync similar artists from last.fm for each artist using arid
//    -> Table: similar
// 6. Sync credits
//    -> Bleve: xxx

func (m *Music) syncBucketTracks() error {
	m.deleteTracks() // !!!
	_, err := m.syncBucketTracksSince(time.Time{})
	return err
}

func (m *Music) syncBucketTracksSince(lastSync time.Time) (modified bool, err error) {
	for _, b := range m.buckets {
		trackCh, err := m.syncFromBucket(b, lastSync)
		if err != nil {
			log.Printf("got sync err %s\n", err)
			return false, err
		}
		for t := range trackCh {
			//log.Printf("sync: %s/%s/%s\n", t.Artist, t.Release, t.Title)
			t.Artist = fixName(t.Artist)
			t.Release = fixName(t.Release)
			t.Title = fixName(t.Title)
			// TODO: title may have underscores - picard
			m.createTrack(t)
			modified = true
		}
		err = m.updateTrackCount()
	}
	return
}

func (m *Music) trackArtistsSince(lastSync time.Time) []Artist {
	tracks := m.tracksAddedSince(lastSync)
	var artists []Artist
	h := make(map[string]bool)
	for _, t := range tracks {
		_, ok := h[t.Artist]
		if ok {
			continue
		}
		h[t.Artist] = true
		a, err := m.Artist(t.Artist)
		if err == nil {
			artists = append(artists, a)
		}

	}
	return artists
}

// Obtain all releases for each track artist. This will update
// existing releases as well.
func (m *Music) syncReleases() error {
	return m.syncReleasesFor(m.Artists())
}

func (m *Music) syncReleasesFor(artists []Artist) error {
	for _, a := range artists {
		var releases []Release
		log.Printf("releases for %s\n", a.Name)
		if a.Name == VariousArtists {
			// various artists has many thousands of releases so
			// instead of getting all releases, search for them by
			// name and then get releases
			names := m.artistTrackReleases(a.Name)
			for _, name := range names {
				result, _ := m.mbz.SearchReleaseGroup(a.ARID, name)
				for _, rg := range result.ReleaseGroups {
					r, _ := m.mbz.Releases(rg.ID)
					for _, v := range r {
						releases = append(releases, doRelease(a.Name, v))
					}
				}
			}
		} else {
			r, _ := m.mbz.ArtistReleases(a.Name, a.ARID)
			for _, v := range r {
				releases = append(releases, doRelease(a.Name, v))
			}
		}
		for _, r := range releases {
			r.Name = fixName(r.Name)
			r.SingleName = fixName(r.SingleName)
			for i := range r.Media {
				r.Media[i].Name = fixName(r.Media[i].Name)
			}

			curr, err := m.release(r.REID)
			if err != nil {
				err := m.createRelease(&r)
				if err != nil {
					log.Println(err)
					return err
				}
				for _, d := range r.Media {
					err := m.createMedia(&d)
					if err != nil {
						log.Println(err)
						return err
					}
				}
			} else {
				if curr.Artist != r.Artist {
					log.Printf("release artist conflict '%s' vs. '%s'\n", curr.Artist, r.Artist)
				}
				err := m.replaceRelease(curr, r)
				if err != nil {
					log.Println(err)
					return err
				}
				// update any assigned tracks
				tracks := m.ReleaseTracks(r)
				for _, t := range tracks {
					m.assignTrackRelease(t, r)
				}
				// delete existing release and (re)add new
				m.deleteReleaseMedia(r.REID)
				for _, d := range r.Media {
					err := m.createMedia(&d)
					if err != nil {
						log.Println(err)
						return err
					}
				}
			}
		}
	}
	return nil
}

func (m *Music) checkMissingArtwork() error {
	missing := m.releasesWithoutArtwork()
	for _, r := range missing {
		err := m.checkReleaseArtwork(r)
		if err != nil {
			log.Printf("err was %s\n", err)
		}
	}
	return nil
}

func (m *Music) checkReleaseArtwork(r Release) error {
	if r.Artwork && r.FrontArtwork == false {
		log.Printf("need artwork for %s / %s\n", r.Artist, r.Name)
		// have artwork but no front cover
		art, err := m.mbz.CoverArtArchive(r.REID, r.RGID)
		if err != nil {
			return err
		}
		if len(art.Images) > 0 {
			id := art.Images[0].ID
			r.OtherArtwork = id
			err = m.updateOtherArtwork(r, id)
			if err != nil {
				return err
			}
		}
	} else if r.Artwork == false {
		log.Printf("check artwork for %s / %s\n", r.Artist, r.Name)
		art, err := m.mbz.CoverArtArchive(r.REID, r.RGID)
		if err != nil {
			return err
		}
		front, back := false, false
		for _, img := range art.Images {
			if img.Front {
				front = true
			}
			if img.Back {
				back = true
			}
		}
		err = m.updateArtwork(r, front, back, art.FromGroup)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	fuzzyArtistRegexp = regexp.MustCompile(`[^a-zA-Z0-9& -]`)
	fuzzyNameRegexp   = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

func fuzzyArtist(name string) string {
	return fuzzyArtistRegexp.ReplaceAllString(name, "")
}

func FuzzyName(name string) string {
	// treat "№" the same as "No" for comparison - STP album №4
	name = strings.Replace(name, "№", "No", -1)
	// treat "p·u·l·s·e" as "pulse" for comparison - Pink Floyd album Pulse
	name = strings.Replace(name, "p·u·l·s·e", "Pulse", -1)
	// TODO need to configurize this stuff
	return fuzzyNameRegexp.ReplaceAllString(name, "")
}

func fixName(name string) string {
	// TODO: use Map?
	name = strings.Replace(name, "–", "-", -1)
	name = strings.Replace(name, "‐", "-", -1)
	name = strings.Replace(name, "’", "'", -1)
	name = strings.Replace(name, "‘", "'", -1)
	name = strings.Replace(name, "“", "\"", -1)
	name = strings.Replace(name, "”", "\"", -1)
	name = strings.Replace(name, "…", "...", -1)
	return name
}

func releaseKey(t Track) string {
	return fmt.Sprintf("%s/%s/%d/%d", t.Artist, t.Release, t.TrackCount, t.DiscCount)
}

// Assign a track to a specific MusicBrainz REID. This isn't exact and
// instead will pick the first release with the same name with the
// same number of tracks. This way original release dates are
// presented to the user. An attempt is also made to match using
// disambiguations, things like:
// Weezer:
//
//	Weezer (Blue Album)
//	Weezer - Blue Album
//	Weezer Blue Album
//	Weezer [Blue Album]
//	Blue Album
//
// David Bowie:
//
//	★ (Blackstar)
//	Blackstar
func (m *Music) assignTrackReleases() (bool, error) {
	modified := false
	notFound := make(map[string]struct{})
	artChecked := make(map[string]struct{})
	releaseCache := make(map[string]Release)

	tracks := m.tracksWithoutAssignedRelease()

	for _, t := range tracks {
		cacheKey := releaseKey(t)
		if _, ok := notFound[cacheKey]; ok {
			continue
		}

		r, ok := releaseCache[cacheKey]
		if !ok {
			r, err := m.findTrackRelease(t)
			if err != nil {
				r, err = m.findTrackReleaseDisambiguate(t)
				if err != nil {
					notFound[cacheKey] = struct{}{}
					log.Printf("track release not found: %s\n", cacheKey)
					continue
				}
			}
			releaseCache[cacheKey] = r
		}

		//log.Printf("assign track release %s\n", cacheKey)
		err := m.assignTrackRelease(t, r)
		modified = true
		if err != nil {
			return modified, err
		}

		// ensure releases assigned to tracks have artwork
		if _, ok := artChecked[r.REID]; !ok {
			err := m.checkReleaseArtwork(r)
			if err != nil {
				log.Println(err)
				// could be 404 continue
			}
			artChecked[r.REID] = struct{}{}
		}
	}

	return modified, nil
}

// this is primarily for local tracks where the REID was obtained from tags but
// remaining release metadata still needs to be assigned.
func (m *Music) assignTrackReleaseDates() (bool, error) {
	var err error

	tracks := m.tracksWithoutAssignedReleaseDate()
	releaseCache := make(map[string]Release)

	for _, t := range tracks {
		r, ok := releaseCache[t.REID]
		if !ok {
			r, err = m.release(t.REID)
			if err != nil {
				log.Println("REID not found for", t.Artist, t.Release, t.REID)
				// REID not found, find new one
				r, err = m.findTrackRelease(t)
				if err != nil {
					// still not found, try harder
					r, err = m.findTrackReleaseDisambiguate(t)
					if err != nil {
						return false, err
					}
				}
			}
			releaseCache[t.REID] = r
		}
		// this will reassign REID & RGID as needed
		m.assignTrackRelease(t, r)
	}

	return len(tracks) > 0, nil
}

var cachedCountryMap map[string]int

func (m *Music) countryMap() map[string]int {
	if len(cachedCountryMap) == 0 {
		cachedCountryMap = make(map[string]int)
		for i, v := range m.config.Music.ReleaseCountries {
			cachedCountryMap[v] = i
		}
	}
	return cachedCountryMap
}

var unwantedDisambRegexp = regexp.MustCompile(`(exclusive|deluxe|edition)`)

func (m *Music) pickRelease(releases []Release) int {
	first, second, third, fourth := -1, -1, -1, -1
	firstRank := -1
	countryMap := m.countryMap()
	for i, r := range releases {
		rank, preferred := countryMap[r.Country]
		if preferred && r.FrontArtwork && r.Disambiguation == "" && r.Official() {
			if first == -1 || rank < firstRank {
				first = i
				firstRank = rank
			}
		} else if r.FrontArtwork && r.Disambiguation == "" && r.Official() {
			second = i
		} else if r.FrontArtwork && preferred {
			if unwantedDisambRegexp.MatchString(r.Disambiguation) {
				fourth = i
			} else {
				third = i
			}
		} else if r.FrontArtwork {
			fourth = i
		}
	}

	if first != -1 {
		return first
	} else if second != -1 {
		return second
	} else if third != -1 {
		return third
	} else if fourth != -1 {
		return fourth
	} else if len(releases) > 0 {
		return 0
	}
	return -1
}

func (m *Music) pickDisambiguation(t Track, releases []Release) int {
	countryMap := m.countryMap()
	first, second, third := -1, -1, -1
	firstRank := -1
	for i, r := range releases {
		name1 := fmt.Sprintf("%s (%s)", r.Name, r.Disambiguation)
		name2 := fmt.Sprintf("%s - %s", r.Name, r.Disambiguation)
		name3 := fmt.Sprintf("%s %s", r.Name, r.Disambiguation)
		name4 := fmt.Sprintf("%s [%s]", r.Name, r.Disambiguation)
		name5 := fmt.Sprintf("%s", r.Disambiguation)
		if strings.EqualFold(name1, t.Release) ||
			strings.EqualFold(name2, t.Release) ||
			strings.EqualFold(name3, t.Release) ||
			strings.EqualFold(name4, t.Release) ||
			strings.EqualFold(name5, t.Release) {
			rank, preferred := countryMap[r.Country]
			if preferred && r.FrontArtwork && r.Official() {
				if first == -1 || rank < firstRank {
					first = i
					firstRank = rank
				}
			} else if r.FrontArtwork {
				second = i
			} else {
				third = i
			}
		} //  else {
		// 	fmt.Print("no match %s\n", t.Release)
		// }
	}
	if first != -1 {
		return first
	} else if second != -1 {
		return second
	} else if third != -1 {
		return third
	}
	return -1
}

func (m *Music) findTrackRelease(t Track) (Release, error) {
	releases := m.trackReleases(t)
	pick := m.pickRelease(releases)
	if pick == -1 {
		return Release{}, ErrReleaseNotFound
	}
	return releases[pick], nil
}

// try using disambiguation
func (m *Music) findTrackReleaseDisambiguate(t Track) (Release, error) {
	releases := m.disambiguate(t.Artist, t.TrackCount, t.DiscCount)
	pick := m.pickDisambiguation(t, releases)
	if pick == -1 {
		return Release{}, ErrReleaseNotFound
	}
	return releases[pick], nil
}

// Fix track release names using various pattern matching and name variants.
func (m *Music) fixTrackReleases() (bool, error) {
	modified := false
	fixReleases := make(map[string]struct{})
	var fixTracks []map[string]interface{}
	//tracks := m.tracksWithoutReleases()
	tracks := m.tracksWithoutAssignedRelease()

	for _, t := range tracks {
		artist, err := m.Artist(t.Artist)
		if err != nil {
			log.Printf("artist not found: %s\n", t.Artist)
			continue
		}

		_, ok := fixReleases[t.Release]
		if ok {
			continue
		}

		releases := m.artistReleasesLike(artist, t.Release, t.TrackCount, t.DiscCount)
		// if len(releases) == 0 {
		// 	log.Printf("no releases for %s/%s/%d/%d\n",
		// 		artist.Name, t.Release, t.TrackCount, t.DiscCount)
		// }

		if len(releases) > 0 {
			pick := 0
			if len(releases) > 1 {
				pick = m.pickRelease(releases)
				if pick == -1 {
					log.Printf("pick failed for %s/%s/%d/%d\n",
						artist.Name, t.Release, t.TrackCount, t.DiscCount)
					continue
				}
			}
			r := releases[pick]
			fixReleases[t.Release] = struct{}{}
			fixTracks = append(fixTracks, map[string]interface{}{
				"artist":     artist.Name,
				"from":       t.Release,
				"to":         r.Name,
				"trackCount": r.TrackCount,
				"discCount":  r.DiscCount,
			})
		} else {
			releases = m.releases(artist)
			matched := false
			for _, r := range releases {
				// try fuzzy match
				if strings.EqualFold(FuzzyName(t.Release), FuzzyName(r.Name)) &&
					t.TrackCount == r.TrackCount {
					fixReleases[t.Release] = struct{}{}
					fixTracks = append(fixTracks, map[string]interface{}{
						"artist":     artist.Name,
						"from":       t.Release,
						"to":         r.Name,
						"trackCount": r.TrackCount,
						"discCount":  r.DiscCount,
					})
					matched = true
					break
				}
			}
			if !matched {
				// log.Printf("unmatched %s/%s/%d\n",
				// 	t.Artist, t.Release, t.TrackCount)
				fixReleases[t.Release] = struct{}{}
			}
		}
	}

	if len(fixReleases) > 0 {
		modified = true
	}

	for _, v := range fixTracks {
		err := m.updateTrackRelease(
			v["artist"].(string),
			v["from"].(string),
			v["to"].(string),
			v["trackCount"].(int),
			v["discCount"].(int))
		if err != nil {
			return modified, err
		}
	}

	return modified, nil
}

// Generate a ReleaseTitle for each track which in most cases will be the
// release name. In multi-disc sets the individual media may have a more
// specific name so that is included also.
func (m *Music) fixTrackReleaseTitles() error {
	artists := m.Artists()
	return m.fixTrackReleaseTitlesFor(artists)
}

func (m *Music) fixTrackReleaseTitlesFor(artists []Artist) error {
	for _, a := range artists {
		//log.Printf("release titles for %s\n", a.Name)
		releases := m.ArtistReleases(a)
		for _, r := range releases {
			media := m.releaseMedia(r)
			names := make(map[int]Media)
			for i := range media {
				names[media[i].Position] = media[i]
			}

			tracks := m.ReleaseTracks(r)
			for i := range tracks {
				var mediaTitle, releaseTitle string
				name := names[tracks[i].DiscNum].Name
				if name != "" && name != r.Name {
					mediaTitle = name
					// TODO make this format configureable
					releaseTitle =
						fmt.Sprintf("%s (%s)", name, r.Name)
				} else {
					mediaTitle = ""
					releaseTitle = r.Name
				}
				if mediaTitle != tracks[i].MediaTitle ||
					releaseTitle != tracks[i].ReleaseTitle {
					tracks[i].MediaTitle = mediaTitle
					tracks[i].ReleaseTitle = releaseTitle
					err := m.updateTrackReleaseTitles(tracks[i])
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Sync popular tracks for each artist from Last.fm.
func (m *Music) syncPopular() error {
	return m.syncPopularFor(m.Artists())
}

// Sync popular from lastfm or listenbrainz. This tries lastfm first and if no
// results (no api keys configured or error) will use listenbrainz.
func (m *Music) syncPopularFor(artists []Artist) error {
	for _, a := range artists {
		log.Printf("popular for %s\n", a.Name)
		count, err := m.syncLastfmPopular(a)
		if err != nil {
			return err
		}
		if count == 0 {
			_, err = m.syncListenBrainzPopular(a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type TopTrack interface {
	Track() string
	Rank() int
}

func (m *Music) syncLastfmPopular(a Artist) (int, error) {
	tracks := m.lastfm.ArtistTopTracks(a.ARID)
	if len(tracks) == 0 {
		return 0, nil
	}

	top := make([]TopTrack, len(tracks))
	for i := range tracks {
		top[i] = tracks[i]
	}
	err := m.doTopTracks(a, top)

	return len(tracks), err
}

func (m *Music) syncListenBrainzPopular(a Artist) (int, error) {
	tracks, err := m.lbz.ArtistTopTracks(a.ARID)
	if len(tracks) == 0 || err != nil {
		return 0, err
	}

	top := make([]TopTrack, len(tracks))
	for i := range tracks {
		top[i] = tracks[i]
	}
	err = m.doTopTracks(a, top)

	return len(tracks), err
}

func (m *Music) doTopTracks(a Artist, tracks []TopTrack) error {
	m.deletePopularFor(a.Name)
	for i, t := range tracks {
		if i == m.config.Music.PopularLimit {
			break
		}
		p := Popular{
			Artist: a.Name,
			Title:  t.Track(),
			Rank:   t.Rank(),
		}
		m.createPopular(&p)
	}
	return nil
}

// Sync similar artists for each artist from Last.fm.
func (m *Music) syncSimilar() error {
	return m.syncSimilarFor(m.Artists())
}

func (m *Music) syncSimilarFor(artists []Artist) error {
	for _, a := range artists {
		log.Printf("similar for %s\n", a.Name)
		rank := m.lastfm.SimilarArtists(a.ARID)
		if len(rank) == 0 {
			continue
		}
		// remove what we have now
		m.deleteSimilarFor(a.Name)

		mbids := make([]string, 0, len(rank))
		for k := range rank {
			mbids = append(mbids, k)
		}

		list := m.artistsByMBID(mbids)
		sort.Slice(list, func(i, j int) bool {
			return rank[list[i].ARID] > rank[list[j].ARID]
		})

		var similar []Similar
		for index, v := range list {
			similar = append(similar, Similar{
				Artist: a.Name,
				ARID:   v.ARID,
				Rank:   index,
			})
		}

		for _, s := range similar {
			// TODO how to check for specific error?
			// - UNIQUE constraint failed
			m.createSimilar(&s)
		}
	}
	return nil
}

func (m *Music) syncMissingArtwork() error {
	return m.checkMissingArtwork()
}

// Sync artwork from Fanart
func (m *Music) syncArtwork() error {
	return m.syncArtworkFor(m.Artists())
}

func (m *Music) syncArtworkFor(artists []Artist) error {
	for _, a := range artists {
		log.Printf("artwork for %s\n", a.Name)
		artwork := m.fanart.ArtistArt(a.ARID)
		if artwork == nil {
			continue
		}
		source := "fanart"
		for _, art := range artwork.ArtistBackgrounds {
			bg := ArtistBackground{
				Artist: a.Name,
				URL:    art.URL,
				Source: source,
				Rank:   str.Atoi(art.Likes),
			}
			m.createArtistBackground(&bg)
		}
		for _, art := range artwork.ArtistThumbs {
			img := ArtistImage{
				Artist: a.Name,
				URL:    art.URL,
				Source: source,
				Rank:   str.Atoi(art.Likes),
			}
			m.createArtistImage(&img)
		}
	}
	return nil
}

func (m *Music) resolve() error {
	// try to resolve any missing artists
	err := m.syncArtists()
	if err != nil {
		return err
	}
	// try to fix track releases
	_, err = m.fixTrackReleases()
	if err != nil {
		return err
	}
	// try to assign any unassigned releases
	_, err = m.assignTrackReleases()
	if err != nil {
		return err
	}
	// fix up any track release dates
	_, err = m.assignTrackReleaseDates()
	if err != nil {
		return err
	}
	// fix up titles
	err = m.fixTrackReleaseTitles()
	return err
}

// Get the artist names from tracks and try to find the correct artist
// from MusicBrainz. This doesn't always work since there can be
// multiple artists with the same name. Last.fm is used to help.
func (m *Music) syncArtists() error {
	artists, err := m.trackArtists()
	if err != nil {
		return err
	}

	for _, a := range artists {
		name, arid := a[0], a[1]
		_, err := m.syncArtist(name, arid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Music) syncArtist(name, arid string) (Artist, error) {
	var tags []ArtistTag
	artist, err := m.Artist(name)
	if err != nil {
		if arid != "" {
			// first try arid
			var v musicbrainz.Artist
			v, err = m.mbz.SearchArtistID(arid)
			if err == nil {
				artist, tags = doArtist(v)
			}
		}
		if err != nil {
			// next try name
			artist, tags, err = m.resolveArtist(name)
		}
	}

	if err != nil {
		err := errors.New(fmt.Sprintf("'%s' artist not found", name))
		log.Printf("%s\n", err)
		return artist, err
	}

	artist.Name = fixName(artist.Name)
	log.Printf("creating %s\n", artist.Name)
	m.createArtist(&artist)
	for _, t := range tags {
		t.Artist = artist.Name
		m.createArtistTag(&t)
	}

	if name != artist.Name {
		// fix track artist name: AC_DC -> AC/DC
		log.Printf("fixing name %s to %s\n", name, artist.Name)
		m.updateTrackAlbumArtist(name, artist.Name)
	}

	detail, err := m.mbz.ArtistDetail(artist.ARID)
	if err != nil {
		log.Printf("%s\n", err)
		return artist, nil // TODO ignore error?
	}
	artist.Disambiguation = detail.Disambiguation
	artist.Country = detail.Country
	artist.Area = detail.Area.Name
	artist.Date = date.ParseDate(detail.LifeSpan.Begin)
	artist.EndDate = date.ParseDate(detail.LifeSpan.End)
	artist.Genre = detail.PrimaryGenre()
	m.updateArtist(&artist)
	return artist, nil
}

// Try MusicBrainz and Last.fm to find an artist. Fortunately Last.fm
// will give up the ARID so MusicBrainz can still be used.
func (m *Music) resolveArtist(name string) (Artist, []ArtistTag, error) {
	var artist Artist
	var tags []ArtistTag
	var v musicbrainz.Artist

	err := ErrArtistNotFound
	arid, ok := m.config.Music.UserArtistID(name)
	if ok {
		v, err = m.mbz.SearchArtistID(arid)
		if err == nil {
			artist, tags = doArtist(v)
		}
	} else {
		v, err = m.mbz.SearchArtist(name)
		if err == nil {
			artist, tags = doArtist(v)
		}
	}
	if err != nil {
		// try again
		fuzzy := fuzzyArtist(name)
		if fuzzy != name {
			v, err = m.mbz.SearchArtist(fuzzy)
			if err == nil {
				artist, tags = doArtist(v)
			}
		}
	}
	if err != nil {
		// try lastfm
		lastName, lastID := m.lastfm.ArtistSearch(name)
		if lastName != "" && lastID != "" {
			log.Printf("try lastfm got %s mbid:'%s'\n", lastName, lastID)
			v, err = m.mbz.SearchArtistID(lastID)
			if err == nil {
				artist, tags = doArtist(v)
			}
		}
	}
	return artist, tags, err
}

func (m *Music) findRelease(rgid string, trackCount int) (string, error) {
	group, err := m.mbz.ReleaseGroup(rgid)
	if err != nil {
		return "", err
	}
	for _, r := range group.Releases {
		log.Printf("find %d vs %d\n", r.TotalTracks(), trackCount)
		if r.TotalTracks() == trackCount {
			return r.ID, nil
		}
	}
	return "", errors.New("release not found")
}

func (m *Music) releaseIndex(release Release) (search.IndexMap, error) {
	var err error
	tracks := m.ReleaseTracks(release)

	reid := release.REID
	if reid == "" {
		// is this still needed?
		reid, err = m.findRelease(release.RGID, len(tracks))
		if err != nil {
			return nil, err
		}
	}

	indices, err := m.creditsIndex(reid)
	if err != nil {
		return nil, err
	}

	newIndex := make(search.IndexMap)
	for _, index := range indices {
		matched := false
		for _, t := range tracks {
			if t.DiscNum == index.DiscNum &&
				t.TrackNum == index.TrackNum {
				// use track key
				newIndex[t.Key] = index.Fields
				if t.Title != index.Title {
					m.updateTrackTitle(t, index.Title)
				}
				if index.RID != "" {
					m.updateTrackRID(t, index.RID)
				}
				if index.Artist != "" {
					m.assignTrackArtist(t, index.Artist)
				}
				matched = true
			}
		}
		if !matched {
			// likely video discs
			log.Printf("no match %d/%d/%s\n",
				index.DiscNum, index.TrackNum, index.Title)
		}
	}

	// Popular artist tracks mapped to the first release where the tracks
	// appeared. If this is that release, add popularty fields for those
	// tracks below.
	popularityMap := make(map[string]int)
	a, err := m.Artist(release.Artist)
	if err == nil {
		for rank, t := range m.ArtistPopularTracks(a) {
			popularityMap[t.Key] = rank + 1
		}
	}

	// update type field with single
	singles := make(map[string]bool)
	for _, t := range m.ReleaseSingles(release) {
		singles[t.Key] = true
	}
	for k := range singles {
		fields, ok := newIndex[k]
		if ok {
			addField(fields, FieldType, TypeSingle)
		}
	}

	// update type field with popular
	popular := make(map[string]bool)
	for _, t := range m.ReleasePopular(release) {
		popular[t.Key] = true
	}
	for k := range popular {
		fields, ok := newIndex[k]
		if ok {
			addField(fields, FieldType, TypePopular)

			rank, pop := popularityMap[k]
			if pop {
				// add popularity rank
				//log.Printf("popularity %s -> %d", k, rank)
				addField(fields, FieldPopularity, rank)
			}
		}
	}

	return newIndex, nil
}

func (m *Music) artistIndex(a Artist) ([]search.IndexMap, error) {
	var indices []search.IndexMap
	releases := m.ArtistReleases(a)
	//log.Printf("got %d releases\n", len(releases))
	for _, r := range releases {
		//log.Printf("%s\n", r.Name)
		index, err := m.releaseIndex(r)
		if err != nil {
			return indices, err
		}
		indices = append(indices, index)
	}
	return indices, nil
}

func (m *Music) syncIndexFor(artists []Artist) error {
	s, err := m.newSearch()
	if err != nil {
		return err
	}
	defer s.Close()

	for _, a := range artists {
		log.Printf("index for %s\n", a.Name)
		index, err := m.artistIndex(a)
		//log.Println(index, err)
		if err != nil {
			return err
		}
		for _, idx := range index {
			s.Index(idx)
		}
	}
	return nil
}

func (m *Music) syncIndex() error {
	artists := m.Artists()
	return m.syncIndexFor(artists)
}

func doArtist(artist musicbrainz.Artist) (a Artist, tags []ArtistTag) {
	a = Artist{
		Name:     artist.Name,
		SortName: artist.SortName,
		ARID:     string(artist.ID)}
	for _, t := range artist.Tags {
		at := ArtistTag{
			Artist: a.Name,
			Tag:    t.Name,
			Count:  t.Count}
		tags = append(tags, at)
	}
	return
}

// MusicBrainz has release tiles that are primarily singles which are
// multi-title, generally side-a / side-b. Some even have 4 or 5 titles,
// separated by slash.
//
// title / title [ / title ... ]
func singleNames(name string) []string {
	names := strings.Split(name, " / ")
	return names
}

func doRelease(artist string, r musicbrainz.Release) Release {
	disambiguation := r.Disambiguation
	if disambiguation == "" {
		disambiguation = r.ReleaseGroup.Disambiguation
	}

	var media []Media
	for _, m := range r.Media {
		media = append(media, Media{
			REID:       string(r.ID),
			Name:       m.Title,
			Position:   m.Position,
			Format:     m.Format,
			TrackCount: m.TrackCount})
	}

	var singleName string
	if r.ReleaseGroup.PrimaryType == musicbrainz.PrimaryTypeSingle {
		singleName = r.Title
		// try to get a primary single title name
		// title a / title b will yield "title a"
		names := singleNames(singleName)
		if names[0] != singleName {
			singleName = names[0]
		}
	}

	return Release{
		Artist:         artist,
		Name:           r.Title,
		Disambiguation: disambiguation,
		REID:           string(r.ID),
		RGID:           string(r.ReleaseGroup.ID),
		Type:           r.ReleaseGroup.PrimaryType,
		SecondaryType:  r.ReleaseGroup.SecondaryType(),
		Asin:           r.Asin,
		Country:        r.Country,
		TrackCount:     r.TotalTracks(),
		DiscCount:      r.TotalDiscs(),
		Artwork:        r.CoverArtArchive.Artwork,
		FrontArtwork:   r.CoverArtArchive.Front,
		BackArtwork:    r.CoverArtArchive.Back,
		Media:          media,
		Date:           r.ReleaseGroup.FirstReleaseTime(),
		ReleaseDate:    date.ParseDate(r.Date),
		Status:         r.Status,
		SingleName:     singleName,
	}
}

func (m *Music) SyncCovers(c client.Getter) error {
	return m.syncCoversFor(c, m.Artists())
}

func (m *Music) syncCoversFor(client client.Getter, artists []Artist) error {
	for _, a := range artists {
		releases := m.ArtistReleases(a)
		for _, r := range releases {
			img := CoverArtArchiveImage(r)
			if img != "" {
				log.Printf("sync %s/%s %s\n", a.Name, r.Name, img)
				client.Get(img)
			}
		}
	}
	return nil
}

func (m *Music) SyncFanArt(c client.Getter) error {
	return m.syncFanArtFor(c, m.Artists())
}

func (m *Music) syncFanArtFor(client client.Getter, artists []Artist) error {
	for _, a := range artists {
		thumbs := m.artistImages(a)
		for _, img := range thumbs {
			log.Printf("sync %s thumb %s\n", a.Name, img)
			client.Get(img)
		}
		bgs := m.artistBackgrounds(a)
		for _, img := range bgs {
			log.Printf("sync %s bg %s\n", a.Name, img)
			client.Get(img)
		}
	}
	return nil
}
