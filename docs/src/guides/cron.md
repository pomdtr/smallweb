# Cron Tasks

You can register configure cron tasks from your `smallweb.json[c]` or the `smallweb` field from your `deno.json[c]`.

```json
{
    "crons": [
        {
            "name": "refresh-cache",
            "schedule": "0 0 * * *",
            "command": "./refresh-cache.ts",
            "args": []
        }
    ]
}
```

The schedule field is a cron expression that defines when the task should run. It follows the standard cron syntax, with five fields separated by spaces. You can also use the following shortcuts:

- `@hourly`: Run once an hour, at the beginning of the hour.
- `@daily`: Run once a day, at midnight.
- `@weekly`: Run once a week, at midnight on Sunday.
- `@monthly`: Run once a month, at midnight on the first day of the month.
- `@yearly`: Run once a year, at midnight on January 1st.

Cron tasks can be defined in any language, as long as the file is executable and has the appropriate shebang line. You'll probably want to use deno as the runtime, more information on how to do that can be found in the [deno docs](https://docs.deno.com/runtime/tutorials/hashbang/).

You can trigger a cron task manually by running the following command:

```sh
smallweb cron trigger <cron>
```

If you're not in the app directory, you can specify the app name with the `--app` flag:

```sh
smallweb cron trigger --app <app> <cron>
```

If you want to see a list of all cron tasks, you can run:

```sh
smallweb cron list
```

If you're not in the app directory, you can specify the app name with the `--app` flag, or use the `--all` flag to list all cron tasks:

```sh
# list all cron tasks
smallweb cron list --all
# list all cron tasks for a specific domain
smallweb cron list --domain <domain>
# list all cron tasks for a specific app
smallweb cron list --app <app>
```
