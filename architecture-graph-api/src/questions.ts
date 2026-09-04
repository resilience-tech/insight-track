import type { ArchitectureRepository } from "./repository.ts";
import type {
  ArchitectureNode,
  EvidencePath,
  QuestionAnswer,
  QuestionInput,
  QuestionIntent,
  QuestionPlan
} from "./types.ts";

export class QuestionPlanningError extends Error {
  readonly suggestions: ArchitectureNode[];

  constructor(message: string, suggestions: ArchitectureNode[] = []) {
    super(message);
    this.suggestions = suggestions;
  }
}

const dataKinds = new Set(["database", "table", "column", "constraint", "cache"]);
const riskKinds = new Set([
  "service",
  "database",
  "table",
  "cache",
  "repository",
  "external_service",
  "identity_provider",
  "queue",
  "topic",
  "infrastructure"
]);

function normalized(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9/{}]+/g, " ").replace(/\s+/g, " ").trim();
}

function uniqueNodes(nodes: ArchitectureNode[]): ArchitectureNode[] {
  return [...new Map(nodes.map((node) => [node.id, node])).values()];
}

export function resolveMentionedNodes(question: string, nodes: ArchitectureNode[]): ArchitectureNode[] {
  const source = normalized(question);
  const direct = nodes.map((node) => {
    const name = normalized(node.name);
    const kind = normalized(node.kind);
    const aliases = [name, normalized(node.id)];
    if (name.endsWith(` ${kind}`)) aliases.push(name.slice(0, -(kind.length + 1)));
    const positions = aliases
      .filter((value) => value.length >= 3 && source.includes(value))
      .map((value) => source.indexOf(value));
    return { node, position: positions.length > 0 ? Math.min(...positions) : -1 };
  }).filter((match) => match.position >= 0).sort((left, right) =>
    left.position - right.position
      || normalized(right.node.name).length - normalized(left.node.name).length
      || left.node.name.localeCompare(right.node.name)
  ).map((match) => match.node);

  const withoutContainedNames = direct.filter((node, index) => !direct.some((other, otherIndex) =>
    otherIndex < index && normalized(other.name).includes(normalized(node.name))
  ));
  return uniqueNodes(withoutContainedNames);
}

function inferIntent(question: string, source: ArchitectureNode | undefined, target: ArchitectureNode | undefined): QuestionIntent {
  const value = normalized(question);
  if (/\b(summary|overview|inventory|counts?|complete architecture|what is in)\b/.test(value) && !source) {
    return "summary";
  }
  if (target && /\b(path|route|flow|trace|connection|between|from)\b/.test(value)) {
    return "shortest_path";
  }
  if (/\b(root cause|root causes|cause|causes|causing|why)\b/.test(value) && /\b(error|errors|fail|fails|failure|broken|down)\b/.test(value)) {
    return "root_causes";
  }
  if (/\b(blast radius|impact|affected|breaks|stop working|fails what)\b/.test(value)) {
    return "blast_radius";
  }
  if (/\b(what|which|who)\s+(else\s+)?(depends on|uses|calls)\b/.test(value) || /\b(callers?|upstream|consumers?)\b/.test(value)) {
    return "dependents";
  }
  if (/\b(database|databases|table|tables|column|columns|cache|data|read|reads|write|writes)\b/.test(value)) {
    return "data_access";
  }
  if (source) return "dependencies";
  return "summary";
}

export function planQuestion(input: QuestionInput, nodes: ArchitectureNode[]): QuestionPlan {
  const byId = new Map(nodes.map((node) => [node.id, node]));
  const mentioned = resolveMentionedNodes(input.question, nodes);
  let source = input.focus_node_id ? byId.get(input.focus_node_id) : mentioned[0];
  let target = input.target_node_id ? byId.get(input.target_node_id) : mentioned.find((node) => node.id !== source?.id);

  if (input.focus_node_id && !source) {
    throw new QuestionPlanningError(`focus_node_id '${input.focus_node_id}' was not found`, nodes.slice(0, 10));
  }
  if (input.target_node_id && !target) {
    throw new QuestionPlanningError(`target_node_id '${input.target_node_id}' was not found`, nodes.slice(0, 10));
  }

  const question = normalized(input.question);
  if (!source && /\b(browser|frontend|web app|web page|ui)\b/.test(question)) {
    source = nodes.find((node) => node.kind === "frontend") || nodes.find((node) => node.kind === "page");
  }

  const intent = input.target_node_id && target
    ? "shortest_path"
    : inferIntent(input.question, source, target);
  if (intent === "shortest_path" && (!source || !target)) {
    throw new QuestionPlanningError("A path question needs two named nodes or focus_node_id and target_node_id", mentioned);
  }
  if (intent !== "summary" && !source) {
    throw new QuestionPlanningError(
      "I could not identify the architecture node in the question; provide focus_node_id or use an exact node name",
      nodes.slice(0, 10)
    );
  }

  return {
    intent,
    source,
    target,
    max_depth: input.max_depth || 8,
    limit: input.limit || 50
  };
}

function pathEndpoint(path: EvidencePath, intent: QuestionIntent): ArchitectureNode | undefined {
  if (intent === "dependents" || intent === "blast_radius") return path.nodes[0];
  return path.nodes[path.nodes.length - 1];
}

function filteredPaths(intent: QuestionIntent, source: ArchitectureNode, paths: EvidencePath[], question: string): EvidencePath[] {
  if (intent === "root_causes") {
    return paths.filter((path) => {
      const endpoint = path.nodes[path.nodes.length - 1];
      return endpoint && endpoint.id !== source.id && riskKinds.has(endpoint.kind);
    });
  }
  if (intent === "data_access") {
    const value = normalized(question);
    const requestedKind = /\bendpoints?\b/.test(value) ? "endpoint"
      : /\bservices?\b/.test(value) ? "service"
        : /\brepositories?\b/.test(value) ? "repository"
          : /\btables?\b/.test(value) ? "table"
            : /\bdatabases?\b/.test(value) ? "database"
              : /\bcaches?\b/.test(value) ? "cache"
                : undefined;
    return paths.filter((path) => {
      const endpoint = path.nodes[path.nodes.length - 1];
      const related = dataKinds.has(source.kind) ? path.nodes[0] : endpoint;
      if (!related) return false;
      if (requestedKind && related.kind !== requestedKind) return false;
      if (/\b(write|writes|writing|written)\b/.test(value)
        && !path.relationships.some((edge) => edge.type === "WRITES_TO")) return false;
      if (/\b(read|reads|reading)\b/.test(value)
        && !path.relationships.some((edge) => edge.type === "READS_FROM")) return false;
      return dataKinds.has(source.kind) || dataKinds.has(related.kind);
    });
  }
  return paths;
}

function answerFor(intent: QuestionIntent, source: ArchitectureNode, target: ArchitectureNode | undefined, paths: EvidencePath[]): string {
  if (intent === "shortest_path") {
    if (paths.length === 0) return `No path was found between ${source.name} and ${target?.name || "the target"}.`;
    return paths[0].nodes.map((node) => node.name).join(" → ");
  }

  const endpoints = uniqueNodes(paths.map((path) => pathEndpoint(path, intent)).filter(Boolean) as ArchitectureNode[]);
  if (endpoints.length === 0) {
    const descriptions: Record<QuestionIntent, string> = {
      summary: "components",
      dependencies: "downstream dependencies",
      dependents: "upstream dependents",
      shortest_path: "paths",
      root_causes: "downstream failure points",
      blast_radius: "affected components",
      data_access: dataKinds.has(source.kind) ? "components using this data node" : "data dependencies"
    };
    return `No ${descriptions[intent]} were found for ${source.name}.`;
  }

  const names = endpoints.slice(0, 12).map((node) => node.name).join(", ");
  const remainder = endpoints.length > 12 ? `, and ${endpoints.length - 12} more` : "";
  const prefixes: Record<Exclude<QuestionIntent, "summary" | "shortest_path">, string> = {
    dependencies: `${source.name} depends on`,
    dependents: `Components depending on ${source.name}`,
    root_causes: `Potential downstream causes for ${source.name}`,
    blast_radius: `Components potentially affected by ${source.name}`,
    data_access: dataKinds.has(source.kind) ? `Components accessing ${source.name}` : `Data used by ${source.name}`
  };
  return `${prefixes[intent as keyof typeof prefixes]}: ${names}${remainder}.`;
}

export class ArchitectureQuestionService {
  private readonly repository: ArchitectureRepository;

  constructor(repository: ArchitectureRepository) {
    this.repository = repository;
  }

  async ask(graphId: string, input: QuestionInput): Promise<QuestionAnswer> {
    const graph = await this.repository.getGraph(graphId);
    if (!graph) throw new QuestionPlanningError("Architecture graph was not found");
    const nodes = await this.repository.listNodes(graphId, "", "", 5_000);
    const plan = planQuestion(input, nodes);

    if (plan.intent === "summary") {
      const summary = await this.repository.architectureSummary(graphId);
      const kinds = summary.nodes_by_kind.map((row) => `${row.count} ${row.kind}`).join(", ");
      return {
        question: input.question,
        intent: plan.intent,
        answer: `${graph.name} contains ${summary.node_count} nodes and ${summary.relationship_count} relationships: ${kinds || "no nodes"}.`,
        matched_entities: [],
        evidence: { paths: [], summary: { ...summary } },
        graph_query: {
          template: "architecture_summary",
          cypher: summary.cypher,
          parameters: summary.parameters
        }
      };
    }

    const source = plan.source as ArchitectureNode;
    let result;
    switch (plan.intent) {
      case "shortest_path":
        result = await this.repository.shortestPath(graphId, source.id, (plan.target as ArchitectureNode).id, plan.max_depth);
        break;
      case "dependents":
      case "blast_radius":
        result = await this.repository.dependents(graphId, source.id, plan.max_depth, plan.limit);
        break;
      case "data_access":
        result = dataKinds.has(source.kind)
          ? await this.repository.dependents(graphId, source.id, plan.max_depth, plan.limit)
          : await this.repository.dependencies(graphId, source.id, plan.max_depth, plan.limit);
        break;
      case "dependencies":
      case "root_causes":
        result = await this.repository.dependencies(graphId, source.id, plan.max_depth, plan.limit);
        break;
    }

    const paths = filteredPaths(plan.intent, source, result.paths, input.question);
    return {
      question: input.question,
      intent: plan.intent,
      answer: answerFor(plan.intent, source, plan.target, paths),
      matched_entities: uniqueNodes([source, ...(plan.target ? [plan.target] : [])]),
      evidence: { paths },
      graph_query: {
        template: result.template,
        cypher: result.cypher,
        parameters: result.parameters
      }
    };
  }
}
