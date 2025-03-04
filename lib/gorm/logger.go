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

package gorm // import "takeoutfm.dev/takeout/lib/gorm"

import (
	"time"

	"gorm.io/gorm/logger"
	"takeoutfm.dev/takeout/lib/log"
)

var (
	DebugLogger = logger.New(log.Logger,
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		})

	DefaultLogger = logger.New(log.Logger,
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		})
)

func Logger(v string) logger.Interface {
	switch v {
	case "debug":
		return DebugLogger
	case "discard":
		return logger.Discard
	default:
		//return DebugLogger
		return DefaultLogger
	}
}
