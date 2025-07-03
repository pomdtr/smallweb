#!/bin/sh

# handle -h and --help options
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
  echo "Usage: edit.sh <app>"
  echo "Edit the source code of the specified app in the Smallweb directory."
  echo "Example: edit.sh myapp"
  exit 0
fi

if [ $# -eq 0 ]; then
  echo "Usage: edit.sh <app>"
  exit 1
fi

exec code "$SMALLWEB_DIR/$1"
