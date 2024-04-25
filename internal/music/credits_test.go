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

package music

import (
	"testing"

	"github.com/takeoutfm/takeout/lib/search"
)

func TestAddField(t *testing.T) {
	fields := make(search.FieldMap)

	addField(fields, "test.string", "test string value")
	if fields["test.string"] != "test string value" {
		t.Error("expect string value")
	}

	addField(fields, "Test.STRING", "2nd string value")
	if len(fields["test.string"].([]string)) != 2 {
		t.Error("expect string array")
	}

	addField(fields, "test.string", "third string value")
	if len(fields["test.string"].([]string)) != 3 {
		t.Error("expect string array")
	}

	addField(fields, "test key", "test key value")
	if fields["test key"] != nil {
		t.Errorf("expect not found - got %s", fields["test key"])
	}
	if fields["test_key"] == "" {
		t.Error("expect with with underscore")
	}

	addField(fields, "test int", 123)
	if fields["test_int"] != 123 {
		t.Error("expect test int")
	}
	addField(fields, "test int", 345)
	if fields["test_int"] != 345 {
		t.Error("expect test int")
	}

	addField(fields, "slide guitar", "test artist")
	if fields["slide_guitar"] != "test artist" {
		t.Error("expect slide guitar")
	}
	if fields["guitar"] != "test artist" {
		t.Error("expect guitar")
	}
}
