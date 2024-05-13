# Takeout Quick Start

## Overview

The *takeout run* command can be used to quickly setup and run takeout. This
process will use command line options, environment variables, and/or simple
yaml files to create the required configuration as described in
[setup.md](setup.md) and start the takeout server.

After using *takeout run* you can switch to using *takeout serve* with the
created configuration, as described in [setup.md](setup.md).

Use the *--setup_only* option to only create configuration without starting the
server.

## Quick Start

Download and install Go from [https://go.dev/](https://go.dev/) as needed. Run
the following command to ensure Go is correctly installed. You should have Go
1.22 or higher.

```console
$ go version
```

Download and build the Takeout server from Github.

```console
$ go install github.com/takeoutfm/takeout/cmd/takeout@latest
```

This will build & install Takeout in $GOPATH/bin. Don't worry if you don't have
a GOPATH environment variable defined, Go will default to your home directory
(~/go/bin). Ensure that $GOPATH/bin is in your command path. You should see a
Takeout version displayed.

```console
$ takeout version
```

Now you can use *takeout run* to setup and start takeout. Below are some
example commands. Please see *takeout run --help* for further details.

```console
$ takeout run --dir /home/takeout --music s3://mybucket/Music --endpoint s3.someprovider.net --region us-west --access_key_id MY_KEY_ID --secret_access_key MY_SECRET_KEY --log /tmp/takeout.log
```

```console
$ takeout run --dir /home/takeout --music /media/music --movies /media/movies --log /tmp/takeout.log
```

Note that S3 credentials can be obtained from the following environment variables:

* AWS\_ENDPOINT\_URL
* AWS\_DEFAULT\_REGION
* AWS\_ACCESS\_KEY\_ID
* AWS\_SECRET\_ACCESS\_KEY

Options can also be stored in a yaml configuration file located here:

* $HOME/.config/takeout/config.yaml
* $HOME/.takeout/config.yaml

Supported file options are the same as the command line options, without the
leading dashes.
