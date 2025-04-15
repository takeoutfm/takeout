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

// Package config collects all configuration for the server with a single model
// which allows for easy viper-based configuration files.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"takeoutfm.dev/takeout"
	"takeoutfm.dev/takeout/lib/bucket"
	"takeoutfm.dev/takeout/lib/client"
	"takeoutfm.dev/takeout/lib/fanart"
	g "takeoutfm.dev/takeout/lib/gorm"
	"takeoutfm.dev/takeout/lib/lastfm"
	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/lib/systemd"
	"takeoutfm.dev/takeout/lib/search"
	"takeoutfm.dev/takeout/lib/tmdb"

	"gopkg.in/yaml.v3"
)

var (
	ErrTestConfig   = errors.New("missing test config")
	ErrInvalidCache = errors.New("invalid cache entry")
)

const (
	MediaMusic = "music"
	MediaFilm  = "film"
	MediaTV    = "tv"
)

type DatabaseConfig struct {
	Driver string
	Source string
	Logger string
}

func (c DatabaseConfig) GormConfig() *gorm.Config {
	return &gorm.Config{Logger: gormLogger(c.Logger)}
}

type Template struct {
	Text  string
	templ *template.Template
}

func (t *Template) Template() *template.Template {
	if t.templ == nil {
		t.templ = template.Must(template.New("t").Parse(t.Text))
	}
	return t.templ
}

func (t *Template) Execute(vars interface{}) string {
	var buf bytes.Buffer
	_ = t.Template().Execute(&buf, vars)
	return buf.String()
}

type AssistantResponse struct {
	Speech Template
	Text   Template
}

type AssistantConfig struct {
	ProjectID       string
	TrackLimit      int
	RecentLimit     int
	Welcome         AssistantResponse
	Play            AssistantResponse
	Error           AssistantResponse
	Link            AssistantResponse
	Linked          AssistantResponse
	Guest           AssistantResponse
	Recent          AssistantResponse
	Release         AssistantResponse
	SuggestionAuth  string
	SuggestionNew   string
	MediaObjectName Template
	MediaObjectDesc Template
}

type ContentDescription struct {
	ContentType string `json:"contentType"`
	URL         string `json:"url"`
}

type RadioStream struct {
	Creator     string
	Title       string
	Image       string
	Description string
	Source      []ContentDescription
}

type MusicConfig struct {
	ArtistFile           string
	ArtistRadioBreadth   int
	ArtistRadioDepth     int
	TrackRadioBreadth    int
	TrackRadioDepth      int
	DB                   DatabaseConfig
	DeepLimit            int
	PopularLimit         int
	RadioGenres          []string
	RadioLimit           int
	RadioOther           map[string]string
	RadioSearchLimit     int
	RadioSeries          []string
	RadioStreams         []RadioStream
	Recent               time.Duration
	RecentLimit          int
	ReleaseCountries     []string
	SearchIndexName      string
	SearchLimit          int
	SimilarArtistsLimit  int
	SimilarReleases      time.Duration
	SimilarReleasesLimit int
	SinglesLimit         int
	artistMap            map[string]string
	SyncInterval         time.Duration
	PopularSyncInterval  time.Duration
	SimilarSyncInterval  time.Duration
	CoverSyncInterval    time.Duration
	RelatedArtists       time.Duration
}

type FilmConfig struct {
	DB                   DatabaseConfig
	ReleaseCountries     []string
	CastLimit            int
	CrewJobs             []string
	Recent               time.Duration
	RecentLimit          int
	SearchIndexName      string
	SearchLimit          int
	Recommend            RecommendConfig
	SyncInterval         time.Duration
	PosterSyncInterval   time.Duration
	BackdropSyncInterval time.Duration
	DuplicateResolution  string
}

func (c FilmConfig) SortedCast(credits tmdb.Credits) []tmdb.Cast {
	return sortedCast(credits, c.CastLimit)
}

func (c FilmConfig) RelevantCrew(credits tmdb.Credits) []tmdb.Crew {
	return relevantCrew(credits, c.CrewJobs)
}

type TVConfig struct {
	DB                   DatabaseConfig
	ReleaseCountries     []string
	CastLimit            int
	CrewJobs             []string
	Recent               time.Duration
	RecentLimit          int
	SearchIndexName      string
	SearchLimit          int
	SyncInterval         time.Duration
	PosterSyncInterval   time.Duration
	BackdropSyncInterval time.Duration
	StillSyncInterval    time.Duration
}

func (c TVConfig) SortedCast(credits tmdb.Credits) []tmdb.Cast {
	return sortedCast(credits, c.CastLimit)
}

func (c TVConfig) SortedGuests(credits tmdb.Credits) []tmdb.Cast {
	return sortedGuests(credits, c.CastLimit)
}

func (c TVConfig) RelevantCrew(credits tmdb.Credits) []tmdb.Crew {
	return relevantCrew(credits, c.CrewJobs)
}

type PodcastConfig struct {
	DB              DatabaseConfig
	Series          []string
	Client          client.Config
	RecentLimit     int
	EpisodeLimit    int
	SyncInterval    time.Duration
	SearchIndexName string
	SearchLimit     int
}

type ProgressConfig struct {
	DB DatabaseConfig
}

type ActivityConfig struct {
	DB                DatabaseConfig
	RecentMoviesTitle string
	RecentTracksTitle string
	TrackLimit        int
	MovieLimit        int
	TopArtistsLimit   int
	TopArtistsTitle   string
	TopTracksLimit    int
	TopTracksTitle    string
	TopReleasesLimit  int
	TopReleasesTitle  string
	TopMoviesLimit    int
	TopMoviesTitle    string
}

type RecommendConfig struct {
	When []DateRecommend
}

type DateRecommend struct {
	Name   string
	Layout string
	Match  string
	Query  string
}

type TMDBAPIConfig struct {
	tmdb.Config    `mapstructure:",squash"`
	FileTemplate   Template
	SeriesTemplate Template
}

type SetlistAPIConfig struct {
	ApiKey string
}

type TOTPConfig struct {
	Issuer string
}

type TokenConfig struct {
	Issuer     string
	Age        time.Duration
	Secret     string
	SecretFile string
}

type AuthConfig struct {
	DB              DatabaseConfig
	SessionAge      time.Duration
	CodeAge         time.Duration
	SecureCookies   bool
	AccessToken     TokenConfig
	MediaToken      TokenConfig
	CodeToken       TokenConfig
	FileToken       TokenConfig
	TOTP            TOTPConfig
	PasswordEntropy int
}

type ServerConfig struct {
	Listen      string
	KeyDir      string // exists for dollar expansion
	DataDir     string // exsists for dollar expansion
	MediaDir    string
	ImageClient client.Config
	IncludeDirs []string
	ExcludeDirs []string
}

type Config struct {
	Auth      AuthConfig
	Buckets   []bucket.Config
	Client    client.Config
	Fanart    fanart.Config
	LastFM    lastfm.Config
	Music     MusicConfig
	TMDB      TMDBAPIConfig
	Search    search.Config
	Server    ServerConfig
	Film      FilmConfig
	TV        TVConfig
	Assistant AssistantConfig
	Podcast   PodcastConfig
	Progress  ProgressConfig
	Activity  ActivityConfig
}

func (c Config) NewGetter() client.Getter {
	return client.NewGetter(c.Client)
}

func (c Config) NewGetterWith(o client.Config) client.Getter {
	newConfig := c.Client
	newConfig.Merge(o)
	return client.NewGetter(newConfig)
}

func (c Config) NewCacheOnlyGetter() client.Getter {
	return client.NewCacheOnlyGetter(c.Client)
}

func (c Config) NewSearcher() search.Searcher {
	return search.NewSearcher(c.Search)
}

func (mc *MusicConfig) UserArtistID(name string) (string, bool) {
	mbid, ok := mc.artistMap[name]
	return mbid, ok
}

func readJsonStringMap(file string, m *map[string]string) (err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(data), m)
	return
}

func (mc *MusicConfig) readMaps() {
	if mc.ArtistFile != "" {
		readJsonStringMap(mc.ArtistFile, &mc.artistMap)
	}
}

func configDefaults(v *viper.Viper) {
	v.SetDefault("Server.Listen", "127.0.0.1:3000")
	v.SetDefault("Server.DataDir", systemd.GetStateDirectory("."))
	v.SetDefault("Server.MediaDir", systemd.GetStateDirectory("."))
	v.SetDefault("Server.ImageClient.CacheDir", filepath.Join(systemd.GetCacheDirectory("."), "imagecache"))
	v.SetDefault("Server.ImageClient.UserAgent", userAgent())
	v.SetDefault("Server.ImageClient.MaxAge", "720h") // 30 days
	// potential include could be /media, /mnt, /opt, /srv
	v.SetDefault("Server.IncludeDirs", []string{})
	// by default provide some reasonable excludes
	v.SetDefault("Server.ExcludeDirs", []string{
		"/bin/", "/boot/", "/dev/", "/etc/", "/lib/", "/proc/", "/run/", "/sbin/", "/root/", "/sys/",
	})

	v.SetDefault("Auth.DB.Driver", "sqlite3")
	v.SetDefault("Auth.DB.Logger", "default")
	v.SetDefault("Auth.DB.Source", "${Server.DataDir}/auth.db")
	v.SetDefault("Auth.SessionAge", "720h") // 30 days
	v.SetDefault("Auth.CodeAge", "5m")
	v.SetDefault("Auth.SecureCookies", "true")
	v.SetDefault("Auth.AccessToken.Age", "4h")
	v.SetDefault("Auth.AccessToken.Issuer", "takeout")
	v.SetDefault("Auth.AccessToken.Secret", "")     // must be assigned in config file
	v.SetDefault("Auth.AccessToken.SecretFile", "") // must be assigned in config file
	v.SetDefault("Auth.MediaToken.Age", "8766h")    // 1 year
	v.SetDefault("Auth.MediaToken.Issuer", "takeout")
	v.SetDefault("Auth.MediaToken.Secret", "")     // must be assigned in config file
	v.SetDefault("Auth.MediaToken.SecretFile", "") // must be assigned in config file
	v.SetDefault("Auth.CodeToken.Age", "5m")
	v.SetDefault("Auth.CodeToken.Issuer", "takeout")
	v.SetDefault("Auth.CodeToken.Secret", "")     // must be assigned in config file
	v.SetDefault("Auth.CodeToken.SecretFile", "") // must be assigned in config file
	v.SetDefault("Auth.FileToken.Age", "1h")
	v.SetDefault("Auth.FileToken.Issuer", "takeout")
	v.SetDefault("Auth.FileToken.Secret", "")     // must be assigned in config file
	v.SetDefault("Auth.FileToken.SecretFile", "") // must be assigned in config file
	v.SetDefault("Auth.PasswordEntropy", "60")    // 50-70 bits is reasonable

	v.SetDefault("Progress.DB.Driver", "sqlite3")
	v.SetDefault("Progress.DB.Source", "${Server.DataDir}/progress.db")
	v.SetDefault("Progress.DB.Logger", "default")

	v.SetDefault("Activity.DB.Driver", "sqlite3")
	v.SetDefault("Activity.DB.Source", "${Server.DataDir}/activity.db")
	v.SetDefault("Activity.DB.Logger", "default")
	v.SetDefault("Activity.RecentMoviesTitle", "Recently Watched")
	v.SetDefault("Activity.RecentTracksTitle", "Recently Played")
	v.SetDefault("Activity.MovieLimit", "999")
	v.SetDefault("Activity.TrackLimit", "999")
	v.SetDefault("Activity.TopArtistsLimit", "999")
	v.SetDefault("Activity.TopArtistsTitle", "Top Artists")
	v.SetDefault("Activity.TopTracksLimit", "9999") // year of track listens
	v.SetDefault("Activity.TopTracksTitle", "Top Tracks")
	v.SetDefault("Activity.TopReleasesLimit", "999")
	v.SetDefault("Activity.TopReleasesTitle", "Top Releases")
	v.SetDefault("Activity.TopMoviesLimit", "999")
	v.SetDefault("Activity.TopMoviesTitle", "Top Movies")

	// TODO apply as default
	// v.SetDefault("Bucket.URLExpiration", "15m")

	v.SetDefault("Client.CacheDir", filepath.Join(systemd.GetCacheDirectory("."), "httpcache"))
	v.SetDefault("Client.MaxAge", "720h") // 30 days
	v.SetDefault("Client.UserAgent", userAgent())

	v.SetDefault("Fanart.ProjectKey", "93ede276ba6208318031727060b697c8")

	v.SetDefault("LastFM.Key", "")
	v.SetDefault("LastFM.Secret", "")

	v.SetDefault("Music.ArtistRadioBreadth", "10")
	v.SetDefault("Music.ArtistRadioDepth", "3")
	v.SetDefault("Music.TrackRadioBreadth", "10")
	v.SetDefault("Music.TrackRadioDepth", "5")
	v.SetDefault("Music.DeepLimit", "50")
	v.SetDefault("Music.PopularLimit", "50")
	v.SetDefault("Music.RadioLimit", "25")
	v.SetDefault("Music.RadioSearchLimit", "1000")

	v.SetDefault("Music.Recent", "8760h") // 1 year
	v.SetDefault("Music.RecentLimit", "50")
	v.SetDefault("Music.SearchIndexName", "music")
	v.SetDefault("Music.SearchLimit", "100")
	v.SetDefault("Music.SimilarArtistsLimit", "10")
	v.SetDefault("Music.SimilarReleases", "8760h") // +/- 1 year
	v.SetDefault("Music.SimilarReleasesLimit", "10")
	v.SetDefault("Music.SinglesLimit", "50")
	v.SetDefault("Music.SyncInterval", "1h")
	v.SetDefault("Music.PopularSyncInterval", "24h")
	v.SetDefault("Music.SimilarSyncInterval", "24h")
	v.SetDefault("Music.CoverSyncInterval", "24h")
	v.SetDefault("Music.RelatedArtists", "43800h") // +/- 5 years

	// see https://wiki.musicbrainz.org/Release_Country
	v.SetDefault("Music.ReleaseCountries", []string{
		"US", // United States
		"XW", // Worldwide
		"XE", // Europe
	})

	v.SetDefault("Music.DB.Driver", "sqlite3")
	v.SetDefault("Music.DB.Source", "music.db")
	v.SetDefault("Music.DB.Logger", "default")

	v.SetDefault("TMDB.Key", "903a776b0638da68e9ade38ff538e1d3")
	v.SetDefault("TMDB.Language", "en-US")
	v.SetDefault("TMDB.FileTemplate.Text",
		"{{.Title}} ({{.Year}}){{if .Definition}} - {{.Definition}}{{end}}{{.Extension}}")
	v.SetDefault("TMDB.SeriesTemplate.Text",
		"{{.Series}} ({{.Year}}) - S{{.Season}}E{{.Episode}} - {{.Name}}{{.Extension}}")

	v.SetDefault("Search.IndexDir", ".")

	v.SetDefault("Film.DB.Driver", "sqlite3")
	v.SetDefault("Film.DB.Source", "film.db")
	v.SetDefault("Film.DB.Logger", "default")
	v.SetDefault("Film.ReleaseCountries", []string{
		"US",
	})
	v.SetDefault("Film.CastLimit", "25")
	v.SetDefault("Film.CrewJobs", []string{
		"Director",
		"Executive Producer",
		"Novel",
		"Producer",
		"Screenplay",
		"Story",
	})
	v.SetDefault("Film.Recent", "8760h") // 1 year
	v.SetDefault("Film.RecentLimit", "50")
	v.SetDefault("Film.SearchIndexName", "film")
	v.SetDefault("Film.SearchLimit", "100")
	v.SetDefault("Film.SyncInterval", "1h")
	v.SetDefault("Film.PosterSyncInterval", "24h")
	v.SetDefault("Film.BackdropSyncInterval", "24h")
	v.SetDefault("Film.DuplicateResolution", "largest")

	v.SetDefault("TV.DB.Driver", "sqlite3")
	v.SetDefault("TV.DB.Source", "tv.db")
	v.SetDefault("TV.DB.Logger", "default")
	v.SetDefault("TV.ReleaseCountries", []string{
		"US",
	})
	v.SetDefault("TV.CastLimit", "25")
	v.SetDefault("TV.CrewJobs", []string{
		"Director",
		"Executive Producer",
		"Novel",
		"Producer",
		"Screenplay",
		"Story",
	})
	v.SetDefault("TV.Recent", "8760h") // 1 year
	v.SetDefault("TV.RecentLimit", "50")
	v.SetDefault("TV.SearchIndexName", "tv")
	v.SetDefault("TV.SearchLimit", "100")
	v.SetDefault("TV.SyncInterval", "1h")
	v.SetDefault("TV.PosterSyncInterval", "24h")
	v.SetDefault("TV.BackdropSyncInterval", "24h")
	v.SetDefault("TV.StillSyncInterval", "24h")

	// see https://musicbrainz.org/search (series)
	v.SetDefault("Music.RadioSeries", []string{
		"The Rolling Stone Magazine's 500 Greatest Songs of All Time",
	})

	v.SetDefault("Music.RadioOther", map[string]string{
		"Series Hits": "+series:*",
		"Top Hits":    "+popularity:1",
		"Top 3 Hits":  "+popularity:<4",
		"Top 5 Hits":  "+popularity:<6",
		"Top 10 Hits": "+popularity:<11",
		"Covers":      "+type:cover",
		"Live Hits":   "+type:live +popularity:<3",
	})

	v.SetDefault("Assistant.ProjectID", "undefined")
	v.SetDefault("Assistant.TrackLimit", "10")
	v.SetDefault("Assistant.RecentLimit", "3")
	v.SetDefault("Assistant.Welcome.Speech.Text", "Welcome to Takeout")
	v.SetDefault("Assistant.Welcome.Text.Text", "Welcome to Takeout")
	v.SetDefault("Assistant.Play.Speech.Text", "Enjoy the music")
	v.SetDefault("Assistant.Play.Text.Text", "")
	v.SetDefault("Assistant.Error.Speech.Text", "Please try again")
	v.SetDefault("Assistant.Error.Text.Text", "Please try again")
	v.SetDefault("Assistant.Link.Speech.Text", "Link this device to Takeout using code {{.Code}}")
	v.SetDefault("Assistant.Link.Text.Text", "Link code is: {{.Code}}")
	v.SetDefault("Assistant.Linked.Speech.Text", "Takeout is now linked")
	v.SetDefault("Assistant.Linked.Text.Text", "Takeout is now linked")
	v.SetDefault("Assistant.Guest.Speech.Text", "Guest not supported. A verified user is required.")
	v.SetDefault("Assistant.Guest.Text.Text", "Guest not supported. A verified user is required.")
	v.SetDefault("Assistant.Recent.Speech.Text", "Recently added albums are ")
	v.SetDefault("Assistant.Recent.Text.Text", "Recent Albums: ")
	v.SetDefault("Assistant.Release.Speech.Text", "{{.Name}} by {{.Artist}}")
	v.SetDefault("Assistant.Release.Text.Text", "{{.Artist}} \u2022 {{.Name}}")
	v.SetDefault("Assistant.SuggestionAuth", "Next")
	v.SetDefault("Assistant.SuggestionNew", "What's new")
	v.SetDefault("Assistant.MediaObjectName.Text", "{{.Title}}")
	v.SetDefault("Assistant.MediaObjectDesc.Text", "{{.Artist}} \u2022 {{.Release}}")

	v.SetDefault("Podcast.Client.MaxAge", "15m")
	v.SetDefault("Podcast.DB.Driver", "sqlite3")
	v.SetDefault("Podcast.DB.Source", "podcast.db")
	v.SetDefault("Podcast.DB.Logger", "default")
	v.SetDefault("Podcast.EpisodeLimit", "52")
	v.SetDefault("Podcast.RecentLimit", "25")
	v.SetDefault("Podcast.SearchIndexName", "podcast")
	v.SetDefault("Podcast.SearchLimit", "100")
	v.SetDefault("Podcast.SyncInterval", "1h")
	v.SetDefault("Podcast.Series", []string{
		"https://feeds.twit.tv/twit.xml",
		"https://feeds.twit.tv/sn.xml",
		"https://feeds.twit.tv/twig.xml",
		"https://feeds.eff.org/howtofixtheinternet",
		"https://feeds.npr.org/510019/podcast.xml", // all songs considered
	})
}

func userAgent() string {
	return takeout.AppName + "/" + takeout.Version + " (" + takeout.Contact + ")"
}

func readConfig(v *viper.Viper) (*Config, error) {
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	rootDir := filepath.Dir(v.ConfigFileUsed())
	return makeConfig(v, rootDir)
}

func makeConfig(v *viper.Viper, rootDir string) (*Config, error) {
	postProcessConfig(v, rootDir)

	var config Config
	err := v.Unmarshal(&config)
	config.Music.readMaps()
	return &config, err
}

const opInclude = "include"

func postProcessConfig(v *viper.Viper, rootDir string) {
	for _, k := range v.AllKeys() {
		if k == opInclude {
			var paths []string
			path := v.GetString(k)
			if path != "" {
				paths = append(paths, path)
			} else {
				p := v.GetStringSlice(k)
				paths = append(paths, p...)
			}
			for _, path := range paths {
				doInclude(v, path, rootDir)
			}
		}
	}
	v.Set(opInclude, "") // remove include
	for _, k := range v.AllKeys() {
		postProcessKey(v, rootDir, k)
	}
}

func expandValue(v *viper.Viper, val string) string {
	if strings.Contains(val, "$") {
		// expand $var or ${var}
		val = os.Expand(val, func(s string) string {
			r := v.Get(s)
			if r == nil {
				log.Panicf("'%s' not found for %s\n", s, val)
			}
			if _, ok := r.(string); !ok {
				log.Panicf("'%s' not a string for %s\n", s, val)
			}
			return r.(string)
		})
		if strings.Contains(val, "$") {
			// keep going
			val = expandValue(v, val)
		}
	}
	return val
}

var pathRegexp = regexp.MustCompile(`(file|dir|source)$`)

func resolvePathRef(v *viper.Viper, rootDir, key, val string) string {
	if pathRegexp.MatchString(key) {
		// resolve relative paths only
		if strings.HasPrefix(val, "/") == false &&
			strings.Contains(val, "@") == false &&
			strings.Contains(val, "::") == false {
			val = filepath.Join(rootDir, val)
		}
	}
	return val
}

func processMap(v *viper.Viper, rootDir string, m map[string]any) {
	for key, val := range m {
		switch val.(type) {
		case map[string]any:
			// nested struct
			processMap(v, rootDir, val.(map[string]any))
		case string:
			// string within struct
			m[key] = expandValue(v, val.(string))
		}
	}
}

func postProcessKey(v *viper.Viper, rootDir, key string) {
	val := v.Get(key)
	switch val.(type) {
	case []any:
		// viper AllKeys includes nested structs as a.b.c except for
		// arrays so they need to be handled separately
		list := val.([]any)
		for i, x := range list {
			switch x.(type) {
			case map[string]any:
				// array of struct
				processMap(v, rootDir, x.(map[string]any))
			case string:
				// array of string
				list[i] = expandValue(v, x.(string))
			}
		}
	case string:
		sval := expandValue(v, v.GetString(key))
		sval = resolvePathRef(v, rootDir, key, sval)
		v.Set(key, sval)
	}
}

// Include directive can be used as follows to include files from various
// sources (yaml example) as an array of strings or a single string:
//
// include:
//   - file.yaml
//   - /path/to/file.yaml
//   - file:///path/to/file.yaml
//   - http://host/path/to/file.yaml
//   - https://host/path/to/file.yaml
//
// include: file.yaml # or any of the above
func doInclude(v *viper.Viper, inc, rootDir string) {
	if strings.Contains(inc, "://") == false &&
		strings.HasPrefix(inc, "/") == false {
		// resolve relative paths
		inc = filepath.Join(rootDir, inc)
	}
	vv, err := includeConfig(inc)
	if err != nil {
		log.Panicf("include '%s': %s", inc, err)
	}
	// use included config to (re)set this value in the parent config
	for _, k := range vv.AllKeys() {
		v.Set(k, vv.Get(k))
	}
}

func includeConfig(path string) (*viper.Viper, error) {
	log.Printf("include '%s'\n", path)
	body, err := client.Get(path)
	if err != nil {
		return nil, err
	}
	// need extension for reading
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	// load include config
	vv := viper.New()
	vv.SetConfigType(ext)
	err = vv.ReadConfig(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return vv, nil
}

// func TestConfig() (*Config, error) {
// 	testDir := os.Getenv("TEST_CONFIG")
// 	if testDir == "" {
// 		return nil, ErrTestConfig
// 	}
// 	v := viper.New()
// 	configDefaults(v)
// 	v.SetConfigFile(filepath.Join(testDir, "test.yaml"))
// 	v.SetDefault("Music.DB.Source", filepath.Join(testDir, "music.db"))
// 	v.SetDefault("Auth.DB.Source", filepath.Join(testDir, "auth.db"))
// 	return readConfig(v)
// }

func TestingConfig() (*Config, error) {
	v := viper.New()
	configDefaults(v)
	v.SetConfigFile("testing.yaml")

	memory := "file::memory:?cache=shared"
	v.SetDefault("Activity.DB.Source", memory)
	v.SetDefault("Auth.DB.Source", "${Activity.DB.Source}")
	v.SetDefault("Music.DB.Source", "${Activity.DB.Source}")
	v.SetDefault("Podcast.DB.Source", "${Activity.DB.Source}")
	v.SetDefault("Progress.DB.Source", "${Activity.DB.Source}")
	v.SetDefault("Film.DB.Source", "${Activity.DB.Source}")

	v.SetDefault("Auth.AccessToken.Issuer", "takeout.test")
	v.SetDefault("Auth.AccessToken.Age", "5m")
	v.SetDefault("Auth.AccessToken.Secret", "Wtex5hJ3vxZbkCSs")
	v.SetDefault("Auth.MediaToken.Issuer", "takeout.test")
	v.SetDefault("Auth.MediaToken.Age", "5m")
	v.SetDefault("Auth.MediaToken.Secret", "H1ys/pP/iNiQUl4k")
	v.SetDefault("Auth.CodeToken.Issuer", "takeout.test")
	v.SetDefault("Auth.CodeToken.Age", "5m")
	v.SetDefault("Auth.CodeToken.Secret", "Rg3ac20IPqyL7oAC")
	v.SetDefault("Auth.FileToken.Issuer", "takeout.test")
	v.SetDefault("Auth.FileToken.Age", "5m")
	v.SetDefault("Auth.FileToken.Secret", "38614926l1LxpUUW")

	v.SetDefault("Search.IndexDir", "")
	v.SetDefault("Music.SearchIndexName", "")
	v.SetDefault("Podcast.SearchIndexName", "")
	v.SetDefault("Film.SearchIndexName", "")

	return makeConfig(v, "/tmp")
}

var configFile, configPath, configName string

func SetConfigFile(path string) {
	configFile = path
}

func AddConfigPath(path string) {
	configPath = path
}

func SetConfigName(name string) {
	configName = name
}

// GetConfig uses viper to load the default configuration.
func GetConfig() (*Config, error) {
	v := viper.New()
	if configFile != "" {
		v.SetConfigFile(configFile)
	}
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	if configName != "" {
		v.SetConfigName(configName)
	}
	configDefaults(v)
	return readConfig(v)
}

var dirConfigCache = make(map[string]interface{})

// LoadConfig uses viper to load a config file in the provided directory. The
// result is cached.
func LoadConfig(dir string) (*Config, error) {
	if val, ok := dirConfigCache[dir]; ok {
		switch val.(type) {
		case *Config:
			return val.(*Config), nil
		case error:
			return nil, val.(error)
		}
		log.Panicln(ErrInvalidCache)
	}
	v := viper.New()
	v.AddConfigPath(dir)
	configDefaults(v)
	c, err := readConfig(v)
	if err != nil {
		// cache the error and don't try again
		log.Println("LoadConfig failed: ", err)
		dirConfigCache[dir] = err
	} else {
		// cache the loaded config and don't load again
		dirConfigCache[dir] = c
		// TODO revisit watching and rebuilding all services (music,
		// film, podcast) would need to be reconstructed and not sure
		// if that's desired.
	}
	return c, err
}

func gormLogger(name string) logger.Interface {
	return g.Logger(name)
}

func (c Config) Write(w io.Writer) error {
	buf, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

func sortedCast(credits tmdb.Credits, limit int) []tmdb.Cast {
	cast := credits.SortedCast()
	if len(cast) > limit {
		cast = cast[:limit]
	}
	return cast
}

func sortedGuests(credits tmdb.Credits, limit int) []tmdb.Cast {
	cast := credits.SortedGuests()
	if len(cast) > limit {
		cast = cast[:limit]
	}
	return cast
}

func relevantCrew(credits tmdb.Credits, jobs []string) []tmdb.Crew {
	return credits.CrewWithJobs(jobs)
}
