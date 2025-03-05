#!/bin/sh

if [ $# -eq 0 ]; then
  echo "Usage: edit.sh <app>"
  exit 1
fi

exec code "$SMALLWEB_DIR/$1"
