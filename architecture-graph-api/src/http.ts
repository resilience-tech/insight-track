import { createHash, randomUUID, timingSafeEqual } from "node:crypto";
import type { IncomingMessage, ServerResponse } from "node:http";
import { NODE_KINDS, RELATIONSHIP_TYPES } from "./types.ts";
import type { Neo4jClient } from "./neo4j.ts";
import { Neo4jQueryError } from "./neo4j.ts";
import type { ArchitectureRepository } from "./repository.ts";
import type { ArchitectureQuestionService } from "./questions.ts";
import { QuestionPlanningError } from "./questions.ts";
import {
  ValidationError,
  validateGraphId,
  validateGraphInput,
  validateQuestion,
  validateSnapshot
} from "./validation.ts";

const maximumBodyBytes = 2 * 1024 * 1024;

export interface Logger {
  debug(message: string, attributes?: Record<string, unknown>): void;
  info(message: string, attributes?: Record<string, unknown>): void;
  warn(message: string, attributes?: Record<string, unknown>): void;
  error(message: string, attributes?: Record<string, unknown>): void;
}

export interface HttpDependencies {
  repository: ArchitectureRepository;
  questions: ArchitectureQuestionService;
  neo4j: Neo4jClient;
  apiToken: string;
  logger: Logger;
}

class HttpError extends Error {
  readonly status: number;
  readonly slug: string;

  constructor(status: number, slug: string, message: string) {
    super(message);
    this.status = status;
    this.slug = slug;
  }
}

function requestId(request: IncomingMessage): string {
  const supplied = String(request.headers["x-request-id"] || "").trim();
  return /^[A-Za-z0-9._-]{1,128}$/.test(supplied) ? supplied : randomUUID();
}

function sendJson(response: ServerResponse, status: number, value: unknown, id: string): void {
  const body = JSON.stringify(value);
  response.statusCode = status;
  response.setHeader("Cache-Control", "no-store");
  response.setHeader("Content-Length", Buffer.byteLength(body));
  response.setHeader("Content-Type", "application/json; charset=utf-8");
  response.setHeader("X-Content-Type-Options", "nosniff");
  response.setHeader("X-Frame-Options", "DENY");
  response.setHeader("X-Request-ID", id);
  response.end(body);
}

function sendProblem(response: ServerResponse, error: HttpError, id: string): void {
  const body = JSON.stringify({
    type: `https://insight-track.example/problems/${error.slug}`,
    title: error.name === "Error" ? error.slug.replaceAll("-", " ") : error.name,
    status: error.status,
    detail: error.message,
    request_id: id
  });
  response.statusCode = error.status;
  response.setHeader("Cache-Control", "no-store");
  response.setHeader("Content-Length", Buffer.byteLength(body));
  response.setHeader("Content-Type", "application/problem+json; charset=utf-8");
  response.setHeader("X-Content-Type-Options", "nosniff");
  response.setHeader("X-Frame-Options", "DENY");
  response.setHeader("X-Request-ID", id);
  if (error.status === 401) response.setHeader("WWW-Authenticate", 'Bearer realm="architecture-graph-api"');
  response.end(body);
}

function authenticated(request: IncomingMessage, expectedDigest: Buffer): boolean {
  const match = String(request.headers.authorization || "").trim().match(/^Bearer\s+(.+)$/i);
  const actualDigest = createHash("sha256").update(match?.[1] || "").digest();
  return Boolean(match) && timingSafeEqual(actualDigest, expectedDigest);
}

async function readJson(request: IncomingMessage): Promise<unknown> {
  const contentType = String(request.headers["content-type"] || "").split(";", 1)[0].trim().toLowerCase();
  if (contentType !== "application/json") {
    throw new HttpError(415, "unsupported-media-type", "Content-Type must be application/json");
  }
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of request) {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += buffer.length;
    if (size > maximumBodyBytes) throw new HttpError(413, "payload-too-large", "Request body exceeds 2 MiB");
    chunks.push(buffer);
  }
  try {
    return JSON.parse(Buffer.concat(chunks).toString("utf8"));
  } catch {
    throw new HttpError(422, "validation-error", "Request body must contain valid JSON");
  }
}

function positiveQuery(url: URL, name: string, fallback: number, maximum: number): number {
  const raw = url.searchParams.get(name);
  if (!raw) return fallback;
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < 1 || value > maximum) {
    throw new HttpError(422, "validation-error", `${name} must be between 1 and ${maximum}`);
  }
  return value;
}

function pathSegments(url: URL): string[] {
  try {
    return url.pathname.split("/").filter(Boolean).map(decodeURIComponent);
  } catch {
    throw new HttpError(400, "invalid-path", "Request path contains invalid encoding");
  }
}

async function route(
  request: IncomingMessage,
  response: ServerResponse,
  id: string,
  dependencies: HttpDependencies
): Promise<void> {
  const method = request.method || "GET";
  const url = new URL(request.url || "/", "http://localhost");
  const segments = pathSegments(url);

  if (method === "GET" && url.pathname === "/health/live") {
    sendJson(response, 200, { status: "ok" }, id);
    return;
  }
  if (method === "GET" && url.pathname === "/health/ready") {
    await dependencies.neo4j.ping();
    sendJson(response, 200, { status: "ok" }, id);
    return;
  }

  if (segments[0] !== "v1") throw new HttpError(404, "not-found", "Route was not found");

  if (method === "GET" && segments.length === 2 && segments[1] === "schema") {
    sendJson(response, 200, {
      node_kinds: NODE_KINDS,
      relationship_types: RELATIONSHIP_TYPES,
      question_intents: ["summary", "dependencies", "dependents", "shortest_path", "root_causes", "blast_radius", "data_access"],
      example_questions: [
        "What can cause errors in Project Page?",
        "Show the path from Project Page to projects table",
        "What breaks if Identity Provider fails?",
        "Which endpoints write to project_members?"
      ]
    }, id);
    return;
  }

  if (segments[1] !== "graphs") throw new HttpError(404, "not-found", "Route was not found");

  if (method === "GET" && segments.length === 2) {
    sendJson(response, 200, { items: await dependencies.repository.listGraphs() }, id);
    return;
  }

  if (segments.length < 3) throw new HttpError(404, "not-found", "Route was not found");
  const graphId = validateGraphId(segments[2]);

  if (method === "PUT" && segments.length === 3) {
    const graph = await dependencies.repository.upsertGraph(graphId, validateGraphInput(await readJson(request)));
    sendJson(response, 200, graph, id);
    return;
  }
  if (method === "GET" && segments.length === 3) {
    const graph = await dependencies.repository.getGraph(graphId);
    if (!graph) throw new HttpError(404, "graph-not-found", "Architecture graph was not found");
    sendJson(response, 200, graph, id);
    return;
  }
  if (method === "PUT" && segments.length === 4 && segments[3] === "snapshot") {
    if (!await dependencies.repository.getGraph(graphId)) {
      throw new HttpError(404, "graph-not-found", "Create the architecture graph before importing a snapshot");
    }
    const snapshot = await dependencies.repository.replaceSnapshot(graphId, validateSnapshot(await readJson(request)));
    sendJson(response, 200, snapshot, id);
    return;
  }
  if (method === "GET" && segments.length === 4 && segments[3] === "snapshot") {
    if (!await dependencies.repository.getGraph(graphId)) {
      throw new HttpError(404, "graph-not-found", "Architecture graph was not found");
    }
    sendJson(response, 200, await dependencies.repository.getSnapshot(graphId), id);
    return;
  }
  if (method === "POST" && segments.length === 4 && segments[3] === "questions") {
    const answer = await dependencies.questions.ask(graphId, validateQuestion(await readJson(request)));
    sendJson(response, 200, answer, id);
    return;
  }
  if (method === "GET" && segments.length === 4 && segments[3] === "nodes") {
    const limit = positiveQuery(url, "limit", 100, 1_000);
    const search = (url.searchParams.get("search") || "").trim();
    const kind = (url.searchParams.get("kind") || "").trim();
    if (search.length > 200) throw new HttpError(422, "validation-error", "search is too long");
    if (kind && !(NODE_KINDS as readonly string[]).includes(kind)) {
      throw new HttpError(422, "validation-error", "kind is unsupported");
    }
    sendJson(response, 200, { items: await dependencies.repository.listNodes(graphId, search, kind, limit) }, id);
    return;
  }
  if (segments.length >= 5 && segments[3] === "nodes") {
    const nodeId = segments[4];
    if (method === "GET" && segments.length === 5) {
      const node = await dependencies.repository.getNode(graphId, nodeId);
      if (!node) throw new HttpError(404, "node-not-found", "Architecture node was not found");
      sendJson(response, 200, node, id);
      return;
    }
    if (method === "GET" && segments.length === 6 && (segments[5] === "dependencies" || segments[5] === "dependents")) {
      const maxDepth = positiveQuery(url, "max_depth", 8, 15);
      const limit = positiveQuery(url, "limit", 50, 200);
      const node = await dependencies.repository.getNode(graphId, nodeId);
      if (!node) throw new HttpError(404, "node-not-found", "Architecture node was not found");
      const result = segments[5] === "dependencies"
        ? await dependencies.repository.dependencies(graphId, nodeId, maxDepth, limit)
        : await dependencies.repository.dependents(graphId, nodeId, maxDepth, limit);
      sendJson(response, 200, result, id);
      return;
    }
  }
  if (method === "GET" && segments.length === 4 && segments[3] === "paths") {
    const from = url.searchParams.get("from")?.trim() || "";
    const to = url.searchParams.get("to")?.trim() || "";
    if (!from || !to) throw new HttpError(422, "validation-error", "from and to node IDs are required");
    const maxDepth = positiveQuery(url, "max_depth", 10, 15);
    const result = await dependencies.repository.shortestPath(graphId, from, to, maxDepth);
    sendJson(response, 200, result, id);
    return;
  }

  throw new HttpError(404, "not-found", "Route was not found");
}

export function createHttpHandler(dependencies: HttpDependencies): (request: IncomingMessage, response: ServerResponse) => void {
  const expectedDigest = createHash("sha256").update(dependencies.apiToken).digest();
  return (request, response) => {
    const id = requestId(request);
    const startedAt = performance.now();
    const url = new URL(request.url || "/", "http://localhost");
    response.on("finish", () => dependencies.logger.info("request", {
      request_id: id,
      method: request.method,
      path: url.pathname,
      status: response.statusCode,
      duration_ms: Math.round((performance.now() - startedAt) * 100) / 100
    }));

    if (url.pathname.startsWith("/v1/") && !authenticated(request, expectedDigest)) {
      sendProblem(response, new HttpError(401, "unauthorized", "A valid bearer token is required"), id);
      return;
    }

    route(request, response, id, dependencies).catch((error: unknown) => {
      if (error instanceof HttpError) {
        sendProblem(response, error, id);
        return;
      }
      if (error instanceof ValidationError) {
        sendProblem(response, new HttpError(422, "validation-error", error.message), id);
        return;
      }
      if (error instanceof QuestionPlanningError) {
        const problem = new HttpError(422, "question-needs-context", error.message);
        dependencies.logger.debug("question could not be planned", { request_id: id, suggestions: error.suggestions.map((node) => node.id) });
        sendProblem(response, problem, id);
        return;
      }
      if (error instanceof Neo4jQueryError) {
        dependencies.logger.error("Neo4j query failed", { request_id: id, code: error.code, error: error.message });
        sendProblem(response, new HttpError(502, "graph-store-error", "Neo4j could not complete the graph operation"), id);
        return;
      }
      dependencies.logger.error("unhandled request error", { request_id: id, error: error instanceof Error ? error.stack : String(error) });
      sendProblem(response, new HttpError(500, "internal-error", "An unexpected error occurred"), id);
    });
  };
}
