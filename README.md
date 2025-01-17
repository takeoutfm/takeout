# TakeoutFM

TakeoutFM is a copyleft, secure, and private media system that allows you to
manage and stream your media on your own terms. It has a small and fast server,
with mobile, watch, tv, and console apps for media streaming on your devices.
Support for storing media in the cloud using S3 is a primary design goal and
local media files are supported as well. See
[takeoutfm.com](https://takeoutfm.com/) for further details.

## Takeout

The TakeoutFM server, known as Takeout, indexes organized media files in S3 (or
local) using MusicBrainz, Last.fm, Fanart.tv, and TMDB. Media is browsed using
Takeout and securely streamed directly from S3 or local storage. Music, movies,
TV shows, podcasts, and radio are supported. Takeout is built as a single
binary that includes all server functionality including media syncing,
streaming, REST APIs, and a builtin web UI.

## Features

* Free and open source with AGPLv3 license
* Written in [Go](https://go.dev/), with
  [SQLite3](https://sqlite.org/index.html), [Bleve](https://blevesearch.com/),
  [Viper](https://github.com/spf13/viper), and
  [Cobra](https://github.com/spf13/cobra)
* Music metadata from [MusicBrainz](https://musicbrainz.org/) and
  [Last.fm](https://last.fm/)
* Album covers from the [Cover Art Archive](https://coverartarchive.org/)
* Artist artwork from [Fanart.tv](https://fanart.tv/)
* Powerful [search](doc/search.md) and playlists
* Movie & TV metadata and artwork from [The Movie Database (TMDB)](https://www.themoviedb.org/)
* Podcasts with series and episode metadata
* Internet radio stations (pls)
* Media streaming directly from S3 using pre-signed time-based URLs
* Media streaming for local files using JWT tokens
* User-based access control using cookies, JWT tokens, and 2FA using TOTP
* Server-based playlist API using [jsonpatch](http://jsonpatch.com/)
* [XSPF ("spiff")](https://xspf.org/) and JSPF playlists
* Supports [caching](https://github.com/gregjones/httpcache) of API metadata
  for faster (re)syncing
* REST APIs are available to build custom interfaces

## Documentation

More details are available in the *doc* directory, including [quick
start](doc/run.md), [setup documentation](doc/setup.md), [security
design](doc/security.md), and [media naming conventions](doc/bucket.md).

Visit the GitHub [organization page](https://github.com/takeoutfm) for all
project source code repositories.
