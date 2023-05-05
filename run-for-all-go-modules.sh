#!/bin/sh -e
SELF_DIR="$(dirname "$(readlink -f "$0")")"


for GO_MOD_DIR in $(find . -name 'go.mod' -exec dirname {} \; ) ; do 
    cd "${SELF_DIR}"
    cd "${GO_MOD_DIR}"

    echo
    echo
    echo "Running commands for Go module ${GO_MOD_DIR}..."
    echo "============================================================="
    echo
    sh -e -c "$*"
done