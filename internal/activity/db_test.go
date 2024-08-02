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

func TestReleaseEvent(t *testing.T) {
	user := "takeout"
	reid := "8b3ca77d-647d-4e3e-b3a9-e7d5dd17f3e0"
	rgid := "c5e5e8ad-dc89-319e-8b2d-b3ff5e59fcea"

	a := makeActivity(t)
	e := model.ReleaseEvent{
		User: user,
		Date: time.Now(),
		REID: reid,
		RGID: rgid,
	}
	err := a.createReleaseEvent(&e)
	if err != nil {
		t.Fatal(err)
	}

	events := a.releaseEvents(user)
	if len(events) == 0 {
		t.Error("expect events")
	}
	if events[0].REID != reid {
		t.Errorf("expect %s", reid)
	}
	if events[0].RGID != rgid {
		t.Errorf("expect %s", rgid)
	}

	a.deleteReleaseEvents(user)
	if len(a.releaseEvents(user)) != 0 {
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
