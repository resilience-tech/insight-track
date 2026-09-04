import {
  NODE_KINDS,
  RELATIONSHIP_TYPES,
  type ArchitectureGraphInput,
  type ArchitectureSnapshotInput,
  type Metadata,
  type QuestionInput
} from "./types.ts";

export class ValidationError extends Error {}

const identifierPattern = /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$/;
const nodeKinds = new Set<string>(NODE_KINDS);
const relationshipTypes = new Set<string>(RELATIONSHIP_TYPES);

function object(value: unknown, field: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new ValidationError(`${field} must be an object`);
  }
  return value as Record<string, unknown>;
}

function allowedKeys(value: Record<string, unknown>, field: string, keys: string[]): void {
  const allowed = new Set(keys);
  const unknown = Object.keys(value).filter((key) => !allowed.has(key));
  if (unknown.length > 0) {
    throw new ValidationError(`${field} contains unknown field '${unknown[0]}'`);
  }
}

function text(value: unknown, field: string, minimum: number, maximum: number): string {
  if (typeof value !== "string") throw new ValidationError(`${field} must be a string`);
  const normalized = value.trim();
  if (normalized.length < minimum || normalized.length > maximum) {
    throw new ValidationError(`${field} must contain ${minimum}-${maximum} characters`);
  }
  return normalized;
}

function optionalText(value: unknown, field: string, maximum: number): string | undefined {
  if (value === undefined) return undefined;
  return text(value, field, 0, maximum);
}

function identifier(value: unknown, field: string): string {
  const normalized = text(value, field, 1, 128);
  if (!identifierPattern.test(normalized)) {
    throw new ValidationError(`${field} must be a portable identifier`);
  }
  return normalized;
}

function metadata(value: unknown, field: string): Metadata | undefined {
  if (value === undefined) return undefined;
  const source = object(value, field);
  const entries = Object.entries(source);
  if (entries.length > 64) throw new ValidationError(`${field} supports at most 64 entries`);
  const result: Metadata = {};
  for (const [key, item] of entries) {
    if (!identifierPattern.test(key) || key.length > 64) {
      throw new ValidationError(`${field} contains an invalid key`);
    }
    if (!["string", "number", "boolean"].includes(typeof item)) {
      throw new ValidationError(`${field}.${key} must be a string, number, or boolean`);
    }
    if (typeof item === "string" && item.length > 2_000) {
      throw new ValidationError(`${field}.${key} is too long`);
    }
    if (typeof item === "number" && !Number.isFinite(item)) {
      throw new ValidationError(`${field}.${key} must be finite`);
    }
    result[key] = item as string | number | boolean;
  }
  return result;
}

export function validateGraphId(value: string): string {
  return identifier(value, "graphId");
}

export function validateGraphInput(value: unknown): ArchitectureGraphInput {
  const source = object(value, "body");
  allowedKeys(source, "body", ["name", "description"]);
  return {
    name: text(source.name, "name", 1, 160),
    description: optionalText(source.description, "description", 2_000)
  };
}

export function validateSnapshot(value: unknown): ArchitectureSnapshotInput {
  const source = object(value, "body");
  allowedKeys(source, "body", ["nodes", "relationships"]);
  if (!Array.isArray(source.nodes) || source.nodes.length > 5_000) {
    throw new ValidationError("nodes must be an array containing at most 5000 items");
  }
  if (!Array.isArray(source.relationships) || source.relationships.length > 10_000) {
    throw new ValidationError("relationships must be an array containing at most 10000 items");
  }

  const nodeIds = new Set<string>();
  const nodes = source.nodes.map((item, index) => {
    const node = object(item, `nodes[${index}]`);
    allowedKeys(node, `nodes[${index}]`, ["id", "kind", "name", "description", "technology", "metadata"]);
    const id = identifier(node.id, `nodes[${index}].id`);
    if (nodeIds.has(id)) throw new ValidationError(`nodes contains duplicate id '${id}'`);
    nodeIds.add(id);
    const kind = text(node.kind, `nodes[${index}].kind`, 1, 64);
    if (!nodeKinds.has(kind)) throw new ValidationError(`nodes[${index}].kind is unsupported`);
    return {
      id,
      kind: kind as ArchitectureSnapshotInput["nodes"][number]["kind"],
      name: text(node.name, `nodes[${index}].name`, 1, 240),
      description: optionalText(node.description, `nodes[${index}].description`, 2_000),
      technology: optionalText(node.technology, `nodes[${index}].technology`, 160),
      metadata: metadata(node.metadata, `nodes[${index}].metadata`)
    };
  });

  const relationshipIds = new Set<string>();
  const relationships = source.relationships.map((item, index) => {
    const edge = object(item, `relationships[${index}]`);
    allowedKeys(edge, `relationships[${index}]`, ["id", "type", "from", "to", "description", "metadata"]);
    const id = identifier(edge.id, `relationships[${index}].id`);
    if (relationshipIds.has(id)) throw new ValidationError(`relationships contains duplicate id '${id}'`);
    relationshipIds.add(id);
    const type = text(edge.type, `relationships[${index}].type`, 1, 64);
    if (!relationshipTypes.has(type)) throw new ValidationError(`relationships[${index}].type is unsupported`);
    const from = identifier(edge.from, `relationships[${index}].from`);
    const to = identifier(edge.to, `relationships[${index}].to`);
    if (!nodeIds.has(from) || !nodeIds.has(to)) {
      throw new ValidationError(`relationships[${index}] refers to a missing node`);
    }
    if (from === to) throw new ValidationError(`relationships[${index}] cannot connect a node to itself`);
    return {
      id,
      type: type as ArchitectureSnapshotInput["relationships"][number]["type"],
      from,
      to,
      description: optionalText(edge.description, `relationships[${index}].description`, 2_000),
      metadata: metadata(edge.metadata, `relationships[${index}].metadata`)
    };
  });

  return { nodes, relationships };
}

export function validateQuestion(value: unknown): QuestionInput {
  const source = object(value, "body");
  allowedKeys(source, "body", ["question", "focus_node_id", "target_node_id", "max_depth", "limit"]);
  const integer = (field: "max_depth" | "limit", fallback: number, maximum: number): number => {
    const value = source[field];
    if (value === undefined) return fallback;
    if (!Number.isSafeInteger(value) || (value as number) < 1 || (value as number) > maximum) {
      throw new ValidationError(`${field} must be between 1 and ${maximum}`);
    }
    return value as number;
  };
  return {
    question: text(source.question, "question", 3, 1_000),
    focus_node_id: source.focus_node_id === undefined ? undefined : identifier(source.focus_node_id, "focus_node_id"),
    target_node_id: source.target_node_id === undefined ? undefined : identifier(source.target_node_id, "target_node_id"),
    max_depth: integer("max_depth", 8, 15),
    limit: integer("limit", 50, 200)
  };
}
