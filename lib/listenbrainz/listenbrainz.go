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

package listenbrainz

import (
	// "sort"
	// "strconv"

	"github.com/takeoutfm/takeout/lib/client"
)

type ListenBrainz struct {
	client client.Getter
}

func NewListenBrainz(client client.Getter) *ListenBrainz {
	return &ListenBrainz{
		client: client,
	}
}

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

type Result struct {
	// ignore artist_mbids
	// ignore artists
	ArtistName    string `json:"artist_name"`
	Length        int    `json:"length"`
	RID           string `json:"recording_mbid"`
	RecordingName string `json:"recording_name"`
	REID          string `json:"release_mbid"`
	ReleaseName   string `json:"release_name"`
	ListenCount   int    `json:"total_listen_count"`
	UserCount     int    `json:"user_count"`
}

// ArtistTopTracks returns all top tracks from ListenBrainz.
func (l *ListenBrainz) ArtistTopTracks(arid string) ([]TopTrack, error) {
	var results []Result

	client.DefaultLimiter.RateLimit("listenbrainz.org")

	url := "https://api.listenbrainz.org/1/popularity/top-recordings-for-artist/" + arid
	err := l.client.GetJson(url, &results)
	if err != nil {
		return nil, err
	}

	var tracks []TopTrack
	for i, track := range results {
		tracks = append(tracks, TopTrack{track: track.RecordingName, rank: i + 1})
	}

	return tracks, nil
}
