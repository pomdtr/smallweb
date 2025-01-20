#!/bin/sh

curl -v smtp://localhost:2525 \
  --mail-from 'me@localhost' \
  --mail-rcpt 'inbox@smallweb.localhost' \
  --upload-file email.eml
