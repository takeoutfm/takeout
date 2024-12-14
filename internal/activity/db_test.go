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

package activity

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/model"
)

func makeActivity(t *testing.T) *Activity {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	a := NewActivity(config)
	err = a.Open()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestMovieEvent(t *testing.T) {
	user := "takeout"
	tmid := "278"

	a := makeActivity(t)
	e := model.MovieEvent{
		User: user,
		Date: time.Now(),
		TMID: tmid,
	}
	err := a.createMovieEvent(&e)
	if err != nil {
		t.Fatal(err)
	}

	events := a.movieEvents(user)
	if len(events) == 0 {
		t.Error("expect events")
	}
	if events[0].TMID != tmid {
		t.Errorf("expect %s", tmid)
	}

	a.deleteMovieEvents(user)
	if len(a.movieEvents(user)) != 0 {
		t.Error("expect no events")
	}
}

func TestTrackEvent(t *testing.T) {
	user := "takeout"
	rid := "7b486d22-ade1-4d61-940b-334071aad0cf"
	rgid := "c5e5e8ad-dc89-319e-8b2d-b3ff5e59fcea"

	a := makeActivity(t)
	e := model.TrackEvent{
		User: user,
		Date: time.Now(),
		RID:  rid,
		RGID: rgid,
	}
	err := a.createTrackEvent(&e)
	if err != nil {
		t.Fatal(err)
	}

	events := a.trackEvents(user)
	if len(events) == 0 {
		t.Error("expect events")
	}
	if events[0].RID != rid {
		t.Errorf("expect %s", rid)
	}
	if events[0].RGID != rgid {
		t.Errorf("expect %s", rgid)
	}

	a.deleteTrackEvents(user)
	if len(a.trackEvents(user)) != 0 {
		t.Error("expect no events")
	}
}

func TestEpisodeEvent(t *testing.T) {
	user := "takeout"
	eid := "5c3b551b626a8e9fa04186b448f2d3ed"

	a := makeActivity(t)
	e := model.EpisodeEvent{
		User: user,
		Date: time.Now(),
		EID:  eid,
	}
	err := a.createEpisodeEvent(&e)
	if err != nil {
		t.Fatal(err)
	}

	events := a.episodeEvents(user)
	if len(events) == 0 {
		t.Error("expect events")
	}
	if events[0].EID != eid {
		t.Errorf("expect %s", eid)
	}

	a.deleteEpisodeEvents(user)
	if len(a.episodeEvents(user)) != 0 {
		t.Error("expect no events")
	}
}

func TestTopTrackEvents(t *testing.T) {
	user := "takeout"
	rid := "7b486d22-ade1-4d61-940b-334071aad0cf"
	rgid := "c5e5e8ad-dc89-319e-8b2d-b3ff5e59fcea"

	a := makeActivity(t)

	for i := 0; i < 2; i++ {
		e := model.TrackEvent{
			User: user,
			Date: time.Now(),
			RID:  rid,
			RGID: rgid,
		}
		err := a.createTrackEvent(&e)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s\n", e.Date.String())
	}

	end := time.Now()
	start := end.Add(time.Hour*-1)
	events := a.topTrackEventsFrom("takeout", start, end, 10)

	if len(events) != 1 {
		t.Error("expect 1 event")
	}

	if events[0].Count != 2 {
		t.Error("expect count is 2")
	}

	for _, e := range events {
		t.Logf("%d - %s\n", e.Count, e.TrackEvent.Date.String())
	}

	a.deleteTrackEvents(user)
	if len(a.trackEvents(user)) != 0 {
		t.Error("expect no events")
	}
}

func TestTrackEventsFrom(t *testing.T) {
	user := "takeout"
	rid := "7b486d22-ade1-4d61-940b-334071aad0cf"
	rgid := "c5e5e8ad-dc89-319e-8b2d-b3ff5e59fcea"

	a := makeActivity(t)
	e := model.TrackEvent{
		User: user,
		Date: time.Now(),
		RID:  rid,
		RGID: rgid,
	}
	err := a.createTrackEvent(&e)
	if err != nil {
		t.Fatal(err)
	}

	end := time.Now()
	start := end.Add(time.Hour*-1)
	events := a.trackEventsFrom("takeout", start, end, 10)

	if len(events) != 1 {
		t.Error("expect 1 event")
	}

	a.deleteTrackEvents(user)
	if len(a.trackEvents(user)) != 0 {
		t.Error("expect no events")
	}
}

func TestTrackDayCountsFrom(t *testing.T) {
	user := "takeout"
	rid := "7b486d22-ade1-4d61-940b-334071aad0cf"
	rgid := "c5e5e8ad-dc89-319e-8b2d-b3ff5e59fcea"

	a := makeActivity(t)

	for i := 0; i < 10; i++ {
		e := model.TrackEvent{
			User: user,
			Date: time.Now(),
			RID:  rid,
			RGID: rgid,
		}
		err := a.createTrackEvent(&e)
		if err != nil {
			t.Fatal(err)
		}
	}

	end := time.Now()
	start := end.Add(time.Hour*-1)
	counts := a.trackDayCountsFrom("takeout", start, end, start.Location(), 100)

	// for _, c := range counts {
	// 	t.Logf("%+v\n", c)
	// }

	if len(counts) != 1 {
		t.Error("expect 1 counts")
	}

	if counts[0].Count != 10 {
		t.Error("expect count is 10")
	}

	a.deleteTrackEvents(user)
	if len(a.trackEvents(user)) != 0 {
		t.Error("expect no events")
	}
}
