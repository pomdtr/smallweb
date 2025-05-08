#!/bin/bash
set -e

# Get current UID/GID from environment or use defaults
USER_ID=${PUID:-$(stat -c '%u' "$SMALLWEB_DIR")}
GROUP_ID=${PGID:-$(stat -c '%g' "$SMALLWEB_DIR")}

if [ "$USER_ID" = "0" ]; then
    exec /usr/local/bin/smallweb "$@"
fi

if [ "$(id -u smallweb)" != "$USER_ID" ]; then
  echo "Updating user 'smallweb' with new UID -> $USER_ID"
  usermod -u "$USER_ID" smallweb
fi

if [ "$(id -g smallweb)" != "$GROUP_ID" ]; then
  echo "Updating group 'smallweb' with new GID -> $GROUP_ID"
  groupmod -g "$GROUP_ID" smallweb
fi

# Execute the command as the smallweb user
exec gosu smallweb:smallweb /usr/local/bin/smallweb "$@"
