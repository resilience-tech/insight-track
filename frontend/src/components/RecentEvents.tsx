import { AlertTriangle, ArrowUpCircle, CheckCircle2, XCircle } from "lucide-react";
import type { RecentEvent } from "../types";

const toneIcon: Record<RecentEvent["tone"], { icon: typeof CheckCircle2; className: string }> = {
  healthy: { icon: CheckCircle2, className: "text-emerald-500" },
  recovered: { icon: CheckCircle2, className: "text-blue-500" },
  degraded: { icon: AlertTriangle, className: "text-amber-500" },
  down: { icon: XCircle, className: "text-red-500" },
  info: { icon: ArrowUpCircle, className: "text-blue-500" },
};

export function RecentEvents({ events }: { events: RecentEvent[] }) {
  return (
    <aside className="w-80 shrink-0 rounded-lg border border-slate-200 bg-white p-4">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-900">Recent Events</h2>
        <button type="button" className="text-xs font-medium text-blue-600 hover:underline">
          View all
        </button>
      </div>
      <ul className="space-y-3">
        {events.map((event, i) => {
          const { icon: Icon, className } = toneIcon[event.tone];
          return (
            <li key={i} className="flex gap-2.5">
              <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${className}`} />
              <div className="min-w-0">
                <p className="text-sm font-medium text-slate-900">
                  {event.service}
                  <span className="ml-2 text-xs font-normal text-slate-400">{event.time}</span>
                </p>
                <p className="text-xs text-slate-500">{event.description}</p>
              </div>
            </li>
          );
        })}
      </ul>
    </aside>
  );
}
