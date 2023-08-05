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
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/takeoutfm/takeout/internal/playout"
	"github.com/takeoutfm/takeout/lib/date"
)

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		if visual == false && simple == false {
			visual = true
		}
		return doPlay()
	},
}

func doPlay() error {
	var options playout.PlayOptions

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
		options.Query = sb.String()
	} else {
		options.Query = query
	}

	options.Stream = stream
	options.Radio = radio
	options.Repeat = repeat
	options.Shuffle = shuffle
	options.Visual = visual
	options.Simple = simple

	return NewPlayout().Play(options)
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

var after string
var artist string
var before string
var cover bool
var genre string
var live bool
var popular bool
var query string
var radio string
var release string
var repeat bool
var shuffle bool
var simple bool
var single bool
var stream string
var title string
var visual bool

func init() {
	playCmd.Flags().StringVarP(&query, "query", "q", "", "search query")
	playCmd.Flags().StringVar(&stream, "stream", "", "name of radio stream")
	playCmd.Flags().StringVar(&radio, "radio", "", "name of radio station")

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

	playCmd.Flags().BoolVar(&simple, "simple", false, "use simple text interface")
	playCmd.Flags().BoolVar(&visual, "visual", false, "use visual text interface")

	rootCmd.AddCommand(playCmd)
}
