import type { SVGProps } from "react";

type IconProps = SVGProps<SVGSVGElement>;

const iconProps = {
  "aria-hidden": true,
  fill: "none",
  focusable: false,
  stroke: "currentColor",
  strokeLinecap: "round",
  strokeLinejoin: "round",
  strokeWidth: 1.4,
  viewBox: "0 0 24 24",
} as const;

export function HomeIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <path d="m3 10.25 9-7.75 9 7.75v9.5a1.75 1.75 0 0 1-1.75 1.75H4.75A1.75 1.75 0 0 1 3 19.75Z" />
      <path d="M8.5 21.5v-6.75h7v6.75" />
    </svg>
  );
}

export function TrendsIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <path d="m3.5 15.75 5.25-5.25 4.5 3.5 7.25-4.75" />
      <circle cx="3.5" cy="15.75" r="1.45" fill="white" />
      <circle cx="8.75" cy="10.5" r="1.45" fill="white" />
      <circle cx="13.25" cy="14" r="1.45" fill="white" />
      <circle cx="20.5" cy="9.25" r="1.45" fill="white" />
    </svg>
  );
}

export function AnalyticsIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <path d="M3 21V13.5h5V21Z" />
      <path d="M10 21V3h5v18Z" />
      <path d="M17 21V8.5h4V21Z" />
    </svg>
  );
}

export function FunnelIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <path d="M2.5 3.25h19L14.5 12v6.75l-5 2.5V12Z" />
    </svg>
  );
}

export function UsersIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <circle cx="12" cy="7" r="4.25" />
      <path d="M3 21v-2.5a6 6 0 0 1 6-6h6a6 6 0 0 1 6 6V21" />
    </svg>
  );
}

export function DataIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <ellipse cx="12" cy="5" rx="8.25" ry="3.25" />
      <path d="M3.75 5v6.5c0 1.8 3.7 3.25 8.25 3.25s8.25-1.45 8.25-3.25V5" />
      <path d="M3.75 11.5V18c0 1.8 3.7 3.25 8.25 3.25s8.25-1.45 8.25-3.25v-6.5" />
    </svg>
  );
}

export function IntegrationsIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <g transform="translate(-1.8 -1.8) scale(1.15)">
        <path d="M8.6 3.5a2.4 2.4 0 1 1 4.8 0V7H17a2.5 2.5 0 1 1 0 5h-1v3.5h-3.5v1a2.5 2.5 0 1 1-5 0v-1H4V11h3.5V7h1.1Z" />
      </g>
    </svg>
  );
}

export function SettingsIcon(props: IconProps) {
  return (
    <svg {...iconProps} {...props}>
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.86 2.86-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1.1V21H9.55v-.1A1.7 1.7 0 0 0 8.5 19.4a1.7 1.7 0 0 0-1.88.34l-.06.06-2.86-2.86.06-.06A1.7 1.7 0 0 0 4.1 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1.1-.4H2.3V9.55h.1A1.7 1.7 0 0 0 4.1 8.5a1.7 1.7 0 0 0-.34-1.88L3.7 6.56 6.56 3.7l.06.06A1.7 1.7 0 0 0 8.5 4.1a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1.1V2.3h4.05v.1A1.7 1.7 0 0 0 15 4.1a1.7 1.7 0 0 0 1.88-.34l.06-.06 2.86 2.86-.06.06A1.7 1.7 0 0 0 19.4 8.5c.17.4.38.74.7 1 .3.25.68.4 1.1.4h.1v4.05h-.1A1.7 1.7 0 0 0 19.4 15Z" />
    </svg>
  );
}
