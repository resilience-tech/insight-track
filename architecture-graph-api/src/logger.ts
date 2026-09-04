import type { Logger } from "./http.ts";

const levels = { debug: 10, info: 20, warn: 30, error: 40 } as const;

export function createLogger(minimum: keyof typeof levels): Logger {
  const write = (level: keyof typeof levels, message: string, attributes: Record<string, unknown> = {}): void => {
    if (levels[level] < levels[minimum]) return;
    const entry = JSON.stringify({ time: new Date().toISOString(), level, message, ...attributes });
    if (level === "error") console.error(entry);
    else console.log(entry);
  };
  return {
    debug: (message, attributes) => write("debug", message, attributes),
    info: (message, attributes) => write("info", message, attributes),
    warn: (message, attributes) => write("warn", message, attributes),
    error: (message, attributes) => write("error", message, attributes)
  };
}
