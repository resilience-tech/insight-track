import assert from "node:assert/strict";
import test from "node:test";
import { ValidationError, validateSnapshot } from "../src/validation.ts";

test("accepts a valid architecture snapshot", () => {
  const result = validateSnapshot({
    nodes: [
      { id: "service:api", kind: "service", name: "Project API" },
      { id: "db:postgres", kind: "database", name: "PostgreSQL" }
    ],
    relationships: [
      { id: "api-uses-postgres", type: "USES_DATABASE", from: "service:api", to: "db:postgres" }
    ]
  });
  assert.equal(result.nodes.length, 2);
  assert.equal(result.relationships[0].type, "USES_DATABASE");
});

test("rejects relationships whose endpoints are missing", () => {
  assert.throws(() => validateSnapshot({
    nodes: [{ id: "service:api", kind: "service", name: "Project API" }],
    relationships: [{ id: "missing", type: "CALLS", from: "service:api", to: "service:missing" }]
  }), ValidationError);
});

test("rejects duplicate node IDs", () => {
  assert.throws(() => validateSnapshot({
    nodes: [
      { id: "service:api", kind: "service", name: "One" },
      { id: "service:api", kind: "service", name: "Two" }
    ],
    relationships: []
  }), /duplicate id/);
});

test("rejects unknown relationship types", () => {
  assert.throws(() => validateSnapshot({
    nodes: [
      { id: "service:api", kind: "service", name: "Project API" },
      { id: "db:postgres", kind: "database", name: "PostgreSQL" }
    ],
    relationships: [
      { id: "invalid", type: "MAYBE_USES", from: "service:api", to: "db:postgres" }
    ]
  }), /unsupported/);
});
