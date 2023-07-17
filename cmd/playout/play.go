// Copyright 2023 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/takeoutfm/takeout/client/api"
	"github.com/takeoutfm/takeout/client/player"
	"github.com/takeoutfm/takeout/lib/date"
	"github.com/takeoutfm/takeout/lib/spiff"
	"github.com/takeoutfm/takeout/lib/str"
	"github.com/takeoutfm/takeout/progress"
)

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doPlay()
	},
}

func doPlay() error {
	playout := NewPlayout()

	result, err := api.Progress(playout)
	if err != nil {
		return err
	}

	offsets := make(map[string]progress.Offset)
	for _, o := range result.Offsets {
		offsets[o.ETag] = o
	}

	var playlist *spiff.Playlist

	if len(query) == 0 {
		var sb strings.Builder
		if len(artist) > 0 {
			eq(&sb, "artist", artist)
		}
		if len(release) > 0 {
			eq(&sb, "release", release)
		}
		if len(title) > 0 {
			eq(&sb, "title", title)
		}
		if len(genre) > 0 {
			eq(&sb, "genre", genre)
		}
		if popular {
			eq(&sb, "type", "popular")
		}
		if single {
			eq(&sb, "type", "single")
		}
		if cover {
			eq(&sb, "type", "cover")
		}
		if live {
			eq(&sb, "type", "live")
		}
		if len(before) > 0 {
			// before is inclusive
			lte(&sb, "first_date", date.ParseDate(before))
		}
		if len(after) > 0 {
			// after is inclusive
			gte(&sb, "first_date", date.ParseDate(after))
		}
		query = sb.String()
	}

	if len(query) > 0 {
		playlist, err = api.Replace(playout, query, shuffle)
	} else {
		playlist, err = api.Playlist(playout)
	}
	if err != nil {
		return err
	}

	for i, t := range playlist.Spiff.Entries {
		fmt.Printf("%2d. %-37s %-s\n", i,
			str.TrimLength(t.Creator, 37),
			str.TrimLength(t.Title, 50))
	}

	onTrack := func(p *player.Player) {
		if p.IsStream() == false {
			go func() {
				api.Position(playout, p.Index(), 0)
			}()
		}
	}

	options := &player.Options{Repeat: repeat, OnTrack: onTrack, OnError: onError}
	player := player.NewPlayer(playout, playlist, options)
	done := make(chan struct{})
	go func() {
		seconds := time.Tick(time.Second * 1)
		for {
			select {
			case <-seconds:
				update(player)
			case <-done:
				return
			}
		}
	}()
	player.Start()
	done <- struct{}{}

	return nil
}

func onError(p *player.Player, err error) {
	fmt.Printf("Got err %v\n", err)
	p.Next()
}

func mmss(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func update(p *player.Player) {
	pos, len := p.Position()
	fmt.Printf("[%s - %s] %s / %s\r", mmss(pos), mmss(len), p.Artist(), p.Title())
}

func add(sb *strings.Builder, key, op, value string) {
	if sb.Len() > 0 {
		sb.WriteString(` `)
	}
	sb.WriteString(`+`)
	sb.WriteString(key)
	sb.WriteString(op)
	sb.WriteString(`"`)
	sb.WriteString(value)
	sb.WriteString(`"`)
}

func eq(sb *strings.Builder, key, value string) {
	add(sb, key, ":", value)
}

func lte(sb *strings.Builder, key string, value time.Time) {
	add(sb, key, ":<=", value.Format("2006-01-02"))
}

func gte(sb *strings.Builder, key string, value time.Time) {
	add(sb, key, ":>=", value.Format("2006-01-02"))
}

var query string
var shuffle bool

var genre string
var artist string
var release string
var title string
var single bool
var popular bool
var cover bool
var live bool
var before string
var after string
var repeat bool

func init() {
	playCmd.Flags().StringVarP(&query, "query", "q", "", "search query")
	playCmd.Flags().StringVarP(&genre, "genre", "g", "", "genre")
	playCmd.Flags().StringVarP(&artist, "artist", "a", "", "artist")
	playCmd.Flags().StringVarP(&release, "release", "r", "", "release/album name")
	playCmd.Flags().StringVarP(&title, "title", "t", "", "song title")
	playCmd.Flags().BoolVarP(&shuffle, "shuffle", "x", false, "radio shuffle")
	playCmd.Flags().BoolVarP(&single, "singles", "s", false, "songs released as singles")
	playCmd.Flags().BoolVarP(&popular, "popular", "p", false, "popular songs")
	playCmd.Flags().BoolVarP(&cover, "cover", "c", false, "cover songs")
	playCmd.Flags().BoolVarP(&live, "live", "l", false, "songs performed live")
	playCmd.Flags().StringVar(&before, "before", "", "released in/on or before")
	playCmd.Flags().StringVar(&after, "after", "", "released in/on or after")
	playCmd.Flags().BoolVar(&repeat, "repeat", false, "repeat playlist")
	rootCmd.AddCommand(playCmd)
}
