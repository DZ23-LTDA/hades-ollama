import { useEffect, useState } from "react";

export type InterfaceMode = "simple" | "advanced";

// Dispatched on the window whenever the interface mode changes, so surfaces like
// the sidebar can react within the same tab (the native `storage` event only
// fires across tabs).
export const INTERFACE_MODE_EVENT = "hades:interface-mode";

export function getStoredInterfaceMode(storage?: Pick<Storage, "getItem">): InterfaceMode {
  try {
    return storage?.getItem("hades_interface_mode") === "advanced" ? "advanced" : "simple";
  } catch {
    return "simple";
  }
}

export function persistInterfaceMode(
  mode: InterfaceMode,
  storage?: Pick<Storage, "setItem">,
): void {
  try {
    storage?.setItem("hades_interface_mode", mode);
  } catch {
    // Private browsing or a restricted webview may deny storage.
  }
  try {
    if (typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent(INTERFACE_MODE_EVENT, { detail: mode }));
    }
  } catch {
    // CustomEvent may be unavailable in some non-DOM environments.
  }
}

// useInterfaceMode reads the stored mode and keeps it in sync when the user
// toggles Simple/Advanced (via the custom event) or another tab changes it.
export function useInterfaceMode(): InterfaceMode {
  const [mode, setMode] = useState<InterfaceMode>(() =>
    typeof window !== "undefined"
      ? getStoredInterfaceMode(window.localStorage)
      : "simple",
  );
  useEffect(() => {
    if (typeof window === "undefined") return;
    const sync = () => setMode(getStoredInterfaceMode(window.localStorage));
    window.addEventListener(INTERFACE_MODE_EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(INTERFACE_MODE_EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, []);
  return mode;
}
