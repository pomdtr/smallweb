# CLI Reference

## smallweb

Host websites from your internet folder

### Options

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
  -h, --help            help for smallweb
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb cron

Manage cron jobs

### Options

```
  -h, --help   help for cron
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb init

Init a new smallweb app

```
smallweb init [dir] [flags]
```

### Options

```
  -h, --help              help for init
  -t, --template string   The template to use
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb open

Open the smallweb app specified by dir in the browser

```
smallweb open [app] [flags]
```

### Options

```
  -h, --help   help for open
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb service

Manage smallweb service

### Options

```
  -h, --help   help for service
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb service edit

Edit smallweb service configuration

```
smallweb service edit [flags]
```

### Options

```
  -h, --help   help for edit
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb token

Generate a random token

```
smallweb token [flags]
```

### Options

```
  -h, --help   help for token
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb types

Print smallweb types

```
smallweb types [flags]
```

### Options

```
  -h, --help   help for types
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
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

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```

## smallweb upgrade

Upgrade to the latest version

```
smallweb upgrade [version] [flags]
```

### Options

```
  -h, --help   help for upgrade
```

### Options inherited from parent commands

```
  -c, --config string   config file (default "/Users/pomdtr/.config/smallweb/config.json")
```



<!-- markdownlint-disable-file -->
