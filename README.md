About Jazigo
=============

Jazigo is a tool written in Go for retrieving configuration for multiple devices, similar to [rancid](http://www.shrubbery.net/rancid/), [fetchconfig](https://github.com/udhos/fetchconfig) and [oxidized](https://github.com/ytti/oxidized).

Supported Platforms
===================

Please send pull requests for new plataforms.

- [Cisco IOS](https://github.com/udhos/jazigo/blob/master/dev/model_cisco.go)
- [Cisco IOS XR](https://github.com/udhos/jazigo/blob/master/dev/model_cisco_iosxr.go)
- [Juniper JunOS](https://github.com/udhos/jazigo/blob/master/dev/model_junos.go)
- [HTTP](https://github.com/udhos/jazigo/blob/master/dev/model_http.go) (collect output of http GET method)
- [Linux](https://github.com/udhos/jazigo/blob/master/dev/model_linux.go) (collect output of SSH commands)

Features
========

- Written in [Go](https://golang.org/).
- Spawns multiple concurrent lightweight goroutines to quickly handle large number of devices.
- Very easy to add support for new platforms.
- Configured with [YAML](http://yaml.org).
- Backup files can be accessed from web UI.

Quick Start
===========

1\. Setup GOPATH as usual

Example:

    export GOPATH=~/go
    mkdir $GOPATH

2\. Get source code

`go get github.com/udhos/jazigo`

3\. Compile and install

`go install github.com/udhos/jazigo/jazigo`

4\. Decide where to store config and backup files

Example:

    # These env vars are not meaningful to jazigo.
    # They're just handy pointers used in step 5 below.
    export APP_ETC=/etc/jazigo            ;# app config dir
    export APP_CONF=$APP_ETC/jazigo.conf. ;# last dot is required
    export APP_REPO=/var/jazigo/repo      ;# backup repository
    mkdir -p $APP_ETC $APP_REPO

5\. Run jazigo once (see -runOnce option)

`$GOPATH/bin/jazigo -configPathPrefix $APP_CONF -repositoryPath $APP_REPO -runOnce`

Watch messages logged to standard output for errors.

6\. Run jazigo forever

`$GOPATH/bin/jazigo -configPathPrefix $APP_CONF -repositoryPath $APP_REPO`

7\. Open the web interface

Point web browser at: [http://localhost:8080/jazigo](http://localhost:8080/jazigo)
