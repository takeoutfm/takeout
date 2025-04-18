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
	"fmt"
	"strings"

	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/lib/musicbrainz"
	"takeoutfm.dev/takeout/lib/search"
)

const (
	FieldArtist      = "artist"
	FieldAsin        = "asin"
	FieldDate        = "date"
	FieldFirstDate   = "first_date"
	FieldGenre       = "genre"
	FieldLabel       = "label"
	FieldLength      = "length"
	FieldMedia       = "media"
	FieldMediaTitle  = "media_title"
	FieldPopularity  = "popularity"
	FieldRating      = "rating"
	FieldRelease     = "release"
	FieldReleaseDate = "release_date"
	FieldSeries      = "series"
	FieldStatus      = "status"
	FieldTag         = "tag"
	FieldTitle       = "title"
	FieldTrack       = "track"
	FieldType        = "type"

	FieldBass      = "base"
	FieldClarinet  = "clarinet"
	FieldDrums     = "drums"
	FieldFlute     = "flute"
	FieldGuitar    = "guitar"
	FieldPiano     = "piano"
	FieldSaxophone = "saxophone"
	FieldVocals    = "vocals"

	TypePopular = "popular"
	TypeSingle  = "single"
	TypeCover   = "cover"
	TypeLive    = "live"
)

type trackIndex struct {
	// these fields are used to match against existing tracks
	DiscNum  int
	TrackNum int
	Title    string
	Artist   string
	RID      string
	// these are the indexed fields to store in the search db
	Fields   search.FieldMap
}

func (m *Music) creditsIndex(reid string) ([]trackIndex, error) {
	rel, err := m.mbz.Release(reid)
	if err != nil {
		return nil, err
	}

	fields := make(search.FieldMap)

	// general fields
	if rel.Disambiguation != "" {
		addField(fields, FieldRelease, fmt.Sprintf("%s (%s)", rel.Title, rel.Disambiguation))
	} else {
		addField(fields, FieldRelease, rel.Title)
	}
	addField(fields, FieldAsin, rel.Asin)
	addField(fields, FieldStatus, rel.Status)
	if rel.ReleaseGroup.Rating.Votes > 0 {
		addField(fields, FieldRating, rel.ReleaseGroup.Rating.Value)
	}
	for _, l := range rel.LabelInfo {
		addField(fields, FieldLabel, l.Label.Name)
	}

	// dates
	//   date: first release date of any release associated with this track
	//   release_date: date of the release associated with this track
	//   first_date: first release date of this track
	addField(fields, FieldDate, rel.ReleaseGroup.FirstReleaseDate)
	addField(fields, FieldReleaseDate, rel.Date)
	addField(fields, FieldFirstDate, rel.ReleaseGroup.FirstReleaseDate) // refined later

	// genres for artist and release group
	for _, a := range rel.ArtistCredit {
		if a.Name == VariousArtists {
			// this has many genres and tags so don't add
			continue
		}

		// use top 3 genres; could also just use PrimaryGenre()
		for i, g := range a.Artist.SortedGenres() {
			if i < 3 && g.Count > 0 {
				addField(fields, FieldGenre, g.Name)
			}
		}

		for _, t := range a.Artist.Tags {
			if t.Count > 0 {
				addField(fields, FieldTag, t.Name)
			}
		}
	}

	// use top 3 genres
	for i, g := range rel.ReleaseGroup.SortedGenres() {
		if i < 3 && g.Count > 0 {
			addField(fields, FieldGenre, g.Name)
		}
	}

	for _, t := range rel.ReleaseGroup.Tags {
		if t.Count > 0 {
			addField(fields, FieldTag, t.Name)
		}
	}

	relationCredits(fields, rel.Relations)

	var indices []trackIndex

	for _, m := range rel.FilteredMedia() {
		for _, t := range m.Tracks {
			trackFields := search.CloneFields(fields)
			addField(trackFields, FieldMedia, m.Position)
			if m.Title != "" {
				// include media specific title
				// Eagles / The Long Run (Legacy)
				addField(trackFields, FieldMediaTitle, m.Title)
			}
			addField(trackFields, FieldTrack, t.Position)
			// replace with first release date of this track
			if len(t.Recording.FirstReleaseDate) > 0 {
				setField(trackFields, FieldFirstDate, t.Recording.FirstReleaseDate)
			}
			addField(trackFields, FieldTitle, t.Recording.Title)
			addField(trackFields, FieldLength, t.Recording.Length/1000)
			relationCredits(trackFields, t.Recording.Relations)
			for _, a := range t.ArtistCredit {
				addField(trackFields, FieldArtist, a.Name)
			}

			index := trackIndex{
				DiscNum:  m.Position,
				TrackNum: t.Position,
				Title:    fixName(t.Recording.Title),
				Artist:   fixName(t.Artist()),
				RID:      t.Recording.ID,
				Fields:   trackFields,
			}
			// fmt.Printf("%d/%d/%s/%s\n", index.DiscNum, index.TrackNum, index.Title, index.RID)
			indices = append(indices, index)
		}
	}

	return indices, nil
}

func setField(c search.FieldMap, key string, value interface{}) search.FieldMap {
	// first remove
	k := strings.Replace(key, " ", "_", -1)
	delete(c, k)
	// now add
	return addField(c, key, value)
}

// TODO refactor to use search.AddField
func addField(c search.FieldMap, key string, value interface{}) search.FieldMap {
	key = strings.ToLower(key)
	keys := []string{key}

	// drums = drums (drum set)
	// guitar = lead guitar, slide guitar, rhythm guitar, acoustic
	// bass = bass guitar, electric bass guitar8eb5ae9e-ba52-4a8f-8513-822a5ccde819
	// vocals = lead vocals, backing vocals
	alternates := []string{
		FieldBass,
		FieldClarinet,
		FieldDrums,
		FieldFlute,
		FieldGuitar,
		FieldPiano,
		FieldSaxophone,
		FieldVocals,
	}
	for _, alt := range alternates {
		if strings.Contains(key, alt) {
			keys = append(keys, alt)
			// only match one; order matters
			break
		}
	}

	for _, k := range keys {
		k := strings.Replace(k, " ", "_", -1)
		switch value.(type) {
		case string:
			svalue := value.(string)
			svalue = fixName(svalue)
			if v, ok := c[k]; ok {
				switch v.(type) {
				case string:
					// string becomes array of 2 strings
					c[k] = []string{v.(string), svalue}
				case []string:
					// array of 3+ strings
					s := v.([]string)
					s = append(s, svalue)
					c[k] = s
				default:
					log.Panicln("bad field types")
				}
			} else {
				// single string
				c[k] = svalue
			}
		default:
			// numeric, date, etc.
			c[k] = value
		}
	}
	return c
}

func relationCredits(c search.FieldMap, relations []musicbrainz.Relation) search.FieldMap {
	for _, r := range relations {
		if "performance" == r.Type {
			for _, wr := range r.Work.Relations {
				switch wr.Type {
				case "arranger", "arrangement", "composer",
					"lyricist", "orchestrator", "orchestration",
					"writer":
					addField(c, wr.Type, wr.Artist.Name)
				case "based on", "medley", "misc",
					"instrument arranger", "named after",
					"other version", "revised by",
					"revision of", "parts", "premiere",
					"publishing", "translator", "vocal arranger":
					// ignore these
				default:
					//log.Printf("** ignore performance work relation '%s'\n", wr.Type)
				}
			}
			// check if this song is a cover
			if len(r.AttributeIds.Cover) > 0 || hasAttribute(r.Attributes, "cover") {
				addField(c, FieldType, TypeCover)
			}
			// check if this song is performed live
			if len(r.AttributeIds.Live) > 0 || hasAttribute(r.Attributes, "live") {
				addField(c, FieldType, TypeLive)
			}
		} else if "instrument" == r.Type {
			for _, a := range r.Attributes {
				addField(c, a, r.Artist.Name)
			}
		} else if "part of" == r.Type && "series" == r.TargetType {
			addField(c, FieldSeries, r.Series.Name)
		} else {
			if len(r.Attributes) > 0 {
				attr := r.Attributes[0]
				switch attr {
				case "co":
					addField(c, fmt.Sprintf("%s-%s", r.Attributes[0], r.Type), r.Artist.Name)
				case "additional", "assistant":
					addField(c, fmt.Sprintf("%s %s", r.Attributes[0], r.Type), r.Artist.Name)
				case "lead vocals":
					addField(c, attr, r.Artist.Name)
				}
			} else {
				addField(c, r.Type, r.Artist.Name)
			}
		}
	}
	return c
}

func hasAttribute(attrs []string, name string) bool {
	for _, a := range attrs {
		if a == name {
			return true
		}
	}
	return false
}
