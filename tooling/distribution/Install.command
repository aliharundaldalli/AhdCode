#!/bin/sh
set -eu
cd -- "$(dirname -- "$0")"
printf 'AhdCode setup installs into your user Library and adds one owned PATH block to .zprofile.\nPress Enter to install, or Control-C to cancel.\n'
read -r answer
exec sh ./install.sh --setup-path
