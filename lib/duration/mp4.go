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

package duration

import (
	"time"
)

type mp4Box struct {
	typ  string
	data int64 // offset of box data
	end  int64 // offset after the box
}

func readBox(s src, off, limit int64) (mp4Box, error) {
	h, err := s.read(off, 8)
	if err != nil {
		return mp4Box{}, err
	}
	size := int64(be32(h))
	hdr := int64(8)
	switch size {
	case 0:
		size = limit - off
	case 1:
		l, err := s.read(off+8, 8)
		if err != nil {
			return mp4Box{}, err
		}
		size = int64(be64(l))
		hdr = 16
	}
	if size < hdr || off+size > limit || size < 0 {
		return mp4Box{}, ErrInvalid
	}
	return mp4Box{typ: string(h[4:8]), data: off + hdr, end: off + size}, nil
}

// findBox returns the first box of type typ in [off, limit).
func findBox(s src, off, limit int64, typ string) (mp4Box, error) {
	for i := 0; off+8 <= limit && i < 1024; i++ {
		b, err := readBox(s, off, limit)
		if err != nil {
			return mp4Box{}, err
		}
		if b.typ == typ {
			return b, nil
		}
		off = b.end
	}
	return mp4Box{}, ErrNoDuration
}

func mp4Duration(s src) (time.Duration, error) {
	moov, err := findBox(s, 0, s.size, "moov")
	if err != nil {
		return 0, err
	}
	mvhd, err := findBox(s, moov.data, moov.end, "mvhd")
	if err != nil {
		return 0, err
	}
	b, err := s.read(mvhd.data, 32)
	if err != nil {
		return 0, err
	}
	var scale, duration uint64
	if b[0] == 1 {
		// version, flags, creation(8), modification(8), timescale, duration(8)
		scale = uint64(be32(b[20:24]))
		duration = be64(b[24:32])
		if duration == 1<<64-1 {
			duration = 0
		}
	} else {
		// version, flags, creation, modification, timescale, duration
		scale = uint64(be32(b[12:16]))
		duration = uint64(be32(b[16:20]))
		if duration == 1<<32-1 {
			duration = 0
		}
	}
	if duration == 0 {
		// fragmented files may have the total in mvex/mehd
		duration = mehdDuration(s, moov)
	}
	if duration == 0 || scale == 0 {
		return 0, ErrNoDuration
	}
	return samples(duration, scale), nil
}

func mehdDuration(s src, moov mp4Box) uint64 {
	mvex, err := findBox(s, moov.data, moov.end, "mvex")
	if err != nil {
		return 0
	}
	mehd, err := findBox(s, mvex.data, mvex.end, "mehd")
	if err != nil {
		return 0
	}
	b, err := s.read(mehd.data, 12)
	if err != nil {
		return 0
	}
	if b[0] == 1 {
		return be64(b[4:12])
	}
	return uint64(be32(b[4:8]))
}
