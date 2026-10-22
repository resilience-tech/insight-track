import { getAccessToken } from "../auth/keycloak";

const controlApiBase = "/api/control";

export async function controlApiFetch(
  path: string,
  init: RequestInit = {},
) {
  const token = await getAccessToken();
  const headers = new Headers(init.headers);
  headers.set("Authorization", `Bearer ${token}`);

  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return fetch(`${controlApiBase}/${path.replace(/^\/+/, "")}`, {
    ...init,
    headers,
  });
}
