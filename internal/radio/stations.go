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
	"hash/fnv"
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
	Shows       []config.RadioShow
}

// plan sums up what decides the station's schedule, so the radio job can
// tell when it changed.
func (s station) plan() string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%q %s %+v", s.Query, s.MinContent, s.Shows)
	return fmt.Sprintf("%x", h.Sum64())
}

// onAir reports whether the station's date window includes day.
func (s station) onAir(day time.Time) bool {
	return s.Layout == "" || date.MatchTime(s.Layout, s.Match, day)
}

// lineup returns the configured stations in order: the lineup, then genres,
// decades and series. Only the Radio settings make stations; a name used
// twice keeps its first station.
func lineup(cfg *config.Config) []station {
	var list []station

	for _, e := range cfg.Radio.Lineup {
		list = append(list, station{
			Name:        e.Name,
			Query:       e.Music,
			Image:       e.Image,
			Description: e.Description,
			Layout:      e.Layout,
			Match:       e.Match,
			MinContent:  e.MinContent,
			Shows:       e.Shows,
		})
	}

	for _, genre := range cfg.Radio.Genres {
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

	for _, s := range cfg.Radio.Series {
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
