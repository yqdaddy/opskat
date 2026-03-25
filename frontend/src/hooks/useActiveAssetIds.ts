import { useMemo } from "react";
import { useTerminalStore, getTerminalActiveAssetIds } from "../stores/terminalStore";
import { useTabStore } from "../stores/tabStore";
import { getQueryActiveAssetIds } from "../stores/queryStore";

/**
 * Returns the set of asset IDs that are currently "active" across all tab types.
 * - Terminal tabs: asset has at least one connected pane
 * - Query tabs (database/redis): tab is open
 */
export function useActiveAssetIds(): Set<number> {
  // Subscribe to the state slices that affect active IDs
  const tabData = useTerminalStore((s) => s.tabData);
  const tabs = useTabStore((s) => s.tabs);

  return useMemo(() => {
    const terminalIds = getTerminalActiveAssetIds();
    const queryIds = getQueryActiveAssetIds();
    if (queryIds.size === 0) return terminalIds;
    return new Set([...terminalIds, ...queryIds]);
  }, [tabData, tabs]);
}
