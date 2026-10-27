#!/bin/sh
# page-history.sh — DEPRECATED since v0.21.0, kept so that existing sites build.
#
# The generator is a Go program now (cmd/pagehistory): same JSON, byte for byte,
# but it runs git with an explicit argument vector, is bounded, and has tests.
# Call it directly:
#
#     go run github.com/tannevaled/hextra/cmd/pagehistory@v0.21.0 > data/pagehistory.json
#
# This shim runs that same program from the theme's own checkout (it needs a Go
# toolchain). Run it from the site's root:
#
#     sh themes/hextra/scripts/page-history.sh > data/pagehistory.json
#
# It will be removed in the next MINOR release.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
echo "page-history.sh: deprecated, use 'go run github.com/tannevaled/hextra/cmd/pagehistory@<tag>' (see the theme's docs, Page History)" >&2
exec go run -C "$here/../cmd/pagehistory" . --root "$PWD" "$@"
