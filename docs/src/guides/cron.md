# Cron Jobs

You can register configure cron tasks from your `smallweb.json[c]` or the `smallweb` field from your `deno.json[c]`.

```json
{
    "crons": [
        {
            "path": "/refresh",
            "schedule": "0 0 * * *",
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

The `path` field is the path to the endpoint that should be called when the cron task runs.

If you want to see a list of all cron tasks, you can run:

```sh
smallweb crons
```
