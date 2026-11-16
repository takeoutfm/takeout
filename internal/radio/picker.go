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
	"math/rand/v2"
	"time"

	. "takeoutfm.dev/takeout/model"
)

const (
	// a track isn't repeated within this many tracks, or half the pool if
	// that's more, while the pool allows
	minRecent = 200
	// the same artist isn't played within this many tracks
	artistTracks = 4
	// nor within this long, when the pool allows
	artistTime = time.Hour
)

// Title is a track in a station's pool.
type Title struct {
	RID     string // recording ID, the schedule key
	Artist  string
	Release string // release group, or release name
	Seconds int
}

func (t Title) duration() time.Duration {
	return time.Duration(t.Seconds) * time.Second
}

func poolDuration(pool []Title) time.Duration {
	var d time.Duration
	for _, t := range pool {
		d += t.duration()
	}
	return d
}

// usable drops tracks without a length or recording ID, and repeats of a
// recording (the same song on several releases), keeping the first.
func usable(pool []Title) []Title {
	seen := make(map[string]bool)
	var result []Title
	for _, t := range pool {
		if t.RID == "" || t.Seconds <= 0 || seen[t.RID] {
			continue
		}
		seen[t.RID] = true
		result = append(result, t)
	}
	return result
}

// played is a track already on the schedule.
type played struct {
	Title
	Start time.Time
}

// pick schedules tracks from pool back to back from start until the last one
// starting before until. history is what the station played before start,
// oldest first. It's a pure function of its arguments: the same pool, history
// and seed give the same schedule.
//
// Rules, most important first: no repeats within the recent window, artist
// separation, and no two tracks from one release in a row. When no track
// meets them all, they're relaxed from the least important up rather than
// leave silence.
func pick(pool []Title, history []played, start, until time.Time, seed uint64) []RadioTrack {
	pool = usable(pool)
	n := len(pool)
	if n == 0 {
		return nil
	}
	// recent window: larger of minRecent and half the pool, but always
	// leaving a quarter of the pool to choose from
	window := max(minRecent, n/2)
	if limit := n - n/4; window > limit {
		window = limit
	}

	shuffled := make([]Title, n)
	copy(shuffled, pool)
	rng := rand.New(rand.NewPCG(seed, 0x726164696f))
	rng.Shuffle(n, func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	seq := append([]played(nil), history...)
	recent := make(map[string]int)
	for _, p := range seq[max(0, len(seq)-window):] {
		recent[p.RID]++
	}
	lastPlayed := make(map[string]int)
	for i, p := range seq {
		lastPlayed[p.RID] = i
	}

	notRecent := func(t Title, _ time.Time) bool {
		return recent[t.RID] == 0
	}
	artistOK := func(t Title, at time.Time, hour bool) bool {
		if t.Artist == "" {
			return true
		}
		for i := len(seq) - 1; i >= 0; i-- {
			p := seq[i]
			inTracks := len(seq)-i <= artistTracks
			inTime := hour && p.Start.After(at.Add(-artistTime))
			if !inTracks && !inTime {
				break
			}
			if p.Artist == t.Artist {
				return false
			}
		}
		return true
	}
	releaseOK := func(t Title, _ time.Time) bool {
		if t.Release == "" || len(seq) == 0 {
			return true
		}
		return seq[len(seq)-1].Release != t.Release
	}
	levels := []func(Title, time.Time) bool{
		func(t Title, at time.Time) bool {
			return notRecent(t, at) && artistOK(t, at, true) && releaseOK(t, at)
		},
		func(t Title, at time.Time) bool {
			return notRecent(t, at) && artistOK(t, at, false) && releaseOK(t, at)
		},
		func(t Title, at time.Time) bool {
			return notRecent(t, at) && artistOK(t, at, false)
		},
		notRecent,
	}

	next := 0
	choose := func(at time.Time) Title {
		for _, ok := range levels {
			for i := 0; i < n; i++ {
				j := (next + i) % n
				if ok(shuffled[j], at) {
					next = (j + 1) % n
					return shuffled[j]
				}
			}
		}
		// everything was played lately: the least recently played
		best := 0
		for j := 1; j < n; j++ {
			if lastPlayed[shuffled[j].RID] < lastPlayed[shuffled[best].RID] {
				best = j
			}
		}
		next = (best + 1) % n
		return shuffled[best]
	}

	var out []RadioTrack
	for t := start; t.Before(until); {
		title := choose(t)
		end := t.Add(title.duration())
		out = append(out, RadioTrack{RID: title.RID, Start: t, End: end})

		seq = append(seq, played{Title: title, Start: t})
		lastPlayed[title.RID] = len(seq) - 1
		recent[title.RID]++
		if old := len(seq) - 1 - window; old >= 0 {
			recent[seq[old].RID]--
		}
		t = end
	}
	return out
}
