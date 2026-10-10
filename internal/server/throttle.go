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
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/lib/log"
)

// failureLimiter tracks failed attempts per key. After max failures inside
// window the key is locked out. Each consecutive lockout doubles in length (up
// to maxLock) and the strike count only decays once the key has been quiet for
// a full window.
type failureLimiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	baseLock time.Duration
	maxLock  time.Duration
	maxKeys  int
	entries  map[string]*failureEntry
	now      func() time.Time
}

type failureEntry struct {
	failures    int
	windowStart time.Time
	strikes     int
	lockedUntil time.Time
}

func newFailureLimiter(max int, window, baseLock, maxLock time.Duration) *failureLimiter {
	return &failureLimiter{
		max:      max,
		window:   window,
		baseLock: baseLock,
		maxLock:  maxLock,
		maxKeys:  10000,
		entries:  make(map[string]*failureEntry),
		now:      time.Now,
	}
}

// Allow returns false and how long to wait if the key is currently locked out.
func (l *failureLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return true, 0
	}
	now := l.now()
	if now.Before(e.lockedUntil) {
		return false, e.lockedUntil.Sub(now)
	}
	return true, 0
}

// Fail records a failed attempt.
func (l *failureLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.maxKeys {
			l.evict(now)
		}
		e = &failureEntry{windowStart: now}
		l.entries[key] = e
	}
	if now.Sub(e.windowStart) > l.window && now.After(e.lockedUntil) {
		// quiet for a full window, forgive
		e.failures = 0
		e.strikes = 0
		e.windowStart = now
	}
	e.failures++
	if e.failures >= l.max {
		e.strikes++
		lock := l.baseLock << (e.strikes - 1)
		if lock > l.maxLock || lock <= 0 {
			lock = l.maxLock
		}
		e.lockedUntil = now.Add(lock)
		e.failures = 0
		e.windowStart = now
	}
}

// Reset forgets a key. Only use this for keys an attacker can't reset by
// logging in successfully with an account of their own (i.e. not IP keys).
func (l *failureLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// evict drops idle entries, and if that isn't enough, arbitrary ones so memory
// stays bounded. Caller must hold the lock.
func (l *failureLimiter) evict(now time.Time) {
	for k, e := range l.entries {
		if now.After(e.lockedUntil) && now.Sub(e.windowStart) > l.window {
			delete(l.entries, k)
		}
	}
	for k := range l.entries {
		if len(l.entries) < l.maxKeys {
			break
		}
		delete(l.entries, k)
	}
}

const (
	maxLoginBodyBytes = 64 << 10
	maxUserKeyLen     = 128
)

var (
	// A password guess (or TOTP guess) from one address. TOTP users incur one
	// counted failure per login because the first attempt is made without a
	// passcode ("passcode required"), so keep this comfortably above that.
	loginIPLimiter = newFailureLimiter(20, 15*time.Minute, time.Minute, time.Hour)
	// Guesses against one account from anywhere. Deliberately higher than the
	// IP limit and with a short maximum lock so an attacker can't keep a real
	// user locked out for long.
	loginUserLimiter = newFailureLimiter(30, 15*time.Minute, 30*time.Second, 15*time.Minute)

	warnNoClientIP sync.Once
)

// clientIP returns the address to throttle on. When Server.ClientIPHeader is
// set (for example "X-Real-IP", or "X-Forwarded-For" behind a proxy you
// control) the last value in that header is used since that is the one a
// trusted proxy appended. Otherwise the TCP peer is used. If the peer is a
// loopback address and no header is configured, we are almost certainly behind
// a reverse proxy and every client would share one key, so "" is returned and
// IP throttling is skipped.
func clientIP(ctx Context, r *http.Request) string {
	if h := ctx.Config().Server.ClientIPHeader; h != "" {
		v := r.Header.Get(h)
		if i := strings.LastIndex(v, ","); i >= 0 {
			v = v[i+1:]
		}
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		warnNoClientIP.Do(func() {
			log.Println("login requests arrive from loopback: set Server.ClientIPHeader to enable per-IP login throttling")
		})
		return ""
	}
	return host
}

// loginAttempt is a throttled login in progress; see beginLogin.
type loginAttempt struct {
	ipKey   string
	userKey string
}

// beginLogin checks the throttles for this request and user. If the request is
// locked out it writes a 429 and returns false.
func beginLogin(ctx Context, w http.ResponseWriter, r *http.Request, user string) (*loginAttempt, bool) {
	user = strings.ToLower(strings.TrimSpace(user))
	if len(user) > maxUserKeyLen {
		user = user[:maxUserKeyLen]
	}
	a := &loginAttempt{userKey: user}
	if ip := clientIP(ctx, r); ip != "" {
		a.ipKey = ip
	}

	var wait time.Duration
	blocked := false
	if a.ipKey != "" {
		if ok, d := loginIPLimiter.Allow(a.ipKey); !ok {
			blocked, wait = true, d
		}
	}
	if ok, d := loginUserLimiter.Allow(a.userKey); !ok {
		blocked = true
		if d > wait {
			wait = d
		}
	}
	if blocked {
		secs := int(wait.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		handleErr(w, "too many attempts", http.StatusTooManyRequests)
		return nil, false
	}
	return a, true
}

// finish records the result of the login. err == nil means a session was
// created. Anything else, including "passcode required", counts against the
// limits since it still tells the caller something about the credentials.
func (a *loginAttempt) finish(err error) {
	if err == nil {
		loginUserLimiter.Reset(a.userKey)
		return
	}
	if a.ipKey != "" {
		loginIPLimiter.Fail(a.ipKey)
	}
	loginUserLimiter.Fail(a.userKey)
}

// loginErr reports a failed login. Credential problems all return the same
// generic 401 so the response can't be used to find valid usernames or to tell
// a wrong username from a wrong password. Everything else is an opaque 500.
//
// Note "passcode required" is still returned as a server error as before: the
// client has to be told a passcode is needed, which unavoidably confirms the
// password. The throttle above is what limits guessing.
func loginErr(w http.ResponseWriter, err error) {
	if auth.CredentialsError(err) {
		authErr(w, ErrUnauthorized)
	} else {
		serverErr(w, err)
	}
}
