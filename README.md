[![GoDoc][https://godoc.org/github.com/udhos/jazigo?status.svg](https://godoc.org/github.com/udhos/jazigo)]
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/jazigo)](https://goreportcard.com/report/github.com/udhos/jazigo)

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
- Support for SSH and TELNET.

Requirements
============

- You need a [system with the Go language](https://golang.org/dl/) in order to build the application. There is no special requirement for running it.

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

    export JAZIGO_HOME=/var/jazigo
    mkdir -p $JAZIGO_HOME/etc $JAZIGO_HOME/repo $JAZIGO_HOME/log

Hint: See command line options to fine tune filesystem locations.

6\. Run jazigo once (see -runOnce option)

`$GOPATH/bin/jazigo -runOnce`

Watch messages logged to standard output for errors.

7\. Run jazigo forever

`$GOPATH/bin/jazigo -disableStdoutLog`

8\. Open the web interface

Point web browser at: [http://localhost:8080/jazigo](http://localhost:8080/jazigo)

Global Settings
===============

You might want to adjust global settings. See the Jazigo *admin* window under [http://localhost:8080/jazigo/admin](http://localhost:8080/jazigo/admin).

    maxconfigfiles: 120
    holdtime: 12h0m0s
    scaninterval: 10m0s
    maxconcurrency: 20

**maxconfigfiles**: This option limits the amount of files stored per device. When this limit is reached, older files are discarded.

**holdtime**: When a successful backup is saved for a device, the software will only contact that specific device again *after* expiration of the 'holdtime' timer.

**scaninterval**: The interval between two device table scans. If the device table is fully processed before the 'scaninterval' timer, the software will wait idly for the next scan cycle. If the full table scan takes longer than 'scaninterval', the next cycle will start immediately.

**maxconcurrency**: This option limits the number of concurrent backup jobs. You should raise this value if you need faster scanning of all devices. Keep in mind that if your devices use a centralized authentication system (for example, Cisco Secure ACS), the authentication server might become a bottleneck for high concurrency.
