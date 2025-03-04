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

package lastfm // import "takeoutfm.dev/takeout/lib/lastfm"

import (
	lfm "github.com/shkh/lastfm-go/lastfm"
	"sort"
	"strconv"

	"takeoutfm.dev/takeout/lib/client"
)

type Config struct {
	Key    string
	Secret string
}

type Lastfm struct {
	config Config
	client client.Getter
}

func NewLastfm(config Config, client client.Getter) *Lastfm {
	return &Lastfm{
		config: config,
		client: client,
	}
}

// Lastfm is used for:
// * getting popular tracks for each artist
// * getting similar artists for each artist
// * looking up artists not found by MusicBrainz to get their MBID

type TopTrack struct {
	track string
	rank  int
}

func (t TopTrack) Track() string {
	return t.track
}

func (t TopTrack) Rank() int {
	return t.rank
}

func (m *Lastfm) ArtistTopTracks(arid string) []TopTrack {
	if m.config.Key == "" || m.config.Secret == "" {
		return make([]TopTrack, 0)
	}

	client.DefaultLimiter.RateLimit("last.fm")
	api := lfm.New(m.config.Key, m.config.Secret)
	result, _ := api.Artist.GetTopTracks(lfm.P{"mbid": arid})
	sort.Slice(result.Tracks, func(i, j int) bool {
		a, _ := strconv.Atoi(result.Tracks[i].PlayCount)
		b, _ := strconv.Atoi(result.Tracks[j].PlayCount)
		return a > b
	})

	var tracks []TopTrack
	for _, track := range result.Tracks {
		rank, _ := strconv.Atoi(track.Rank)
		tracks = append(tracks, TopTrack{track: track.Name, rank: rank})
	}

	return tracks
}

func (m *Lastfm) SimilarArtists(arid string) map[string]float64 {
	if m.config.Key == "" || m.config.Secret == "" {
		return make(map[string]float64)
	}

	client.DefaultLimiter.RateLimit("last.fm")
	api := lfm.New(m.config.Key, m.config.Secret)
	result, _ := api.Artist.GetSimilar(lfm.P{"mbid": arid})

	//var mbids []string
	rank := make(map[string]float64)
	for _, similar := range result.Similars {
		//mbids = append(mbids, similar.Mbid)
		rank[similar.Mbid], _ = strconv.ParseFloat(similar.Match, 64)
	}

	// artists := m.artistsByMBID(mbids)
	// sort.Slice(artists, func(i, j int) bool {
	// 	return rank[artists[i].ARID] > rank[artists[j].ARID]
	// })

	// var similar []Similar
	// for index, a := range artists {
	// 	similar = append(similar, Similar{Artist: artist.Name, ARID: a.ARID, Rank: index})
	// }

	//return similar
	return rank
}

func (m *Lastfm) ArtistSearch(name string) (string, string) {
	if m.config.Key == "" || m.config.Secret == "" {
		return "", ""
	}

	client.DefaultLimiter.RateLimit("last.fm")
	api := lfm.New(m.config.Key, m.config.Secret)
	result, _ := api.Artist.Search(lfm.P{"artist": name})

	//var artist *Artist
	for index, match := range result.ArtistMatches {
		if index == 0 {
			//artist = &Artist{Name: match.Name, ARID: match.Mbid}
			return match.Name, match.Mbid
		}
	}

	return "", ""
}
