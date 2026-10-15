import type { Service } from "../types";
import { statusStyles } from "../statusStyles";
import { HealthHistory } from "./HealthHistory";
import { Sparkline } from "./Sparkline";

export function ServiceTable({ services }: { services: Service[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[920px] border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs font-medium text-slate-500">
            <th className="py-2 pr-3">Service</th>
            <th className="py-2 pr-3">Status</th>
            <th className="py-2 pr-3">Recent History</th>
            <th className="py-2 pr-3">CPU (cur / avg / peak)</th>
            <th className="py-2 pr-3">Memory (cur / avg / peak)</th>
            <th className="py-2 pr-3">Avg Time</th>
            <th className="py-2 pr-3">P95</th>
            <th className="py-2 pr-3">Error Rate</th>
            <th className="py-2 pr-3">Requests/s</th>
            <th className="py-2 pr-3">Uptime</th>
            <th className="py-2 pr-3">Last Seen</th>
            <th className="py-2 pr-3">Health History</th>
          </tr>
        </thead>
        <tbody>
          {services.map((service) => {
            const tone = statusStyles[service.status];
            return (
              <tr key={service.name} className="border-b border-slate-100 last:border-0">
                <td className="py-2.5 pr-3 font-medium text-slate-900">{service.name}</td>
                <td className="py-2.5 pr-3">
                  <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${tone.badge}`}>
                    <span className={`h-1.5 w-1.5 rounded-full ${tone.dot}`} />
                    {tone.label}
                  </span>
                </td>
                <td className="py-2.5 pr-3">
                  <Sparkline values={service.history} className="h-6 w-20" stroke="#64748b" />
                </td>
                <td className="py-2.5 pr-3 text-slate-600">{service.cpu.join(" / ")}%</td>
                <td className="py-2.5 pr-3 text-slate-600">{service.memory.join(" / ")}</td>
                <td className="py-2.5 pr-3 text-slate-600">{service.avgTime}</td>
                <td className="py-2.5 pr-3 text-slate-600">{service.p95}</td>
                <td className="py-2.5 pr-3 text-slate-600">{service.errorRate}</td>
                <td className="py-2.5 pr-3 text-slate-600">{service.requestsPerSec}</td>
                <td className="py-2.5 pr-3 text-slate-600">{service.uptime}</td>
                <td className="py-2.5 pr-3 text-slate-500">{service.lastSeen}</td>
                <td className="py-2.5 pr-3">
                  <HealthHistory history={service.healthHistory} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
