#!/bin/bash -e

# check if .ssh directory exists
if [ ! -d "/home/${USERNAME}/.ssh" ]; then
  mkdir -p "/home/${USERNAME}/.ssh"
  chmod 700 "/home/${USERNAME}/.ssh"
  echo "$AUTHORIZED_KEYS" > "/home/${USERNAME}/.ssh/authorized_keys"
fi

# persist ssh keys between container restarts
mkdir -p .ssh_keys
if [[ "$(ls .ssh_keys/*_key)" = "" ]]; then
  sudo cp /etc/ssh/*_key .ssh_keys
else
  sudo cp .ssh_keys/*_key /etc/ssh
fi

if [ ! -f "/home/${USERNAME}/.config/smallweb/config.json" ]; then
  mkdir -p "/home/${USERNAME}/.config/smallweb"
  cat <<EOF > "/home/${USERNAME}/.config/smallweb/config.json"
{
  "domains": {
    "$FLY_APP_NAME.fly.dev": "~/$FLY_APP_NAME.fly.dev"
  }
}
EOF
    mkdir -p "/home/${USERNAME}/$FLY_APP_NAME.fly.dev"
    echo "Hello, World!" > "/home/${USERNAME}/$FLY_APP_NAME.fly.dev/index.html"
fi

sudo /usr/sbin/sshd

exec "$@"
