import { issuer } from "npm:@openauthjs/openauth@^0.3.7";
import { GithubProvider } from "npm:/@openauthjs/openauth@^0.3.7/provider/github";
import { Octokit } from "npm:octokit@^4.1.0";
import { THEME_SST } from "npm:/@openauthjs/openauth@^0.3.7/ui/theme";
import { MemoryStorage } from "npm:/@openauthjs/openauth@^0.3.7/storage/memory";
import { createSubjects } from "npm:@openauthjs/openauth@^0.3.7/subject";
import { object, string } from "npm:valibot@1.0.0"
import { signingKeys } from "npm:@openauthjs/openauth@^0.3.7/keys";
import { jwtVerify, SignJWT } from "npm:jose"
import * as fs from "jsr:@std/fs@^1.0.11";

const { GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET } = Deno.env.toObject();
if (!GITHUB_CLIENT_ID || !GITHUB_CLIENT_SECRET) {
    throw new Error("Missing GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET");
}

const storage = MemoryStorage({
    persist: "./data/db.json",
})

const iss = issuer({
    theme: THEME_SST,
    providers: {
        github: GithubProvider({
            clientID: GITHUB_CLIENT_ID,
            clientSecret: GITHUB_CLIENT_SECRET,
            scopes: ["user:email"],
        }),
    },
    storage,
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


export default {
    fetch: async (req: Request) => {
        await fs.ensureDir("./data");
        const url = new URL(req.url);

        if (url.pathname === "/.well-known/openid-configuration") {
            const resp = await iss.request(new URL("/.well-known/oauth-authorization-server", url))
            const oauth2Config = await resp.json()
            return Response.json({
                ...oauth2Config,
                userinfo_endpoint: new URL("/userinfo", url).toString(),
                scopes_supported: ["openid", "email"],
                id_token_signing_alg_values_supported: ["ES256"],
            })
        }

        if (url.pathname === "/token") {
            if (req.headers.get("content-type") !== "application/x-www-form-urlencoded") {
                return new Response("Invalid content type", {
                    status: 400,
                })
            }

            const params = new URLSearchParams(await req.text())
            if (!params.has("client_id")) {
                return new Response("Missing client_id", {
                    status: 400,
                })
            }

            const resp = await iss.request(req.url, {
                method: req.method,
                headers: req.headers,
                body: params.toString(),
            })

            if (!resp.ok) {
                return resp
            }

            const tokens = await resp.json()

            const signinKey = await signingKeys(storage).then((keys) => keys[0])
            const access_token = await jwtVerify<{
                properties: {
                    email: string
                }
            }>(tokens.access_token, signinKey.public)
            const jwt = new SignJWT({
                aud: access_token.payload.aud,
                iss: access_token.payload.iss,
                sub: access_token.payload.sub,
                exp: access_token.payload.exp,
                email: access_token.payload.properties.email,
            })

            jwt.setProtectedHeader(access_token.protectedHeader)
            jwt.sign(signinKey.private)


            return Response.json({
                id_token: await jwt.sign(signinKey.private),
                ...tokens,
            })
        }

        return iss.fetch(req);
    }
}
