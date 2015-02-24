#!/bin/bash

list="$(find -path '.*test.go')"

for d in $list; do
	path=$(echo $d | awk -F/ '{for (i=1; i<NF; i++) printf $i"/"}')

	go test -v -coverprofile=cover.out -covermode=count $path

	$HOME/gopath/bin/goveralls -coverprofile=cover.out -service=travis-ci
done