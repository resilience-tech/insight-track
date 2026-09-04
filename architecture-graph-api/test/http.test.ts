import assert from "node:assert/strict";
import { createServer } from "node:http";
import test from "node:test";
import { createHttpHandler, type Logger } from "../src/http.ts";
import type { Neo4jClient } from "../src/neo4j.ts";
import type { ArchitectureQuestionService } from "../src/questions.ts";
import type { ArchitectureRepository } from "../src/repository.ts";

const token = "0123456789abcdef0123456789abcdef";
const logger: Logger = { debug() {}, info() {}, warn() {}, error() {} };

async function withApi(run: (url: string) => Promise<void>): Promise<void> {
  const repository = {
    listGraphs: async () => []
  } as unknown as ArchitectureRepository;
  const neo4j = { ping: async () => {} } as unknown as Neo4jClient;
  const questions = {} as ArchitectureQuestionService;
  const server = createServer(createHttpHandler({ repository, neo4j, questions, apiToken: token, logger }));
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("test server did not bind");
  try {
    await run(`http://127.0.0.1:${address.port}`);
  } finally {
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
}

test("health endpoint is public", async () => withApi(async (url) => {
  const response = await fetch(`${url}/health/live`);
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { status: "ok" });
  assert.ok(response.headers.get("x-request-id"));
}));

test("graph routes require an internal bearer token", async () => withApi(async (url) => {
  const response = await fetch(`${url}/v1/graphs`);
  assert.equal(response.status, 401);
  assert.match(response.headers.get("www-authenticate") || "", /Bearer/);
}));

test("authorized requests reach graph routes", async () => withApi(async (url) => {
  const response = await fetch(`${url}/v1/graphs`, {
    headers: { Authorization: `Bearer ${token}` }
  });
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { items: [] });
}));
