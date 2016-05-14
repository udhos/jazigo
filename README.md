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

2\. Get dependencies

    go get github.com/icza/gowut/gwu
    go get github.com/udhos/lockfile
    go get gopkg.in/yaml.v2
    go get golang.org/x/crypto/ssh

3\. Get source code

`go get github.com/udhos/jazigo`

4\. Compile and install

`go install github.com/udhos/jazigo/jazigo`

5\. Decide where to store config, backup and log files

Example:

    # These env vars are not meaningful to jazigo.
    # They're just handy pointers used in step 5 below.
    export APP_HOME=/var/jazigo
    export APP_CONF=$APP_HOME/etc/jazigo.conf. ;# last dot required
    export APP_REPO=$APP_HOME/repo             ;# backup repository
    export APP_LOG=$APP_HOME/log/jazigo.log.   ;# last dot required
    mkdir -p $APP_HOME/etc $APP_REPO $APP_HOME/log

6\. Run jazigo once (see -runOnce option)

`$GOPATH/bin/jazigo -configPathPrefix $APP_CONF -repositoryPath $APP_REPO -runOnce`

Watch messages logged to standard output for errors.

7\. Run jazigo forever

`$GOPATH/bin/jazigo -configPathPrefix $APP_CONF -repositoryPath $APP_REPO`

8\. Open the web interface

Point web browser at: [http://localhost:8080/jazigo](http://localhost:8080/jazigo)
