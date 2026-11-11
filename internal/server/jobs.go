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
	"net/http"

	"github.com/go-co-op/gocron"

	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/internal/film"
	"takeoutfm.dev/takeout/internal/music"
	"takeoutfm.dev/takeout/internal/podcast"
	"takeoutfm.dev/takeout/internal/progress"
	"takeoutfm.dev/takeout/internal/tv"
	"takeoutfm.dev/takeout/lib/duration"
	"takeoutfm.dev/takeout/lib/log"
	"time"
)

type syncFunc func(ctx context.Context, config *config.Config, mediaConfig *config.Config) error

func schedule(ctx context.Context, config *config.Config) {
	scheduler := gocron.NewScheduler(time.UTC)

	mediaSync := func(ctx context.Context, d time.Duration, doit syncFunc, startImmediately bool) {
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
				err = doit(ctx, config, mediaConfig)
				if err != nil {
					log.Println(err)
				}
			}
		})
	}

	// music
	mediaSync(ctx, config.Music.SyncInterval, syncMusic, false)
	mediaSync(ctx, config.Music.PopularSyncInterval, syncMusicPopular, false)
	mediaSync(ctx, config.Music.SimilarSyncInterval, syncMusicSimilar, false)
	mediaSync(ctx, config.Music.CoverSyncInterval, syncMusicCovers, false)
	mediaSync(ctx, config.Music.DurationSyncInterval, syncMusicDurations, false)

	// podcasts
	mediaSync(ctx, config.Podcast.SyncInterval, syncPodcasts, false)

	// film
	mediaSync(ctx, config.Film.SyncInterval, syncFilm, false)
	mediaSync(ctx, config.Film.PosterSyncInterval, syncFilmPosters, false)
	mediaSync(ctx, config.Film.BackdropSyncInterval, syncFilmBackdrops, false)
	mediaSync(ctx, config.Film.DurationSyncInterval, syncFilmDurations, false)

	// tv
	mediaSync(ctx, config.TV.SyncInterval, syncTV, false)
	mediaSync(ctx, config.TV.PosterSyncInterval, syncTVPosters, false)
	mediaSync(ctx, config.TV.BackdropSyncInterval, syncTVBackdrops, false)
	mediaSync(ctx, config.TV.StillSyncInterval, syncTVStills, false)
	mediaSync(ctx, config.TV.DurationSyncInterval, syncTVDurations, false)

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

func syncMusic(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	syncOptions := music.NewSyncOptions()
	syncOptions.Since = m.LastModified()
	return m.Sync(ctx, syncOptions)
}

func syncArtwork(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	return m.SyncArtwork()
}

func syncWithOptions(ctx context.Context, mediaConfig *config.Config, syncOptions music.SyncOptions) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	return m.Sync(ctx, syncOptions)
}

func syncMusicPopular(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	return syncWithOptions(ctx, mediaConfig, music.NewSyncPopular())
}

func syncMusicSimilar(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	return syncWithOptions(ctx, mediaConfig, music.NewSyncSimilar())
}

func syncMusicCovers(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	m.SyncMissingArtwork()
	m.SyncCovers(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncMusicFanArt(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	m.SyncFanArt(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncFilm(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.SyncSince(ctx, f.LastModified())
}

func syncTV(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open(ctx)
	if err != nil {
		return err
	}
	defer tv.Close()
	return tv.SyncSince(ctx, tv.LastModified())
}

func syncFilmPosters(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()
	f.SyncPosters(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncFilmBackdrops(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()
	f.SyncBackdrops(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncFilmProfileImages(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()
	f.SyncProfileImages(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVProfileImages(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open(ctx)
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncProfileImages(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVBackdrops(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open(ctx)
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncBackdrops(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVPosters(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open(ctx)
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncPosters(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

func syncTVStills(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open(ctx)
	if err != nil {
		return err
	}
	defer tv.Close()
	tv.SyncStills(config.NewGetterWith(config.Server.ImageClient))
	return nil
}

// Range requests for reading media file headers.
var durationClient = &http.Client{Timeout: time.Minute}

func syncFilmDurations(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	f := film.NewFilm(mediaConfig)
	err := f.Open(ctx)
	if err != nil {
		return err
	}
	defer f.Close()
	results := f.SyncDurations(ctx, durationClient, mediaConfig.Film.DurationSyncLimit)
	validateDurations(config, "movie", results)
	return nil
}

func syncTVDurations(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	tv := tv.NewTV(mediaConfig)
	err := tv.Open(ctx)
	if err != nil {
		return err
	}
	defer tv.Close()
	results := tv.SyncDurations(ctx, durationClient, mediaConfig.TV.DurationSyncLimit)
	validateDurations(config, "episode", results)
	return nil
}

func syncMusicDurations(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	results := m.SyncDurations(ctx, durationClient, mediaConfig.Music.DurationSyncLimit)
	logDurations("track", results, 0, 0)
	return nil
}

// validateDurations compares scanned video durations with those reported by
// players in watch progress, and logs any that differ by more than a minute.
func validateDurations(config *config.Config, kind string, results []duration.Result) {
	if len(results) == 0 {
		return
	}
	p := progress.NewProgress(config)
	if err := p.Open(); err != nil {
		log.Println(err)
		logDurations(kind, results, 0, 0)
		return
	}
	defer p.Close()

	var etags []string
	for _, r := range results {
		if r.Seconds > 0 {
			etags = append(etags, r.ETag)
		}
	}
	reported := p.Durations(etags)
	checked, mismatched := 0, 0
	for _, r := range results {
		d, ok := reported[r.ETag]
		if !ok || r.Seconds <= 0 {
			continue
		}
		checked++
		if diff := r.Seconds - d; diff > 60 || diff < -60 {
			mismatched++
			log.Printf("duration mismatch %s: scanned %ds, player %ds\n", r.Name, r.Seconds, d)
		}
	}
	logDurations(kind, results, checked, mismatched)
}

func logDurations(kind string, results []duration.Result, checked, mismatched int) {
	if len(results) == 0 {
		return
	}
	unreadable := 0
	for _, r := range results {
		if r.Seconds < 0 {
			unreadable++
		}
	}
	log.Printf("durations: %d %s files scanned, %d unreadable, %d checked against players, %d mismatched\n",
		len(results), kind, unreadable, checked, mismatched)
}

func syncPodcasts(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	p := podcast.NewPodcast(mediaConfig)
	err := p.Open()
	if err != nil {
		return err
	}
	defer p.Close()
	return p.Sync()
}

func createStations(ctx context.Context, config *config.Config, mediaConfig *config.Config) error {
	m := music.NewMusic(mediaConfig)
	err := m.Open(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	m.DeleteStations()
	m.CreateStations()
	return nil
}

func Job(ctx context.Context, config *config.Config, name string) error {
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
		case "artwork":
			syncArtwork(ctx, config, mediaConfig)
		case "backdrops":
			syncTVBackdrops(ctx, config, mediaConfig)
			syncFilmBackdrops(ctx, config, mediaConfig)
		case "durations":
			syncFilmDurations(ctx, config, mediaConfig)
			syncTVDurations(ctx, config, mediaConfig)
			syncMusicDurations(ctx, config, mediaConfig)
		case "covers":
			syncMusicCovers(ctx, config, mediaConfig)
		case "fanart":
			syncMusicFanArt(ctx, config, mediaConfig)
		case "lastfm":
			syncMusicPopular(ctx, config, mediaConfig)
			syncMusicSimilar(ctx, config, mediaConfig)
		case "media":
			syncMusic(ctx, config, mediaConfig)
			syncFilm(ctx, config, mediaConfig)
			syncTV(ctx, config, mediaConfig)
			syncPodcasts(ctx, config, mediaConfig)
		case "images":
			syncMusicCovers(ctx, config, mediaConfig)
			syncMusicFanArt(ctx, config, mediaConfig)
			syncFilmPosters(ctx, config, mediaConfig)
			syncFilmBackdrops(ctx, config, mediaConfig)
			syncFilmProfileImages(ctx, config, mediaConfig)
			syncTVPosters(ctx, config, mediaConfig)
			syncTVBackdrops(ctx, config, mediaConfig)
			syncTVStills(ctx, config, mediaConfig)
			syncTVProfileImages(ctx, config, mediaConfig)
		case "music":
			syncMusic(ctx, config, mediaConfig)
		case "popular":
			syncMusicPopular(ctx, config, mediaConfig)
		case "podcasts":
			syncPodcasts(ctx, config, mediaConfig)
		case "posters":
			syncTVPosters(ctx, config, mediaConfig)
			syncFilmPosters(ctx, config, mediaConfig)
		case "profiles":
			syncTVProfileImages(ctx, config, mediaConfig)
			syncFilmProfileImages(ctx, config, mediaConfig)
		case "similar":
			syncMusicSimilar(ctx, config, mediaConfig)
		case "stills":
			syncTVStills(ctx, config, mediaConfig)
		case "film":
			syncFilm(ctx, config, mediaConfig)
		case "tv":
			syncTV(ctx, config, mediaConfig)
		case "stations":
			createStations(ctx, config, mediaConfig)
		}
	}
	return nil
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := contextValue(r)
	name := r.PathValue("name")
	go func() {
		err := Job(r.Context(), ctx.Config(), name)
		if err != nil {
			log.Println(name, err)
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}
