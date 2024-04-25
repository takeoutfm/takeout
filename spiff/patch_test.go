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

package spiff

import (
	"strings"
	"testing"
)

func TestPatch(t *testing.T) {
	data := `{"playlist":{"track":[]}}`
	patch := `[{"op":"add","path":"/playlist/track/-","value":{"title":"test title"}}]`
	result, err := Patch([]byte(data), []byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result), `"test title"`) == false {
		t.Error("expect add test title")
	}

	patch = `[{"op":"replace","path":"/playlist/track","value":[]}]`
	result, err = Patch([]byte(data), []byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result), `"test title"`) == true {
		t.Error("expect remove test title")
	}
}


func TestCompare(t *testing.T) {
	p1 := `{"playlist":{"track":[{"title":"test title1"}]}}`
	p2 := `{"playlist":{"track":[{"title":"test title2"}]}}`

	if v, _ := Compare([]byte(p1), []byte(p1)); v == false {
		t.Error("expect p1,p1 same")
	}
	if v, _ := Compare([]byte(p2), []byte(p2)); v == false {
		t.Error("expect p2,p2 same")
	}
	if v, _ := Compare([]byte(p1), []byte(p2)); v == true {
		t.Error("expect p1,p2 different")
	}
	if v, _ := Compare([]byte(p2), []byte(p1)); v == true {
		t.Error("expect p2,p1 different")
	}
}
