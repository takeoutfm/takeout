// Copyright 2024 defsub
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

// Package bucket provides support for listing S3 bucket contents and creating
// presigned URLs for fetching media. The AWS SDK is used to provide S3
// functionality, however, any S3 compatible backend service is supported.

package bucket // import "takeoutfm.dev/takeout/lib/bucket"

import (
	"net/url"
	"os"
	"path/filepath"
	"time"

	"takeoutfm.dev/takeout/lib/hash"
	"takeoutfm.dev/takeout/lib/log"
)

type FSConfig struct {
	Root string
}

type fileBucket struct {
	config Config
}

func newFSBucket(config Config) *fileBucket {
	return &fileBucket{config: config}
}

func (f *fileBucket) IsLocal() bool {
	return true //f.config.Local
}

func (f *fileBucket) List(lastSync time.Time) (objectCh chan *Object, err error) {
	objectCh = make(chan *Object)

	walk := func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.ModTime().After(lastSync) {
				var etag string
				etag, err = hash.MD5Sum(path)
				if err != nil {
					log.Printf("etag %s: %s\n", path, err)
				} else {
					objectCh <- &Object{
						Key:          path,
						Path:         rewrite(f.config.RewriteRules, path),
						ETag:         etag,
						Size:         info.Size(),
						LastModified: info.ModTime(),
					}
				}
			}
		}
		return err
	}

	go func() {
		defer close(objectCh)
		err = filepath.WalkDir(f.config.FS.Root, walk)
	}()

	return
}

func (fileBucket) ObjectURL(key string) *url.URL {
	url, err := url.Parse("file://" + key)
	log.CheckError(err)
	return url
}
