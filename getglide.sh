#!/usr/bin/env bash

set -e

TARBALL="/tmp/glide.tar.gz"
VERSION=$(curl -fsSL "https://glide.sh/version")
URL="https://github.com/Masterminds/glide/releases/download/${VERSION}/glide-${VERSION}-linux-amd64.tar.gz"

curl -fsSL $URL -o $TARBALL
tar --strip-components=1 -C "${GOPATH}/bin" -zxvf $TARBALL
rm $TARBALL
