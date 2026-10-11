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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"takeoutfm.dev/takeout/internal/config"
)

// Run with -race. Before the fix mediaConfig wrote to the shared cached
// config on every call and config.LoadConfig used an unguarded map.
func TestMediaConfigConcurrent(t *testing.T) {
	dir := t.TempDir()
	root := &config.Config{}
	root.Server.MediaDir = dir
	root.Server.Listen = "127.0.0.1:1234"

	names := []string{"a", "b", "c"}
	for _, n := range names {
		d := filepath.Join(dir, n)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "config.yaml"), []byte("Server:\n  Listen: ignored\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	first := make(map[string]*config.Config)
	for _, n := range names {
		c, err := mediaConfig(root, n)
		if err != nil {
			t.Fatal(err)
		}
		first[n] = c
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				n := names[(i+j)%len(names)]
				c, err := mediaConfig(root, n)
				if err != nil {
					t.Error(err)
					return
				}
				// cached: same pointer every time, server config merged
				if c != first[n] {
					t.Errorf("%s: config was rebuilt", n)
				}
				if c.Server.Listen != root.Server.Listen {
					t.Errorf("%s: server config not merged", n)
				}
			}
		}(i)
	}
	wg.Wait()
}
