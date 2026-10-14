import type { HealthStatus } from "../types";

const dotColor: Record<HealthStatus, string> = {
  healthy: "bg-emerald-500",
  degraded: "bg-amber-400",
  down: "bg-red-500",
  recovered: "bg-blue-500",
};

// Compact strip of colored bars summarizing recent service health checks.
export function HealthHistory({ history }: { history: HealthStatus[] }) {
  return (
    <div className="flex items-center gap-[2px]">
      {history.map((status, i) => (
        <span key={i} className={`h-4 w-[3px] rounded-full ${dotColor[status]}`} />
      ))}
    </div>
  );
}
