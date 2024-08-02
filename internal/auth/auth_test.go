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
	"testing"

	"github.com/takeoutfm/takeout/internal/config"
)

func makeAuth(t *testing.T) *Auth {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	a := NewAuth(config)
	err = a.Open()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAddUser(t *testing.T) {
	user := "defsub@defsub.com"
	pass := "test_Pa$$/1234,;&w0rd"

	a := makeAuth(t)
	err := a.AddUser(user, pass)
	if err != nil {
		t.Fatal(err)
	}

	session, err := a.Login(user, pass)
	if err != nil {
		t.Fatal(err)
	}
	if (session.User != user) {
		t.Error("expect user")
	}
	if len(session.Token) == 0 {
		t.Error("expect cookie")
	}
}

func TestChangePass(t *testing.T) {
	user := "defsub@defsub.com"
	pass1 := "test_Pa$$/1234,;&w0rd"
	pass2 := "other&pass;test@_1234"

	a := makeAuth(t)
	err := a.AddUser(user, pass1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = a.Login(user, pass1)
	if err != nil {
		t.Fatal(err)
	}

	err = a.ChangePass(user, pass2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = a.Login(user, pass1)
	if err == nil {
		t.Fatal("expect login err")
	}

	_, err = a.Login(user, pass2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewAccessToken(t *testing.T) {
	user := "defsub@defsub.com"
	pass := "other&pass;test@_1234"

	a := makeAuth(t)
	session, err := a.Login(user, pass)
	if err != nil {
		t.Fatal(err)
	}

	token, err := a.NewAccessToken(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Error("expect token")
	}

	err = a.CheckAccessToken(token)
	if err != nil {
		t.Fatal("expect good token")
	}

	u, err := a.CheckAccessTokenUser(token)
	if err != nil {
		t.Fatal("expect good token")
	}
	if user != u.Name {
		t.Error("expect same user")
	}

}

func TestNewMediaToken(t *testing.T) {
	user := "defsub@defsub.com"
	pass := "other&pass;test@_1234"

	a := makeAuth(t)
	session, err := a.Login(user, pass)
	if err != nil {
		t.Fatal(err)
	}

	token, err := a.NewMediaToken(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Error("expect token")
	}

	err = a.CheckMediaToken(token)
	if err != nil {
		t.Fatal("expect good token")
	}

	u, err := a.CheckMediaTokenUser(token)
	if err != nil {
		t.Fatal("expect good token")
	}
	if user != u.Name {
		t.Error("expect same user")
	}

}

func TestNewCodeToken(t *testing.T) {
	user := "defsub@defsub.com"
	pass := "other&pass;test@_1234"

	a := makeAuth(t)

	// get a code
	code := a.GenerateCode()
	if code == nil {
		t.Fatal("expect code")
	}
	if len(code.Value) == 0 {
		t.Fatal("expect code value")
	}

	// get a jwt token to check code
	checkToken, err := a.NewCodeToken(code.Value)
	if err != nil {
		t.Fatal(err)
	}
	if len(checkToken) == 0 {
		t.Error("expect code token")
	}

	// check the token is valid
	err = a.CheckCodeToken(checkToken)
	if err != nil {
		t.Fatal(err)
	}
	// check the code isn't linked yet
	c := a.LookupCode(code.Value)
	if c == nil {
		t.Error("expect lookup code")
	}
	if c.Linked() != false {
		t.Error("expect unlinked")
	}

	// create a session to link code with user
	session, err := a.Login(user, pass)
	if err != nil {
		t.Fatal(err)
	}

	// link the code to the session
	err = a.AuthorizeCode(code.Value, session.Token)
	if err != nil {
		t.Fatal(err)
	}

	// check the code *is* linked now
	c = a.LookupCode(code.Value)
	if c == nil {
		t.Error("expect lookup code")
	}
	if c.Linked() != true {
		t.Fatal("expect linked")
	}

	// find the session for this token
	_, err = a.TokenSession(c.Token)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewFileToken(t *testing.T) {
	path := "/path/to file/file.mp3"

	a := makeAuth(t)

	token, err := a.NewFileToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) == 0 {
		t.Error("expect token")
	}

	err = a.CheckFileToken(token, path)
	if err != nil {
		t.Fatal("expect good token")
	}
}

func TestExpireAll(t *testing.T) {
	user := "defsub@defsub.com"
	pass := "other&pass;test@_1234"

	a := makeAuth(t)
	session, err := a.Login(user, pass)
	if err != nil {
		t.Fatal(err)
	}

	if session.Valid() == false {
		t.Fatal("session should be valid")
	}

	err = a.ExpireAll(user)
	if err != nil {
		t.Fatal(err)
	}

	s, err := a.findSession(session.Token)
	if err != nil {
		t.Fatal(err)
	}
	if s.Valid() == true {
		t.Fatal("expire not valid")
	}
}

func TestPasscode(t *testing.T) {
	user := "defotp"
	pass := "test_Pa$$/1234,;&w0rd"
	url := "otpauth://totp/takeout.fm:defotp?algorithm=SHA1&digits=6&issuer=takeout.fm&period=30&secret=XNTZPZRUIKTRKRDUAPW3AWHEOY7AXKOO"

	a := makeAuth(t)
	err := a.AddUser(user, pass)
	if err != nil {
		t.Fatal(err)
	}

	err = a.AssignTOTP(user, url)
	if err != nil {
		t.Fatal(err)
	}

	_, err = a.Login(user, pass)
	if err == nil {
		t.Fatal("expected to fail, missing totp")
	}

	secret, err := SecretFromURL(url)
	if err != nil {
		t.Fatal(err)
	}

	passcode, err := GeneratePasscode(secret)
	if err != nil {
		t.Fatal(err)
	}

	session, err := a.PasscodeLogin(user, pass, passcode)
	if err != nil {
		t.Fatal("expected session")
	}

	if (session.User != user) {
		t.Error("expect user")
	}
	if len(session.Token) == 0 {
		t.Error("expect cookie")
	}
}
