import { cli } from 'npm:gunshi@0.17.0'

export default {
    async run(args: string[]) {
        await cli(args, {
            options: {
                name: {
                    type: "string",
                    default: "world",
                }
            },
            run: (ctx) => {
                console.log(`Hello ${ctx.values.name}!`)
            }
        })
    }
}
