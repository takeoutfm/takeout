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

// Package playout is a command line music/podcast/radio player with a text and
// visual view. There's support for ListenBrainz and ASCII art covers. See the
// player package for supported platforms, media and formats.
package playout

import (
	"net/http"

	"github.com/spf13/viper"

	"github.com/takeoutfm/takeout"
)

const (
	UserAgent = "Playout/" + takeout.Version + " (" + takeout.Contact + ")"

	TokenAccess       = "accesstoken"
	TokenRefresh      = "refreshtoken"
	TokenMedia        = "mediatoken"
	TokenCode         = "codetoken"
	TokenListenBrainz = "lbztoken"

	EnableListenBrainz  = "enableListenBrainz"
	EnableTrackActivity = "enableTrackActivity"

	Code     = "code"
	Endpoint = "endpoint"
)

type Playout struct {
	config *viper.Viper
	tokens *viper.Viper
}

func NewPlayout(config *viper.Viper, tokens *viper.Viper) *Playout {
	return &Playout{config: config, tokens: tokens}
}

func (p *Playout) UpdateAccessCode(code, access string) error {
	p.tokens.Set(Code, code)
	p.tokens.Set(TokenCode, access)
	return p.writeTokens()
}

func (p *Playout) UpdateTokens(access, refresh, media string) error {
	p.tokens.Set(TokenAccess, access)
	p.tokens.Set(TokenRefresh, refresh)
	p.tokens.Set(TokenMedia, media)
	return p.writeTokens()
}

func (p *Playout) UpdateAccessToken(value string) {
	p.tokens.Set(TokenAccess, value)
	err := p.writeTokens()
	if err != nil {
		panic(err)
	}
}

func (p *Playout) UpdateListenBrainzToken(value string) error {
	p.tokens.Set(TokenListenBrainz, value)
	return p.writeTokens()
}

func (p *Playout) writeTokens() error {
	err := p.tokens.WriteConfig()
	if err != nil {
		err = p.tokens.SafeWriteConfig()
	}
	return err
}

func (p *Playout) Endpoint() string {
	return p.config.GetString(Endpoint)
}

func (p *Playout) UseListenBrainz() bool {
	return p.config.GetBool(EnableListenBrainz)
}

func (p *Playout) UseTrackActivity() bool {
	return p.config.GetBool(EnableTrackActivity)
}

func (p *Playout) UserAgent() string {
	return UserAgent
}

func (p *Playout) Code() string {
	return p.tokens.GetString(Code)
}

func (p *Playout) CodeToken() string {
	return p.tokens.GetString(TokenCode)
}

func (p *Playout) AccessToken() string {
	return p.tokens.GetString(TokenAccess)
}

func (p *Playout) RefreshToken() string {
	return p.tokens.GetString(TokenRefresh)
}

func (p *Playout) MediaToken() string {
	return p.tokens.GetString(TokenMedia)
}

func (p *Playout) ListenBrainzToken() string {
	return p.tokens.GetString(TokenListenBrainz)
}

func (p *Playout) Transport() http.RoundTripper {
	return nil
}
