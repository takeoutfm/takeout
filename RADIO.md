# Radio

Status: phase 1 (server) is implemented; the apps are not. See "Implementation notes" at the
end for where the code differs from this text.

Live radio from the library. A live station is a schedule of tracks, and tuning in means playing
the track that's on now from `now - track.start`. Everyone hears the same thing and joins
mid-song, like real radio, and nothing plays on the server.

Radio has two kinds of station: **live**, scheduled from the library, and **stream**, internet
radio. Both come only from the `Radio:` config, and `internal/radio` owns them in its own
`radio.db`. Nothing of the older radio is kept: no generated stations, no station rows in
music.db, no `Music:` radio settings and no migration.

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

## What it replaces

The older radio had generated stations in music.db (genre, decade, series, artist, similar and
"other", built by the `stations` job from `Music.Radio*` settings), each a fresh shuffled
playlist with skipping, plus stream stations in the same table. All of that is gone:

- the generated station types and the `stations` job (and `takeout radio`);
- station rows in music.db (its old `stations` table is no longer read, and can be dropped);
- per-user station fields (`User`, `Shared`, `Ref`, `Playlist`) and the unrouted create, update
  and delete handlers;
- the `/api/stations/{id}` routes, replaced by `/api/radio/{id}`;
- the `Music.RadioGenres`, `RadioSeries` and `RadioOther` settings.

**The deliberate loss is skipping.** Genre and decade radio no longer skip or reshuffle. That's
the point of real radio, but it's a change. Artist and track radio (`&radio=1` search refs and
`/music/tracks/{id}/radio`) aren't stations and stay on-demand.

## Configuration

A `Radio:` section per media collection, like `Guide:`, for live and stream stations. Artist,
track and search radio are on-demand music playlists, not stations, so their settings
(`ArtistRadio*`, `TrackRadio*`, `RadioLimit`, `RadioSearchLimit`) stay under `Music:`.

```yaml
Radio:
  Timezone: America/Los_Angeles
  Genres: [jazz, rock]
  Decades: [1960, 1970, 1980, 1990, 2000, 2010, 2020]
  Series: ["Billboard Hot 100"]
  Lineup:
    - {Name: Jazz FM, Music: "genre:jazz", Image: "https://…", Description: "Jazz all day"}
    - {Name: Christmas Radio, Layout: Jan, Match: Dec, Music: "genre:christmas"}
  Streams:
    - Creator: SomaFM
      Title: Groove Salad
      Image: "https://…"
      Source: [{ContentType: audio/mpeg, URL: "https://…/groovesalad.pls"}]
```

| Setting | Default | Meaning |
|---|---|---|
| `Timezone` | server local | For `Layout`/`Match` windows. |
| `SyncInterval` | `1h` | How often the radio job runs. `0` disables it. |
| `Horizon` | `24h` | How far ahead schedules are built. Nobody browses radio a week ahead, and a short horizon keeps the table small: 40 stations at about 4 minutes a track is about 15,000 rows. |
| `Genres` | none | Automatic genre stations, with today's query (`+genre:"…" +type:single +popularity:<11 -artist:"Various Artists"`). |
| `Decades` | none | Automatic decade stations ("1980s Top Tracks"), with today's query. |
| `Series` | none | Automatic series (chart) stations. |
| `Lineup` | none | Hand-written stations: `Name`, `Music` (a query, or a list of them; see "Several queries"), optional `Layout`/`Match`, `Image`, `Description`, `MinContent`, `Shows` (see "Shows"). |
| `MinContent` | `2h` | A station whose pool is shorter than this is skipped and logged. Can be set per station. |
| `SearchLimit` | `1000` | Search limit for a station's pool. |
| `Streams` | none | Internet radio stations: `Creator`, `Title`, `Image`, `Description`, `Source`. Was `Music.RadioStreams`. |

**Order:** `Lineup` first in its order, then genres, decades and series.

**Order:** streams are listed in their config order.

**Several queries:** in a search query, `+` means required, so `+artist:"A" +artist:"B"` finds
only tracks credited to both. Terms without `+` are alternatives, but only while the query has no
`+` terms at all: with one, they only affect ranking, and the query string has no parentheses or
`OR` to group them. So a lineup entry's or show's `Music` can be a list, and the pool is every
track any of them finds (each searched up to `SearchLimit`), without repeats:

```yaml
    - Name: Shoegaze Singles
      Music:
        - '+artist:"DIIV" +type:single'
        - '+artist:"Slowdive" +type:single'
        - '+artist:"My Bloody Valentine" +type:single'
```

A single string is one query, commas and all.

**No migration:** `Music.RadioGenres`, `RadioSeries` and `RadioOther` are gone and aren't read,
and `Music.RadioStreams` is now `Radio.Streams`.
**Names** are unique across live stations and streams; a repeated name keeps the first station
(live before streams), and the radio job logs the rest.

## Schedule

The radio job, every `SyncInterval`:

1. Works out which live stations are on the air: in season, with at least `MinContent` of music.
2. Makes the `stations` rows in radio.db match the config, matched by name: one per configured
   live station (on the air or not) and one per stream. A row keeps its ID while its name stays
   the same, so a seasonal station has the same ID every year. Rows for stations no longer in
   the config are deleted, with their schedules.
3. Extends each on-air station's schedule to `Horizon`. The schedule is keyed by station name,
   like guide channels, so renaming a station starts a fresh schedule (and a new ID).
4. Deletes tracks that ended more than an hour ago, and the schedules of stations that went off
   the air.

Tracks run back to back with no alignment, each `Start` the previous `End`. After downtime a
schedule restarts at now rather than filling the past.

**Changes:** scheduled tracks don't change, except when a station's plan does: its `Music`,
`MinContent` or `Shows`. Then the radio job deletes the station's tracks after the one on now
and schedules again from there, so a new show is on the next run rather than a day later.

**Pool:** tracks matching the query, or any of the queries (the same `Music.Search` as today's refs), that have a recording
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

## Shows

A show is a regular slot on a live station, like "Gary Numan Fridays" or "New Music Saturday":
for its time the station plays the show's music instead of its own. Shows are listed under a
lineup entry (not the automatic genre, decade and series stations):

```yaml
Radio:
  Lineup:
    - Name: Post-Punk
      Music: '+genre:"post-punk" +type:single +popularity:<11'
      Shows:
        - Name: Gary Numan Fridays
          Layout: Mon
          Match: Fri
          At: "18:00"
          Length: 3h
          Music: '+artist:"Gary Numan"'
          Image: /img/mb/re/<release id>/front
    - Name: Top Hits
      Music: "+popularity:1"
      Shows:
        - Name: New Music Saturday
          Layout: Mon
          Match: Sat
          At: "09:00"
          Length: 3h
          Released: 720h      # first released in the last 30 days
```

| Setting | Default | Meaning |
|---|---|---|
| `Name` | required | Shown with the station while the show is on. |
| `Layout`, `Match` | every day | Which days, as in `DateRecommend`: `Layout: Mon`, `Match: Fri` is Fridays. |
| `At` | `00:00` | Local start time, in `Radio.Timezone`. |
| `Length` | `24h` | How long it's on. With the defaults, a show takes over its whole day. |
| `Music` | | Search query for the show's music, or a list of them whose results are combined. |
| `Released` | none | Only recordings first released this recently (`first_date`), for new music; with several queries, each is limited. With no `Music`, every such track. |
| `Image`, `Description` | none | For the apps. |

- **Edges:** the track on when a show starts plays to its end, so the show starts a little late
  rather than cutting a song. The show's last track can run past its end, as any track does.
- **Picker:** the same rules, with the show's own pool and the station's history, so a show
  doesn't repeat what the station just played. A single-artist show can't keep artists apart, so
  that rule relaxes by itself.
- **Too little music:** a show needs `Length` of music, or the station's `MinContent` if that's
  less. Otherwise it's skipped and logged, and the station plays its usual music.
- **Overlaps:** if two shows on a station overlap, the one that started first runs to its end.
- **Released** is worked out when the radio job runs, so "the last 30 days" moves with it.

## API

```
GET /api/radio
```

Stations grouped by kind: `Live` (on the air now) and `Stream`. Nothing else.

```json
{"Live": [{"ID": 41, "Name": "Jazz FM", "Creator": "Takeout", "Type": "live",
           "Image": "…", "Description": "Jazz all day"}],
 "Stream": [{"ID": 7, "Name": "KEXP", "Creator": "KEXP", "Type": "stream", "Image": "…"}]}
```

Both are in config order: live stations as the lineup, then genres, decades and series; streams
as listed. The order is stored on the row (`Rank`, not in JSON). Station objects have `ID`,
`Name`, `Type`, `Creator`, `Image` and `Description` (plus gorm's timestamps). A live station's
`Creator` is "Takeout".

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
                           "Image": "…", "Location": "…", "ETag": "…", "ID": 1234,
                           "Show": "Jazz at Night"}],
               "Shows": [{"Name": "Jazz at Night", "Image": "…", "Description": "…"}]}]}
```

- **No "now playing" field.** As with `/api/guide`, the server sends a time window and the apps
  work out what's on now. A now-playing field would go stale when the track changed, and the
  apps need the schedule anyway to know when to refresh.
- **Stations** carry the `ID` and `Name` from `/api/radio`, so the schedule reads on its own.
- **`Time`** is the server's time in milliseconds, as in `/api/index` and `/api/guide`. Apps use
  it for offsets, since device clocks can be wrong.
- **A track's `ID`** is its track row, for "go to album" and favorites.
- **Shows:** a track in a show has the show's name in `Show` (left out otherwise), and the
  station's `Shows` lists each show named by its tracks, once, with its image and description.
  Apps show it with the station: "Jazz FM · Jazz at Night".
- **Safeguard:** if a schedule ends within 2 hours of the request (the job didn't run because
  the server was down, say), the request extends it before answering.

```
GET /api/radio/{id}/playlist[.xspf]
```

For a live station, the schedule from the current track onward as a playlist. It plays close to
live (from the start of the current track, with skipping) in anything that takes a playlist,
including VLC and other third-party players. For a stream, the stream's sources (`.pls` sources
resolved). The playlist's `Location` is `/api/radio/{id}`.

A station off the air isn't found, by ID or by name. Search (`/api/search`, and `radio:` or
`station:` queries) and voice match stations by name or creator. Playlist refs
`/music/stations/{id or name}` still resolve.

## Listens

Live radio listens follow the same rules as any other listen; being live changes nothing.

- **The rule:** a scheduled track counts as listened when it has been heard for 4 minutes or for
  half its length, whichever comes first: `countsAsListened` in the apps' `:shared`, and the
  same test in the web UI's `player.js`. Heard time is time actually played, so joining a track
  part way through, pausing, or jumping to live only counts what was heard. A track joined near
  its end may not count, as when starting any track and stopping early.
- **Library tracks, not streams:** scheduled tracks are library tracks with an `ETag`, so they're
  judged by the library rule above, never by the internet-stream rule (ICY titles, heard from
  the start).
- **Takeout:** each client sends its listens to `/api/activity`, as for any track. The server
  records them and does nothing else with them.
- **ListenBrainz is client-only.** Each client submits to ListenBrainz itself, with the user's
  own token and only when the user has turned it on, exactly as for any other track: "playing
  now" when a track starts and the listen when it counts. **The server never sends listens to
  ListenBrainz**, for radio or anything else: ListenBrainz accounts are the user's, set up and
  managed in each client, separately from Takeout. (The server's `ListenBrainz` setting is only
  for looking up popular tracks.)
- **Behind live** makes no difference: what's heard is what's counted.

Checked in the web UI on the dev server: each scheduled track heard past the rule showed up in
`activity.db` with its recording ID, at the time it started playing.

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
- **Shows:** while a show is on, the radio screen and the player show its name (and image, if
  it has one) with the station's, and the media session's subtitle names it.
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
- **Listens:** exactly as for any library track (see "Listens"): `ListenTracker` judges each
  scheduled track with `countsAsListened`, sends it to Takeout, and to ListenBrainz when the
  user has turned that on. Live mode must keep every scheduled track a normal media item with
  its `ETag`, not a stream, so the stream rules never apply.
- **Android TV:** the music section's radio screen, not the guide grid.
- **Third-party players** use the playlist fallback.

## Web UI

The built-in web UI (`internal/server/res`) is the quickest place to try live radio, since it's
in this repository and needs no app release. Before this, a live station in the web UI played
the playlist fallback: it was loaded into the queue like an album, started at the top of the
current track, and could be skipped and shuffled. The web UI now has a live mode in the player
(`player.js`), and no other radio.

### Radio page

- **Live** tiles show what's on under the station name: "Artist – Title", and the show's name
  while one is on ("Gary Numan Fridays"). **Streams** stay as they are.
- The page renders the tiles; `player.js` fills the "on now" lines from one
  `/api/radio/schedule` call when the page is shown (`takeout:page`), and again when the
  earliest current track ends, while the page is open.
- A live tile has `data-live="<station id>"` instead of `data-ref`, so clicking it tunes in
  rather than loading a playlist. The same `station-tile` template is used by search results,
  so stations found by search tune in too.

### Live mode in the player

- **Off the queue:** live mode plays from its own small list of scheduled tracks, not the
  server's queue (`/api/playlist`). The queue is left as it was, so stopping the radio (or
  playing anything else) goes back to it, and other apps don't get a queue of 100 scheduled
  tracks they'd treat as a playlist. The station's ID is kept in `localStorage`, so a reload
  tunes in again (to live) instead of losing the radio.
- **Tuning in:** fetch the station's schedule, work out the server's time from `Time` (plus
  the time since the response), find the track on now and play its `Location` from
  `now - Start`. The browser seeks with a range request, which the track redirect supports.
- **Queue and refresh:** as in the apps: the schedule's tracks from now (about 30 minutes),
  refetched each time a track starts and when 2 or fewer are left, appending after the last
  one. A failed refetch keeps playing what's there.
- **Next track:** each track plays to its end and the next starts from the top. A second,
  preloaded `<audio>` element for the next track keeps the gap short; it's swapped in on
  `ended`. If playback is more than 5 seconds off the schedule, the next track starts at its
  offset instead.
- **Controls:** play/pause only. Previous, next, the seek bar and Shuffle are hidden, and the
  LIVE badge (already there for streams) shows. Keyboard arrows do nothing; space pauses.
- **Progress:** a progress bar that can't be dragged and the time left in the track ("−1:23"),
  in Now Playing and the mini player, from what's heard rather than the schedule, so it's right
  behind live too. Now Playing also shows "Next: Artist – Title". The lock screen gets the
  track's position.
- **Pause and resume:** as in the apps: under 5 minutes behind (or the paused track's remaining
  time), resume carries on as a delayed copy, with "Behind live · Jump to live" in Now Playing
  and the mini player; further behind, resume jumps to live.
- **Display:** the title and artist are the track's; the second line is the station, with the
  show while one is on ("Jazz FM · Jazz at Night"). Cover art is the track's, or the show's
  or station's image if the track has none. The Now Playing queue becomes "Up next on Jazz
  FM": the scheduled tracks, which can't be clicked or removed. The media session (lock
  screen, keyboard media keys) shows the same and offers only play and pause.
- **Listens:** as for any music track (see "Listens"): to Takeout by `/api/activity`, by heard
  time. The web UI has no ListenBrainz support, for live radio or anything else.
- **Leaving live mode:** playing anything else, or "Stop radio" in Now Playing (where the
  queue's Shuffle and Clear were), stops it, forgets the station and shows the queue again.

### Server changes

- **`GET /api/radio/{id}/schedule[?start][&end]`:** one live station's schedule, in the same
  shape as `/api/radio/schedule` with only that station in `Stations` (404 for a stream or an
  unknown station). Stations in a schedule now also carry their `Image`. The player refetches every few minutes, and the
  full schedule of every station is about 3 KB a station for 30 minutes; the apps want this too.
- **Station tile:** `data-live` for live stations, `data-ref` for streams; the station's `Type`
  is already in the view.
- **gzip for `/api` responses** (pages are already gzipped) would also cut the schedule's size
  by about ten times. Optional.

### Removed

- Live stations no longer play through the playlist fallback in the web UI: no
  `/music/stations/{id}` ref for them, so no skipping, shuffling or queueing of a live station.
  (Streams still go through the queue, which is how the player plays them now.)
- Nothing else of the old radio is left in the web UI: the Radio page and search have only live
  and stream stations already. Artist radio (the Radio button on an artist page) isn't a station
  and stays.

### Not included

- A schedule view (what's on later today) or upcoming shows on the Radio page.
- Live mode for Android TV, phones, Wear OS or Auto (phases 2 and 3).

### Testing

No JavaScript tests exist for the web UI, so this is checked by hand on the dev server: tuning
in mid-track lands within a second or two of the schedule; track changes are close to gapless;
pause under and over 5 minutes; a reload tunes in again; a show's name appears when it's on;
the Radio page's "on now" lines change as tracks end; and the queue is unchanged after leaving
live mode. Go tests cover the new endpoint and the tile's attributes.

## Code

- **`internal/radio`:** a `Radio` type with its own `radio.db` (tables `stations` and
  `radio_tracks`), the radio job, the picker, station lookup and search, and the schedule
  query. `model.Station` is the station row.
- **Config:** `config.RadioConfig` under `Radio`, with its own `DB` like `Guide`.
- **Server:** the `radio` job in `jobs.go` (it also runs after `takeout run` syncs music),
  `/api/radio`, `/api/radio/schedule` and `/api/radio/{id}`, playlists in `RefreshStation`,
  `RadioView` with `Live` and `Stream`, and the web UI's radio page.
- **Music:** nothing about stations. Artist, track and search radio stay, with their `Music:`
  settings.


## Phases

1. **Server:** `internal/radio`, the picker with tests, the `Radio` config, the
   radio job, `Live` stations in `/api/radio`, `/api/radio/schedule`, and the playlist
   fallback. Testable with the playlist before any app work.
2. **Phone and Linux apps:** live mode in the music player, and live stations through every entry
   point.
3. **Android TV, Wear OS and Android Auto.**

There's no clean-up phase: the old radio was removed in phase 1. Until the apps are updated
they show only what they understand of `/api/radio`, and anything matching a playing queue
against `/api/stations/{id}` needs to use `/api/radio/{id}`.

## Implementation notes

Phase 1 as built:

- **Package and job:** `internal/radio` with `Radio` (`radio.db`), the picker (`pick`, a pure
  function) and `Sync`. The `radio` job runs every `Radio.SyncInterval` and at startup, and can
  be run by name like other jobs. It needs the music library only for station pools.
- **Types:** `radio.TypeLive` (`"live"`) and `radio.TypeStream` (`"stream"`). `Station.OnAir`
  (not in JSON) marks a live station with a schedule now; `Rank` holds the config order.
- **Stream sources** are stored on the row as JSON (`Source`, not in JSON output) and resolved
  when the playlist is requested.
- **Station images** can be server paths, such as an album cover
  (`/img/mb/re/<release id>/front`), since the apps resolve them against the server.
- **Pools** dedupe a recording on several releases, and separate releases by release group.
- **Schedule API:** lists on-air stations in config order and skips a track that has left the
  library. Both it and a live station's playlist extend schedules first when one ends within 2
  hours.
- **Playlist:** a live station's playlist is up to 100 tracks from the current one, about six
  hours.
- **Times** are stored in UTC, since SQLite compares them as text.
