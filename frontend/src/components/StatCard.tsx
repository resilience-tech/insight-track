import type { ComponentType } from "react";
import type { Stat } from "../types";

const toneClasses: Record<Stat["status"], { icon: string; iconBg: string }> = {
  neutral: { icon: "text-slate-600", iconBg: "bg-slate-100" },
  healthy: { icon: "text-emerald-600", iconBg: "bg-emerald-50" },
  degraded: { icon: "text-amber-600", iconBg: "bg-amber-50" },
  down: { icon: "text-red-600", iconBg: "bg-red-50" },
  recovered: { icon: "text-blue-600", iconBg: "bg-blue-50" },
  info: { icon: "text-blue-600", iconBg: "bg-blue-50" },
};

interface StatCardProps {
  stat: Stat;
  icon: ComponentType<{ className?: string }>;
}

export function StatCard({ stat, icon: Icon }: StatCardProps) {
  const tone = toneClasses[stat.status];

  return (
    <div className="flex flex-1 items-start gap-3 rounded-lg border border-slate-200 bg-white p-4">
      <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-md ${tone.iconBg}`}>
        <Icon className={`h-5 w-5 ${tone.icon}`} />
      </span>
      <div className="min-w-0">
        <p className="text-xs font-medium text-slate-500">{stat.label}</p>
        <p className="mt-0.5 text-2xl font-semibold text-slate-900">{stat.value}</p>
        <p className="mt-0.5 truncate text-xs text-slate-500">{stat.caption}</p>
      </div>
    </div>
  );
}
