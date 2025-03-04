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

package player // import "takeoutfm.dev/takeout/player"

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	maxMetadataLength = 1024
	maxIntervalLength = 4 * 1024 * 1024
)

var (
	ErrInvalidMetadataLength = errors.New("icy metadata length is invalid")
	ErrInvalidIntervalLength = errors.New("icy metadata interval is invalid")

	metaRegexp = regexp.MustCompile(`^(\w+)=["'](.+)["']$`)

	HeaderIcyBr          = http.CanonicalHeaderKey("Icy-Br")
	HeaderIcyDescription = http.CanonicalHeaderKey("Icy-Description")
	HeaderIcyGenre       = http.CanonicalHeaderKey("Icy-Genre")
	HeaderIcyMetaData    = http.CanonicalHeaderKey("Icy-MetaData")
	HeaderIcyMetaInt     = http.CanonicalHeaderKey("Icy-MetaInt")
	HeaderIcyName        = http.CanonicalHeaderKey("Icy-Name")
	HeaderIcyPublic      = http.CanonicalHeaderKey("Icy-Pub")
	HeaderIcyUrl         = http.CanonicalHeaderKey("Icy-Url")
)

type Icy struct {
	OnMetadata func(IcyMetadata)
	reader     io.ReadCloser
	interval   int
	offset     int
	metadata   []byte
}

type IcyHeaders struct {
	Bitrate     int
	Description string
	Genre       string
	Interval    int
	Name        string
	Public      bool
	Url         string
}

type IcyMetadata struct {
	StreamTitle string
	StreamUrl   string
}

func NewIcyReader(interval int, reader io.ReadCloser, onMetaData func(IcyMetadata)) *Icy {
	return &Icy{interval: interval, reader: reader, OnMetadata: onMetaData}
}

func (icy *Icy) Read(p []byte) (n int, err error) {
	m := icy.interval - icy.offset
	if len(p) > m {
		n, err = io.ReadFull(icy.reader, p[:m])
	} else {
		n, err = icy.reader.Read(p)
	}
	if err != nil {
		return
	}
	icy.offset += n
	if icy.offset == icy.interval {
		if icy.metadata == nil {
			icy.metadata = make([]byte, 64)
		}
		_, err = icy.reader.Read(icy.metadata[:1])
		if err != nil {
			return
		}
		l := int(icy.metadata[0]) * 16
		if l > maxMetadataLength {
			err = ErrInvalidMetadataLength
			return
		}
		// zero length is expected unless metadata has changed
		if l > 0 {
			if l > len(icy.metadata) {
				icy.metadata = make([]byte, l)
			}
			_, err = io.ReadFull(icy.reader, icy.metadata[:l])
			if err != nil {
				return
			}
			var data IcyMetadata
			//StreamTitle='Joy Division - Ceremony';
			//StreamTitle='The Stream Title';StreamUrl='https://example.com';
			parts := strings.Split(string(icy.metadata), ";")
			for _, part := range parts {
				matches := metaRegexp.FindStringSubmatch(part)
				if matches != nil {
					name, value := matches[1], matches[2]
					if strings.EqualFold(name, "StreamTitle") {
						data.StreamTitle = value
					} else if strings.EqualFold(name, "StreamUrl") {
						data.StreamUrl = value
					}
				}
			}
			if icy.OnMetadata != nil {
				icy.OnMetadata(data)
			}
		}
		icy.offset = 0
	}

	return n, err
}

func (icy *Icy) Close() error {
	return icy.reader.Close()
}
