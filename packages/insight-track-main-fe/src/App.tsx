import { useState } from "react";
import { Sidebar, type NavigationId } from "./components/Sidebar";

export default function App() {
  const [activeItem, setActiveItem] = useState<NavigationId>("home");

  return (
    <div className="grid h-dvh min-h-[620px] w-full grid-cols-[120px_minmax(0,1fr)] overflow-hidden border border-[#e1e3e6] bg-white max-sm:grid-cols-[88px_minmax(0,1fr)]">
      <Sidebar activeItem={activeItem} onNavigate={setActiveItem} />
      <main
        aria-label={`${activeItem} workspace`}
        className="workspace-canvas relative min-w-0"
      >
        <h1 className="sr-only">{activeItem}</h1>
      </main>
    </div>
  );
}
