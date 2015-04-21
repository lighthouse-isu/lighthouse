#!/bin/bash

set -e

list="$(find -path '.*.go' | sort | uniq -u)"

echo "mode: count\n" > profile.cov

last=""

for d in $list; do
	path=$(echo $d | awk -F/ '{for (i=1; i<NF; i++) printf $i"/"}')

	if [[ $last != $path ]]
	then
		go test $path -v -coverprofile=cover.out -covermode=count
	    tail -n +2 cover.out >> profile.cov
	fi

	last=$path
done

cat profile.cov | awk -F/ '!match($NF, "_testing_utils.go")' > no_utils.cov

if [[ $1 == "local" ]]
then
	go tool cover -html=no_utils.cov -o coverage.html
	rm profile.cov cover.out no_utils.cov
else
	$HOME/gopath/bin/goveralls -coverprofile=no_utils.cov -service=travis-ci
fi

toFmt=$(gofmt -l .)
if [[ $toFmt != "" ]]
then
	echo "Unformatted files:"
	for file in $toFmt 
	do
		echo $file
	done
	exit 1
fi