#!/bin/sh

curl smtp://localhost:2525 \
  --mail-from 'myself@example.com' \
  --mail-rcpt 'inbox@smallweb.localhost' \
  --upload-file email.eml
