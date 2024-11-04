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

package date

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidZoneFormat = errors.New("invalid zone format")
)

// Parse a date string to time in format yyyy-mm-dd, yyyy-mm, yyyy.
func ParseDate(date string) (t time.Time) {
	if date == "" {
		return t
	}
	var err error
	// TODO is this done with a single call?
	t, err = time.Parse("2006-1-2", date)
	if err != nil {
		t, err = time.Parse("2006-1", date)
		if err != nil {
			t, err = time.Parse("2006", date)
			if err != nil {
				t = DayZero()
			}
		}
	}
	return t
}

const (
	RFC1123_1  = "Mon, _2 Jan 2006 15:04:05 MST"
	RFC1123_2  = "Mon, 2 Jan 2006 15:04:05 MST"
	RFC1123Z_1 = "Mon, _2 Jan 2006 15:04:05 -0700"
	RFC1123Z_2 = "Mon, 2 Jan 2006 15:04:05 -0700"

	RFC3339_Local = "2006-01-02T15:04:05Z"
)

// Mon, 02 Jan 2006 15:04:05 MST
// Tue, 07 Dec 2021 19:57:22 -0500
// Fri, 6 Nov 2020 19:32:35 +0000
func ParseRFC1123(date string) (t time.Time) {
	if date == "" {
		return t
	}
	var err error
	layouts := []string{time.RFC1123, time.RFC1123Z, RFC1123_1, RFC1123_2, RFC1123Z_1, RFC1123Z_2}
	for _, layout := range layouts {
		t, err = time.Parse(layout, date)
		if err == nil {
			return t
		}
	}
	return DayZero()
}

// const (
// 	Simple12 = "Jan 02 03:04 PM"
// 	Simple24 = "Jan 02 15:04"
// )

// func Format(t time.Time) string {
// 	return t.Format(Simple12)
// }

func FormatJson(t time.Time) string {
	return t.Format(time.RFC3339)
}

func DayZero() time.Time {
	return time.Time{}
}

func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

func StartOfYesterday(t time.Time) time.Time {
	return StartOfDay(BackDay(t))
}

func EndOfYesterday(t time.Time) time.Time {
	return EndOfDay(BackDay(t))
}

// weeks are ISO weeks that are Monday..Sunday
// https://en.wikipedia.org/wiki/ISO_8601

func StartOfWeek(t time.Time) time.Time {
	t = StartOfDay(t)
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}
	return t
}

func EndOfWeek(t time.Time) time.Time {
	t = EndOfDay(t)
	for t.Weekday() != time.Sunday {
		t = t.AddDate(0, 0, 1)
	}
	return t
}

func StartOfPreviousWeek(t time.Time) time.Time {
	t = StartOfWeek(t)
	t = t.AddDate(0, 0, -7)
	return t
}

func EndOfPreviousWeek(t time.Time) time.Time {
	t = EndOfWeek(t)
	t = t.AddDate(0, 0, -7)
	return t
}

func StartOfMonth(t time.Time) time.Time {
	t = StartOfDay(t)
	for t.Day() != 1 {
		t = t.AddDate(0, 0, -1)
	}
	return t
}

func EndOfMonth(t time.Time) time.Time {
	t = EndOfDay(t)
	m := t.Month()
	for t.Month() == m {
		t = t.AddDate(0, 0, 1)
	}
	t = t.AddDate(0, 0, -1)
	return t
}

func StartOfPreviousMonth(t time.Time) time.Time {
	t = StartOfDay(BackMonth(t))
	for t.Day() != 1 {
		t = t.AddDate(0, 0, -1)
	}
	return t
}

func EndOfPreviousMonth(t time.Time) time.Time {
	t = EndOfDay(BackMonth(t))
	m := t.Month()
	for t.Month() == m {
		t = t.AddDate(0, 0, 1)
	}
	t = t.AddDate(0, 0, -1)
	return t
}

func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
}

func EndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.December, 31, 23, 59, 59, 0, t.Location())
}

func StartOfPreviousYear(t time.Time) time.Time {
	t = t.AddDate(-1, 0, 0)
	return StartOfYear(t)
}

func EndOfPreviousYear(t time.Time) time.Time {
	t = t.AddDate(-1, 0, 0)
	return EndOfYear(t)
}

func BackDay(t time.Time) time.Time {
	return BackDays(t, 1)
}

func BackDays(t time.Time, days int) time.Time {
	t = t.AddDate(0, 0, -days)
	return t
}

func BackMonth(t time.Time) time.Time {
	t = t.AddDate(0, -1, 0)
	return t
}

func BackYear(t time.Time) time.Time {
	t = t.AddDate(-1, 0, 0)
	return t
}

// 1996-12-19T16:39:57-08:00[America/Los_Angeles]
func ParseRFC9557(value string) (time.Time, error) {
	s := strings.Index(value, "[")
	if s == -1 || s+1 == len(value) {
		return DayZero(), ErrInvalidZoneFormat
	}
	e := strings.Index(value, "]")
	if e == -1 || e+1 != len(value) {
		return DayZero(), ErrInvalidZoneFormat
	}

	zone := value[s+1:e]
	value = value[0:s]

	location, err := time.LoadLocation(zone)
	if err != nil {
		return DayZero(), err
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return DayZero(), err
	}
	return t.In(location), nil
}
