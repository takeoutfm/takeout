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
	"github.com/spf13/cobra"
	"github.com/takeoutfm/takeout/client/api"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return auth()
	},
}

func auth() error {
	if codeGet {
		return doCodeGet()
	}
	if codeCheck {
		return doCodeCheck()
	}
	return nil
}

func doCodeGet() error {
	playout := NewPlayout()

	accessCode, err := api.Code(playout)
	if err != nil {
		return err
	}

	err = playout.UpdateAccessCode(accessCode.Code, accessCode.AccessToken)
	if err == nil {
		fmt.Printf("code is %s\n", accessCode.Code)
	}

	return err
}

func doCodeCheck() error {
	playout := NewPlayout()

	tokens, err := api.CheckCode(playout)
	if err != nil {
		return err
	}
	err = playout.UpdateTokens(tokens.AccessToken, tokens.RefreshToken, tokens.MediaToken)

	return err
}

var codeGet bool
var codeCheck bool

func init() {
	authCmd.Flags().BoolVarP(&codeGet, "get", "g", false, "code get")
	authCmd.Flags().BoolVarP(&codeCheck, "check", "c", false, "code check")
	rootCmd.AddCommand(authCmd)
}
