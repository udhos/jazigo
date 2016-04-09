About Jazigo
=============

Jazigo is a tool written in Go for retrieving configuration for multiple devices, similar to rancid and oxidized.

Quick Start
===========

1\. Setup GOPATH as usual

Example:

    export GOPATH=~/go
    mkdir $GOPATH

2\. Get source code

`go get github.com/udhos/jazigo`

3\. Compile and install

`go install github.com/udhos/jazigo`

4\. Decide where to store config and backup files

Example:

    # These env vars are not meaningful to jazigo.
    # They're just handy pointers used in step 5 below.
    export APP_ETC=/etc/jazigo  ;# app config
    export APP_REPO=/var/jazigo ;# backup repository
    mkdir $APP_ETC $APP_REPO

5\. Run jazigo

`$GOPATH/bin/jazigo -configPathPrefix $APP_ETC -repositoryPath $APP_REPO`

6\. Look at the web interface

Point web browser at: [http://localhost:8080/jazigo](http://localhost:8080/jazigo)
