import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { EdgeRevealStrip } from "@/components/layout/EdgeRevealStrip";

describe("EdgeRevealStrip", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders a button", () => {
    render(<EdgeRevealStrip onClick={vi.fn()} />);
    const strip = screen.getByRole("button");
    expect(strip).toBeInTheDocument();
  });

  it("calls onClick when clicked", () => {
    const onClick = vi.fn();
    render(<EdgeRevealStrip onClick={onClick} />);
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
