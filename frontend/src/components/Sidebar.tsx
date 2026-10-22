import { Activity, Home } from "lucide-react";

export function Sidebar() {
  return (
    <aside className="flex w-56 shrink-0 flex-col border-r border-slate-200 bg-white px-3 py-4">
      <div className="flex items-center gap-2 px-2 pb-6">
        <span className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-900 text-white">
          <Activity className="h-4 w-4" />
        </span>
        <span className="text-sm font-semibold text-slate-900">Insight Track</span>
      </div>

      <nav className="flex flex-1 flex-col gap-1">
        <div className="flex items-center gap-3 rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white">
          <Home className="h-4 w-4" />
          Home
        </div>
      </nav>
    </aside>
  );
}
