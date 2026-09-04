# Architecture Graph API

A Node.js and TypeScript REST service that stores complete software architecture in Neo4j and answers architecture questions by executing bounded, parameterized Cypher queries.

The included InsightTrack seed follows this chain all the way down to storage details:

```text
System → Frontend/Service → Page/Endpoint → Handler → Function → Repository → Database → Table → Column/Constraint
```

It also models middleware, caches, identity providers, email services, telemetry, and relationships such as `CALLS`, `READS_FROM`, `WRITES_TO`, and `USES_MIDDLEWARE`.

## Questions it can answer

- What can cause errors in Project Page?
- Show the path from Project Page to projects table.
- What breaks if Identity Provider fails?
- Which endpoints write to project_members?
- What does `POST /projects/{id}/invite` depend on?
- Give me an overview of the architecture.

`POST /v1/graphs/{graphId}/questions` recognizes seven safe intents:

| Intent | Graph operation |
| --- | --- |
| `summary` | Count nodes by kind and relationships by type |
| `dependencies` | Traverse outgoing dependency paths |
| `dependents` | Traverse incoming dependency paths |
| `shortest_path` | Find the shortest connection between two nodes |
| `root_causes` | Trace downstream failure points |
| `blast_radius` | Trace upstream components affected by a failure |
| `data_access` | Find databases, tables, columns, caches, or their users |

The answer includes both a short explanation and the complete evidence path. It also returns the Cypher template and parameters used, so every answer is inspectable.

## Why the question endpoint is safe

Natural language selects a predefined query template and resolves known graph nodes. It never runs arbitrary Cypher generated from user text. Every query:

- is scoped by `graph_id`;
- uses parameters for user-controlled values;
- limits path depth to 15 hops;
- limits returned paths to 200;
- runs with Neo4j read routing for question operations.

For ambiguous questions, provide `focus_node_id` and optionally `target_node_id`.

## Run locally

Requirements: Docker with Compose v2.

```bash
docker compose up --build --detach
make seed
```

Services:

- REST API: `http://localhost:8083`
- Neo4j Browser: `http://localhost:7474`
- Neo4j Bolt: `neo4j://localhost:7687`

Local credentials are defined only for development in `compose.yaml`.

## Ask a question

```bash
curl --fail --silent --show-error \
  -H 'Authorization: Bearer local-architecture-token-0123456789abcdef' \
  -H 'Content-Type: application/json' \
  -d '{
    "question": "Show the path from Project Page to projects table",
    "focus_node_id": "page:project",
    "target_node_id": "table:projects",
    "max_depth": 12
  }' \
  http://localhost:8083/v1/graphs/insight-track/questions
```

Example response shape:

```json
{
  "question": "Show the path from Project Page to projects table",
  "intent": "shortest_path",
  "answer": "Project Page → React Web App → Project List → loadProjects → GET /projects → ListProjectsHandler → listProjects → ProjectRepository → projects table",
  "matched_entities": [],
  "evidence": {
    "paths": [
      {
        "nodes": [],
        "relationships": [],
        "hops": 8
      }
    ]
  },
  "graph_query": {
    "template": "shortest_path",
    "cypher": "MATCH ...",
    "parameters": {
      "graphId": "insight-track",
      "sourceId": "page:project",
      "targetId": "table:projects"
    }
  }
}
```

The real response contains every node and relationship in each evidence path; arrays above are shortened for readability.

## REST endpoints

| Method | Endpoint | Purpose |
| --- | --- | --- |
| `GET` | `/health/live` | Process liveness |
| `GET` | `/health/ready` | Neo4j readiness |
| `GET` | `/v1/schema` | Supported node kinds, edges, and question examples |
| `GET` | `/v1/graphs` | List architecture graphs |
| `PUT` | `/v1/graphs/{graphId}` | Create or update graph metadata |
| `GET` | `/v1/graphs/{graphId}` | Read graph metadata and counts |
| `PUT` | `/v1/graphs/{graphId}/snapshot` | Atomically replace all nodes and edges |
| `GET` | `/v1/graphs/{graphId}/snapshot` | Export a portable JSON snapshot |
| `POST` | `/v1/graphs/{graphId}/questions` | Ask an architecture question |
| `GET` | `/v1/graphs/{graphId}/nodes` | Search nodes by name or kind |
| `GET` | `/v1/graphs/{graphId}/nodes/{nodeId}` | Read one node |
| `GET` | `/v1/graphs/{graphId}/nodes/{nodeId}/dependencies` | Trace downstream dependencies |
| `GET` | `/v1/graphs/{graphId}/nodes/{nodeId}/dependents` | Trace upstream dependents |
| `GET` | `/v1/graphs/{graphId}/paths?from=...&to=...` | Find a shortest connection |

All `/v1` endpoints require the configured internal bearer token. In a browser-facing deployment, the main PostgreSQL/OIDC control plane should authenticate the user and authorize access to the requested architecture graph before proxying requests here.

## Snapshot format

```json
{
  "nodes": [
    {
      "id": "endpoint:list-projects",
      "kind": "endpoint",
      "name": "GET /projects",
      "technology": "HTTP",
      "metadata": { "method": "GET", "path": "/projects" }
    },
    {
      "id": "handler:list-projects",
      "kind": "handler",
      "name": "ListProjectsHandler"
    }
  ],
  "relationships": [
    {
      "id": "list-projects-handler",
      "type": "HANDLED_BY",
      "from": "endpoint:list-projects",
      "to": "handler:list-projects"
    }
  ]
}
```

Node and relationship IDs must be unique inside the snapshot. Every relationship endpoint must exist. Import is an atomic replacement, so export the current snapshot before making manual bulk changes.

## Development

This project uses Node.js 24's native TypeScript execution and has no runtime package dependencies.

```bash
node --test test/*.test.ts
node --check src/*.ts
docker compose config --quiet
./scripts/smoke.sh
```

The smoke test starts Neo4j, applies constraints and indexes, imports all 83 nodes and 115 relationships from `seed/insight-track.json`, and verifies path, root-cause, blast-radius, and authorization behavior.

See `openapi.yaml` for the complete HTTP contract and `migrations/001_schema.cypher` for the Neo4j schema.
