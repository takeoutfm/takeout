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
	"html/template"
	"strings"
	"testing"
)

func TestPlainText(t *testing.T) {
	tests := []struct{ in, want string }{
		{"hello", "hello"},
		{"<p>one</p><p>two</p>", "one\ntwo"},
		{"line<br>break<br/>again", "line\nbreak\nagain"},
		{`<a href="https://example.com">link text</a>`, "link text"},
		{"<script>alert(1)</script>after", "after"},
		{"<SCRIPT SRC=x></SCRIPT>after", "after"},
		{"<style>body{}</style>after", "after"},
		{"fish &amp; chips", "fish & chips"},
		{"a\n\n\n\n\nb", "a\n\nb"},
		{"  padded  ", "padded"},
	}
	for _, tc := range tests {
		if got := plainText(tc.in); got != tc.want {
			t.Errorf("plainText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Whatever plainText returns must come out of html/template with no live
// markup, including payloads that survive tag stripping.
func TestPlainTextTemplateOutputIsInert(t *testing.T) {
	tmpl := template.Must(template.New("t").Funcs(template.FuncMap{
		"plainText": plainText,
	}).Parse(`<div>{{ . | plainText }}</div>`))

	payloads := []string{
		`<script>alert(1)</script>`,
		`<img src=x onerror=alert(1)>`,
		`<svg/onload=alert(1)>`,
		`<scr<script>ipt>alert(1)</scr</script>ipt>`,
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`&amp;lt;img src=x onerror=alert(1)&amp;gt;`,
		`<a href="javascript:alert(1)">x</a>`,
		`<<script>script>alert(1)`,
		`<script`,
		`< img src=x onerror=alert(1) `,
		`"><script>alert(1)</script>`,
	}
	for _, p := range payloads {
		var sb strings.Builder
		if err := tmpl.Execute(&sb, p); err != nil {
			t.Fatal(err)
		}
		out := sb.String()
		inner := strings.TrimSuffix(strings.TrimPrefix(out, "<div>"), "</div>")
		if strings.ContainsAny(inner, "<>") {
			t.Errorf("payload %q produced live markup: %q", p, out)
		}
	}
}
