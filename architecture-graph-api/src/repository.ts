import type { Neo4jClient } from "./neo4j.ts";
import type {
  ArchitectureGraph,
  ArchitectureGraphInput,
  ArchitectureNode,
  ArchitectureRelationship,
  ArchitectureSnapshot,
  ArchitectureSnapshotInput,
  EvidencePath,
  ExecutedGraphQuery,
  Metadata
} from "./types.ts";

interface RawGraphRow extends Record<string, unknown> {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
  node_count: number;
  relationship_count: number;
}

const NODE_PROJECTION = `{
  id: node.id,
  graph_id: node.graph_id,
  kind: node.kind,
  name: node.name,
  description: node.description,
  technology: node.technology,
  metadata_json: node.metadata_json
}`;

const PATH_PROJECTION = `
  [node IN nodes(path) | ${NODE_PROJECTION}] AS nodes,
  [edge IN relationships(path) | {
    id: edge.id,
    graph_id: edge.graph_id,
    type: type(edge),
    from: startNode(edge).id,
    to: endNode(edge).id,
    description: edge.description,
    metadata_json: edge.metadata_json
  }] AS relationships,
  length(path) AS hops`;

function numberValue(value: unknown): number {
  if (typeof value === "number") return value;
  if (typeof value === "string") return Number(value);
  if (value && typeof value === "object" && "low" in value) {
    return Number((value as { low: unknown }).low);
  }
  return 0;
}

function metadataValue(value: unknown): Metadata {
  if (typeof value !== "string" || !value) return {};
  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as Metadata : {};
  } catch {
    return {};
  }
}

function nodeValue(value: unknown): ArchitectureNode {
  const node = value as Record<string, unknown>;
  return {
    id: String(node.id || ""),
    graph_id: String(node.graph_id || ""),
    kind: String(node.kind || "") as ArchitectureNode["kind"],
    name: String(node.name || ""),
    description: String(node.description || ""),
    technology: String(node.technology || ""),
    metadata: metadataValue(node.metadata_json)
  };
}

function relationshipValue(value: unknown): ArchitectureRelationship {
  const edge = value as Record<string, unknown>;
  return {
    id: String(edge.id || ""),
    graph_id: String(edge.graph_id || ""),
    type: String(edge.type || "") as ArchitectureRelationship["type"],
    from: String(edge.from || ""),
    to: String(edge.to || ""),
    description: String(edge.description || ""),
    metadata: metadataValue(edge.metadata_json)
  };
}

function pathRows(rows: Array<Record<string, unknown>>): EvidencePath[] {
  return rows.map((row) => ({
    nodes: Array.isArray(row.nodes) ? row.nodes.map(nodeValue) : [],
    relationships: Array.isArray(row.relationships) ? row.relationships.map(relationshipValue) : [],
    hops: numberValue(row.hops)
  }));
}

function graphValue(row: RawGraphRow): ArchitectureGraph {
  return {
    id: row.id,
    name: row.name,
    description: row.description || "",
    created_at: row.created_at,
    updated_at: row.updated_at,
    node_count: numberValue(row.node_count),
    relationship_count: numberValue(row.relationship_count)
  };
}

export interface ArchitectureSummaryResult {
  node_count: number;
  relationship_count: number;
  nodes_by_kind: Array<{ kind: string; count: number }>;
  relationships_by_type: Array<{ type: string; count: number }>;
  cypher: string[];
  parameters: Record<string, unknown>;
}

export class ArchitectureRepository {
  private readonly client: Neo4jClient;

  constructor(client: Neo4jClient) {
    this.client = client;
  }

  async migrate(statements: string[]): Promise<void> {
    for (const statement of statements) {
      await this.client.query({ statement, accessMode: "WRITE" });
    }
  }

  async upsertGraph(id: string, input: ArchitectureGraphInput): Promise<ArchitectureGraph> {
    const statement = `
      MERGE (graph:ArchitectureGraph {id: $graphId})
      ON CREATE SET graph.created_at = datetime()
      SET graph.name = $name,
          graph.description = $description,
          graph.updated_at = datetime()
      WITH graph
      OPTIONAL MATCH (graph)-[:CONTAINS]->(node:ArchitectureNode)
      WITH graph, count(node) AS node_count
      OPTIONAL MATCH (source:ArchitectureNode)-[edge]->(target:ArchitectureNode)
      WHERE source.graph_id = graph.id AND target.graph_id = graph.id
      RETURN graph.id AS id,
             graph.name AS name,
             graph.description AS description,
             toString(graph.created_at) AS created_at,
             toString(graph.updated_at) AS updated_at,
             node_count,
             count(edge) AS relationship_count`;
    const rows = await this.client.query<RawGraphRow>({
      statement,
      parameters: { graphId: id, name: input.name, description: input.description || "" },
      accessMode: "WRITE"
    });
    return graphValue(rows[0]);
  }

  async listGraphs(): Promise<ArchitectureGraph[]> {
    const statement = `
      MATCH (graph:ArchitectureGraph)
      OPTIONAL MATCH (graph)-[:CONTAINS]->(node:ArchitectureNode)
      WITH graph, count(node) AS node_count
      OPTIONAL MATCH (source:ArchitectureNode)-[edge]->(target:ArchitectureNode)
      WHERE source.graph_id = graph.id AND target.graph_id = graph.id
      RETURN graph.id AS id,
             graph.name AS name,
             graph.description AS description,
             toString(graph.created_at) AS created_at,
             toString(graph.updated_at) AS updated_at,
             node_count,
             count(edge) AS relationship_count
      ORDER BY toLower(graph.name), graph.id`;
    const rows = await this.client.query<RawGraphRow>({ statement, accessMode: "READ" });
    return rows.map(graphValue);
  }

  async getGraph(id: string): Promise<ArchitectureGraph | null> {
    const statement = `
      MATCH (graph:ArchitectureGraph {id: $graphId})
      OPTIONAL MATCH (graph)-[:CONTAINS]->(node:ArchitectureNode)
      WITH graph, count(node) AS node_count
      OPTIONAL MATCH (source:ArchitectureNode)-[edge]->(target:ArchitectureNode)
      WHERE source.graph_id = graph.id AND target.graph_id = graph.id
      RETURN graph.id AS id,
             graph.name AS name,
             graph.description AS description,
             toString(graph.created_at) AS created_at,
             toString(graph.updated_at) AS updated_at,
             node_count,
             count(edge) AS relationship_count`;
    const rows = await this.client.query<RawGraphRow>({
      statement,
      parameters: { graphId: id },
      accessMode: "READ"
    });
    return rows[0] ? graphValue(rows[0]) : null;
  }

  async replaceSnapshot(graphId: string, snapshot: ArchitectureSnapshotInput): Promise<ArchitectureSnapshot> {
    const nodes = snapshot.nodes.map((node) => ({
      graph_id: graphId,
      id: node.id,
      kind: node.kind,
      name: node.name,
      description: node.description || "",
      technology: node.technology || "",
      metadata_json: JSON.stringify(node.metadata || {})
    }));
    const relationships = snapshot.relationships.map((edge) => ({
      graph_id: graphId,
      id: edge.id,
      type: edge.type,
      from: edge.from,
      to: edge.to,
      description: edge.description || "",
      metadata_json: JSON.stringify(edge.metadata || {})
    }));
    const statement = `
      MATCH (graph:ArchitectureGraph {id: $graphId})
      SET graph.updated_at = datetime()
      CALL (graph) {
        OPTIONAL MATCH (graph)-[:CONTAINS]->(old:ArchitectureNode)
        DETACH DELETE old
      }
      CALL (graph) {
        UNWIND $nodes AS item
        CREATE (node:ArchitectureNode)
        SET node = item
        CREATE (graph)-[:CONTAINS]->(node)
      }
      CALL () {
        UNWIND $relationships AS item
        MATCH (source:ArchitectureNode {graph_id: $graphId, id: item.from})
        MATCH (target:ArchitectureNode {graph_id: $graphId, id: item.to})
        CREATE (source)-[edge:$(item.type)]->(target)
        SET edge.id = item.id,
            edge.graph_id = item.graph_id,
            edge.description = item.description,
            edge.metadata_json = item.metadata_json
      }
      RETURN size($nodes) AS node_count, size($relationships) AS relationship_count`;
    const rows = await this.client.query({
      statement,
      parameters: { graphId, nodes, relationships },
      accessMode: "WRITE"
    });
    if (!rows[0]) throw new Error("architecture graph does not exist");
    return this.getSnapshot(graphId);
  }

  async listNodes(graphId: string, search = "", kind = "", limit = 1_000): Promise<ArchitectureNode[]> {
    const statement = `
      MATCH (node:ArchitectureNode {graph_id: $graphId})
      WHERE ($kind = '' OR node.kind = $kind)
        AND ($search = '' OR toLower(node.name) CONTAINS toLower($search)
          OR toLower(node.id) CONTAINS toLower($search))
      RETURN ${NODE_PROJECTION} AS node
      ORDER BY toLower(node.name), node.id
      LIMIT $limit`;
    const rows = await this.client.query<{ node: unknown }>({
      statement,
      parameters: { graphId, search, kind, limit },
      accessMode: "READ"
    });
    return rows.map((row) => nodeValue(row.node));
  }

  async getNode(graphId: string, nodeId: string): Promise<ArchitectureNode | null> {
    const statement = `
      MATCH (node:ArchitectureNode {graph_id: $graphId, id: $nodeId})
      RETURN ${NODE_PROJECTION} AS node`;
    const rows = await this.client.query<{ node: unknown }>({
      statement,
      parameters: { graphId, nodeId },
      accessMode: "READ"
    });
    return rows[0] ? nodeValue(rows[0].node) : null;
  }

  async getSnapshot(graphId: string): Promise<ArchitectureSnapshot> {
    const nodes = await this.listNodes(graphId, "", "", 5_000);
    const statement = `
      MATCH (source:ArchitectureNode {graph_id: $graphId})-[edge]->(target:ArchitectureNode {graph_id: $graphId})
      RETURN {
        id: edge.id,
        graph_id: edge.graph_id,
        type: type(edge),
        from: source.id,
        to: target.id,
        description: edge.description,
        metadata_json: edge.metadata_json
      } AS relationship
      ORDER BY type(edge), source.id, target.id, edge.id`;
    const rows = await this.client.query<{ relationship: unknown }>({
      statement,
      parameters: { graphId },
      accessMode: "READ"
    });
    return { graph_id: graphId, nodes, relationships: rows.map((row) => relationshipValue(row.relationship)) };
  }

  async dependencies(graphId: string, nodeId: string, maxDepth: number, limit: number): Promise<ExecutedGraphQuery> {
    const statement = this.directedPathStatement("dependencies", maxDepth);
    return this.executePathQuery("dependencies", statement, { graphId, nodeId, limit });
  }

  async dependents(graphId: string, nodeId: string, maxDepth: number, limit: number): Promise<ExecutedGraphQuery> {
    const statement = this.directedPathStatement("dependents", maxDepth);
    return this.executePathQuery("dependents", statement, { graphId, nodeId, limit });
  }

  async shortestPath(graphId: string, sourceId: string, targetId: string, maxDepth: number): Promise<ExecutedGraphQuery> {
    const statement = `
      MATCH (source:ArchitectureNode {graph_id: $graphId, id: $sourceId})
      MATCH (target:ArchitectureNode {graph_id: $graphId, id: $targetId})
      MATCH path = shortestPath((source)-[*1..${maxDepth}]-(target))
      WHERE all(node IN nodes(path) WHERE node.graph_id = $graphId)
      RETURN ${PATH_PROJECTION}`;
    return this.executePathQuery("shortest_path", statement, { graphId, sourceId, targetId });
  }

  async architectureSummary(graphId: string): Promise<ArchitectureSummaryResult> {
    const nodeStatement = `
      MATCH (node:ArchitectureNode {graph_id: $graphId})
      RETURN node.kind AS kind, count(node) AS count
      ORDER BY kind`;
    const relationshipStatement = `
      MATCH (source:ArchitectureNode {graph_id: $graphId})-[edge]->(target:ArchitectureNode {graph_id: $graphId})
      RETURN type(edge) AS type, count(edge) AS count
      ORDER BY type`;
    const parameters = { graphId };
    const [nodeRows, relationshipRows] = await Promise.all([
      this.client.query<{ kind: string; count: number }>({ statement: nodeStatement, parameters, accessMode: "READ" }),
      this.client.query<{ type: string; count: number }>({ statement: relationshipStatement, parameters, accessMode: "READ" })
    ]);
    const nodesByKind = nodeRows.map((row) => ({ kind: row.kind, count: numberValue(row.count) }));
    const relationshipsByType = relationshipRows.map((row) => ({ type: row.type, count: numberValue(row.count) }));
    return {
      node_count: nodesByKind.reduce((total, row) => total + row.count, 0),
      relationship_count: relationshipsByType.reduce((total, row) => total + row.count, 0),
      nodes_by_kind: nodesByKind,
      relationships_by_type: relationshipsByType,
      cypher: [nodeStatement.replace(/\s+/g, " ").trim(), relationshipStatement.replace(/\s+/g, " ").trim()],
      parameters
    };
  }

  private directedPathStatement(direction: "dependencies" | "dependents", maxDepth: number): string {
    const pattern = direction === "dependencies"
      ? `(focus)-[*1..${maxDepth}]->(related:ArchitectureNode)`
      : `(related:ArchitectureNode)-[*1..${maxDepth}]->(focus)`;
    return `
      MATCH (focus:ArchitectureNode {graph_id: $graphId, id: $nodeId})
      MATCH path = ${pattern}
      WHERE all(node IN nodes(path) WHERE node.graph_id = $graphId)
        AND all(node IN nodes(path) WHERE single(candidate IN nodes(path) WHERE candidate = node))
      RETURN ${PATH_PROJECTION}
      ORDER BY hops, toLower(related.name), related.id
      LIMIT $limit`;
  }

  private async executePathQuery(
    template: string,
    statement: string,
    parameters: Record<string, unknown>
  ): Promise<ExecutedGraphQuery> {
    const rows = await this.client.query<Record<string, unknown>>({
      statement,
      parameters,
      accessMode: "READ"
    });
    return {
      template,
      cypher: statement.replace(/\s+/g, " ").trim(),
      parameters,
      paths: pathRows(rows)
    };
  }
}
