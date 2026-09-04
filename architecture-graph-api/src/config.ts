export interface Config {
  httpHost: string;
  httpPort: number;
  apiToken: string;
  neo4jUrl: string;
  neo4jDatabase: string;
  neo4jUsername: string;
  neo4jPassword: string;
  neo4jTimeoutMs: number;
  logLevel: "debug" | "info" | "warn" | "error";
}

function required(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function positiveInteger(name: string, fallback: number): number {
  const raw = process.env[name]?.trim();
  if (!raw) return fallback;
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error(`${name} must be a positive integer`);
  }
  return value;
}

export function loadConfig(): Config {
  const apiToken = required("API_TOKEN");
  if (apiToken.length < 32) {
    throw new Error("API_TOKEN must contain at least 32 characters");
  }

  const neo4jUrl = process.env.NEO4J_HTTP_URL?.trim() || "http://localhost:7474";
  const parsedUrl = new URL(neo4jUrl);
  if (!(["http:", "https:"] as string[]).includes(parsedUrl.protocol)) {
    throw new Error("NEO4J_HTTP_URL must be an absolute HTTP(S) URL");
  }

  const neo4jDatabase = process.env.NEO4J_DATABASE?.trim() || "neo4j";
  if (!/^[A-Za-z][A-Za-z0-9._-]{0,62}$/.test(neo4jDatabase)) {
    throw new Error("NEO4J_DATABASE is invalid");
  }

  const logLevel = (process.env.LOG_LEVEL?.trim().toLowerCase() || "info") as Config["logLevel"];
  if (!(new Set(["debug", "info", "warn", "error"])).has(logLevel)) {
    throw new Error("LOG_LEVEL must be debug, info, warn, or error");
  }

  return {
    httpHost: process.env.HTTP_HOST?.trim() || "0.0.0.0",
    httpPort: positiveInteger("HTTP_PORT", 8080),
    apiToken,
    neo4jUrl: parsedUrl.toString().replace(/\/$/, ""),
    neo4jDatabase,
    neo4jUsername: process.env.NEO4J_USERNAME?.trim() || "neo4j",
    neo4jPassword: required("NEO4J_PASSWORD"),
    neo4jTimeoutMs: positiveInteger("NEO4J_TIMEOUT_MS", 10_000),
    logLevel
  };
}
