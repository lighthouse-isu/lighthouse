#!/bin/bash

list="$(find -path '.*test.go')"

echo "mode: count\n" > profile.out

for d in $list; do
	path=$(echo $d | awk -F/ '{for (i=1; i<NF; i++) printf $i"/"}')


	go test $path -v -coverprofile=cover.out -covermode=count
    tail -n +2 cover.out >> profile.out
done

$HOME/gopath/bin/goveralls -coverprofile=profile.cov -service=travis-ci