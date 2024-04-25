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

package client

import (
	"strings"
	"testing"
)

func TestPatchAppend(t *testing.T) {
	patch := patchAppend("test-ref")
	if len(patch) != 1 {
		t.Fatal("expect patches")
	}
	for _, p := range patch {
		if p["op"] != "add" {
			t.Error("expect op add")
		}
		if strings.HasSuffix(p["path"].(string), "/-") == false {
			t.Error("expect dash")
		}
		if p["value"].(M)["$ref"] != "test-ref" {
			t.Error("expect ref")
		}
	}
}

func TestPatchClear(t *testing.T) {
	patch := patchClear()
	if len(patch) != 1 {
		t.Fatal("expect patches")
	}
	for _, p := range patch {
		if p["op"] != "replace" {
			t.Error("expect op replace")
		}
		if p["path"] != "/playlist/track" {
			t.Error("expect playlist")
		}
		if len(p["value"].(L)) != 0 {
			t.Error("expect empty list")
		}
	}

}

func TestPatchPosition(t *testing.T) {
	patch := patchPosition(0, 0.1)
	if len(patch) != 2 {
		t.Fatal("expect patches")
	}

	if patch[0]["op"] != "replace" {
		t.Error("expect op replace")
	}
	if patch[0]["path"] != "/index" {
		t.Error("expect index")
	}
	if patch[0]["value"] != 0 {
		t.Error("expect index")
	}

	if patch[1]["op"] != "replace" {
		t.Error("expect op replace")
	}
	if patch[1]["path"] != "/position" {
		t.Error("expect position")
	}
	if patch[1]["value"] != 0.1 {
		t.Error("expect pos")
	}

}

func TestPatchReplace(t *testing.T) {
	patch := patchReplace("test ref", "test spiff", "test creator", "test title")
	if len(patch) != 7 {
		t.Fatal("expect patches")
	}
}
