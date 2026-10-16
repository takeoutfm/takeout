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
	"net/http"
	"strings"

	"takeoutfm.dev/takeout/internal/auth"
	"takeoutfm.dev/takeout/lib/header"
)

type bits uint8

const (
	AllowCookie bits = 1 << iota
	AllowAccessToken
	AllowMediaToken

	BearerAuthorization = "Bearer"
)

// doCodeAuth creates a login session and binds to the provided code value.
func doCodeAuth(ctx Context, user, pass, passcode, value string) error {
	var err error
	var session auth.Session
	if passcode == "" {
		session, err = doLogin(ctx, user, pass)
	} else {
		session, err = doPasscodeLogin(ctx, user, pass, passcode)
	}
	if err != nil {
		return err
	}
	err = ctx.Auth().AuthorizeCode(value, session.Token)
	if err != nil {
		return ErrInvalidCode
	}
	return nil
}

// getAuthToken returns the bearer token from the request, if any.
func getAuthToken(r *http.Request) string {
	value := r.Header.Get(header.Authorization)
	if value == "" {
		return ""
	}
	result := strings.Split(value, " ")
	var token string
	switch len(result) {
	case 1:
		// Authorization: <token>
		token = result[0]
	case 2:
		// Authorization: Bearer <token>
		if strings.EqualFold(result[0], BearerAuthorization) {
			token = result[1]
		}
	}
	return token
}

// authorizeAccessToken validates the provided JWT access token for API access.
func authorizeAccessToken(ctx Context, w http.ResponseWriter, r *http.Request) (auth.User, error) {
	token := getAuthToken(r)
	if token == "" {
		return auth.User{}, ErrMissingAccessToken
	}
	// token should be a JWT
	user, err := ctx.Auth().CheckAccessTokenUser(token)
	if err != nil {
		return auth.User{}, err
	}
	return user, nil
}

// authorizeMediaToken validates the provided JWT media token for API access.
func authorizeMediaToken(ctx Context, w http.ResponseWriter, r *http.Request) (auth.User, error) {
	token := getAuthToken(r)
	if token == "" {
		return auth.User{}, ErrMissingMediaToken
	}
	// token should be a JWT
	user, err := ctx.Auth().CheckMediaTokenUser(token)
	if err != nil {
		return auth.User{}, err
	}
	return user, nil
}

// authorizeCodeToken validates the provided JWT code token for code auth access.
func authorizeCodeToken(ctx Context, w http.ResponseWriter, r *http.Request) error {
	token := getAuthToken(r)
	if token == "" {
		return ErrMissingCodeToken
	}
	// token should be a JWT with valid code in the subject
	err := ctx.Auth().CheckCodeToken(token)
	if err != nil {
		return err
	}

	return err
}

// authorizeFileToken validates the provided JWT media token for file access.
func authorizeFileToken(ctx Context, w http.ResponseWriter, r *http.Request, path string) error {
	token := r.URL.Query().Get(QueryToken)
	if token == "" {
		return ErrMissingToken
	}

	err := ctx.Auth().CheckFileToken(token, path)
	if err != nil {
		return err
	}
	return nil
}

// authorizeCookie validates the provided cookie for API or web view access.
func authorizeCookie(ctx Context, w http.ResponseWriter, r *http.Request) (auth.User, error) {
	a := ctx.Auth()
	cookie, err := r.Cookie(auth.CookieName)
	if err != nil {
		if err != http.ErrNoCookie {
			http.SetCookie(w, auth.ExpireCookie(cookie)) // what cookie is this?
		}
		return auth.User{}, ErrAccessDeniedRedirect
	}

	session, err := a.CookieSession(cookie)
	if err != nil {
		http.SetCookie(w, auth.ExpireCookie(cookie))
		return auth.User{}, ErrAccessDeniedRedirect
	} else if session.Expired() {
		a.DeleteSession(&session)
		http.SetCookie(w, auth.ExpireCookie(cookie))
		return auth.User{}, ErrAccessDeniedRedirect
	}

	user, err := a.SessionUser(session)
	if err != nil {
		// session with no user?
		a.DeleteSession(&session)
		http.SetCookie(w, auth.ExpireCookie(cookie))
		return auth.User{}, ErrAccessDeniedRedirect
	}

	// send back an updated cookie
	auth.UpdateCookie(session, cookie)
	http.SetCookie(w, cookie)

	return user, nil
}

// authorizeRefreshToken validates the provided refresh token for API access.
func authorizeRefreshToken(ctx Context, w http.ResponseWriter, r *http.Request) (auth.Session, error) {
	token := getAuthToken(r)
	if token == "" {
		return auth.Session{}, ErrUnauthorized
	}
	// token should be a refresh token not JWT
	a := ctx.Auth()
	session, err := a.TokenSession(token)
	if err != nil {
		// no session for token
		return auth.Session{}, err
	} else if session.Expired() {
		// session expired
		a.DeleteSession(&session)
		return auth.Session{}, ErrUnauthorized
	} else if session.Duration() < ctx.Config().Auth.AccessToken.Age {
		// session will expire before token
		return auth.Session{}, ErrUnauthorized
	}
	// session still valid
	return session, nil
}

// authorizeRequest authorizes the request with one or more of the allowed
// authorization methods.
func authorizeRequest(ctx Context, w http.ResponseWriter, r *http.Request, mask bits) (auth.User, error) {
	if mask&AllowAccessToken != 0 {
		user, err := authorizeAccessToken(ctx, w, r)
		if err == nil || err != ErrMissingAccessToken {
			return user, err
		}
	}

	if mask&AllowMediaToken != 0 {
		user, err := authorizeMediaToken(ctx, w, r)
		if err == nil || err != ErrMissingMediaToken {
			return user, err
		}
	}

	if mask&AllowCookie != 0 {
		user, err := authorizeCookie(ctx, w, r)
		if err == nil || err == ErrAccessDeniedRedirect {
			return user, err
		}
	}

	return auth.User{}, ErrUnauthorized
}

// refreshTokenAuthHandler handles requests intended to refresh and access token.
func refreshTokenAuthHandler(ctx RequestContext, handler http.HandlerFunc) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		session, err := authorizeRefreshToken(ctx, w, r)
		if err != nil {
			authErr(w, ErrUnauthorized)
		} else {
			ctx := sessionContext(ctx, session)
			handler.ServeHTTP(w, withContext(r, ctx))
		}
	}
	return http.HandlerFunc(fn)
}

// authHandler authorizes and handles all (except refresh) requests based on
// allowed auth methods.
func authHandler(ctx RequestContext, handler http.HandlerFunc, mask bits) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := authorizeRequest(ctx, w, r, mask)
		if err != nil {
			if err == ErrAccessDeniedRedirect {
				http.Redirect(w, r, LoginRedirect, http.StatusTemporaryRedirect)
			} else {
				authErr(w, err)
			}
			return
		}
		ctx, err := upgradeContext(ctx, user)
		if err != nil {
			serverErr(w, err)
		} else {
			handler.ServeHTTP(w, withContext(r, ctx))
		}
	}
	return http.HandlerFunc(fn)
}

// mediaTokenAuthHandler handles media access requests using the media token (or cookie).
func mediaTokenAuthHandler(ctx RequestContext, handler http.HandlerFunc) http.Handler {
	return authHandler(ctx, handler, AllowMediaToken|AllowCookie)
}

// accessTokenAuthHandler handles non-media requests using the access token (or cookie).
func accessTokenAuthHandler(ctx RequestContext, handler http.HandlerFunc) http.Handler {
	return authHandler(ctx, handler, AllowAccessToken|AllowCookie)
}

func codeTokenAuthHandler(ctx RequestContext, handler http.HandlerFunc) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		err := authorizeCodeToken(ctx, w, r)
		if err != nil {
			authErr(w, err)
		} else {
			handler.ServeHTTP(w, withContext(r, ctx))
		}
	}
	return http.HandlerFunc(fn)
}

func fileAuthHandler(ctx RequestContext, handler http.HandlerFunc, prefix string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, prefix)

		// check for bad paths
		if len(path) == 0 || strings.Contains(path, "..") {
			accessDenied(w)
			return
		}

		// ensure file is included and/or not excluded
		include := len(ctx.Config().Server.IncludeDirs) == 0
		for _, d := range ctx.Config().Server.IncludeDirs {
			if strings.HasPrefix(path, d) {
				include = true
				break
			}
		}
		exclude := false
		for _, d := range ctx.Config().Server.ExcludeDirs {
			if strings.HasPrefix(path, d) {
				exclude = true
				break
			}
		}

		if include && !exclude {
			err := authorizeFileToken(ctx, w, r, path)
			if err == nil {
				handler.ServeHTTP(w, withContext(r, ctx))
				return
			}
		}

		accessDenied(w)
	}
	return http.HandlerFunc(fn)
}
