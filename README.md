[![GoDoc](https://godoc.org/github.com/udhos/jazigo/jazigo?status.svg)](http://godoc.org/github.com/udhos/jazigo/jazigo)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/jazigo)](https://goreportcard.com/report/github.com/udhos/jazigo)
[![Build Status](https://travis-ci.org/udhos/jazigo.svg?branch=master)](https://travis-ci.org/udhos/jazigo)
[![gocover dev](http://gocover.io/_badge/github.com/udhos/jazigo/dev)](http://gocover.io/github.com/udhos/jazigo/dev)
[![gocover store](http://gocover.io/_badge/github.com/udhos/jazigo/store)](http://gocover.io/github.com/udhos/jazigo/store)

About Jazigo
=============

Jazigo is a tool written in Go for retrieving configuration for multiple devices, similar to [rancid](http://www.shrubbery.net/rancid/), [fetchconfig](https://github.com/udhos/fetchconfig), [oxidized](https://github.com/ytti/oxidized), [Sweet](https://github.com/AppliedTrust/sweet).

Installation and usage are supposed to be dead simple. If you hit any surprising difficulty, please [report](https://github.com/udhos/jazigo/issues/new).

Supported Platforms
===================

Please send pull requests for new plataforms.

- [Cisco IOS](https://github.com/udhos/jazigo/blob/master/dev/model_cisco.go)
- [Cisco IOS XR](https://github.com/udhos/jazigo/blob/master/dev/model_cisco_iosxr.go)
- [Cisco ACI APIC](https://github.com/udhos/jazigo/blob/master/dev/model_cisco_apic.go)
- [Juniper JunOS](https://github.com/udhos/jazigo/blob/master/dev/model_junos.go)
- [Mikrotik](https://github.com/udhos/jazigo/blob/master/dev/model_mikrotik.go)
- [HTTP](https://github.com/udhos/jazigo/blob/master/dev/model_http.go) (collect output of http GET method)
- [Linux](https://github.com/udhos/jazigo/blob/master/dev/model_linux.go) (collect output of SSH commands)
- [Run](https://github.com/udhos/jazigo/blob/master/dev/model_run.go) (run external program and collect its output)

Features
========

- Written in [Go](https://golang.org/). Single executable file. No runtime dependency.
- Straightforward usage: run the binary then point browser to web UI. Default settings should work out-of-the-box.
- Tool configuration is automatically saved as [YAML](http://yaml.org). However one is NOT supposed to edit configuration file directly.
- Spawns multiple concurrent lightweight goroutines to quickly handle large number of devices.
- Very easy to add support for new platforms. See the [Cisco IOS model](https://github.com/udhos/jazigo/blob/master/dev/model_cisco.go) as example.
- Backup files can be accessed from web UI.
- Support for SSH and TELNET.
- Can directly store backup files into AWS S3 bucket.
- Can call an external program and collect its output.

Requirements
============

- You need a [system with the Go language](https://golang.org/dl/) in order to build the application. There is no special requirement for running it.

Quick Start
===========

Installation and usage are supposed to be dead simple. If you hit any surprising difficulty, please [report](https://github.com/udhos/jazigo/issues/new).

1\. Setup GOPATH as usual

Example:

    export GOPATH=~/go
    mkdir $GOPATH

2\. Get dependencies

    go get github.com/icza/gowut/gwu
    go get github.com/udhos/lockfile
    go get github.com/udhos/equalfile
    go get gopkg.in/yaml.v2
    go get golang.org/x/crypto/ssh
    go get github.com/aws/aws-sdk-go

3\. Get source code

`go get github.com/udhos/jazigo`

4\. Compile and install

`go install github.com/udhos/jazigo/jazigo`

5\. Decide where to store config, backup and log files

Example:

    export JAZIGO_HOME=$HOME/jazigo
    mkdir -p $JAZIGO_HOME/etc $JAZIGO_HOME/repo $JAZIGO_HOME/log

Hint: See command line options to fine tune filesystem locations.

6\. Run jazigo once (see -runOnce option)

`$GOPATH/bin/jazigo -runOnce`

Watch messages logged to standard output for errors.

Hint: Since root privileges are usually not needed, run Jazigo as a regular user.

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

Importing Many Devices
======================

You can use the Web UI to add devices, but it is not designed for importing a large number of devices.

The easiest way to include many devices is by using the command line option **-deviceImport**.

1\. Build a device table using this format:

    $ cat table.txt
    #
    # model   id   hostport      transports username password enable-password
    #
    cisco-ios lab1 router1905lab telnet     san      fran     sanjose
    cisco-ios lab2 router3925lab telnet     san      fran     sanjose
    junos     auto ex4200lab     ssh,telnet backup   juniper1 not-used
    junos     auto 1.1.1.1:2222  ssh        backup   juniper1 not-used
    $

Hint: The device id must be unique. You can generate a meaningful device id manually as you like. You can also let Jazigo create id's automatically by specifying the special id **auto**.

2\. Then load the table with the option -deviceImport:

    $ $GOPATH/bin/jazigo -deviceImport < table.txt

Using AWS S3
============

Quick recipe for using S3 bucket:

1\. Create a bucket 'bucketname' on AWS region 'regionname'.

2\. Authorize the client to access the bucket

An usual way is to create an IAM user, add key/secret, and put those credentials into ~/.aws/credentials:

    $ cat ~/.aws/credentials
    [default]
    aws_access_key_id = key
    aws_secret_access_key = secret

3\. Run jazigo pointing its config and repository paths to S3 bucket ARN:

**S3 bucket ARN**: arn:aws:s3:regionname::bucketname/foldername

    # Example
    ARN=arn:aws:s3:regionname::bucketname/foldername
    $GOPATH/bin/jazigo -configPathPrefix=$ARN/etc/jazigo.conf. -repositoryPath=$ARN/repo

Hint: You could point config and repository to distinct buckets.

Calling an external program
===========================

You can use the pseudo model **run** to call an external program to collect custom configuration.

Create a device using the model **run**, then specify the program arguments in the attribute **runprog**:

Example:

    # This example calls: /bin/bash -c "env | egrep ^JAZIGO_"
    runprog:
    - /bin/bash
    - -c
    - env | egrep ^JAZIGO_

The external program invoked by the model **run** will receive its device authentication credentials as environment variables:

    JAZIGO_DEV_ID=deviceid
    JAZIGO_DEV_HOSTPORT=host[:port] -- port is optional
    JAZIGO_DEV_USER=username
    JAZIGO_DEV_PASS=password

The external program is expected to issue captured configuration to stdout and then to exit with zero exit status.
