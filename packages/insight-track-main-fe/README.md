# Insight Track main frontend

A responsive React application shell built with TypeScript, Vite, and Tailwind CSS.
It reproduces the supplied navigation-rail design while keeping the workspace empty
for feature teams to compose dashboards and project views.

## Commands

```bash
npm install --workspaces=false
npm run dev
npm test
npm run build
```

The desktop rail is 120 px wide with 64 px navigation targets. On compact screens,
it scales to an 88 px rail with 56 px targets. Every item has an accessible name,
keyboard focus styling, a tooltip, and an interactive active state.
