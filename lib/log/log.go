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

package log // import "takeoutfm.dev/takeout/lib/log"

import (
	"io"
	l "log"
	"os"
)

const (
	FlagsNone = 0
	FlagsStd  = l.LstdFlags
	FlagsFile = l.Lshortfile
)

type logger interface {
	SetOutput(io.Writer)
	SetFlags(int)
	// Print followed by Panic
	Panicf(format string, v ...interface{})
	Panicln(v ...interface{})
	// Print followed by Exit
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})
	// Print
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

var Logger = defaultLogger()

func defaultLogger() logger {
	return l.New(os.Stdout, "", FlagsStd)
}

func SetOutput(w io.Writer) {
	Logger.SetOutput(w)
}

func SetFlags(flags int) {
	Logger.SetFlags(flags)
}

// Panic if err
func CheckError(err error) {
	if err != nil {
		Logger.Panicln(err)
	}
}

func Panicf(format string, v ...interface{}) {
	Logger.Panicf(format, v...)
}

func Panicln(v ...interface{}) {
	Logger.Panicln(v...)
}

func Fatalf(format string, v ...interface{}) {
	Logger.Fatalf(format, v...)
}

func Fatalln(v ...interface{}) {
	Logger.Fatalln(v...)
}

func Printf(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}

func Println(v ...interface{}) {
	Logger.Println(v...)
}
