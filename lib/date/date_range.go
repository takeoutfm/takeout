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
	"fmt"
	"time"
)

type DateRange struct {
	Start time.Time
	End   time.Time
}

func NewDateRange(a, b time.Time) DateRange {
	return DateRange{Start: a, End: b}
}

func (d *DateRange) IsZero() bool {
	return d.Start.IsZero() && d.End.IsZero()
}

func (d *DateRange) AfterDate() string {
	return YMD(d.Start)
}

func (d *DateRange) BeforeDate() string {
	return YMD(d.End)
}

func YMD(t time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d", t.Year(), t.Month(), t.Day())
}

func NewInterval(t time.Time, name string) DateRange {
	var start, end time.Time
	switch name {
	case "recent":
		// TODO make 30 configurable
		start, end = StartOfDay(BackDays(t, 30)), EndOfDay(t)
	case "today", "day":
		start, end = StartOfDay(t), EndOfDay(t)
	case "yesterday":
		start, end = StartOfYesterday(t), EndOfYesterday(t)
	case "week", "thisweek":
		start, end = StartOfWeek(t), EndOfWeek(t)
	case "month", "thismonth":
		start, end = StartOfMonth(t), EndOfMonth(t)
	case "year", "thisyear":
		start, end = StartOfYear(t), EndOfYear(t)
	case "lastweek":
		start, end = StartOfPreviousWeek(t), EndOfPreviousWeek(t)
	case "lastmonth":
		start, end = StartOfPreviousMonth(t), EndOfPreviousMonth(t)
	case "lastyear":
		start, end = StartOfPreviousYear(t), EndOfPreviousYear(t)
	case "all", "":
		start, end = DayZero(), time.Now()
	default:
		d := ParseDate(name)
		if d.IsZero() {
			start, end = d, d
		} else {
			start = StartOfDay(d)
			end = EndOfDay(d)
		}
	}
	return NewDateRange(start, end)
}
