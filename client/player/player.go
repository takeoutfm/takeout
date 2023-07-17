// Copyright 2023 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package player

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/takeoutfm/takeout/client/api"
	"github.com/takeoutfm/takeout/lib/spiff"
	"github.com/takeoutfm/takeout/lib/str"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
)

type playing struct {
	format   beep.Format
	streamer beep.StreamSeekCloser
	headers  IcyHeaders
	metadata IcyMetadata
}

type playerAction int32

const (
	ActionPlay playerAction = iota
	ActionNext
	ActionPrevious
	ActionPause
	ActionStop

)

var (
	HeaderContentType = http.CanonicalHeaderKey("Content-type")

	HeaderIcyBr          = http.CanonicalHeaderKey("Icy-Br")
	HeaderIcyDescription = http.CanonicalHeaderKey("Icy-Description")
	HeaderIcyGenre       = http.CanonicalHeaderKey("Icy-Genre")
	HeaderIcyMetaData    = http.CanonicalHeaderKey("Icy-MetaData")
	HeaderIcyMetaInt     = http.CanonicalHeaderKey("Icy-MetaInt")
	HeaderIcyName        = http.CanonicalHeaderKey("Icy-Name")
	HeaderIcyPublic      = http.CanonicalHeaderKey("Icy-Pub")
	HeaderIcyUrl         = http.CanonicalHeaderKey("Icy-Url")
)

type Options struct {
	Repeat  bool
	Buffer  time.Duration
	OnTrack func(*Player)
	OnError func(*Player, error)
}

func NewOptions() *Options {
	return &Options{Repeat: false, Buffer: 0}
}

type Player struct {
	context  api.ApiContext
	playlist *spiff.Playlist
	playing  *playing
	options  *Options
	control  chan playerAction
	errors   chan error
	done     chan struct{}
}

func NewPlayer(context api.ApiContext, playlist *spiff.Playlist, options *Options) *Player {
	return &Player{context: context, playlist: playlist, options: options}
}

func (p *Player) Next() {
	p.control <- ActionNext
}

func (p *Player) Previous() {
	p.control <- ActionPrevious
}

func (p *Player) Stop() {
	p.control <- ActionStop
}

func (p *Player) Position() (time.Duration, time.Duration) {
	if p.playing == nil {
		return 0, 0
	}
	pos := p.playing.format.SampleRate.D(p.playing.streamer.Position()).Round(time.Second)
	len := p.playing.format.SampleRate.D(p.playing.streamer.Len()).Round(time.Second)
	return pos, len
}

func (p *Player) Index() int {
	return p.playlist.Index
}

func (p *Player) Length() int {
	return len(p.playlist.Spiff.Entries)
}

func (p *Player) Title() string {
	if p.IsStream() && len(p.playing.metadata.StreamTitle) > 0 {
		return p.playing.metadata.StreamTitle
	}
	return p.current().Title
}

func (p *Player) Artist() string {
	return p.current().Creator
}

func (p *Player) Album() string {
	return p.current().Album
}

func (p *Player) Image() string {
	return p.current().Image
}

func (p *Player) Track() (string, string, string, string) {
	track := p.current()
	return track.Title, track.Album, track.Creator, track.Image
}

func (p *Player) Start() {
	p.control = make(chan playerAction)
	p.done = make(chan struct{})
	p.play()

	for {
		select {
		case action := <-p.control:
			//fmt.Printf("got action %v\n", action)
			switch action {
			case ActionNext:
				p.next()
			case ActionPrevious:
				p.prev()
			case ActionStop:
				p.stop()
				// case ActionPause:
			}
		case err := <-p.errors:
			p.handle(err)
		case <-p.done:
			return
		}
	}
}

func (p *Player) handle(err error) {
	if p.options != nil && p.options.OnError != nil {
		p.options.OnError(p, err)
	} else {
		fmt.Println(err)
	}
}

func (p *Player) optionRepeat() bool {
	if p.options != nil {
		return p.options.Repeat
	}
	return false
}

func (p *Player) optionBuffer() time.Duration {
	if p.options != nil && p.options.Buffer != 0 {
		return p.options.Buffer
	}
	return time.Second
}

func (p *Player) current() *spiff.Entry {
	index := p.playlist.Index
	return &p.playlist.Spiff.Entries[index]
}

func (p *Player) stop() {
	p.clear()
	if p.done != nil {
		p.done <- struct{}{}
		p.done = nil
	}
}

func (p *Player) clear() {
	if p.playing != nil {
		p.playing.streamer.Close()
		p.playing = nil
	}
	speaker.Clear()
}

func (p *Player) IsStream() bool {
	return p.playlist.Type == spiff.TypeStream
}

func (p *Player) IsPodcast() bool {
	return p.playlist.Type == spiff.TypePodcast
}

func (p *Player) IsMusic() bool {
	return p.playlist.Type == spiff.TypeMusic
}

func (p *Player) play() {
	p.clear()

	index := p.playlist.Index
	if index < 0 || index >= p.Length() {
		index = 0
	}

	track := p.current()

	var url *url.URL
	var err error
	if p.IsStream() {
		url, err = url.Parse(track.Location[0])
		if err != nil {
			p.errors <- err
			return
		}
	} else {
		url, err = api.Locate(p.context, track.Location[0])
		if err != nil {
			p.errors <- err
			return
		}
	}

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		p.errors <- err
		return
	}

	if p.IsStream() {
		req.Header.Add(HeaderIcyMetaData, "1")
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		p.errors <- err
		return
	}
	reader := resp.Body

	var headers *IcyHeaders
	if p.IsStream() && resp.Header.Get(HeaderIcyMetaInt) != "" {
		headers = &IcyHeaders{
			Bitrate:     str.Atoi(resp.Header.Get(HeaderIcyBr)),
			Description: resp.Header.Get(HeaderIcyDescription),
			Genre:       resp.Header.Get(HeaderIcyGenre),
			Interval:    str.Atoi(resp.Header.Get(HeaderIcyMetaInt)),
			Name:        resp.Header.Get(HeaderIcyName),
			Public:      resp.Header.Get(HeaderIcyPublic) == "true",
			Url:         resp.Header.Get(HeaderIcyUrl),
		}
		if headers.Interval > 0 {
			if headers.Interval > maxIntervalLength {
				panic(ErrInvalidIntervalLength)
			} else if headers.Interval > 0 {
				reader = NewIcyReader(headers.Interval, reader, p.onIcyMetadata)
			}
		}
	}

	streamer, format, err := decoder(reader, resp.Header.Get(HeaderContentType), url.Path)
	if err != nil {
		reader.Close()
		p.errors <- err
		return
	}

	//TODO use ctrl to simulate pause
	//ctrl := &beep.Ctrl{Streamer: streamer}

	p.playing = &playing{streamer: streamer, format: format}
	if headers != nil {
		p.playing.headers = *headers
	}

	bufferSize := format.SampleRate.N(p.optionBuffer())
	speaker.Init(format.SampleRate, bufferSize)
	speaker.Play(beep.Seq(p.playing.streamer, beep.Callback(func() {
		if p.hasNext() || p.optionRepeat() {
			p.Next()
		} else {
			p.Stop()
		}
	})))
	p.onTrack()
}

func decoder(reader io.ReadCloser, contentType, path string) (beep.StreamSeekCloser, beep.Format, error) {
	if contentType == "audio/flac" ||
		contentType == "audio/x-flac" ||
		strings.HasSuffix(path, ".flac") {
		return flac.Decode(reader)
	} else if contentType == "audio/mp3" ||
		contentType == "audio/mpeg" ||
		strings.HasSuffix(path, ".mp3") {
		return mp3.Decode(reader)
	} else if contentType == "audio/ogg" ||
		strings.HasSuffix(path, ".ogg") {
		return vorbis.Decode(reader)
	} else if contentType == "audio/wav" ||
		strings.HasSuffix(path, ".wav") {
		return wav.Decode(reader)
	}
	panic("unsupported audio format")
}

func (p *Player) onTrack() {
	if p.options != nil && p.options.OnTrack != nil {
		p.options.OnTrack(p)
	}
}

func (p *Player) onIcyMetadata(data IcyMetadata) {
	p.playing.metadata = data
	p.onTrack()
}

func (p *Player) hasNext() bool {
	return (p.playlist.Index + 1) < p.Length()
}

func (p *Player) next() {
	index := p.playlist.Index + 1
	if index >= p.Length() {
		// repeat is checked elsewhere
		index = 0
	}
	p.playlist.Index = index
	p.play()
}

func (p *Player) prev() {
	index := p.playlist.Index - 1
	if index < 0 {
		index = 0
	}
	p.playlist.Index = index
	p.play()
}
