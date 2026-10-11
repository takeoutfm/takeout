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
	"context"
	"path/filepath"
	"sync"
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

// mediaConfigs caches the media config merged with the server config. Every
// authenticated request goes through mediaConfig, so it must be cheap and must
// not modify anything shared between requests.
type mediaConfigKey struct {
	root *config.Config
	name string
}

var mediaConfigs = make(map[mediaConfigKey]*config.Config)
var mediaConfigMu sync.Mutex

func mediaConfig(root *config.Config, mediaName string) (*config.Config, error) {
	key := mediaConfigKey{root: root, name: mediaName}

	mediaConfigMu.Lock()
	defer mediaConfigMu.Unlock()
	if c, ok := mediaConfigs[key]; ok {
		return c, nil
	}

	path := filepath.Join(root.Server.MediaDir, mediaName)
	// load relative media configuration (config.LoadConfig caches this
	// and returns a shared pointer, so it must not be modified)
	loaded, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	// XXX merge, copy server config for now. This used to assign to the
	// shared loaded config on every request, which is a data race; work on
	// a private copy that is built once and then only read.
	merged := *loaded
	merged.Server = root.Server
	mediaConfigs[key] = &merged
	return &merged, nil
}

// mediaMap is shared by every request goroutine; mediaMu guards it. The lock is
// held while a new Media is opened so two first requests for the same name
// can't both open (and leak) the databases.
var mediaMap map[string]*Media = make(map[string]*Media)
var mediaMu sync.Mutex

func makeMedia(ctx context.Context, name string, config *config.Config) *Media {
	mediaMu.Lock()
	defer mediaMu.Unlock()
	media, ok := mediaMap[name]
	if !ok {
		var err error
		media = &Media{}
		media.music, err = media.makeMusic(ctx, config)
		log.CheckError(err)
		media.film, err = media.makeFilm(ctx, config)
		log.CheckError(err)
		media.podcast, err = media.makePodcast(ctx, config)
		log.CheckError(err)
		media.tv, err = media.makeTV(ctx, config)
		log.CheckError(err)
		mediaMap[name] = media
	}
	return media
}

func (Media) makeMusic(ctx context.Context, config *config.Config) (*music.Music, error) {
	m := music.NewMusic(config)
	err := m.Open(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (Media) makeFilm(ctx context.Context, config *config.Config) (*film.Film, error) {
	f := film.NewFilm(config)
	err := f.Open(ctx)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (Media) makePodcast(ctx context.Context, config *config.Config) (*podcast.Podcast, error) {
	p := podcast.NewPodcast(config)
	err := p.Open()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (Media) makeTV(ctx context.Context, config *config.Config) (*tv.TV, error) {
	tv := tv.NewTV(config)
	err := tv.Open(ctx)
	if err != nil {
		return nil, err
	}
	return tv, nil
}
