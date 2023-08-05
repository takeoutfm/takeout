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

package client

type M map[string]interface{}
type L []string

func patchAppend(ref string) []M {
	return []M{{
		"op":    "add",
		"path":  "/playlist/track/-",
		"value": M{"$ref": ref},
	}}
}

func patchClear() []M {
	return []M{{
		"op":    "replace",
		"path":  "/playlist/track",
		"value": L{},
	}}
}

func patchPosition(index int, position float64) []M {
	return []M{
		{
			"op":    "replace",
			"path":  "/index",
			"value": index,
		},
		{
			"op":    "replace",
			"path":  "/position",
			"value": position,
		},
	}
}

func patchReplace(ref, spiffType, creator, title string) []M {
	var list []M
	return append(list,
		M{"op": "replace", "path": "/index", "value": "0"},
		M{"op": "replace", "path": "/position", "value": "0"},
		M{"op": "replace", "path": "/type", "value": spiffType},
		M{"op": "replace", "path": "/playlist/creator", "value": creator},
		M{"op": "replace", "path": "/playlist/title", "value": title},
		M{"op": "replace", "path": "/playlist/track", "value": L{}},
		M{"op": "add", "path": "/playlist/track/-", "value": M{"$ref": ref}})
}
