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

package playout

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/qeesung/image2ascii/ascii"
	"takeoutfm.dev/takeout/lib/log"
	"takeoutfm.dev/takeout/player"
)

type VisualView struct {
	screen tcell.Screen
}

func NewVisualView() Viewer {
	tcell.SetEncodingFallback(tcell.EncodingFallbackASCII)
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Panicln(err)
	}
	return &VisualView{screen: screen}
}

func (v *VisualView) OnStart(p *player.Player) {
	err := v.screen.Init()
	if err != nil {
		log.Panicln(err)
	}

	events := make(chan tcell.Event)
	go func() {
		for {
			events <- v.screen.PollEvent()
		}
	}()

	go func() {
		seconds := time.Tick(time.Second * 1)
	loop:
		for {
			select {
			case event := <-events:
				//fmt.Printf("got %v\n", event)
				switch event := event.(type) {
				case *tcell.EventKey:
					switch event.Key() {
					case tcell.KeyESC, tcell.KeyCtrlC:
						break loop
					case tcell.KeyCtrlL:
						v.update(p)
					}
					switch event.Rune() {
					case 'q', 's':
						break loop
					case '<', ',', 'p':
						p.SkipBackward()
					case '>', '.', 'n':
						p.SkipForward()
					case ' ':
						p.Pause()
					}
				case *tcell.EventResize:
					v.update(p)
				}
			case <-seconds:
				v.update(p)
			}
		}
		p.Stop()
	}()
}

func (v *VisualView) OnTrack(p *player.Player) {
	v.update(p)
}

func (v *VisualView) OnError(p *player.Player, err error) {
	log.Printf("Error %v\n", err)
}

func (v *VisualView) OnStop() {
	v.screen.Fini()
}

func (v *VisualView) update(p *player.Player) {
	w, h := v.screen.Size()
	v.screen.Clear()

	hasProgress := p.IsMusic() || p.IsPodcast()

	imgHeight := h - 1
	if hasProgress {
		imgHeight--
	}
	img := coverImage(p, w, imgHeight)
	if img.err == nil {
		renderImage(img, v.screen)
	}

	renderArtistTrack(p, v.screen, hasProgress)

	if hasProgress {
		renderProgress(p, v.screen)
	}

	v.screen.Show()
}

func mmss(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		s.SetContent(x, y, c, comb, style)
		x += w
	}
}

type raster struct {
	source string
	w, h   int
	pixels [][]ascii.CharPixel
	err    error
}

var currentImage raster

func coverImage(p *player.Player, w, h int) raster {
	source := p.Image()
	if strings.HasPrefix(source, "/img/") {
		source = strings.Join([]string{p.Context().Endpoint(), source}, "")
	}
	if source == currentImage.source &&
		w == currentImage.w &&
		h == currentImage.h {
		return currentImage
	}

	currentImage.source = source
	currentImage.w = w
	currentImage.h = h
	u, err := url.Parse(source)
	if err != nil {
		currentImage.err = err
		return currentImage
	}

	currentImage.pixels, currentImage.err = asciiImage(p.Context(), u, w, h)
	return currentImage
}

func renderImage(img raster, s tcell.Screen) {
	style := tcell.StyleDefault
	for y := 0; y < len(img.pixels); y++ {
		for x, pixel := range img.pixels[y] {
			color := tcell.NewRGBColor(
				int32(pixel.R),
				int32(pixel.G),
				int32(pixel.B))
			style = style.Background(color)
			s.SetCell(x, y, style, rune(pixel.Char))
		}
	}
}

func renderArtistTrack(p *player.Player, s tcell.Screen, hasProgress bool) {
	_, h := s.Size()

	style := tcell.StyleDefault
	x, y := 0, h-1
	if hasProgress {
		y--
	}
	emitStr(s, x, y, style, p.Artist())

	x += len(p.Artist())
	s.SetCell(x, y, style, rune(' '))
	x++
	s.SetCell(x, y, style, tcell.RuneBullet)
	x++
	s.SetCell(x, y, style, rune(' '))
	x++
	emitStr(s, x, y, style, p.Title())
}

func renderProgress(p *player.Player, s tcell.Screen) {
	style := tcell.StyleDefault

	w, h := s.Size()

	x, y := 0, h-1
	pos, dur := p.Position()
	posText := mmss(pos)
	emitStr(s, x, y, style, posText)

	durText := mmss(dur)
	x = w - len(durText)
	emitStr(s, x, y, style, durText)

	barLen := w - len(posText) - len(durText) - 2
	f := int(float64(pos.Seconds()/dur.Seconds()) * float64(barLen))

	x, y = len(posText)+1, h-1
	style = tcell.StyleDefault.Foreground(tcell.ColorLightGrey)
	b := 0
	for ; b < f; b++ {
		s.SetCell(x+b, y, style, tcell.RuneBlock)
	}
	style = tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	for ; b < barLen; b++ {
		s.SetCell(x+b, y, style, tcell.RuneHLine)
	}
}
