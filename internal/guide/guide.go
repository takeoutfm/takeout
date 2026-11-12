// Copyright 2026 defsub
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

// Package guide schedules TV-style channels of movies and shows. A channel
// is only a schedule: tuning in plays the existing file at an offset, so
// nothing is broadcast and an unwatched channel costs nothing.
package guide

import (
	"errors"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"takeoutfm.dev/takeout/internal/config"
	. "takeoutfm.dev/takeout/model"
)

type Guide struct {
	config *config.Config
	db     *gorm.DB
}

func NewGuide(config *config.Config) *Guide {
	return &Guide{config: config}
}

func (g *Guide) Open() error {
	cfg := g.config.Guide.DB.GormConfig()
	var err error
	switch g.config.Guide.DB.Driver {
	case "sqlite3":
		g.db, err = gorm.Open(sqlite.Open(g.config.Guide.DB.Source), cfg)
	default:
		err = errors.New("driver not supported")
	}
	if err != nil {
		return err
	}
	return g.db.AutoMigrate(&Channel{}, &Program{}, &SpecialAiring{}, &FeatureTitle{})
}

func (g *Guide) Close() {
	if g.db == nil {
		return
	}
	conn, err := g.db.DB()
	if err != nil {
		return
	}
	conn.Close()
}

// Horizon is how far ahead the schedule is built.
func (g *Guide) Horizon() time.Duration {
	return g.config.Guide.Horizon
}

func (g *Guide) Location() *time.Location {
	return g.config.Guide.Location()
}

// Resolver finds the current library rows for scheduled titles.
type Resolver interface {
	Movie(tmid int64) (Movie, bool)
	Episode(tvid int64, season, episode int) (TVSeries, TVEpisode, bool)
	Track(rid string) (Track, bool)
}

// Entry is a scheduled program with the movie or episode it plays. Both are
// nil when the title has left the library.
type Entry struct {
	Program
	Movie     *Movie
	Series    *TVSeries
	TVEpisode *TVEpisode
	Track     *Track // a break track
	// for a reserved feature slot, the feature's Description
	Description string
}

// OffAir reports whether the title is no longer in the library.
func (e Entry) OffAir() bool {
	return e.Kind != ProgramReserved && e.Movie == nil && e.TVEpisode == nil && e.Track == nil
}

// Listing is a channel with its programs in a time range.
type Listing struct {
	Channel  Channel
	Programs []Entry
}

// Listings returns the lineup channels with programs that overlap [start, end),
// in lineup order, resolved with r. Channels with nothing in the range are left
// out. A nil resolver leaves every entry unresolved.
func (g *Guide) Listings(start, end time.Time, r Resolver) []Listing {
	var result []Listing
	for _, cc := range g.config.Guide.Lineup {
		ch, err := g.findChannel(cc.Name)
		if err != nil {
			continue
		}
		programs := g.programs(ch.ID, start, end)
		if len(programs) == 0 {
			continue
		}
		listing := Listing{Channel: ch}
		for _, p := range programs {
			e := resolve(p, r)
			if p.Kind == ProgramReserved {
				for _, f := range cc.Features {
					if f.Name == p.Block {
						e.Description = f.Description
					}
				}
			}
			listing.Programs = append(listing.Programs, e)
		}
		result = append(result, listing)
	}
	return result
}

func resolve(p Program, r Resolver) Entry {
	e := Entry{Program: p}
	if r == nil {
		return e
	}
	switch p.Kind {
	case ProgramMovie:
		if m, ok := r.Movie(p.TMID); ok {
			e.Movie = &m
		}
	case ProgramEpisode:
		if s, ep, ok := r.Episode(p.TVID, p.Season, p.Episode); ok {
			e.Series, e.TVEpisode = &s, &ep
		}
	case ProgramTrack:
		if t, ok := r.Track(p.RID); ok {
			e.Track = &t
		}
	}
	return e
}

// HasGuide reports whether any lineup channel has programs from now on.
func (g *Guide) HasGuide() bool {
	if len(g.config.Guide.Lineup) == 0 {
		return false
	}
	var names []string
	for _, cc := range g.config.Guide.Lineup {
		names = append(names, cc.Name)
	}
	var count int64
	g.db.Model(&Program{}).
		Where("end_at > ? and channel_id in (select id from channels where name in (?) and deleted_at is null)",
			time.Now().UTC(), names).
		Limit(1).Count(&count)
	return count > 0
}

func (g *Guide) findChannel(name string) (Channel, error) {
	var ch Channel
	err := g.db.First(&ch, "name = ?", name).Error
	return ch, err
}

// ensureChannel returns the row for name, creating it if needed.
func (g *Guide) ensureChannel(name string) (Channel, error) {
	ch, err := g.findChannel(name)
	if err == nil {
		return ch, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ch, err
	}
	ch = Channel{Name: name}
	err = g.db.Create(&ch).Error
	return ch, err
}

// programs returns a channel's programs that overlap [start, end).
func (g *Guide) programs(channelID uint, start, end time.Time) []Program {
	var list []Program
	g.db.Where("channel_id = ? and start_at < ? and end_at > ?", channelID, end.UTC(), start.UTC()).
		Order("start_at").Find(&list)
	return list
}

// history returns all of a channel's programs, in order.
func (g *Guide) history(channelID uint) []Program {
	var list []Program
	g.db.Where("channel_id = ?", channelID).Order("start_at").Find(&list)
	return list
}

// airing returns all programs that end after t, on any channel.
func (g *Guide) airing(t time.Time) []Program {
	var list []Program
	g.db.Where("end_at > ?", t.UTC()).Order("channel_id, start_at").Find(&list)
	return list
}

func (g *Guide) createPrograms(list []Program) error {
	if len(list) == 0 {
		return nil
	}
	return g.db.CreateInBatches(list, 200).Error
}

// deleteEnded removes programs that ended before t.
func (g *Guide) deleteEnded(t time.Time) error {
	return g.db.Unscoped().Where("end_at < ?", t.UTC()).Delete(&Program{}).Error
}

// specialAirings returns a channel's specials, in start order.
func (g *Guide) specialAirings(channelID uint) []SpecialAiring {
	var list []SpecialAiring
	g.db.Where("channel_id = ?", channelID).Order("start_at").Find(&list)
	return list
}

// specialNear reports whether a special named name airs on another channel
// within d of t.
func (g *Guide) specialNear(name string, channelID uint, t time.Time, d time.Duration) bool {
	var count int64
	g.db.Model(&SpecialAiring{}).
		Where("name = ? and channel_id <> ? and start_at > ? and start_at < ?",
			name, channelID, t.Add(-d).UTC(), t.Add(d).UTC()).
		Count(&count)
	return count > 0
}

func (g *Guide) createSpecialAiring(a *SpecialAiring) error {
	return g.db.Create(a).Error
}

// reservedBefore returns a channel's reserved feature slots starting before t.
func (g *Guide) reservedBefore(channelID uint, t time.Time) []Program {
	var list []Program
	g.db.Where("channel_id = ? and kind = ? and start_at < ?", channelID, ProgramReserved, t.UTC()).
		Order("start_at").Find(&list)
	return list
}

// featureKeys returns the keys of the titles a feature has shown on a channel.
func (g *Guide) featureKeys(channelID uint, feature string) map[string]bool {
	var list []FeatureTitle
	g.db.Where("channel_id = ? and feature = ?", channelID, feature).Find(&list)
	keys := make(map[string]bool)
	for _, f := range list {
		keys[f.Key] = true
	}
	return keys
}

// replaceReserved swaps a reserved slot for its programs, recording the titles
// the feature showed.
func (g *Guide) replaceReserved(reserved Program, programs []Program, shown []FeatureTitle) error {
	return g.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Delete(&Program{}, reserved.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// filled elsewhere already
			return nil
		}
		if len(programs) > 0 {
			if err := tx.CreateInBatches(programs, 200).Error; err != nil {
				return err
			}
		}
		if len(shown) > 0 {
			if err := tx.Create(&shown).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// hasReservedBefore reports whether any reserved slot starts before t.
func (g *Guide) hasReservedBefore(t time.Time) bool {
	var count int64
	g.db.Model(&Program{}).Where("kind = ? and start_at < ?", ProgramReserved, t.UTC()).Count(&count)
	return count > 0
}
