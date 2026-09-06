#!/bin/sh
# Kept for the ZIP and for anyone who opens the DMG's script directly. The
# normal macOS experience is the .pkg, which needs no terminal at all.
set -eu
cd -- "$(dirname -- "$0")"
exec sh ./install.sh --setup-path
