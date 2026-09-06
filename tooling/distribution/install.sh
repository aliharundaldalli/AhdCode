#!/bin/sh
set -eu
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
version=$(cat "$source_dir/VERSION")
case "$version" in *[!0-9A-Za-z.-]*|'') echo 'Invalid version' >&2; exit 1;; esac
case $(uname -s) in Darwin) default_prefix="$HOME/Library/Application Support/AhdCode";; *) default_prefix="$HOME/.local/share/ahdcode";; esac
prefix=$default_prefix
setup_path=false
uninstall=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --prefix) [ "$#" -ge 2 ] || { echo '--prefix needs an absolute installation root.' >&2; exit 2; }; prefix=$2; shift 2;;
    --setup-path) setup_path=true; shift;;
    --uninstall) uninstall=true; shift;;
    *) echo "Unknown installer option: $1" >&2; exit 2;;
  esac
done
case "$prefix" in /*) ;; *) echo 'Installation prefix must be absolute.' >&2; exit 2;; esac
case "$prefix" in /|/usr|/usr/local|/opt|"$HOME") echo 'Unsafe installation prefix.' >&2; exit 2;; esac
if [ -L "$prefix" ]; then echo 'Installation root must not be a symlink.' >&2; exit 1; fi
if [ -e "$prefix" ] && [ ! -f "$prefix/.ahdcode-install" ]; then
  echo 'Refusing to change an existing directory not owned by this installer.' >&2; exit 1
fi
if "$uninstall"; then
  [ -f "$prefix/.ahdcode-install" ] || { echo 'No owned installation found.' >&2; exit 1; }
  if [ "$prefix" = "$default_prefix" ]; then
    for profile in "$HOME/.zprofile" "$HOME/.profile"; do
      if [ -f "$profile" ] && [ ! -L "$profile" ]; then
        temporary=$(mktemp "${profile}.ahdcode.XXXXXX")
        awk '/^# >>> AhdCode PATH >>>$/{skip=1;next} /^# <<< AhdCode PATH <<<$/{skip=0;next} !skip' "$profile" > "$temporary"
        cat "$temporary" > "$profile"; rm "$temporary"
      fi
    done
  fi
  rm -rf -- "$prefix"
  echo 'AhdCode removed. Projects, databases, registry, and caches were preserved.'
  exit 0
fi
mkdir -p "$prefix/versions" "$prefix/bin"
printf 'AhdCode installer v1\n' > "$prefix/.ahdcode-install"
target="$prefix/versions/$version"
if [ -e "$target" ]; then echo "Version $version is already installed; no files changed."; exit 0; fi
staging=$(mktemp -d "$prefix/versions/.install.XXXXXX")
trap 'rm -rf -- "$staging"' EXIT HUP INT TERM
cp -R "$source_dir/payload/." "$staging/"
cp "$source_dir/install.sh" "$staging/install.sh"
printf '%s\n' "$version" > "$staging/VERSION"
"$staging/bin/ahdcode" --version
mv "$staging" "$target"
ln -s "versions/$version" "$prefix/.current-new"
case $(uname -s) in Darwin) mv -fh "$prefix/.current-new" "$prefix/current";; *) mv -Tf "$prefix/.current-new" "$prefix/current";; esac
ln -sfn ../current/bin/ahdcode "$prefix/bin/ahdcode"
if "$setup_path"; then
  [ "$prefix" = "$default_prefix" ] || { echo '--setup-path is supported only for the default installation root.' >&2; exit 2; }
  case $(uname -s) in Darwin) profile="$HOME/.zprofile"; path_line='export PATH="$HOME/Library/Application Support/AhdCode/bin:$PATH"';; *) profile="$HOME/.profile"; path_line='export PATH="$HOME/.local/share/ahdcode/bin:$PATH"';; esac
  [ ! -L "$profile" ] || { echo 'Refusing to edit a symlinked shell profile.' >&2; exit 1; }
  touch "$profile"
  if ! grep -q '^# >>> AhdCode PATH >>>$' "$profile"; then
    printf '\n# >>> AhdCode PATH >>>\n%s\n# <<< AhdCode PATH <<<\n' "$path_line" >> "$profile"
  fi
fi
printf 'Installed %s\nCLI: %s/bin/ahdcode\nOpen a new shell after PATH setup.\n' "$version" "$prefix"
printf 'Uninstall: sh "%s/current/install.sh" --prefix "%s" --uninstall\n' "$prefix" "$prefix"
