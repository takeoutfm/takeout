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

package main

import (
	"fmt"
	"github.com/takeoutfm/takeout/client/api"
	"github.com/spf13/cobra"
)

var playlistCmd = &cobra.Command{
	Use:   "playlist",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		return playlist()
	},
}

func playlist() error {
	playout := NewPlayout()
	playlist, err := api.Playlist(playout)
	if err != nil {
		return err
	}
	for _, t := range playlist.Spiff.Entries {
		fmt.Printf("%s / %s / %s / %s / %d\n", t.Creator, t.Album, t.Title, t.Location[0], t.Size[0])
	}
	return nil
}

func init() {
	rootCmd.AddCommand(playlistCmd)
}
