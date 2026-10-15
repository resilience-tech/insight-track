import type { HealthStatus } from "./types";

export const statusStyles: Record<HealthStatus, { dot: string; text: string; badge: string; label: string }> = {
  healthy: { dot: "bg-emerald-500", text: "text-emerald-700", badge: "bg-emerald-50 text-emerald-700", label: "Healthy" },
  degraded: { dot: "bg-amber-500", text: "text-amber-700", badge: "bg-amber-50 text-amber-700", label: "Degraded" },
  down: { dot: "bg-red-500", text: "text-red-700", badge: "bg-red-50 text-red-700", label: "Down" },
  recovered: { dot: "bg-blue-500", text: "text-blue-700", badge: "bg-blue-50 text-blue-700", label: "Recovered" },
};
