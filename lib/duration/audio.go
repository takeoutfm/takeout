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
	"bytes"
	"encoding/binary"
	"time"
)

// streamInfo parses a 34 byte FLAC STREAMINFO block.
func streamInfo(si []byte) (time.Duration, error) {
	if len(si) < 18 {
		return 0, ErrInvalid
	}
	// sample rate (20), channels (3), bits per sample (5), total samples (36)
	v := be64(si[10:18])
	rate := v >> 44
	total := v & (1<<36 - 1)
	if rate == 0 || total == 0 {
		return 0, ErrNoDuration
	}
	return samples(total, rate), nil
}

func flacDuration(s src, off int64) (time.Duration, error) {
	b, err := s.read(off, 4+4+34)
	if err != nil {
		return 0, err
	}
	if string(b[:4]) != "fLaC" || b[4]&0x7f != 0 {
		return 0, ErrInvalid
	}
	return streamInfo(b[8:])
}

// skipID3 returns the offset after any ID3v2 tags at off.
func skipID3(s src, off int64) (int64, error) {
	for i := 0; i < 4; i++ {
		b, err := s.readUpTo(off, 10)
		if err != nil {
			return 0, err
		}
		if len(b) < 10 || string(b[:3]) != "ID3" {
			return off, nil
		}
		for _, x := range b[6:10] {
			if x&0x80 != 0 {
				return 0, ErrInvalid
			}
		}
		size := int64(b[6])<<21 | int64(b[7])<<14 | int64(b[8])<<7 | int64(b[9])
		off += 10 + size
		if b[5]&0x10 != 0 {
			off += 10 // footer
		}
	}
	return off, nil
}

const oggTail = 65536

func oggDuration(s src) (time.Duration, error) {
	head, err := s.readUpTo(0, 512)
	if err != nil {
		return 0, err
	}
	if len(head) < 28 || string(head[:4]) != "OggS" {
		return 0, ErrInvalid
	}
	serial := binary.LittleEndian.Uint32(head[14:18])
	nsegs := int(head[26])
	start := 27 + nsegs
	if start >= len(head) {
		return 0, ErrInvalid
	}
	packet := head[start:]

	var rate, preskip uint64
	switch {
	case len(packet) >= 16 && packet[0] == 1 && string(packet[1:7]) == "vorbis":
		rate = uint64(binary.LittleEndian.Uint32(packet[12:16]))
	case len(packet) >= 12 && string(packet[:8]) == "OpusHead":
		// opus granule positions are always 48kHz
		rate = 48000
		preskip = uint64(binary.LittleEndian.Uint16(packet[10:12]))
	case len(packet) >= 13+34 && packet[0] == 0x7f && string(packet[1:5]) == "FLAC" &&
		string(packet[9:13]) == "fLaC":
		si := packet[17:]
		rate = be64(si[10:18]) >> 44
	default:
		return 0, ErrUnknownFormat
	}
	if rate == 0 {
		return 0, ErrNoDuration
	}

	// last page with a granule position for this stream
	off := max(s.size-oggTail, 0)
	tail, err := s.read(off, int(s.size-off))
	if err != nil {
		return 0, err
	}
	for i := len(tail) - 27; i >= 0; i-- {
		i = bytes.LastIndex(tail[:i+4], []byte("OggS"))
		if i < 0 || i+27 > len(tail) {
			break
		}
		p := tail[i:]
		if p[4] != 0 || binary.LittleEndian.Uint32(p[14:18]) != serial {
			continue
		}
		granule := binary.LittleEndian.Uint64(p[6:14])
		if granule == 1<<64-1 {
			continue
		}
		if granule <= preskip {
			return 0, ErrNoDuration
		}
		return samples(granule-preskip, rate), nil
	}
	return 0, ErrNoDuration
}

var (
	mp3Rates = [4][3]uint64{
		{11025, 12000, 8000},  // MPEG 2.5
		{0, 0, 0},             // reserved
		{22050, 24000, 16000}, // MPEG 2
		{44100, 48000, 32000}, // MPEG 1
	}
	// kbps by [mpeg1][layer], index 0 is free format
	mp3Bitrates = [2][4][16]uint64{
		{ // MPEG 2 & 2.5
			{},
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // layer III
			{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},      // layer II
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256}, // layer I
		},
		{ // MPEG 1
			{},
			{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},     // layer III
			{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},    // layer II
			{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448}, // layer I
		},
	}
)

type mp3Frame struct {
	version int // 0=2.5, 2=2, 3=1
	layer   int // 1=III, 2=II, 3=I
	bitrate uint64
	rate    uint64
	mono    bool
	length  int64
	spf     uint64 // samples per frame
}

func parseFrame(h []byte) (mp3Frame, bool) {
	if len(h) < 4 || h[0] != 0xff || h[1]&0xe0 != 0xe0 {
		return mp3Frame{}, false
	}
	f := mp3Frame{
		version: int(h[1]>>3) & 3,
		layer:   int(h[1]>>1) & 3,
	}
	bri := int(h[2] >> 4)
	sri := int(h[2]>>2) & 3
	pad := int64(h[2]>>1) & 1
	if f.version == 1 || f.layer == 0 || bri == 0 || bri == 15 || sri == 3 {
		return mp3Frame{}, false
	}
	mpeg1 := 0
	if f.version == 3 {
		mpeg1 = 1
	}
	f.bitrate = mp3Bitrates[mpeg1][f.layer][bri] * 1000
	f.rate = mp3Rates[f.version][sri]
	f.mono = h[3]>>6 == 3
	switch f.layer {
	case 3:
		f.spf = 384
		f.length = (int64(12*f.bitrate/f.rate) + pad) * 4
	case 2:
		f.spf = 1152
		f.length = int64(144*f.bitrate/f.rate) + pad
	case 1:
		f.spf = 1152
		if mpeg1 == 0 {
			f.spf = 576
		}
		f.length = int64(f.spf/8*f.bitrate/f.rate) + pad
	}
	return f, f.length > 4
}

func (f mp3Frame) sideInfo() int {
	if f.version == 3 {
		if f.mono {
			return 17
		}
		return 32
	}
	if f.mono {
		return 9
	}
	return 17
}

func mp3Duration(s src, off int64) (time.Duration, error) {
	buf, err := s.readUpTo(off, 64*1024)
	if err != nil {
		return 0, err
	}

	// find the first frame that is followed by another matching frame
	pos := -1
	var f mp3Frame
	for i := 0; i+4 <= len(buf); i++ {
		var ok bool
		f, ok = parseFrame(buf[i:])
		if !ok {
			continue
		}
		next := i + int(f.length)
		if next+4 <= len(buf) {
			g, ok := parseFrame(buf[next:])
			if !ok || g.version != f.version || g.layer != f.layer || g.rate != f.rate {
				continue
			}
		} else if off+int64(next) < s.size {
			// can't confirm within the buffer; accept only at the start
			if i != 0 {
				continue
			}
		}
		pos = i
		break
	}
	if pos < 0 {
		return 0, ErrNoDuration
	}
	frame := buf[pos:]

	// Xing / Info header
	x := 4 + f.sideInfo()
	if x+12 <= len(frame) {
		tag := string(frame[x : x+4])
		if tag == "Xing" || tag == "Info" {
			flags := be32(frame[x+4 : x+8])
			if flags&1 != 0 {
				frames := uint64(be32(frame[x+8 : x+12]))
				if frames > 0 {
					return samples(frames*f.spf, f.rate), nil
				}
			}
		}
	}

	// VBRI header, always 32 bytes after the frame header
	v := 4 + 32
	if v+18 <= len(frame) && string(frame[v:v+4]) == "VBRI" {
		frames := uint64(be32(frame[v+14 : v+18]))
		if frames > 0 {
			return samples(frames*f.spf, f.rate), nil
		}
	}

	// estimate from size and bitrate, reliable only for constant bitrate
	audio := s.size - (off + int64(pos))
	if audio <= 0 {
		return 0, ErrNoDuration
	}
	return samples(uint64(audio)*8, f.bitrate), nil
}
