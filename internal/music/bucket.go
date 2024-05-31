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
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/dhowden/tag"
	"github.com/dhowden/tag/mbz"
	"github.com/takeoutfm/takeout/lib/bucket"
	"github.com/takeoutfm/takeout/lib/str"
	. "github.com/takeoutfm/takeout/model"
)

// Asynchronously obtain all tracks from the bucket.
func (m *Music) syncFromBucket(bucket bucket.Bucket, lastSync time.Time) (trackCh chan *Track, err error) {
	trackCh = make(chan *Track)

	go func() {
		defer close(trackCh)
		objectCh, err := bucket.List(lastSync)
		if err != nil {
			return
		}
		for o := range objectCh {
			checkObject(bucket, o, trackCh)
		}
	}()

	return
}

func checkObject(b bucket.Bucket, object *bucket.Object, trackCh chan *Track) {
	t := &Track{
		Key:          object.Key,
		ETag:         object.ETag,
		Size:         object.Size,
		LastModified: object.LastModified,
	}

	if b.IsLocal() {
		url := b.ObjectURL(t.Key)
		err := parseMetadata(url, t)
		if err == nil {
			trackCh <- t
			return
		}
		// failed so try regexps
	}

	matchPath(b, object.Path, t, trackCh, func(t *Track, trackCh chan *Track) {
		trackCh <- t
	})
}

// Examples:
// The Raconteurs / Help Us Stranger (2019) / 01-Bored and Razed.flac
// Tubeway Army / Replicas - The First Recordings (2019) / 1-01-You Are in My Vision (early version).flac
// Tubeway Army / Replicas - The First Recordings (2019) / 2-01-Replicas (early version 2).flac
var coverRegexp = regexp.MustCompile(`cover\.(png|jpg)$`)

var pathRegexp = regexp.MustCompile(`([^\/]+)\/([^\/]+)\/([^\/]+)$`)

func matchPath(b bucket.Bucket, path string, t *Track, trackCh chan *Track,
	doMatch func(t *Track, music chan *Track)) {
	matches := pathRegexp.FindStringSubmatch(path)
	if matches != nil {
		t.Artist = matches[1]
		release, date := matchRelease(matches[2])
		if release != "" && date != "" {
			t.Release = release
			t.Date = date
		} else {
			t.Release = release
		}
		if matchTrack(matches[3], t) {
			doMatch(t, trackCh)
		}
	}
}

var releaseRegexp = regexp.MustCompile(`(.+?)\s*(\(([\d]+)\))?\s*$`)

// 1|1|Airlane|Music/Gary Numan/The Pleasure Principle (1998)/01-Airlane.flac
// 1|1|Airlane|Music/Gary Numan/The Pleasure Principle (2009)/1-01-Airlane.flac
//
// The Pleasure Principle
// 1: The Pleasure Principle
//
// The Pleasure Principle (2000)
// 1: The Pleasure Principle
// 2: (2000)
// 3: 2000
//
// The Pleasure Principle (Live)
// 1: The Pleasure Principle (Live)
//
// The Pleasure Principle (Live) (2000)
// 1: The Pleasure Principle (Live)
// 2: (2000)
// 3: 2000
func matchRelease(release string) (string, string) {
	var name, date string
	matches := releaseRegexp.FindStringSubmatch(release)
	if matches != nil {
		if len(matches) == 2 {
			name = matches[1]
		} else if len(matches) == 4 {
			name = matches[1]
			date = matches[3]
		}
	}
	return name, date
}

var trackRegexp = regexp.MustCompile(`(?:([1-9]+[0-9]?)-)?([\d]+)-(.+)\.(mp3|flac|ogg|m4a)$`)
var singleDiscRegexp = regexp.MustCompile(`([\d]+)-([^+]+)\.(mp3|flac|ogg|m4a)$`)
var numericRegexp = regexp.MustCompile(`^[\d\s-]+$`)

func matchTrack(file string, t *Track) bool {
	matches := trackRegexp.FindStringSubmatch(file)
	if matches == nil {
		return false
	}
	disc := str.Atoi(matches[1])
	track := str.Atoi(matches[2])
	t.DiscNum = disc
	t.TrackNum = track
	t.Title = matches[3]
	if t.DiscNum == 0 {
		t.DiscNum = 1
	}

	// potentially not multi-disc so assume single disc if too many
	// TODO make this configurable?
	// eg: 18-19-2000 (Soulchild remix).flac
	// Beatles in Mono - 13 discs
	// Eagles Legacy - 12 discs
	// Kraftwerk The Catalogue - 8 discs
	if t.DiscNum > 13 {
		matches := singleDiscRegexp.FindStringSubmatch(file)
		if matches == nil {
			return false
		}
		t.DiscNum = 1
		t.TrackNum = str.Atoi(matches[1])
		t.Title = matches[2]
	}

	// all numeric assume is single disc since most are single
	// eg: 11-19-2000.flac
	// eg: 4-36-22-36.flac
	// but 2-02-1993.flac is not a single disc track
	if numericRegexp.MatchString(t.Title) {
		matches := singleDiscRegexp.FindStringSubmatch(file)
		if matches == nil {
			return false
		}
		t.DiscNum = 1
		t.TrackNum = str.Atoi(matches[1])
		t.Title = matches[2]
	}

	return true
}

// Generate a presigned url which expires based on config settings.
func (m *Music) bucketURL(t *Track) *url.URL {
	// TODO FIXME assume first bucket!!!
	return m.buckets[0].ObjectURL(t.Key)
}

func parseMetadata(u *url.URL, t *Track) error {
	if u.Scheme != "file" {
		panic("scheme not supported")
	}

	path := u.RawPath
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	m, err := tag.ReadFrom(file)
	if err != nil {
		return err
	}

	t.Artist = m.AlbumArtist()
	t.Release = m.Album()
	t.Title = m.Title()
	t.TrackArtist = m.Artist()
	t.TrackNum, t.TrackCount = m.Track()
	t.DiscNum, t.DiscCount = m.Disc()

	info := mbz.Extract(m)
	t.RID = info.Get(mbz.Recording)
	t.RGID = info.Get(mbz.ReleaseGroup)
	t.REID = info.Get(mbz.Album)
	t.ARID = info.Get(mbz.AlbumArtist)

	return nil
}
