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

package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/alessio/shellescape.v1"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/str"
	"takeoutfm.dev/takeout/lib/tmdb"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		doit()
	},
}

var optQuery string
var optFile string
var optDef string
var optExt string

func fixColon(name string) string {
	// change "foo: bar" to "foo - bar"
	colon := regexp.MustCompile(`([A-Za-z0-9])\s*(:)\s`)
	name = colon.ReplaceAllString(name, "${1} - ")
	return name
}

func doit() {
	var query string
	season, episode, year := 0, 0, 0
	config := getConfig()
	if optFile != "" {
		tvRegexp := regexp.MustCompile(
			`(.+?)\s*\(([\d]+)\)\s+[^\d]*(S\d+E\d+)[^\d]*?(?:\s-\s(.+))?(_t\d+)?\.(mkv|mp4)$`)
		matches := tvRegexp.FindStringSubmatch(optFile)
		if matches != nil && len(matches) >= 4 {
			series := matches[1]
			year = str.Atoi(matches[2])
			detail := matches[3]
			ext := "." + matches[len(matches)-1]
			episodeRegexp := regexp.MustCompile(`(?i)S(\d+)E(\d+)`)
			matches := episodeRegexp.FindStringSubmatch(detail)
			if len(matches) == 3 {
				season = str.Atoi(matches[1])
				episode = str.Atoi(matches[2])
			}
			query = series
			query = strings.Replace(query, "_", " ", -1)
			if optExt == "" {
				optExt = ext
			}
		}
		if season == 0 && episode == 0 {
			// assume movie
			fileRegexp := regexp.MustCompile(`([^\/]+)_t\d+(\.mkv)$`)
			matches := fileRegexp.FindStringSubmatch(optFile)
			if matches != nil {
				query = matches[1]
				query = strings.Replace(query, "_", " ", -1)
				if optExt == "" {
					optExt = matches[2]
				}
			}
		}
	} else if optQuery != "" {
		query = optQuery
	}
	if query != "" {
		if season > 0 && episode > 0 {
			doSeries(config, query, year, season, episode)
		} else {
			doMovie(config, query)
		}
	}
}

func doSeries(config *config.Config, query string, year, season, episode int) {
	m := tmdb.NewTMDB(config.TMDB.Config, config.NewGetter())
	results, err := m.TVSearch(query)
	if err != nil {
		panic(err)
	}
	for _, v := range results {
		y := date.ParseDate(v.FirstAirDate).Year()
		if year > 0 && year != y {
			continue
		}

		vars := map[string]interface{}{
			"Series":    fixColon(v.Name),
			"Title":     fixColon(v.Name),
			"Year":      y,
			"Season":    fmt.Sprintf("%02d", season),
			"Episode":   fmt.Sprintf("%02d", episode),
			"Extension": optExt,
			"Ext":       optExt,
		}

		detail, err := m.TVDetail(v.ID)
		if err != nil {
			panic(err)
		}

		found := false
		for _, s := range detail.Seasons {
			if s.SeasonNumber == season && episode <= s.EpisodeCount {
				e, err := m.EpisodeDetail(v.ID, season, episode)
				if err != nil {
					panic(err)
				}
				vars["Name"] = fixColon(e.Name)
				found = true
				break
			}
		}
		if !found {
			continue
		}

		result := config.TMDB.SeriesTemplate.Execute(vars)

		vars["Ext"] = ".jpg"
		vars["Extension"] = ".jpg"
		cover := config.TMDB.SeriesTemplate.Execute(vars)

		fmt.Printf("%s\n", result)
		poster := m.OriginalPoster(v.PosterPath).String()
		fmt.Printf("%s\n", poster)
		if len(v.GenreIDs) > 0 {
			for i, id := range v.GenreIDs {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(m.TVGenre(id))
			}
			fmt.Println()
		}
		if optFile != "" {
			fmt.Printf("mv %s %s\n", shellescape.Quote(optFile),
				shellescape.Quote(result))
			fmt.Printf("wget -O %s %s\n", shellescape.Quote(cover),
				shellescape.Quote(poster))
		}
		fmt.Printf("\n")
	}
}

func doMovie(config *config.Config, query string) {
	m := tmdb.NewTMDB(config.TMDB.Config, config.NewGetter())
	results, err := m.MovieSearch(query)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	for _, v := range results {
		vars := map[string]interface{}{
			"Title":      fixColon(v.Title),
			"Year":       date.ParseDate(v.ReleaseDate).Year(),
			"Definition": optDef,
			"Def":        optDef,
			"Extension":  optExt,
			"Ext":        optExt,
		}
		title := config.TMDB.FileTemplate.Execute(vars)

		vars["Ext"] = ".jpg"
		vars["Extension"] = ".jpg"
		cover := config.TMDB.FileTemplate.Execute(vars)

		fmt.Printf("%s\n", title)
		poster := m.OriginalPoster(v.PosterPath).String()
		fmt.Printf("%s\n", poster)
		if len(v.GenreIDs) > 0 {
			for i, id := range v.GenreIDs {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(m.MovieGenre(id))
			}
			fmt.Println()
		}
		if optFile != "" {
			fmt.Printf("mv %s %s\n", shellescape.Quote(optFile),
				shellescape.Quote(title))
			fmt.Printf("wget -O %s %s\n", shellescape.Quote(cover),
				shellescape.Quote(poster))
		}
		fmt.Printf("\n")
	}
}

func init() {
	searchCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file")
	searchCmd.Flags().StringVarP(&optQuery, "query", "q", "", "search query")
	searchCmd.Flags().StringVarP(&optFile, "file", "f", "", "search file")
	searchCmd.Flags().StringVarP(&optDef, "def", "d", "", "SD, HD, UHD, 4k, etc")
	searchCmd.Flags().StringVarP(&optDef, "ext", "e", "", "file extension w/ dot")
	rootCmd.AddCommand(searchCmd)
}
