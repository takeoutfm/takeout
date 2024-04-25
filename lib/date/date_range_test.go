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
	if v.after.IsZero() || v.before.IsZero() {
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
