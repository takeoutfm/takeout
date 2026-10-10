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

package server

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func testLimiter(max int) (*failureLimiter, *fakeClock) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	l := newFailureLimiter(max, 15*time.Minute, time.Minute, time.Hour)
	l.now = clock.now
	return l, clock
}

func TestLimiterLocksAfterMax(t *testing.T) {
	l, _ := testLimiter(3)
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("attempt %d should be allowed", i)
		}
		l.Fail("a")
	}
	ok, wait := l.Allow("a")
	if ok {
		t.Fatal("expected lockout after max failures")
	}
	if wait <= 0 || wait > time.Minute {
		t.Fatalf("unexpected wait %v", wait)
	}
	// other keys are unaffected
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other key should not be locked")
	}
}

func TestLimiterBackoffDoublesAndCaps(t *testing.T) {
	l, clock := testLimiter(1)
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute}
	for i, w := range want {
		l.Fail("a")
		ok, wait := l.Allow("a")
		if ok || wait != w {
			t.Fatalf("strike %d: ok=%v wait=%v want %v", i+1, ok, wait, w)
		}
		clock.advance(w + time.Second)
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("strike %d: should be allowed after lock expires", i+1)
		}
	}
	for i := 0; i < 20; i++ {
		l.Fail("a")
		clock.advance(l.maxLock + time.Second)
	}
	l.Fail("a")
	if _, wait := l.Allow("a"); wait > l.maxLock {
		t.Fatalf("lock %v exceeds cap %v", wait, l.maxLock)
	}
}

func TestLimiterForgivesAfterQuietWindow(t *testing.T) {
	l, clock := testLimiter(3)
	l.Fail("a")
	l.Fail("a")
	clock.advance(16 * time.Minute)
	l.Fail("a") // window restarted, so this is failure #1 not #3
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("old failures should have been forgiven")
	}
}

func TestLimiterReset(t *testing.T) {
	l, _ := testLimiter(2)
	l.Fail("a")
	l.Fail("a")
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("expected lockout")
	}
	l.Reset("a")
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("reset should clear lockout")
	}
}

func TestLimiterBoundedMemory(t *testing.T) {
	l, _ := testLimiter(5)
	l.maxKeys = 100
	for i := 0; i < 1000; i++ {
		l.Fail(fmt.Sprintf("k%d", i))
	}
	if n := len(l.entries); n > 100 {
		t.Fatalf("entries grew to %d, cap is 100", n)
	}
}

func TestLimiterConcurrent(t *testing.T) {
	l := newFailureLimiter(5, time.Minute, time.Second, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			k := fmt.Sprintf("k%d", i%5)
			for j := 0; j < 100; j++ {
				l.Allow(k)
				l.Fail(k)
				if j%10 == 0 {
					l.Reset(k)
				}
			}
		}(i)
	}
	wg.Wait()
}
