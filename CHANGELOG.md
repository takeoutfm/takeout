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
