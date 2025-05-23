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

// Package auth manages users, passwords, tokens and sessions.
package auth

import (
	"bytes"
	"crypto/rand"
	"errors"
	"net/http"
	"time"

	"github.com/gokyle/filecache"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/scrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/log"
)

const (
	CookieName = takeout.AppName
)

var (
	ErrBadDriver                = errors.New("driver not supported")
	ErrUserNotFound             = errors.New("user not found")
	ErrKeyMismatch              = errors.New("key mismatch")
	ErrSessionNotFound          = errors.New("session not found")
	ErrSessionExpired           = errors.New("session expired")
	ErrCodeNotFound             = errors.New("code not found")
	ErrCodeExpired              = errors.New("code has expired")
	ErrCodeAlreadyUsed          = errors.New("code already authorized")
	ErrInvalidTokenSubject      = errors.New("invalid subject")
	ErrInvalidTokenAudience     = errors.New("invalid audience")
	ErrInvalidTokenMethod       = errors.New("invalid token method")
	ErrInvalidTokenIssuer       = errors.New("invalid token issuer")
	ErrInvalidTokenClaims       = errors.New("invalid token claims")
	ErrInvalidAccessTokenSecret = errors.New("invalid access token secret")
	ErrInvalidMediaTokenSecret  = errors.New("invalid media token secret")
	ErrInvalidCodeTokenSecret   = errors.New("invalid code token secret")
	ErrInvalidFileTokenSecret   = errors.New("invalid file token secret")
	ErrInvalidTokenSecret       = errors.New("invalid token secret")
	ErrTokenExpired             = errors.New("token expired")
	ErrMissingTOTP              = errors.New("missing totp")
	ErrInvalidPasscodeIssuer    = errors.New("invalid passcode issuer")
	ErrInvalidPasscode          = errors.New("invalid passcode")
	ErrPasscodeRequired         = errors.New("passcode required")
	ErrLoginFailed              = errors.New("login failed")
)

type User struct {
	gorm.Model
	Name  string `gorm:"uniqueIndex:idx_user_name"`
	Key   []byte
	Salt  []byte
	Media string
	TOTP  string
}

// A Session is an authenticated user login session associated with a token and
// expiration date.
type Session struct {
	gorm.Model
	User    string    `gorm:"uniqueIndex:idx_session;index:idx_session_user"`
	Token   string    `gorm:"uniqueIndex:idx_session;index:idx_session_token"`
	Expires time.Time `gorm:"index:idx_session_expires"`
}

var noUser = User{}
var noSession = Session{}

// Expired returns whether or not the session is expired.
func (s *Session) Expired() bool {
	now := time.Now()
	return now.After(s.Expires)
}

// Valid returns whether or not the session is not expired.
func (s *Session) Valid() bool {
	return !s.Expired()
}

type Auth struct {
	config    *config.Config
	db        *gorm.DB
	fileCache *filecache.FileCache
}

func NewAuth(config *config.Config) *Auth {
	auth := &Auth{config: config, fileCache: filecache.NewDefaultCache()}

	// ensure secrets exist before going any further
	_, err := auth.readSecret(config.Auth.AccessToken)
	if err != nil {
		log.Panicln(ErrInvalidAccessTokenSecret)
	}
	_, err = auth.readSecret(config.Auth.MediaToken)
	if err != nil {
		log.Panicln(ErrInvalidMediaTokenSecret)
	}
	_, err = auth.readSecret(config.Auth.CodeToken)
	if err != nil {
		log.Panicln(ErrInvalidCodeTokenSecret)
	}
	_, err = auth.readSecret(config.Auth.FileToken)
	if err != nil {
		log.Panicln(ErrInvalidCodeTokenSecret)
	}

	err = auth.fileCache.Start()
	if err != nil {
		log.Panicln(err)
	}

	return auth
}

func (a *Auth) Open() (err error) {
	cfg := a.config.Music.DB.GormConfig()

	if a.config.Auth.DB.Driver == "sqlite3" {
		a.db, err = gorm.Open(sqlite.Open(a.config.Auth.DB.Source), cfg)
	} else {
		err = ErrBadDriver
	}

	if err != nil {
		return
	}

	err = a.db.AutoMigrate(&Code{}, &Session{}, &User{})
	return
}

func (a *Auth) Close() {
	conn, err := a.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

// AddUser adds a new user to the user database.
func (a *Auth) AddUser(userid, pass string) error {
	err := a.validatePassword(pass)
	if err != nil {
		return err
	}

	salt := make([]byte, 8)
	_, err = rand.Read(salt)
	if err != nil {
		return err
	}

	key, err := a.key(pass, salt)
	if err != nil {
		return err
	}

	u := User{Name: userid, Key: key, Salt: salt}

	return a.createUser(&u)
}

func (a *Auth) validatePassword(password string) error {
	return passwordvalidator.Validate(password, float64(a.config.Auth.PasswordEntropy))
}

// User returns the user found with the provded userid.
func (a *Auth) User(userid string) (User, error) {
	var u User
	err := a.db.Where("name = ?", userid).First(&u).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

// Check will check if the provided userid and password match a user in the
// database.
func (a *Auth) check(userid, pass string) (User, error) {
	u, err := a.User(userid)
	if err != nil {
		return u, ErrUserNotFound
	}

	key, err := a.key(pass, u.Salt)
	if err != nil {
		return User{}, err
	}

	if !bytes.Equal(u.Key, key) {
		return User{}, ErrKeyMismatch
	}

	return u, nil
}

func (a *Auth) checkPasscode(u User, passcode string) (bool, error) {
	if u.TOTP == "" {
		return false, ErrMissingTOTP
	}

	secret, err := SecretFromURL(u.TOTP)
	if err != nil {
		return false, err
	}

	return ValidatePasscode(passcode, secret), nil
}

func CredentialsError(err error) bool {
	switch err {
	case ErrUserNotFound, ErrKeyMismatch, ErrMissingTOTP:
		return true
	default:
		return false
	}
}

// Login will create a new login session after authenticating the userid and
// password.
func (a *Auth) Login(userid, pass string) (Session, error) {
	u, err := a.loginCheck(userid, pass)
	if err != nil {
		return noSession, err
	}
	session := a.session(u)
	err = a.createSession(&session)
	if err != nil {
		return noSession, err
	}
	return session, err
}

// LoginSession will create a new login session for the given userid. No
// password or passcode are required so use with caution.
func (a *Auth) LoginSession(userid string) (Session, error) {
	u, err := a.User(userid)
	if err != nil {
		return noSession, ErrUserNotFound
	}
	session := a.session(u)
	err = a.createSession(&session)
	if err != nil {
		return noSession, err
	}
	return session, err
}

func (a *Auth) loginCheck(userid, pass string) (User, error) {
	u, err := a.check(userid, pass)
	if err != nil {
		return noUser, err
	}

	if u.TOTP != "" {
		// user has TOTP, require a passcode login
		return noUser, ErrPasscodeRequired
	}
	return u, nil
}

func (a *Auth) PasscodeLogin(userid, pass, passcode string) (Session, error) {
	u, err := a.passcodeLoginCheck(userid, pass, passcode)
	if err != nil {
		return noSession, err
	}
	session := a.session(u)
	err = a.createSession(&session)
	if err != nil {
		return noSession, err
	}
	return session, err
}

func (a *Auth) passcodeLoginCheck(userid, pass, passcode string) (User, error) {
	u, err := a.check(userid, pass)
	if err != nil {
		return noUser, err
	}
	valid, err := a.checkPasscode(u, passcode)
	if err != nil {
		return noUser, err
	}
	if valid == false {
		return noUser, ErrInvalidPasscode
	}
	return u, err
}

// ChangePass changes the password associated with the provided userid.  User
// Check prior to this if you'd like to verify the current password.
//
// TODO this should trigger a TOTP change as well.
func (a *Auth) ChangePass(userid, newpass string) error {
	err := a.validatePassword(newpass)
	if err != nil {
		return err
	}

	u, err := a.User(userid)
	if err != nil {
		return ErrUserNotFound
	}

	salt := make([]byte, 8)
	_, err = rand.Read(salt)
	if err != nil {
		return err
	}

	key, err := a.key(newpass, salt)
	if err != nil {
		return err
	}

	u.Salt = salt
	u.Key = key

	return a.db.Model(u).Update("salt", u.Salt).Update("key", u.Key).Error
}

// assign a TOTP to a user
//
// The TOTP secret is not stored encrypted. May change this later but would
// need a way to protect passwords used to encrypt secrets.
//
// Entire otpauth URL is stored to support future use of different parameters.
func (a *Auth) AssignTOTP(userid, url string) error {
	u, err := a.User(userid)
	if err != nil {
		return ErrUserNotFound
	}

	u.TOTP = url

	return a.db.Model(u).Update("totp", u.TOTP).Error
}

// readSecret returns secret from configured string or file
func (a *Auth) readSecret(cfg config.TokenConfig) ([]byte, error) {
	if cfg.Secret != "" {
		return []byte(cfg.Secret), nil
	}

	data, err := a.fileCache.ReadFile(cfg.SecretFile)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrInvalidTokenSecret
	}
	return data, nil
}

// newToken creates a new JWT token
func (a *Auth) newToken(subject string, cfg config.TokenConfig) (string, error) {
	age := int(cfg.Age.Seconds())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.StandardClaims{
			Issuer:    cfg.Issuer,
			Subject:   subject,
			ExpiresAt: time.Now().Add(time.Second * time.Duration(age)).Unix(),
		})
	secret, err := a.readSecret(cfg)
	if err != nil {
		return "", err
	}
	return token.SignedString(secret)
}

// newSessionToken creates a new JWT token associated with the provided session.
func (a *Auth) newSessionToken(s Session, cfg config.TokenConfig) (string, error) {
	return a.newToken(s.User, cfg)
}

// newFileToken creates a new JWT token for a file path or uri
func (a *Auth) newFileToken(file string, cfg config.TokenConfig) (string, error) {
	age := int(cfg.Age.Seconds())
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.StandardClaims{
			Issuer:    cfg.Issuer,
			Audience:  file,
			ExpiresAt: time.Now().Add(time.Second * time.Duration(age)).Unix(),
		})
	secret, err := a.readSecret(cfg)
	if err != nil {
		return "", err
	}
	return token.SignedString(secret)
}

// NewAccessToken creates a new JWT token associated with the provided session.
func (a *Auth) NewAccessToken(s Session) (string, error) {
	return a.newSessionToken(s, a.config.Auth.AccessToken)
}

// NewMediaToken creates a new JWT token associated with the provided session.
func (a *Auth) NewMediaToken(s Session) (string, error) {
	return a.newSessionToken(s, a.config.Auth.MediaToken)
}

// NewCodeToken creates a new JWT token for code-based authentication
func (a *Auth) NewCodeToken(subject string) (string, error) {
	return a.newToken(subject, a.config.Auth.CodeToken)
}

// NewFileToken creates a new JWT token for file auth
func (a *Auth) NewFileToken(path string) (string, error) {
	return a.newFileToken(path, a.config.Auth.FileToken)
}

// NewCookie creates a new cookie associated with the provided session.
func (a *Auth) NewCookie(session *Session) http.Cookie {
	return http.Cookie{
		Name:     CookieName,
		Value:    session.Token,
		MaxAge:   session.timeRemaining(),
		Path:     "/",
		Secure:   a.config.Auth.SecureCookies,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true}
}

// ExpireCookie will update cookie fields to ensure it's expired.
func ExpireCookie(cookie *http.Cookie) *http.Cookie {
	cookie.MaxAge = 0
	cookie.Expires = time.Now().Add(-24 * time.Hour)
	return cookie
}

// CookieSession will find the session associated with the provided cookie.
func (a *Auth) CookieSession(cookie *http.Cookie) (Session, error) {
	if cookie == nil || cookie.Name != CookieName {
		return noSession, ErrSessionNotFound
	}
	return a.findCookieSession(cookie)
}

// TokenSession will find the session associated with this provided token.
func (a *Auth) TokenSession(token string) (Session, error) {
	return a.findSession(token)
}

func (a *Auth) CheckCookie(cookie *http.Cookie) error {
	session, err := a.CookieSession(cookie)
	if err != nil {
		return err
	}
	if session.Expired() {
		return ErrSessionExpired
	}
	return nil
}

func (a *Auth) CheckAccessToken(signedToken string) error {
	_, _, err := a.processToken(signedToken, a.config.Auth.AccessToken)
	return err
}

func (a *Auth) CheckAccessTokenUser(signedToken string) (User, error) {
	_, claims, err := a.processToken(signedToken, a.config.Auth.AccessToken)
	if err != nil {
		return User{}, err
	}
	return a.User(claims.Subject)
}

func (a *Auth) CheckMediaToken(signedToken string) error {
	_, _, err := a.processToken(signedToken, a.config.Auth.MediaToken)
	return err
}

func (a *Auth) CheckMediaTokenUser(signedToken string) (User, error) {
	_, claims, err := a.processToken(signedToken, a.config.Auth.MediaToken)
	if err != nil {
		return User{}, err
	}
	return a.User(claims.Subject)
}

func (a *Auth) CheckCodeToken(signedToken string) error {
	_, claims, err := a.processToken(signedToken, a.config.Auth.CodeToken)
	if err != nil {
		return err
	}
	code := a.ValidCode(claims.Subject)
	if code == nil {
		return ErrInvalidTokenSubject
	}
	return nil
}

func (a *Auth) CheckFileToken(signedToken string, path string) error {
	_, claims, err := a.processToken(signedToken, a.config.Auth.FileToken)
	if err != nil {
		return err
	}
	if claims.Audience != path {
		return ErrInvalidTokenAudience
	}
	return err
}

// processToken parses and verfies the signed token is valid.
func (a *Auth) processToken(signedToken string, cfg config.TokenConfig) (*jwt.Token, *jwt.StandardClaims, error) {
	token, err := jwt.ParseWithClaims(
		signedToken,
		&jwt.StandardClaims{},
		func(token *jwt.Token) (interface{}, error) {
			secret, err := a.readSecret(cfg)
			return secret, err
		})
	if err != nil {
		return nil, nil, err
	}
	if token.Method != jwt.SigningMethodHS256 {
		return nil, nil, ErrInvalidTokenMethod
	}
	claims, ok := token.Claims.(*jwt.StandardClaims)
	if !ok {
		return nil, nil, ErrInvalidTokenClaims
	}
	if claims.Issuer != cfg.Issuer {
		return nil, nil, ErrInvalidTokenIssuer
	}
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, nil, ErrTokenExpired
	}
	if claims.Subject == "" && claims.Audience == "" {
		// need subject or audience
		return nil, nil, ErrInvalidTokenClaims
	}
	return token, claims, nil
}

// UpdateCookie will update the cookie age based on the time left for the session.
func UpdateCookie(session Session, cookie *http.Cookie) {
	cookie.MaxAge = session.timeRemaining()
}

// RefreshCookie will renew a session and cookie.
func (a *Auth) RefreshCookie(session *Session, cookie *http.Cookie) error {
	err := a.Refresh(session)
	if err != nil {
		return err
	}
	cookie.MaxAge = session.timeRemaining()
	return nil
}

// DeleteSession will delete the provided session
func (a *Auth) DeleteSession(session *Session) {
	a.db.Delete(session)
}

func (a *Auth) DeleteSessions(u User) error {
	return a.db.Delete(Session{}, "name = ?", u.Name).Error
}

func (a *Auth) DeleteExpiredSessions() error {
	now := time.Now()
	return a.db.Unscoped().Where("expires < ?", now).Delete(Session{}).Error
}

// Expire all user sessions. This will expire all cookies and refresh tokens.
// All other tokens will be valid until their ExpireAt.
func (a *Auth) ExpireAll(userid string) error {
	user, err := a.User(userid)
	if err != nil {
		return err
	}
	now := time.Now()
	return a.db.Model(Session{}).Where("user = ?", user.Name).Updates(Session{Expires: now}).Error
}

func (a *Auth) SessionUser(session Session) (User, error) {
	u, err := a.User(session.User)
	if err != nil {
		return u, ErrUserNotFound
	}
	return u, nil
}

func (a *Auth) Refresh(session *Session) error {
	if session == nil {
		return ErrSessionNotFound
	}
	return a.touch(session)
}

func (a *Auth) key(pass string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(pass), salt, 32768, 8, 1, 32)
}

func (a *Auth) findCookieSession(cookie *http.Cookie) (Session, error) {
	return a.findSession(cookie.Value)
}

func (a *Auth) findSession(token string) (Session, error) {
	var session Session
	err := a.db.Where("token = ?", token).First(&session).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return noSession, ErrSessionNotFound
	}
	return session, nil
}

func (a *Auth) session(u User) Session {
	token := uuid.New().String()
	expires := time.Now().Add(a.config.Auth.SessionAge)
	session := Session{User: u.Name, Token: token, Expires: expires}
	return session
}

func (a *Auth) touch(s *Session) error {
	expires := time.Now().Add(a.config.Auth.SessionAge)
	return a.db.Model(s).Update("expires", expires).Error
}

func (a *Auth) createUser(u *User) (err error) {
	err = a.db.Create(u).Error
	return
}

func (a *Auth) updateUser(u *User) (err error) {
	err = a.db.Save(u).Error
	return
}

func (a *Auth) createSession(s *Session) (err error) {
	err = a.db.Create(s).Error
	return
}

// timeRemaing returns the number of number of seconds remaining in this session.
func (s *Session) timeRemaining() int {
	return int(s.Duration().Seconds())
}

// Duration returns the remain time for this session.
func (s *Session) Duration() time.Duration {
	return s.Expires.Sub(time.Now())
}
