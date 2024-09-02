# Cron Jobs

You can register configure cron tasks from your `smallweb.json[c]` or the `smallweb` field from your `deno.json[c]`.

```json
{
    "crons": [
        {
            "name": "daily-task",
            "schedule": "0 0 * * *",
            "args": [],
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

In order to handle the cron tasks, your app default export should have a `run` method that will be called when the task is triggered.

```ts
export default {
    run(_args: string[]) {
        console.log("Running daily task");
    }
}
```

If you want to see a list of all cron tasks, you can run:

```sh
smallweb cron ls
```

To trigger one, you can use either the `smallweb cron trigger` or the `smallweb run` command:

```sh
smallweb cron trigger daily-task
```
