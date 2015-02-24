#!/bin/bash

list="$(find -path '.*test.go')"

echo "mode: count\n" > test_coverage.out

for d in $list; do
	path=$(echo $d | awk -F/ '{for (i=1; i<NF; i++) printf $i"/"}')

	go test -v -coverprofile=cover.out -covermode=count $path
	tail -n +2 cover.out >> test_coverage.out
done

$HOME/gopath/bin/goveralls -coverprofile=test_coverage.out -service=travis-ci