# Changelog

## 0.16.0

- new runtime allowing process to be reused for multiple requests
- new plugin default dirs ($SMALLWEB_DIR/.smallweb/plugins and $XDG_DATA_HOME/smallweb/plugins)
- admin apps now have access to the cli!
- fixed deno permission when an app dir is a symlink

## 0.15.0

Full release notes at <https://blog.smallweb.run/posts/v0.15>.

- most smallweb commands now works wherever you're smallweb dir is synced
- apps now only have write access to the `data` subdirectory (ex: `~/smallweb/<app>/data`)
- autentication is now handled at the deno level
- admin apps replace the legacy rest api
- global env vars should be set in `$SMALLWEB_DIR/.env`
- You should explicitely use `smallweb up --cron` flag to launch the cron daemon
- `smallweb logs` now support an `--app` flag

## 0.14.6

- add `SMALLWEB_DATA_DIR` environment variable to specify the data directory

## 0.14.5

- fix `manifest` not showing up in `/v0/apps`

## 0.14.4

- add support for manifest in `/v0/apps` and `/v0/apps/{app}`

## 0.14.3

- fix `smallweb log` parsing errors

## 0.14.2

- fix usage of `smallweb:file-server` in the code (instead of `smallweb:static`)

## 0.14.1

- fixed unmarshaling of json with comments
- rename `smallweb app fork` to `smallweb app clone`
- fix some unmarshaling issues with logs

## 0.14.0

- token scopes
- new command smallweb logs
- dropped smallweb terminal
- added smallweb rest api
- smallweb static server now supports tsx and jsx files
- docs can be hosted as a smallweb app
- new app commands: smallweb app rename, smallweb app clone, smallweb app delete

## 0.13.6

- Fixed file content getting overwritten by html in the webdav server
