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
	"net/url"
	"strings"
	"testing"
)

const testPassword = "correct horse battery staple 9"

func postForm(ctx Context, handler http.HandlerFunc, path string, form url.Values,
	cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	handler(w, withContext(req, ctx))
	return w
}

func expectRedirect(t *testing.T, w *httptest.ResponseRecorder, location string) {
	t.Helper()
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != location {
		t.Fatalf("got %d to %q, want %q", w.Code, w.Header().Get("Location"), location)
	}
}

func addTestUser(t *testing.T, ctx *TestContext, name string) {
	t.Helper()
	if err := ctx.Auth().AddUser(name, testPassword); err != nil {
		t.Fatal(err)
	}
}

func TestLoginFailed(t *testing.T) {
	ctx := NewTestContext(t)
	addTestUser(t, ctx, "login-failed")
	w := postForm(ctx, loginHandler, "/login",
		url.Values{FormUser: {"login-failed"}, FormPass: {"wrong password"}}, nil)
	expectRedirect(t, w, LoginRedirect+"?failed")
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("cookie set for a failed sign-in")
	}

	w = postForm(ctx, loginHandler, "/login",
		url.Values{FormUser: {"login-failed"}, FormPass: {testPassword}}, nil)
	expectRedirect(t, w, SuccessRedirect)
}

func newCode(t *testing.T, ctx *TestContext) string {
	t.Helper()
	code, err := ctx.Auth().GenerateCode()
	if err != nil {
		t.Fatal(err)
	}
	return code.Value
}

func TestLinkWithPassword(t *testing.T) {
	ctx := NewTestContext(t)
	addTestUser(t, ctx, "link-password")
	code := newCode(t, ctx)

	// no credentials, not signed in
	w := postForm(ctx, linkHandler, "/link", url.Values{FormCode: {code}}, nil)
	expectRedirect(t, w, LinkRedirect+"?failed")
	if ctx.Auth().LookupCode(code).Linked() {
		t.Fatal("linked without signing in")
	}

	// lower case, as a phone might type it
	w = postForm(ctx, linkHandler, "/link", url.Values{
		FormCode: {strings.ToLower(code)}, FormUser: {"link-password"}, FormPass: {testPassword},
	}, nil)
	expectRedirect(t, w, LinkRedirect+"?linked")
	if !ctx.Auth().LookupCode(code).Linked() {
		t.Fatal("code not linked")
	}

	// a code works once
	w = postForm(ctx, linkHandler, "/link", url.Values{
		FormCode: {code}, FormUser: {"link-password"}, FormPass: {testPassword},
	}, nil)
	expectRedirect(t, w, LinkRedirect+"?failed")
}

func TestLinkSignedIn(t *testing.T) {
	ctx := NewTestContext(t)
	a := ctx.Auth()
	addTestUser(t, ctx, "link-cookie")
	session, err := a.LoginSession("link-cookie")
	if err != nil {
		t.Fatal(err)
	}
	cookie := a.NewCookie(&session)
	code := newCode(t, ctx)

	w := postForm(ctx, linkHandler, "/link", url.Values{FormCode: {code}}, &cookie)
	expectRedirect(t, w, LinkRedirect+"?linked")

	linked := a.LookupCode(code)
	if !linked.Linked() {
		t.Fatal("code not linked")
	}
	if linked.Token == session.Token {
		t.Fatal("device shares the browser's session")
	}
	device, err := a.TokenSession(linked.Token)
	if err != nil {
		t.Fatal(err)
	}
	if user, err := a.SessionUser(device); err != nil || user.Name != "link-cookie" {
		t.Fatalf("device session for %q: %v", user.Name, err)
	}

	// an unknown code
	w = postForm(ctx, linkHandler, "/link", url.Values{FormCode: {"ZZZZZZ"}}, &cookie)
	expectRedirect(t, w, LinkRedirect+"?failed")
}
