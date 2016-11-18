#!/bin/sh

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
go test github.com/udhos/jazigo/store
