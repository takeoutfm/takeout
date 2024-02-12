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

// Package config collects all configuration for the server with a single model
// which allows for easy viper-based configuration files.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/viper"
	"github.com/takeoutfm/takeout"
	"github.com/takeoutfm/takeout/lib/bucket"
	"github.com/takeoutfm/takeout/lib/client"
	"github.com/takeoutfm/takeout/lib/fanart"
	g "github.com/takeoutfm/takeout/lib/gorm"
	"github.com/takeoutfm/takeout/lib/lastfm"
	"github.com/takeoutfm/takeout/lib/log"
	"github.com/takeoutfm/takeout/lib/search"
	"github.com/takeoutfm/takeout/lib/tmdb"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrTestConfig   = errors.New("missing test config")
	ErrInvalidCache = errors.New("invalid cache entry")
)

const (
	MediaMusic = "music"
	MediaVideo = "video"
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
}

type VideoConfig struct {
	DB                   DatabaseConfig
	ReleaseCountries     []string
	CastLimit            int
	CrewJobs             []string
	Recent               time.Duration
	RecentLimit          int
	SearchLimit          int
	Recommend            RecommendConfig
	SyncInterval         time.Duration
	PosterSyncInterval   time.Duration
	BackdropSyncInterval time.Duration
}

type PodcastConfig struct {
	DB           DatabaseConfig
	Series       []string
	Client       client.Config
	RecentLimit  int
	EpisodeLimit int
	SyncInterval time.Duration
	SearchLimit  int
}

type ProgressConfig struct {
	DB DatabaseConfig
}

type ActivityConfig struct {
	DB                 DatabaseConfig
	ActivityLimit      int
	RecentLimit        int
	PopularLimit       int
	RecentMoviesTitle  string
	RecentTracksTitle  string
	PopularMoviesTitle string
	PopularTracksTitle string
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
	tmdb.Config  `mapstructure:",squash"`
	FileTemplate Template
}

type SetlistAPIConfig struct {
	ApiKey string
}

type TokenConfig struct {
	Issuer     string
	Age        time.Duration
	Secret     string
	SecretFile string
}

type AuthConfig struct {
	DB            DatabaseConfig
	SessionAge    time.Duration
	CodeAge       time.Duration
	SecureCookies bool
	AccessToken   TokenConfig
	MediaToken    TokenConfig
	CodeToken     TokenConfig
}

type ServerConfig struct {
	Listen      string
	DataDir     string
	MediaDir    string
	ImageClient client.Config
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
	Video     VideoConfig
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

func (mc *MusicConfig) UserArtistID(name string) (string, bool) {
	mbid, ok := mc.artistMap[name]
	return mbid, ok
}

func readJsonStringMap(file string, m *map[string]string) (err error) {
	data, err := ioutil.ReadFile(file)
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
	v.SetDefault("Server.DataDir", ".")
	v.SetDefault("Server.MediaDir", ".")
	v.SetDefault("Server.ImageClient.CacheDir", "imagecache")
	v.SetDefault("Server.ImageClient.UserAgent", userAgent())

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

	v.SetDefault("Progress.DB.Driver", "sqlite3")
	v.SetDefault("Progress.DB.Source", "${Server.DataDir}/progress.db")
	v.SetDefault("Progress.DB.Logger", "default")

	v.SetDefault("Activity.DB.Driver", "sqlite3")
	v.SetDefault("Activity.DB.Source", "${Server.DataDir}/activity.db")
	v.SetDefault("Activity.DB.Logger", "default")
	v.SetDefault("Activity.ActivityLimit", "50")
	v.SetDefault("Activity.RecentLimit", "50")
	v.SetDefault("Activity.PopularLimit", "50")
	v.SetDefault("Activity.RecentMoviesTitle", "Recently Watched")
	v.SetDefault("Activity.RecentTracksTitle", "Recently Played")
	v.SetDefault("Activity.PopularMoviesTitle", "Popular Tracks")
	v.SetDefault("Activity.PopularTracksTitle", "Popular Tracks")

	// TODO apply as default
	// v.SetDefault("Bucket.URLExpiration", "15m")
	// v.SetDefault("Bucket.UseSSL", "true")

	v.SetDefault("Client.CacheDir", ".httpcache")
	v.SetDefault("Client.MaxAge", "720h") // 30 days in hours
	v.SetDefault("Client.UserAgent", userAgent())

	v.SetDefault("Fanart.ProjectKey", "93ede276ba6208318031727060b697c8")

	v.SetDefault("LastFM.Key", "")
	v.SetDefault("LastFM.Secret", "")

	v.SetDefault("Music.ArtistRadioBreadth", "10")
	v.SetDefault("Music.ArtistRadioDepth", "3")
	v.SetDefault("Music.DeepLimit", "50")
	v.SetDefault("Music.PopularLimit", "50")
	v.SetDefault("Music.RadioLimit", "25")
	v.SetDefault("Music.RadioSearchLimit", "1000")

	radioStreams := []RadioStream{{
		Creator:     "Ted Leibowitz",
		Title:       "BAGeL Radio",
		Image:       "https://cdn-profiles.tunein.com/s187420/images/logod.jpg",
		Description: "",
		Source: []ContentDescription{
			{ContentType: "audio/mpeg", URL: "https://www.bagelradio.com/s/bagelradio.pls"},
			{ContentType: "audio/aac", URL: "http://ais-sa3.cdnstream1.com/2606_128.mp3"},
		}}}
	radioStreams = append(radioStreams, somafmStreams...)
	v.SetDefault("Music.RadioStreams", radioStreams)

	v.SetDefault("Music.Recent", "8760h") // 1 year
	v.SetDefault("Music.RecentLimit", "50")
	v.SetDefault("Music.SearchLimit", "100")
	v.SetDefault("Music.SimilarArtistsLimit", "10")
	v.SetDefault("Music.SimilarReleases", "8760h") // +/- 1 year
	v.SetDefault("Music.SimilarReleasesLimit", "10")
	v.SetDefault("Music.SinglesLimit", "50")
	v.SetDefault("Music.SyncInterval", "1h")
	v.SetDefault("Music.PopularSyncInterval", "24h")
	v.SetDefault("Music.SimilarSyncInterval", "24h")
	v.SetDefault("Music.CoverSyncInterval", "24h")

	// see https://wiki.musicbrainz.org/Release_Country
	// v.SetDefault("Music.ReleaseCountries", []string{
	// 	"US", // United States
	// 	"XW", // Worldwide
	// 	"XE", // Europe
	// })

	v.SetDefault("Music.DB.Driver", "sqlite3")
	v.SetDefault("Music.DB.Source", "music.db")
	v.SetDefault("Music.DB.Logger", "default")

	v.SetDefault("TMDB.Key", "903a776b0638da68e9ade38ff538e1d3")
	v.SetDefault("TMDB.Language", "en-US")
	v.SetDefault("TMDB.FileTemplate.Text",
		"{{.Title}} ({{.Year}}){{if .Definition}} - {{.Definition}}{{end}}{{.Extension}}")

	v.SetDefault("Search.BleveDir", ".")

	v.SetDefault("Video.DB.Driver", "sqlite3")
	v.SetDefault("Video.DB.Source", "video.db")
	v.SetDefault("Video.DB.Logger", "default")
	v.SetDefault("Video.ReleaseCountries", []string{
		"US",
	})
	v.SetDefault("Video.CastLimit", "25")
	v.SetDefault("Video.CrewJobs", []string{
		"Director",
		"Executive Producer",
		"Novel",
		"Producer",
		"Screenplay",
		"Story",
	})
	v.SetDefault("Video.Recent", "8760h") // 1 year
	v.SetDefault("Video.RecentLimit", "50")
	v.SetDefault("Video.SearchLimit", "100")
	v.SetDefault("Video.SyncInterval", "1h")
	v.SetDefault("Video.PosterSyncInterval", "24h")
	v.SetDefault("Video.BackdropSyncInterval", "24h")
	v.SetDefault("Video.Recommend.When", []DateRecommend{
		// day of week + day of month
		{Match: "Fri 13", Layout: "Mon 02", Name: "Friday 13th Movies", Query: `+character:voorhees`},
		// day of month
		{Match: "Jan 03", Layout: "Jan 02", Name: "Tolkien Movies", Query: `+writing:tolkien`},
		{Match: "Feb 02", Layout: "Jan 02", Name: "Groundhog Day Movies", Query: `+keyword:groundhog`},
		{Match: "Feb 14", Layout: "Jan 02", Name: "Valentine's Day Movies", Query: `+genre:Romance`},
		{Match: "Mar 02", Layout: "Jan 02", Name: "Dr. Seuss Movies", Query: `+writing:seuss`},
		{Match: "Mar 12", Layout: "Jan 02", Name: "Hitchcock Movies", Query: `+directing:hitchcock`},
		{Match: "Mar 17", Layout: "Jan 02", Name: "St. Patrick's Day Movies", Query: `+keyword:leprechaun`},
		{Match: "Mar 27", Layout: "Jan 02", Name: "Tarantino Movies", Query: `+directing:tarantino`},
		{Match: "Apr 01", Layout: "Jan 02", Name: "April Fool's Movies", Query: `+keyword:"april fool's day"`},
		{Match: "Apr 28", Layout: "Jan 02", Name: "Superhero Movies", Query: `+keyword:superhero`},
		{Match: "May 02", Layout: "Jan 02", Name: "Harry Potter Movies", Query: `+title:"harry potter"`},
		{Match: "May 04", Layout: "Jan 02", Name: "Star Wars Movies", Query: `+title:"star wars"`},
		{Match: "May 11", Layout: "Jan 02", Name: "Twilight Zone Movies", Query: `+title:"twilight zone"`},
		{Match: "Jul 04", Layout: "Jan 02", Name: "July 4th Movies", Query: `keyword:patriotism keyword:patriotic keyword:independence`},
		{Match: "Jul 04", Layout: "Jan 02", Name: "Alice in Wonderland",
			Query: `character:"Alice Kingsleigh" character:"Mad Hatter" character:"Red Queen"`},
		{Match: "Aug 11", Layout: "Jan 02", Name: "Spider-man Movies", Query: `+title:"spider-man"`},
		{Match: "Sep 17", Layout: "Jan 02", Name: "Batman Movies", Query: `+character:batman`},
		{Match: "Sep 22", Layout: "Jan 02", Name: "Hobbit Movies", Query: `+keyword:hobbit`},
		{Match: "Oct 21", Layout: "Jan 02", Name: "Back to the Future Movies", Query: `+title:"back to the future"`},
		{Match: "Dec 23", Layout: "Jan 02", Name: "It's Festivus", Query: `+keyword:festivus`},
		// months
		{Match: "Oct", Layout: "Jan", Name: "Halloween Movies", Query: `+keyword:halloween`},
		{Match: "Dec", Layout: "Jan", Name: "Christmas Movies", Query: `+keyword:christmas +keyword:holiday`},
	})

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
	v.SetDefault("Podcast.SearchLimit", "100")
	v.SetDefault("Podcast.SyncInterval", "1h")
	v.SetDefault("Podcast.Series", []string{
		"https://feeds.twit.tv/twit.xml",
		"https://feeds.twit.tv/sn.xml",
		"https://feeds.twit.tv/twig.xml",
		"https://feeds.twit.tv/floss.xml",
		"https://www.pbs.org/newshour/feeds/rss/podcasts/show",
		"http://feeds.feedburner.com/TEDTalks_audio",
		"https://feeds.eff.org/howtofixtheinternet",
		"https://feeds.npr.org/510019/podcast.xml", // all songs considered
		"https://rss.art19.com/rotten-tomatoes-is-wrong",
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
	return postProcessConfig(v, rootDir)
}

func postProcessConfig(v *viper.Viper, rootDir string) (*Config, error) {
	var config Config
	var pathRegexp = regexp.MustCompile(`(file|dir|source)$`)
	for _, k := range v.AllKeys() {
		val := v.Get(k)
		if _, ok := val.(string); ok {
			// expand $var or ${var} on any string values
			sval := val.(string)
			if strings.Contains(sval, "$") {
				v.Set(k, os.Expand(sval, func(s string) string {
					r := v.Get(s)
					if r == nil {
						log.Panicf("'%s' not found for %s\n", s, sval)
					}
					if _, ok := r.(string); !ok {
						log.Panicf("'%s' not a string for %s\n", s, sval)
					}
					return r.(string)
				}))
			}
		}
		if pathRegexp.MatchString(k) {
			val := v.Get(k)
			// resolve relative paths only
			if strings.HasPrefix(val.(string), "/") == false &&
				strings.Contains(val.(string), "@") == false &&
				strings.Contains(val.(string), "::") == false {
				val = fmt.Sprintf("%s/%s", rootDir, val.(string))
				v.Set(k, val)
			}
		}
	}
	err := v.Unmarshal(&config)
	config.Music.readMaps()
	return &config, err
}

func TestConfig() (*Config, error) {
	testDir := os.Getenv("TEST_CONFIG")
	if testDir == "" {
		return nil, ErrTestConfig
	}
	v := viper.New()
	configDefaults(v)
	v.SetConfigFile(filepath.Join(testDir, "test.yaml"))
	v.SetDefault("Music.DB.Source", filepath.Join(testDir, "music.db"))
	v.SetDefault("Auth.DB.Source", filepath.Join(testDir, "auth.db"))
	return readConfig(v)
}

func TestingConfig() (*Config, error) {
	v := viper.New()
	configDefaults(v)
	v.SetConfigFile("testing.yaml")

	memory := "file::memory:?cache=shared"
	v.SetDefault("Activity.DB.Source", memory)
	v.SetDefault("Auth.DB.Source", "${Activity.DB.Source}")
	v.SetDefault("Music.DB.Source", "${Activity.DB.Source}")

	v.SetDefault("Auth.AccessToken.Issuer", "takeout.test")
	v.SetDefault("Auth.AccessToken.Age", "5m")
	v.SetDefault("Auth.AccessToken.Secret", "Wtex5hJ3vxZbkCSs")
	v.SetDefault("Auth.MediaToken.Issuer", "takeout.test")
	v.SetDefault("Auth.MediaToken.Age", "5m")
	v.SetDefault("Auth.MediaToken.Secret", "H1ys/pP/iNiQUl4k")
	v.SetDefault("Auth.CodeToken.Issuer", "takeout.test")
	v.SetDefault("Auth.CodeToken.Age", "5m")
	v.SetDefault("Auth.CodeToken.Secret", "Rg3ac20IPqyL7oAC")
	return postProcessConfig(v, "/tmp")
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

// GetConfig uses viper loads the default configuration.
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

// LoadConfig uses viper to load a config file in the provided directory.  The
// result is returned as a Config and cached.
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
		// video, podcast) would need to be reconstructed and not sure
		// if that's desired.
	}
	return c, err
}

func gormLogger(name string) logger.Interface {
	return g.Logger(name)
}
