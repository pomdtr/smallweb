#!/bin/bash -e

mkdir -p "/home/${USERNAME}/.ssh"
echo "$AUTHORIZED_KEYS" > "/home/${USERNAME}/.ssh/authorized_keys"
sudo /usr/sbin/sshd

exec "$@"
