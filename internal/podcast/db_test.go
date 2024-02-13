// Copyright 2024 defsub
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

package podcast

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/model"
)

func makePodcast(t *testing.T) *Podcast {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	p := NewPodcast(config)
	err = p.Open()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSeries(t *testing.T) {
	p := makePodcast(t)

	sid := "51973e22a70f1c75fb94404aef50c9ac"
	s := model.Series{
		SID:         sid,
		Title:       "this week in testing",
		Description: "this week in testing description",
		Author:      "Mr. Podcast",
		Link:        "https://podcast.com/foo.html",
		Image:       "https://podcast.com/foo.png",
		Copyright:   "Mr. Podcast, LLC",
		Date:        time.Now(),
		TTL:         999,
	}

	err := p.createSeries(&s)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID == 0 {
		t.Error("expect ID")
	}

	// TODO why findSeries and LookupSID?
	if p.findSeries(sid) == nil {
		t.Error("expect series")
	}
	_, err = p.LookupSID(sid)
	if err != nil {
		t.Error("expect series by SID")
	}
	_, err = p.LookupSeries(int(s.ID))
	if err != nil {
		t.Error("expect series by ID")
	}

	if len(p.Series()) != 1 {
		t.Error("expect series list")
	}
	if p.SeriesCount() != 1 {
		t.Error("expect series count")
	}

	if len(p.seriesFor([]string{sid})) != 1 {
		t.Error("expect seriesFor")
	}

	p.deleteSeries(sid)

	if p.SeriesCount() != 0 {
		t.Error("expect no series")
	}
}

func TestEpisode(t *testing.T) {
	p := makePodcast(t)

	sid := "51973e22a70f1c75fb94404aef50c9ac"
	eid := "8a868b69-abd7-4704-8a56-576c99bd0998"
	e := model.Episode{
		SID:         sid,
		EID:         eid,
		Title:       "this week in testing episode",
		Description: "this week in testing episode description",
		Author:      "Mr. Podcast",
		Link:        "https://podcast.com/foo.html",
		Image:       "https://podcast.com/foo.png",
		Date:        time.Now(),
		ContentType: "audio/mpeg",
		Size:        99999999,
		URL:         "https://podcast.com/foo.mp3",
	}

	err := p.createEpisode(&e)
	if err != nil {
		t.Fatal(err)
	}
	if e.ID == 0 {
		t.Error("expect ID")
	}

	if p.findEpisode(eid) == nil {
		t.Error("expect to find episode")
	}
	_, err = p.LookupEpisode(int(e.ID))
	if err != nil {
		t.Error("expect to lookup episode")
	}
	_, err = p.LookupEID(eid)
	if err != nil {
		t.Error("expect to lookup eid")
	}

	s := model.Series{SID: sid}
	if len(p.Episodes(s)) != 1 {
		t.Error("expect 1 episode")
	}

	if len(p.episodesFor([]string{eid})) != 1 {
		t.Error("expect episodesfor")
	}

	p.retainEpisodes(&s, []string{eid})
	if p.findEpisode(eid) == nil {
		t.Error("expect retained")
	}

	p.deleteEpisode(eid)

	if p.findEpisode(eid) != nil {
		t.Error("expect no episode")
	}
}

func TestRecentEpisodes(t *testing.T) {
	p := makePodcast(t)

	sid := "51973e22a70f1c75fb94404aef50c9ac"

	episodes := []model.Episode{
		{
			SID:   sid,
			EID:   "8a868b69-abd7-4704-8a56-576c99bd0998",
			Title: "this week in testing episode 3",
			Date: time.Now().Add(-time.Hour*48),
		},
		{
			SID:   sid,
			EID:   "953b6450-8c14-42f8-8c44-28dcab471df2",
			Title: "this week in testing episode 2",
			Date: time.Now().Add(-time.Hour*24),
		},
		{
			SID:   sid,
			EID:   "97596b00-ee8b-4240-903c-fbbecb99c7cd",
			Title: "this week in testing episode 1",
			Date: time.Now(),
		},
	}

	for _, e := range episodes {
		err := p.createEpisode(&e)
		if err != nil {
			t.Fatal(err)
		}
	}

	list := p.RecentEpisodes()
	if len(list) == 0 {
		t.Error("expect episodes")
	}
	if list[0].Title != "this week in testing episode 1" {
		t.Error("episode 1")
	}

	list = p.Episodes(model.Series{SID: sid})
	if len(list) == 0 {
		t.Error("expect episodes")
	}
	if list[0].Title != "this week in testing episode 1" {
		t.Error("episode 1")
	}

	p.deleteSeriesEpisodes(sid)
}

func TestSubscription(t *testing.T) {
	p := makePodcast(t)

	user := "takeout"
	sid := "51973e22a70f1c75fb94404aef50c9ac"
	s := model.Series{
		SID:         sid,
		Title:       "this week in testing",
		Description: "this week in testing description",
		Author:      "Mr. Podcast",
		Link:        "https://podcast.com/foo.html",
		Image:       "https://podcast.com/foo.png",
		Copyright:   "Mr. Podcast, LLC",
		Date:        time.Now(),
		TTL:         999,
	}

	err := p.createSeries(&s)
	if err != nil {
		t.Fatal(err)
	}

	if p.HasSubscriptions(user) == true {
		t.Error("expect no subscriptions")
	}

	err = p.Subscribe(sid, user)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.SubscriptionsFor(user)) != 1 {
		t.Error("expect subscription")
	}

	if p.HasSubscriptions(user) == false {
		t.Error("expect subscriptions")
	}

	p.Unsubscribe(sid, user)

	if p.HasSubscriptions(user) == true {
		t.Error("expect no subscriptions again")
	}
}
