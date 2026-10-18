# Insight Track Frontend

React + TypeScript + Tailwind CSS dashboard UI for Insight Track, consuming
`postgres-project-tenancy`, `clickhouse-telemetry-api` and
`architecture-graph-api` (see [../API_REFERENCE.txt](../API_REFERENCE.txt)).

## Develop

```sh
npm install
VITE_KEYCLOAK_URL=http://localhost:8081 npm run dev
```

The app uses the `insight-track-frontend` Keycloak client with Authorization
Code and PKCE. In the combined Compose deployment it automatically uses
`https://<APP_DOMAIN>/auth`; no frontend rebuild is required when the domain
changes.
