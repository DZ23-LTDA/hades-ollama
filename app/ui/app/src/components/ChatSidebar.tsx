import { useChats } from "@/hooks/useChats";
import { useRenameChat } from "@/hooks/useRenameChat";
import { useDeleteChat } from "@/hooks/useDeleteChat";
import { useQueryClient } from "@tanstack/react-query";
import { getChat } from "@/api";
import { Link } from "@/components/ui/link";
import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import { ChatsResponse } from "@/gotypes";
import { AppNavigation } from "@/components/AppSidebar";
import { PencilSquareIcon, TrashIcon } from "@heroicons/react/24/outline";

// there's a hidden debug feature to copy a chat's data to the clipboard by
// holding shift and clicking this many times within this many seconds
const DEBUG_SHIFT_CLICKS_REQUIRED = 5;
const DEBUG_SHIFT_CLICK_WINDOW_MS = 7000; // 7 seconds
interface ChatSidebarProps {
  currentChatId?: string;
}

export function ChatSidebar({ currentChatId }: ChatSidebarProps) {
  const { data, isLoading, error } = useChats();
  const queryClient = useQueryClient();
  const renameMutation = useRenameChat();
  const deleteMutation = useDeleteChat();
  const [editingChatId, setEditingChatId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const [shiftClicks, setShiftClicks] = useState<Record<string, number[]>>({});
  const [copiedChatId, setCopiedChatId] = useState<string | null>(null);
  // In-app context menu. The native webview context menu is only implemented
  // on macOS (window.menu never resolves on Windows, hanging the right-click
  // "Renomear/Excluir" action), so we render our own menu, which works on
  // every platform.
  const [contextMenu, setContextMenu] = useState<{
    chatId: string;
    chatTitle: string;
    x: number;
    y: number;
  } | null>(null);

  const handleMouseEnter = useCallback(
    (chatId: string) => {
      queryClient.prefetchQuery({
        queryKey: ["chat", chatId],
        queryFn: () => getChat(chatId),
        staleTime: 1500,
      });
    },
    [queryClient],
  );

  const startEditing = useCallback((chatId: string, currentTitle: string) => {
    setEditingChatId(chatId);
    setEditValue(currentTitle);
  }, []);

  const saveRename = useCallback(async () => {
    if (!editingChatId || !editValue.trim()) {
      setEditingChatId(null);
      return;
    }

    const newTitle = editValue.trim();
    const chatId = editingChatId;

    // Exit edit mode immediately to prevent flash
    setEditingChatId(null);
    setEditValue("");

    // Optimistically update the cache
    queryClient.setQueryData(
      ["chats"],
      (oldData: ChatsResponse | undefined) => {
        if (!oldData?.chatInfos) return oldData;
        return {
          ...oldData,
          chatInfos: oldData.chatInfos.map((chat) =>
            chat.id === chatId ? { ...chat, title: newTitle } : chat,
          ),
        };
      },
    );

    try {
      await renameMutation.mutateAsync({
        chatId: chatId,
        title: newTitle,
      });
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
    } catch (error: unknown) {
      // Revert optimistic update on error
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    }
  }, [editingChatId, editValue, renameMutation, queryClient]);

  useEffect(() => {
    if (editingChatId && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [editingChatId]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        inputRef.current &&
        !inputRef.current.contains(event.target as Node)
      ) {
        saveRename();
      }
    };

    if (editingChatId) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => {
        document.removeEventListener("mousedown", handleClickOutside);
      };
    }
  }, [editingChatId, editValue, saveRename]);

  const sortedChats = useMemo(() => {
    if (!data?.chatInfos) return [];
    return [...data.chatInfos].sort((a, b) => {
      const comparison = b.updatedAt.getTime() - a.updatedAt.getTime();
      if (comparison === 0) {
        return b.id.localeCompare(a.id);
      }
      return comparison;
    });
  }, [data?.chatInfos]);

  // Group chats by time period
  const groupedChats = useMemo(() => {
    const isToday = (date: Date) => {
      const today = new Date();
      return (
        date.getDate() === today.getDate() &&
        date.getMonth() === today.getMonth() &&
        date.getFullYear() === today.getFullYear()
      );
    };
    const isThisWeek = (date: Date) => {
      const now = new Date();
      const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
      return date > weekAgo && !isToday(date);
    };
    const groups = {
      today: [] as typeof sortedChats,
      thisWeek: [] as typeof sortedChats,
      older: [] as typeof sortedChats,
    };

    sortedChats.forEach((chat) => {
      if (isToday(chat.updatedAt)) {
        groups.today.push(chat);
      } else if (isThisWeek(chat.updatedAt)) {
        groups.thisWeek.push(chat);
      } else {
        groups.older.push(chat);
      }
    });

    return groups;
  }, [sortedChats]);

  const chatGroups = useMemo(() => {
    return [
      { name: "Hoje", chats: groupedChats.today },
      { name: "Esta semana", chats: groupedChats.thisWeek },
      { name: "Mais antigas", chats: groupedChats.older },
    ].filter((group) => group.chats.length > 0);
  }, [groupedChats]);

  const handleDeleteChat = useCallback(
    async (chatId: string) => {
      const confirmed = window.confirm(
        `Tem certeza de que deseja remover esta conversa?`,
      );

      if (!confirmed) return;

      try {
        await deleteMutation.mutateAsync(chatId);
      } catch (error) {
        console.error("Failed to delete chat:", error);
      }
    },
    [deleteMutation],
  );

  // implementation of the hidden debug feature to copy a chat's data to the clipboard
  const handleShiftClick = useCallback(
    async (e: React.MouseEvent, chatId: string) => {
      if (!e.shiftKey) return false;

      e.preventDefault();
      const now = Date.now();

      const clicks = shiftClicks[chatId] || [];
      const recentClicks = clicks.filter(
        (timestamp) => now - timestamp < DEBUG_SHIFT_CLICK_WINDOW_MS,
      );
      recentClicks.push(now);

      setShiftClicks((prev) => ({
        ...prev,
        [chatId]: recentClicks,
      }));

      if (recentClicks.length >= DEBUG_SHIFT_CLICKS_REQUIRED) {
        try {
          const chatData = await getChat(chatId);
          const jsonString = JSON.stringify(chatData, null, 2);
          await navigator.clipboard.writeText(jsonString);

          // visual feedback
          setCopiedChatId(chatId);
          setTimeout(() => setCopiedChatId(null), 2000);

          setShiftClicks((prev) => ({
            ...prev,
            [chatId]: [],
          }));
        } catch (error) {
          console.error("Failed to copy chat data:", error);
        }
      }

      return true;
    },
    [shiftClicks],
  );

  const handleContextMenu = useCallback(
    (e: React.MouseEvent, chatId: string, chatTitle: string) => {
      e.preventDefault();
      setContextMenu({ chatId, chatTitle, x: e.clientX, y: e.clientY });
    },
    [],
  );

  // Dismiss the context menu on outside click, Escape, scroll or resize.
  useEffect(() => {
    if (!contextMenu) return;
    const close = () => setContextMenu(null);
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("click", close);
    window.addEventListener("resize", close);
    window.addEventListener("keydown", onKey);
    window.addEventListener("scroll", close, true);
    return () => {
      window.removeEventListener("click", close);
      window.removeEventListener("resize", close);
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("scroll", close, true);
    };
  }, [contextMenu]);

  return (
    <nav
      aria-busy={isLoading || undefined}
      className="flex flex-1 flex-col min-h-0 select-none"
    >
      {/* Single scroll region: the navigation (with the profile) and the chat
          history share one scrollable column so nothing is cut off on short
          viewports (they used to compete for height). */}
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain scrollbar-gutter">
      <div className="flex flex-col gap-0.5 px-4 pb-2">
        <AppNavigation current="chat" />
      </div>
      <div className="flex flex-col px-4 py-1">
        {error ? (
          <div className="px-2 pt-4 text-sm text-red-500">
            Erro ao carregar as conversas
          </div>
        ) : (
          <div className="flex flex-col gap-3 pt-4">
            {chatGroups.map((group) => (
              <div key={group.name} className="flex flex-col gap-0.5">
                <h3 className="text-xs font-medium text-neutral-400 dark:text-neutral-500 px-2 py-1 select-none">
                  {group.name}
                </h3>
                {group.chats.map((chat) => (
                  <div
                    key={chat.id}
                    className={`allow-context-menu group/chat flex items-center relative text-sm text-neutral-800 dark:text-neutral-400 rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 ${
                      chat.id === currentChatId
                        ? "bg-neutral-100 text-black dark:bg-neutral-800"
                        : ""
                    }`}
                    onMouseEnter={() => handleMouseEnter(chat.id)}
                    onContextMenu={(e) =>
                      handleContextMenu(
                        e,
                        chat.id,
                        chat.title ||
                          chat.userExcerpt ||
                          chat.createdAt.toLocaleString(),
                      )
                    }
                  >
                    {editingChatId === chat.id ? (
                      <div className="flex-1 flex items-center min-w-0 px-2 py-2 bg-neutral-100 text-black dark:bg-neutral-800 rounded-lg">
                        <span className="truncate font-sans text-sm w-full">
                          <input
                            ref={inputRef}
                            type="text"
                            value={editValue}
                            onChange={(e) => setEditValue(e.target.value)}
                            onKeyDown={(e) => {
                              if (e.key === "Enter") {
                                e.preventDefault();
                                saveRename();
                              } else if (e.key === "Escape") {
                                setEditingChatId(null);
                                setEditValue("");
                              }
                            }}
                            className="bg-transparent border-0 focus:outline-none w-full dark:text-white"
                            style={{
                              font: "inherit",
                              lineHeight: "inherit",
                              padding: 0,
                              margin: 0,
                            }}
                          />
                        </span>
                      </div>
                    ) : (
                      <Link
                        to="/c/$chatId"
                        params={{ chatId: chat.id }}
                        className="flex-1 flex items-center min-w-0 px-2 py-2 select-none"
                        onClick={(e) => {
                          handleShiftClick(e, chat.id);
                        }}
                        draggable={false}
                      >
                        <span className="truncate font-sans text-sm">
                          {chat.title ||
                            chat.userExcerpt ||
                            chat.createdAt.toLocaleString()}
                        </span>
                        {copiedChatId === chat.id && (
                          <span className="ml-2 text-xs text-green-600 dark:text-green-400">
                            Copiado!
                          </span>
                        )}
                      </Link>
                    )}
                    {editingChatId !== chat.id && (
                      <div className="absolute right-1 top-1/2 flex -translate-y-1/2 items-center gap-0.5 rounded-md bg-neutral-100/95 opacity-0 transition-opacity focus-within:opacity-100 group-hover/chat:opacity-100 dark:bg-neutral-800/95">
                        <button
                          type="button"
                          aria-label="Renomear conversa"
                          title="Renomear"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            startEditing(
                              chat.id,
                              chat.title ||
                                chat.userExcerpt ||
                                chat.createdAt.toLocaleString(),
                            );
                          }}
                          className="rounded p-1 text-neutral-500 hover:bg-neutral-200 hover:text-neutral-900 dark:hover:bg-neutral-700 dark:hover:text-white"
                        >
                          <PencilSquareIcon className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          aria-label="Excluir conversa"
                          title="Excluir"
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            handleDeleteChat(chat.id);
                          }}
                          className="rounded p-1 text-neutral-500 hover:bg-red-100 hover:text-red-700 dark:hover:bg-red-950/50 dark:hover:text-red-300"
                        >
                          <TrashIcon className="h-4 w-4" />
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ))}
          </div>
        )}
      </div>
      </div>
      {contextMenu && (
        <div
          role="menu"
          aria-label="Ações da conversa"
          className="fixed z-50 min-w-[160px] rounded-lg border border-neutral-200 bg-white py-1 shadow-lg dark:border-neutral-700 dark:bg-neutral-800"
          style={{ top: contextMenu.y, left: contextMenu.x }}
          onClick={(e) => e.stopPropagation()}
          onContextMenu={(e) => e.preventDefault()}
        >
          <button
            type="button"
            role="menuitem"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-neutral-800 hover:bg-neutral-100 dark:text-neutral-200 dark:hover:bg-neutral-700"
            onClick={() => {
              startEditing(contextMenu.chatId, contextMenu.chatTitle);
              setContextMenu(null);
            }}
          >
            <PencilSquareIcon className="h-4 w-4" />
            Renomear
          </button>
          <button
            type="button"
            role="menuitem"
            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-red-600 hover:bg-neutral-100 dark:text-red-400 dark:hover:bg-neutral-700"
            onClick={() => {
              handleDeleteChat(contextMenu.chatId);
              setContextMenu(null);
            }}
          >
            <TrashIcon className="h-4 w-4" />
            Excluir
          </button>
        </div>
      )}
    </nav>
  );
}
