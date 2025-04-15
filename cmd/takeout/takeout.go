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
	"os"
	"path/filepath"
	"github.com/spf13/cobra"
	"takeoutfm.dev/takeout/internal/config"
	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/lib/systemd"
)

var rootCmd = &cobra.Command{
	Use:   "takeout",
	Short: "TakeoutFM server",
	Long:  `https://takeoutfm.com/`,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var configFile string
var configName string

func getConfig() (*config.Config, error) {
	if configFile != "" {
		config.SetConfigFile(configFile)
		return config.GetConfig()
	}

	if configName == "" {
		configName = os.Getenv("TAKEOUT_CONFIG")
	}

	config.AddConfigPath(".")

	configNames := []string{configName, "takeout", "config"}
	var err error
	var cfg *config.Config
	for _, name := range configNames {
		config.SetConfigName(name)
		cfg, err = config.GetConfig()
		if err == nil {
			break
		}
	}
	return cfg, err
}

func main() {
	if systemd.HasSystemd() {
		// allow systemd to run takeout w/o any args
		stateDir := systemd.GetStateDirectory("")
		if stateDir != "" {
			os.Chdir(stateDir)
		}
		if configFile == "" {
			// try /etc followed by /var/lib
			configDir := systemd.GetConfigDirectory("")
			if configDir == "" {
				configDir = systemd.GetStateDirectory("")
			}
			if configDir != "" {
				configFile = filepath.Join(configDir, "takeout.yaml")
			}
		}
		// use serve by default
		args := append([]string{serveCmd.Use}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
