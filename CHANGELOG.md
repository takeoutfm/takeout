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
