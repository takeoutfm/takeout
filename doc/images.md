# Images

The Takeout server uses server-relative image URLs to refer to metadata
images. A client can contruct these URLs using metadata information to obtain
images directly from the Takeout server. The image responses can be pre-cached
and will return the cached image data, or the response will be a redirect to
the original metadata image.

The */img* handler only serves pre-cached images. There are jobs that run in
the background that work to download and cache images. These jobs can also be
run manually as needed using the `takeout job` command.

## Cover Art Archive

```
/img/mb/rg/:rgid
/img/mb/rg/:rgid/:side
/img/mb/re/:reid
/img/mb/re/:reid/:side
```

Key:
* rgid - MusicBrainz release group ID (rg)
* reid - MusicBrains release ID (re)
* size - front (default), back or other


## The Movie Database

```
/img/tm/:size/:path
```

Key:
* size - w154 (poster small), w342 (poster), w1280 (backdrop), w185 (profile)
* path - TMDB poster/backgdrop/profile image path

## Fanart

```
/img/fa/:arid/t/:path
/img/fa/:arid/b/:path
```

Key
* arid - MusicBrainz artist ID
* path - Fanart thumb (t) or background (b) image path.
