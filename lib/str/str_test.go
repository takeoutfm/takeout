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

package str

import (
	"testing"
)

func TestSplit(t *testing.T) {
	if len(Split("")) != 0 {
		t.Error("expect empty")
	}

	v := Split(" a ,  b  ,    c , d")
	if v[0] != "a" {
		t.Error("expect a")
	}
	if v[1] != "b" {
		t.Error("expect b")
	}
	if v[2] != "c" {
		t.Error("expect c")
	}
	if v[3] != "d" {
		t.Error("expect d")
	}
}

func TestAtoi(t *testing.T) {
	if Atoi("10") != 10 {
		t.Error("expect 10")
	}
	if Atoi("-10") != -10 {
		t.Error("expect -10")
	}
	if Atoi("foo") != 0 {
		t.Error("expect 0 on error")
	}
}

func TestItoa(t *testing.T) {
	if Itoa(10) != "10" {
		t.Error("expect 10")
	}
	if Itoa(-10) != "-10" {
		t.Error("expect -10")
	}
}

func TestSortTitle(t *testing.T) {
	if SortTitle("The Matrix Reloaded") != "Matrix Reloaded, The" {
		t.Error("expect, The")
	}
	if SortTitle("A Quiet Place") != "Quiet Place, A" {
		t.Error("expect, A")
	}
	if SortTitle("An American WereWolf in London") != "American WereWolf in London, An" {
		t.Error("expect, An")
	}
}

func TestTrimLength(t *testing.T) {
	if TrimLength("The Matrix Reloaded", 8) != "The M..." {
		t.Error("expect The M...")
	}
}

func TestTrimNulls(t *testing.T) {
	b := []byte{'h', 'e', 'l', 'l', 'o', 0}
	s := string(b)
	if len(s) != 6 {
		t.Error("expect 6")
	}
	s = TrimNulls(s)
	if len(s) != 5 {
		t.Error("expect 5")
	}
}
