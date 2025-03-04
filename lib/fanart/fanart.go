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

// Package fanart provides an interface to obtain artist artwork from
// Fanart.tv.
package fanart // import "takeoutfm.dev/takeout/lib/fanart"

import (
	"fmt"
	"takeoutfm.dev/takeout/lib/client"
)

type Config struct {
	PersonalKey string
	ProjectKey  string
}

type Fanart struct {
	config Config
	client client.Getter
}

func NewFanart(config Config, client client.Getter) *Fanart {
	return &Fanart{
		config: config,
		client: client,
	}
}

type Art struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Likes string `json:"likes"`
}

type Album struct {
	AlbumCovers []Art `json:"albumcover"`
	CDArtwork   []Art `json:"cdart"`
}

type Artist struct {
	Name              string           `json:"name"`
	MBID              string           `json:"mbid_id"`
	Albums            map[string]Album `json:"albums"`
	ArtistBackgrounds []Art            `json:"artistbackground"`
	ArtistThumbs      []Art            `json:"artistthumb"`
	HDMusicLogos      []Art            `json:"hdmusiclogo"`
	MusicLogos        []Art            `json:"musiclogo"`
	MusicBanners      []Art            `json:"musicbanner"`
}

func (f *Fanart) ArtistArt(arid string) *Artist {
	key := f.config.PersonalKey
	if key == "" {
		key = f.config.ProjectKey
	}
	if key == "" {
		return nil
	}

	url := fmt.Sprintf("http://webservice.fanart.tv/v3/music/%s?api_key=%s",
		arid, key)

	var result Artist
	f.client.GetJson(url, &result)
	return &result
}
