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

package main

import (
	"fmt"
	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"
	"os"
	"takeoutfm.dev/takeout/internal/auth"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "user admin",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return doit()
	},
}

var user, pass, media, link string
var add, change, expire, generateTOTP bool

func doit() error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}
	a := auth.NewAuth(cfg)
	err = a.Open()
	if err != nil {
		return err
	}
	defer a.Close()

	if user != "" && pass != "" {
		if add {
			err := a.AddUser(user, pass)
			if err != nil {
				return err
			}
		} else if change {
			err := a.ChangePass(user, pass)
			if err != nil {
				return err
			}
		}
	}

	if user != "" && media != "" {
		err := a.AssignMedia(user, media)
		if err != nil {
			return err
		}
	}

	if expire && user != "" {
		err := a.ExpireAll(user)
		if err != nil {
			return err
		}
	}

	if user != "" && link != "" {
		session, err := a.LoginSession(user)
		if err != nil {
			return err
		}
		err = a.AuthorizeCode(link, session.Token)
		if err != nil {
			return err
		}
	}

	if generateTOTP && user != "" {
		url, err := auth.GenerateTOTP(cfg.Auth.TOTP, user)
		if err != nil {
			return err
		}

		err = a.AssignTOTP(user, url)
		if err != nil {
			return err
		}

		config := qrterminal.Config{
			Level:     qrterminal.L,
			Writer:    os.Stdout,
			BlackChar: qrterminal.WHITE,
			WhiteChar: qrterminal.BLACK,
			QuietZone: 1,
		}
		qrterminal.GenerateWithConfig(url, config)
		fmt.Println(url)
	}

	return nil
}

func init() {
	userCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file")
	userCmd.Flags().StringVarP(&user, "user", "u", "", "user")
	userCmd.Flags().StringVarP(&pass, "pass", "p", "", "pass")
	userCmd.Flags().StringVarP(&media, "media", "m", "", "media")
	userCmd.Flags().BoolVarP(&add, "add", "a", false, "add")
	userCmd.Flags().BoolVarP(&change, "change", "n", false, "change")
	userCmd.Flags().BoolVarP(&expire, "expire", "x", false, "expire all sessions")
	userCmd.Flags().BoolVar(&generateTOTP, "generate_totp", false, "generate & assign user a TOTP")
	userCmd.Flags().StringVarP(&link, "link", "l", "", "link code to new user session")
	rootCmd.AddCommand(userCmd)
}
