import { AlertTriangle, Bell, Boxes, CheckCircle2, TrendingUp, XCircle } from "lucide-react";
import { stats, products, recentEvents } from "../data/mockData";
import { StatCard } from "../components/StatCard";
import { ProductSection } from "../components/ProductSection";
import { RecentEvents } from "../components/RecentEvents";

const statIcons = [Boxes, CheckCircle2, AlertTriangle, XCircle, Bell, TrendingUp];

export function Home() {
  return (
    <div className="flex h-full min-h-0 flex-col gap-5 overflow-y-auto p-6">
      <div>
        <h1 className="text-xl font-semibold text-slate-900">Service Monitoring Matrix</h1>
        <p className="text-sm text-slate-500">
          Real-time overview of all products, services and infrastructure health.
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        {stats.map((stat, i) => (
          <StatCard key={stat.label} stat={stat} icon={statIcons[i]} />
        ))}
      </div>

      <div className="flex flex-1 gap-5">
        <div className="flex min-w-0 flex-1 flex-col gap-4">
          {products.map((product) => (
            <ProductSection key={product.name} product={product} />
          ))}
        </div>
        <RecentEvents events={recentEvents} />
      </div>
    </div>
  );
}
