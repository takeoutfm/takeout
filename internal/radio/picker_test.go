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

package radio

import (
	"fmt"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

// pool makes n tracks by artists artists, each artist's tracks spread over
// releases of 10 tracks.
func pool(n, artists int) []Title {
	var list []Title
	for i := 0; i < n; i++ {
		a := i % artists
		list = append(list, Title{
			RID:     fmt.Sprintf("r%d", i),
			Artist:  fmt.Sprintf("a%d", a),
			Release: fmt.Sprintf("a%d-rel%d", a, i/artists/10),
			Seconds: 180 + i%60,
		})
	}
	return list
}

func TestPickBackToBack(t *testing.T) {
	until := t0.Add(24 * time.Hour)
	out := pick(pool(500, 100), nil, t0, until, 1)
	if len(out) == 0 || !out[0].Start.Equal(t0) {
		t.Fatalf("starts %v", out)
	}
	for i := 1; i < len(out); i++ {
		if !out[i].Start.Equal(out[i-1].End) {
			t.Fatalf("gap at %d", i)
		}
	}
	last := out[len(out)-1]
	if !last.Start.Before(until) || last.End.Before(until) {
		t.Errorf("last %v %v", last.Start, last.End)
	}
}

func TestPickDeterministic(t *testing.T) {
	a := pick(pool(300, 50), nil, t0, t0.Add(6*time.Hour), 7)
	b := pick(pool(300, 50), nil, t0, t0.Add(6*time.Hour), 7)
	c := pick(pool(300, 50), nil, t0, t0.Add(6*time.Hour), 8)
	if len(a) != len(b) {
		t.Fatal("lengths differ")
	}
	differs := false
	for i := range a {
		if a[i].RID != b[i].RID {
			t.Fatalf("same seed differs at %d", i)
		}
		if i < len(c) && a[i].RID != c[i].RID {
			differs = true
		}
	}
	if !differs {
		t.Error("another seed gave the same schedule")
	}
}

func TestPickRules(t *testing.T) {
	p := pool(1000, 200)
	artist := make(map[string]string)
	release := make(map[string]string)
	for _, x := range p {
		artist[x.RID], release[x.RID] = x.Artist, x.Release
	}
	out := pick(p, nil, t0, t0.Add(48*time.Hour), 3)
	window := 500 // half the pool
	for i, x := range out {
		for j := max(0, i-window); j < i; j++ {
			if out[j].RID == x.RID {
				t.Fatalf("%s repeated after %d tracks", x.RID, i-j)
			}
		}
		for j := i - 1; j >= 0; j-- {
			recent := i-j <= artistTracks || out[j].Start.After(x.Start.Add(-artistTime))
			if !recent {
				break
			}
			if artist[out[j].RID] == artist[x.RID] {
				t.Fatalf("artist %s again after %d tracks", artist[x.RID], i-j)
			}
		}
		if i > 0 && release[out[i-1].RID] == release[x.RID] {
			t.Fatalf("release %s twice in a row", release[x.RID])
		}
	}
}

func TestPickHistory(t *testing.T) {
	p := pool(300, 60)
	first := pick(p, nil, t0, t0.Add(2*time.Hour), 1)
	var history []played
	for _, x := range first {
		for _, y := range p {
			if y.RID == x.RID {
				history = append(history, played{Title: y, Start: x.Start})
			}
		}
	}
	second := pick(p, history, first[len(first)-1].End, t0.Add(4*time.Hour), 2)
	seen := make(map[string]bool)
	for _, x := range first {
		seen[x.RID] = true
	}
	for _, x := range second {
		if seen[x.RID] {
			t.Fatalf("%s repeated from history", x.RID)
		}
	}
	if a := history[len(history)-1].Artist; artistOfRID(p, second[0].RID) == a {
		t.Errorf("artist %s right after history", a)
	}
}

func artistOfRID(p []Title, rid string) string {
	for _, x := range p {
		if x.RID == rid {
			return x.Artist
		}
	}
	return ""
}

func TestPickSmallPool(t *testing.T) {
	// one artist, one release: every rule but repeats has to give way
	var p []Title
	for i := 0; i < 8; i++ {
		p = append(p, Title{RID: fmt.Sprintf("r%d", i), Artist: "a", Release: "x", Seconds: 200})
	}
	out := pick(p, nil, t0, t0.Add(3*time.Hour), 1)
	if d := out[len(out)-1].End.Sub(t0); d < 3*time.Hour {
		t.Fatalf("only %s scheduled", d)
	}
	// with 8 tracks the window is 6: no repeat within 6 tracks
	for i := range out {
		for j := max(0, i-6); j < i; j++ {
			if out[j].RID == out[i].RID {
				t.Fatalf("%s repeated after %d tracks", out[i].RID, i-j)
			}
		}
	}
}

func TestPickUnusable(t *testing.T) {
	p := []Title{
		{RID: "", Seconds: 100},
		{RID: "zero", Seconds: 0},
		{RID: "a", Seconds: 100},
		{RID: "a", Seconds: 120}, // same recording on another release
	}
	out := pick(p, nil, t0, t0.Add(10*time.Minute), 1)
	for _, x := range out {
		if x.RID != "a" || x.End.Sub(x.Start) != 100*time.Second {
			t.Fatalf("got %+v", x)
		}
	}
	if pick(nil, nil, t0, t0.Add(time.Hour), 1) != nil {
		t.Error("empty pool scheduled something")
	}
}
