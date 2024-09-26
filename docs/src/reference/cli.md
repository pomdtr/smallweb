# CLI Reference

## smallweb

Host websites from your internet folder

### Options

```
  -h, --help   help for smallweb
```

## smallweb api

Interact with the smallweb API

```
smallweb api [flags]
```

### Options

```
  -d, --data string          Data to send in the request body
  -H, --header stringArray   HTTP headers to use
  -h, --help                 help for api
  -X, --method string        HTTP method to use (default "GET")
```

## smallweb api

Interact with the smallweb API

```
smallweb api [flags]
```

### Options

```
  -d, --data string          Data to send in the request body
  -H, --header stringArray   HTTP headers to use
  -h, --help                 help for api
  -X, --method string        HTTP method to use (default "GET")
```

## smallweb capture

Extension capture

```
smallweb capture [flags]
```

### Options

```
  -h, --help   help for capture
```

## smallweb changelog

Show the changelog

```
smallweb changelog [flags]
```

### Options

```
  -h, --help   help for changelog
```

## smallweb completion

Generate the autocompletion script for the specified shell

### Synopsis

Generate the autocompletion script for smallweb for the specified shell.
See each sub-command's help for details on how to use the generated script.


### Options

```
  -h, --help   help for completion
```

## smallweb completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(smallweb completion bash)

To load completions for every new session, execute once:

#### Linux:

	smallweb completion bash > /etc/bash_completion.d/smallweb

#### macOS:

	smallweb completion bash > $(brew --prefix)/etc/bash_completion.d/smallweb

You will need to start a new shell for this setup to take effect.


```
smallweb completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

## smallweb completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	smallweb completion fish | source

To load completions for every new session, execute once:

	smallweb completion fish > ~/.config/fish/completions/smallweb.fish

You will need to start a new shell for this setup to take effect.


```
smallweb completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

## smallweb completion help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type completion help [path to command] for full details.

```
smallweb completion help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

## smallweb completion powershell

Generate the autocompletion script for powershell

### Synopsis

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	smallweb completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.


```
smallweb completion powershell [flags]
```

### Options

```
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

## smallweb completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(smallweb completion zsh)

To load completions for every new session, execute once:

#### Linux:

	smallweb completion zsh > "${fpath[1]}/_smallweb"

#### macOS:

	smallweb completion zsh > $(brew --prefix)/share/zsh/site-functions/_smallweb

You will need to start a new shell for this setup to take effect.


```
smallweb completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

## smallweb config

Open the smallweb config in your editor

```
smallweb config [flags]
```

### Options

```
  -h, --help   help for config
  -j, --json   Output as JSON
```

## smallweb create

Create a new smallweb app

```
smallweb create <app> [flags]
```

### Options

```
  -h, --help              help for create
  -t, --template string   The template to use
```

## smallweb cron

Manage cron jobs

### Options

```
  -h, --help   help for cron
```

## smallweb cron help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type cron help [path to command] for full details.

```
smallweb cron help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

## smallweb cron list

List cron jobs

```
smallweb cron list [flags]
```

### Options

```
      --app string   filter by app
  -h, --help         help for list
      --json         output as json
```

## smallweb cron trigger

Trigger a cron job

```
smallweb cron trigger <id> [flags]
```

### Options

```
  -h, --help   help for trigger
```

## smallweb delete

Delete an app

```
smallweb delete [flags]
```

### Options

```
  -h, --help   help for delete
```

## smallweb docs

Generate smallweb cli documentation

```
smallweb docs [flags]
```

### Options

```
  -h, --help   help for docs
```

## smallweb edit

Open an app in your editor

```
smallweb edit <app> [flags]
```

### Options

```
  -h, --help   help for edit
```

## smallweb gallery

Extension gallery

```
smallweb gallery [flags]
```

### Options

```
  -h, --help   help for gallery
```

## smallweb help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type smallweb help [path to command] for full details.

```
smallweb help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

## smallweb list

List all smallweb apps

```
smallweb list [flags]
```

### Options

```
  -h, --help   help for list
      --json   output as json
```

## smallweb open

Open an app in the browser

```
smallweb open [app] [flags]
```

### Options

```
  -h, --help   help for open
```

## smallweb run

Run an app cli

```
smallweb run <app> [args...] [flags]
```

### Options

```
  -h, --help   help for run
```

## smallweb service

Manage smallweb service

### Options

```
  -h, --help   help for service
```

## smallweb service help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type service help [path to command] for full details.

```
smallweb service help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

## smallweb service install

Install smallweb as a service

```
smallweb service install [flags]
```

### Options

```
  -h, --help   help for install
```

## smallweb service logs

Print service logs

```
smallweb service logs [flags]
```

### Options

```
  -f, --follow   Follow log output
  -h, --help     help for logs
```

## smallweb service restart

Restart smallweb service

```
smallweb service restart [flags]
```

### Options

```
  -h, --help   help for restart
```

## smallweb service start

Start smallweb service

```
smallweb service start [flags]
```

### Options

```
  -h, --help   help for start
```

## smallweb service status

View service status

```
smallweb service status [flags]
```

### Options

```
  -h, --help   help for status
```

## smallweb service stop

Stop smallweb service

```
smallweb service stop [flags]
```

### Options

```
  -h, --help   help for stop
```

## smallweb service uninstall

Uninstall smallweb service

```
smallweb service uninstall [flags]
```

### Options

```
  -h, --help   help for uninstall
```

## smallweb token

Manage api tokens

### Options

```
  -h, --help   help for token
```

## smallweb token create

Create a new token

```
smallweb token create [flags]
```

### Options

```
  -d, --description string   description of the token
  -h, --help                 help for create
```

## smallweb token help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type token help [path to command] for full details.

```
smallweb token help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

## smallweb token list

List all tokens

```
smallweb token list [flags]
```

### Options

```
  -h, --help   help for list
  -j, --json   output as JSON
```

## smallweb token remove

Remove a token

```
smallweb token remove <id> [flags]
```

### Options

```
  -h, --help   help for remove
```

## smallweb up

Start the smallweb evaluation server

```
smallweb up [flags]
```

### Options

```
  -h, --help   help for up
```

## smallweb version

Manage smallweb versions

### Options

```
  -h, --help   help for version
```

## smallweb version help

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type version help [path to command] for full details.

```
smallweb version help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

## smallweb version list

List available versions

```
smallweb version list [flags]
```

### Options

```
  -h, --help   help for list
```

## smallweb version upgrade

Upgrade to the latest version

```
smallweb version upgrade [version] [flags]
```

### Options

```
  -h, --help   help for upgrade
```

## smallweb view

Extension view

```
smallweb view [flags]
```

### Options

```
  -h, --help   help for view
```



<!-- markdownlint-disable-file -->
