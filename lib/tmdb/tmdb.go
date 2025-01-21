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

// Package tmdb provides a good coverage for the TMDB API, supporting movies,
// shows, people and images.
package tmdb

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/takeoutfm/takeout/lib/client"
)

const (
	// ISO 639-1
	LanguageEnglish = "en-US"
)

const (
	Backdrop300      = "w300"
	Backdrop780      = "w780"
	Backdrop1280     = "w1280"
	BackdropOriginal = "original"
)

const (
	Poster92       = "w92"
	Poster154      = "w154"
	Poster185      = "w185"
	Poster342      = "w342"
	Poster500      = "w500"
	Poster780      = "w780"
	PosterOriginal = "original"
)

const (
	Profile45       = "w45"
	Profile185      = "w185"
	Profile632      = "h632"
	ProfileOriginal = "original"
)

const (
	Still92       = "w92"
	Still185      = "w185"
	Still300      = "w300"
	StillOriginal = "original"
)

var (
	ErrReleaseTypeNotFound = errors.New("release type not found")
)

type Config struct {
	Key      string
	Language string
}

type TMDB struct {
	config      Config
	client      client.Getter
	apiConfig   apiConfig
	movieGenres Genres
	tvGenres    Genres
}

func NewTMDB(config Config, client client.Getter) *TMDB {
	return &TMDB{
		config: config,
		client: client,
	}
}

type Collection struct {
	ID           int    `json:"id"` // unique collection ID
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	BackdropPath string `json:"backdrop_path"`
	PosterPath   string `json:"poster_path"`
	// Parts []MovieResult
}

type Movie struct {
	ID               int        `json:"id"` // unique movie ID
	IMDB_ID          string     `json:"imdb_id"`
	Adult            bool       `json:"adult"`
	BackdropPath     string     `json:"backdrop_path"`
	Collection       Collection `json:"belongs_to_collection"`
	Genres           []Genre    `json:"genres"`
	OriginalLanguage string     `json:"original_language"`
	OriginalTitle    string     `json:"original_title"`
	Overview         string     `json:"overview"`
	Popularity       float32    `json:"populartity"`
	PosterPath       string     `json:"poster_path"`
	ReleaseDate      string     `json:"release_date"`
	Tagline          string     `json:"tagline"`
	Title            string     `json:"title"`
	Video            bool       `json:"video"`
	VoteAverage      float32    `json:"vote_average"`
	VoteCount        int        `json:"vote_count"`
	Budget           int64      `json:"budget"`
	Revenue          int64      `json:"revenue"`
	Runtime          int        `json:"runtime"`
}

type TV struct {
	ID               int       `json:"id"` // unique tv ID
	BackdropPath     string    `json:"backdrop_path"`
	PosterPath       string    `json:"poster_path"`
	Genres           []Genre   `json:"genres"`
	FirstAirDate     string    `json:"first_air_date"`
	LastAirDate      string    `json:"last_air_date"`
	Name             string    `json:"name"`
	OriginalName     string    `json:"original_name"`
	OriginalLanguage string    `json:"original_language"`
	Overview         string    `json:"overview"`
	Networks         []Network `json:"networks"`
	NumberOfEpisodes int       `json:"number_of_episodes"`
	NumberOfSeasons  int       `json:"number_of_seasons"`
	Popularity       float32   `json:"populartity"`
	Seasons          []Season  `json:"seasons"`
	Status           string    `json:"status"`
	Tagline          string    `json:"tagline"`
	Type             string    `json:"type"`
	VoteAverage      float32   `json:"vote_average"`
	VoteCount        int       `json:"vote_count"`
}

type Network struct {
	ID            int    `json:"id"` // unique network ID
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

type Season struct {
	ID           int    `json:"id"` // unique season ID
	AirDate      string `json:"air_date"`
	EpisodeCount int    `json:"episode_count"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
	SeasonNumber int    `json:"season_number"`
}

type Episode struct {
	ID            int     `json:"id"` // unique episode ID
	StillPath     string  `json:"still_path"`
	AirDate       string  `json:"air_date"`
	Name          string  `json:"name"`
	Overview      string  `json:"overview"`
	SeasonNumber  int     `json:"season_number"`
	EpisodeNumber int     `json:"episode_number"`
	VoteAverage   float32 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Runtime       int     `json:"runtime"`
}

type ContentRatings struct {
	ID      int             `json:"id"`
	Results []ContentRating `json:"results"`
}

type ContentRating struct {
	Country string `json:"iso_3166_1"`
	Rating  string `json:"rating"`
}

type Cast struct {
	ID           int    `json:"id"` // unique person ID
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	ProfilePath  string `json:"profile_path"`
	Character    string `json:"character"`
	Order        int    `json:"order"`
}

type Crew struct {
	ID           int    `json:"id"` // unique person ID
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	ProfilePath  string `json:"profile_path"`
	Department   string `json:"department"`
	Job          string `json:"job"`
}

type Credits struct {
	ID     int    `json:"id"` // unique movie or tv ID
	Cast   []Cast `json:"cast"`
	Crew   []Crew `json:"crew"`
	Guests []Cast `json:"guest_stars"` // only tv
}

func (c *Credits) SortedCast() []Cast {
	cast := make([]Cast, len(c.Cast))
	copy(cast, c.Cast)
	sort.Slice(cast, func(i, j int) bool {
		// sort by order
		return cast[i].Order < cast[j].Order
	})
	return cast
}

func (c *Credits) SortedGuests() []Cast {
	guests := make([]Cast, len(c.Cast))
	copy(guests, c.Guests)
	sort.Slice(guests, func(i, j int) bool {
		return guests[i].Order < guests[j].Order
	})
	return guests
}

func (c *Credits) CrewWithJobs(jobs []string) []Crew {
	var crew []Crew
	jobRegexp := regexp.MustCompile("^(" + strings.Join(jobs, "|") + ")$")
	for _, c := range c.Crew {
		matches := jobRegexp.FindStringSubmatch(c.Job)
		if matches != nil {
			crew = append(crew, c)
		}
	}
	return crew
}

type Person struct {
	ID          int    `json:"id"` // unique person ID
	IMDB_ID     string `json:"imdb_id"`
	Name        string `json:"name"`
	ProfilePath string `json:"profile_path"`
	Birthday    string `json:"birthday"`
	Deathday    string `json:"deathday"`
	Biography   string `json:"biography"`
	Birthplace  string `json:"place_of_birth"`
}

// https://developers.themoviedb.org/3/movies/get-movie-release-dates
const (
	TypePremiere = iota + 1
	TypeTheatricalLimited
	TypeTheatrical
	TypeDigital
	TypePhysical
	TypeTV
)

type Release struct {
	Certification string `json:"certification"`
	Date          string `json:"release_date"`
	Note          string `json:"note"`
	Type          int    `json:"type"`
}

type ReleaseCountry struct {
	CountryCode string    `json:"iso_3166_1"`
	Releases    []Release `json:"release_dates"`
}

type Releases struct {
	ID      int              `json:"id"`
	Results []ReleaseCountry `json:"results"`
}

type searchResult struct {
	ID               int     `json:"id"`
	BackdropPath     string  `json:"backdrop_path"`
	GenreIDs         []int   `json:"genre_ids"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	Popularity       float32 `json:"populartity"`
	PosterPath       string  `json:"poster_path"`
	VoteAverage      float32 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
}

type MovieResult struct {
	searchResult
	Adult         bool   `json:"adult"`
	OriginalTitle string `json:"original_title"`
	ReleaseDate   string `json:"release_date"`
	Title         string `json:"title"`
	Video         bool   `json:"video"`
}

type TVResult struct {
	searchResult
	FirstAirDate string `json:"first_air_date"`
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
}

type page struct {
	Page         int `json:"page"`
	TotalPages   int `json:"total_pages"`
	TotalResults int `json:"total_results"`
}

type moviePage struct {
	page
	Results []MovieResult `json:"results"`
}

type tvPage struct {
	page
	Results []TVResult `json:"results"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type genreList struct {
	Genres []Genre `json:"genres"`
}

type Genres map[int]string

type Keywords struct {
	ID       int       `json:"id"`
	Keywords []Keyword `json:"keywords"`
}

type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Videos struct {
	ID      int     `json:"id"`
	Results []Video `json:"results"`
}

const (
	TypeTrailer = "Trailer"

	SiteYouTube = "YouTube"
)

type Video struct {
	ID          string `json:"id"`
	Language    string `json:"iso_639_1"`
	Country     string `json:"iso_3166_1"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Site        string `json:"site"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Official    bool   `json:"official"`
	PublishDate string `json:"published_at"`
}

func (v Video) YouTube() bool {
	return v.Site == SiteYouTube
}

// https://www.youtube.com/watch?v={key}
func (v Video) YouTubeLink() string {
	return strings.Join([]string{"https://www.youtube.com/watch?v=", v.Key}, "")
}

func (v Video) Trailer() bool {
	return v.Type == TypeTrailer
}

type imagesConfig struct {
	BaseURL       string   `json:"base_url"`
	SecureBaseURL string   `json:"secure_base_url"`
	BackdropSizes []string `json:"backdrop_sizes"`
	LogoSizes     []string `json:"logo_sizes"`
	PosterSizes   []string `json:"poster_sizes"`
	ProfileSizes  []string `json:"profile_sizes"`
	StillSizes    []string `json:"still_sizes"`
}

type apiConfig struct {
	Images    imagesConfig `json:"images"`
	ChangeKey []string     `json:"change_keys"`
}

func (c *apiConfig) empty() bool {
	return c.Images.SecureBaseURL == ""
}

const (
	endpoint = "api.themoviedb.org"
)

func (m *TMDB) moviePage(q string, page int) (moviePage, error) {
	url := fmt.Sprintf(
		"https://%s/3/search/movie?api_key=%s&language=%s&query=%s&page=%d",
		endpoint,
		m.config.Key,
		m.config.Language,
		url.QueryEscape(q), page)
	var result moviePage
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) MovieSearch(q string) ([]MovieResult, error) {
	// TODO only supports one page right now
	page, err := m.moviePage(q, 1)
	return page.Results, err
}

func (m *TMDB) MovieDetail(tmid int) (Movie, error) {
	url := fmt.Sprintf(
		"https://%s/3/movie/%d?api_key=%s&language=%s",
		endpoint, tmid,
		m.config.Key,
		m.config.Language)
	var result Movie
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) MovieCredits(tmid int) (Credits, error) {
	url := fmt.Sprintf(
		"https://%s/3/movie/%d/credits?api_key=%s&language=%s",
		endpoint, tmid,
		m.config.Key,
		m.config.Language)
	var result Credits
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) MovieReleases(tmid int) (map[string][]Release, error) {
	url := fmt.Sprintf(
		"https://%s/3/movie/%d/release_dates?api_key=%s",
		endpoint, tmid, m.config.Key)
	var result Releases
	var countryMap map[string][]Release
	err := m.client.GetJson(url, &result)
	if err == nil {
		countryMap = make(map[string][]Release)
		for _, rc := range result.Results {
			countryMap[rc.CountryCode] = rc.Releases
		}
	}
	return countryMap, err
}

func (m *TMDB) MovieReleaseType(tmid int, country string, releaseType int) (Release, error) {
	url := fmt.Sprintf(
		"https://%s/3/movie/%d/release_dates?api_key=%s&language=%s",
		endpoint, tmid,
		m.config.Key,
		m.config.Language)
	var result Releases
	err := m.client.GetJson(url, &result)
	if err != nil {
		return Release{}, err
	}

	for _, rc := range result.Results {
		if rc.CountryCode == country {
			for _, r := range rc.Releases {
				if r.Type == releaseType {
					return r, nil
				}
			}
		}
	}
	return Release{}, ErrReleaseTypeNotFound
}

func (m *TMDB) MovieVideos(tmid int) (Videos, error) {
	url := fmt.Sprintf(
		"https://%s/3/movie/%d/videos?api_key=%s&language=%s",
		endpoint, tmid,
		m.config.Key,
		m.config.Language)
	var result Videos
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) PersonDetail(peid int) (Person, error) {
	url := fmt.Sprintf(
		"https://%s/3/person/%d?api_key=%s&language=%s",
		endpoint, peid,
		m.config.Key,
		m.config.Language)
	var result Person
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) movieGenreList() (genreList, error) {
	url := fmt.Sprintf(
		"https://%s/3/genre/movie/list?api_key=%s&language=%s",
		endpoint,
		m.config.Key,
		m.config.Language)
	var result genreList
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) tvGenreList() (genreList, error) {
	url := fmt.Sprintf(
		"https://%s/3/genre/tv/list?api_key=%s&language=%s",
		endpoint,
		m.config.Key,
		m.config.Language)
	var result genreList
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) populateGenreCache() error {
	if m.movieGenres != nil || m.tvGenres != nil {
		return nil
	}

	movieList, err := m.movieGenreList()
	if err != nil {
		return err
	}
	tvList, err := m.tvGenreList()
	if err != nil {
		return err
	}

	m.movieGenres = make(Genres)
	if err == nil {
		for _, g := range movieList.Genres {
			m.movieGenres[g.ID] = g.Name
		}
	}
	m.tvGenres = make(Genres)
	if err == nil {
		for _, g := range tvList.Genres {
			m.tvGenres[g.ID] = g.Name
		}
	}
	return nil
}

func (m *TMDB) MovieGenre(id int) string {
	m.populateGenreCache()
	return m.movieGenres[id]
}

func (m *TMDB) MovieGenreNames() []string {
	m.populateGenreCache()
	var result []string
	for _, v := range m.movieGenres {
		result = append(result, v)
	}
	return result
}

func (m *TMDB) TVGenre(id int) string {
	m.populateGenreCache()
	return m.tvGenres[id]
}

func (m *TMDB) TVGenreNames() []string {
	m.populateGenreCache()
	var result []string
	for _, v := range m.tvGenres {
		result = append(result, v)
	}
	return result
}

func (m *TMDB) MovieKeywordNames(id int) ([]string, error) {
	url := fmt.Sprintf(
		"https://%s/3/movie/%d/keywords?api_key=%s", endpoint, id, m.config.Key)
	var result Keywords
	err := m.client.GetJson(url, &result)
	var names []string
	for _, v := range result.Keywords {
		names = append(names, v.Name)
	}
	return names, err

}

func (m *TMDB) TVKeywordNames(tvid int) ([]string, error) {
	url := fmt.Sprintf(
		"https://%s/3/tv/%d/keywords?api_key=%s", endpoint, tvid, m.config.Key)
	var result Keywords
	err := m.client.GetJson(url, &result)
	var names []string
	for _, v := range result.Keywords {
		names = append(names, v.Name)
	}
	return names, err
}

func (m *TMDB) tvPage(q string, page int) (tvPage, error) {
	url := fmt.Sprintf(
		"https://%s/3/search/tv?api_key=%s&language=%s&query=%s&page=%d",
		endpoint,
		m.config.Key,
		m.config.Language,
		url.QueryEscape(q), page)
	var result tvPage
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) TVSearch(q string) ([]TVResult, error) {
	// TODO only supports one page right now
	page, err := m.tvPage(q, 1)
	return page.Results, err
}

func (m *TMDB) TVDetail(tvid int) (TV, error) {
	url := fmt.Sprintf(
		"https://%s/3/tv/%d?api_key=%s&language=%s",
		endpoint, tvid,
		m.config.Key,
		m.config.Language)
	var result TV
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) EpisodeDetail(tvid, season, episode int) (Episode, error) {
	url := fmt.Sprintf(
		"https://%s/3/tv/%d/season/%d/episode/%d?api_key=%s&language=%s",
		endpoint, tvid, season, episode,
		m.config.Key,
		m.config.Language)
	var result Episode
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) SeriesCredits(tvid int) (Credits, error) {
	url := fmt.Sprintf(
		"https://%s/3/tv/%d/credits?api_key=%s&language=%s",
		endpoint, tvid,
		m.config.Key,
		m.config.Language)
	var result Credits
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) EpisodeCredits(tvid, season, episode int) (Credits, error) {
	url := fmt.Sprintf(
		"https://%s/3/tv/%d/season/%d/episode/%d/credits?api_key=%s&language=%s",
		endpoint, tvid, season, episode,
		m.config.Key,
		m.config.Language)
	var result Credits
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) TVContentRatings(tvid int) (ContentRatings, error) {
	url := fmt.Sprintf(
		"https://%s/3/tv/%d/content_ratings?api_key=%s&language=%s",
		endpoint, tvid,
		m.config.Key,
		m.config.Language)
	var result ContentRatings
	err := m.client.GetJson(url, &result)
	return result, err
}

func (m *TMDB) configuration() (apiConfig, error) {
	url := fmt.Sprintf(
		"https://%s/3/configuration?api_key=%s", endpoint, m.config.Key)
	var result apiConfig
	err := m.client.GetJson(url, &result)
	return result, err
}

/*
   https://developers.themoviedb.org/3/configuration/get-api-configuration

   To build an image URL, you will need 3 pieces of data. The base_url, size and
   file_path. Simply combine them all and you will have a fully qualified
   URL. Here’s an example URL:

   https://image.tmdb.org/t/p/w500/8uO0gUM8aNqYLs1OsTBQiXu0fEv.jpg
*/

func poster(c apiConfig, size string, posterPath string) string {
	url := fmt.Sprintf("%s%s%s", c.Images.SecureBaseURL, size, posterPath)
	return url
}

func backdrop(c apiConfig, size string, backdropPath string) string {
	url := fmt.Sprintf("%s%s%s", c.Images.SecureBaseURL, size, backdropPath)
	return url
}

func still(c apiConfig, size string, stillPath string) string {
	url := fmt.Sprintf("%s%s%s", c.Images.SecureBaseURL, size, stillPath)
	return url
}

func profile(c apiConfig, size string, profilePath string) string {
	url := fmt.Sprintf("%s%s%s", c.Images.SecureBaseURL, size, profilePath)
	return url
}

func (m *TMDB) OriginalPoster(posterPath string) *url.URL {
	return m.Poster(posterPath, PosterOriginal)
}

func (m *TMDB) Poster(posterPath string, size string) *url.URL {
	err := m.checkApiConfig()
	if err != nil {
		return nil
	}
	v := poster(m.apiConfig, size, posterPath)
	url, err := url.Parse(v)
	if err != nil {
		return nil
	}
	return url
}

func (m *TMDB) Backdrop(backdropPath string, size string) *url.URL {
	err := m.checkApiConfig()
	if err != nil {
		return nil
	}
	v := backdrop(m.apiConfig, size, backdropPath)
	url, err := url.Parse(v)
	if err != nil {
		return nil
	}
	return url
}

func (m *TMDB) Still(stillPath string, size string) *url.URL {
	err := m.checkApiConfig()
	if err != nil {
		return nil
	}
	v := still(m.apiConfig, size, stillPath)
	url, err := url.Parse(v)
	if err != nil {
		return nil
	}
	return url
}

func (m *TMDB) PersonProfile(profilePath string, size string) *url.URL {
	err := m.checkApiConfig()
	if err != nil {
		return nil
	}
	v := profile(m.apiConfig, size, profilePath)
	url, err := url.Parse(v)
	if err != nil {
		return nil
	}
	return url
}

func (m *TMDB) checkApiConfig() error {
	var err error
	if m.apiConfig.empty() {
		m.apiConfig, err = m.configuration()
	}
	return err
}
