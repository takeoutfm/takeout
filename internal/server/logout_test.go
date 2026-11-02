// Copyright 2026 defsub
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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/auth"
)

func logout(ctx Context, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/logout", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	logoutHandler(w, withContext(req, ctx))
	return w
}

func TestLogout(t *testing.T) {
	ctx := NewTestContext(t)
	a := ctx.Auth()
	const user = "logout-test"
	if err := a.AddUser(user, "correct horse battery staple 9"); err != nil {
		t.Fatal(err)
	}
	session, err := a.LoginSession(user)
	if err != nil {
		t.Fatal(err)
	}
	other, err := a.LoginSession(user) // another device, stays signed in
	if err != nil {
		t.Fatal(err)
	}
	cookie := a.NewCookie(&session)
	if err := a.CheckCookie(&cookie); err != nil {
		t.Fatalf("cookie before logout: %s", err)
	}

	w := logout(ctx, &cookie)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != LoginRedirect {
		t.Fatalf("got %d to %q", w.Code, w.Header().Get("Location"))
	}
	if err := a.CheckCookie(&cookie); err == nil {
		t.Fatal("session still valid after logout")
	}
	if _, err := a.TokenSession(other.Token); err != nil {
		t.Fatalf("other session ended too: %s", err)
	}

	cleared := w.Result().Cookies()
	if len(cleared) != 1 || !cleared[0].Expires.Before(time.Now()) {
		t.Fatalf("cookie not expired: %+v", cleared)
	}
	checkCookieAttrs(t, cleared[0])
}

func TestLogoutWithoutCookie(t *testing.T) {
	w := logout(NewTestContext(t), nil)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != LoginRedirect {
		t.Fatalf("got %d to %q", w.Code, w.Header().Get("Location"))
	}
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("no cookie to clear")
	}
}

// the cookie a request carries has only a name and value; what the server
// sends back must be the full cookie it set
func checkCookieAttrs(t *testing.T, c *http.Cookie) {
	t.Helper()
	if c.Name != auth.CookieName || c.Path != "/" || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie attributes: %+v", c)
	}
}

func authorize(ctx Context, cookie *http.Cookie) (*httptest.ResponseRecorder, error) {
	req := httptest.NewRequest("GET", "/api/index", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	_, err := authorizeCookie(ctx, w, req)
	return w, err
}

func TestAuthorizeCookieRefresh(t *testing.T) {
	ctx := NewTestContext(t)
	a := ctx.Auth()
	const user = "cookie-test"
	if err := a.AddUser(user, "correct horse battery staple 9"); err != nil {
		t.Fatal(err)
	}
	session, err := a.LoginSession(user)
	if err != nil {
		t.Fatal(err)
	}
	w, err := authorize(ctx, &http.Cookie{Name: auth.CookieName, Value: session.Token})
	if err != nil {
		t.Fatal(err)
	}
	sent := w.Result().Cookies()
	if len(sent) != 1 || sent[0].Value != session.Token || sent[0].MaxAge <= 0 {
		t.Fatalf("refreshed cookie: %+v", sent)
	}
	checkCookieAttrs(t, sent[0])
}

func TestAuthorizeCookieUnknown(t *testing.T) {
	ctx := NewTestContext(t)
	w, err := authorize(ctx, &http.Cookie{Name: auth.CookieName, Value: "no-such-session"})
	if err != ErrAccessDeniedRedirect {
		t.Fatalf("got %v", err)
	}
	sent := w.Result().Cookies()
	if len(sent) != 1 || !sent[0].Expires.Before(time.Now()) {
		t.Fatalf("cookie not cleared: %+v", sent)
	}
	checkCookieAttrs(t, sent[0])
}
