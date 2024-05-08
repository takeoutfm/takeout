// Copyright 2024 defsub
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
	rando "math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/internal/server"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "takeout server",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

const (
	passwordSize = 12
	tokenSize  = 16
	secretChars = "0123456789abcdefghijklmnpqrstuvwxyzABCDEFGHILKMNOPQRSTUVWXYZ~`!@#$%^&*()_-+={[}];:,<.>/?"
)

func generateSecret(size int) string {
	var secret string
	rando.Seed(time.Now().UnixNano())
	for i := 0; i < size; i++ {
		n := rando.Intn(len(secretChars))
		secret += string(secretChars[n])
	}
	return secret
}

func writeSecret(dir, file, secret string) error {
	path := strings.Join([]string{dir, file}, "/")
	return os.WriteFile(path, []byte(secret), 0600)
}

func makeSecret(dir, file string) string {
	path := filepath.Join(dir, file)
	_, err := os.ReadFile(path)
	if err != nil {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			panic(err)
		}
		secret := generateSecret(tokenSize)
		err = writeSecret(dir, file, secret)
		if err != nil {
			panic(err)
		}
	}
	return path
}

func writeConfig(dir, file string, content []string) error {
	path := filepath.Join(dir, file)
	return os.WriteFile(path, []byte(strings.Join(content, "\n")), 0600)
}

func addUser(cfg *config.Config, userid, mediaName string) (string, error) {
	a := auth.NewAuth(cfg)
	err := a.Open()
	if err != nil {
		return "", err
	}

	var password string
	_, err = a.User(userid)
	if err != nil {
		password = generateSecret(passwordSize)
		err = a.AddUser(userid, password)
		if err != nil {
			return "", err
		}
		err = a.AssignMedia(userid, mediaName)
		if err != nil {
			return "", err
		}
	}
	return password, nil
}

var takeoutDir string
var cacheDir string
var musicDir string
var moviesDir string

func run() error {
	os.MkdirAll(takeoutDir, 0700)
	os.MkdirAll(cacheDir, 0700)

	imageCacheDir := filepath.Join(cacheDir, "imagecache")
	httpCacheDir := filepath.Join(cacheDir, "httpcache")
	keysDir := filepath.Join(takeoutDir, "keys")
	mediaDir := takeoutDir

	// create server configuration
	writeConfig(takeoutDir, "takeout.yaml", []string{
		"Server:",
		"  DataDir: " + takeoutDir,
		"  MediaDir: " + mediaDir,
		"  ImageClient:",
		"    CacheDir: " + imageCacheDir,
		"",
		"Auth:",
		"  AccessToken:",
		"    SecretFile: " + makeSecret(keysDir, "access.key"),
		"  MediaToken:",
		"    SecretFile: " + makeSecret(keysDir, "media.key"),
		"  CodeToken:",
		"    SecretFile: " + makeSecret(keysDir, "code.key"),
		"  FileToken:",
		"    SecretFile: " + makeSecret(keysDir, "file.key"),
		"",
	})

	// create media configuration
	mymedia := "mymedia"
	myMediaDir := filepath.Join(mediaDir, mymedia)
	os.MkdirAll(myMediaDir, 0700)
	config := []string{"Buckets:"}
	if musicDir != "" {
		config = append(config,
			"  - Media: music",
			"    FS:",
			"      Root: " + musicDir, "")
	}
	if moviesDir != "" {
		config = append(config,
			"  - Media: video",
			"    FS:",
			"      Root: " + moviesDir, "")
	}
	config = append(config,
		"Client:",
		"  CacheDir: " + httpCacheDir, "")
	writeConfig(myMediaDir, "config.yaml", config)

	os.Chdir(takeoutDir)
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	// add user as needed
	userid := "takeout"
	password, err := addUser(cfg, userid, mymedia)

	// sync media
	server.Job(cfg, "media")
	server.Job(cfg, "stations")

	if password != "" {
		fmt.Println("userid", userid, "password", password)
	}

	return server.Serve(cfg)
}

func init() {
	runCmd.Flags().String("listen", "127.0.0.1:3000", "Address to listen on")
	runCmd.Flags().StringVar(&takeoutDir, "dir", "/var/lib/takeout", "Takeout directory")
	runCmd.Flags().StringVar(&cacheDir, "cache", "/var/cache/takeout", "Takeout cache directory")
	runCmd.Flags().StringVar(&musicDir, "music", "", "Music directory")
	runCmd.Flags().StringVar(&moviesDir, "movies", "", "Movies directory")
	rootCmd.AddCommand(runCmd)
	viper.BindPFlag("Server.Listen", serveCmd.Flags().Lookup("listen"))
}
