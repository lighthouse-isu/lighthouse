#!/bin/bash

list="$(find -path '.*test.go')"

base=$(pwd)

for d in $list; do
	path=$(echo $d | awk -F/ '{for (i=1; i<NF; i++) printf $i"/"}')

	cd $path

	$HOME/gopath/bin/goveralls -service=travis-ci

	cd $base
done