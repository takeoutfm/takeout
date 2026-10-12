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
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	CodeChars = "123456789ABCDEFGHILKMNPQRSTUVWXYZ"
	CodeSize  = 6
)

type Code struct {
	gorm.Model
	Value   string    `gorm:"uniqueIndex:idx_codes_value"`
	Expires time.Time `gorm:"index:idx_codes_expires"`
	Token   string
}

func randomCode() string {
	var code strings.Builder
	max := big.NewInt(int64(len(CodeChars)))
	for range CodeSize {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand failing is not recoverable
			panic(err)
		}
		code.WriteByte(CodeChars[n.Int64()])
	}
	return code.String()
}

func (a *Auth) createCode(c Code) (err error) {
	err = a.db.Create(&c).Error
	return
}

func (a *Auth) findCode(value string) *Code {
	var code Code
	err := a.db.Where("value = ?", value).First(&code).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return &code
}

func (a *Auth) deleteCode(c *Code) {
	a.db.Delete(c)
}

func (a *Auth) DeleteExpiredCodes() error {
	now := time.Now()
	return a.db.Unscoped().Where("expires < ?", now).Delete(Code{}).Error
}

// maxCodesPerMinute bounds how many link codes can be created across all
// callers (API clients, the assistant webhook, ...). Codes live for CodeAge
// (5m by default), so this caps the number of live rows at a few hundred no
// matter how many clients, or which addresses, are asking.
const maxCodesPerMinute = 120

// maxCodeAttempts is how many random values to try if inserting a code fails,
// normally because the value collides with an existing row.
const maxCodeAttempts = 5

// windowCounter is a fixed window rate limit.
type windowCounter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	start  time.Time
	count  int
}

func (c *windowCounter) allow(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if now.Sub(c.start) >= c.window {
		c.start = now
		c.count = 0
	}
	if c.count >= c.limit {
		return false
	}
	c.count++
	return true
}

var codeRate = &windowCounter{limit: maxCodesPerMinute, window: time.Minute}

// GenerateCode creates a new link code. It returns ErrTooManyCodes when codes
// are being requested faster than the global limit.
func (a *Auth) GenerateCode() (*Code, error) {
	now := time.Now()
	if !codeRate.allow(now) {
		return nil, ErrTooManyCodes
	}
	expires := now.Add(a.codeAge())
	var err error
	for i := 0; i < maxCodeAttempts; i++ {
		c := &Code{Value: randomCode(), Expires: expires}
		// a collision with an existing (possibly expired but not yet
		// cleaned up) value violates the unique index, so try another
		if err = a.db.Create(c).Error; err == nil {
			return c, nil
		}
	}
	return nil, err
}

func (c *Code) expired() bool {
	now := time.Now()
	return now.After(c.Expires)
}

func (c *Code) Linked() bool {
	return c.Token != ""
}

func (a *Auth) LookupCode(value string) *Code {
	return a.findCode(value)
}

func (a *Auth) ValidCode(value string) *Code {
	code := a.findCode(value)
	if code == nil || code.expired() {
		return nil
	}
	return code
}

func (a *Auth) LinkedCode(value string) *Code {
	code := a.findCode(value)
	if code == nil || code.Token == "" || code.expired() {
		return nil
	}
	return code
}

// This assumes token is valid
func (a *Auth) AuthorizeCode(value, token string) error {
	code := a.findCode(value)
	if code == nil {
		return ErrCodeNotFound
	}
	if code.expired() {
		return ErrCodeExpired
	}
	if code.Token != "" {
		return ErrCodeAlreadyUsed
	}
	return a.db.Model(code).Update("token", token).Error
}

func (a *Auth) codeAge() time.Duration {
	return a.config.Auth.CodeAge
}
