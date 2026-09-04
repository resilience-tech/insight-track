export const NODE_KINDS = [
  "system",
  "frontend",
  "page",
  "component",
  "action",
  "service",
  "endpoint",
  "handler",
  "function",
  "middleware",
  "repository",
  "cache",
  "database",
  "table",
  "column",
  "constraint",
  "external_service",
  "identity_provider",
  "queue",
  "topic",
  "job",
  "library",
  "runtime",
  "infrastructure",
  "environment",
  "config",
  "event"
] as const;

export type NodeKind = (typeof NODE_KINDS)[number];

export const RELATIONSHIP_TYPES = [
  "HAS_SERVICE",
  "HAS_FRONTEND",
  "CONTAINS",
  "EXPOSES",
  "HANDLED_BY",
  "TRIGGERS",
  "CALLS",
  "USES_MIDDLEWARE",
  "USES_DATABASE",
  "USES_CACHE",
  "USES_IDENTITY_PROVIDER",
  "USES_EXTERNAL_SERVICE",
  "READS_FROM",
  "WRITES_TO",
  "HAS_COLUMN",
  "HAS_CONSTRAINT",
  "PUBLISHES_TO",
  "CONSUMES_FROM",
  "DEPENDS_ON",
  "DEPLOYED_TO",
  "EMITS"
] as const;

export type RelationshipType = (typeof RELATIONSHIP_TYPES)[number];
export type MetadataValue = string | number | boolean;
export type Metadata = Record<string, MetadataValue>;

export interface ArchitectureGraphInput {
  name: string;
  description?: string;
}

export interface ArchitectureGraph extends ArchitectureGraphInput {
  id: string;
  node_count: number;
  relationship_count: number;
  created_at: string;
  updated_at: string;
}

export interface ArchitectureNodeInput {
  id: string;
  kind: NodeKind;
  name: string;
  description?: string;
  technology?: string;
  metadata?: Metadata;
}

export interface ArchitectureNode extends ArchitectureNodeInput {
  graph_id: string;
  description: string;
  technology: string;
  metadata: Metadata;
}

export interface ArchitectureRelationshipInput {
  id: string;
  type: RelationshipType;
  from: string;
  to: string;
  description?: string;
  metadata?: Metadata;
}

export interface ArchitectureRelationship extends ArchitectureRelationshipInput {
  graph_id: string;
  description: string;
  metadata: Metadata;
}

export interface ArchitectureSnapshotInput {
  nodes: ArchitectureNodeInput[];
  relationships: ArchitectureRelationshipInput[];
}

export interface ArchitectureSnapshot {
  graph_id: string;
  nodes: ArchitectureNode[];
  relationships: ArchitectureRelationship[];
}

export interface EvidencePath {
  nodes: ArchitectureNode[];
  relationships: ArchitectureRelationship[];
  hops: number;
}

export type QuestionIntent =
  | "summary"
  | "dependencies"
  | "dependents"
  | "shortest_path"
  | "root_causes"
  | "blast_radius"
  | "data_access";

export interface QuestionPlan {
  intent: QuestionIntent;
  source?: ArchitectureNode;
  target?: ArchitectureNode;
  max_depth: number;
  limit: number;
}

export interface QuestionInput {
  question: string;
  focus_node_id?: string;
  target_node_id?: string;
  max_depth?: number;
  limit?: number;
}

export interface ExecutedGraphQuery {
  template: string;
  cypher: string;
  parameters: Record<string, unknown>;
  paths: EvidencePath[];
}

export interface QuestionAnswer {
  question: string;
  intent: QuestionIntent;
  answer: string;
  matched_entities: ArchitectureNode[];
  evidence: {
    paths: EvidencePath[];
    summary?: Record<string, unknown>;
  };
  graph_query: {
    template: string;
    cypher: string | string[];
    parameters: Record<string, unknown>;
  };
}
