import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { Sidebar, type NavigationId } from "./Sidebar";

function TestSidebar() {
  const [activeItem, setActiveItem] = useState<NavigationId>("home");
  return <Sidebar activeItem={activeItem} onNavigate={setActiveItem} />;
}

describe("Sidebar", () => {
  it("renders the complete navigation with Home selected", () => {
    render(<TestSidebar />);

    expect(
      screen.getByRole("navigation", { name: "Primary navigation" }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("button")).toHaveLength(8);
    expect(screen.getByRole("button", { name: "Home" })).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  it("moves the active state when another destination is selected", async () => {
    const user = userEvent.setup();
    render(<TestSidebar />);

    await user.click(screen.getByRole("button", { name: "Analytics" }));

    expect(screen.getByRole("button", { name: "Analytics" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("button", { name: "Home" })).not.toHaveAttribute(
      "aria-current",
    );
  });
});
