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

package main

import (
	"context"
	"fmt"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "client",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		client()
	},
}

func client() {
	ctx := context.Background()
	fmt.Println("client")
	conn, _, _, err := ws.DefaultDialer.Dial(ctx, "wss://takeout.fm/live")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = wsutil.WriteClientText(conn, []byte(`/auth b0a3e836-ccae-4628-b6a9-0d2548327a4c`))
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		msg, _, err := wsutil.ReadServerData(conn)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("got: %s\n", string(msg))
	}
}

func init() {
	rootCmd.AddCommand(clientCmd)
}
