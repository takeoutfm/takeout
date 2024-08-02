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

package progress

import (
	"testing"
	"time"

	"github.com/takeoutfm/takeout/internal/config"
	"github.com/takeoutfm/takeout/model"
)

func makeProgress(t *testing.T) *Progress {
	config, err := config.TestingConfig()
	if err != nil {
		t.Fatal(err)
	}
	p := NewProgress(config)
	err = p.Open()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOffset(t *testing.T) {
	p := makeProgress(t)

	user := "takeout"
	etag := "f99f8e12a4e59264292e930f09b4d4d1"

	o := model.Offset{
		User:     user,
		ETag:     etag,
		Offset:   5,
		Duration: 0,
		Date:     time.Now(),
	}

	err := p.createOffset(&o)
	if err != nil {
		t.Fatal(err)
	}
	if o.ID == 0 {
		t.Error("expect id")
	}

	if o.Valid() != true {
		t.Error("expect valid")
	}

	o.Duration = 10
	err = p.updateOffset(&o)
	if err != nil {
		t.Error(err)
	}

	oo, err := p.lookupOffset(int(o.ID))
	if err != nil {
		t.Error(err)
	}

	if oo.Duration != 10 {
		t.Error("expect duration")
	}

	_, err = p.lookupUserOffset(user, int(o.ID))
	if err != nil {
		t.Error("expect to find user offset with id")
	}

	_, err = p.lookupUserOffsetEtag(user, etag)
	if err != nil {
		t.Error("expect to find user offset with etag")
	}

	if len(p.userOffsets(user)) == 0 {
		t.Error("expect user offsets")
	}

	p.deleteOffset(&o)

	if len(p.userOffsets(user)) != 0 {
		t.Error("expect no user offsets")
	}
}
