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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestUnknown(t *testing.T) {
	data := []byte("this is not a media file at all")
	_, err := Probe(bytes.NewReader(data), int64(len(data)))
	if !errors.Is(err, ErrUnknownFormat) {
		t.Fatalf("got %v", err)
	}
	if Retryable(err) {
		t.Fatal("parse error should not be retryable")
	}
}

func TestTruncated(t *testing.T) {
	for _, data := range [][]byte{
		{0x1a, 0x45, 0xdf, 0xa3, 0x84},
		[]byte("\x00\x00\x00\x20ftypisom"),
		[]byte("fLaC\x00\x00\x00\x22"),
		[]byte("OggS\x00"),
		[]byte("ID3\x04\x00\x00\x00\x00\x00\x00"),
	} {
		_, err := Probe(bytes.NewReader(data), int64(len(data)))
		if err == nil {
			t.Errorf("%q: expected error", data)
		}
	}
}

func TestFlacStreamInfo(t *testing.T) {
	// 44.1kHz, 2ch, 16 bit, 441000 samples = 10s
	si := make([]byte, 34)
	v := uint64(44100)<<44 | uint64(1)<<41 | uint64(15)<<36 | 441000
	for i := 0; i < 8; i++ {
		si[10+i] = byte(v >> (56 - 8*i))
	}
	data := append([]byte("fLaC\x80\x00\x00\x22"), si...)
	d, err := Probe(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if d != 10*time.Second {
		t.Fatalf("got %s", d)
	}
}

func TestContentRange(t *testing.T) {
	if n := contentRangeSize("bytes 0-99/1234"); n != 1234 {
		t.Fatal(n)
	}
	if n := contentRangeSize("bytes 0-99/*"); n != -1 {
		t.Fatal(n)
	}
}

type fixture struct {
	name string
	args []string
	// expected duration and allowed error
	want time.Duration
	tol  time.Duration
}

const seconds = 7.5

// ffmpeg generated files, skipped if ffmpeg isn't installed.
func fixtures() []fixture {
	video := []string{"-f", "lavfi", "-i", "testsrc=size=64x48:rate=10:duration=7.5",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=7.5"}
	audio := []string{"-f", "lavfi", "-i", "sine=frequency=440:duration=7.5"}
	want := time.Duration(seconds * float64(time.Second))
	cat := func(a ...[]string) []string {
		var r []string
		for _, x := range a {
			r = append(r, x...)
		}
		return r
	}
	return []fixture{
		{"video.mkv", cat(video, []string{"-c:v", "mpeg4", "-b:v", "1M", "-c:a", "aac"}), want, 100 * time.Millisecond},
		{"video.mp4", cat(video, []string{"-c:v", "mpeg4", "-b:v", "1M", "-c:a", "aac"}), want, 100 * time.Millisecond},
		{"faststart.mp4", cat(video, []string{"-c:v", "mpeg4", "-b:v", "1M", "-c:a", "aac", "-movflags", "+faststart"}), want, 100 * time.Millisecond},
		{"fragmented.mp4", cat(video, []string{"-c:v", "mpeg4", "-b:v", "1M", "-c:a", "aac", "-movflags", "frag_keyframe+empty_moov"}), 0, 0},
		{"audio.m4a", cat(audio, []string{"-c:a", "aac"}), want, 100 * time.Millisecond},
		{"audio.flac", cat(audio, []string{"-c:a", "flac"}), want, time.Millisecond},
		{"tagged.flac", cat(audio, []string{"-c:a", "flac", "-metadata", "title=Test"}), want, time.Millisecond},
		{"vorbis.ogg", cat(audio, []string{"-c:a", "libvorbis"}), want, 10 * time.Millisecond},
		{"opus.ogg", cat(audio, []string{"-c:a", "libopus"}), want, 10 * time.Millisecond},
		{"flac.ogg", cat(audio, []string{"-c:a", "flac"}), want, 10 * time.Millisecond},
		{"vbr.mp3", cat(audio, []string{"-c:a", "libmp3lame", "-q:a", "4", "-metadata", "title=Test"}), want, 100 * time.Millisecond},
		{"cbr.mp3", cat(audio, []string{"-c:a", "libmp3lame", "-b:a", "128k", "-write_xing", "0", "-id3v2_version", "3", "-metadata", "title=Test"}), want, 200 * time.Millisecond},
		{"mono.mp3", cat(audio, []string{"-c:a", "libmp3lame", "-ac", "1", "-ar", "22050", "-b:a", "48k"}), want, 100 * time.Millisecond},
	}
}

func generate(t *testing.T) string {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	for _, f := range fixtures() {
		args := append([]string{"-hide_banner", "-loglevel", "error", "-y"}, f.args...)
		args = append(args, filepath.Join(dir, f.name))
		cmd := exec.Command(ffmpeg, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("skip %s: %s %s", f.name, err, out)
		}
	}
	return dir
}

func checkFixture(t *testing.T, f fixture, d time.Duration, err error) {
	t.Helper()
	if f.want == 0 {
		if err == nil {
			t.Errorf("%s: expected error, got %s", f.name, d)
		}
		return
	}
	if err != nil {
		t.Errorf("%s: %v", f.name, err)
		return
	}
	if diff := (d - f.want).Abs(); diff > f.tol {
		t.Errorf("%s: got %s want %s", f.name, d, f.want)
	}
}

func TestFiles(t *testing.T) {
	dir := generate(t)
	for _, f := range fixtures() {
		path := filepath.Join(dir, f.name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		u := &url.URL{Scheme: "file", Path: path}
		d, _, err := ProbeURL(context.Background(), nil, u, 0)
		checkFixture(t, f, d, err)
	}
}

func TestHTTP(t *testing.T) {
	dir := generate(t)
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	defer srv.Close()

	for _, f := range fixtures() {
		path := filepath.Join(dir, f.name)
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		u, _ := url.Parse(srv.URL + "/" + f.name)
		for _, size := range []int64{0, fi.Size()} {
			d, reads, err := ProbeURL(context.Background(), srv.Client(), u, size)
			checkFixture(t, f, d, err)
			if reads > 2 {
				t.Errorf("%s: %d reads", f.name, reads)
			}
		}
	}

	u, _ := url.Parse(srv.URL + "/missing.mkv")
	_, _, err := ProbeURL(context.Background(), srv.Client(), u, 1000)
	if !errors.Is(err, ErrNotFound) || Retryable(err) {
		t.Errorf("missing file should not be retryable, got %v", err)
	}
	secs, err := Seconds(context.Background(), srv.Client(), u, 1000)
	if secs != -1 || err != nil {
		t.Errorf("missing file got %d %v", secs, err)
	}

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer broken.Close()
	u, _ = url.Parse(broken.URL + "/video.mkv")
	secs, err = Seconds(context.Background(), broken.Client(), u, 1000)
	if secs != 0 || !Retryable(err) {
		t.Errorf("unavailable got %d %v", secs, err)
	}
}
