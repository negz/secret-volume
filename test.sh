#!/usr/bin/env sh

set -e
echo "" > coverage.txt

for d in $(glide novendor); do
    go test -v -tags 'debug integration' -race -coverprofile=profile.out $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
