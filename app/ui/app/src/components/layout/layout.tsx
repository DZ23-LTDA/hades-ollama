import { Link } from "@tanstack/react-router";
import { ChatIcon } from "@/components/ChatIcon";
import { isWindowsPlatform } from "@/lib/platform";
import { useEffect, useState } from "react";

// The agentic desktop shell is navigation-first. Keep the sidebar visible by
// default and preserve the operator's choice only for the current session.
let sessionSidebarOpen = true;

// The sidebar width is resizable via a draggable divider and is remembered
// across sessions, so the operator can fix it where they want.
const SIDEBAR_MIN_WIDTH = 176;
const SIDEBAR_MAX_WIDTH = 440;
const SIDEBAR_DEFAULT_WIDTH = 208;

function getStoredSidebarWidth(): number {
  try {
    const raw = window.localStorage.getItem("hades_sidebar_width");
    const value = raw ? parseInt(raw, 10) : NaN;
    if (!Number.isNaN(value) && value >= SIDEBAR_MIN_WIDTH && value <= SIDEBAR_MAX_WIDTH) {
      return value;
    }
  } catch {
    // Restricted storage: fall back to the default width.
  }
  return SIDEBAR_DEFAULT_WIDTH;
}

export function SidebarLayout({
  sidebar,
  title,
  children,
}: React.PropsWithChildren<{
  sidebar: React.ReactNode;
  title?: string;
}>) {
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== "undefined" && window.innerWidth < 768,
  );
  const [sidebarOpen, setSidebarOpen] = useState(() => (isMobile ? false : sessionSidebarOpen));
  const [sidebarWidth, setSidebarWidth] = useState(getStoredSidebarWidth);
  const [resizing, setResizing] = useState(false);
  const isWindows = isWindowsPlatform();

  // Drag-to-resize: while the divider is held, track the pointer and clamp the
  // width; release ends the drag. The width is persisted below so it sticks.
  useEffect(() => {
    if (!resizing) return;
    const onMove = (event: MouseEvent) => {
      const next = Math.min(
        SIDEBAR_MAX_WIDTH,
        Math.max(SIDEBAR_MIN_WIDTH, Math.round(event.clientX)),
      );
      setSidebarWidth(next);
    };
    const onUp = () => setResizing(false);
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    const previousCursor = document.body.style.cursor;
    const previousSelect = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    return () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousSelect;
    };
  }, [resizing]);

  useEffect(() => {
    try {
      window.localStorage.setItem("hades_sidebar_width", String(sidebarWidth));
    } catch {
      // Ignore storage failures (private mode / restricted webview).
    }
  }, [sidebarWidth]);

  useEffect(() => {
    const updateViewport = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (!mobile) {
        setSidebarOpen(true);
        sessionSidebarOpen = true;
      }
    };
    updateViewport();
    window.addEventListener("resize", updateViewport);
    return () => window.removeEventListener("resize", updateViewport);
  }, []);

  useEffect(() => {
    if (!isMobile || !sidebarOpen) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        sessionSidebarOpen = false;
        setSidebarOpen(false);
      }
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [isMobile, sidebarOpen]);

  const toggleSidebar = () => {
    sessionSidebarOpen = !sidebarOpen;
    setSidebarOpen(sessionSidebarOpen);
  };

  return (
    <div className="flex h-screen w-full overflow-hidden dark:bg-neutral-900">
      <div
        className={`absolute z-50 mx-2 flex items-center py-2 text-neutral-500 transition-[left] duration-300 dark:text-neutral-400 ${isMobile ? (sidebarOpen ? "left-[17rem]" : "left-2") : sidebarOpen ? (isWindows ? "left-2" : "left-[140px]") : "left-2"}`}
      >
        <button
          onClick={toggleSidebar}
          onMouseDown={(e) => {
            e.stopPropagation();
          }}
          className="h-9 w-9 flex items-center justify-center rounded-full hover:bg-neutral-100 dark:hover:bg-neutral-700/75 cursor-pointer"
          aria-label={sidebarOpen ? "Ocultar barra lateral" : "Mostrar barra lateral"}
          title={sidebarOpen ? "Ocultar barra lateral" : "Mostrar barra lateral"}
        >
          <svg
            className="h-5 w-5 fill-current"
            viewBox="0 0 24 19"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path d="M7.76132 16.6344H9.58103V1.59842H7.76132V16.6344ZM4.20898 18.2316H19.124C21.6518 18.2316 23.1293 16.6963 23.1293 14.0209V4.2205C23.1293 1.54512 21.6518 0.00351715 19.124 0.00351715H4.20898C1.54336 0.00351715 0 1.54512 0 4.2205V14.0209C0 16.6963 1.54336 18.2316 4.20898 18.2316ZM4.31191 16.3184C2.79628 16.3184 1.91327 15.4434 1.91327 13.926V4.31542C1.91327 2.79979 2.79628 1.91678 4.31191 1.91678H18.8174C20.333 1.91678 21.216 2.79979 21.216 4.31542V13.926C21.216 15.4434 20.333 16.3184 18.8174 16.3184H4.31191ZM5.85116 5.50038C6.1951 5.50038 6.49217 5.20507 6.49217 4.87968C6.49217 4.54628 6.1951 4.25722 5.85116 4.25722H3.8412C3.49725 4.25722 3.20819 4.54628 3.20819 4.87968C3.20819 5.20507 3.49725 5.50038 3.8412 5.50038H5.85116ZM5.85116 8.1158C6.1951 8.1158 6.49217 7.82049 6.49217 7.4871C6.49217 7.1537 6.1951 6.8744 5.85116 6.8744H3.8412C3.49725 6.8744 3.20819 7.1537 3.20819 7.4871C3.20819 7.82049 3.49725 8.1158 3.8412 8.1158H5.85116ZM5.85116 10.725C6.1951 10.725 6.49217 10.4439 6.49217 10.1105C6.49217 9.77713 6.1951 9.48983 5.85116 9.48983H3.8412C3.49725 9.48983 3.20819 9.77713 3.20819 10.1105C3.20819 10.4439 3.49725 10.725 3.8412 10.725H5.85116Z" />
          </svg>
        </button>
        {!title && (
          <Link
            to="/c/$chatId"
            params={{ chatId: "new" }}
            title="Nova conversa"
            className={`flex ml-1 items-center justify-center rounded-full transition-opacity duration-375 h-9 w-9 hover:bg-neutral-100 dark:hover:bg-neutral-700 ${
              sidebarOpen ? "opacity-0 pointer-events-none" : "opacity-100"
            }`}
          >
            <ChatIcon />
          </Link>
        )}
      </div>
      {isMobile && sidebarOpen && (
        <button
          type="button"
          aria-label="Fechar menu"
          className="fixed inset-0 z-30 cursor-default bg-black/30 backdrop-blur-[1px] md:hidden"
          onClick={() => {
            sessionSidebarOpen = false;
            setSidebarOpen(false);
          }}
        />
      )}
      <div
        style={!isMobile && sidebarOpen ? { width: sidebarWidth } : undefined}
        className={`flex max-h-screen flex-col overflow-hidden border-neutral-200 bg-neutral-50 ${resizing ? "" : "transition-[width,transform] duration-300"} dark:border-neutral-800 dark:bg-neutral-950/40 ${
          isMobile
            ? `fixed inset-y-0 left-0 z-40 w-72 border-r shadow-xl ${sidebarOpen ? "translate-x-0" : "-translate-x-full"}`
            : sidebarOpen
              ? "relative border-r"
              : "relative w-0"
        }`}
      >
        <div
          onDoubleClick={() => window.doubleClick && window.doubleClick()}
          onMouseDown={() => window.drag && window.drag()}
          className="h-13 w-full flex-none"
        ></div>
        {sidebar}
      </div>
      {!isMobile && sidebarOpen && (
        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Redimensionar a barra lateral"
          title="Arraste para redimensionar · duplo clique para restaurar"
          onMouseDown={(event) => {
            event.preventDefault();
            setResizing(true);
          }}
          onDoubleClick={() => setSidebarWidth(SIDEBAR_DEFAULT_WIDTH)}
          className={`relative z-20 w-1 shrink-0 cursor-col-resize transition-colors hover:bg-violet-400 dark:hover:bg-violet-500 ${
            resizing ? "bg-violet-400 dark:bg-violet-500" : "bg-neutral-200 dark:bg-neutral-800"
          }`}
        />
      )}
      <main className="flex min-w-0 flex-1 flex-col transition-all duration-300">
        <div
          className={`h-13 z-10 flex w-full flex-none items-center bg-white dark:bg-neutral-900 ${title ? "" : isWindows ? "xl:hidden" : "xl:fixed xl:bg-transparent xl:dark:bg-transparent"}`}
          onDoubleClick={() => window.doubleClick && window.doubleClick()}
          onMouseDown={() => window.drag && window.drag()}
        >
          {title && (
            <h1
              className={`${isMobile ? "pl-16" : sidebarOpen ? "pl-6" : isWindows ? "pl-16" : "pl-36"} font-rounded text-md font-medium transition-[padding-left] duration-300 dark:text-white`}
            >
              {title}
            </h1>
          )}
        </div>
        {children}
      </main>
    </div>
  );
}
