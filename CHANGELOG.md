## 0.31.1

- channels: Guide.MinContent (default 48h) sets how much content a channel
  needs to be scheduled, in place of the fixed two days, and a Lineup entry
  can set its own, such as a lower one for a channel that loops a short
  series

## 0.31.0

Channels: a TV-style guide of what's on, scheduled from your library and
played live by the apps. The design and configuration are in CHANNELS.md.

- channels: a schedule of movies and shows from search queries, set up per
  media collection under Guide.Lineup (Name, Movies and/or Shows queries,
  optional Layout/Match date window for seasonal channels, Align, MinBreak)
  with Guide.Timezone, Horizon (168h), Align (5m), MinBreak and SyncInterval
  (1h); the "guide" job extends each schedule and runs at startup. Series
  play in order, movies don't repeat for a while, and the same title isn't
  on two channels at once
- channel specials: Lineup entries can have Specials (Every, At, MaxLength,
  Collections, MinFilms, Skip, Themes with Name, Movies/Shows and Order:
  date) for an occasional themed block, such as a film collection marathon,
  rotated so each comes round in turn; its programs have Block set to the
  special's name
- channel features: Lineup entries can have Features (Name, Layout/Match
  days, At, Length, Source added or released, or Movies/Shows, and
  Description), a regular slot reserved in the guide as Kind "reserved"
  with the feature's name and filled about a day ahead with titles the
  feature hasn't shown; with nothing new it becomes regular programming
- channel music breaks: gaps between programs are filled with tracks from
  Guide.BreakMusic or a Lineup entry's BreakMusic (any track if unset) that
  have a scanned duration, as Kind "track" programs with title, artist,
  cover and Location
- programming before a special or feature slot picks titles that leave the
  least dead air, rather than leaving a break no title fits
- GET /api/guide[?start=&end=] returns Channels, what's on each one (default
  now to 3 hours ahead, RFC3339 times, capped at the horizon) with each
  program's Kind (movie, episode, track, reserved or offair for titles no
  longer in the library), title, images, Block and Location, and the
  server's Time; it fills a feature slot about to air if the guide job
  hasn't
- GET /api/index has HasGuide
- movies, episodes and tracks have a Duration in seconds, read from the file
  headers (mkv, mp4/m4a, flac, ogg, mp3) with range requests; new hourly jobs
  scan files in batches (Film/TV/Music.DurationSyncInterval and
  DurationSyncLimit), also runnable as the "durations" job. 0 means not
  scanned yet (as does an empty value in databases from before this
  release, which the jobs also scan) and -1 means the duration couldn't be
  read

## 0.30.2

- GET /api/code returns ExpiresIn (seconds until the code expires) and
  Interval (seconds to wait between POST /api/code checks)

## 0.30.1

- added /api/me

## 0.30.0

- new web UI, still built into the server: pages are rendered from templates
  with real URLs (/releases/12, /tv/series/4, /search?q=...) and swapped in
  place, so playback continues while browsing; the old /v?... pages are gone
- web UI layout follows the Mac and Linux apps: sidebar, page headers with
  Play, Shuffle, Radio and Favorite, tile grids, track lists, and a mini
  player with a Now Playing screen and queue
- new pages: Recently Added, New Releases, Favorites; Activity redesigned
- music, radio and podcasts play in the web UI; movies and TV are browse-only
  (watch in the apps)
- web player uses the server playlist, counts listens like the apps (4 minutes
  or half the track), and saves and resumes podcast progress
- podcast episode page with Resume, Start Over and Mark as Played
- sign out, redesigned sign-in, and Link a device (no password needed when
  already signed in)
- session cookie is always sent with Path=/, HttpOnly, Secure and
  SameSite=Strict; stale per-path cookies from older versions are cleared
- Content-Security-Policy now allows only the server's own scripts and styles
- static files served from memory, gzipped, with hashed URLs; Chart.js
  replaced by a small SVG chart
- sharper TMDB images for posters, stills and cast photos

## 0.29.6

- claude security updated

## 0.29.5

- updated auth db indices

## 0.29.4

- use txn for favorite inserts
- updated all db indices with help from claude

## 0.29.3

- fix delete

## 0.29.2

- fav artist fix

## 0.29.1

- fav fixes

## 0.29.0

- ListenBrainz needs authorization header now
- Add support for favorite artists, movies, tracks and shows

## 0.28.0

- added MovieGenres to index
- added HasRecommendMovies to index

## 0.27.5

- strip quotes in search when necessary
- use year in tmdb search to match "the dead (2010)"
- fix tests

## 0.27.4

- fix track bg cache

## 0.27.3

- added background to spiff and spiff tracks

## 0.27.2

- added "artwork" job

## 0.27.1

- added Background to ReleaseView
- added Image and Background to Artist
- added support for Authorization header as query parameter (tentative)
- activity errors with Errorf

## 0.27.0

- upgrade dependencies, including aws s3 sdk v2 changes
- need to prefix http:// or https:// with s3 endpoints now
- added context parameters to various calls to allow for future improvements
- use go 1.26.1
- added preferred ReleaseFormats; defaults to prefer CD over Digital Media
- more music track file sync improvements

## 0.26.2

- removed CheckError/panic usage in sync; replace with explicit error checks

## 0.26.1

- increase default track limit for moderate 1 year of tracks (9999)
- really fix the RSS title and author decoding
- use gopxl/beep in player
- mod updates
- use go 1.24.2

## 0.26.0

- removed docs, now on takeoutfm.com
- include guest stars in tv credits
- systemd support
- updated all dependencies

## 0.25.1

- fix auth indices

## 0.25.0

- use takeoutfm.dev

## 0.24.6

- HTML unescape RSS title and author
- ensure good password entropy
- fix users and code unique index

## 0.24.4-5

- support TV list and TV series calls

## 0.24.3

- preserve TV series & episode IDs during resync

## 0.24.2

- fix people links
- test fixes
- added "images" job

## 0.24.1

- support for TV shows
- refactor video to film
- support movie trailers

## 0.23.5

- refactor activity db count grouping
- use this sqlite statement to update existing activity dates to UTC

  `update track_events set date = datetime(date, 'utc')`

## 0.23.4

- use UTC for activity events
- move date group by and counting outside of sqlite and use go instead

## 0.23.3

- fix month math and labels

## 0.23.2

- more chart fixes and colors

## 0.23.1

- fixes for year and month charts

## 0.23.0

- charts

## 0.22.3+

- return errors for invalid activity events
- use listen rules in web player

## 0.22.2

- small activity updates

## 0.22.1

- increment artist/release counts by track count not one

## 0.22.0

- add totals and counts to track stats
- order top activity by date also
- adjust activity track limits

## 0.21.2+

- return non-nil slices when resolving events
- added html view for activity
- more fixes to html
- call fix missing in music covers job

## 0.21.1

- fix top artists to use name not arid

## 0.21.0

- Support smallest or largest movie duplicate resolution
- Refactored activty to stats and history
  * /api/activity/tracks/{today,yesterday,lastweek,year,etc.}
  * /api/activity/tracks/.../stats

## 0.20.1

- fix activity recent tracks

## 0.20.0

- add counts in activity popular tracks

## 0.19.4

- store activity dates in local timezone
- check validitiy of activity events
- different track etag lookup

## 0.19.3

- fix indexing: mbz different "track" vs "tracks" in json responses
- return bad request with invalid search

## 0.19.2

- changed track matching to use musicbrainz to resolve questionable tracks
- fully ignore video media in music releases
- fixes multiple release handling with different media dates (Substance 1987)
- handle multi-artist searching earlier
- support artist aliases: Kanye West is now Ye

## 0.19.1

- disable lastfm artist search during sync

## 0.19.0

- fix to not include unresolved activity events in response
- updated activity pointer and error handling
- added smarter artist matching
  * unpack names with `&`
  * search with mbz aliases
  * search with recording to derive arid
  * limit artists matches and err earlier
- added smarter release matching
  * use per media track numbers and position
- add index on releases.artist

## 0.18.6, 0.18.7

- fixed first track bug recently introduced

## 0.18.5

- more pointer/error refactoring

## 0.18.4

- refactoring
  * updates to pointer and error handling
  * added playlist Length() & Empty()

## 0.18.3

- added RelatedArtists
- use related artists when there are no similar artists

## 0.18.2

- don't use track ARID since it's not populated for s3 buckets yet

## 0.18.0, 0.18.1

- use better random seed for codes
- use new random w/ seed for shuffling
- added track-based radio
  * /api/tracks/{id}/playlist
- added config for TrackRadioBreadth, TrackRadioDepth

## 0.17.6

- add playlists to web ui
- de-dup playlists
- add has playlists to index

## 0.17.5

- add track count to playlists

## 0.17.4

- fix nulls found in embedded tags

## 0.17.3

- fixed api_test
- fixed REID not found during assignment

## 0.17.2

- fixed serve listen option
- fixed spiff json track null after resolve
- fixed assigned track release dates

## 0.17.1

- fix *takeout run* video option

## 0.17.0

- perform *takeout run* syncs in the background
- added control requests via unix socket
  * profiling
  * config
  * run jobs
  * curl -s -N --unix-socket /tmp/takeout.sock http://takeout/config/mymedia
  * curl -s -N --unix-socket /tmp/takeout.sock 'http://takeout/debug/pprof/goroutine?debug=2'
- added listenbrainz popular
- fixed *takeout run* user
- added user playlist API, with tests
  * GET /api/playlists
  * POST /api/playlists
  * GET /api/playlists/{id}
  * GET /api/playlists/{id}/playlist
  * PATCH /api/playlists/{id}/playlist
  * DELETE /api/playlists/{id}
- added view model *Playlist* and *Playlists* with ID and Name
- added playlist ref */music/playlists/1*
- removed older */api/login* API, apps used */api/token* or */api/code*

## 0.16.0

- added "takeout user --user=defsub --link=code"
- improved search for artists, releases and stations

## 0.15.1

- support for local file music metadata instead of naming conventions

## 0.15.0

- added TOTP support, optional but recommended
- added "takeout user --user=defsub --generate\_totp"
- added /api/link for app-based code linking

## 0.14.4, 0.14.5

- fixed embeds
- expanded "takeout run" options
- added "takeout user --user userid --expire" to expire all sessions

## 0.14.3

- fix template station ref
- code cleanup
- fix disambiguation and release countries

## 0.14.1, 0.14.2

- improved $ support in configs
- added "takeout run"

## 0.14.0

- use Go 1.22 routing enhancements (and remove pat)
- breaking API changes:
  * changed /api/radio/stations/{id} to /api/stations/{id}
  * changed /api/movies/genres/{name} to /api/movie-genres/{name}
  * changed /api/movies/keywords/{name} to /api/movie-keywords/{name}
- updated to go 1.22.2
- support local filesystem buckets
- breaking config change buckets, s3 and fs have own configs now
- added server config include & exclude dirs
- support config include directive
- removed SomaFM streams from default config
- removed recommended from defeault config

## 0.13.2

- Added unit tests, currently 23.9% coverage
- updated to go 1.22.0, updated all module dependencies

## 0.13.1

- fix for spaces in station name ref
- add description to station model

## 0.13.0

- support best match (&m=1) in search refs
- support radio stations in search refs
- playout: support track activity; config enableTrackActivity=true (default false)
- playout: added config enableListenBrainz=true (default false)
- fix client with sync.Map for concurrent writes

## 0.12.2

- fix playout radio stream selection

## 0.12.0, 0.12.1

- support multiple radio stream formats and enable client to choose
- updated video recommendation keywords
- support podcast episode images
- include all SomaFM streams in default config with multiple stream types
- updated to go 1.21.6, updated all module dependencies

## 0.11.10, 0.11.11

- fix redirects

## 0.11.9

- order podcast subscriptions by date desc
- added redirects for /login, /login.html, /link, /link.html

## 0.11.8

- added podcasts subscriptions
