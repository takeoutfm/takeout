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

package film

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"takeoutfm.dev/takeout/lib/bucket"
	"takeoutfm.dev/takeout/model"
)

type dirBucket struct{ dir string }

func (b dirBucket) List(context.Context, time.Time) (chan *bucket.Object, error) { return nil, nil }
func (b dirBucket) IsLocal() bool                                                { return true }
func (b dirBucket) ObjectURL(ctx context.Context, key string) *url.URL {
	return &url.URL{Scheme: "file", Path: filepath.Join(b.dir, key)}
}

// flac10s is a minimal FLAC header for 10 seconds of audio.
func flac10s() []byte {
	si := make([]byte, 34)
	v := uint64(44100)<<44 | uint64(1)<<41 | uint64(15)<<36 | 441000
	for i := 0; i < 8; i++ {
		si[10+i] = byte(v >> (56 - 8*i))
	}
	return append([]byte("fLaC\x80\x00\x00\x22"), si...)
}

func TestSyncDurations(t *testing.T) {
	f := makeFilm(t)
	dir := t.TempDir()
	f.buckets = []bucket.Bucket{dirBucket{dir}}

	os.WriteFile(filepath.Join(dir, "good.mkv"), flac10s(), 0o644)
	os.WriteFile(filepath.Join(dir, "bad.mkv"), []byte("not a movie"), 0o644)

	movies := []model.Movie{
		{TMID: 9001, Title: "Good", Key: "good.mkv"},
		{TMID: 9002, Title: "Bad", Key: "bad.mkv"},
		{TMID: 9003, Title: "Missing", Key: "missing.mkv"},
		{TMID: 9004, Title: "No Key"},
		{TMID: 9005, Title: "Done", Key: "good.mkv", Duration: 42},
		{TMID: 9006, Title: "Upgraded", Key: "good.mkv"},
	}
	for i := range movies {
		if err := f.createMovie(&movies[i]); err != nil {
			t.Fatal(err)
		}
	}
	// A row from before the duration column existed: adding the column left it NULL, not 0.
	if err := f.db.Exec("update movies set duration = null where tm_id = 9006").Error; err != nil {
		t.Fatal(err)
	}

	results := f.SyncDurations(t.Context(), nil, 100)
	if len(results) != 4 {
		t.Fatalf("got %d results", len(results))
	}

	want := map[int64]int{9001: 10, 9002: -1, 9003: -1, 9004: 0, 9005: 42, 9006: 10}
	for tmid, secs := range want {
		m, err := f.LookupTMID(int(tmid))
		if err != nil {
			t.Fatal(err)
		}
		if m.Duration != secs {
			t.Errorf("%s: got %d want %d", m.Title, m.Duration, secs)
		}
	}

	// nothing left to scan
	if results := f.SyncDurations(t.Context(), nil, 100); len(results) != 0 {
		t.Errorf("rescanned %d", len(results))
	}
}
