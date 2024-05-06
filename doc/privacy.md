# TakeoutFM Privacy Policy

TakeoutFM is a copyleft personal media system that may be entirely managed by
the end user, a service provider, or a combination of the two. Privacy as it
relates to the original intended design of TakeoutFM, independent to how it is
managed, is described within this document. TakeoutFM refers to the Takeout
server, mobile apps, TV apps, watch apps, console apps, and the web interface.

If you have questions regarding this privacy policy, please contact
privacy@takeoutfm.com.

## Design Considerations

TakeoutFM is designed around a server and APIs for clients/apps to browse and
consume media. Clients are not required to store any information but may do so
to improve performance. State may be stored on the server such that activity
and progress can be resumed or shared across multiple client devices.

### Media Access

When using media stored in S3, the Takeout server does not store or manage any
media and instead media in the S3 bucket is available for the server to index
and for clients to access directly using time-based pre-signed URLs. The
Takeout server does not directly access any S3 media.

When using local media files, The Takeout server will index media directories,
compute ETags for media files, and serve media files using time-based
pre-signed URLs.

## Personal Information

The Takeout server requires a userid and password for each user to access
related media and services. The userids and passwords are stored in the server
*auth* database. The userid is stored in the clear and the password is stored
with [Scrypt](https://en.wikipedia.org/wiki/Scrypt). No other personal
information is requested or stored by TakeoutFM.

The Takeout server may temporarily store access logs that contain client
request information and IP addresses. This information, if used, is only used
for debugging or development purposes.

The Takeout server is recommended to be configured with TLS to ensure all
communication is encrypted to avoid unintended disclosure of userids,
passwords, cookies, and tokens.

## Cookies and JWTs

Cookies are small tokens or files stored on your device as part of the user
login process to uniquely identify you later without requiring you to provide
your userid and password again. Each cookie is a UUID comprised of 122 random
bits, stored within the Takeout server *auth* database, and within the client
app or web browser. Cookies are valid for a limited time (based on server
configuration) and when expired, you will be required to login again.

The Takeout server uses JWTs for API and media access authorization. JWTs are
not stored by the server, however, they are stored on app/client devices
similar to cookies. The *Refresh Token* is the same as a cookie and stored by
the Takeout server. Please see the security design for further details.

JWTs claims are plain-text within the token. The Takeout server uses subject
and audience claims which specificy a userid and media file name respectively.
Unintended disclosure of a JWT could explose a userid or media file name.

The Takeout server is recommended to be configured with TLS to ensure all
communication is encrypted to avoid unintended disclosure of cookies and
tokens.

## Media in S3

The Takeout server requires access to your S3 bucket to obtain a listing of
media stored within the S3 bucket. The bucket object file names are used to
obtain further metadata related to music and video files. These object names
are stored in the corresponding *music*, *video* and *search* databases to
enable media streaming or downloading directly from your S3 bucket using
time-based pre-signed URLs.

The Takeout server does not access your media, it does not parse your media
containers, and it does not parse any embedded tags or related information in
your media. All related metadata is obtained using third-party services based
on file naming conventions.

Media stored in your S3 bucket can potentially be visible to the S3 bucket
service provider. Contact your service provider to obtain further information
regarding the S3 bucket privacy policy. Personal S3 bucket hosting options,
such as [Minio](https://min.io/), are available.

Local media can also be used without requiring S3. However, similar to when
using S3, file names are stored in databases and files are streamed using their
original file names with embedded tokens.

## Metadata

The Takeout server uses the following services to discover your media metadata:

- [Cover Art Archive](https://coverartarchive.org/) - obtain links to cover images
- [Fanart.tv](https://fanart.tv/) - obtain links to Artist images (requires API key)
- [MusicBrainz](https://musicbrainz.org/) - obtain music metadata
- [Last.fm](https://www.last.fm/) - obtain popular tracks, artist name resolving (requires API key)
- [The Movie Database](https://www.themoviedb.org/) - obtain movie metadata (requires API key)

The Takeout server uses the respective service APIs to query and store related
metadata based on your media object/file names. Requests to the service APIs
will include an API key (where required), media information (such as artist or
movie name), and the Takeout server IP address. Third-party services can infer
information about your music or movies that are being indexed and potentially
relate the media to a unique IP address. No other information is directly
provided to these third-party services.

Metadata related to your object/file names is stored in the respective *music*,
*video*, and *search* databases to improve performance and reduce the overall
impact on third-party services. Similarly, API responses can also be cached to
avoid repeated or duplicate requests for the same information.

Metadata includes links or URLs to images such as covers, posters, and
profiles. The Takeout server includes such URLs or information to construct
URLs in responses to API clients. The URLs are used by clients to render
associated media images in their UI. Third-party services can infer information
about your media from image requests and relate to a unique IP address. No
other information is directly provided to these third-party services.

Podcast URLs that have been added to the Takeout server configuration are
periodically queried and metadata is stored in the *podcast* database. Requests
to the podcast providers can infer information about your interests and
potentially relate the podcasts to a unique IP address. No other information is
directly provided to the podcast providers.

Radio stream URLs that have been added to the Takeout server configuration are
used to create a radio station in the *music* database. The URLs for these
streams are sent directly to API clients and clients can use them to directly
stream media. Requests to radio stream providers can infer information about
your interests and potentially relate streams to a unique IP address. No other
information is directly provided to the radio providers.

## Progress

The Takeout server provides APIs for clients to store media watch/listen
progress which is intended to allow playback to be conveniently resumed on the
same or other devices at a later time. The Takeout server stores progress using
the ETag (or entity tag) of the media and an offset in the media stream. An
ETag is generally an MD5 digest of the media content obtained from the S3
bucket or directly computed from a local file. This design of ETag based
progress provides a layer of indirection such that user media consumption is
not readily available. It's possible, with access to the *progress* database
and the S3 bucket or local files, to reconstruct media consumption information
by mapping progress ETags to media.

## Activity

The Takeout server provides APIs for clients to optionally store activity
events which are intended to allow the user to easily access recently consumed
media on the same or other devices. Activity events can relate to music, video,
and podcasts. Activity events are stored in the *activity* database and each
event uses third-party identifiers (MBIDs, GUIDs, IMDB IDs) such that activity
data is stable. It's possible, with access to the *activity* database, to
reconstruct media consumption information by mapping third-party identifiers to
actual metadata.

## ListenBrainz

Takeout apps/clients can be optionally enabled to submit music listening
activity to the ListenBrainz service. Please refer to the ListenBrainz goals
and terms of service for further information.

## Assistant

The Takeout assistant app is intended to be an offline voice assistant that
supports voice commands for media and some limited home control. All voice
recording and processing is local (offline) and does not use any cloud
services. Voice data is not stored or sent to any third-party services. Voice
data is converted to text and processed internally. Portions of this text may
be sent to the Takeout server for further processing such as searching media to
create playlists.

Home control support as of this writing is limited to the Philips Hue Bridge
(and lights) which is locally accessed directly without any other apps or
remote services. Please refer to the Philips Hue Bridge documenation for
further information.

## Information Disclosure

TakeoutFM does not directly disclose any information to any outside parties
beyond what is needed to obtain metadata or access media.

## Children’s Online Privacy Protection Act Compliance

TakeoutFM is directed at people that are 13 years old or older. If the Takeout
server is in the USA, and you are under age of 13, per the requirements of
COPPA (Children’s Online Privacy Protection Act), do not use the Takeout
server.

## Consent

By using TakeoutFM, you consent to this privacy policy.

## Changes

Any changes made to this privacy policy will be made available on
[github](https://github.com/takeoutfm/takeout/tree/main/doc/).
