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
	"sync"
	"testing"
	"time"
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

func TestWindowCounter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	c := &windowCounter{limit: 3, window: time.Minute}
	for i := 0; i < 3; i++ {
		if !c.allow(now) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if c.allow(now) {
		t.Fatal("4th request in the window should be refused")
	}
	if c.allow(now.Add(59 * time.Second)) {
		t.Fatal("still inside the window")
	}
	if !c.allow(now.Add(time.Minute)) {
		t.Fatal("new window should allow again")
	}
}

// A code token must only work for the code it was issued for.
func TestCodeTokenBoundToCode(t *testing.T) {
	a := makeAuth(t)

	codeA, err := a.GenerateCode()
	if err != nil {
		t.Fatal(err)
	}
	codeB, err := a.GenerateCode()
	if err != nil {
		t.Fatal(err)
	}
	tokenA, err := a.NewCodeToken(codeA.Value)
	if err != nil {
		t.Fatal(err)
	}

	if err := a.CheckCodeTokenFor(tokenA, codeA.Value); err != nil {
		t.Fatalf("token should work for its own code: %v", err)
	}
	if err := a.CheckCodeTokenFor(tokenA, codeB.Value); err == nil {
		t.Fatal("token for one code must not work for another")
	}
	if err := a.CheckCodeTokenFor(tokenA, ""); err == nil {
		t.Fatal("empty code must be rejected")
	}
}

func TestWindowCounterConcurrent(t *testing.T) {
	c := &windowCounter{limit: 100, window: time.Hour}
	now := time.Unix(1_700_000_000, 0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if c.allow(now) {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	if allowed != 100 {
		t.Fatalf("allowed %d, want exactly 100", allowed)
	}
}
