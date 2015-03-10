#!/bin/bash

set -e

list="$(find -path '.*test.go')"

echo "mode: count\n" > profile.cov

for d in $list; do
	path=$(echo $d | awk -F/ '{for (i=1; i<NF; i++) printf $i"/"}')

	go test $path -v -coverprofile=cover.out -covermode=count
    tail -n +2 cover.out >> profile.cov
done

cat profile.cov | awk -F/ '!match($NF, "_testing_utils.go")' > no_utils.cov

if [[ $1 == "local" ]]
then
	go tool cover -html=no_utils.cov -o coverage.html
	rm profile.cov cover.out no_utils.cov
else
	$HOME/gopath/bin/goveralls -coverprofile=no_utils.cov -service=travis-ci
fi
