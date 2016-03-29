#!/bin/sh

gofmt -s -w *.go
go tool vet *.go
go tool fix *.go
go install github.com/udhos/jazigo
go test
