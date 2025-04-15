// Copyright 2025 defsub
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

package systemd // import "takeoutfm.dev/takeout/lib/systemd"

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestWatchdog(t *testing.T) {
	os.Setenv("WATCHDOG_USEC", "20000000")
	os.Setenv("WATCHDOG_PID", fmt.Sprintf("%d", os.Getpid()))

	interval, err := WatchdogInterval()
	if err != nil {
		t.Fatal(err)
	}

	if time.Duration(interval).Seconds() != 20 {
		t.Fatal("expect 20 seconds")
	}
}

func TestNotify(t *testing.T) {
	err := Notify(NotifyReady)
	if err == nil {
		t.Fatal("expect error")
	}
}

func xTestWatchdogNotify(t *testing.T) {
	os.Setenv("WATCHDOG_USEC", "20000000")
	os.Setenv("WATCHDOG_PID", fmt.Sprintf("%d", os.Getpid()))

	done := make(chan bool, 1)
	err := WatchdogNotify(done)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(30) * time.Second)
	t.Log("sending done")
	done <- true
	time.Sleep(time.Duration(1) * time.Second)
}

func xTestStartWatchdogNotify(t *testing.T) {
	os.Setenv("WATCHDOG_USEC", "20000000")
	os.Setenv("WATCHDOG_PID", fmt.Sprintf("%d", os.Getpid()))
	StartWatchdogNotify()
	t.Log("waiting")
	time.Sleep(time.Duration(60) * time.Second)
}
