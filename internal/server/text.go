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

package server

import (
	"html"
	"regexp"
	"strings"
)

var (
	reScriptStyle = regexp.MustCompile(`(?is)<\s*(script|style)\b.*?<\s*/\s*(script|style)\s*>`)
	reLineBreak   = regexp.MustCompile(`(?i)<\s*(br\s*/?|/p|/div|/li|/h[1-6]|/blockquote|/tr)\s*>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]*>`)
	reBlankLines  = regexp.MustCompile(`\n[ \t]*(\n[ \t]*)+`)
)

// plainText converts untrusted HTML (such as a podcast description from a
// third party feed) to plain text for display.
//
// The result is an ordinary string and is NOT marked as safe HTML, so
// html/template escapes it on output. That makes this safe by construction:
// even if the tag stripping below misses something, whatever is left is
// rendered as visible text and can never run as markup or script.
func plainText(s string) string {
	s = reScriptStyle.ReplaceAllString(s, "")
	s = reLineBreak.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = reBlankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
