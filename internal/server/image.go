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

package server

import (
	"fmt"
	"net/http"

	"github.com/takeoutfm/takeout/lib/header"
)

const (
	CoverArtArchivePrefix = "https://coverartarchive.org"
	TMDBPrefix            = "https://image.tmdb.org"
	FanArtPrefix          = "https://assets.fanart.tv/fanart"
)

// The image cache has two different use cases: reading & writing. Writing is
// performed using a client that will check for updates at the source and cache
// the result locally. A forced max-age can be used to extend the the age of
// the cached image. Reading will use a cache-only client which only reads
// pre-cached images and will not try to fetch from the source. When nothing is
// cached, the reader will redirect to the original source. Same config is used
// for both use cases.

func checkImageCache(w http.ResponseWriter, r *http.Request, url string) {
	ctx := contextValue(r)
	client := ctx.ImageClient()
	hdr, img, err := client.Get(url)
	if err == nil && len(img) > 0 {
		for k, v := range hdr {
			switch k {
			case header.ContentType, header.ContentLength, header.ETag,
				header.LastModified, header.CacheControl:
				w.Header().Set(k, v[0])
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(img)
	} else {
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func imgRelease(w http.ResponseWriter, r *http.Request) {
	reid := r.PathValue("reid")
	side := r.PathValue("side")
	url := fmt.Sprintf("%s/release/%s/%s-250", CoverArtArchivePrefix, reid, side)
	checkImageCache(w, r, url)
}

func imgReleaseFront(w http.ResponseWriter, r *http.Request) {
	reid := r.PathValue("reid")
	url := fmt.Sprintf("%s/release/%s/front-250", CoverArtArchivePrefix, reid)
	checkImageCache(w, r, url)
}

func imgReleaseGroup(w http.ResponseWriter, r *http.Request) {
	rgid := r.PathValue("rgid")
	side := r.PathValue("side")
	url := fmt.Sprintf("%s/release-group/%s/%s-250", CoverArtArchivePrefix, rgid, side)
	checkImageCache(w, r, url)
}

func imgReleaseGroupFront(w http.ResponseWriter, r *http.Request) {
	rgid := r.PathValue("rgid")
	url := fmt.Sprintf("%s/release-group/%s/front-250", CoverArtArchivePrefix, rgid)
	checkImageCache(w, r, url)
}

func imgTMDB(w http.ResponseWriter, r *http.Request) {
	size := r.PathValue("size")
	path := r.PathValue("path")
	url := fmt.Sprintf("%s/t/p/%s/%s", TMDBPrefix, size, path)
	checkImageCache(w, r, url)
}

func imgArtistThumb(w http.ResponseWriter, r *http.Request) {
	arid := r.PathValue("arid")
	path := r.PathValue("path")
	url := fmt.Sprintf("%s/music/%s/artistthumb/%s", FanArtPrefix, arid, path)
	checkImageCache(w, r, url)
}

func imgArtistBackground(w http.ResponseWriter, r *http.Request) {
	arid := r.PathValue("arid")
	path := r.PathValue("path")
	url := fmt.Sprintf("%s/music/%s/artistbackground/%s", FanArtPrefix, arid, path)
	checkImageCache(w, r, url)
}
