[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/jazigo/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/jazigo)](https://goreportcard.com/report/github.com/udhos/jazigo)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/jazigo.svg)](https://pkg.go.dev/github.com/udhos/jazigo)

Table of Contents
=================

* [About Jazigo](#about-jazigo)
* [Supported Platforms](#supported-platforms)
* [Features](#features)
* [Requirements](#requirements)
* [Quick Start \- Short version](#quick-start---short-version)
* [Quick Start \- Detailed version](#quick-start---detailed-version)
* [Global Settings](#global-settings)
* [Importing Many Devices](#importing-many-devices)
* [SSH Ciphers](#ssh-ciphers)
* [Using AWS S3](#using-aws-s3)
* [Calling an external program](#calling-an-external-program)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

About Jazigo
=============

Jazigo is a tool written in Go for retrieving configuration for multiple devices, similar to [rancid](http://www.shrubbery.net/rancid/), [fetchconfig](https://github.com/udhos/fetchconfig), [oxidized](https://github.com/ytti/oxidized), [Sweet](https://github.com/AppliedTrust/sweet).

Installation and usage are supposed to be dead simple. If you hit any surprising difficulty, please [report](https://github.com/udhos/jazigo/issues/new).

Supported Platforms
===================

Please send pull requests for new plataforms.

- [Cisco ACI APIC](https://github.com/udhos/jazigo/blob/master/dev/model_cisco_apic.go)
- [Cisco IOS](https://github.com/udhos/jazigo/blob/master/dev/model_cisco.go)
- [Cisco IOS XR](https://github.com/udhos/jazigo/blob/master/dev/model_cisco_iosxr.go)
- [Cisco NGA](https://github.com/udhos/jazigo/blob/master/dev/model_cisco_nga.go)
- [Datacom DmSwitch](https://github.com/udhos/jazigo/blob/master/dev/model_datacom_dmswitch.go)
- [Fortigate FortiOS](https://github.com/udhos/jazigo/blob/master/dev/model_fortios.go)
- [HTTP](https://github.com/udhos/jazigo/blob/master/dev/model_http.go) (collect output of http GET method)
- [Huawei VRP](https://github.com/udhos/jazigo/blob/master/dev/model_huawei_vrp.go)
- [Juniper JunOS](https://github.com/udhos/jazigo/blob/master/dev/model_junos.go)
- [Linux](https://github.com/udhos/jazigo/blob/master/dev/model_lin.go) (collect output of SSH commands)
- [Mikrotik](https://github.com/udhos/jazigo/blob/master/dev/model_mikrotik.go)
- [Run](https://github.com/udhos/jazigo/blob/master/dev/model_run.go) (run external program and collect its output)

Features
========

- Written in [Go](https://golang.org/). Single executable file. No runtime dependency.
- Straightforward usage: run the binary then point browser to web UI. Default settings should work out-of-the-box.
- Tool configuration is automatically saved as [YAML](http://yaml.org). However one is NOT supposed to edit configuration file directly.
- Spawns multiple concurrent lightweight goroutines to quickly handle large number of devices.
- Very easy to add support for new platforms. See the [Cisco IOS model](https://github.com/udhos/jazigo/blob/master/dev/model_cisco.go) as example.
- Backup files can be accessed from web UI.
- See file differences directly from the web UI.
- Support for SSH and TELNET.
- Can directly store backup files into AWS S3 bucket.
- Can call an external program and collect its output.

Requirements
============

- You need a [system with the Go language](https://golang.org/dl/) in order to build the application. There is no special requirement for running it.

Quick Start - Short version
===========================

This is how to boot up Jazigo very quickly:

    git clone https://github.com/udhos/jazigo ;# clone outside of GOPATH
    cd jazigo
    go install ./jazigo
    mkdir etc repo log
    JAZIGO_HOME=$PWD ~/go/bin/jazigo

Open jazigo interface - http://localhost:8080/jazigo/

Quick Start - Detailed version
==============================

Installation and usage are supposed to be dead simple. If you hit any surprising difficulty, please [report](https://github.com/udhos/jazigo/issues/new).

If you want to build from source code, start from step 1.

If you downloaded the executable binary file, start from step 2.

1\. Build from source

    git clone https://github.com/udhos/jazigo
    cd jazigo
    go install ./...

2\. Decide where to store config, backup, log and static www files

Example:

    export JAZIGO_HOME=$HOME/jazigo
    mkdir $JAZIGO_HOME
    cd $JAZIGO_HOME
    mkdir etc repo log www

Hint:
By default, Jazigo looks for directories 'etc', 'repo', 'log', and 'www' under $JAZIGO_HOME.
If left undefined, JAZIGO_HOME defaults to /var/jazigo.
See command line options to fine tune filesystem locations.

3\. Copy static files (CSS and images) to $JAZIGO_HOME/www

Example:

    # If you have downloaded jazigo using 'go get':
    cp ~/go/src/github.com/udhos/jazigo/www/* $JAZIGO_HOME/www

    # Otherwise get static files from https://github.com/udhos/jazigo/tree/master/www
    cd $JAZIGO_HOME/www
    wget https://raw.githubusercontent.com/udhos/jazigo/master/www/fail-small.png
    wget https://raw.githubusercontent.com/udhos/jazigo/master/www/ok-small.png
    wget https://raw.githubusercontent.com/udhos/jazigo/master/www/jazigo.css
    wget https://raw.githubusercontent.com/udhos/jazigo/master/www/GitHub-Mark-32px.png

4\. Run jazigo once (see -runOnce option)

`~/go/bin/jazigo -runOnce`

Watch messages logged to standard output for errors.

Hint: Since root privileges are usually not needed, run Jazigo as a regular user.

5\. Run jazigo forever

`~/go/bin/jazigo -disableStdoutLog`

6\. Open the web interface

Point web browser at: [http://localhost:8080/jazigo](http://localhost:8080/jazigo)
      
Global Settings
===============

You might want to adjust global settings. See the Jazigo *admin* window under [http://localhost:8080/jazigo/admin](http://localhost:8080/jazigo/admin).

    maxconfigfiles: 120
    holdtime: 12h0m0s
    scaninterval: 10m0s
    maxconcurrency: 20
    maxconfigloadsize: 10000000

**maxconfigfiles**: This option limits the amount of files stored per device. When this limit is reached, older files are discarded.

**holdtime**: When a successful backup is saved for a device, the software will only contact that specific device again *after* expiration of the 'holdtime' timer.

**scaninterval**: The interval between two device table scans. If the device table is fully processed before the 'scaninterval' timer, the software will wait idly for the next scan cycle. If the full table scan takes longer than 'scaninterval', the next cycle will start immediately.

**maxconcurrency**: This option limits the number of concurrent backup jobs. You should raise this value if you need faster scanning of all devices. Keep in mind that if your devices use a centralized authentication system (for example, Cisco Secure ACS), the authentication server might become a bottleneck for high concurrency.

**maxconfigloadsize**: This limit puts restriction into the amount of data the tool loads from a file to memory. Intent is to protect the servers' memory from exhaustion while trying to handle multiple very large configuration files.

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

    $ ~/go/bin/jazigo -deviceImport < table.txt

SSH Ciphers
===========

You can control ciphers for the SSH transport by editing these device properties:

* sshclearciphers: if enabled remove all default ciphers.
* sshaddciphers: list of ciphers to add.

Example:

    sshclearciphers: true # remove all default ciphers
    sshaddciphers:
        - aes128-ctr      # add cipher aes128-ctr

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
    ~/go/bin/jazigo -configPathPrefix=$ARN/etc/jazigo.conf. -repositoryPath=$ARN/repo

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
