export interface CypherQuery {
  statement: string;
  parameters?: Record<string, unknown>;
  accessMode?: "READ" | "WRITE";
}

interface Neo4jResponse {
  data?: {
    fields?: string[];
    values?: unknown[][];
  };
  errors?: Array<{ code?: string; message?: string }>;
}

export class Neo4jQueryError extends Error {
  readonly code?: string;

  constructor(message: string, code?: string) {
    super(message);
    this.code = code;
  }
}

export class Neo4jClient {
  private readonly endpoint: string;
  private readonly authorization: string;
  private readonly timeoutMs: number;

  constructor(
    baseUrl: string,
    database: string,
    username: string,
    password: string,
    timeoutMs: number
  ) {
    this.endpoint = `${baseUrl.replace(/\/$/, "")}/db/${encodeURIComponent(database)}/query/v2`;
    this.authorization = `Basic ${Buffer.from(`${username}:${password}`).toString("base64")}`;
    this.timeoutMs = timeoutMs;
  }

  async query<T extends Record<string, unknown>>(query: CypherQuery): Promise<T[]> {
    const response = await fetch(this.endpoint, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Access-Mode": query.accessMode || "WRITE",
        Authorization: this.authorization,
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        statement: query.statement.replace(/\s+/g, " ").trim(),
        parameters: query.parameters || {}
      }),
      signal: AbortSignal.timeout(this.timeoutMs)
    });

    const raw = await response.text();
    let body: Neo4jResponse;
    try {
      body = raw ? JSON.parse(raw) as Neo4jResponse : {};
    } catch {
      throw new Neo4jQueryError(`Neo4j returned invalid JSON with status ${response.status}`);
    }

    if (!response.ok || (body.errors && body.errors.length > 0)) {
      const first = body.errors?.[0];
      throw new Neo4jQueryError(first?.message || `Neo4j returned status ${response.status}`, first?.code);
    }

    const fields = body.data?.fields || [];
    return (body.data?.values || []).map((values) => Object.fromEntries(
      fields.map((field, index) => [field, values[index]])
    ) as T);
  }

  async ping(): Promise<void> {
    await this.query({ statement: "RETURN 1 AS value", accessMode: "READ" });
  }
}
