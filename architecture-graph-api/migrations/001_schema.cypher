CREATE CONSTRAINT architecture_graph_id IF NOT EXISTS
FOR (graph:ArchitectureGraph)
REQUIRE graph.id IS UNIQUE;

CREATE CONSTRAINT architecture_node_key IF NOT EXISTS
FOR (node:ArchitectureNode)
REQUIRE (node.graph_id, node.id) IS UNIQUE;

CREATE INDEX architecture_node_name IF NOT EXISTS
FOR (node:ArchitectureNode)
ON (node.graph_id, node.name);

CREATE INDEX architecture_node_kind IF NOT EXISTS
FOR (node:ArchitectureNode)
ON (node.graph_id, node.kind);
