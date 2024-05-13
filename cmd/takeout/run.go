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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/takeoutfm/takeout/internal/auth"
	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/internal/server"
	"github.com/takeoutfm/takeout/lib/log"
)

var options *viper.Viper

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "takeout server",
	Long:  `TODO`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run(options)
	},
}

const (
	passwordSize = 12
	tokenSize    = 16
	secretChars  = "0123456789abcdefghijklmnpqrstuvwxyzABCDEFGHILKMNOPQRSTUVWXYZ~`!@#$%^&*()_-+={[}];:,<.>/?"
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

func addUser(cfg *config.Config, userid, password, mediaName string) (bool, error) {
	added := false
	a := auth.NewAuth(cfg)
	err := a.Open()
	if err != nil {
		return false, err
	}
	_, err = a.User(userid)
	if err != nil {
		err = a.AddUser(userid, password)
		if err != nil {
			return false, err
		}
		added = true
		err = a.AssignMedia(userid, mediaName)
		if err != nil {
			return added, err
		}
	}
	return added, nil
}

func run(opts *viper.Viper) error {
	logFile := opts.GetString("log")
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		log.CheckError(err)
		log.SetOutput(file)
	}

	err := createConfig(opts)
	if err != nil {
		return err
	}

	os.Chdir(opts.GetString("dir"))

	cfg, err := getConfig()
	if err != nil {
		return err
	}

	// add user as needed
	userid := opts.GetString("user")
	password := opts.GetString("password")
	mediaName := opts.GetString("name")
	newPassword := false
	if password == "" {
		newPassword = true
		password = generateSecret(passwordSize)
	}
	added, err := addUser(cfg, user, password, mediaName)

	if opts.GetBool("setup_only") == true {
		if added && newPassword {
			fmt.Println("userid", userid, "password", password)
		}
		return nil
	}

	// run sync jobs
	jobs := []string{"media", "stations"}
	for _, job := range jobs {
		server.Job(cfg, job)
	}

	if added && newPassword {
		fmt.Println("userid", userid, "password", password)
	}

	// start the server
	return server.Serve(cfg)
}

func createConfig(opts *viper.Viper) error {
	takeoutDir := opts.GetString("dir")
	cacheDir := opts.GetString("cache")

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
	mediaName := opts.GetString("name")
	myMediaDir := filepath.Join(mediaDir, mediaName)
	os.MkdirAll(myMediaDir, 0700)
	config := []string{"Buckets:"}

	doit := func(mediaType, uri string) {
		if uri != "" {
			u, err := url.Parse(uri)
			if err != nil {
				panic(err)
			}
			if u.Scheme == "s3" {
				config = append(config,
					"  - Media: "+mediaType,
					"    S3:",
					"      Endpoint: "+opts.GetString("endpoint"),
					"      Region: "+opts.GetString("region"),
					"      AccessKeyID: "+opts.GetString("access_key_id"),
					"      SecretAccessKey: "+opts.GetString("secret_access_key"),
					"      BucketName: "+u.Host,
					"      ObjectPrefix: "+strings.TrimPrefix(u.Path, "/"),
					"      URLExpiration: "+"15m",
					"")
			} else {
				config = append(config,
					"  - Media: "+mediaType,
					"    FS:",
					"      Root: "+u.Path, "")
			}
		}
	}
	doit("music", opts.GetString("music"))
	doit("video", opts.GetString("video"))

	config = append(config,
		"Client:",
		"  CacheDir: "+httpCacheDir, "")
	writeConfig(myMediaDir, "config.yaml", config)

	return nil
}

var optFile string

func makeOptions() *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	if optFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.config/takeout")
		v.AddConfigPath("$HOME/.takeout")
	}
	v.ReadInConfig()
	return v
}

func init() {
	runCmd.Flags().StringVar(&optFile, "file", "", "configuration file")
	runCmd.Flags().Bool("setup_only", false, "Setup configuration only")
	runCmd.Flags().String("listen", "127.0.0.1:3000", "Address to listen on")
	runCmd.Flags().String("log", "", "Log output file")
	runCmd.Flags().String("user", "takeout", "Takeout userid")
	runCmd.Flags().String("password", "", "Takeout password")
	runCmd.Flags().String("dir", "/var/lib/takeout", "Takeout directory")
	runCmd.Flags().String("cache", "/var/cache/takeout", "Takeout cache directory")
	runCmd.Flags().String("name", "mymedia", "media name")
	runCmd.Flags().String("music", "", "dir or s3://bucket/prefix")
	runCmd.Flags().String("movies", "", "dir or s3://bucket/prefix")
	runCmd.Flags().String("endpoint", os.Getenv("AWS_ENDPOINT_URL"), "s3 endpoint (host name)")
	runCmd.Flags().String("region", os.Getenv("AWS_DEFAULT_REGION"), "s3 region")
	runCmd.Flags().String("access_key_id", os.Getenv("AWS_ACCESS_KEY_ID"), "s3 access key id")
	runCmd.Flags().String("secret_access_key", os.Getenv("AWS_SECRET_ACCESS_KEY"), "s3 secret access key")

	options = makeOptions()
	options.BindPFlags(runCmd.Flags())

	rootCmd.AddCommand(runCmd)
}
