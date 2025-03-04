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

package auth

import (
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"takeoutfm.dev/takeout/internal/config"
)

func GenerateTOTP(config config.TOTPConfig, userid string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.Issuer,
		AccountName: userid,
	})
	if err != nil {
		return "", err
	}
	return key.URL(), nil
}

// for unit testing
func GeneratePasscode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

func SecretFromURL(url string) (string, error) {
	key, err := otp.NewKeyFromURL(url)
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

func ValidatePasscode(passcode, secret string) bool {
	return totp.Validate(passcode, secret)
}
