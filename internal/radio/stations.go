// Copyright 2026 defsub
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

package radio

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/date"
)

// station is a live station from the config: a hand-written lineup entry or
// an automatic genre, decade or series station.
type station struct {
	Name        string
	Query       string // music search query
	Image       string
	Description string
	Layout      string // optional date window, as in DateRecommend
	Match       string
	MinContent  time.Duration // 0 is Radio.MinContent
}

// onAir reports whether the station's date window includes day.
func (s station) onAir(day time.Time) bool {
	return s.Layout == "" || date.MatchTime(s.Layout, s.Match, day)
}

// lineup returns the configured stations in order: the lineup, then genres,
// decades and series. Unset Radio settings fall back to the older Music.Radio*
// ones, and genres to every artist genre (from genres). A name used twice
// keeps its first station.
func lineup(cfg *config.Config, genres func() []string) []station {
	var list []station

	entries := cfg.Radio.Lineup
	if len(entries) == 0 {
		// Music.RadioOther is a map, so give it a stable order
		var names []string
		for name := range cfg.Music.RadioOther {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			entries = append(entries, config.RadioEntry{Name: name, Music: cfg.Music.RadioOther[name]})
		}
	}
	for _, e := range entries {
		list = append(list, station{
			Name:        e.Name,
			Query:       e.Music,
			Image:       e.Image,
			Description: e.Description,
			Layout:      e.Layout,
			Match:       e.Match,
			MinContent:  e.MinContent,
		})
	}

	g := cfg.Radio.Genres
	if len(g) == 0 {
		g = cfg.Music.RadioGenres
	}
	if len(g) == 0 && genres != nil {
		g = genres()
	}
	for _, genre := range g {
		if genre == "" {
			continue
		}
		list = append(list, station{
			Name: strings.Title(genre),
			Query: fmt.Sprintf(`+genre:"%s" +type:single +popularity:<11 -artist:"Various Artists"`,
				genre),
		})
	}

	for _, d := range cfg.Radio.Decades {
		list = append(list, station{
			Name: fmt.Sprintf("%ds Top Tracks", d),
			Query: fmt.Sprintf(`+first_date:>="%d-01-01" +first_date:<="%d-12-31" +type:single +popularity:<11`,
				d, d+9),
		})
	}

	series := cfg.Radio.Series
	if len(series) == 0 {
		series = cfg.Music.RadioSeries
	}
	for _, s := range series {
		list = append(list, station{Name: s, Query: fmt.Sprintf(`+series:"%s"`, s)})
	}

	seen := make(map[string]bool)
	var result []station
	for _, s := range list {
		if s.Name == "" || s.Query == "" || seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		result = append(result, s)
	}
	return result
}
