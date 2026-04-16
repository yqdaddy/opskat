import { describe, it, expect } from "vitest";

// Test the compression mode thresholds
describe("Tab compression mode calculation", () => {
  function getCompressionMode(containerWidth: number, tabCount: number) {
    if (tabCount === 0) return "normal";
    const perTab = containerWidth / tabCount;
    if (perTab >= 150) return "normal";
    if (perTab >= 80) return "compressed";
    return "icon-only";
  }

  it("returns normal when few tabs fit comfortably", () => {
    expect(getCompressionMode(900, 5)).toBe("normal"); // 180px per tab
  });

  it("returns compressed when tabs are moderately crowded", () => {
    expect(getCompressionMode(900, 10)).toBe("compressed"); // 90px per tab
  });

  it("returns icon-only when tabs are extremely crowded", () => {
    expect(getCompressionMode(900, 15)).toBe("icon-only"); // 60px per tab
  });

  it("returns normal for zero tabs", () => {
    expect(getCompressionMode(900, 0)).toBe("normal");
  });

  it("returns normal at exactly 150px boundary", () => {
    expect(getCompressionMode(600, 4)).toBe("normal"); // exactly 150
  });

  it("returns compressed at exactly 80px boundary", () => {
    expect(getCompressionMode(800, 10)).toBe("compressed"); // exactly 80
  });

  it("returns icon-only just below 80px boundary", () => {
    expect(getCompressionMode(790, 10)).toBe("icon-only"); // 79px
  });
});

// Test indicator color logic
describe("Tab indicator color logic", () => {
  function shouldShowIndicator(isActive: boolean, indicatorColor?: string) {
    return isActive || !!indicatorColor;
  }

  function getIndicatorStyle(isActive: boolean, indicatorColor?: string) {
    if (!shouldShowIndicator(isActive, indicatorColor)) return null;
    if (indicatorColor) return { backgroundColor: indicatorColor };
    return "bg-primary";
  }

  it("shows primary indicator for active tab without custom color", () => {
    expect(getIndicatorStyle(true)).toBe("bg-primary");
  });

  it("shows custom color for active tab with custom color", () => {
    expect(getIndicatorStyle(true, "#ef4444")).toEqual({ backgroundColor: "#ef4444" });
  });

  it("shows custom color for inactive tab with custom color", () => {
    expect(getIndicatorStyle(false, "#22c55e")).toEqual({ backgroundColor: "#22c55e" });
  });

  it("shows no indicator for inactive tab without custom color", () => {
    expect(getIndicatorStyle(false)).toBeNull();
  });
});
