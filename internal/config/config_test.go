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

package config

import (
	"testing"
	"strings"
)

func TestDefaultConfig(t *testing.T) {
	config, err := TestingConfig()
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(config.Auth.DB.Source, ":memory:") == false {
		t.Errorf("expect ${} to work - %s", config.Auth.DB.Source)
	}

	if config.LastFM.Key != "" {
		t.Error("expect no lastfm key")
	}
	if config.LastFM.Secret != "" {
		t.Error("expect no lastfm secret")
	}

	if len(config.Music.RadioStreams) == 0 {
		t.Error("expect radio streams")
	}
	if len(config.Podcast.Series) == 0 {
		t.Error("expect podcast streams")
	}

	if config.Auth.AccessToken.Secret == "" {
		t.Error("expect access token secret")
	}
	if config.Auth.MediaToken.Secret == "" {
		t.Error("expect media token secret")
	}
	if config.Auth.CodeToken.Secret == "" {
		t.Error("expect code token secret")
	}
}
