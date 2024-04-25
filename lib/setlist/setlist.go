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

package setlist

import (
	"fmt"
	"time"

	"github.com/takeoutfm/takeout/lib/client"
)

type Config struct {
	ApiKey string
}

type Client struct {
	config Config
	client client.Getter
}

func NewClient(config Config, client client.Getter) *Client {
	return &Client{
		config: config,
		client: client,
	}
}

type Artist struct {
	Mbid           string `json:"mbid"`
	Tmid           int    `json:"tmid"`
	Name           string `json:"name"`
	SorName        string `json:"sortName"`
	Disambiguation string `json:"disambiguation"`
	Url            string `json:"url"`
}

type Country struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type City struct {
	Id        string  `json:"id"`
	Name      string  `json:"name"`
	State     string  `json:"state"`
	StateCode string  `json:"stateCode"`
	Country   Country `json:"country"`
}

type Venue struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
	City City   `json:"city"`
}

type Tour struct {
	Name string `json:"name"`
}

type Song struct {
	Name string `json:"name"`
	Info string `json:"info"`
	Tape bool   `json:"tape"`
}

type Set struct {
	Encore int    `json:"encore"`
	Name   string `json:"name"`
	Songs  []Song `json:"song"`
}

type Sets struct {
	Set []Set `json:"set"`
}

type Setlist struct {
	Id          string `json:"id"`
	VersionId   string `json:"versionId"`
	EventDate   string `json:"eventDate"`
	LastUpdated string `json:"lastUpdated"`
	Artist      Artist `json:"artist"`
	Venue       Venue  `json:"venue"`
	Tour        Tour   `json:"tour"`
	Sets        Sets   `json:"sets"`
	Info        string `json:"info"`
	Url         string `json:"url"`
}

func (s Setlist) EventTime() time.Time {
	t, err := time.Parse("2-1-2006", s.EventDate)
	if err != nil {
		return time.Time{}
	}
	return t
}

type SetlistPage struct {
	Type         string    `json:"type"`
	ItemsPerPage int       `json:"itemsPerPage"`
	Page         int       `json:"page"`
	Total        int       `json:"total"`
	Setlist      []Setlist `json:"setlist"`
}

func (s *Client) ArtistYear(arid string, year int) []Setlist {
	format := fmt.Sprintf("https://api.setlist.fm/rest/1.0/search/setlists?artistMbid=%s&year=%d",
		arid, year)
	return s.fetchAll(format + "&p=%d")
}

func (s *Client) fetchAll(format string) []Setlist {
	var list []Setlist
	total := 0

	for page := 1; ; page++ {
		url := fmt.Sprintf(format, page)
		result := s.fetchPage(url)
		list = append(list, result.Setlist...)
		total += len(result.Setlist)
		if total >= result.Total {
			break
		}
	}
	return list
}

func (s *Client) fetchPage(url string) *SetlistPage {
	time.Sleep(time.Second)

	headers := map[string]string{
		"Accept":    "application/json",
		"x-api-key": s.config.ApiKey,
	}
	var result SetlistPage
	err := s.client.GetJsonWith(headers, url, &result)
	if err != nil {
		fmt.Println(err)
	}
	return &result
}
