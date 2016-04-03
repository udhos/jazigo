#!/bin/sh

src=`find . -type f | egrep '\.go$'`

gofmt -s -w $src
go tool vet $src
go tool fix $src
go install github.com/udhos/jazigo

# go get honnef.co/go/simple/cmd/gosimple
s=$GOPATH/bin/gosimple
simple() {
    # gosimple cant handle source files from multiple packages
    $s *.go
    $s conf/*.go
    $s dev/*.go
}
[ -x "$s" ] && simple

#go test
go test github.com/udhos/jazigo/dev
