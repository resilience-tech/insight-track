import { useState } from "react";
import { ChevronDown, Server as ServerIcon, ShoppingCart, type LucideIcon } from "lucide-react";
import type { Product } from "../types";
import { statusStyles } from "../statusStyles";
import { Sparkline } from "./Sparkline";
import { ServiceTable } from "./ServiceTable";
import { ServerTable } from "./ServerTable";

const productIcons: Record<string, LucideIcon> = {
  "Commerce Platform": ShoppingCart,
};

export function ProductSection({ product }: { product: Product }) {
  const [open, setOpen] = useState(true);
  const tone = statusStyles[product.status];
  const Icon = productIcons[product.name] ?? ServerIcon;

  return (
    <section className="rounded-lg border border-slate-200 bg-white">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-4 px-4 py-3"
      >
        <div className="flex items-center gap-2.5">
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-slate-100 text-slate-600">
            <Icon className="h-4 w-4" />
          </span>
          <span className="font-semibold text-slate-900">{product.name}</span>
          <span className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${tone.badge}`}>
            <span className={`h-1.5 w-1.5 rounded-full ${tone.dot}`} />
            {tone.label}
          </span>
          <span className="text-xs text-slate-500">
            {product.services.length} services | {product.servers.length} servers
          </span>
        </div>
        <div className="flex items-center gap-4">
          <span className="text-xs font-medium text-slate-500">{product.uptime}</span>
          <Sparkline values={product.history} className="h-6 w-24" stroke="#2563eb" />
          <ChevronDown className={`h-4 w-4 text-slate-400 transition-transform ${open ? "rotate-180" : ""}`} />
        </div>
      </button>

      {open && (
        <div className="space-y-5 border-t border-slate-100 px-4 py-4">
          <ServiceTable services={product.services} />
          <div>
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-400">
              Servers ({product.servers.length})
            </p>
            <ServerTable servers={product.servers} />
          </div>
        </div>
      )}
    </section>
  );
}
