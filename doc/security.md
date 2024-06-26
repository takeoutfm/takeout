# Takeout Security Design

## Authentication

User credentials are userid and password. The password is stored in the auth
database using a random salt and key encrypted with scrypt as follows:

    scrypt.Key([]byte("some password"), salt, 32768, 8, 1, 32)

During authentication, the user provides their userid and password. The salt
for the userid and password are used to generate an scrypt key to perform a
byte-wise comparison with the stored key. An exact match is required for
successful authentication.

The userid can be a name, email or any unique indentifier.

There are currently no rules for password entropy, but you should obviously use
strong passwords.

Each user has a list of media bucket names they are allowed to access. This is
the only level of media access control.

## Two-Factor Authentication (2FA)

Takeout supports time-based one-time passwords (TOTP) for 2FA. The 6 digit code
which changes every 30 seconds is referred to as a *passcode*.

Users can optionally be assigned a TOTP using the *takeout user* command. Once
assigned, the corresponding passcode must be provided along with the userid and
password. The QR code and otpauth URL are printed to the screen when assigned.
Use your mobile authenticator app to scan the QR code or enter secret as
needed. By default a period of 30 seconds, 6 digits, and SHA1 are used. An
issuer must be provded in the *Auth.TOTP.Issuer* configuration.

## Sessions

An authenticated user login will create a unique session that is stored in the
auth database. Each session has an associated v4 UUID with 122 random bits.
This UUID is the session token. Sessions expire based a configurable duration
of time. (default is 30 days)

## Authorization

Token types:

- Cookie - Unique token used for Cookie authorization (value is session UUID)
- Refresh Token - Unique token to refresh access and media tokens (value is session UUID)
- Access Token - JWT used to authorize API calls
- Media Token - JWT used to authorize media access
- Code Token - JWT used to authorize code access
- File Token - JWT used to authorize file access (for local file buckets)

A success login will return three tokens: Access, Media, Refresh. File tokens
are embedded in URLs to access local media files. Note that S3 buckets have
their own tokens generated by the S3 service.

The *Access Token* is intended to be short-lived (a few hours) and is used to
authorize access to API calls. The *Access Token* can be refreshed using the
*Refresh Token* which should be valid for days (default is 30). The *Media Token*
is used only during playback to authorize media access. The *Media Token* is
designed to be long-lived (months) since media access URLs can only be obtained
using using an *Access Token*.

The *Cookie* and *Refresh Token* are intended to expire and force a re-login.

## Codes

The Takeout server can generate temporary codes to easily authenticate devices
like watches and TVs. The watch/TV app sends a code request and receives a 6
character code along with a *Code Token*. The code can be used to login with
another device using valid userid and password. After successul login, the
watch/TV app can use the *Code Token* and obtain all the required tokens. Code
should be short-lived (minutes), just enough time to allow someone to login on
another device.

## Media Access

The Takeout server provides access to media using location URLs. These URLs
offer a level indirection to the actual media URLs and can be cached. A media
player can be preloaded with many location URLs and with a valid *Media Token*,
can access the actual media when needed via authorized HTTP redirect.

The overall goal is protect media and enable players to work without much
hassle. *Access Tokens* are required to obtain location URLs and *Media Tokens*
are required to locate the actual media URLs. Location URLs are not guessable
and in most cases have a random UUID. Media URLs are unique, signed, and
time-based, using embedded tokens.

### S3 Media

Media stored in S3 is accessed using pre-signed time-based URLs, generated
using the S3 bucket credentials. Each media object URL is unique and limited
use.

### File Media

Media stored in local files is accessed using a *File Token*. This token
includes a signature of the file being accessed along with expiration time.
Each media file URL is unique and limited use.
