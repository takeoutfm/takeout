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

// This file is based on the following:
// https://github.com/coreos/go-systemd/blob/main/daemon/sdnotify.go
// https://github.com/miniflux/v2/blob/main/internal/systemd/systemd.go

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"takeoutfm.dev/takeout/lib/log"
	"time"
)

const (
	EnvCacheDirectory   = "CACHE_DIRECTORY"
	EnvConfigDirectory  = "CONFIGURATION_DIRECTORY"
	EnvLogsDirectory    = "LOGS_DIRECTORY"
	EnvRuntimeDirectory = "RUNTIME_DIRECTORY"
	EnvStateDirectory   = "STATE_DIRECTORY"

	EnvNotifySocket     = "NOTIFY_SOCKET"
	EnvWatchdogInterval = "WATCHDOG_USEC"
	EnvWatchdogPid      = "WATCHDOG_PID"

	NotifyReady     = "READY=1"
	NotifyReloading = "RELOADING=1"
	NotifyStopping  = "STOPPING=1"
	NotifyWatchdog  = "WATCHDOG=1"
)

func getenv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}
	return val
}

func GetCacheDirectory(def string) string {
	return getenv(EnvCacheDirectory, def)
}

func GetConfigDirectory(def string) string {
	return getenv(EnvConfigDirectory, def)
}

func GetLogsDirectory(def string) string {
	return getenv(EnvLogsDirectory, def)
}

func GetRuntimeDirectory(def string) string {
	return getenv(EnvRuntimeDirectory, def)
}

func GetStateDirectory(def string) string {
	return getenv(EnvStateDirectory, def)
}

// Notify sends a message to the init daemon. Error can be ignored.
func Notify(state string) error {
	socketAddr := &net.UnixAddr{
		Name: os.Getenv(EnvNotifySocket),
		Net:  "unixgram",
	}
	if socketAddr.Name == "" {
		return fmt.Errorf("NOTIFY_SOCKET not available")
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err = conn.Write([]byte(state)); err != nil {
		return err
	}
	return nil
}

func HasSystemd() bool {
	return os.Getenv(EnvNotifySocket) != ""
}

func HasWatchdog() bool {
	wpid := os.Getenv(EnvWatchdogPid)
	if wpid == "" {
		return false
	}
	p, err := strconv.Atoi(wpid)
	if err != nil {
		return false
	}
	return os.Getpid() == p
}

// WatchdogInterval returns watchdog information for a service. Processes
// should call Notify(NotifyWatchdog) every time / 2.
func WatchdogInterval() (time.Duration, error) {
	if !HasWatchdog() {
		return 0, fmt.Errorf("no watchdog")
	}

	wusec := os.Getenv(EnvWatchdogInterval)
	if wusec == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(wusec)
	if err != nil {
		return 0, fmt.Errorf("error converting WATCHDOG_USEC: %s", err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("error WATCHDOG_USEC must be a positive number")
	}
	interval := time.Duration(n) * time.Microsecond

	return interval, nil
}

func WatchdogNotify(done <-chan bool) error {
	interval, err := WatchdogInterval()
	if err != nil {
		return err
	}

	go func() {
		err = Notify(NotifyReady)
		if err != nil {
			log.Println("notify ready", err)
		}
		for {
			select {
			case <-done:
				goto stop
			case <-time.After(interval / 2):
				err = Notify(NotifyWatchdog)
				if err != nil {
					log.Println("notify watchdog", err)
				}
			}
		}
	stop:
		err = Notify(NotifyStopping)
		if err != nil {
			log.Println("notify stop", err)
		}
	}()

	return nil
}

func StartWatchdogNotify() {
	if HasWatchdog() == false {
		return
	}
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		close(quit)
		done <- true
	}()
	WatchdogNotify(done)
}
