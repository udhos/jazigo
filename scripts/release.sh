#!/bin/bash

list() {
    cat <<EOF
darwin amd64
freebsd 386
freebsd amd64
freebsd arm
linux 386
linux amd64
linux arm
linux mips
linux mips64
linux mips64le
linux mipsle
linux s390x
netbsd 386
netbsd amd64
netbsd arm
openbsd 386
openbsd amd64
windows 386
windows amd64
EOF
}

app=jazigo

# const appVersion = "0.13.0"
version=$(grep "const appVersion" jazigo/main.go | awk '{print $4}' | tr -d '"')

mkdir -p tmp
rm -f tmp/*

go env -w CGO_ENABLED=0

list | while read i; do
    set -- $i
    os=$1
    arch=$2

    go env -w GOOS=$os
    go env -w GOARCH=$arch

    extension=''
    [ $os == windows ] && extension=.exe

    output=tmp/${app}_${os}_${arch}_${version}${extension}
    echo output=$output

    #go build -tags netgo,osusergo -o $output ./cmd/sqs-to-sns
    go build -o $output ./$app
done

go env -u GOOS
go env -u GOARCH
go env -u CGO_ENABLED
