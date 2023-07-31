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
	"github.com/takeoutfm/takeout/client/playout"
)

func systemConfig() *viper.Viper {
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

var rootCmd = &cobra.Command{
	Use:   "playout",
	Short: "playout",
	Long:  `https://takeout.fm/`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO
	},
}

func NewPlayout() *playout.Playout {
	config := systemConfig()
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

	return playout.NewPlayout(config, tokens)
}

func main() {
	// viper.SetConfigType("yaml")
	// viper.AddConfigPath(".")
	// viper.AddConfigPath("$HOME/.config/playout")
	// viper.AddConfigPath("$HOME/.playout")
	// viper.AddConfigPath("/etc/playout")
	// if err := viper.ReadInConfig(); err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
