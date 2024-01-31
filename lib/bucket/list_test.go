// Copyright 2023 defsub
//
// This file is part of Takeout.
//
// Takeout is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// Takeout is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Takeout.  If not, see <https://www.gnu.org/licenses/>.

package bucket

import (
	"testing"
)

func TestRewrite(t *testing.T) {
	rules := []RewriteRule{
		// test fixing artist & release names
		{Pattern: "^(.+/)Dr. Octagon(/Dr. Octagon, Part II.+/.+)$", Replace: "$1Kool Keith$2"},
		{Pattern: "^(.+/White Zombie/La Sexorcisto_ Devil Music, )Volume One(.+/.+)$", Replace: "$1Vol. 1$2"},
		{Pattern: "^(.+/)Gary Numan(/Premier Hits \\([0-9]+\\)/.+)$", Replace: "$1Tubeway Army$2"},
	}

	b := Bucket{config: Config{RewriteRules: rules}}

	if b.Rewrite("/bucket/Unchanged Artist/Unchanged Album (2022)/Unchanged Song.flac") !=
		"/bucket/Unchanged Artist/Unchanged Album (2022)/Unchanged Song.flac" {
		t.Error("expect unchanged")
	}
	if b.Rewrite("/bucket/Music/Dr. Octagon/Dr. Octagon, Part II (2004)/1-Song.flac") !=
		"/bucket/Music/Kool Keith/Dr. Octagon, Part II (2004)/1-Song.flac" {
		t.Error("expect kool keith")
	}
	if b.Rewrite("/bucket/Music/White Zombie/La Sexorcisto_ Devil Music, Volume One (1992)/1-Dragula.flac") !=
		"/bucket/Music/White Zombie/La Sexorcisto_ Devil Music, Vol. 1 (1992)/1-Dragula.flac" {
		t.Error("expect vol 1")
	}
	if b.Rewrite("/bucket/Music/Gary Numan/Premier Hits (1996)/09-Foo.flac") !=
		"/bucket/Music/Tubeway Army/Premier Hits (1996)/09-Foo.flac" {
		t.Error("expect tubeway army")
	}

}

func TestRewrite2(t *testing.T) {
	rules := []RewriteRule{
		// test stacking
		{Pattern: "^(.+/)Artist/Album(/.+)$", Replace: "$1Artist X/Album X$2"},
		{Pattern: "^(.+/)Artist X/Album X(/.+)$", Replace: "$1Artist Y/Album Y$2"},
		{Pattern: "^(.+/)Artist Y/Album Y(/.+)$", Replace: "$1Artist Z/Album Z$2"},
	}

	b := Bucket{config: Config{RewriteRules: rules}}

	if b.Rewrite("/bucket/Music/Artist/Album/1-Track.flac") !=
		"/bucket/Music/Artist Z/Album Z/1-Track.flac" {
	}
}
