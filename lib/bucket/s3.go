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
	"context"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	config2 "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	config  Config
	s3      *s3.Client
	presign *s3.PresignClient
}

func newS3Bucket(ctx context.Context, config Config) (*s3bucket, error) {
	cfg, err := config2.LoadDefaultConfig(ctx,
		config2.WithRegion(config.S3.Region),
		config2.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				config.S3.AccessKeyID,
				config.S3.SecretAccessKey,
				"", // only needed for temporary AWS credentials
			),
		),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(config.S3.Endpoint)
		o.UsePathStyle = true
	})

	bucket := &s3bucket{
		config:  config,
		s3:      client,
		presign: s3.NewPresignClient(client),
	}

	return bucket, nil
}

func (b *s3bucket) IsLocal() bool {
	return b.config.Local
}

func (b *s3bucket) List(ctx context.Context, lastSync time.Time) (objectCh chan *Object, err error) {
	objectCh = make(chan *Object)
	go func() {
		defer close(objectCh)
		paginator := s3.NewListObjectsV2Paginator(b.s3, &s3.ListObjectsV2Input{
			Bucket: aws.String(b.config.S3.BucketName),
			Prefix: aws.String(b.config.S3.ObjectPrefix),
		})
		for paginator.HasMorePages() {
			resp, err := paginator.NextPage(ctx)
			if err != nil {
				break
			}
			for _, obj := range resp.Contents {
				if obj.LastModified != nil &&
					obj.LastModified.After(lastSync) {
					objectCh <- &Object{
						Key:          aws.ToString(obj.Key),
						Path:         rewrite(b.config.RewriteRules, aws.ToString(obj.Key)),
						ETag:         aws.ToString(obj.ETag),
						Size:         *obj.Size,
						LastModified: *obj.LastModified,
					}

				}
			}
		}
	}()
	return
}

// Generate a presigned url which expires based on config settings.
func (b *s3bucket) ObjectURL(ctx context.Context, key string) *url.URL {
	req, _ := b.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.config.S3.BucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = b.config.S3.URLExpiration
	})
	url, _ := url.Parse(req.URL)
	return url
}
