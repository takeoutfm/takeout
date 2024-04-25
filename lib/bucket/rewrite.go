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

package bucket

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/takeoutfm/takeout/lib/log"
)

type RewriteRule struct {
	Pattern string
	Replace string
}

func rewrite(rules []RewriteRule, path string) string {
	result := path
	for _, rule := range rules {
		re := regexp.MustCompile(rule.Pattern)
		matches := re.FindStringSubmatch(result)
		if matches != nil {
			result = rule.Replace
			for i := range matches {
				result = strings.ReplaceAll(result, fmt.Sprintf("$%d", i), matches[i])
			}
		}
	}
	if result != path {
		log.Printf("rewrite %s -> %s\n", path, result)
	}
	return result
}
