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

package player

import (
	"github.com/faiface/beep"
)

type NotifyFunc func()

func Notify(streamer beep.StreamSeekCloser, halfway NotifyFunc) *notify {
	return &notify{s: streamer, f: &funcs{midpoint: halfway}}
}

type funcs struct {
	midpoint NotifyFunc
}

type notify struct {
	s beep.StreamSeekCloser
	f *funcs
}

func (p *notify) Stream(samples [][2]float64) (int, bool) {
	if p.f.midpoint != nil {
		// FIXME Len() may return zero
		if p.s.Position() > (p.s.Len() / 2) {
			p.f.midpoint()
			p.f.midpoint = nil
		}
	}
	return p.s.Stream(samples)
}

func (p *notify) Close() error {
	return p.s.Close()
}

func (p *notify) Len() int {
	return p.s.Len()
}

func (p *notify) Position() int {
	return p.s.Position()
}

func (p *notify) Err() error {
	return p.s.Err()
}

func (p *notify) Seek(n int) error {
	return p.s.Seek(n)
}
