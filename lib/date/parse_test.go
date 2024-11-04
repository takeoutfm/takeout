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
	"testing"
	"time"
)

func TestParse1123(t *testing.T) {
	list := []string{
		"Fri, 6 Nov 2020 19:32:35 +0000",
		"Fri,  6 Nov 2020 19:32:35 +0000",
		"Fri, 06 Nov 2020 19:32:35 +0000",
	}
	for _, s := range list {
		d := ParseRFC1123(s)
		if d.Day() != 6 {
			t.Errorf("wrong day got %d\n", d.Day())
		}
		if d.Month() != time.November {
			t.Errorf("wrong month got %s\n", d.Month())
		}
		if d.Year() != 2020 {
			t.Errorf("wrong year got %d\n", d.Year())
		}
	}
}

func TestStartOfWeek(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	s := StartOfWeek(d)
	if s.Weekday() != time.Monday {
		t.Errorf("expect Monday")
	}
	if s.Day() != 21 {
		t.Errorf("expect 21st")
	}
}

func TestEndOfWeek(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	e := EndOfWeek(d)
	if e.Weekday() != time.Sunday {
		t.Errorf("expect Sunday")
	}
	if e.Day() != 27 {
		t.Errorf("expect 27th")
	}
}

func TestStartOfPreviousWeek(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	s := StartOfPreviousWeek(d)
	if s.Weekday() != time.Monday {
		t.Errorf("expect Monday")
	}
	if s.Day() != 14 {
		t.Errorf("expect 14th")
	}
}

func TestEndOfPreviousWeek(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	e := EndOfPreviousWeek(d)
	if e.Weekday() != time.Sunday {
		t.Errorf("expect Sunday")
	}
	if e.Day() != 20 {
		t.Errorf("expect 20th")
	}
}

func TestStartOfMonth(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	s := StartOfMonth(d)
	if s.Weekday() != time.Tuesday {
		t.Errorf("expect Tuesday")
	}
}

func TestEndMonth(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	s := EndOfMonth(d)
	if s.Weekday() != time.Thursday {
		t.Errorf("expect Thursday")
	}
}

func TestStartOfYear(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	s := StartOfYear(d)
	if s.Weekday() != time.Monday {
		t.Errorf("expect Monday")
	}
}

func TestEndOfYear(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	e := EndOfYear(d)
	if e.Weekday() != time.Tuesday {
		t.Errorf("expect Tuesday")
	}
}

func TestStartOfPreviousYear(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	s := StartOfPreviousYear(d)
	if s.Weekday() != time.Sunday {
		t.Errorf("expect Sunday")
	}
}

func TestEndOfPreviousYear(t *testing.T) {
	d := ParseRFC1123("Fri, 25 Oct 2024 16:54:06 -0700")
	e := EndOfPreviousYear(d)
	if e.Weekday() != time.Sunday {
		t.Errorf("expect Sunday")
	}
}

func TestParseRFC9557(t *testing.T) {
	ts, err := ParseRFC9557("2024-10-31T11:12:13-07:00[America/Los_Angeles]")
	if err != nil {
		t.Error(err)
	}
	if ts.Year() != 2024 {
		t.Error("expect 2024")
	}
	if ts.Month() != time.October {
		t.Error("expect oct")
	}
	if ts.Day() != 31 {
		t.Error("expect 21")
	}
	if ts.Hour() != 11 {
		t.Errorf("expect hour 11 got %d", ts.Hour())
	}
	if ts.Minute() != 12 {
		t.Error("expect hour 12")
	}
	if ts.Second() != 13 {
		t.Error("expect hour 13")
	}
	tz, offset := ts.Zone()
	if tz != "PDT" {
		t.Error("expect PDT")
	}
	if offset != -25200 {
		t.Error("expect offset")
	}
}

func TestParseRFC9557Z(t *testing.T) {
	ts, err := ParseRFC9557("2024-10-31T17:49:44Z[America/Los_Angeles]")
	// expect 10:49:44 AM PDT
	if err != nil {
		t.Error(err)
	}
	if ts.Year() != 2024 {
		t.Error("expect 2024")
	}
	if ts.Month() != time.October {
		t.Error("expect oct")
	}
	if ts.Day() != 31 {
		t.Error("expect 21")
	}
	if ts.Hour() != 10 {
		t.Errorf("expect hour 10 got %d", ts.Hour())
	}
	if ts.Minute() != 49 {
		t.Error("expect hour 49")
	}
	if ts.Second() != 44 {
		t.Error("expect hour 44")
	}
	tz, offset := ts.Zone()
	if tz != "PDT" {
		t.Error("expect PDT")
	}
	if offset != -25200 {
		t.Error("expect offset")
	}
}

func TestCombined(t *testing.T) {
	ts, err := ParseRFC9557("2024-10-31T17:49:44Z[America/Los_Angeles]")
	if err != nil {
		t.Error(err)
	}
	s := StartOfYear(ts)
	if s.Weekday() != time.Monday {
		t.Errorf("expect Monday")
	}
	e := EndOfYear(ts)
	if e.Weekday() != time.Tuesday {
		t.Errorf("expect Tuesday")
	}
}
