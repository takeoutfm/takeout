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

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"takeoutfm.dev/takeout/internal/auth"
)

func TestCodeGetExpiresIn(t *testing.T) {
	ctx := NewTestContext(t)
	req := httptest.NewRequest("GET", "/api/code", nil)
	w := httptest.NewRecorder()
	apiCodeGet(w, withContext(req, ctx))
	if w.Code != http.StatusOK {
		t.Fatalf("got status %d", w.Code)
	}

	var resp codeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code == "" || resp.AccessToken == "" {
		t.Fatalf("missing code or token: %+v", resp)
	}

	// never longer than the code or its token lives
	limit := min(ctx.Config().Auth.CodeAge, ctx.Config().Auth.CodeToken.Age)
	if resp.ExpiresIn <= 0 || time.Duration(resp.ExpiresIn)*time.Second > limit {
		t.Fatalf("ExpiresIn %d outside (0, %v]", resp.ExpiresIn, limit)
	}
	if resp.Interval != int(auth.CodePollInterval/time.Second) {
		t.Fatalf("Interval %d, want %v", resp.Interval, auth.CodePollInterval)
	}
}
