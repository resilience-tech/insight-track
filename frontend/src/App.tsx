import { useState } from "react";
import { Sidebar, type NavigationId } from "./components/Sidebar";
import { Topbar } from "./components/Topbar";
import { Home } from "./pages/Home";
import { Placeholder } from "./pages/Placeholder";

export default function App() {
  const [activeItem, setActiveItem] = useState<NavigationId>("home");

  return (
    <div className="flex h-dvh w-full bg-slate-50">
      <Sidebar activeItem={activeItem} onNavigate={setActiveItem} />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="min-h-0 flex-1">
          {activeItem === "home" ? <Home /> : <Placeholder title={activeItem} />}
        </main>
      </div>
    </div>
  );
}
