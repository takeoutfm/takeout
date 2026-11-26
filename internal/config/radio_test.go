// Copyright 2026 defsub
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

package config

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestRadioMusicQueries checks that Music is one query (never split at
// commas) or a list, and that other settings decode as before.
func TestRadioMusicQueries(t *testing.T) {
	yaml := `
Radio:
  Lineup:
    - Name: One
      Music: '+artist:"Crosby, Stills, Nash & Young" +type:single'
      MinContent: 1h
      Shows:
        - Name: Two Artists
          Music:
            - '+artist:"DIIV"'
            - '+artist:"Slowdive"'
          Length: 2h
    - Name: Empty
  Genres: [jazz, rock]
Music:
  ReleaseCountries: US,XW
`
	v := viper.New()
	configDefaults(v)
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(yaml)); err != nil {
		t.Fatal(err)
	}
	cfg, err := makeConfig(v, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	lineup := cfg.Radio.Lineup
	if len(lineup) != 2 {
		t.Fatalf("lineup %+v", lineup)
	}
	if got := fmt.Sprintf("%q", lineup[0].Music); got != `["+artist:\"Crosby, Stills, Nash & Young\" +type:single"]` {
		t.Errorf("one query %s", got)
	}
	if lineup[0].MinContent != time.Hour {
		t.Errorf("duration %s", lineup[0].MinContent)
	}
	show := lineup[0].Shows[0]
	if got := fmt.Sprintf("%q", show.Music); got != `["+artist:\"DIIV\"" "+artist:\"Slowdive\""]` {
		t.Errorf("list %s", got)
	}
	if len(lineup[1].Music) != 0 {
		t.Errorf("no music %q", lineup[1].Music)
	}
	if fmt.Sprint(cfg.Radio.Genres) != "[jazz rock]" {
		t.Errorf("genres %v", cfg.Radio.Genres)
	}
	// other string lists still split at commas, as before
	if fmt.Sprint(cfg.Music.ReleaseCountries) != "[US XW]" {
		t.Errorf("countries %v", cfg.Music.ReleaseCountries)
	}
}
