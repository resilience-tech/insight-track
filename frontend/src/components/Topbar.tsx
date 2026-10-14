import { Bell, ChevronDown, RefreshCw, Search } from "lucide-react";

export function Topbar() {
  return (
    <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b border-slate-200 bg-white px-6">
      <button
        type="button"
        className="flex items-center gap-1.5 rounded-md border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
      >
        Default Workspace
        <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
      </button>

      <div className="relative hidden max-w-md flex-1 sm:block">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
        <input
          type="search"
          placeholder="Search products, services, logs, traces, metrics..."
          className="w-full rounded-md border border-slate-200 bg-slate-50 py-1.5 pl-9 pr-3 text-sm text-slate-700 placeholder:text-slate-400 focus:border-slate-300 focus:outline-none"
        />
      </div>

      <div className="flex items-center gap-3">
        <button
          type="button"
          className="flex items-center gap-1.5 rounded-md border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
        >
          Last 15 min
          <ChevronDown className="h-3.5 w-3.5 text-slate-400" />
        </button>
        <button
          type="button"
          aria-label="Refresh"
          className="rounded-md border border-slate-200 p-1.5 text-slate-500 hover:bg-slate-50"
        >
          <RefreshCw className="h-4 w-4" />
        </button>
        <button
          type="button"
          aria-label="Notifications"
          className="rounded-md border border-slate-200 p-1.5 text-slate-500 hover:bg-slate-50"
        >
          <Bell className="h-4 w-4" />
        </button>
        <div className="flex items-center gap-2 pl-2">
          <span className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-200 text-xs font-semibold text-slate-600">
            AC
          </span>
          <div className="hidden text-left leading-tight md:block">
            <p className="text-sm font-medium text-slate-900">Alex Chen</p>
            <p className="text-xs text-slate-500">Operations</p>
          </div>
        </div>
      </div>
    </header>
  );
}
