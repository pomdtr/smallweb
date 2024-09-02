# Syncing Files

Smallweb automatically starts a webdav server at on the `webdav` subdomain.

You can easily mount the webdav server to your local filesystem on a Windows, MacOS, or Linux machine.

## Windows

1. Open the File Explorer.
2. Click on the `Computer` tab.
3. Click on `Map network drive`.
4. Enter the URL of the webdav server in the `Folder` field.
5. Click `Finish`.
6. Enter your Smallweb username and password.
7. Click `OK`.

## MacOS

1. Open the Finder.
2. Click on `Go` in the menu bar.
3. Click on `Connect to Server`.
4. Enter the URL of the webdav server in the `Server Address` field.
5. Click `Connect`.
6. Enter your Smallweb username and password.
7. Click `Connect`.
8. Click `Done`.

## Linux (Ubuntu)

1. Open Nautilus / Files.
2. Click on `Other Locations`.
3. Enter the URL of the webdav server, and prefix with `davs://` (ex: `davs://webdav.example.com`).
4. Click `Connect`.
5. Enter your Smallweb username and password.
6. Click `Connect`.

## Android

[Material Files](https://play.google.com/store/apps/details?id=me.zhanghai.android.files) has built-in support for WebDAV.
