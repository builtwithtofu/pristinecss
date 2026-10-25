#!/bin/sh
# Fetch pinned archives; capgen checks the extracted data.json hashes.
set -eu
mkdir -p "$1"
cd "$1"

fetch() {
    archive=$1 url=$2 hash=$3 folder=$4
    [ -f "$archive" ] || curl -fL "$url" -o "$archive"
    printf '%s  %s\n' "$hash" "$archive" | shasum -a 256 -c -
    mkdir -p "$folder"
    tar -xzf "$archive" -C "$folder"
}

fetch web-features-3.44.0.tgz \
    https://registry.npmjs.org/web-features/-/web-features-3.44.0.tgz \
    24a8b4bdbf043ba386e2e9a309cd036547c530b532b097c653efff49d65106ed \
    web-features-3.44.0
fetch mdn-browser-compat-data-8.1.7.tgz \
    https://registry.npmjs.org/@mdn/browser-compat-data/-/browser-compat-data-8.1.7.tgz \
    4e7a8747cac26a68087df5f38405f834a352245283af1128f4cf189989bd691e \
    browser-compat-data-8.1.7
