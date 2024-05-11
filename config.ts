import * as path from "@std/path";
import { exists } from "@std/fs/exists";

type Config = {
  root: string;
  port?: number;
  env?: Record<string, string>;
};

const homeDir = Deno.env.get("HOME") || Deno.env.get("USERPROFILE")!;

const defaultConfigPath = path.join(homeDir, ".config", "mods", "config.json");

export async function loadConfig(
  configPath: string = defaultConfigPath
): Promise<Config> {
  if (!(await exists(configPath))) {
    return { root: Deno.cwd() };
  }

  const raw = await Deno.readTextFile(configPath);
  const config = (await JSON.parse(raw)) as Config;
  if (!config.root) {
    config.root = Deno.cwd();
  }

  if (config.root.startsWith("~/")) {
    config.root = path.join(homeDir, config.root.slice(2));
  }

  return config;
}

export const config = await loadConfig();
