#!/bin/sh

get() {
    i=$1
    echo 2>&1 fetching $i
    go get $i
}

get github.com/icza/gowut/gwu
get github.com/udhos/lockfile
get github.com/udhos/equalfile
get gopkg.in/yaml.v2
get golang.org/x/crypto/ssh
get github.com/aws/aws-sdk-go
get honnef.co/go/simple/cmd/gosimple

src=`find . -type f | egrep '\.go$'`

gofmt -s -w $src
go tool fix $src
go tool vet .
pkg=github.com/udhos/jazigo
go install $pkg/jazigo

# go get honnef.co/go/simple/cmd/gosimple
s=$GOPATH/bin/gosimple
simple() {
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
    # golint cant handle source files from multiple packages
    $l jazigo/*.go
    $l conf/*.go
    $l dev/*.go
    $l store/*.go
    $l temp/*.go
}
[ -x "$l" ] && lint

go test github.com/udhos/jazigo/dev

if [ -z "$JAZIGO_S3_REGION" ]; then
    echo >&2 JAZIGO_S3_REGION undefined -- set JAZIGO_S3_REGION=region
    exit 1
fi
if [ -z "$JAZIGO_S3_FOLDER" ]; then
    echo >&2 JAZIGO_S3_FOLDER undefined -- set JAZIGO_S3_FOLDER=bucket/folder
    exit 1
fi

go test github.com/udhos/jazigo/store
