import Keycloak, { type KeycloakTokenParsed } from "keycloak-js";

const keycloakUrl =
  import.meta.env.VITE_KEYCLOAK_URL ?? `${window.location.origin}/auth`;
const realm = import.meta.env.VITE_KEYCLOAK_REALM ?? "project-tenancy";
const clientId =
  import.meta.env.VITE_KEYCLOAK_CLIENT_ID ?? "insight-track-frontend";

export const keycloak = new Keycloak({
  url: keycloakUrl,
  realm,
  clientId,
});

let initialization: Promise<boolean> | undefined;

export function initializeKeycloak() {
  initialization ??= keycloak.init({
    onLoad: "login-required",
    pkceMethod: "S256",
    checkLoginIframe: false,
  });

  return initialization;
}

export async function getAccessToken() {
  if (!keycloak.authenticated) {
    await keycloak.login({ redirectUri: window.location.href });
    throw new Error("Authentication is required.");
  }

  try {
    await keycloak.updateToken(30);
  } catch {
    await keycloak.login({ redirectUri: window.location.href });
    throw new Error("The session could not be refreshed.");
  }

  if (!keycloak.token) {
    throw new Error("Keycloak did not provide an access token.");
  }

  return keycloak.token;
}

export interface AuthenticatedUser {
  subject: string;
  name: string;
  email?: string;
  initials: string;
}

interface InsightTrackToken extends KeycloakTokenParsed {
  email?: string;
  name?: string;
  preferred_username?: string;
}

export function currentUser(): AuthenticatedUser | null {
  const token = keycloak.tokenParsed as InsightTrackToken | undefined;
  if (!keycloak.authenticated || !token?.sub) {
    return null;
  }

  const name = token.name?.trim() || token.preferred_username?.trim() || "User";
  const initials = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("") || "U";

  return {
    subject: token.sub,
    name,
    email: token.email,
    initials,
  };
}
