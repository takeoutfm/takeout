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
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	ObjectPrefix    string
	URLExpiration   time.Duration
}

type s3bucket struct {
	config Config
	s3     *s3.S3
}

func newS3Bucket(config Config) (*s3bucket, error) {
	creds := credentials.NewStaticCredentials(
		config.S3.AccessKeyID,
		config.S3.SecretAccessKey, "")
	s3Config := &aws.Config{
		Credentials:      creds,
		Endpoint:         aws.String(config.S3.Endpoint),
		Region:           aws.String(config.S3.Region),
		S3ForcePathStyle: aws.Bool(true)}
	session, err := session.NewSession(s3Config)
	if err != nil {
		return nil, err
	}
	bucket := &s3bucket{
		config: config,
		s3:     s3.New(session),
	}
	return bucket, nil
}
func (b *s3bucket) IsLocal() bool {
	return b.config.Local
}

func (b *s3bucket) List(lastSync time.Time) (objectCh chan *Object, err error) {
	objectCh = make(chan *Object)

	go func() {
		defer close(objectCh)

		var continuationToken *string
		continuationToken = nil
		for {
			req := s3.ListObjectsV2Input{
				Bucket: aws.String(b.config.S3.BucketName),
				Prefix: aws.String(b.config.S3.ObjectPrefix)}
			if continuationToken != nil {
				req.ContinuationToken = continuationToken
			}
			resp, err := b.s3.ListObjectsV2(&req)
			if err != nil {
				break
			}
			for _, obj := range resp.Contents {
				if obj.LastModified != nil &&
					obj.LastModified.After(lastSync) {
					objectCh <- &Object{
						Key:          *obj.Key,
						Path:         rewrite(b.config.RewriteRules, *obj.Key),
						ETag:         *obj.ETag,
						Size:         *obj.Size,
						LastModified: *obj.LastModified,
					}
				}
			}
			if !*resp.IsTruncated {
				break
			}
			continuationToken = resp.NextContinuationToken
		}
	}()

	return
}

// Generate a presigned url which expires based on config settings.
func (b *s3bucket) ObjectURL(key string) *url.URL {
	req, _ := b.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(b.config.S3.BucketName),
		Key:    aws.String(key)})
	urlStr, _ := req.Presign(b.config.S3.URLExpiration)
	url, _ := url.Parse(urlStr)
	return url
}
