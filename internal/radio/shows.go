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

// show is a regular show on a live station.
type show struct {
	Name        string
	Query       string
	Released    time.Duration
	Image       string
	Description string
	Layout      string
	Match       string
	hour, min   int
	Length      time.Duration
}

// newShow checks a show from the config: it needs a name, music and a valid
// start time.
func newShow(c config.RadioShow) (show, error) {
	s := show{
		Name:        c.Name,
		Query:       c.Music,
		Released:    c.Released,
		Image:       c.Image,
		Description: c.Description,
		Layout:      c.Layout,
		Match:       c.Match,
		Length:      c.Length,
	}
	if s.Name == "" {
		return s, fmt.Errorf("show without a name")
	}
	if s.Query == "" && s.Released <= 0 {
		return s, fmt.Errorf("show %s has no Music or Released", s.Name)
	}
	if s.Length <= 0 {
		s.Length = 24 * time.Hour
	}
	if c.At != "" {
		at, err := time.Parse("15:04", c.At)
		if err != nil {
			return s, fmt.Errorf("show %s: bad At %q", s.Name, c.At)
		}
		s.hour, s.min = at.Hour(), at.Minute()
	}
	return s, nil
}

// query is the show's search query as of now: its Music, limited to
// recordings first released within Released.
func (s show) query(now time.Time) string {
	q := s.Query
	if s.Released > 0 {
		since := now.Add(-s.Released).Format("2006-01-02")
		q = strings.TrimSpace(fmt.Sprintf(`%s +first_date:>="%s"`, q, since))
	}
	return q
}

// slot is one airing of a show.
type slot struct {
	Show       int // index into the station's shows
	Start, End time.Time
}

// slots returns the airings of shows overlapping [from, to), by start time,
// with start times in loc.
func slots(shows []show, from, to time.Time, loc *time.Location) []slot {
	var list []slot
	for i, s := range shows {
		// a show that started a day or more earlier can still be on
		first := from.Add(-s.Length).In(loc)
		day := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, loc)
		for ; day.Before(to); day = day.AddDate(0, 0, 1) {
			if s.Layout != "" && !date.MatchTime(s.Layout, s.Match, day) {
				continue
			}
			start := time.Date(day.Year(), day.Month(), day.Day(), s.hour, s.min, 0, 0, loc)
			end := start.Add(s.Length)
			if start.Before(to) && end.After(from) {
				list = append(list, slot{Show: i, Start: start, End: end})
			}
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Start.Before(list[j].Start) })
	return list
}

// at returns the slot on at t, the earliest starting if they overlap, and
// otherwise the start of the next slot (or the zero time).
func at(list []slot, t time.Time) (slot, bool, time.Time) {
	for _, s := range list {
		if !s.Start.After(t) && s.End.After(t) {
			return s, true, time.Time{}
		}
	}
	for _, s := range list {
		if s.Start.After(t) {
			return slot{}, false, s.Start
		}
	}
	return slot{}, false, time.Time{}
}
