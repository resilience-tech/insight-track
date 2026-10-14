import type { Server } from "../types";
import { statusStyles } from "../statusStyles";

function Gauge({ value }: { value: number }) {
  const tone = value >= 80 ? "bg-red-500" : value >= 65 ? "bg-amber-500" : "bg-emerald-500";
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-16 overflow-hidden rounded-full bg-slate-100">
        <div className={`h-full rounded-full ${tone}`} style={{ width: `${value}%` }} />
      </div>
      <span className="text-xs text-slate-600">{value}%</span>
    </div>
  );
}

export function ServerTable({ servers }: { servers: Server[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[760px] border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs font-medium text-slate-500">
            <th className="py-2 pr-3">Server</th>
            <th className="py-2 pr-3">Status</th>
            <th className="py-2 pr-3">CPU</th>
            <th className="py-2 pr-3">RAM</th>
            <th className="py-2 pr-3">Disk</th>
            <th className="py-2 pr-3">Network</th>
            <th className="py-2 pr-3">Load Avg</th>
            <th className="py-2 pr-3">Uptime</th>
            <th className="py-2 pr-3">Last Seen</th>
          </tr>
        </thead>
        <tbody>
          {servers.map((server) => {
            const tone = statusStyles[server.status];
            return (
              <tr key={server.name} className="border-b border-slate-100 last:border-0">
                <td className="py-2.5 pr-3 font-medium text-slate-900">{server.name}</td>
                <td className="py-2.5 pr-3">
                  <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${tone.badge}`}>
                    <span className={`h-1.5 w-1.5 rounded-full ${tone.dot}`} />
                    {tone.label}
                  </span>
                </td>
                <td className="py-2.5 pr-3"><Gauge value={server.cpu} /></td>
                <td className="py-2.5 pr-3"><Gauge value={server.ram} /></td>
                <td className="py-2.5 pr-3"><Gauge value={server.disk} /></td>
                <td className="py-2.5 pr-3 text-slate-600">{server.network}</td>
                <td className="py-2.5 pr-3 text-slate-600">{server.loadAvg.join(", ")}</td>
                <td className="py-2.5 pr-3 text-slate-600">{server.uptime}</td>
                <td className="py-2.5 pr-3 text-slate-500">{server.lastSeen}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
