# Changelog

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
