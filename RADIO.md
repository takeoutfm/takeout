# Radio

Status: phase 1 (server) is implemented; the apps are not. See "Implementation notes" at the
end for where the code differs from this text.

Live radio from the library. A live station is a schedule of tracks, and tuning in means playing
the track that's on now from `now - track.start`. Everyone hears the same thing and joins
mid-song, like real radio, and nothing plays on the server.

Live stations supersede the generated stations (genre, decade, series and other). Afterwards
`/api/radio` has two kinds of station: **live**, from the library, and **stream**, internet
radio from `RadioStreams`, unchanged.

Radio is separate from the TV guide ([CHANNELS.md](CHANNELS.md)). `/api/guide` is for video and
`/api/radio` is for music. The two share ideas but no code: the guide is mostly about fitting
movies and episodes into aligned slots, specials and features, none of which radio needs.

## Why not ICY

An ICY (Shoutcast/Icecast) stream from the server was considered and rejected. ICY is one
continuous encoded stream, and FLAC files can't be joined into one that radio players accept, so
the server would have to decode and re-encode the library, normally to MP3. That means ffmpeg
(or a Go encoder), a process per station while anyone listens, and lossy audio from lossless
files.

Without a stream, only Takeout's apps can play live stations properly, because only they know how
to follow the schedule. Other players get the playlist fallback (see "API").

## What changes from today

Today `CreateStations` (the `stations` job) generates shared stations from search refs:

- one per genre in `RadioGenres`, or every artist genre in the library when that's unset;
- seven decades, 1960s to 2020s;
- one per `RadioSeries` entry, and one per `RadioOther` entry.

Each plays a fresh shuffled playlist of up to `RadioLimit` tracks, with skipping. Users can't
create stations: the POST, PUT, PATCH and DELETE handlers in `api.go` aren't routed. Artist and
track radio are ad-hoc `&radio=1` search refs, not stations, and aren't affected.

Live stations replace the generated ones, and there's no migration: the generated stations and
their `Music.RadioGenres`, `RadioSeries` and `RadioOther` settings go away, and live stations
come only from the `Radio:` section. A server with no `Radio:` stations has only stream radio.

**The deliberate loss is skipping.** Genre and decade radio no longer skip or reshuffle. That's
the point of real radio, but it's a change. Artist and track radio stay on-demand.

## Configuration

A `Radio:` section per media collection, like `Guide:`:

```yaml
Radio:
  Timezone: America/Los_Angeles
  Genres: [jazz, rock]
  Decades: [1960, 1970, 1980, 1990, 2000, 2010, 2020]
  Series: ["Billboard Hot 100"]
  Lineup:
    - {Name: Jazz FM, Music: "genre:jazz", Image: "https://…", Description: "Jazz all day"}
    - {Name: Christmas Radio, Layout: Jan, Match: Dec, Music: "genre:christmas"}
```

| Setting | Default | Meaning |
|---|---|---|
| `Timezone` | server local | For `Layout`/`Match` windows. |
| `SyncInterval` | `1h` | How often the radio job runs. `0` disables it. |
| `Horizon` | `24h` | How far ahead schedules are built. Nobody browses radio a week ahead, and a short horizon keeps the table small: 40 stations at about 4 minutes a track is about 15,000 rows. |
| `Genres` | none | Automatic genre stations, with today's query (`+genre:"…" +type:single +popularity:<11 -artist:"Various Artists"`). |
| `Decades` | none | Automatic decade stations ("1980s Top Tracks"), with today's query. |
| `Series` | none | Automatic series (chart) stations. |
| `Lineup` | none | Hand-written stations: `Name`, `Music` query, optional `Layout`/`Match`, `Image`, `Description`, `MinContent`. |
| `MinContent` | `2h` | A station whose pool is shorter than this is skipped and logged. Can be set per station. |
| `SearchLimit` | `1000` | Pool search limit. |

**Order:** `Lineup` first in its order, then genres, decades and series.

**No migration:** `Music.RadioGenres`, `RadioSeries` and `RadioOther` are gone and aren't read.
`RadioStreams` stays where it is, and `RadioLimit` and `RadioSearchLimit` stay for artist and
track radio.

## Schedule

The radio job, every `SyncInterval`:

1. Works out the stations: the lineup entries in season, plus the automatic ones.
2. Creates or updates a shared `Station` row of type `live` for each, and deletes `live` rows for
   stations that are gone. The `Station` row is what `/api/radio` lists; the schedule is keyed by
   station name, like guide channels, so renaming a station starts a fresh schedule.
3. Extends each station's schedule to `Horizon`.
4. Deletes tracks that ended more than an hour ago.

Tracks run back to back with no alignment, each `Start` the previous `End`. After downtime a
schedule restarts at now rather than filling the past.

**Pool:** tracks matching the query (the same `Music.Search` as today's refs) that have a recording
ID (`RID`, the schedule key) and a scanned `Duration`, the same rule as break tracks.

**Picker:** a pure function of pool, recent history and seed (station name and start time), so it
can be unit tested without a database. Rules, in priority order:

1. **No repeats:** a track isn't repeated within the larger of 200 tracks and half the pool,
   but always leaving a quarter of the pool to choose from.
2. **Artist separation:** not the same artist within 4 tracks, or within an hour when the pool
   allows.
3. **Release separation:** two tracks from the same release are never back to back.

**Small pools:** hit-only queries can match only 50–150 tracks, which is a few hours of music. When
nothing satisfies all the rules, relax them from the bottom up rather than leave silence. A
24-hour schedule from a small pool does repeat; that's expected, and is why `MinContent` is lower
than the guide's.

**Not shared with the guide:** the picker follows the shape of the guide's break picker (the
`pick` closure in `internal/guide/breaks.go`: a seeded shuffle, skipping recently played tracks)
but lives in `internal/radio` on its own. The two packages don't import each other, and the
guide's breaks stay as they are; they're short enough that artist separation hardly shows.

## API

```
GET /api/radio
```

Stations grouped by kind, as today, but with only `Live` and `Stream` groups. The `Artist`,
`Genre`, `Similar`, `Period`, `Series` and `Other` groups go. `Station` objects keep their
current fields.

```json
{"Live": [{"ID": 41, "Name": "Jazz FM", "Creator": "Takeout", "Type": "live",
           "Image": "…", "Description": "Jazz all day"}],
 "Stream": [{"ID": 7, "Name": "KEXP", "Creator": "KEXP", "Type": "stream", "Image": "…"}]}
```

`Live` is in configured order: the lineup, then genres, decades and series. Row IDs don't keep
that order as seasonal stations come and go, so the radio job stores it on the row (or the server
sorts by config when answering).

```
GET /api/radio/schedule[?start=<RFC3339>][&end=<RFC3339>]
```

Every live station's tracks overlapping the range, plus the server's time. `start` defaults to
now and `end` to `start` + 30m, enough for what's on now and next. `end` is capped at now +
`Horizon`, and `end` before `start` is an error. There's no per-station filter: 40 stations over
30 minutes is about 300 small rows. One can be added later if that matters on Wear OS or Android
Auto.

```json
{"Time": 1763240000000,
 "Stations": [{"ID": 41, "Name": "Jazz FM",
               "Tracks": [{"Start": "…", "End": "…", "Title": "So What",
                           "Artist": "Miles Davis", "Release": "Kind of Blue",
                           "Image": "…", "Location": "…", "ETag": "…", "ID": 1234}]}]}
```

- **No "now playing" field.** As with `/api/guide`, the server sends a time window and the apps
  work out what's on now. A now-playing field would go stale when the track changed, and the
  apps need the schedule anyway to know when to refresh.
- **Stations** carry the `ID` and `Name` from `/api/radio`, so the schedule reads on its own.
- **`Time`** is the server's time in milliseconds, as in `/api/index` and `/api/guide`. Apps use
  it for offsets, since device clocks can be wrong.
- **A track's `ID`** is its track row, for "go to album" and favorites.
- **Safeguard:** if a schedule ends within 2 hours of the request (the job didn't run because
  the server was down, say), the request extends it before answering.

```
GET /api/radio/{id}/playlist[.xspf]
```

For a live station, the schedule from the current track onward as a playlist. It plays close to
live (from the start of the current track, with skipping) in anything that takes a playlist,
including VLC and other third-party players. Stream stations are unchanged.

## Apps (`takeout_native`)

Live stations play in the music player (`TakeoutMusicService`), so background play, the lock
screen, media notifications, Bluetooth, Android Auto and Wear OS all work.

- **Every entry point:** the radio screen, voice ("play Jazz FM"), Android Auto's browse tree, the
  Wear radio list and search results all play a station through `stationQueue(id)`. For a live
  station that has to start live mode, not load a playlist, so `Station` needs a live kind in
  `:shared`.
- **On now:** a pure function in `:shared`, like the guide's `onNow`, returning a station's
  current track, the offset into it and the next track for a given server time. The radio screen,
  the player and tests all use it.
- **Radio screen:** one `/api/radio/schedule` call shows what's on each live station ("Miles
  Davis – So What") and refetches when the earliest current track ends.
- **Tuning in:** fetch the schedule, and with the server's time open the current track at
  `now - Start`. Seeking into a FLAC file is an HTTP range request on the normal `Location` URL,
  which ExoPlayer and beep already do. Only the first track after tuning in starts mid-song;
  the rest play from the top.
- **Queue:** the tracks from the default 30-minute window (the current track and about 6–8
  more), at least the current track and the next, so transitions are gapless. Apps on devices
  that may lose the network, such as Wear OS and Android Auto, can ask for `end=now+1h`.
- **Refresh:** refetch the schedule each time a track starts, and always when 2 or fewer tracks
  are left in the queue. Append the new tracks after the last queued one (matched by `Start`),
  and never replace the playing track. Behind live, refetch with `start` set to the last queued
  track's `End`, so the delayed copy has no gap. If a refetch fails, keep playing the queue and
  retry at the next track.
- **Staying in sync:** `Duration` is whole seconds, so a file can run up to a second longer or
  shorter than its slot. Let each file finish and start the next from the top, and only correct
  when playback is more than 5 seconds off the schedule. Correct by starting the next track at its
  offset, never by cutting a track short. Behind live, there's no correction.
- **Controls:** play/pause, no skip, no seek, and a progress bar for the current track. Media
  session metadata changes on each track.
- **Pause and resume:** pausing is allowed. On resume, the player keeps playing from where it
  paused if it's behind live by less than 5 minutes or the paused track's remaining time,
  whichever is longer. It shows "Behind live · Jump to live" and plays the queued tracks in
  order, a delayed copy of the station. Further behind than that, resume jumps to live: it
  refetches and starts the track that's on now at its offset, like tuning in. This also avoids
  a delayed queue that can't be refilled, since the server keeps ended tracks for only an hour.
  "Jump to live" does the same at any time.
- **Listens:** on, as for today's stations. Live stations replace them, so turning listens off
  would quietly stop scrobbling what's scrobbled now.
- **Android TV:** the music section's radio screen, not the guide grid.
- **Third-party players** use the playlist fallback.

## Code

- **`internal/radio`:** a `Radio` type with its own `radio.db` (tables `radio_tracks`, with
  `station`, `r_id`, `start_at`, `end_at`), the radio job, the picker and the schedule
  query.
- **Config:** `config.RadioConfig` under `Radio`, with its own `DB` like `Guide`.
- **Server:** a `radio` job in `jobs.go` replacing `stations`, `/api/radio/schedule`, live
  playlists in `RefreshStation`, and `RadioView` with only `Live` and `Stream`.
- **Music:** `Music.CreateStations` keeps only `RadioStreams`. `TypeLive` (`"live"`) joins
  `TypeStream`; the other types go in clean-up.

## Clean-up

Once live stations replace the generated ones:

- `TypeArtist`, `TypeGenre`, `TypeSimilar`, `TypePeriod`, `TypeSeries`, `TypeOther` and their
  `RadioView` groups.
- The unrouted `apiRadioPost` and `apiStation` handlers, and `recvStation`.
- `Station.User`, now always the Takeout user, and `Station.Playlist`, which is never saved.

## Phases

1. **Server:** `internal/radio`, the picker with tests, the `Radio` config, the
   radio job, `Live` stations in `/api/radio`, `/api/radio/schedule`, and the playlist
   fallback. Testable with the playlist before any app work.
2. **Phone and Linux apps:** live mode in the music player, and live stations through every entry
   point.
3. **Android TV, Wear OS and Android Auto.**
4. **Clean-up,** once the apps ship.

## Implementation notes

Phase 1 as built:

- **Package and job:** `internal/radio` with `Radio` (`radio.db`, table `radio_tracks`), the
  picker (`pick`, a pure function) and `Sync`. The `radio` job runs every `Radio.SyncInterval`
  and at startup, and can be run by name like other jobs.
- **Station type:** the constant is `music.TypeLiveStation` (`"live"`), since `music.TypeLive`
  already names live recordings. `Station.Rank` (not in JSON) holds the lineup order.
- **Old stations:** the radio job deletes generated stations (genre, period, series or other)
  left from older releases; live stations are new rows with new IDs. Stream stations are never
  touched. The `stations` job (and `takeout radio`) now only creates stream stations, and no
  longer deletes live ones.
- **Settings:** stations come only from `Radio.Lineup`, `Genres`, `Decades` and `Series`, none of
  which have defaults. The old `Music.RadioGenres`, `RadioSeries` and `RadioOther` settings are
  removed, and ignored if a config still has them.
- **Names are unique across all stations.** A live station whose name is taken by a stream
  station isn't created or scheduled; the radio job logs it.
- **Station images** can be server paths, such as an album cover
  (`/img/mb/re/<release id>/front`), since the apps resolve them against the server.
- **Pools** dedupe a recording on several releases, and separate releases by release group.
- **API:** `/api/radio` has the new `Live` group sorted by `Rank`; the older groups are still in
  the response (empty once the radio job has run) until clean-up. `/api/radio/schedule` lists
  stations in lineup order and skips a track that has left the library. Both it and a live
  station's playlist extend schedules first when one ends within 2 hours.
- **Playlist:** a live station's playlist is up to 100 tracks from the current one, about six
  hours.
- **Times** are stored in UTC, since SQLite compares them as text.
