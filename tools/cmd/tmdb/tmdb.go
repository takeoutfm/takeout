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
	"os"

	"github.com/spf13/cobra"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/log"
)

var rootCmd = &cobra.Command{
	Use:   "tmdb",
	Short: "",
	Long:  `https://takeout.fm/`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO
	},
}

var configFile string
var configPath string
var configName string

func getConfig() *config.Config {
	if configPath == "" {
		configPath = os.Getenv("TAKEOUT_HOME")
	}
	if configName == "" {
		configName = os.Getenv("TMDB_CONFIG")
	}
	if configFile != "" {
		config.SetConfigFile(configFile)
	} else {
		if configPath == "" {
			configPath = "."
		}
		if configName == "" {
			configName = "tmdb"
		}
		config.AddConfigPath(configPath)
		config.SetConfigName(configName)
	}
	config, err := config.GetConfig()
	log.CheckError(err)
	return config
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
