import type { Product, RecentEvent, Stat } from "../types";

export const stats: Stat[] = [
  { label: "Total Products", value: "3", caption: "All monitored products", status: "neutral" },
  { label: "Healthy", value: "21", caption: "74% of total services", status: "healthy" },
  { label: "Degraded", value: "5", caption: "18% of total services", status: "degraded" },
  { label: "Down", value: "2", caption: "7% of total services", status: "down" },
  { label: "Active Alerts", value: "7", caption: "3 critical, 4 warning", status: "down" },
  { label: "Avg Availability", value: "99.72%", caption: "Across all products (+0.14%)", status: "info" },
];

const sparkline = (seed: number) =>
  Array.from({ length: 16 }, (_, i) => 40 + Math.round(30 * Math.sin((i + seed) / 2.5)) + seed);

const healthStrip = (pattern: string): Product["services"][number]["healthHistory"] =>
  pattern.split("").map((c) =>
    c === "d" ? "down" : c === "w" ? "degraded" : "healthy",
  );

export const products: Product[] = [
  {
    name: "Commerce Platform",
    status: "healthy",
    uptime: "99.94% uptime (24h)",
    history: sparkline(2),
    services: [
      { name: "frontend", status: "healthy", history: sparkline(1), cpu: [12, 18, 45], memory: ["512 MB", "620 MB", "1.2 GB"], avgTime: "42 ms", p95: "120 ms", errorRate: "0.01%", requestsPerSec: 452, uptime: "99.99%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "catalog", status: "healthy", history: sparkline(3), cpu: [28, 32, 71], memory: ["1.1 GB", "1.6 GB", "2.8 GB"], avgTime: "68 ms", p95: "210 ms", errorRate: "0.03%", requestsPerSec: 320, uptime: "99.98%", lastSeen: "1m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "cart", status: "degraded", history: sparkline(5), cpu: [64, 52, 92], memory: ["2.4 GB", "2.1 GB", "3.8 GB"], avgTime: "120 ms", p95: "480 ms", errorRate: "0.35%", requestsPerSec: 186, uptime: "99.12%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhwwwwhhhhhhhh") },
      { name: "payment", status: "healthy", history: sparkline(2), cpu: [18, 22, 68], memory: ["1.2 GB", "1.4 GB", "2.1 GB"], avgTime: "85 ms", p95: "260 ms", errorRate: "0.04%", requestsPerSec: 210, uptime: "99.97%", lastSeen: "3m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "order", status: "healthy", history: sparkline(4), cpu: [26, 30, 74], memory: ["1.8 GB", "2.0 GB", "3.6 GB"], avgTime: "96 ms", p95: "310 ms", errorRate: "0.08%", requestsPerSec: 178, uptime: "99.96%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "inventory", status: "down", history: sparkline(9), cpu: [92, 76, 98], memory: ["3.1 GB", "2.8 GB", "4.2 GB"], avgTime: "2.4 s", p95: "5.8 s", errorRate: "6.12%", requestsPerSec: 12, uptime: "98.21%", lastSeen: "4m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhddddwww") },
      { name: "recommendation", status: "healthy", history: sparkline(3), cpu: [34, 28, 66], memory: ["1.6 GB", "1.9 GB", "3.1 GB"], avgTime: "72 ms", p95: "190 ms", errorRate: "0.05%", requestsPerSec: 142, uptime: "99.95%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "search", status: "recovered", history: sparkline(1), cpu: [22, 40, 81], memory: ["1.3 GB", "1.7 GB", "2.9 GB"], avgTime: "110 ms", p95: "360 ms", errorRate: "0.22%", requestsPerSec: 98, uptime: "99.76%", lastSeen: "5m ago", healthHistory: healthStrip("hhhhhhhhhhhhddwwwhhh") },
    ],
    servers: [
      { name: "cp-app-01", status: "healthy", cpu: 28, ram: 62, disk: 71, network: "120 Mbps", loadAvg: [0.32, 0.41, 0.38], uptime: "21 days", lastSeen: "1m ago" },
      { name: "cp-app-02", status: "healthy", cpu: 34, ram: 68, disk: 66, network: "98 Mbps", loadAvg: [0.48, 0.52, 0.49], uptime: "18 days", lastSeen: "1m ago" },
      { name: "cp-db-01", status: "healthy", cpu: 46, ram: 72, disk: 81, network: "210 Mbps", loadAvg: [0.91, 0.88, 0.76], uptime: "27 days", lastSeen: "2m ago" },
      { name: "cp-cache-01", status: "degraded", cpu: 72, ram: 78, disk: 69, network: "320 Mbps", loadAvg: [1.25, 1.12, 0.98], uptime: "14 days", lastSeen: "1m ago" },
    ],
  },
  {
    name: "Identity & Auth",
    status: "degraded",
    uptime: "99.41% uptime (24h)",
    history: sparkline(6),
    services: [
      { name: "auth-api", status: "healthy", history: sparkline(1), cpu: [18, 24, 62], memory: ["820 MB", "1.1 GB", "2.4 GB"], avgTime: "56 ms", p95: "180 ms", errorRate: "0.02%", requestsPerSec: 310, uptime: "99.98%", lastSeen: "1m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "token-service", status: "healthy", history: sparkline(2), cpu: [22, 28, 68], memory: ["1.1 GB", "1.6 GB", "3.1 GB"], avgTime: "72 ms", p95: "240 ms", errorRate: "0.05%", requestsPerSec: 280, uptime: "99.96%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "user-profile", status: "degraded", history: sparkline(7), cpu: [68, 54, 96], memory: ["2.8 GB", "2.4 GB", "4.2 GB"], avgTime: "210 ms", p95: "620 ms", errorRate: "1.12%", requestsPerSec: 96, uptime: "99.01%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhwwwwwhhhhhh") },
    ],
    servers: [
      { name: "id-app-01", status: "healthy", cpu: 31, ram: 58, disk: 64, network: "110 Mbps", loadAvg: [0.38, 0.4, 0.36], uptime: "32 days", lastSeen: "1m ago" },
      { name: "id-app-02", status: "healthy", cpu: 29, ram: 55, disk: 61, network: "104 Mbps", loadAvg: [0.35, 0.37, 0.33], uptime: "32 days", lastSeen: "1m ago" },
      { name: "id-db-01", status: "degraded", cpu: 58, ram: 74, disk: 70, network: "180 Mbps", loadAvg: [0.82, 0.79, 0.7], uptime: "19 days", lastSeen: "2m ago" },
    ],
  },
  {
    name: "Analytics Engine",
    status: "healthy",
    uptime: "99.87% uptime (24h)",
    history: sparkline(4),
    services: [
      { name: "ingest", status: "healthy", history: sparkline(2), cpu: [24, 30, 58], memory: ["1.4 GB", "1.8 GB", "3.0 GB"], avgTime: "64 ms", p95: "200 ms", errorRate: "0.02%", requestsPerSec: 420, uptime: "99.97%", lastSeen: "1m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "query-engine", status: "healthy", history: sparkline(3), cpu: [40, 44, 76], memory: ["2.2 GB", "2.6 GB", "3.9 GB"], avgTime: "180 ms", p95: "520 ms", errorRate: "0.06%", requestsPerSec: 150, uptime: "99.94%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "analytics-worker", status: "down", history: sparkline(8), cpu: [88, 70, 95], memory: ["2.9 GB", "2.5 GB", "3.8 GB"], avgTime: "1.8 s", p95: "4.2 s", errorRate: "4.5%", requestsPerSec: 8, uptime: "98.4%", lastSeen: "6m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhdddwww") },
      { name: "recommendation-model", status: "healthy", history: sparkline(2), cpu: [20, 26, 54], memory: ["980 MB", "1.3 GB", "2.1 GB"], avgTime: "54 ms", p95: "160 ms", errorRate: "0.01%", requestsPerSec: 190, uptime: "99.98%", lastSeen: "1m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "export-service", status: "healthy", history: sparkline(1), cpu: [16, 20, 48], memory: ["640 MB", "820 MB", "1.6 GB"], avgTime: "48 ms", p95: "140 ms", errorRate: "0.01%", requestsPerSec: 112, uptime: "99.99%", lastSeen: "2m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
      { name: "scheduler", status: "healthy", history: sparkline(2), cpu: [14, 18, 40], memory: ["420 MB", "560 MB", "1.1 GB"], avgTime: "30 ms", p95: "90 ms", errorRate: "0.00%", requestsPerSec: 60, uptime: "99.99%", lastSeen: "3m ago", healthHistory: healthStrip("hhhhhhhhhhhhhhhhhhhh") },
    ],
    servers: [
      { name: "an-app-01", status: "healthy", cpu: 36, ram: 64, disk: 58, network: "140 Mbps", loadAvg: [0.5, 0.48, 0.45], uptime: "11 days", lastSeen: "1m ago" },
      { name: "an-worker-01", status: "healthy", cpu: 44, ram: 70, disk: 62, network: "160 Mbps", loadAvg: [0.6, 0.58, 0.55], uptime: "11 days", lastSeen: "1m ago" },
      { name: "an-db-01", status: "healthy", cpu: 38, ram: 66, disk: 75, network: "150 Mbps", loadAvg: [0.52, 0.5, 0.47], uptime: "26 days", lastSeen: "2m ago" },
    ],
  },
];

export const recentEvents: RecentEvent[] = [
  { service: "search", time: "2 minutes ago", description: "Service recovered after 5m downtime", tone: "recovered" },
  { service: "inventory", time: "4 minutes ago", description: "Service is down", tone: "down" },
  { service: "user-profile", time: "6 minutes ago", description: "High latency detected (P95 > 600ms)", tone: "degraded" },
  { service: "cp-cache-01", time: "8 minutes ago", description: "High CPU usage (72%)", tone: "degraded" },
  { service: "payment", time: "12 minutes ago", description: "Error rate back to normal", tone: "healthy" },
  { service: "auth-api", time: "18 minutes ago", description: "Deployed version 2.4.1", tone: "info" },
  { service: "order", time: "23 minutes ago", description: "Increased error rate (0.8%)", tone: "down" },
  { service: "token-service", time: "28 minutes ago", description: "Service recovered", tone: "healthy" },
  { service: "analytics-worker", time: "32 minutes ago", description: "Service is down", tone: "down" },
  { service: "recommendation", time: "45 minutes ago", description: "New version deployed (v1.9.0)", tone: "info" },
];
