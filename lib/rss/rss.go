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

// Package rss provide support for parsing RSS files as used in podcast feeds.
package rss // import "takeoutfm.dev/takeout/lib/rss"

import (
	"encoding/xml"
	"html"
	"strings"
	"time"

	"takeoutfm.dev/takeout/lib/client"
	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/lib/hash"
)

type RSS struct {
	client client.Getter
}

func NewRSS(client client.Getter) *RSS {
	return &RSS{
		client: client,
	}
}

func (rss *RSS) Fetch(url string) (Channel, error) {
	var result Rss
	err := rss.client.GetXML(url, &result)
	return result.Channel, err
}

func unescape(s string) string {
	return html.UnescapeString(s)
}

func (rss *RSS) FetchPodcast(url string) (Podcast, error) {
	result, err := rss.Fetch(url)
	podcast := Podcast{
		Title:         unescape(result.Title),
		Description:   result.Description,
		Author:        unescape(result.Author),
		Link:          result.Link(),
		Image:         result.Image.Link,
		Copyright:     result.Copyright,
		LastBuildTime: result.LastBuildTime(),
		TTL:           result.TTL,
	}
	for _, i := range result.Items {
		episode := Episode{
			Title:       unescape(i.ItemTitle()),
			Author:      unescape(i.Author),
			Link:        i.Link,
			Description: i.Description,
			ContentType: i.ContentType(),
			Size:        i.Size(),
			URL:         i.URL(),
			PublishTime: i.PublishTime(),
			GUID:        i.ItemGUID(),
			Image:       i.ItemImage(),
		}
		podcast.Episodes = append(podcast.Episodes, episode)
	}
	return podcast, err
}

type Podcast struct {
	Title         string
	Description   string
	Author        string
	Link          string
	Image         string
	Copyright     string
	LastBuildTime time.Time
	TTL           int
	Episodes      []Episode
}

type Episode struct {
	Title       string
	Link        string
	Author      string
	Description string
	ContentType string
	Size        int64
	URL         string
	PublishTime time.Time
	GUID        string
	Image       string
}

type Image struct {
	Title     string `xml:"title"`
	URL       string `xml:"url"`
	Link      string `xml:"link"`
	Width     int    `xml:"width"`
	Height    int    `xml:"height"`
	Reference string `xml:"href,attr"`
}

type Content struct {
	Title     string   `xml:"title,media"`
	Thumbnail string   `xml:"thumbnail,media"`
	URL       string   `xml:"url,attr"`
	FileSize  int64    `xml:"fileSize,attr"`
	Type      string   `xml:"type,attr"`
	Medium    string   `xml:"medium,attr"`
	Credits   []string `xml:"credit,media"`
}

type Enclosure struct {
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
	URL    string `xml:"url,attr"`
}

type GUID struct {
	PermaLink bool   `xml:"isPermaLink,attr"`
	Value     string `xml:",chardata"`
}

type Item struct {
	XMLName     xml.Name  `xml:"item"`
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	PubDate     string    `xml:"pubDate"`
	Description string    `xml:"description"`
	Categories  []string  `xml:"category"`
	GUID        GUID      `xml:"guid"`
	Author      string    `xml:"author,itunes"`
	Episode     string    `xml:"episode,itunes"`
	Image       Image     `xml:"image,itunes"`
	Duration    string    `xml:"duration,itunes"`
	Content     Content   `xml:"content,media"`
	Enclosure   Enclosure `xml:"enclosure"`
	Comments    string    `xml:"comments"`
}

func (i *Item) ItemTitle() string {
	if i.Content.Title != "" {
		return i.Content.Title
	}
	return i.Title
}

func (i *Item) PublishTime() time.Time {
	return date.ParseRFC1123(i.PubDate)
}

func (i *Item) Size() int64 {
	if i.Content.FileSize > 0 {
		return i.Content.FileSize
	}
	return i.Enclosure.Length
}

func (i *Item) ContentType() string {
	if i.Content.Title != "" {
		return i.Content.Type
	}
	return i.Enclosure.Type
}

func (i *Item) URL() string {
	if i.Content.URL != "" {
		return i.Content.URL
	}
	return i.Enclosure.URL
}

func (i *Item) ItemGUID() string {
	guid := i.GUID.Value
	if guid == "" {
		guid = hash.MD5Hex(i.Link)
	} else if i.GUID.PermaLink || strings.Contains(guid, "://") {
		guid = hash.MD5Hex(guid)
	}
	return guid
}

func (i *Item) ItemImage() string {
	return i.Image.Reference
}

type Channel struct {
	XMLName       xml.Name `xml:"channel"`
	Title         string   `xml:"title"`
	Links         []string `xml:"link"` // want <link> not <atom:link>
	Description   string   `xml:"description"`
	Author        string   `xml:"author,itunes"`
	Language      string   `xml:"language"`
	Copyright     string   `xml:"copyright"`
	LastBuildDate string   `xml:"lastBuildDate"`
	Image         Image    `xml:"image"`
	TTL           int      `xml:"ttl"`
	Items         []Item   `xml:"item"`
}

type Rss struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

func (c *Channel) LastBuildTime() time.Time {
	return date.ParseRFC1123(c.LastBuildDate)
}

func (c *Channel) Link() string {
	// <atom:link> has no value just href attr
	for _, l := range c.Links {
		if l != "" {
			return l
		}
	}
	return ""
}
