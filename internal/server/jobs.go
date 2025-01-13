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
	"net/http"

	"github.com/go-co-op/gocron"

	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/internal/film"
	"github.com/takeoutfm/takeout/internal/music"
	"github.com/takeoutfm/takeout/internal/podcast"
	"github.com/takeoutfm/takeout/internal/tv"
	"github.com/takeoutfm/takeout/lib/log"
	"time"
)

type syncFunc func(config *config.Config, mediaConfig *config.Config) error

func schedule(config *config.Config) {
	scheduler := gocron.NewScheduler(time.UTC)

	mediaSync := func(d time.Duration, doit syncFunc, startImmediately bool) {
		if d == 0 {
			// job is disabled
			return
		}
		sched := scheduler.Every(d)
		if startImmediately {
			sched = sched.StartImmediately()
		} else {
			sched = sched.WaitForSchedule()
		}
		sched.Do(func() {
			list, err := assignedMedia(config)
			if err != nil {
				log.Println(err)
				return
			}
			for _, mediaName := range list {
				mediaConfig, err := mediaConfig(config, mediaName)
				if err != nil {
					log.Println(err)
					return
				}
				doit(config, mediaConfig)
			}
		})
	}

	// music
	mediaSync(config.Music.SyncInterval, syncMusic, false)
	mediaSync(config.Music.PopularSyncInterval, syncMusicPopular, false)
	mediaSync(config.Music.SimilarSyncInterval, syncMusicSimilar, false)
	mediaSync(config.Music.CoverSyncInterval, syncMusicCovers, false)

	// podcasts
	mediaSync(config.Podcast.SyncInterval, syncPodcasts, false)

	// film
	mediaSync(config.Film.SyncInterval, syncFilm, false)
	mediaSync(config.Film.PosterSyncInterval, syncFilmPosters, false)
	mediaSync(config.Film.BackdropSyncInterval, syncFilmBackdrops, false)

	// tv
	mediaSync(config.TV.SyncInterval, syncTV, false)
	mediaSync(config.TV.PosterSyncInterval, syncTVPosters, false)
	mediaSync(config.TV.BackdropSyncInterval, syncTVBackdrops, false)
	mediaSync(config.TV.StillSyncInterval, syncTVStills, false)

	scheduler.Every(time.Minute * 5).WaitForSchedule().Do(func() {
		a := auth.NewAuth(config)
		err := a.Open()
		if err != nil {
			log.Println(err)
			return
		}
		defer a.Close()
		err = a.DeleteExpiredCodes()
		if err != nil {
			log.Println(err)
		}
		err = a.DeleteExpiredSessions()
		if err != nil {
			log.Println(err)
		}
	})

	scheduler.StartAsync()
}

func assignedMedia(config *config.Config) ([]string, error) {
	a := auth.NewAuth(config)
	err := a.Open()
	if err != nil {
		return []string{}, err
	}
	defer a.Close()
	return a.AssignedMedia(), nil
}

func syncMusic(config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open()
	if err != nil {
		return err
	}
	defer m.Close()
	syncOptions := music.NewSyncOptions()
	syncOptions.Since = m.LastModified()
	m.Sync(syncOptions)
	return nil
}

func syncWithOptions(mediaConfig *config.Config, syncOptions music.SyncOptions) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open()
	if err != nil {
		return err
	}
	defer m.Close()
	m.Sync(syncOptions)
	return nil
}

func syncMusicPopular(config *config.Config, mediaConfig *config.Config) error {
	return syncWithOptions(mediaConfig, music.NewSyncPopular())
}

func syncMusicSimilar(config *config.Config, mediaConfig *config.Config) error {
	return syncWithOptions(mediaConfig, music.NewSyncSimilar())
}

func syncMusicCovers(config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open()
	if err != nil {
		return err
	}
	defer m.Close()
	m.SyncMissingArtwork()
	m.SyncCovers(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncMusicFanArt(config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open()
	if err != nil {
		return err
	}
	defer m.Close()
	m.SyncFanArt(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncFilm(config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SyncSince(f.LastModified())
}

func syncTV(config *config.Config, mediaConfig *config.Config) error {
	log.Println("xxx syncTV")
	tv := tv.NewTV(mediaConfig)
	err := tv.Open()
	if err != nil {
		return err
	}
	defer tv.Close()
	return tv.SyncSince(tv.LastModified())
}

func syncFilmPosters(config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	f.SyncPosters(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncFilmBackdrops(config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	f.SyncBackdrops(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncFilmProfileImages(config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	f.SyncProfileImages(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVProfileImages(config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open()
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncProfileImages(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVBackdrops(config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open()
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncBackdrops(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVPosters(config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open()
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncPosters(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVStills(config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open()
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncStills(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncPodcasts(config *config.Config, mediaConfig *config.Config) error {
	p := podcast.NewPodcast(mediaConfig)
	err := p.Open()
	if err != nil {
		return err
	}
	defer p.Close()
	return p.Sync()
}

func createStations(config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open()
	if err != nil {
		return err
	}
	defer m.Close()
	m.DeleteStations()
	m.CreateStations()
	return nil
}

func Job(config *config.Config, name string) error {
	list, err := assignedMedia(config)
	if err != nil {
		return err
	}
	for _, mediaName := range list {
		mediaConfig, err := mediaConfig(config, mediaName)
		if err != nil {
			return err
		}
		switch name {
		case "backdrops":
			syncTVBackdrops(config, mediaConfig)
			syncFilmBackdrops(config, mediaConfig)
		case "covers":
			syncMusicCovers(config, mediaConfig)
		case "fanart":
			syncMusicFanArt(config, mediaConfig)
		case "lastfm":
			syncMusicPopular(config, mediaConfig)
			syncMusicSimilar(config, mediaConfig)
		case "media":
			syncMusic(config, mediaConfig)
			syncFilm(config, mediaConfig)
			syncPodcasts(config, mediaConfig)
		case "music":
			syncMusic(config, mediaConfig)
		case "popular":
			syncMusicPopular(config, mediaConfig)
		case "podcasts":
			syncPodcasts(config, mediaConfig)
		case "posters":
			syncTVPosters(config, mediaConfig)
			syncFilmPosters(config, mediaConfig)
		case "profiles":
			syncTVProfileImages(config, mediaConfig)
			syncFilmProfileImages(config, mediaConfig)
		case "similar":
			syncMusicSimilar(config, mediaConfig)
		case "stills":
			syncTVStills(config, mediaConfig)
		case "film":
			syncFilm(config, mediaConfig)
		case "tv":
			syncTV(config, mediaConfig)
		case "stations":
			createStations(config, mediaConfig)
		}
	}
	return nil
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	name := r.PathValue("name")
	go func() {
		err := Job(ctx.Config(), name)
		if err != nil {
			log.Println(name, err)
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}
