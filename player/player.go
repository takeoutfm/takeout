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

// Package player provides a simple audio player based on Takeout server-based
// playlists. There's support for FLAC, OGG, MP3 and WAV audio decoding, along
// with Internet radio and ICY metadata. Internally Beep and Oto are used which
// are known to work on Linux, BSDs, macOS, Windows, and more.
package player

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/takeoutfm/takeout/client"
	"github.com/takeoutfm/takeout/spiff"
)

type playing struct {
	format   beep.Format
	streamer beep.StreamSeekCloser
	headers  IcyHeaders
	metadata IcyMetadata
}

type playerAction int32

const (
	ActionNext playerAction = iota
	ActionSkipForward
	ActionSkipBackward
	ActionPause
	ActionStop
)

type Config struct {
	Repeat   bool
	Buffer   time.Duration
	OnTrack  func(*Player)
	OnListen func(*Player)
	OnPause  func(*Player)
	OnError  func(*Player, error)
}

func NewConfig() *Config {
	return &Config{Repeat: false, Buffer: 0}
}

type Player struct {
	context  client.Context
	playlist *spiff.Playlist
	playing  *playing
	config   *Config
	control  chan playerAction
	errors   chan error
	done     chan struct{}
	skipTo   int
	mu       sync.Mutex
}

func NewPlayer(context client.Context, playlist *spiff.Playlist, config *Config) *Player {
	return &Player{context: context, playlist: playlist, config: config, skipTo: -1}
}

func (p *Player) lock() {
	p.mu.Lock()
}

func (p *Player) unlock() {
	p.mu.Unlock()
}

func (p *Player) Context() client.Context {
	return p.context
}

func (p *Player) SkipForward() {
	p.control <- ActionSkipForward
}

func (p *Player) SkipBackward() {
	p.control <- ActionSkipBackward
}

func (p *Player) Next() {
	p.control <- ActionNext
}

func (p *Player) Stop() {
	p.control <- ActionStop
}

func (p *Player) Pause() {
	p.control <- ActionPause
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
	if p.IsStream() && p.playing != nil && len(p.playing.metadata.StreamTitle) > 0 {
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
	p.done = make(chan struct{}, 1)
	p.play()

	for {
		select {
		case action := <-p.control:
			switch action {
			case ActionSkipForward:
				p.skipForward()
			case ActionSkipBackward:
				p.skipBackward()
			case ActionStop:
				p.stop()
			case ActionPause:
				p.pause()
			case ActionNext:
				p.next()
			}
		case err := <-p.errors:
			p.handle(err)
		case <-p.done:
			return
		}
	}
}

func (p *Player) handle(err error) {
	if p.config != nil && p.config.OnError != nil {
		p.config.OnError(p, err)
	} else {
		fmt.Println(err)
	}
}

func (p *Player) optionRepeat() bool {
	if p.config != nil {
		return p.config.Repeat
	}
	return false
}

func (p *Player) optionBuffer() time.Duration {
	if p.config != nil && p.config.Buffer != 0 {
		return p.config.Buffer
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
	}
}

func (p *Player) pause() {
	// TODO
	p.onPause()
}

func (p *Player) clear() {
	if p.playing != nil {
		p.playing.streamer.Close()
		p.playing = nil
	}
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
	p.playIndex(p.playlist.Index)
}

func (p *Player) playIndex(index int) {
	if index < 0 || index >= p.Length() {
		index = 0
	}
	p.playlist.Index = index

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
		url, err = client.Locate(p.context, track.Location[0])
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
	req.Header.Add(client.HeaderUserAgent, p.context.UserAgent())
	if p.IsStream() {
		req.Header.Add(HeaderIcyMetaData, "1")
	}

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		p.errors <- err
		return
	}
	reader := resp.Body

	var headers *IcyHeaders
	if p.IsStream() && resp.Header.Get(HeaderIcyMetaInt) != "" {
		headers = &IcyHeaders{
			Bitrate:     atoi(resp.Header.Get(HeaderIcyBr)),
			Description: resp.Header.Get(HeaderIcyDescription),
			Genre:       resp.Header.Get(HeaderIcyGenre),
			Interval:    atoi(resp.Header.Get(HeaderIcyMetaInt)),
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

	streamer, format, err := decoder(reader, resp.Header.Get(client.HeaderContentType), url.Path)
	if err != nil {
		reader.Close()
		p.errors <- err
		return
	}

	//TODO use ctrl to simulate pause
	//ctrl := &beep.Ctrl{Streamer: streamer}

	if p.config != nil && p.config.OnListen != nil {
		streamer = Notify(streamer, func() { p.config.OnListen(p) })
	}

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

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		i = 0
	}
	return i
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
	if p.config != nil && p.config.OnTrack != nil {
		p.config.OnTrack(p)
	}
}

func (p *Player) onPause() {
	if p.config != nil && p.config.OnPause != nil {
		p.config.OnPause(p)
	}
}

func (p *Player) onIcyMetadata(data IcyMetadata) {
	p.playing.metadata = data
	p.onTrack()
}

func (p *Player) hasNext() bool {
	p.lock()
	defer p.unlock()
	return p.skipTo != -1 || (p.playlist.Index+1) < p.Length()
}

func (p *Player) forwardIndex() int {
	p.lock()
	defer p.unlock()
	index := p.playlist.Index + 1
	if index >= p.Length() {
		index = 0
	}
	return index
}

func (p *Player) backwardIndex() int {
	p.lock()
	defer p.unlock()
	index := p.playlist.Index - 1
	if index < 0 {
		index = 0
	}
	return index
}

func (p *Player) skipForward() {
	p.skipTo = p.forwardIndex()
	p.clear()
}

func (p *Player) skipBackward() {
	p.skipTo = p.backwardIndex()
	p.clear()
}

func (p *Player) next() {
	var index int
	if p.skipTo != -1 {
		index = p.skipTo
		p.skipTo = -1
	} else {
		index = p.forwardIndex()
	}
	p.playIndex(index)
}
