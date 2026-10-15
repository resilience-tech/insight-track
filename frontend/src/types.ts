export type HealthStatus = "healthy" | "degraded" | "down" | "recovered";

export interface Stat {
  label: string;
  value: string;
  caption: string;
  status: HealthStatus | "neutral" | "info";
}

export interface Service {
  name: string;
  status: HealthStatus;
  history: number[];
  cpu: [number, number, number]; // current / avg / peak (%)
  memory: [string, string, string]; // current / avg / peak
  avgTime: string;
  p95: string;
  errorRate: string;
  requestsPerSec: number;
  uptime: string;
  lastSeen: string;
  healthHistory: HealthStatus[];
}

export interface Server {
  name: string;
  status: HealthStatus;
  cpu: number;
  ram: number;
  disk: number;
  network: string;
  loadAvg: [number, number, number];
  uptime: string;
  lastSeen: string;
}

export interface Product {
  name: string;
  status: HealthStatus;
  uptime: string;
  history: number[];
  services: Service[];
  servers: Server[];
}

export interface RecentEvent {
  service: string;
  time: string;
  description: string;
  tone: HealthStatus | "info";
}
