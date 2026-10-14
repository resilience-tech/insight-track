import {
  Activity,
  BarChart3,
  Boxes,
  Home,
  Layers,
  Lock,
  Scroll,
  Settings,
  Share2,
  Waypoints,
} from "lucide-react";

export type NavigationId =
  | "home"
  | "products"
  | "services"
  | "metrics"
  | "logs"
  | "traces"
  | "architecture"
  | "access"
  | "settings";

const items: { id: NavigationId; label: string; icon: typeof Home }[] = [
  { id: "home", label: "Home", icon: Home },
  { id: "products", label: "Products", icon: Boxes },
  { id: "services", label: "Services", icon: Layers },
  { id: "metrics", label: "Metrics", icon: BarChart3 },
  { id: "logs", label: "Logs", icon: Scroll },
  { id: "traces", label: "Traces", icon: Waypoints },
  { id: "architecture", label: "Architecture", icon: Share2 },
  { id: "access", label: "Access", icon: Lock },
  { id: "settings", label: "Settings", icon: Settings },
];

interface SidebarProps {
  activeItem: NavigationId;
  onNavigate: (id: NavigationId) => void;
}

export function Sidebar({ activeItem, onNavigate }: SidebarProps) {
  return (
    <aside className="flex w-56 shrink-0 flex-col border-r border-slate-200 bg-white px-3 py-4">
      <div className="flex items-center gap-2 px-2 pb-6">
        <span className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-900 text-white">
          <Activity className="h-4 w-4" />
        </span>
        <span className="text-sm font-semibold text-slate-900">Insight Track</span>
      </div>

      <nav className="flex flex-1 flex-col gap-1">
        {items.map(({ id, label, icon: Icon }) => {
          const active = id === activeItem;
          return (
            <button
              key={id}
              type="button"
              onClick={() => onNavigate(id)}
              className={`flex items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors ${
                active
                  ? "bg-slate-900 text-white"
                  : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
              }`}
            >
              <Icon className="h-4 w-4" />
              {label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
