// Copyright 2024 defsub
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
	"testing"
	"time"
)

func TestNewDateRange(t *testing.T) {
	v := NewDateRange(time.Now(), time.Now())
	if v.Start.IsZero() || v.End.IsZero() {
		t.Error("expect non-zero time")
	}
}

func TestIsZero(t *testing.T) {
	d := NewDateRange(time.Now(), time.Now())
	if d.IsZero() {
		t.Error("expect not isZero")
	}
	z := NewDateRange(time.Time{}, time.Time{})
	if z.IsZero() == false {
		t.Error("expect zero")
	}
}

func TestAfterDate(t *testing.T) {
	l := time.Now().Location()
	d := NewDateRange(
		time.Date(2024, time.February, 15, 10, 15, 59, 0, l),
		time.Date(2024, time.February, 16, 10, 15, 59, 0, l))
	if d.AfterDate() != "2024-02-15" {
		t.Error("expect after date")
	}
	if d.BeforeDate() != "2024-02-16" {
		t.Error("expect before date")
	}
}

func TestYMD(t *testing.T) {
	now := time.Now()
	if YMD(now) != now.Format("2006-01-02") {
		t.Errorf("expect yyyy-mm-dd got %s", YMD(now))
	}
}

func TestDayCount(t *testing.T) {
	l := time.Now().Location()

	d := NewDateRange(
		time.Date(2024, time.December, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.December, 9, 10, 15, 59, 0, l))
	if d.DayCount() != 9 {
		t.Error("expect 9 days")
	}

	d = NewDateRange(
		time.Date(2024, time.December, 1, 0, 0, 0, 0, l),
		time.Date(2024, time.December, 1, 23, 59, 59, 0, l))
	if d.DayCount() != 1 {
		t.Error("expeect 1 days")
	}

	d = NewDateRange(
		time.Date(2024, time.December, 1, 0, 0, 0, 0, l),
		time.Date(2025, time.January, 1, 23, 59, 59, 0, l))
	if d.DayCount() != 32 {
		t.Error("expeect 32 days")
	}
}

func TestIsDay(t *testing.T) {
	start := StartOfDay(time.Now())
	end := EndOfDay(start)

	d := NewDateRange(start, end)
	if d.IsDay() == false {
		t.Error("expect is day")
	}

	d = NewDateRange(time.Now(), time.Now().AddDate(0, 0, 1))
	if d.IsDay() {
		t.Error("expect not a day")
	}
}

func TestIsWeek(t *testing.T) {
	start := StartOfWeek(time.Now())
	end := EndOfWeek(start)

	d := NewDateRange(start, end)
	if d.IsWeek() == false {
		t.Error("expect is week")
	}

	d = NewDateRange(time.Now(), time.Now().AddDate(0, 0, 6))
	if d.IsWeek() {
		t.Error("expect not a week")
	}
}

func TestIsMonth(t *testing.T) {
	start := StartOfMonth(time.Now())
	end := EndOfMonth(start)

	d := NewDateRange(start, end)
	if d.IsMonth() == false {
		t.Error("expect is month")
	}

	d = NewDateRange(time.Now(), time.Now().AddDate(0, 0, 20))
	if d.IsMonth() {
		t.Error("expect not a month")
	}
}

func TestIsYear(t *testing.T) {
	start := StartOfYear(time.Now())
	end := EndOfYear(start)

	d := NewDateRange(start, end)
	if d.IsYear() == false {
		t.Error("expect is year")
	}

	d = NewDateRange(time.Now(), time.Now().AddDate(0, 0, 200))
	if d.IsYear() {
		t.Error("expect not a year")
	}
}

func TestMonthCount(t *testing.T) {
	l := time.Now().Location()

	d := NewDateRange(
		time.Date(2024, time.December, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.December, 31, 10, 15, 59, 0, l))
	if d.MonthCount() != 1 {
		t.Error("expect 1 month")
	}

	d = NewDateRange(
		time.Date(2024, time.January, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.December, 31, 10, 15, 59, 0, l))
	if d.MonthCount() != 12 {
		t.Error("expect 12 months")
	}

	d = NewDateRange(
		time.Date(2023, time.January, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.January, 31, 10, 15, 59, 0, l))
	if d.MonthCount() != 13 {
		t.Error("expect 13 months")
	}

	d = NewDateRange(
		time.Date(2023, time.January, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.June, 1, 10, 15, 59, 0, l))
	if d.MonthCount() != 18 {
		t.Error("expect 18 months")
	}
}

func TestPreviousYear(t *testing.T) {
	l := time.Now().Location()

	d := NewDateRange(
		time.Date(2024, time.December, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.December, 9, 10, 15, 59, 0, l))

	p := d.PreviousYear()
	if p.Start.Year() != 2023 {
		t.Error("expect start 2023")
	}
	if p.End.Year() != 2023 {
		t.Error("expect end 2023")
	}
	if p.Start.Month() != time.January {
		t.Error("expect start jan")
	}
	if p.End.Month() != time.December {
		t.Error("expect end dec")
	}
	if p.Start.Day() != 1 {
		t.Error("expect start 1st")
	}
	if p.End.Day() != 31 {
		t.Error("expect end 31st")
	}
}

func TestPreviousMonth(t *testing.T) {
	l := time.Now().Location()

	d := NewDateRange(
		time.Date(2024, time.December, 1, 10, 15, 59, 0, l),
		time.Date(2024, time.December, 9, 10, 15, 59, 0, l))

	p := d.PreviousMonth()
	if p.Start.Year() != 2024 {
		t.Error("expect start 2024")
	}
	if p.End.Year() != 2024 {
		t.Error("expect end 2024")
	}
	if p.Start.Month() != time.November {
		t.Error("expect start nov")
	}
	if p.End.Month() != time.November {
		t.Error("expect end nov")
	}
	if p.Start.Day() != 1 {
		t.Error("expect start 1st")
	}
	if p.End.Day() != 30 {
		t.Error("expect end 30th")
	}
}
