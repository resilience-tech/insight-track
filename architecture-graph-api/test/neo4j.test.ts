import assert from "node:assert/strict";
import { createServer } from "node:http";
import test from "node:test";
import { Neo4jClient, Neo4jQueryError } from "../src/neo4j.ts";

test("sends parameterized Cypher through Neo4j Query API", async (context) => {
  let observedPath = "";
  let observedBody: Record<string, unknown> = {};
  let observedMode = "";
  const server = createServer(async (request, response) => {
    observedPath = request.url || "";
    observedMode = String(request.headers["access-mode"] || "");
    const chunks: Buffer[] = [];
    for await (const chunk of request) chunks.push(Buffer.from(chunk));
    observedBody = JSON.parse(Buffer.concat(chunks).toString("utf8"));
    response.statusCode = 202;
    response.setHeader("Content-Type", "application/json");
    response.end(JSON.stringify({ data: { fields: ["value"], values: [[42]] }, errors: [] }));
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  context.after(() => server.close());
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("test server did not bind");

  const client = new Neo4jClient(`http://127.0.0.1:${address.port}`, "neo4j", "neo4j", "secret", 1_000);
  const rows = await client.query<{ value: number }>({
    statement: "MATCH (node {id: $id})\nRETURN node.value AS value",
    parameters: { id: "node:1" },
    accessMode: "READ"
  });

  assert.deepEqual(rows, [{ value: 42 }]);
  assert.equal(observedPath, "/db/neo4j/query/v2");
  assert.equal(observedMode, "READ");
  assert.equal(observedBody.statement, "MATCH (node {id: $id}) RETURN node.value AS value");
  assert.deepEqual(observedBody.parameters, { id: "node:1" });
});

test("turns Neo4j errors into a typed error", async (context) => {
  const server = createServer((_request, response) => {
    response.statusCode = 400;
    response.setHeader("Content-Type", "application/json");
    response.end(JSON.stringify({ errors: [{ code: "Neo.ClientError.Statement.SyntaxError", message: "bad query" }] }));
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  context.after(() => server.close());
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("test server did not bind");

  const client = new Neo4jClient(`http://127.0.0.1:${address.port}`, "neo4j", "neo4j", "secret", 1_000);
  await assert.rejects(() => client.query({ statement: "bad" }), (error: unknown) => {
    assert.ok(error instanceof Neo4jQueryError);
    assert.equal(error.code, "Neo.ClientError.Statement.SyntaxError");
    return true;
  });
});
