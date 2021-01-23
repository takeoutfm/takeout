// Copyright (C) 2020 The Takeout Authors.
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

package music

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/defsub/takeout/log"
	"github.com/defsub/takeout/search"
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
			log.CheckError(m.fixTrackReleases())
			log.Printf("assign track releases\n")
			log.CheckError(m.assignTrackReleases())
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
		if options.Tracks {
			modified, err := m.syncBucketTracksSince(options.Since)
			log.CheckError(err)
			if modified {
				log.CheckError(m.syncArtists())
			}
		}
		var artists []Artist
		if options.Artist != "" {
			artists = []Artist{*m.artist(options.Artist)}
		} else {
			artists = m.trackArtistsSince(options.Since)
		}
		if options.Releases {
			log.CheckError(m.syncReleasesFor(artists))
			log.CheckError(m.fixTrackReleases())
			log.CheckError(m.assignTrackReleases())
			log.CheckError(m.fixTrackReleaseTitles())
		}
		if options.Popular {
			log.CheckError(m.syncPopularFor(artists))
		}
		if options.Similar {
			log.CheckError(m.syncSimilarFor(artists))
		}
		if options.Artwork {
			log.CheckError(m.syncArtworkFor(artists))
			log.CheckError(m.syncMissingArtwork())
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
	_, err :=  m.syncBucketTracksSince(time.Time{})
	return err
}

func (m *Music) syncBucketTracksSince(lastSync time.Time) (modified bool, err error) {
	trackCh, err := m.syncFromBucket(lastSync)
	if err != nil {
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
		a := m.artist(t.Artist)
		if a != nil {
			artists = append(artists, *a)
		}

	}
	return artists
}

// Obtain all releases for each track artist. This will update
// existing releases as well.
func (m *Music) syncReleases() error {
	return m.syncReleasesFor(m.artists())
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
				result, _ := m.MusicBrainzSearchReleaseGroup(a.ARID, name)
				for _, rg := range result.ReleaseGroups {
					r, _ := m.MusicBrainzReleases(&a, rg.ID)
					releases = append(releases, r...)
				}
			}
		} else {
			releases, _ = m.MusicBrainzArtistReleases(&a)
		}
		for _, r := range releases {
			r.Name = fixName(r.Name)
			for i := range r.Media {
				r.Media[i].Name = fixName(r.Media[i].Name)
			}

			curr, _ := m.release(r.REID)
			if curr == nil {
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
				err := m.replaceRelease(curr, &r)
				if err != nil {
					log.Println(err)
					return err
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
		err := m.checkReleaseArtwork(&r)
		if err != nil {
			log.Printf("err was %s\n", err)
		}
	}
	return nil
}

func (m *Music) checkReleaseArtwork(r *Release) error {
	if r.Artwork && r.FrontArtwork == false {
		log.Printf("need artwork for %s / %s\n", r.Artist, r.Name)
		// have artwork but no front cover
		art, err := m.coverArtArchive(r.REID)
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
		art, err := m.coverArtArchive(r.REID)
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
		err = m.updateArtwork(r, front, back)
		if err != nil {
			return err
		}
	}
	return nil
}

func fuzzyArtist(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9& -]`)
	return re.ReplaceAllString(name, "")
}

func fuzzyName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return re.ReplaceAllString(name, "")
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

// Assign a track to a specific MusicBrainz REID. This isn't exact and
// instead will pick the first release with the same name with the
// same number of tracks. This way original release dates are
// presented to the user. An attempt is also made to match using
// disambiguations, things like:
// Weezer:
//   Weezer (Blue Album)
//   Weezer - Blue Album
//   Weezer Blue Album
//   Weezer [Blue Album]
//   Blue Album
// David Bowie:
//   ★ (Blackstar)
//   Blackstar
func (m *Music) assignTrackReleases() error {
	notfound := make(map[string]int)
	artChecked := make(map[string]bool)
	tracks := m.tracksWithoutAssignedRelease()
	// TODO this could be more efficient
	for _, t := range tracks {
		var assignedRelease *Release
		// TODO prefer no disamb., and countries, and CDs
		r := m.trackRelease(&t)
		if r == nil {
			// try using disambiguation
			releases := m.disambiguate(t.Artist, t.TrackCount, t.DiscCount)
			for _, r := range releases {
				if r.Disambiguation != "" {
					name1 := fmt.Sprintf("%s (%s)", r.Name, r.Disambiguation)
					name2 := fmt.Sprintf("%s - %s", r.Name, r.Disambiguation)
					name3 := fmt.Sprintf("%s %s", r.Name, r.Disambiguation)
					name4 := fmt.Sprintf("%s [%s]", r.Name, r.Disambiguation)
					name5 := fmt.Sprintf("%s", r.Disambiguation)
					if name1 == t.Release || name2 == t.Release ||
						name3 == t.Release || name4 == t.Release ||
						name5 == t.Release {
						err := m.assignTrackRelease(&t, &r)
						if err != nil {
							return err
						}
						assignedRelease = &r
						break
					}
				}
			}
			if assignedRelease == nil {
				v := fmt.Sprintf("%s/%s/%d", t.Artist, t.Release, t.TrackCount)
				notfound[v] += 1
				if notfound[v] == 1 {
					log.Printf("release not found for %s\n", v)
				}
			}
		} else {
			err := m.assignTrackRelease(&t, r)
			if err != nil {
				return err
			}
			assignedRelease = r
		}

		if assignedRelease != nil {
			// ensure releases assigned to tracks have artwork
			if _, ok := artChecked[assignedRelease.REID]; !ok {
				err := m.checkReleaseArtwork(assignedRelease)
				if err != nil {
					log.Println(err)
					return err
				}
				artChecked[assignedRelease.REID] = true
			}
		}

	}
	return nil
}

// Fix track release names using various pattern matching and name variants.
func (m *Music) fixTrackReleases() error {
	fixReleases := make(map[string]bool)
	var fixTracks []map[string]interface{}
	//tracks := m.tracksWithoutReleases()
	tracks := m.tracksWithoutAssignedRelease()

	for _, t := range tracks {
		artist := m.artist(t.Artist)
		if artist == nil {
			log.Printf("artist not found: %s\n", t.Artist)
			continue
		}

		_, ok := fixReleases[t.Release]
		if ok {
			continue
		}

		releases := m.artistReleasesLike(artist, t.Release, t.TrackCount, t.DiscCount)
		// for _, r := range releases {
		// 	log.Printf("check %s/%s/%d/%d\n", r.Artist, r.Name, r.TrackCount, t.DiscCount)
		// }
		if len(releases) == 0 {
			log.Printf("no releases for %s/%s/%d/%d\n",
				artist.Name, t.Release, t.TrackCount, t.DiscCount)
		}

		if len(releases) > 0 {
			if len(releases) > 1 {
				// more than one matching release so reorder
				// releases by country ordered user preference;
				// lower is better
				lowest := math.MaxUint32
				priority := make(map[string]int)
				for i, v := range m.config.Music.ReleaseCountries {
					priority[v] = i
				}
				sort.Slice(releases, func(i, j int) bool {
					var p1, p2 int
					if p1, ok = priority[releases[i].Country]; !ok {
						p1 = lowest
					}
					if p2, ok = priority[releases[j].Country]; !ok {
						p2 = lowest
					}
					return p1 < p2
				})
				// log.Printf("picking [%s] %s/%s/%d/%d\n",
				// 	releases[0].Artist, releases[0].Name,
				// 	releases[0].TrackCount, releases[0].DiscCount)
			}
			fixReleases[t.Release] = true
			fixTracks = append(fixTracks, map[string]interface{}{
				"artist":     artist.Name,
				"from":       t.Release,
				"to":         releases[0].Name,
				"trackCount": releases[0].TrackCount,
				"discCount":  releases[0].DiscCount,
			})
		} else {
			releases = m.releases(artist)
			matched := false
			for _, r := range releases {
				// try fuzzy match
				if fuzzyName(t.Release) == fuzzyName(r.Name) &&
					t.TrackCount == r.TrackCount {
					fixReleases[t.Release] = true
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
				fixReleases[t.Release] = false
			}
		}
	}

	for _, v := range fixTracks {
		err := m.updateTrackRelease(
			v["artist"].(string),
			v["from"].(string),
			v["to"].(string),
			v["trackCount"].(int),
			v["discCount"].(int))
		if err != nil {
			return err
		}
	}

	return nil
}

// Generate a ReleaseTitle for each track which in most cases will be the
// release name. In multi-disc sets the individual media may have a more
// specific name so that is included also.
func (m *Music) fixTrackReleaseTitles() error {
	artists := m.artists()
	for _, a := range artists {
		//log.Printf("release titles for %s\n", a.Name)
		releases := m.artistReleases(&a)
		for _, r := range releases {
			media := m.releaseMedia(r)
			names := make(map[int]Media)
			for i := range media {
				names[media[i].Position] = media[i]
			}

			tracks := m.releaseTracks(r)
			for i := range tracks {
				name := names[tracks[i].DiscNum].Name
				if name != "" && name != r.Name {
					tracks[i].MediaTitle = name
					// TODO make this format configureable
					tracks[i].ReleaseTitle =
						fmt.Sprintf("%s (%s)", name, r.Name)
				} else {
					tracks[i].MediaTitle = ""
					tracks[i].ReleaseTitle = r.Name
				}
				err := m.updateTrackReleaseTitles(tracks[i])
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Sync popular tracks for each artist from Last.fm.
func (m *Music) syncPopular() error {
	return m.syncPopularFor(m.artists())
}

func (m *Music) syncPopularFor(artists []Artist) error {
	for _, a := range artists {
		log.Printf("popular for %s\n", a.Name)
		popular := m.lastfmArtistTopTracks(&a)
		for _, p := range popular {
			// TODO how to check for specific error?
			// - UNIQUE constraint failed
			m.createPopular(&p)
		}
	}
	return nil
}

// Sync similar artists for each artist from Last.fm.
func (m *Music) syncSimilar() error {
	return m.syncSimilarFor(m.artists())
}

func (m *Music) syncSimilarFor(artists []Artist) error {
	for _, a := range artists {
		log.Printf("similar for %s\n", a.Name)
		similar := m.lastfmSimilarArtists(&a)
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
	return m.syncArtworkFor(m.artists())
}

func (m *Music) syncArtworkFor(artists []Artist) error {
	for _, a := range artists {
		log.Printf("artwork for %s\n", a.Name)
		artwork := m.fanartArtistArt(&a)
		if artwork == nil {
			continue
		}
		source := "fanart"
		for _, art := range artwork.ArtistBackgrounds {
			bg := ArtistBackground{
				Artist: a.Name,
				URL:    art.URL,
				Source: source,
				Rank:   atoi(art.Likes),
			}
			m.createArtistBackground(&bg)
		}
		for _, art := range artwork.ArtistThumbs {
			img := ArtistImage{
				Artist: a.Name,
				URL:    art.URL,
				Source: source,
				Rank:   atoi(art.Likes),
			}
			m.createArtistImage(&img)
		}
	}
	return nil
}

// Get the artist names from tracks and try to find the correct artist
// from MusicBrainz. This doesn't always work since there can be
// multiple artists with the same name. Last.fm is used to help.
func (m *Music) syncArtists() error {
	artists := m.trackArtistNames()
	for _, name := range artists {
		var tags []ArtistTag
		artist := m.artist(name)
		if artist == nil {
			artist, tags = m.resolveArtist(name)
			if artist != nil {
				artist.Name = fixName(artist.Name)
				log.Printf("creating %s\n", artist.Name)
				m.createArtist(artist)
				for _, t := range tags {
					t.Artist = artist.Name
					m.createArtistTag(&t)
				}
			}
		}

		if artist == nil {
			err := errors.New(fmt.Sprintf("'%s' artist not found", name))
			log.Printf("%s\n", err)
			continue
		}

		if name != artist.Name {
			// fix track artist name: AC_DC -> AC/DC
			log.Printf("fixing name %s to %s\n", name, artist.Name)
			m.updateTrackArtist(name, artist.Name)
		}

		detail, err := m.MusicBrainzArtistDetail(artist)
		if err != nil {
			log.Printf("%s\n", err)
			continue
		}
		artist.Disambiguation = detail.Disambiguation
		artist.Country = detail.Country
		artist.Area = detail.Area.Name
		artist.Date = parseDate(detail.LifeSpan.Begin)
		artist.EndDate = parseDate(detail.LifeSpan.End)
		if len(detail.Genres) > 0 {
			sort.Slice(detail.Genres, func(i, j int) bool {
				return detail.Genres[i].Count > detail.Genres[j].Count
			})
			artist.Genre = detail.Genres[0].Name
		}
		m.updateArtist(artist)
	}
	return nil
}

// Try MusicBrainz and Last.fm to find an artist. Fortunately Last.fm
// will give up the ARID so MusicBrainz can still be used.
func (m *Music) resolveArtist(name string) (artist *Artist, tags []ArtistTag) {
	arid, ok := m.config.Music.UserArtistID(name)
	if ok {
		artist, tags = m.MusicBrainzSearchArtistID(arid)
	} else {
		artist, tags = m.MusicBrainzSearchArtist(name)
	}
	if artist == nil {
		// try again
		fuzzy := fuzzyArtist(name)
		if fuzzy != name {
			artist, tags = m.MusicBrainzSearchArtist(fuzzy)
		}
	}
	if artist == nil {
		// try lastfm
		artist = m.lastfmArtistSearch(name)
		if artist != nil {
			log.Printf("try lastfm got %s mbid:'%s'\n", artist.Name, artist.ARID)
			// resolve with mbz
			if artist.ARID != "" {
				artist, tags = m.MusicBrainzSearchArtistID(artist.ARID)
			} else {
				artist = nil
			}
		}
	}
	return
}

func (m *Music) findRelease(rgid string, trackCount int) (string, error) {
	group, err := m.MusicBrainzReleaseGroup(rgid)
	//log.Printf("got %+v\n", group)
	if err != nil {
		return "", err
	}
	for _, r := range group.Releases {
		log.Printf("find %d vs %d\n", r.totalTracks(), trackCount)
		if r.totalTracks() == trackCount {
			return r.ID, nil
		}
	}
	return "", errors.New("release not found")
}

func (m *Music) releaseIndex(release Release) (search.IndexMap, error) {
	var err error
	tracks := m.releaseTracks(release)

	reid := release.REID
	if reid == "" {
		// is this still needed?
		reid, err = m.findRelease(release.RGID, len(tracks))
		if err != nil {
			return nil, err
		}
	}

	index, err := m.creditsIndex(reid)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^(\d+)-(\d+)-(.+)$`)
	newIndex := make(search.IndexMap)
	for k, v := range index {
		matches := re.FindStringSubmatch(k)
		if matches == nil {
			log.Printf("no re match %s\n", k)
			continue
		}
		discNum := atoi(matches[1])
		trackNum := atoi(matches[2])
		trackTitle := matches[3]
		matched := false
		for _, t := range tracks {
			if t.DiscNum == discNum &&
				t.TrackNum == trackNum {
				// use track key
				newIndex[t.Key] = v
				if t.Title != trackTitle {
					m.updateTrackTitle(t, trackTitle)
				}
				matched = true
			}
		}
		if !matched {
			// likely video discs
			log.Printf("no match %s\n", k)
		}
	}

	// Popular artist tracks mapped to the first release where the tracks
	// appeared. If this is that release, add popularty fields for those
	// tracks below.
	popularityMap := make(map[string]int)
	a := m.artist(release.Artist)
	if a != nil {
		for rank, t := range m.artistPopularTracks(*a) {
			popularityMap[t.Key] = rank + 1
		}
	}

	// update type field with single
	singles := make(map[string]bool)
	for _, t := range m.releaseSingles(release) {
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
	for _, t := range m.releasePopular(release) {
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

	// use first track release date for tracks index
	for k, v := range newIndex {
		tracks := m.tracksFor([]string{k})
		date, err := m.trackFirstReleaseDate(&tracks[0])
		if err != nil {
			continue
		}
		s := fmt.Sprintf("%4d-%02d-%02d", date.Year(), date.Month(), date.Day())
		addField(v, FieldDate, s)
	}

	return newIndex, nil
}

func (m *Music) artistIndex(a *Artist) ([]search.IndexMap, error) {
	var indices []search.IndexMap
	releases := m.artistReleases(a)
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
	s := m.newSearch()
	defer s.Close()

	for _, a := range artists {
		log.Printf("index for %s\n", a.Name)
		index, err := m.artistIndex(&a)
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
	artists := m.artists()
	return m.syncIndexFor(artists)
}
