import { issuer } from "npm:@openauthjs/openauth@^0.3.7";
import { GithubProvider } from "npm:/@openauthjs/openauth@^0.3.7/provider/github";
import { Octokit } from "npm:octokit@^4.1.0";
import { THEME_SST } from "npm:/@openauthjs/openauth@^0.3.7/ui/theme";
import { MemoryStorage } from "npm:/@openauthjs/openauth@^0.3.7/storage/memory";
import { createSubjects } from "npm:@openauthjs/openauth@^0.3.7/subject";
import { object, string } from "npm:valibot@1.0.0"
import * as fs from "jsr:@std/fs@^1.0.11";

const { GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET } = Deno.env.toObject();
if (!GITHUB_CLIENT_ID || !GITHUB_CLIENT_SECRET) {
    throw new Error("Missing GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET");
}

await fs.ensureDir("./data");

export default issuer({
    theme: THEME_SST,
    providers: {
        github: GithubProvider({
            clientID: GITHUB_CLIENT_ID,
            clientSecret: GITHUB_CLIENT_SECRET,
            scopes: ["user:email"],
        }),
    },
    storage: MemoryStorage({
        persist: "./data/db.json",
    }),
    subjects: createSubjects({
        user: object({
            email: string(),
        })
    }),
    success: async (res, input) => {
        switch (input.provider) {
            case "github": {
                const octokit = new Octokit({
                    auth: input.tokenset.access,
                });

                const emails = await octokit.rest.users
                    .listEmailsForAuthenticatedUser();
                const email = emails.data.find((email) => email.primary)
                    ?.email;

                if (!email) throw new Error("No primary email");
                return res.subject("user", {
                    email,
                });
            }
        }
    },
})


