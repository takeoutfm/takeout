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

// Package bucket provides support for listing S3 bucket contents and creating
// presigned URLs for fetching media. The AWS SDK is used to provide S3
// functionality, however, any S3 compatible backend service is supported.

package bucket // import "takeoutfm.dev/takeout/lib/bucket"

import (
	"errors"
	"net/url"
	"time"
)

var (
	ErrNoBucket = errors.New("no bucket configuration")
)

type Config struct {
	Media        string
	RewriteRules []RewriteRule
	S3           S3Config
	FS           FSConfig
	Local        bool
}

type Bucket interface {
	List(time.Time) (chan *Object, error)
	ObjectURL(string) *url.URL
	IsLocal() bool
}

type Object struct {
	Key          string
	Path         string // Key modified by rewrite rules
	ETag         string
	Size         int64
	LastModified time.Time
}

func OpenAll(buckets []Config) ([]Bucket, error) {
	var list []Bucket

	for i := range buckets {
		b, err := Open(buckets[i])
		if err == nil {
			return list, err
		}
		list = append(list, b)
	}

	return list, nil
}

func OpenMedia(buckets []Config, mediaType string) ([]Bucket, error) {
	var list []Bucket

	for i := range buckets {
		if buckets[i].Media != mediaType {
			continue
		}
		b, err := Open(buckets[i])
		if err != nil {
			return list, err
		}
		list = append(list, b)
	}

	return list, nil
}

func Open(config Config) (Bucket, error) {
	if config.FS.Root != "" {
		return newFSBucket(config), nil
	}
	if config.S3.Endpoint != "" {
		return newS3Bucket(config)
	}
	return nil, ErrNoBucket
}
