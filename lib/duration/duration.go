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

// Package duration reads the playing time of media files from their headers
// without decoding them. Supported containers are Matroska (mkv), MP4/M4A,
// FLAC, Ogg (Vorbis, Opus and FLAC) and MP3. Only a few small reads are made,
// so files can be probed remotely with HTTP range requests.
package duration // import "takeoutfm.dev/takeout/lib/duration"

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

var (
	ErrUnknownFormat = errors.New("unknown media format")
	ErrNoDuration    = errors.New("duration not found")
	ErrInvalid       = errors.New("invalid media file")
)

// Durations outside of this range are treated as parse errors.
const maxDuration = 24 * time.Hour

// Probe returns the duration of the media in r, which is size bytes long.
// The format is detected from the content.
func Probe(r io.ReaderAt, size int64) (time.Duration, error) {
	s := src{r: r, size: size}
	head, err := s.readUpTo(0, 16)
	if err != nil {
		return 0, err
	}

	var d time.Duration
	switch {
	case len(head) >= 4 && bytes.Equal(head[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}):
		d, err = mkvDuration(s)
	case len(head) >= 8 && string(head[4:8]) == "ftyp":
		d, err = mp4Duration(s)
	case len(head) >= 4 && string(head[:4]) == "fLaC":
		d, err = flacDuration(s, 0)
	case len(head) >= 4 && string(head[:4]) == "OggS":
		d, err = oggDuration(s)
	case len(head) >= 3 && string(head[:3]) == "ID3":
		// mp3 or flac with a leading id3 tag
		var off int64
		off, err = skipID3(s, 0)
		if err != nil {
			return 0, err
		}
		var b []byte
		b, err = s.readUpTo(off, 4)
		if err == nil && len(b) == 4 && string(b) == "fLaC" {
			d, err = flacDuration(s, off)
		} else {
			d, err = mp3Duration(s, off)
		}
	case len(head) >= 2 && head[0] == 0xff && head[1]&0xe0 == 0xe0:
		d, err = mp3Duration(s, 0)
	default:
		return 0, ErrUnknownFormat
	}
	if err != nil {
		return 0, err
	}
	if d <= 0 || d > maxDuration {
		return 0, ErrNoDuration
	}
	return d, nil
}

// src wraps a ReaderAt with a known size and bounds-checked reads.
type src struct {
	r    io.ReaderAt
	size int64
}

// read returns exactly n bytes at off or an error.
func (s src) read(off int64, n int) ([]byte, error) {
	if off < 0 || n < 0 || off+int64(n) > s.size {
		return nil, ErrInvalid
	}
	buf := make([]byte, n)
	k, err := s.r.ReadAt(buf, off)
	if k == n {
		return buf, nil
	}
	if err == nil || err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return nil, err
}

// readUpTo returns up to n bytes at off, fewer at the end of the file.
func (s src) readUpTo(off int64, n int) ([]byte, error) {
	if off < 0 || off >= s.size {
		return nil, ErrInvalid
	}
	if rem := s.size - off; int64(n) > rem {
		n = int(rem)
	}
	return s.read(off, n)
}

func be16(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
func be32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
func be64(b []byte) uint64 { return binary.BigEndian.Uint64(b) }

// samples converts a sample count at rate to a duration without overflow for
// any realistic length.
func samples(n uint64, rate uint64) time.Duration {
	if rate == 0 {
		return 0
	}
	secs := n / rate
	rem := n % rate
	return time.Duration(secs)*time.Second +
		time.Duration(rem*uint64(time.Second)/rate)
}
