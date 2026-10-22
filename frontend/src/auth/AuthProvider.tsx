import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  currentUser,
  getAccessToken,
  initializeKeycloak,
  keycloak,
  type AuthenticatedUser,
} from "./keycloak";

interface AuthContextValue {
  user: AuthenticatedUser;
  getAccessToken: () => Promise<string>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthenticationScreen({ message }: { message: string }) {
  return (
    <div className="flex min-h-dvh items-center justify-center bg-slate-50 px-6">
      <div className="w-full max-w-sm rounded-lg border border-slate-200 bg-white p-6 text-center shadow-sm">
        <div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-slate-200 border-t-slate-900" />
        <h1 className="text-base font-semibold text-slate-900">Insight Track</h1>
        <p className="mt-2 text-sm text-slate-500">{message}</p>
      </div>
    </div>
  );
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthenticatedUser | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    void initializeKeycloak()
      .then((authenticated) => {
        if (!authenticated) {
          return keycloak.login({ redirectUri: window.location.href });
        }
        if (active) {
          setUser(currentUser());
        }
      })
      .catch((reason: unknown) => {
        if (active) {
          setError(
            reason instanceof Error
              ? reason.message
              : "Authentication could not be initialized.",
          );
        }
      });

    keycloak.onAuthRefreshSuccess = () => {
      if (active) setUser(currentUser());
    };
    keycloak.onAuthLogout = () => {
      if (active) setUser(null);
    };
    keycloak.onTokenExpired = () => {
      void getAccessToken()
        .then(() => {
          if (active) setUser(currentUser());
        })
        .catch(() => undefined);
    };

    return () => {
      active = false;
    };
  }, []);

  const value = useMemo<AuthContextValue | null>(() => {
    if (!user) return null;

    return {
      user,
      getAccessToken,
      logout: () => keycloak.logout({ redirectUri: window.location.origin }),
    };
  }, [user]);

  if (error) {
    return (
      <div className="flex min-h-dvh items-center justify-center bg-slate-50 px-6">
        <div className="w-full max-w-md rounded-lg border border-red-200 bg-white p-6 text-center shadow-sm">
          <h1 className="text-base font-semibold text-slate-900">Unable to sign in</h1>
          <p className="mt-2 text-sm text-red-700">{error}</p>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="mt-5 rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
          >
            Try again
          </button>
        </div>
      </div>
    );
  }

  if (!value) {
    return <AuthenticationScreen message="Signing you in…" />;
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider.");
  }
  return context;
}
