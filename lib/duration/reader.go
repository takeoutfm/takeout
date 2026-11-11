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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ChunkSize is the minimum size of each remote read.
const ChunkSize = 64 * 1024

var (
	// ErrFetch wraps errors reading the file, as opposed to parsing it. These
	// are worth retrying later.
	ErrFetch   = errors.New("fetch")
	ErrNoRange = fmt.Errorf("%w: server does not support range requests", ErrFetch)
	// ErrNotFound is not retryable; the file is gone.
	ErrNotFound = errors.New("file not found")
)

// Retryable reports whether err came from reading the file rather than from
// its content, so probing again later may succeed.
func Retryable(err error) bool {
	return errors.Is(err, ErrFetch) || errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

type chunk struct {
	off  int64
	data []byte
}

// RangeReader is an io.ReaderAt over a remote URL that fetches data with HTTP
// range requests of at least ChunkSize and keeps what it has read.
type RangeReader struct {
	ctx    context.Context
	client *http.Client
	url    string
	size   int64
	chunks []chunk
	// Requests is the number of HTTP requests made.
	Requests int
}

// NewRangeReader returns a reader for u. If size is not positive, the first
// chunk is fetched to learn the size from the response.
func NewRangeReader(ctx context.Context, client *http.Client, u *url.URL, size int64) (*RangeReader, error) {
	r := &RangeReader{ctx: ctx, client: client, url: u.String(), size: size}
	if size <= 0 {
		r.size = -1
		if _, err := r.fetch(0, ChunkSize); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *RangeReader) Size() int64 {
	return r.size
}

func (r *RangeReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, ErrInvalid
	}
	if r.size >= 0 && off >= r.size {
		return 0, io.EOF
	}
	end := off + int64(len(p))
	if r.size >= 0 && end > r.size {
		end = r.size
	}
	for _, c := range r.chunks {
		if off >= c.off && end <= c.off+int64(len(c.data)) {
			n := copy(p, c.data[off-c.off:end-c.off])
			if n < len(p) {
				return n, io.EOF
			}
			return n, nil
		}
	}
	c, err := r.fetch(off, max(int(end-off), ChunkSize))
	if err != nil {
		return 0, err
	}
	n := copy(p, c.data)
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (r *RangeReader) fetch(off int64, n int) (chunk, error) {
	last := off + int64(n) - 1
	if r.size > 0 && last >= r.size {
		last = r.size - 1
	}
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return chunk{}, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, last))
	r.Requests++
	resp, err := r.client.Do(req)
	if err != nil {
		return chunk{}, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPartialContent:
		if r.size < 0 {
			r.size = contentRangeSize(resp.Header.Get("Content-Range"))
			if r.size < 0 {
				return chunk{}, ErrNoRange
			}
		}
	case http.StatusOK:
		// whole file; only usable for the start
		if off != 0 {
			return chunk{}, ErrNoRange
		}
		if r.size < 0 {
			r.size = resp.ContentLength
			if r.size < 0 {
				return chunk{}, ErrNoRange
			}
		}
	case http.StatusNotFound:
		return chunk{}, ErrNotFound
	case http.StatusRequestedRangeNotSatisfiable:
		return chunk{}, io.EOF
	default:
		return chunk{}, fmt.Errorf("%w: range request: %s", ErrFetch, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, last-off+1))
	if err != nil {
		return chunk{}, fmt.Errorf("%w: %w", ErrFetch, err)
	}
	c := chunk{off: off, data: data}
	r.chunks = append(r.chunks, c)
	return c, nil
}

// contentRangeSize parses the total from "bytes 0-99/1234".
func contentRangeSize(v string) int64 {
	i := strings.LastIndexByte(v, '/')
	if i < 0 {
		return -1
	}
	n, err := strconv.ParseInt(v[i+1:], 10, 64)
	if err != nil {
		return -1
	}
	return n
}

// ProbeURL returns the duration of the media at u, which may be an http(s)
// URL such as a presigned S3 URL, or a file URL. Size is the size of the file
// if known, otherwise 0. It also returns the number of reads made.
func ProbeURL(ctx context.Context, client *http.Client, u *url.URL, size int64) (time.Duration, int, error) {
	switch u.Scheme {
	case "file":
		path := u.Path
		if u.Host != "" {
			path = u.Host + u.Path
		}
		f, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, ErrNotFound
		} else if err != nil {
			return 0, 0, fmt.Errorf("%w: %w", ErrFetch, err)
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			return 0, 0, fmt.Errorf("%w: %w", ErrFetch, err)
		}
		d, err := Probe(f, fi.Size())
		return d, 0, err
	case "http", "https":
		r, err := NewRangeReader(ctx, client, u, size)
		if err != nil {
			return 0, 1, err
		}
		d, err := Probe(r, r.Size())
		return d, r.Requests, err
	}
	return 0, 0, fmt.Errorf("unsupported scheme %q", u.Scheme)
}

// Result is the outcome of scanning one file.
type Result struct {
	Name    string
	ETag    string
	Seconds int // > 0 duration, -1 unreadable
}

// Seconds probes the media at u and returns its duration in whole seconds,
// rounded up. It returns -1 for files whose duration can't be read, which
// aren't worth trying again, and an error when it may work later.
func Seconds(ctx context.Context, client *http.Client, u *url.URL, size int64) (int, error) {
	d, _, err := ProbeURL(ctx, client, u, size)
	if err != nil {
		if Retryable(err) {
			return 0, err
		}
		return -1, nil
	}
	secs := int((d + time.Second - 1) / time.Second)
	return max(secs, 1), nil
}
