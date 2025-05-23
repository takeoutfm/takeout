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
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/auth"
	. "takeoutfm.dev/takeout/model"
)

var (
	ErrTrackNotFound    = errors.New("track not found")
	ErrArtistNotFound   = errors.New("artist not found")
	ErrReleaseNotFound  = errors.New("release not found")
	ErrPlaylistNotFound = errors.New("playlist not found")
	ErrStationNotFound  = errors.New("station not found")
)

func (m *Music) openDB() (err error) {
	cfg := m.config.Music.DB.GormConfig()

	switch m.config.Music.DB.Driver {
	case "sqlite3":
		m.db, err = gorm.Open(sqlite.Open(m.config.Music.DB.Source), cfg)
	case "mysql":
		m.db, err = gorm.Open(mysql.Open(m.config.Music.DB.Source), cfg)
	case "postgres":
		// postgres untested
		m.db, err = gorm.Open(postgres.Open(m.config.Music.DB.Source), cfg)
	default:
		err = errors.New("driver not supported")
	}

	if err != nil {
		return
	}

	m.db.AutoMigrate(&Artist{}, &ArtistBackground{}, &ArtistImage{}, &ArtistTag{}, &Media{}, &Playlist{},
		&Popular{}, &Similar{}, &Station{}, &Release{}, &Track{})
	return
}

func (m *Music) closeDB() {
	conn, err := m.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

func (m *Music) lastModified() time.Time {
	var tracks []Track
	m.db.Order("last_modified desc").Limit(1).Find(&tracks)
	if len(tracks) == 1 {
		return tracks[0].LastModified
	} else {
		return time.Time{}
	}
}

func (m *Music) deleteTracks() {
	m.db.Exec("delete from tracks")
}

func (m *Music) createTrack(track *Track) error {
	return m.db.Create(track).Error
}

// Find an artist by name.
func (m *Music) Artist(artist string) (Artist, error) {
	var a Artist
	err := m.db.Where("name = ?", artist).First(&a).Error
	if err != nil {
		return Artist{}, ErrArtistNotFound
	}
	return a, nil
}

// Find an artist by name.
func (m *Music) ArtistLike(artist string) (Artist, error) {
	var a Artist
	err := m.db.Where("name like ?", artist).First(&a).Error
	if err != nil {
		return Artist{}, ErrArtistNotFound
	}
	return a, nil
}

func (m *Music) ArtistsForARIDs(arids []string) []Artist {
	var artists []Artist
	m.db.Where("ar_id in (?)", arids).Find(&artists)
	return artists
}

// Compute and update TrackCount for each track with total number of
// tracks in the associated release/album. This helps to match up
// MusicBrainz releases with tracks, especially with non-exact
// matches.
func (m *Music) updateTrackCount() error {
	rows, err := m.db.Table("tracks").
		Select("artist, `release`, date, count(title), max(disc_num)").
		Group("artist, `release`, date").
		Order("artist, `release`").Rows()
	if err != nil {
		return err
	}
	var results []map[string]interface{}
	for rows.Next() {
		var artist, release, date string
		var trackCount, discCount int
		rows.Scan(&artist, &release, &date, &trackCount, &discCount)
		results = append(results, map[string]interface{}{
			"artist":     artist,
			"release":    release,
			"date":       date,
			"trackCount": trackCount,
			"discCount":  discCount,
		})
	}
	rows.Close()

	for _, v := range results {
		err = m.db.Table("tracks").
			Where("artist = ? and `release` = ? and date = ?", v["artist"], v["release"], v["date"]).
			Updates(Track{TrackCount: v["trackCount"].(int), DiscCount: v["discCount"].(int)}).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// Tracks may have artist names that are modified to meet
// file/directory naming limitations. Update track entries with these
// modified names to the actual artist name.
func (m *Music) updateTrackAlbumArtist(oldName, newName string) (err error) {
	var tracks []Track
	m.db.Where("artist = ?", oldName).Find(&tracks)
	for _, t := range tracks {
		err = m.db.Model(t).Update("artist", newName).Error
		if err != nil {
			break
		}
	}
	return
}

// Tracks may have release names that are modified to meet file/directory
// naming limitations. Update the track entries with these modified names to
// the actual release name.  Also fix disc counts.
func (m *Music) updateTrackRelease(artist, oldName, newName, date string,
	trackCount, discCount int) (err error) {
	var tracks []Track
	m.db.Where("artist = ? and `release` = ? and date = ? and track_count = ?",
		artist, oldName, date, trackCount).Find(&tracks)
	for _, t := range tracks {
		err = m.db.Model(t).
			Update("release", newName).
			Update("disc_count", discCount).Error
		// fmt.Printf("update %s/%s/%s %s %d\n", artist, oldName, date, newName, discCount)
		if err != nil {
			break
		}
	}
	return
}

func (m *Music) artistTrackReleases(artist string) []string {
	var tracks []Track
	m.db.Select("distinct(release)").Where("artist = ?", artist).Find(&tracks)
	var releases []string
	for _, t := range tracks {
		releases = append(releases, t.Release)
	}
	return releases
}

func (m *Music) updateTrackReleaseTitles(t Track) error {
	return m.db.Model(t).
		Update("media_title", t.MediaTitle).
		Update("release_title", t.ReleaseTitle).Error
}

func (m *Music) tracksAddedSince(t time.Time) []Track {
	var tracks []Track
	m.db.Where("last_modified > ?", t).Find(&tracks)
	return tracks
}

// Find all tracks without a corresponding release - same artist,
// release name and track count. These likely have the wrong artist or
// release name.
func (m *Music) tracksWithoutReleases() []Track {
	var tracks []Track
	m.db.Where("not exists" +
		" (select releases.name from releases where" +
		" releases.artist = tracks.artist and releases.name = tracks.release" +
		" and releases.track_count = tracks.track_count)").
		Find(&tracks)
	return tracks
}

// Return a best guess at the media for this track
func (m *Music) trackMedia(t Track) ([]Media, error) {
	var media []Media
	rows, err := m.db.Table("tracks").
		Select("max(track_num), disc_num").
		Where("artist = ? and release = ? and date = ? and disc_count = ? and track_count = ?",
			t.Artist, t.Release, t.Date, t.DiscCount, t.TrackCount).
		Group("disc_num").
		Order("disc_num").Rows()
	if err != nil {
		return media, err
	}
	for rows.Next() {
		var trackCount, discNum int
		rows.Scan(&trackCount, &discNum)
		m := Media{Position: discNum, TrackCount: trackCount}
		media = append(media, m)
	}
	rows.Close()
	return media, nil
}

// Try to pattern match releases names. This can help if the track
// release name contains underscores which match nicely with 'like'.
func (m *Music) artistReleasesLike(a Artist, pattern string, trackCount, discCount int) []Release {
	var releases []Release
	m.db.Where("artist = ? and name like ? and track_count = ? and disc_count = ?",
		a.Name, pattern, trackCount, discCount).Find(&releases)
	if len(releases) == 0 {
		// try w/o disc
		m.db.Where("artist = ? and name like ? and track_count = ?",
			a.Name, pattern, trackCount).Find(&releases)
	}
	return releases
}

// Find the tracks that haven't been assigned a REID or RGID.
func (m *Music) tracksWithoutAssignedRelease() []Track {
	var tracks []Track
	m.db.Where("ifnull(re_id, '') = '' or ifnull(rg_id, '') = ''").
		Find(&tracks)
	return tracks
}

func (m *Music) tracksWithoutArtwork() []Track {
	var tracks []Track
	m.db.Where("artwork = 1 and front_cover = 0 and back_cover = 0").
		Find(&tracks)
	return tracks
}

func (m *Music) tracksWithoutAssignedReleaseDate() []Track {
	var tracks []Track
	m.db.Where("re_id <> '' and rg_id <> ''" +
		" and (ifnull(date, '') = '' or ifnull(release_date, '') == '')").
		Find(&tracks)
	return tracks
}

// Assign a track to a specific MusicBrainz release. Since the
// original data is just file names, the release is selected
// automatically.
func (m *Music) assignTrackRelease(t Track, r Release) error {
	err := m.db.Model(&t).
		Update("re_id", r.REID).
		Update("rg_id", r.RGID).
		Update("date", r.Date.Year()).
		Update("release_date", r.ReleaseDate).
		Update("artwork", r.Artwork).
		Update("front_artwork", r.FrontArtwork).
		Update("back_artwork", r.BackArtwork).
		Update("other_artwork", r.OtherArtwork).
		Update("group_artwork", r.GroupArtwork).Error
	if err != nil {
		return err
	}
	return nil
}

// Replace an existing release with a potentially new one. This allows
// for a re-sync from MusicBrainz, preserving timestamps and also the
// track assignment using the RGID & REID.
func (m *Music) replaceRelease(curr Release, with Release) error {
	with.ID = curr.ID
	with.CreatedAt = curr.CreatedAt
	with.UpdatedAt = curr.UpdatedAt
	with.DeletedAt = curr.DeletedAt
	return m.db.Save(&with).Error
}

func (m *Music) deleteReleaseMedia(reid string) {
	var media []Media
	m.db.Where("re_id = ?", reid).Find(&media)
	for _, d := range media {
		m.db.Unscoped().Delete(d)
	}
}

func (m *Music) updateTrackTitle(t Track, newTitle string) (err error) {
	err = m.db.Model(t).Update("title", newTitle).Error
	return
}

func (m *Music) updateTrackRID(t Track, rid string) (err error) {
	err = m.db.Model(t).Update("r_id", rid).Error
	return
}

// Part of the sync process to find releases that match the track. The
// preferred release will be the first one so dates corresponding to
// original release dates.
func (m *Music) trackReleases(t Track) []Release {
	var releases []Release
	m.db.Where("artist = ? and name = ? and track_count = ? and disc_count = ?",
		t.Artist, t.Release, t.TrackCount, t.DiscCount).
		// Having("date = min(date)").
		// Group("name").
		Order("release_date").Find(&releases)
	return releases
}

// Same as above but prefer those with front cover art
func (m *Music) trackReleasesWithFrontArtwork(t Track) []Release {
	var releases []Release
	m.db.Where("artist = ? and name = ? and track_count = ? and disc_count = ? and front_artwork = 1",
		t.Artist, t.Release, t.TrackCount, t.DiscCount).
		// Having("date = min(date)").
		// Group("name").
		Order("release_date").Find(&releases)
	return releases
}

// During sync try to find releases choices, preferring those with artwork
func (m *Music) trackReleaseChoices(t Track) []Release {
	releases := m.trackReleasesWithFrontArtwork(t)
	if len(releases) == 0 {
		releases = m.trackReleases(t)
	}
	return releases
}

// During sync try to find a single release with artwork to match a track.
func (m *Music) trackRelease(t Track) (Release, error) {
	releases := m.trackReleaseChoices(t)
	if len(releases) == 0 {
		return Release{}, ErrReleaseNotFound
	}
	return releases[0], nil
}

// Find the first release date for the release(s) with this track, including
// an media specific release from a multi-disc set like: Eagles/Legacy or The
// Beatles/The Beatles in Mono. These each have media with titles that
// themselves were previous releases so check them too.
// func (m *Music) trackFirstReleaseDate(t Track) (result time.Time, err error) {
// 	var releases []Release
// 	names := []string{t.Release}
// 	if t.MediaTitle != "" {
// 		// names would be "Legacy" and "Hotel California"
// 		names = append(names, t.MediaTitle)
// 	}
// 	m.db.Where("artist = ? and name in (?)",
// 		t.Artist, names).
// 		Having("date = min(date)").
// 		Group("name").
// 		Order("date").Find(&releases)
// 	if len(releases) > 0 {
// 		result = releases[0].Date
// 		err = nil
// 	} else {
// 		// could be disambiguation like "Weezer (Blue Album)" so just
// 		// use release date for now
// 		r, err := m.assignedRelease(t)
// 		if err == nil {
// 			result = r.Date
// 		}
// 	}
// 	return result, err
// }

// When there's artwork but no front, other_cover will be the ID of the image
// used for some type of artwork.
func (m *Music) updateOtherArtwork(r *Release, id string) error {
	return m.db.Model(r).Update("other_artwork", id).Error
}

func (m *Music) updateArtwork(r *Release) error {
	return m.db.Model(r).
		Update("artwork", r.FrontArtwork || r.BackArtwork).
		Update("front_artwork", r.FrontArtwork).
		Update("back_artwork", r.BackArtwork).
		Update("group_artwork", r.GroupArtwork).Error
}

// select count(*) from releases where artwork = false and re_id in (select distinct re_id from tracks);
func (m *Music) releasesWithoutArtwork() []Release {
	var releases []Release
	m.db.Where("artwork is false and re_id in (select distinct re_id from tracks)").
		Find(&releases)
	return releases
}

// At this point a release couldn't be found easily. Like Weezer has
// multiple albums called Weezer with the same number of tracks. Use
// MusicBrainz disambiguate to look further. This returns all artist
// releases with a specific track count that have a disambiguation.
func (m *Music) disambiguate(artist string, trackCount, discCount int) []Release {
	var releases []Release
	m.db.Where("releases.artist = ? and releases.track_count = ? and releases.disc_count = ? and releases.disambiguation != ''",
		artist, trackCount, discCount).
		Order("release_date").Find(&releases)
	return releases
}

// Find all releases for an artist. This is used during sync to match
// tracks again releases when there aren't exact matches.
func (m *Music) releases(a Artist) []Release {
	var releases []Release
	m.db.Where("artist = ?", a.Name).
		Order("release_date").Find(&releases)
	return releases
}

// Unique list of artist names based on tracks. This is useful to find
// artist names that may need to be modified to actual names.  Use
// artists() otherwise since it uses sortName from MusicBrainz.
func (m *Music) trackArtistNames() []string {
	var tracks []*Track
	m.db.Select("distinct(artist)").Find(&tracks)
	var artists []string
	for _, t := range tracks {
		artists = append(artists, t.Artist)
	}
	return artists
}

// Builds a list of artist name and arid pairs to allow for track metadata to
// provide name and arid which aids in resolving names.
func (m *Music) trackArtists() ([][]string, error) {
	rows, err := m.db.Table("tracks").
		Select("artist, ar_id").
		Group("artist").
		Order("artist").Rows()
	if err != nil {
		return nil, err
	}
	var result [][]string
	for rows.Next() {
		var artist, arid string
		rows.Scan(&artist, &arid)
		result = append(result, []string{artist, arid})
	}
	rows.Close()
	return result, nil
}

func (m *Music) assignTrackArtist(t Track, artist string) error {
	err := m.db.Model(t).
		Update("track_artist", artist).Error
	return err
}

func (m *Music) ArtistSingleTracks(a Artist, limit ...int) []Track {
	var tracks []Track
	l := m.config.Music.SinglesLimit
	if len(limit) == 1 {
		l = limit[0]
	}

	// m.db.Where("b.title is null and tracks.artist = ?"+
	// 	" and tracks.title in (select distinct single_name from releases where artist = ? and type = 'Single')"+
	// 	a.Name, a.Name).
	// 	Joins("left outer join tracks b on tracks.title = b.title and tracks.date > b.date").
	// 	Group("tracks.title").
	// 	Order("tracks.date").
	// 	Limit(l).
	// 	Find(&tracks)

	// "select artist, `release`, title, date from tracks where artist =
	// 'Iron Maiden' and title in (select distinct single_name from
	// releases where artist = 'Iron Maiden' and type = 'Single') group by
	// title having min(date) order by date"
	m.db.Where("artist = ? and title in"+
		" (select distinct single_name from releases where artist = ? and type = 'Single')",
		a.Name, a.Name).
		Group("title").
		Having("min(date)").
		Order("date").
		Limit(l).
		Find(&tracks)

	return tracks
}

func (m *Music) ArtistPopularTracks(a Artist, limit ...int) []Track {
	var tracks []Track
	l := m.config.Music.PopularLimit
	if len(limit) == 1 {
		l = limit[0]
	}

	// select tracks.title, tracks.date from tracks
	// left outer join tracks b on tracks.title = b.title and tracks.date > b.date
	// inner join popular on tracks.artist = popular.artist and tracks.title = popular.title
	// where b.title is null and tracks.artist = 'Gary Numan'
	// group by tracks.title
	// order by popular.rank;

	m.db.Where("b.title is null and tracks.artist = ?", a.Name).
		Joins("left outer join tracks b on tracks.title = b.title and tracks.date > b.date").
		Joins("inner join popular on tracks.artist = popular.artist and tracks.title = popular.title").
		Group("tracks.title").
		Order("popular.rank").
		Limit(l).
		Find(&tracks)
	return tracks
}

func (m *Music) artistDeepTracks(a Artist, limit ...int) []Track {
	var tracks []Track
	l := m.config.Music.DeepLimit
	if len(limit) == 1 {
		l = limit[0]
	}
	popularTracks := "select popular.title from popular where tracks.artist = popular.artist"
	singleTracks := "select releases.single_name from releases where tracks.artist = releases.artist and releases.type = 'Single'"
	m.db.Where("tracks.artist = ?"+
		" and tracks.title not in ("+popularTracks+")"+
		" and tracks.title not in ("+singleTracks+")",
		a.Name).
		Limit(l).
		Group("tracks.artist, tracks.title").
		Find(&tracks)
	return tracks
}

func (m *Music) ArtistTracks(a Artist) []Track {
	return m.artistTracks(a.Name)
}

func (m *Music) artistTracks(artist string) []Track {
	var tracks []Track
	m.db.Where("artist = ?", artist).
		Order("release, date, disc_num, track_num").
		Find(&tracks)
	return tracks
}

// All artist names ordered by sortName from MusicBrainz.
func (m *Music) Artists() []Artist {
	var artists []Artist
	m.db.Order("sort_name asc").Find(&artists)
	return artists
}

// All artists with corresponding MusicBrainz artist IDs.
func (m *Music) artistsByMBID(arids []string) []Artist {
	var artists []Artist
	m.db.Where("ar_id in (?)", arids).Find(&artists)
	return artists
}

// not used
func (m *Music) similarArtistsByTags(a *Artist) []Artist {
	var artists []Artist
	m.db.Order("count(artist_tags.artist) desc").
		Joins("inner join artist_tags on artists.name = artist_tags.artist").
		Where("artist_tags.tag in (select tag from artist_tags where artist = ?)", a.Name).
		Group("artist_tags.artist").Find(&artists)
	return artists
}

// Similar artists based on similarity rank from Last.fm.
func (m *Music) SimilarArtists(a Artist, limit ...int) []Artist {
	var artists []Artist
	l := m.config.Music.SimilarArtistsLimit
	if len(limit) == 1 {
		l = limit[0]
	}

	m.db.Joins("inner join similar on similar.artist = ?", a.Name).
		Where("artists.ar_id = similar.ar_id").
		Order("similar.rank asc").
		Limit(l).
		Find(&artists)

	if len(artists) == 0 {
		artists = m.RelatedArtists(a, limit...)
	}

	return artists
}

// Useful when there are no similar artists. Artists with same genre that
// started around the same time (default is +/- 5 years)
func (m *Music) RelatedArtists(a Artist, limit ...int) []Artist {
	var artists []Artist
	l := m.config.Music.SimilarArtistsLimit
	if len(limit) == 1 {
		l = limit[0]
	}

	after := a.Date.Add(m.config.Music.RelatedArtists * -1)
	before := a.Date.Add(m.config.Music.RelatedArtists)

	m.db.Where("ar_id <> ? and genre = ? and date >= ? and date <= ?", a.ARID, a.Genre, after, before).
		Limit(l).
		Find(&artists)

	return artists
}

// Similar releases based on releases from similar artists in the
// previous and following year.
func (m *Music) SimilarReleases(a Artist, r Release) []Release {
	artists := m.SimilarArtists(a)
	var names []string
	for _, sa := range artists {
		names = append(names, sa.Name)
	}

	after := r.Date.Add(m.config.Music.SimilarReleases * -1)
	before := r.Date.Add(m.config.Music.SimilarReleases)

	var releases []Release
	m.db.Joins("inner join tracks on tracks.artist in (?)", names).
		Where("releases.re_id = tracks.re_id").
		Group("releases.date").
		Having("releases.date >= ? and releases.date <= ?", after, before).
		Limit(m.config.Music.SimilarReleasesLimit).
		Order("releases.date").Find(&releases)
	return releases
}

// All releases for an artist that have corresponing tracks.
func (m *Music) ArtistReleases(a Artist) []Release {
	var releases []Release
	m.db.Where("releases.re_id in (select distinct re_id from tracks where artist = ?)",
		a.Name).Order("date asc").Find(&releases)
	return releases
}

// Recently added releases are ordered by LastModified which comes
// from the track object in the S3 bucket.  Use config Recent and
// RecentLimit to tune the result count.
func (m *Music) RecentlyAdded() []Release {
	var releases []Release

	// select releases.name from releases inner join tracks on tracks.re_id
	// = releases.re_id and tracks.last_modified > '2022-01-01' group by
	// releases.artist, releases.name order by tracks.last_modified desc
	// limit 100;

	m.db.Joins("inner join tracks on tracks.re_id = releases.re_id"+
		" and tracks.last_modified >= ?", time.Now().Add(m.config.Music.Recent*-1)).
		Group("releases.artist, releases.name").
		Order("tracks.last_modified desc").
		Limit(m.config.Music.RecentLimit).
		Find(&releases)
	return releases
}

// Recently released releases are ordered by the MusicBrainz first
// release date of the release, newest first.  Use config Recent and
// RecentLimit to tune the result count.
func (m *Music) RecentlyReleased() []Release {
	var releases []Release
	m.db.Joins("inner join tracks on tracks.re_id = releases.re_id").
		Group("releases.artist, releases.name").
		Having("releases.date >= ?", time.Now().Add(m.config.Music.Recent*-1)).
		Order("releases.date desc").
		Limit(m.config.Music.RecentLimit).
		Find(&releases)
	return releases
}

// Obtain the specfic release for this track based on the assigned
// REID or RGID from MusicBrainz. This is useful for covers.
func (m *Music) assignedRelease(t Track) (Release, error) {
	var release Release
	err := m.db.Where("re_id = ?", t.REID).First(&release).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err = m.db.Where("rg_id = ?", t.RGID).First(&release).Error
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			return Release{}, ErrReleaseNotFound
		}
	}
	return release, err
}

// func (m *Music) releaseGroup(rgid string) (Release, error) {
// 	var release Release
// 	if m.db.Where("rg_id = ?", rgid).First(&release).ErrRecordNotFound() {
// 		return nil, errors.New("release group not found")
// 	}
// 	return release, nil
// }

// Obtain a release using MusicBrainz REID.
func (m *Music) release(reid string) (Release, error) {
	var release Release
	err := m.db.Where("re_id = ?", reid).First(&release).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Release{}, ErrReleaseNotFound
	}
	return release, err
}

// Obtain all the tracks for this release, ordered by disc and track
// number.
func (m *Music) ReleaseTracks(release Release) []Track {
	var tracks []Track
	m.db.Where("re_id = ?", release.REID).Order("disc_num, track_num").Find(&tracks)
	return tracks
}

func (m *Music) releaseMedia(release Release) []Media {
	var media []Media
	m.db.Where("re_id = ?", release.REID).Order("position").Find(&media)
	return media
}

func (m *Music) ReleaseSingles(release Release) []Track {
	var tracks []Track
	m.db.Where("tracks.re_id = ? and"+
		" exists (select releases.single_name from releases where tracks.artist = releases.artist"+
		" and releases.type = 'Single' and releases.single_name = tracks.title)",
		release.REID).
		Order("tracks.disc_num, tracks.track_num").Find(&tracks)
	return tracks
}

func (m *Music) ReleasePopular(release Release) []Track {
	var tracks []Track
	m.db.Where("re_id = ? and"+
		" exists (select popular.title from popular where"+
		" tracks.artist = popular.artist and tracks.title = popular.title)",
		release.REID).
		Order("disc_num, track_num").Find(&tracks)
	return tracks
}

func (m *Music) ReleasesLike(name string) []Release {
	var releases []Release
	m.db.Where("name like ? and re_id in"+
		" (select distinct(re_id) from tracks)", name).
		Order("date").
		Find(&releases)
	return releases
}

func (m *Music) ReleasesForREIDs(reids []string) []Release {
	var releases []Release
	m.db.Where("re_id in (?)", reids).Find(&releases)
	return releases
}

// Lookup a release given the internal record ID.
func (m *Music) LookupRelease(id int) (Release, error) {
	var release Release
	err := m.db.First(&release, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Release{}, ErrReleaseNotFound
	}
	return release, err
}

// Lookup a release given the MusicBrainz REID.
func (m *Music) LookupREID(reid string) (Release, error) {
	var release Release
	err := m.db.First(&release, "re_id = ?", reid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Release{}, ErrReleaseNotFound
	}
	return release, err
}

// Lookup a track given the MusicBrainz RID.
func (m *Music) LookupRID(rid string) (Track, error) {
	var track Track
	err := m.db.First(&track, "r_id = ?", rid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Track{}, ErrTrackNotFound
	}
	return track, err
}

// Lookup a track given the UUID.
func (m *Music) LookupUUID(uuid string) (Track, error) {
	var track Track
	err := m.db.First(&track, "uuid = ?", uuid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Track{}, ErrTrackNotFound
	}
	return track, err
}

// Lookup an artist given the internal record ID.
func (m *Music) LookupArtist(id int) (Artist, error) {
	var artist Artist
	err := m.db.First(&artist, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Artist{}, ErrArtistNotFound
	}
	return artist, err
}

// Lookup an artist given the MusicBrainz ARID.
func (m *Music) LookupARID(arid string) (Artist, error) {
	var artist Artist
	err := m.db.First(&artist, "ar_id = ?", arid).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Artist{}, ErrArtistNotFound
	}
	return artist, err
}

// Lookup a track given the internal record ID.
func (m *Music) LookupTrack(id int) (Track, error) {
	var track Track
	err := m.db.First(&track, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Track{}, ErrTrackNotFound
	}
	return track, err
}

func (m *Music) tracksForRIDs(rids []string) []Track {
	var tracks []Track
	m.db.Where("r_id in (?)", rids).Find(&tracks)
	return tracks
}

func (m *Music) tracksFor(keys []string) []Track {
	var tracks []Track
	m.db.Where("key in (?)", keys).Find(&tracks)
	return tracks
}

// Lookup a track given the etag from the S3 bucket object. Etag can
// be used as a good external identifier (for playlists) since the
// interal record ID can change.
func (m *Music) LookupETag(etag string) (Track, error) {
	var track Track
	err := m.db.First(&track, "e_tag = ?", etag).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Track{}, ErrTrackNotFound
	}
	return track, err
}

// Simple sql search for artists, releases, tracks and stations. Use config
// SearchLimit to change the result count.
func (m *Music) Query(query string) ([]Artist, []Release, []Track, []Station) {
	var artists []Artist
	var releases []Release
	var tracks []Track
	var stations []Station

	queryArtists := func(name string) {
		m.db.Where("name like ?", name).
			Order("sort_name asc").
			Limit(m.config.Music.SearchLimit).Find(&artists)
	}

	queryReleases := func(name string) {
		m.db.Joins("inner join tracks on"+
			" tracks.re_id = releases.re_id and tracks.release like ?", name).
			Group("releases.name, releases.date").
			Order("releases.name").Limit(m.config.Music.SearchLimit).
			Find(&releases)
	}

	queryTitles := func(name string) {
		m.db.Where("title like ?", name).
			Order("title").Limit(m.config.Music.SearchLimit).Find(&tracks)
	}

	queryStations := func(name string) {
		m.db.Where("name like ? or creator like ?", name, name).
			Limit(m.config.Music.SearchLimit).Find(&stations)
	}

	s := strings.Split(query, ":")
	if len(s) > 1 {
		field, terms := s[0], strings.Join(s[1:], "")
		terms = strings.Trim(terms, `"`)
		terms = "%" + terms + "%"
		switch field {
		case "+artist", "artist":
			queryArtists(terms)
		case "+release", "release":
			queryReleases(terms)
		case "+title", "title":
			queryTitles(terms)
		case "+radio", "radio", "+station", "station":
			queryStations(terms)
		}
	} else {
		query = "%" + query + "%"
		queryArtists(query)
		queryReleases(query)
		queryTitles(query)
		queryStations(query)
	}

	return artists, releases, tracks, stations
}

// Lookup user playlist
func (m *Music) LookupPlaylist(user auth.User, id int) (Playlist, error) {
	var p Playlist
	err := m.db.Where("user = ? and id = ?", user.Name, id).First(&p).Error
	if err != nil {
		return Playlist{}, ErrPlaylistNotFound
	}
	return p, nil
}

func (m *Music) PlaylistsLike(user auth.User, name string) []Playlist {
	var playlists []Playlist
	m.db.Where("user = ? and name like ?", user.Name, name).Find(&playlists)
	return playlists
}

func (m *Music) UserPlaylist(user auth.User) (Playlist, error) {
	var p Playlist
	err := m.db.Where("user = ? and ifnull(name, '') = ''", user.Name).First(&p).Error
	if err != nil {
		return Playlist{}, ErrPlaylistNotFound
	}
	return p, nil
}

func (m *Music) UserPlaylists(user auth.User) []Playlist {
	var playlists []Playlist
	m.db.Where("user = ? and ifnull(name, '') <> ''", user.Name).Find(&playlists)
	return playlists
}

func (m *Music) UserPlaylistCount(user auth.User) int64 {
	var count int64
	m.db.Model(&Playlist{}).Where("user = ?", user.Name).Count(&count)
	return count
}

// Save a playlist.
func (m *Music) UpdatePlaylist(p *Playlist) error {
	return m.db.Save(p).Error
}

func (m *Music) DeletePlaylist(user auth.User, id int) error {
	return m.db.Unscoped().Where("user = ? and id = ?", user.Name, id).Delete(Playlist{}).Error
}

// Obtain user stations.
func (m *Music) Stations(user auth.User) []Station {
	var stations []Station
	m.db.Where("user = ? or shared = 1", user.Name).Find(&stations)
	return stations
}

// Stations by name
func (m *Music) StationsLike(name string) []Station {
	var stations []Station
	m.db.Where("name like ? or creator like ?", name, name).Find(&stations)
	return stations
}

func (m *Music) clearStationPlaylists() {
	m.db.Exec(`update stations set playlist = ""`)
}

func (m *Music) deleteStations() {
	m.db.Exec(`delete from stations`)
}

// Obtain user station by id.
func (m *Music) LookupStation(id int) (Station, error) {
	var s Station
	err := m.db.First(&s, id).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return Station{}, ErrStationNotFound
	}
	return s, err
}

// Update a station.
func (m *Music) UpdateStation(s *Station) error {
	return m.db.Save(s).Error
}

func (m *Music) DeleteStation(s *Station) error {
	return m.db.Unscoped().Delete(s).Error
}

func (m *Music) favoriteArtists(limit int) ([]string, error) {
	var artists []string
	rows, err := m.db.Table("tracks").
		Select("artist, count(title)").
		Group("artist").
		Limit(limit).
		Order("count(title) desc").Rows()
	if err != nil {
		return artists, err
	}
	for rows.Next() {
		var artist string
		var count int
		rows.Scan(&artist, &count)
		artists = append(artists, artist)
	}
	rows.Close()
	return artists, nil
}

func (m *Music) artistBackgrounds(a Artist) []string {
	var backgrounds []ArtistBackground
	m.db.Where("artist = ?", a.Name).
		Order("rank desc").
		Find(&backgrounds)
	if len(backgrounds) == 0 {
		return []string{}
	}
	var list []string
	for _, i := range backgrounds {
		list = append(list, i.URL)
	}
	return list
}

func (m *Music) artistImages(a Artist) []string {
	var imgs []ArtistImage
	m.db.Where("artist = ?", a.Name).
		Order("rank desc").
		Find(&imgs)
	if len(imgs) == 0 {
		return []string{}
	}
	var list []string
	for _, i := range imgs {
		list = append(list, i.URL)
	}
	return list
}

func (m *Music) artistGenres() []string {
	var genres []string
	rows, err := m.db.Table("artists").
		Select("distinct(genre)").
		Order("genre").Rows()
	if err != nil {
		return genres
	}
	for rows.Next() {
		var v string
		rows.Scan(&v)
		genres = append(genres, v)
	}
	rows.Close()
	return genres
}

func (m *Music) TrackCount() int64 {
	var count int64
	m.db.Model(&Track{}).Count(&count)
	return count
}

func (m *Music) ArtistCount() int64 {
	var count int64
	m.db.Model(&Artist{}).Count(&count)
	return count
}

func (m *Music) ReleaseCount() int64 {
	var count int64
	m.db.Model(&Track{}).Distinct("re_id").Count(&count)
	return count
}

func (m *Music) searchTracks(title, artist, album string) []Track {
	var tracks []Track
	var tx *gorm.DB
	if len(album) != 0 {
		tx = m.db.Where("title = ? and artist = ? and (`release` = ? or `release_title` = ?)",
			title, artist, album, album)
	} else {
		tx = m.db.Where("title = ? and artist = ?", title, artist)
	}
	tx.Order("date").Find(&tracks)
	return tracks
}

func (m *Music) deletePopularFor(artist string) error {
	return m.db.Unscoped().Where("artist = ?", artist).Delete(Popular{}).Error
}

func (m *Music) deleteSimilarFor(artist string) error {
	return m.db.Unscoped().Where("artist = ?", artist).Delete(Similar{}).Error
}

func (m *Music) updateArtist(a *Artist) error {
	return m.db.Save(a).Error
}

func (m *Music) createArtist(a *Artist) error {
	return m.db.Create(a).Error
}

func (m *Music) createRelease(r *Release) error {
	return m.db.Create(r).Error
}

func (m *Music) createMedia(media *Media) error {
	return m.db.Create(media).Error
}

func (m *Music) createPopular(p *Popular) error {
	return m.db.Create(p).Error
}

func (m *Music) createSimilar(s *Similar) error {
	return m.db.Create(s).Error
}

func (m *Music) createArtistTag(t *ArtistTag) error {
	return m.db.Create(t).Error
}

func (m *Music) CreatePlaylist(p *Playlist) error {
	return m.db.Create(p).Error
}

func (m *Music) CreateStation(s *Station) error {
	return m.db.Create(s).Error
}

func (m *Music) createArtistBackground(bg *ArtistBackground) error {
	return m.db.Create(bg).Error
}

func (m *Music) createArtistImage(img *ArtistImage) error {
	return m.db.Create(img).Error
}

// select artist, name, date from releases where type = 'Album' and
// secondary_type = '' and status = 'Official' and artist = 'Black Sabbath' and
// lower(name) not in (select distinct lower(release) from tracks where artist
// = 'Black Sabbath') group by name, date order by date;
