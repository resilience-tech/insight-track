# Combined public deployment

HAProxy publishes public ports 80 and 443. Port 80 redirects to HTTPS, and
HAProxy terminates TLS on port 443 before routing requests to the internal
services.

## DNS

Create a Route 53 `A` record:

```text
origin-insight.blendology.com -> 51.38.186.83
```

DNS records do not select ports. The host and provider firewalls must allow
inbound TCP traffic on ports 80 and 443.

## TLS certificate

HAProxy loads the wildcard certificate and its matching private key directly
from these host paths:

```text
/obs/cert/server3_latest.pem
/obs/cert/server3_latest_key.pem
```

The certificate file must contain the leaf certificate followed by its complete
chain. The key is mounted read-only as the sidecar key for that certificate.
`HAPROXY_UID` and `HAPROXY_GID` must match the host owner of the private key.

## Start

```sh
cp .env.example .env
# Replace every placeholder secret in .env.
# Keep APP_DATABASE_PASSWORD URL-safe and alphanumeric.
docker compose -f compose.insight-track.yaml config --quiet
docker compose -f compose.insight-track.yaml up --build --detach
docker compose -f compose.insight-track.yaml ps
```

`APP_DOMAIN` is applied when the Keycloak realm is first imported. To change
the domain later, update the frontend client in Keycloak or recreate the
Keycloak database volume before starting the replacement deployment.

The public routes are:

```text
https://origin-insight.blendology.com/                          React frontend
https://origin-insight.blendology.com/auth/                     Keycloak
https://origin-insight.blendology.com/auth/admin/               Keycloak admin
https://origin-insight.blendology.com/api/control/health/ready  Tenancy readiness
https://origin-insight.blendology.com/api/control/v1/           Tenancy API
https://origin-insight.blendology.com/api/telemetry/health/ready Telemetry readiness
https://origin-insight.blendology.com/api/telemetry/v1/         Telemetry API
```

HAProxy itself can be checked locally:

```sh
curl --fail https://origin-insight.blendology.com/_proxy/health
```

The telemetry API still uses its internal bearer token and does not authorize
end-user project membership. Do not put `TELEMETRY_API_TOKEN` in frontend code.
For a production browser flow, proxy telemetry through a service that validates
the user's OIDC token and project membership.
