# App Sandbox

Smallweb apps have access to:

- read and write access to their own directory, and the deno cache directory.
- access to the network, to make HTTP requests.
- access to the env files defined in the global config and in the `.env` file in the app directory.

This sandbox protects the host system from malicious code, and ensures that apps can only access the resources they need.
