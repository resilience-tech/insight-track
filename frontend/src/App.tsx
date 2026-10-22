import { Sidebar } from "./components/Sidebar";
import { Topbar } from "./components/Topbar";
import { Home } from "./pages/Home";

export default function App() {
  return (
    <div className="flex h-dvh w-full bg-slate-50">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="min-h-0 flex-1">
          <Home />
        </main>
      </div>
    </div>
  );
}
