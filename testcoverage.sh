#!/bin/bash

list="$(find -mindepth 1 -path './[^.]*' -type d)"

rm -f coverage.html
echo "mode: count\n" > test_coverage.out

for d in $list; do
    go test -coverprofile=cover.out -covermode=count $d
    tail -n +2 cover.out >> test_coverage.out
done

go tool cover -html=test_coverage.out -o coverage.html

rm test_coverage.out cover.out
