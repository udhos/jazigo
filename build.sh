#!/bin/sh

step=0

msg() {
    step=$((step+1))
    echo >&2 $step. $*
}

get() {
    i=$1
    msg fetching $i
    go get $i
    msg fetching $i - done
}

get github.com/icza/gowut/gwu
get github.com/udhos/lockfile
get github.com/udhos/equalfile
get gopkg.in/yaml.v2
get golang.org/x/crypto/ssh
get github.com/aws/aws-sdk-go
#get honnef.co/go/simple/cmd/gosimple

src=`find . -type f | egrep '\.go$'`

msg fmt
gofmt -s -w $src
msg fix
go tool fix $src
msg vet
go tool vet .

msg install
pkg=github.com/udhos/jazigo
go install $pkg/jazigo

# go get honnef.co/go/simple/cmd/gosimple
s=$GOPATH/bin/gosimple
simple() {
    msg simple - this is slow, please standby
    # gosimple cant handle source files from multiple packages
    $s jazigo/*.go
    $s conf/*.go
    $s dev/*.go
    $s store/*.go
    $s temp/*.go
}
[ -x "$s" ] && simple

# go get github.com/golang/lint/golint
l=$GOPATH/bin/golint
lint() {
    msg lint
    # golint cant handle source files from multiple packages
    $l jazigo/*.go
    $l conf/*.go
    $l dev/*.go
    $l store/*.go
    $l temp/*.go
}
[ -x "$l" ] && lint

msg test dev
go test github.com/udhos/jazigo/dev

msg test store
if [ -z "$JAZIGO_S3_REGION" ]; then
    echo >&2 JAZIGO_S3_REGION undefined -- set JAZIGO_S3_REGION=region
fi
if [ -z "$JAZIGO_S3_FOLDER" ]; then
    echo >&2 JAZIGO_S3_FOLDER undefined -- set JAZIGO_S3_FOLDER=bucket/folder
fi
go test github.com/udhos/jazigo/store
