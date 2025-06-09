import { vt } from "jsr:@pomdtr/vt@1.11.3";

export default {
    run: async (args: string[]) => {
        await vt.parse(args)
    }
};
