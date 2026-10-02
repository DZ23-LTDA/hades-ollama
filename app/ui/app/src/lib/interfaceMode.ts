export type InterfaceMode = "simple" | "advanced";

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
}
