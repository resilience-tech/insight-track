import assert from "node:assert/strict";
import test from "node:test";
import { planQuestion, resolveMentionedNodes } from "../src/questions.ts";
import type { ArchitectureNode } from "../src/types.ts";

const graphId = "insight-track";
const nodes: ArchitectureNode[] = [
  { id: "page:project", graph_id: graphId, kind: "page", name: "Project Page", description: "", technology: "React", metadata: {} },
  { id: "service:project-api", graph_id: graphId, kind: "service", name: "Project API", description: "", technology: "Go", metadata: {} },
  { id: "table:projects", graph_id: graphId, kind: "table", name: "projects table", description: "", technology: "PostgreSQL", metadata: {} },
  { id: "external:identity", graph_id: graphId, kind: "identity_provider", name: "Identity Provider", description: "", technology: "OIDC", metadata: {} }
];

test("resolves exact architecture names from a natural-language question", () => {
  const result = resolveMentionedNodes("Show the path from Project Page to projects table", nodes);
  assert.deepEqual(result.map((node) => node.id), ["page:project", "table:projects"]);
});

test("plans a shortest path question with two mentioned nodes", () => {
  const plan = planQuestion({
    question: "Show the path from Project Page to projects table",
    max_depth: 10,
    limit: 50
  }, nodes);
  assert.equal(plan.intent, "shortest_path");
  assert.equal(plan.source?.id, "page:project");
  assert.equal(plan.target?.id, "table:projects");
  assert.equal(plan.max_depth, 10);
});

test("plans blast-radius and root-cause questions", () => {
  const blastRadius = planQuestion({ question: "What breaks if Identity Provider fails?" }, nodes);
  assert.equal(blastRadius.intent, "blast_radius");
  assert.equal(blastRadius.source?.id, "external:identity");

  const rootCauses = planQuestion({ question: "What can cause errors in Project Page?" }, nodes);
  assert.equal(rootCauses.intent, "root_causes");
  assert.equal(rootCauses.source?.id, "page:project");
});

test("resolves data nodes when the question omits the kind suffix", () => {
  const plan = planQuestion({ question: "Which endpoints write to projects?" }, nodes);
  assert.equal(plan.intent, "data_access");
  assert.equal(plan.source?.id, "table:projects");
});

test("explicit node IDs remove ambiguity", () => {
  const plan = planQuestion({
    question: "How are these connected?",
    focus_node_id: "page:project",
    target_node_id: "table:projects"
  }, nodes);
  assert.equal(plan.intent, "shortest_path");
  assert.equal(plan.source?.id, "page:project");
  assert.equal(plan.target?.id, "table:projects");
});
