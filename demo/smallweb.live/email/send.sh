#!/bin/sh

curl --url "smtp://localhost:2525" \
     --mail-from "sender@example.com" \
     --mail-rcpt "email@smallweb.localhost" \
     --upload-file email.txt
