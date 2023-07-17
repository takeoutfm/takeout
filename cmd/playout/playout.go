// Copyright 2023 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/takeoutfm/takeout"
)

const (
	UserAgent = "Playout/" + takeout.Version + " (" + takeout.Contact + ")"

	TokenAccess  = "accesstoken"
	TokenRefresh = "refreshtoken"
	TokenMedia   = "mediatoken"
	TokenCode    = "codetoken"

	Code     = "code"
	Endpoint = "endpoint"
)

func globalConfig() *viper.Viper {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("$HOME/.config/playout")
	cfg.AddConfigPath("$HOME/.playout")
	cfg.AddConfigPath("/etc/playout")
	return cfg
}

func tokensConfig() *viper.Viper {
	cfg := viper.New()
	cfg.SetConfigName("tokens")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath("$HOME/.config/playout")
	return cfg
}

type Playout struct {
	config *viper.Viper
	tokens *viper.Viper
}

func NewPlayout() *Playout {
	config := globalConfig()
	err := config.ReadInConfig()
	if err != nil {
		panic(err)
	}

	tokens := tokensConfig()
	err = tokens.ReadInConfig()
	if err != nil {
		// TODO err is ok sometimes
		fmt.Printf("tokens %s\n", err)
	}

	// fmt.Printf("Code is %s\n", tokens.GetString(Code))
	// fmt.Printf("CodeToken is %s\n", tokens.GetString(TokenCode))
	// fmt.Printf("AccessToken is %s\n", tokens.GetString(TokenAccess))
	// fmt.Printf("RefreshToken is %s\n", tokens.GetString(TokenRefresh))
	// fmt.Printf("MediaToken is %s\n", tokens.GetString(TokenMedia))

	p := Playout{config: config, tokens: tokens}
	return &p
}

func (p Playout) UpdateAccessCode(code, access string) error {
	p.tokens.Set(Code, code)
	p.tokens.Set(TokenCode, access)
	return p.writeTokens()
}

func (p Playout) UpdateTokens(access, refresh, media string) error {
	p.tokens.Set(TokenAccess, access)
	p.tokens.Set(TokenRefresh, refresh)
	p.tokens.Set(TokenMedia, media)
	return p.writeTokens()
}

func (p Playout) UpdateAccessToken(value string) {
	p.tokens.Set(TokenAccess, value)
	err := p.writeTokens()
	if err != nil {
		panic(err)
	}
}

func (p Playout) writeTokens() error {
	err := p.tokens.WriteConfig()
	if err != nil {
		err = p.tokens.SafeWriteConfig()
	}
	return err
}

func (p Playout) Endpoint() string {
	return p.config.GetString(Endpoint)
}

func (p Playout) UserAgent() string {
	return UserAgent
}

func (p Playout) Code() string {
	return p.tokens.GetString(Code)
}

func (p Playout) CodeToken() string {
	return p.tokens.GetString(TokenCode)
}

func (p Playout) AccessToken() string {
	return p.tokens.GetString(TokenAccess)
}

func (p Playout) RefreshToken() string {
	return p.tokens.GetString(TokenRefresh)
}

func (p Playout) MediaToken() string {
	return p.tokens.GetString(TokenMedia)
}

var rootCmd = &cobra.Command{
	Use:   "playout",
	Short: "playout",
	Long:  `https://takeout.fm/`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO
	},
}

func main() {
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/playout")
	viper.AddConfigPath("$HOME/.playout")
	viper.AddConfigPath("/etc/playout")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
