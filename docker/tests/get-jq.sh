#!/bin/bash

# https://github.com/jqlang/jq/releases/tag/jq-1.7.1
DOWNLOAD_FILE="jq-linux-amd64"

ARCH=$(uname -m)

case "$ARCH" in
"arm64" | "aarch64")
    DOWNLOAD_FILE="jq-linux-arm64"
    ;;
"x86_64")
    DOWNLOAD_FILE="jq-linux-amd64"
    ;;
*)
    echo "get-jq failed: unsupported arch - $ARCH"
    exit 1
    ;;
esac

curl \
    -L \
    --output jq \
    "https://github.com/jqlang/jq/releases/download/jq-1.7.1/$DOWNLOAD_FILE"

chmod +x ./jq

echo '{"got": "jq"}' | ./jq
