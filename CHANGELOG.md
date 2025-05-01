# Changelog

## 0.26.1

- add back `--on-demand-tls` flag in `smallweb up` command

## 0.26.0

Full release notes at <https://blog.smallweb.run/posts/v0.26>.

- remove `on-demand-tls` flag in `smallweb up` command
- move `crons`, `private`, `privateRoutes` and `publicRoutes` fields to the `smallweb.json` config
- add `--log-format` flag to `smallweb up` command, supporting `pretty`, `json` and `text` formats
- add `--force` flag to the smallweb link command
- remove `SMALLWEB_DOMAIN` and `SMALLWEB_ADDITIONAL_DOMAINS` environment variables

## 0.25.0

Full release notes at <https://blog.smallweb.run/posts/v0.25>.

- openid connect support
- add support for receiving emails from smallweb apps
- crons moved from app manifest to global config
- ability to access stdin from the `run` entrypoint
- `smallweb up` emit http/console logs on stdout

## 0.24.0

Full release notes at <https://blog.smallweb.run/posts/v0.24>.

- opentelemetry support
- removal of the `smallweb logs` and `smallweb fetch` commands
- enforces deno version >= 2.2
- new format for git urls: `_@<domain>:<app>.git`
- `smallweb up --cron` is deprecated in favor of `smallweb up --enable-crons`

## 0.23.0

Full release notes at <https://blog.smallweb.run/posts/v0.23>.

- git integration (inspired by [dokku](https://dokku.com/))
- automatic detection of smallweb directory
- add back `smallweb open`
- remove automatic typescript and tsx support in the static server

## 0.22.0

Full release notes at <https://blog.smallweb.run/posts/v0.22>.

- Updated config format
- Removed support for global env vars and secrets
- Admin apps no longer have access to the `.smallweb` dir

## 0.21.0

Full release notes at <https://blog.smallweb.run/posts/v0.21>.

- fix permissions of the built-in sftp server
- add support for acmedns
- add support for SPA in the static server
- remove `smallweb services` command
- remove `smallweb secrets` command
- remove `smallweb sync` command
- remove `smallweb changelog` command
- remove `smallweb upgrade` command
- `--domain` global flag

## 0.20.0

Full release notes at <https://blog.smallweb.run/posts/v0.20>.

- access the smallweb cli and apps cli entrypoint using ssh
- 404 pages and `_redirects` file for the static server
- support for passing additional deno args in the app config

## 0.19.0

Full release notes at <https://blog.smallweb.run/posts/v0.19>.

- new install script to bootstrap smallweb on a new VPS
- `--dir` global flag to specify the smallweb directory
- `smallweb init` command to create a new smallweb directory
- `smallweb logs` api updates
- removal of the default domain
- `addr`, `cert` and `key` are now specified as flags of the `up command`
- clean urls and custom head elements for the static server

## 0.18.0

Full release notes at <https://blog.smallweb.run/posts/v0.18>.

- view console logs using `smallweb logs --console`
- add pretty error pages
- admin apps are set in the global config instead of the app config
- fixed smallweb sync command
- add ability to pass custom domains through an env var
- add ability to set an additionals wildcard domains
- return a 404 when the app is not found
- add docker image, published on `ghcr.io/pomdtr/smallweb`

## 0.17.0

Full release notes at <https://blog.smallweb.run/posts/v0.17>.

- add encryption using sops (through `secrets.enc.env` files)
- add `smallweb doctor` command
- add `--template` flag for `smallweb logs` and `smallweb list`

## 0.16.0

Full release notes at <https://blog.smallweb.run/posts/v0.16>.

- new runtime allowing process to be reused for multiple requests
- new plugin default dirs ($SMALLWEB_DIR/.smallweb/plugins and $XDG_DATA_HOME/smallweb/plugins)
- new `smallweb sync` set of commands
- add a template flag to `smallweb create`
- admin apps now have access to the cli!
- fixed deno permission when an app dir is a symlink
- add support for `index.md` in the static server

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
