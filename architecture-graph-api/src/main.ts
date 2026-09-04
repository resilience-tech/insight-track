import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import { loadConfig } from "./config.ts";
import { createHttpHandler } from "./http.ts";
import { createLogger } from "./logger.ts";
import { Neo4jClient } from "./neo4j.ts";
import { ArchitectureQuestionService } from "./questions.ts";
import { ArchitectureRepository } from "./repository.ts";

function delay(milliseconds: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

async function connect(client: Neo4jClient): Promise<void> {
  let lastError: unknown;
  for (let attempt = 1; attempt <= 30; attempt += 1) {
    try {
      await client.ping();
      return;
    } catch (error) {
      lastError = error;
      await delay(Math.min(attempt * 250, 2_000));
    }
  }
  throw lastError;
}

async function main(): Promise<void> {
  const config = loadConfig();
  const logger = createLogger(config.logLevel);
  const neo4j = new Neo4jClient(
    config.neo4jUrl,
    config.neo4jDatabase,
    config.neo4jUsername,
    config.neo4jPassword,
    config.neo4jTimeoutMs
  );
  await connect(neo4j);

  const repository = new ArchitectureRepository(neo4j);
  const migration = await readFile(new URL("../migrations/001_schema.cypher", import.meta.url), "utf8");
  const statements = migration.split(";").map((value) => value.trim()).filter(Boolean);
  await repository.migrate(statements);

  const questions = new ArchitectureQuestionService(repository);
  const server = createServer(createHttpHandler({
    repository,
    questions,
    neo4j,
    apiToken: config.apiToken,
    logger
  }));
  server.requestTimeout = 30_000;
  server.headersTimeout = 10_000;
  server.keepAliveTimeout = 65_000;
  server.maxHeadersCount = 100;

  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(config.httpPort, config.httpHost, () => {
      logger.info("architecture graph API started", { host: config.httpHost, port: config.httpPort });
      resolve();
    });
  });

  const shutdown = (signal: string): void => {
    logger.info("stopping architecture graph API", { signal });
    server.close((error) => {
      if (error) {
        logger.error("HTTP shutdown failed", { error: error.message });
        process.exitCode = 1;
      }
    });
  };
  process.once("SIGINT", () => shutdown("SIGINT"));
  process.once("SIGTERM", () => shutdown("SIGTERM"));
}

main().catch((error: unknown) => {
  console.error(JSON.stringify({
    time: new Date().toISOString(),
    level: "error",
    message: "architecture graph API failed to start",
    error: error instanceof Error ? error.stack : String(error)
  }));
  process.exitCode = 1;
});
