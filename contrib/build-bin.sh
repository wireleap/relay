#!/bin/sh
set -e

SCRIPT_NAME="$(basename "$0")"

fatal() { echo "FATAL [$SCRIPT_NAME]: $*" 1>&2; exit 1; }
info() { echo "INFO [$SCRIPT_NAME]: $*"; }

usage() {
cat<<EOF
Syntax: $SCRIPT_NAME /path/to/outdir
Helper script to compile Wireleap components

EOF
exit 1
}

[ -n "$1" ] || usage

command -v go >/dev/null || fatal "go not installed"
command -v make >/dev/null || fatal "make not installed"

OUTDIR="$(realpath "$1")"
[ -d "$OUTDIR" ] || mkdir -p "$OUTDIR"

SRCDIR="$(dirname "$(dirname "$(realpath "$0")")")"
GITVERSION="$($SRCDIR/contrib/gitversion.sh)"

info "building ..."
CGO_ENABLED=0 go build -tags "$BUILD_TAGS" -o "$OUTDIR/wireleap-relay" -ldflags "
    -X github.com/wireleap/relay/version.GITREV=$GITVERSION \
    -X github.com/wireleap/common/wlnet.PROTO_VERSION_STRING=$GITVERSION \
    -X github.com/wireleap/common/api/apiversion.VERSION_STRING=$GITVERSION
"

[ -z "$BUILD_USER" ] || chown -R "$BUILD_USER" "$OUTDIR"
[ -z "$BUILD_GROUP" ] || chgrp -R "$BUILD_GROUP" "$OUTDIR"

# defined in contrib/docker/build-bin.sh, change here if changed there
DEPSDIR=/go/deps
if [ -d "$DEPSDIR" ]; then
    [ -z "$BUILD_USER" ] || chown -R "$BUILD_USER" "$DEPSDIR"
    [ -z "$BUILD_GROUP" ] || chgrp -R "$BUILD_GROUP" "$DEPSDIR"
fi
