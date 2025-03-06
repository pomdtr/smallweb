#!/bin/bash
set -e

# Get current UID/GID from environment or use defaults
USER_ID=${PUID:-1000}
GROUP_ID=${PGID:-1000}

echo "Starting with UID: $USER_ID, GID: $GROUP_ID"

# Update the user to match desired UID/GID if needed
if [ "$USER_ID" != "1000" ] || [ "$GROUP_ID" != "1000" ]; then
  echo "Updating user 'smallweb' with new UID:GID -> $USER_ID:$GROUP_ID"
  groupmod -g "$GROUP_ID" smallweb
  usermod -u "$USER_ID" -g "$GROUP_ID" smallweb
fi

# Ensure correct ownership of the application directory
chown -R smallweb:smallweb /smallweb

# Execute the command as the smallweb user
exec gosu smallweb:smallweb /usr/local/bin/smallweb "$@"
