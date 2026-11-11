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
	"math"
	"time"
)

const (
	ebmlHeader     = 0x1a45dfa3
	ebmlSegment    = 0x18538067
	ebmlSeekHead   = 0x114d9b74
	ebmlSeek       = 0x4dbb
	ebmlSeekID     = 0x53ab
	ebmlSeekPos    = 0x53ac
	ebmlInfo       = 0x1549a966
	ebmlTimescale  = 0x2ad7b1
	ebmlDuration   = 0x4489
	ebmlCluster    = 0x1f43b675
	ebmlMaxScan    = 64
	ebmlMaxElement = 1 << 20 // max size of elements read into memory
)

type ebmlElement struct {
	id      uint32
	data    int64 // offset of the element data
	size    int64 // data size, -1 if unknown
	end     int64 // offset after the element, or the limit if size is unknown
	unknown bool
}

// readVint reads an EBML variable length integer. With marker the length
// marker bit is kept, as for element IDs.
func readVint(s src, off int64, marker bool) (uint64, int, bool, error) {
	b, err := s.read(off, 1)
	if err != nil {
		return 0, 0, false, err
	}
	first := b[0]
	if first == 0 {
		return 0, 0, false, ErrInvalid
	}
	n := 1
	for mask := byte(0x80); first&mask == 0; mask >>= 1 {
		n++
	}
	b, err = s.read(off, n)
	if err != nil {
		return 0, 0, false, err
	}
	v := uint64(first)
	if !marker {
		v &= uint64(0xff >> n)
	}
	allOnes := v == uint64(0xff>>n)
	for i := 1; i < n; i++ {
		v = v<<8 | uint64(b[i])
		allOnes = allOnes && b[i] == 0xff
	}
	return v, n, !marker && allOnes, nil
}

func readElement(s src, off, limit int64) (ebmlElement, error) {
	id, n, _, err := readVint(s, off, true)
	if err != nil {
		return ebmlElement{}, err
	}
	if n > 4 {
		return ebmlElement{}, ErrInvalid
	}
	size, m, unknown, err := readVint(s, off+int64(n), false)
	if err != nil {
		return ebmlElement{}, err
	}
	e := ebmlElement{id: uint32(id), data: off + int64(n) + int64(m)}
	if e.data > limit {
		return ebmlElement{}, ErrInvalid
	}
	if unknown {
		e.size, e.unknown, e.end = -1, true, limit
	} else {
		if size > uint64(limit-e.data) {
			return ebmlElement{}, ErrInvalid
		}
		e.size = int64(size)
		e.end = e.data + e.size
	}
	return e, nil
}

func (e ebmlElement) bytes(s src) ([]byte, error) {
	if e.unknown || e.size > ebmlMaxElement {
		return nil, ErrInvalid
	}
	return s.read(e.data, int(e.size))
}

func (e ebmlElement) uint(s src) (uint64, error) {
	b, err := e.bytes(s)
	if err != nil {
		return 0, err
	}
	if len(b) > 8 {
		return 0, ErrInvalid
	}
	var v uint64
	for _, x := range b {
		v = v<<8 | uint64(x)
	}
	return v, nil
}

func (e ebmlElement) float(s src) (float64, error) {
	b, err := e.bytes(s)
	if err != nil {
		return 0, err
	}
	switch len(b) {
	case 4:
		return float64(math.Float32frombits(be32(b))), nil
	case 8:
		return math.Float64frombits(be64(b)), nil
	}
	return 0, ErrInvalid
}

// children calls fn for each child element of parent until fn returns false.
func (e ebmlElement) children(s src, fn func(ebmlElement) (bool, error)) error {
	off := e.data
	for i := 0; off < e.end && i < 1024; i++ {
		c, err := readElement(s, off, e.end)
		if err != nil {
			return err
		}
		more, err := fn(c)
		if err != nil || !more {
			return err
		}
		if c.unknown {
			return nil
		}
		off = c.end
	}
	return nil
}

func mkvDuration(s src) (time.Duration, error) {
	hdr, err := readElement(s, 0, s.size)
	if err != nil || hdr.id != ebmlHeader || hdr.unknown {
		return 0, ErrInvalid
	}

	// find the segment, skipping void or other top-level elements
	var seg ebmlElement
	off := hdr.end
	for i := 0; ; i++ {
		if i >= 8 || off >= s.size {
			return 0, ErrInvalid
		}
		seg, err = readElement(s, off, s.size)
		if err != nil {
			return 0, err
		}
		if seg.id == ebmlSegment {
			break
		}
		if seg.unknown {
			return 0, ErrInvalid
		}
		off = seg.end
	}

	// walk the segment's children for Info, noting where SeekHead says it is
	infoPos := int64(-1)
	var info *ebmlElement
	off = seg.data
	for i := 0; i < ebmlMaxScan && off < seg.end; i++ {
		e, err := readElement(s, off, seg.end)
		if err != nil {
			break
		}
		if e.id == ebmlInfo {
			info = &e
			break
		}
		if e.id == ebmlSeekHead && infoPos < 0 {
			if p, err := seekInfo(s, e); err == nil && p >= 0 {
				infoPos = seg.data + p
			}
		}
		if e.id == ebmlCluster || e.unknown {
			break
		}
		off = e.end
	}

	if info == nil && infoPos >= 0 {
		e, err := readElement(s, infoPos, seg.end)
		if err == nil && e.id == ebmlInfo {
			info = &e
		}
	}
	if info == nil {
		return 0, ErrNoDuration
	}
	return parseInfo(s, *info)
}

// seekInfo returns the position of Info relative to the segment data, or -1.
func seekInfo(s src, head ebmlElement) (int64, error) {
	pos := int64(-1)
	err := head.children(s, func(seek ebmlElement) (bool, error) {
		if seek.id != ebmlSeek {
			return true, nil
		}
		var id uint64
		var p int64 = -1
		err := seek.children(s, func(c ebmlElement) (bool, error) {
			switch c.id {
			case ebmlSeekID:
				b, err := c.bytes(s)
				if err != nil {
					return false, err
				}
				id = 0
				for _, x := range b {
					id = id<<8 | uint64(x)
				}
			case ebmlSeekPos:
				v, err := c.uint(s)
				if err != nil {
					return false, err
				}
				p = int64(v)
			}
			return true, nil
		})
		if err != nil {
			return false, err
		}
		if id == ebmlInfo && p >= 0 {
			pos = p
			return false, nil
		}
		return true, nil
	})
	return pos, err
}

func parseInfo(s src, info ebmlElement) (time.Duration, error) {
	scale := uint64(1000000)
	duration := -1.0
	err := info.children(s, func(c ebmlElement) (bool, error) {
		var err error
		switch c.id {
		case ebmlTimescale:
			scale, err = c.uint(s)
		case ebmlDuration:
			duration, err = c.float(s)
		}
		return err == nil, err
	})
	if err != nil {
		return 0, err
	}
	if duration <= 0 || scale == 0 {
		return 0, ErrNoDuration
	}
	ns := duration * float64(scale)
	if ns > float64(maxDuration) {
		return 0, ErrNoDuration
	}
	return time.Duration(ns), nil
}
