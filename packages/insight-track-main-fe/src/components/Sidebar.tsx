import { type ComponentType, type SVGProps } from "react";
import {
  AnalyticsIcon,
  DataIcon,
  FunnelIcon,
  HomeIcon,
  IntegrationsIcon,
  SettingsIcon,
  TrendsIcon,
  UsersIcon,
} from "./icons";

export type NavigationId =
  | "home"
  | "trends"
  | "analytics"
  | "funnels"
  | "users"
  | "data"
  | "integrations"
  | "settings";

type NavigationItem = {
  id: NavigationId;
  label: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
};

type SidebarProps = {
  activeItem: NavigationId;
  onNavigate: (item: NavigationId) => void;
};

const navigationItems: NavigationItem[] = [
  { id: "home", label: "Home", icon: HomeIcon },
  { id: "trends", label: "Trends", icon: TrendsIcon },
  { id: "analytics", label: "Analytics", icon: AnalyticsIcon },
  { id: "funnels", label: "Funnels", icon: FunnelIcon },
  { id: "users", label: "Users", icon: UsersIcon },
  { id: "data", label: "Data", icon: DataIcon },
  { id: "integrations", label: "Integrations", icon: IntegrationsIcon },
  { id: "settings", label: "Settings", icon: SettingsIcon },
];

export function Sidebar({ activeItem, onNavigate }: SidebarProps) {
  return (
    <aside className="relative z-10 min-h-0 border-r border-[#dfe2e6] bg-white">
      <nav
        aria-label="Primary navigation"
        className="flex h-full flex-col items-center gap-[25.5px] overflow-y-auto px-3 pt-[80px] pb-8 max-sm:gap-[18px] max-sm:pt-8"
      >
        {navigationItems.map(({ id, label, icon: Icon }) => {
          const isActive = id === activeItem;

          return (
            <div className="group relative shrink-0" key={id}>
              <button
                type="button"
                aria-label={label}
                aria-current={isActive ? "page" : undefined}
                onClick={() => onNavigate(id)}
                className={[
                  "grid size-16 cursor-pointer place-items-center rounded-[15px] border text-[#3b3e43] transition-[background-color,border-color,box-shadow,transform] duration-150 max-sm:size-14 max-sm:rounded-[13px]",
                  "hover:-translate-y-px hover:border-[#e6eaf0] hover:bg-[#f8fafc]",
                  isActive
                    ? "border-[#dfe9f9] bg-[#e4edfb] shadow-[0_8px_24px_rgba(80,120,185,0.07)] hover:border-[#d8e4f7] hover:bg-[#dfeafb]"
                    : "border-[#f2f3f5] bg-white shadow-[0_8px_24px_rgba(32,44,64,0.018)]",
                ].join(" ")}
              >
                <Icon className="size-9 max-sm:size-8" />
              </button>
              <span
                role="tooltip"
                className="pointer-events-none absolute top-1/2 left-[calc(100%+12px)] z-20 -translate-y-1/2 rounded-md bg-[#30343b] px-2.5 py-1.5 text-xs font-medium whitespace-nowrap text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
              >
                {label}
              </span>
            </div>
          );
        })}
      </nav>
    </aside>
  );
}
