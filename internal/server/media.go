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
	"path/filepath"
	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/podcast"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/log"
)

type Media struct {
	config  *config.Config
	music   *music.Music
	film    *film.Film
	podcast *podcast.Podcast
	tv      *tv.TV
}

func (m Media) Config() *config.Config {
	return m.config
}

func (m Media) Music() *music.Music {
	return m.music
}

func (m Media) Podcast() *podcast.Podcast {
	return m.podcast
}

func (m Media) Film() *film.Film {
	return m.film
}

func (m Media) TV() *tv.TV {
	return m.tv
}

func mediaConfigFor(root *config.Config, user auth.User) (string, *config.Config, error) {
	// only supports one media collection right now
	mediaName := user.FirstMedia()
	if mediaName == "" {
		return "", nil, ErrNoMedia
	}
	config, err := mediaConfig(root, mediaName)
	return mediaName, config, err
}

func mediaConfig(root *config.Config, mediaName string) (*config.Config, error) {
	path := filepath.Join(root.Server.MediaDir, mediaName)
	// load relative media configuration
	userConfig, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	// XXX merge, copy server config for now
	userConfig.Server = root.Server
	if err != nil {
		return nil, err
	}
	return userConfig, nil
}

var mediaMap map[string]*Media = make(map[string]*Media)

func makeMedia(name string, config *config.Config) *Media {
	media, ok := mediaMap[name]
	if !ok {
		var err error
		media = &Media{}
		media.music, err = media.makeMusic(config)
		log.CheckError(err)
		media.film, err = media.makeFilm(config)
		log.CheckError(err)
		media.podcast, err = media.makePodcast(config)
		log.CheckError(err)
		media.tv, err = media.makeTV(config)
		log.CheckError(err)
		mediaMap[name] = media
	}
	return media
}

func (Media) makeMusic(config *config.Config) (*music.Music, error) {
	m := music.NewMusic(config)
	err := m.Open()
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (Media) makeFilm(config *config.Config) (*film.Film, error) {
	f := film.NewFilm(config)
	err := f.Open()
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (Media) makePodcast(config *config.Config) (*podcast.Podcast, error) {
	p := podcast.NewPodcast(config)
	err := p.Open()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (Media) makeTV(config *config.Config) (*tv.TV, error) {
	tv := tv.NewTV(config)
	err := tv.Open()
	if err != nil {
		return nil, err
	}
	return tv, nil
}
