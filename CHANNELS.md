# Channels

Status: proposal. Phases 1 (durations) and 2 (server schedule) are implemented. In the TV app,
phase 3 (live player) and phase 4 (guide grid, days, Watch Now, Start Over) are built, except
Open to Live TV; live playback hasn't been tested yet. Of phase 5, specials and features are
implemented; music breaks and music channels are not. Phase 6 is not started. See
"Implementation notes" at the end for where the server code differs from this text.

A TV-style guide of named channels with movies and shows "already playing", primarily for
Android TV. Someone used to watching TV sees what's on, picks a channel and joins wherever the
program is now. Each day's schedule can be browsed ahead of time.

The server owns the schedule, so every device agrees. The apps create the illusion of live TV.

## Core idea: no actual broadcast

A channel is only a schedule. Tuning in means direct-playing the existing file from S3 at
`now - program.start`.

- No ffmpeg, transcoding, HLS muxing or per-channel server process.
- A channel nobody is watching costs nothing.
- Streams go through the existing `Location` URLs, the same as on-demand playback.

## Defining channels

Channels are configured per media collection, next to `Film.Recommend`. A channel has the same
shape as a `DateRecommend` entry (name, optional date window, query), with separate queries
because movies and TV are two search indexes.

```yaml
Guide:
  Timezone: America/Los_Angeles
  BreakMusic: "genre:jazz"
  Lineup:
    - {Name: Spooky Season, Layout: Jan, Match: Oct, Movies: "keyword:halloween"}
    - {Name: Horror, Movies: "genre:Horror", BreakMusic: "genre:ambient"}
    - {Name: Sitcoms, Shows: "genre:Comedy"}
    - {Name: Family Night, Movies: "rating:G rating:PG", Shows: "genre:Family", Align: 15m}
```

The query strings are illustrative; check them against the real search index and data,
particularly how multiple terms combine.

| Setting | Default | Meaning |
|---|---|---|
| `Timezone` | server local | Used for date windows and time-of-day rules. In a container, local time is probably UTC, so this should normally be set. |
| `Lineup` | none | The channels, in guide order. |
| `SyncInterval` | `1h` | How often the schedule job runs. `0` disables it, like the other sync intervals. |
| `Horizon` | `168h` | How far ahead the schedule is built, which is also how far ahead the guide can browse. |
| `Align` | `5m` | Programs start on multiples of this. Longer values mean longer music breaks. Can be set per channel. |
| `MinBreak` | `0` | If a gap would be shorter than this, the next program moves to the following mark, giving a longer break. Can be set per channel. |
| `BreakMusic` | whole library | Music search query for breaks. Can be set per channel. |

Rules:

- **Names, not numbers.** Each channel gets a row with a database ID, created or matched by name
  on each job run. The schedule uses that ID; the API uses the name. Renaming a channel creates
  a new row and starts a fresh schedule, and the old entries age out.
- **Order** is the order of `Lineup`. A seasonal channel placed first is at the top while it's
  in season.
- **Seasonal channels** come and go on their own. The job checks the window for each day with
  `date.MatchTime`, and the guide only lists channels with programs in the requested range.
- **Too little content:** skip and log a channel whose query can't fill about two days without
  repeating.
- **Search limit:** `Film.Search` and `TV.Search` take a limit. The default of 100 is too small
  for a channel pool, so pass a larger one.
- **Default:** a collection with no `Channels` section has no channels.

Automatic channels (one per genre from `DistinctGenres`) are left out for now. A hand-written
lineup of 6–10 channels will feel more like real TV than many overlapping genre channels. It
could later be a single setting.

## Specials and features

Both are optional named blocks of programming inside a channel. The difference is surprise
versus appointment: viewers stumble onto a **special**, while a **feature** is a regular slot
they learn to expect. A simple test is whether you'd want viewers to plan around it.

| | Special | Feature |
|---|---|---|
| Feels like | An event: "all the Die Hards tonight" | A regular slot: "Friday night is new movies" |
| Who picks when | The scheduler, roughly every `Every` | You, on a fixed day and time |
| Content | A themed set, rotating through a pool | The same theme each time, different movies |
| Length | As long as the set runs, up to `MaxLength` | A fixed slot (`Length`) |
| Movies chosen | When the schedule is built, about a week ahead | About a day before airing |
| No content | Pick another special | The slot becomes regular programming |

In code they share one scheduling path: a named block placed before regular programming fills
in around it. Only the trigger, the rotation, and when the movies are chosen differ. The config
keeps the two names because they read more clearly than one general list.

```yaml
Lineup:
  # Multiple specials
  - Name: Action
    Movies: "genre:Action"
    Specials:
      Every: 720h                  # one special about every 30 days
      At: "20:00"                  # on an evening the scheduler picks
      MaxLength: 12h               # longest a special can run
      Collections: true            # automatic specials from this channel's collections
      MinFilms: 3
      Skip: ["Twilight Collection"]
      Themes:                      # specials you define
        - Name: Tom Cruise Night
          Movies: 'cast:"Tom Cruise"'
        - Name: Arnold Marathon
          Movies: 'cast:"Arnold Schwarzenegger"'
          Order: date
        - Name: Heist Night
          Movies: "keyword:heist"

  # Multiple features
  - Name: Good Movies
    Movies: "vote:>7"
    Features:
      - Name: New on Friday
        Layout: Mon                # compare the day of the week
        Match: Fri
        At: "20:00"
        Length: 4h
        Source: added
      - Name: Saturday Westerns
        Layout: Mon
        Match: Sat
        At: "20:00"
        Length: 4h
        Movies: "genre:Western"
      - Name: Sunday Matinee
        Layout: Mon
        Match: Sun
        At: "14:00"
        Length: 3h
        Movies: "genre:Family"
      - Name: First of the Month
        Layout: "2"                # compare the day of the month
        Match: "1"
        At: "20:00"
        Length: 4h
        Source: released
```

A channel can have both `Specials` and `Features`.

### Specials

| Setting | Default | Meaning |
|---|---|---|
| `Every` | none | About how often the channel runs a special. |
| `At` | `20:00` | Start time on the day the scheduler picks. |
| `MaxLength` | `12h` | The special stops after the last whole program that fits. Without it, a query like `cast:"Tom Cruise"` could run for days. |
| `Collections` | `false` | Add automatic specials from the channel's collections. |
| `MinFilms` | `3` | How many of a collection's films must be in the channel's results. |
| `Skip` | none | Collection names never to use. |
| `Themes` | none | Hand-written specials: `Name`, a `Movies` and/or `Shows` query, and optional `Order: date` (default shuffled). |

**Automatic collections.** The channel's own movies decide which collections it gets, so a
collection only becomes a special on a channel where it already fits. Collections are already in
the database, one row per movie keyed by that movie's TMID. On each run:

1. Take the TMIDs from the channel's query results.
2. Fetch the `collections` rows with those `tm_id` values, in chunks like `Search` does.
3. Group by `Name` and keep names with at least `MinFilms` rows, minus `Skip`.

For example, Action with `genre:Action` might qualify Die Hard (5 films) and Mission: Impossible
(6), while Alien (2 action-tagged films) doesn't qualify there but would on Horror.

- **What plays:** every film you own from the collection, in release order (`CollectionMovies`),
  even ones that didn't match the channel's query. A marathon missing one film would feel broken.
- **Identity:** a collection's rows carry movie TMIDs, so its name is its identity, as in
  `Collections()` and `CollectionMovies()`. A TMDB rename counts as a new special. Re-syncs are
  safe because the rows come back under the same TMID and name.
- **Display name:** from TMDB, probably "Die Hard Collection". The server could trim the word
  "Collection" or show "Die Hard Marathon".
- **Including a collection the query misses:** add it as a theme with a `collection:` query.

**Scheduling.** Once `Every` has passed since the channel's last special, the scheduler picks an
upcoming evening, varied so it isn't predictable, and places the special at `At` as the schedule
extends into that day. Nothing already scheduled changes.

- **Rotation:** both sources go into one pool. Pick the special that has gone longest without
  airing. A small history table (channel, special name, date) keeps this working after old
  schedule rows are cleaned up.
- **Across channels:** a collection can qualify for several channels. Don't run the same special
  on two channels in the same week.
- **Around a special:** regular programming avoids its titles on that channel for a few days
  either side.

### Features

| Setting | Meaning |
|---|---|
| `Layout`, `Match` | Which days, using the same date matching as channels (`Mon`/`Fri` for weekdays, `"2"`/`"1"` for the 1st of the month). |
| `At` | Start time. |
| `Length` | How long the slot is reserved. |
| `Source` | `added` (`RecentlyAdded`) or `released` (`RecentlyReleased`). |
| `Movies`, `Shows` | A query, instead of `Source`. |
| `Description` | Optional text shown in the guide before the slot is filled. |

- **Reserved, then filled late:** when the schedule reaches a feature's day, the slot is reserved
  with the feature's name. Its programs are chosen about 24 hours before it airs, so a movie added
  on Monday makes that Friday. This is the only part of the schedule filled late, and only inside
  its own reserved slot. Until then the guide shows a single block with the feature's name.
- **Filling:** pick titles not yet shown in that feature that fit the slot. Fill large leftover
  time with regular programming and small leftovers with a music break. If there's nothing new,
  the whole slot becomes regular programming without the feature's name, rather than repeating
  last week.
- **`Source: added`:** `RecentlyAdded` uses the S3 `LastModified` date within `Film.Recent`, so it
  survives re-syncs, but an upgraded file counts as new.
- **Ignores the channel's query:** a source or query belongs to the feature, so New on Friday on
  Good Movies shows any new movie, well-rated or not. Requiring both would leave most Fridays
  empty on a genre channel.

### Conflicts

- A feature's time is fixed, so it wins over a special, and the scheduler picks another evening
  for the special.
- When features overlap, such as First of the Month on a Friday, the first feature listed wins.

## Schedule keys

Schedules are long-lived, so they follow the rule in `model/favorite.go`: external IDs, no
internal sync identifiers.

| Item | Key |
|---|---|
| Movie | `TMID` |
| Episode | `TVID` + season + episode |
| Break track | `RID` (MusicBrainz recording); tracks without one aren't used for breaks |

Channel rows are different: `internal/guide` owns them and media syncs never rebuild them, so
their database IDs are stable.

Don't use media row IDs or UUIDs, because `syncMovie` deletes and recreates the row and a full
music sync runs `deleteTracks`. Don't use ETags
either: upgrading a file changes its ETag, but the same movie should stay on the schedule and
play the new file.

The guide resolves each entry to the current row when requested and builds `Location` then.
If the title has left the library, the slot shows an "off air" slate until the next program.

## Real durations

`Runtime` is whole minutes from TMDB, and the actual file can differ (extended cuts, for
example), so schedules built on it would drift. `Track` has no duration at all, and music breaks
need one. We need file durations in seconds for all three.

### Storage

Add `Duration int` (seconds) to `Movie`, `TVEpisode` and `Track`; `AutoMigrate` adds the column.

| Value | Meaning |
|---|---|
| `0` | Not scanned yet |
| `> 0` | Duration in seconds |
| `-1` | Couldn't be read; don't retry |

- A full music re-sync (`deleteTracks`) wipes the column for tracks. The job refills it
  cheaply. Movies keep it: a re-sync of an unchanged file is skipped as a duplicate, and a
  changed file gets a new row with `Duration = 0`.
- `syncEpisode` updates the existing row and resets `Duration` to `0` when the ETag changes.
- A replaced track file keeps its old row: `createTrack` is a plain `Create` on the unique
  `Key`, so the insert fails (silently) and the row keeps its old ETag, size and duration.
  This predates channels and affects more than durations.
- A missing file (404) is marked `-1` like an unreadable one, so dead rows don't block each
  batch. Network and server errors leave `0` and are retried next run.
- For video, the scheduler uses `Duration` when positive, otherwise `Runtime` plus a safety
  margin. Tracks without a positive `Duration` aren't used for breaks.

### Scan job

Jobs modeled on `syncFilmPosters` and `syncTVPosters`, registered with `mediaSync`, pick up rows
with `Duration = 0`. That covers both the backfill of existing rows and new files. Each run
handles a limited batch, so a large music library catches up over several runs.

The scan reads only file headers and metadata, using plain `net/http` with a `Range` header on
the existing presigned URL (`MovieURL` / `EpisodeURL` / `TrackURL`). The `Bucket` interface
doesn't change. Movie sync only accepts `.mkv` and `.mp4`, and music sync
`.mp3`, `.flac`, `.ogg` and `.m4a`; confirm the video formats for episodes.

| Format | How | Reads |
|---|---|---|
| MKV | Walk the EBML elements `Segment` → `Info`, then `Duration × TimestampScale`. If `Info` isn't in the first read, `SeekHead` gives its offset. | 1–2 |
| MP4, M4A | Walk top-level boxes by size. If `moov` is after `mdat`, its exact offset is known. Then `mvhd` gives `duration / timescale`. | 1–2 |
| FLAC | `STREAMINFO` right after the `fLaC` marker gives total samples and sample rate. Exact. | 1 |
| Ogg | The sample rate is in the first page. The last page's granule position gives the final sample, read from the end of the file. Opus uses 48 kHz, minus its pre-skip. | 2 |
| MP3 | No standard duration field. Skip the ID3v2 tag using its size (embedded art can need a second read), then use the frame count in the Xing/Info or VBRI header. Without one, estimate from file size and bitrate, which is only reliable for constant-bitrate files. | 1–2 |

- The first read is 32–64 KB.
- Files with no usable duration (stream-muxed MKVs, fragmented MP4s, unreadable audio) get `-1`.

This needs only the standard library and adds no ffprobe dependency to the server or its Docker
image. The video readers are an estimated 150–200 lines; the audio formats add to that, with MP3
the fiddliest. The existing `dhowden/tag` dependency reads tags but doesn't provide durations.

**Cost:** media is on Wasabi, whose pricing page lists no egress or API request fees. A scan is
1–2 GETs and at most a few hundred KB per file, once, then only new files. Check Wasabi's current
terms for any reasonable-use limits on egress.

**Validation:** `Offset.Duration` holds durations reported by real players, keyed by ETag.
Comparing the scan against those flags video parser problems without test files. For tracks,
compare against the MusicBrainz recording length (the search index's `length` field), which
should be close but won't match exactly.

## Schedule generation

New package `internal/guide` (type `Guide`) with its own database, shaped like `internal/progress`, added to
the `Media` struct next to `film` and `tv`. Each collection gets its own schedule, shared by the
users assigned to it.

A job in `schedule()` runs through `mediaSync`, starting immediately:

```go
mediaSync(ctx, config.Guide.SyncInterval, syncGuide, true)
```

Each run finds where every channel's schedule ends and adds programs up to now + `Horizon`, and
deletes programs that ended more than about a day ago. It's safe to re-run and catches up after
downtime. Programs already scheduled never change, so a
library scan doesn't alter what's on the guide. New titles can only land in unfilled slots, about
a week out. The one exception is a feature's reserved slot, which is filled shortly before it
airs.

Specials and features are placed first as each day is added; regular programming fills around
them, treating their start times as walls.

| Field | Notes |
|---|---|
| `ChannelID` | The channel's database ID |
| `Start` | When it starts, UTC |
| `End` | When the file actually ends, UTC. The gap before the next `Start` is a break. |
| `Kind` | `movie`, `episode`, `track`, or `reserved` for a feature slot not yet filled |
| `Block` | Name of the special or feature, empty for regular programming |
| `TMID` | Movies |
| `TVID`, `Season`, `Episode` | Episodes |
| `RID` | Break tracks |

Filling a channel:

- **Selection:** a deterministic seeded shuffle of the query results, weighted towards higher
  votes and away from titles recently on that channel.
- **Series order:** episodes run in order. The next episode is the one after the channel's last
  scheduled entry, so no cursor table is needed.
- **Repeats:** some are fine and feel like TV, but don't air the same title on two channels at
  once.
- **Alignment:** the end of each file is rounded up to the next `Align` mark (5 minutes by
  default). A 1:37:12 movie starting at 8:00 ends at 9:37:12 and the next program starts at 9:40.
  When several titles are eligible, prefer ones that leave a small gap or return the channel to
  a round time, without overriding variety.
- **Longer breaks:** a larger `Align` or a `MinBreak` gives more music. With `Align: 15m`, the
  same movie is followed by a 7:48 break and a 9:45 start. With `Align: 5m` and `MinBreak: 3m`,
  its 2:48 gap becomes a 7:48 break, also with a 9:45 start.
- **Music breaks:** the `Trailer` table only links to external sites, so it can't fill gaps.
  Each gap is filled from the `BreakMusic` query with tracks that have known durations, stored
  as `track` rows so every TV hears the same thing and can join a break partway through. The
  last track may run past the end of the gap and fades out at the next `Start`.
- **Later:** time-of-day rules (kids in the morning, R-rated after 9pm).

The fill logic should be a pure function of pool, previous entries and seed, so it can be unit
tested without S3 or a database.

## API

One endpoint:

```
GET /api/guide[?start=<RFC3339>][&end=<RFC3339>]
```

| Parameter | Default | Why |
|---|---|---|
| `start` | now | Most requests are about what's on now. |
| `end` | `start` + 3h | What's on now and next, for the surfing banner and the guide's first screen. |

| App needs | Request |
|---|---|
| Show the Live tab or not | `HasGuide` in `/api/index` |
| What's on now, for surfing | `/api/guide` |
| The guide for a day | `/api/guide?start=…&end=…`, the TV's local midnight to midnight |

- **Overlap:** a program is included if any part of it falls in the range, so at 8:10 the default
  request returns the movie that started at 7:00.
- **Days come from the app:** the server doesn't know the TV's time zone, so it only defaults the
  "now" case.
- **Limits:** `end` is capped at now + `Horizon`, since nothing is scheduled beyond it. `end`
  before `start` is an error.
- **Past programs** are kept for about a day, so a guide opened in the evening still shows the
  day's earlier programs. Older ranges return nothing.

The response contains the server's current time and the channels with programs in the range, in
`Lineup` order, each with its name and programs. Channels with nothing in the range (an
out-of-season channel, for example) are left out. A separate channel list isn't needed, since
every use of channels also needs programs; add one later only if something calls for it.

Apps must use the server's time to compute offsets, because TV clocks can be wrong.

- **Programs** carry start, end, kind, title, subtitle, rating, images, ETag and `Location`.
- **`track` rows** carry title, artist, cover and `Location`.
- **Programs in a special or feature** carry its name and kind.
- **A `reserved` slot** carries only what's known before it's filled:

```json
{"Kind": "reserved", "Block": "New on Friday", "Start": "…20:00", "End": "…24:00",
 "Description": "Movies added this week"}
```

`Description` is an optional setting on each feature.

**Safeguard:** if a guide request includes a `reserved` slot starting within 24 hours, the server
fills it before responding. A slot then never airs empty, even if the job didn't run on time
because the server was down.

Add `HasGuide` to `Index`. Existing `ApiIndexView` flags treat a missing value as "yes" for
older servers; this one must treat missing as "no".

## Apps (`takeout_native`)

### Shared core (`:shared`)

- `channels/TakeoutChannelRepository` for `/api/guide`.
- A pure `onNow(channel, time)` returning the program and offset, tested in `commonTest`, so the
  phone, Apple and Linux apps can reuse it.
- Exclude the guide from `DbResponseCache` reuse, as with progress and activity. Reload it when
  the guide opens and hourly while it's open, so a reserved slot turns into real programs
  without the viewer doing anything.

### Android TV guide

- A new top-level tab, shown only when `HasGuide` is true.
- The guide grid: channel names on the left, program cells sized by duration, a "now" line,
  details and backdrop for the focused program on top, and day switching across the week.
- Consecutive `track` rows appear as a single "Music break" cell, or no cell at all for very
  short gaps.
- Programs in a special or feature show its name as a badge across the block, so it reads as one
  event.
- A `reserved` feature slot is one cell spanning the slot with the feature's name. Focusing it
  shows the description and generic art. Watch Now isn't available until it's filled.
- Specials deserve promotion, such as a "Tonight" card on Home or a Specials row. Features mostly
  need only their name in the guide, since regular viewers know when they're on.
- Program cells have different widths, so default D-pad focus will jump around. Up and down must
  move to whatever overlaps the current time position, so the ViewModel owns focus as channel
  plus time.
- Video preview of the focused channel is probably too expensive on a Chromecast. Start with
  backdrop art.
- Follow the existing TV rules: no heavy work on the main thread, and request focus only after
  items are laid out.

### Live player

- Separate mode from the on-demand player. The existing player already opens at an offset.
- Up and down change channel in `Lineup` order.
- A banner shows the channel, title, time left and what's next.
- Program progress instead of a scrub bar.
- **Pause:** allowed. On resume show "Behind live · Jump to live". At the end of a program, jump
  to whatever is on now.
- **Watch progress:** channel viewing doesn't write progress, so Continue Watching doesn't fill
  with things someone flipped past.
- **Music breaks:** an "Up next" slate with a countdown, the track's title, artist and cover, and
  a fade-out at the next `Start`. Breaks play in the live player, not `TakeoutMusicService`, so
  `ListenTracker` never sees them and they don't count as listens.
- **Slot end:** the player stops at the next `Start` even if a file runs long (for example one
  scheduled with the `Runtime` fallback), so the channel stays on schedule.

### Fast tuning

Every channel change opens a new file, and the Chromecast is the risk.

- Seek to the nearest keyframe (`SeekParameters.CLOSEST_SYNC`); being exact to the second
  doesn't matter.
- Fix the known issue of about 3 `/location` hits per stream start.
- Show the channel banner immediately while the stream buffers.

### Features that make it work

- **Start Over and Watch Now:** any program in the guide, including future ones, can open in the
  normal on-demand player with normal progress. Real TV can't do this.
- **Open to Live TV on launch:** a setting, and what makes this work for someone used to TV.

## Music channels (later)

Start simply: all-day guide blocks that launch an existing `/api/radio` station or radio stream
in the current music player. These don't need wall-clock sync between TVs.

## Phases

1. **Durations:** `Duration` columns on movies, episodes and tracks, header readers, scan jobs,
   validation.
2. **Server schedule:** `Guide` config, `internal/guide`, `/api/guide`, `HasGuide`. Testable before any TV work.
3. **TV live player:** `:shared` repository, channel surfing and a simple "On Now" list, to prove
   tuning speed on the Chromecast before building the grid.
4. **Full guide:** the grid, day browsing, Start Over, Open to Live TV.
5. **Richer programming:** specials (themes, then automatic collections), features with late
   filling, music breaks, music channels.
6. **Other surfaces:** phone and other apps, Android TV home-screen integration, optional
   automatic genre channels.

## Implementation notes

Phase 2 as built, where it differs from or fills in the design above:

- **Names:** the package is `internal/guide` and its main type is `Guide`, like `film.Film`.
  The config section is `Guide:` (with `Lineup`, `Timezone`, `Horizon` and so on), the index
  flag is `HasGuide`, and the job is `guide`. Database rows are `model.Channel` and
  `model.Program`. `guide.Listing` is one channel's programs in a time range. The
  `/api/guide` response is `view.Guide` with `Channels` (`view.Channel`), each with its `Name`
  and `Programs` (`view.Program`).

- **Queries** are bleve query strings. Plain terms are alternatives: `rating:G rating:PG`
  matches either. Prefix `+` to require a term: `+genre:Comedy +vote:>7`.
- **Genres are case-sensitive:** `genre` and `keyword` are exact-match fields, and movie genres
  are stored as TMDB names them, so `genre:Action` and `genre:"Science Fiction"` match but
  `genre:action` matches nothing.
- **Show queries:** the TV index holds only episode fields (`series`, `season`, `episode`,
  `title`, `rating`, `vote`, cast and crew), not series genres or keywords, so
  `Shows: "genre:Comedy"` matches nothing today. Name the series instead:
  `Shows: 'series:"The Office" series:"Parks and Recreation"'`. Adding series genres and
  keywords to each episode's index fields in `syncEpisode` would fix this, after a full TV
  re-sync.
- **Config:** `Guide.SearchLimit` (default 5000) is the pool search limit. Per-channel
  `Align` and `MinBreak` override the collection's. `BreakMusic` isn't read yet (phase 5).
- **Specials** (`internal/guide/special.go`): a channel's specials are its `Themes`, then, with
  `Collections`, every collection with at least `MinFilms` of the channel's pool movies, less
  `Skip`. A collection plays every film owned from it in release order; a theme plays its
  query in release order with `Order: date`, else shuffled (a series' episodes stay in order).
  When `Every` has passed since the channel's last special (the first is due as soon as
  specials are configured), the job picks an evening 0–3 days later at `At` and records it in
  `special_airings`, which outlives the programs. It picks the special that has gone longest
  without airing on the channel, skipping one that airs on another channel within 7 days and
  one with fewer than 2 programs. Specials need at least one regular schedule run to be
  placed, so the first lands after the end of the existing schedule. Each run places every
  channel's specials before filling any channel, so regular programming everywhere avoids
  their titles at the time; regular programming before a special ends in time for it, with
  a longer break if nothing fits. The special's programs, laid out back to back until the
  next wouldn't end within `MaxLength`, carry its name in `Block`; their titles then count
  toward the usual no-repeat window. They aren't kept off the days before the special.
  A special moves to a later evening, up to a week, rather than overlap one of the channel's
  features.
- **Features** (`internal/guide/feature.go`): each day matching a feature's `Layout`/`Match`
  (every day without a `Layout`) gets a slot at `At` for `Length`; when two overlap on a day,
  the first listed wins. As the schedule extends, each slot becomes one `Kind: "reserved"`
  program with the feature's name in `Block`, and the guide shows it with its `Description`.
  Regular programming looks a day past the schedule's end for slots, so nothing runs into one.
  A run fills each slot starting within 24 hours: titles from `Source` (`added`/`released`,
  newest first) or the `Movies`/`Shows` queries (shuffled), skipping ones the feature has
  shown on that channel (`feature_titles`), ones that don't fit what's left, and ones on
  elsewhere then. A leftover of 30 minutes or more is regular programming, a shorter one a
  break. With nothing new the whole slot is regular programming without the feature's name.
  A slot is reserved in one run and filled in a later one.
- **Breaks:** gaps between programs are left empty; the apps can show a slate until music
  breaks are scheduled.
- **Tables** are `channels` and `programs` in `guide.db` next to the other media
  databases. The program times are columns `start_at` and `end_at`, since `end` is an SQL
  keyword. JSON still uses `Start` and `End`.
- **Alignment** is from local midnight in `Timezone`, so marks stay on round local times in
  zones with half-hour offsets and across DST changes. Time zone data is embedded in the
  binary, so `Timezone` works in minimal containers.
- **Selection:** movies and series are picked by weighted random draw, seeded by channel name
  and start time. Weights favor higher votes, slightly favor titles that leave a small gap or
  land on :00/:30, and exclude a movie aired on that channel within half the pool's running
  time (between 1 and 7 days). A series isn't picked twice in a row when others are
  available, and becomes more likely the longer since it last aired. Episodes run in order
  and wrap around; season 0 is skipped.
- **Runtime fallback:** a title without a scanned duration uses its TMDB runtime plus 2
  minutes. Titles with neither aren't scheduled.
- **After downtime** the schedule restarts at the current mark rather than filling the past,
  so the guide may show a gap.
- **Config changes** only affect programs not yet scheduled, up to a week out.
- **Guide programs** carry `TargetID` (the movie or episode row, per `Kind`, for details and
  Watch Now), `TMID` or `TVID`/`Season`/`Episode`, `Title`, `Subtitle` (tagline or episode
  name), `Rating`, `Image`, `Background`, `ETag`, `Size` and `Location`. A title that has left
  the library is `Kind: "offair"` with only its times. The response's `Time` is the server time in
  milliseconds, as in `/api/index`.
