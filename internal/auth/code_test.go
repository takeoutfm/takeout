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

package auth

import (
	"strings"
	"testing"
)

func TestRandomCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		code := randomCode()
		if len(code) != CodeSize {
			t.Fatalf("expected %d chars, got %q", CodeSize, code)
		}
		for _, c := range code {
			if !strings.ContainsRune(CodeChars, c) {
				t.Fatalf("unexpected char %q in %q", c, code)
			}
		}
		seen[code] = true
	}
	// 33^6 possibilities; 1000 draws should almost never collide much
	if len(seen) < 990 {
		t.Fatalf("codes look non-random: %d unique of 1000", len(seen))
	}
}
