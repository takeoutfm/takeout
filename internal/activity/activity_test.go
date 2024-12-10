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

package activity

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/model"
)

func TestFillGaps(t *testing.T) {
	l := time.Now().Location()

	start := time.Date(2024, time.December, 1, 0, 0, 0, 0, l)
	end := time.Date(2024, time.December, 31, 0, 0, 0, 0, l)

	counts := []model.ActivityCount{
		{Date: time.Date(2024, time.December, 4, 0, 0, 0, 0, l), Count: 10},
		{Date: time.Date(2024, time.December, 8, 0, 0, 0, 0, l), Count: 5},
		{Date: time.Date(2024, time.December, 28, 0, 0, 0, 0, l), Count: 1},
	}

	result := fillGaps(start, end, counts)
	total := 0
	zeroCount := 0
	for _, c := range result {
		//t.Logf("%s %d\n", date.YMD(c.Date), c.Count)
		if c.Count == 0 {
			zeroCount++
		} else {
			total += c.Count
		}
	}

	if len(result) != 31 {
		t.Error("expect 31 days")
	}

	if total != 16 {
		t.Error("expect 16 total")
	}

	if zeroCount != 28 {
		t.Error("expect 28 zeros")
	}
}

func TestFillMonthGaps(t *testing.T) {
	l := time.Now().Location()

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, l)
	end := time.Date(2024, time.December, 31, 0, 0, 0, 0, l)

	counts := []model.ActivityCount{
		{Date: time.Date(2024, time.February, 2, 0, 0, 0, 0, l), Count: 10},
		{Date: time.Date(2024, time.May, 8, 0, 0, 0, 0, l), Count: 5},
		{Date: time.Date(2024, time.October, 28, 0, 0, 0, 0, l), Count: 1},
	}

	result := fillMonthGaps(start, end, counts)
	total := 0
	zeroCount := 0
	for _, c := range result {
		//t.Logf("%s %d\n", date.YMD(c.Date), c.Count)
		if c.Count == 0 {
			zeroCount++
		} else {
			total += c.Count
		}
	}

	if len(result) != 12 {
		t.Error("expect 12 months")
	}

	if total != 16 {
		t.Error("expect 16 total")
	}

	if zeroCount != 9 {
		t.Error("expect 9 zeros")
	}
}
