import MessageList from "./MessageList";
import ChatForm from "./ChatForm";
import { FirstModelCard } from "./FirstModelCard";
import { FileUpload } from "./FileUpload";
import { ImportProjectDialog } from "./ImportProjectDialog";
import { DisplayUpgrade } from "./DisplayUpgrade";
import { DisplayStale } from "./DisplayStale";
import { DisplayLogin } from "./DisplayLogin";
import {
  useChat,
  useSendMessage,
  useIsStreaming,
  useIsWaitingForLoad,
  useDownloadProgress,
  useChatError,
  useShouldShowStaleDisplay,
  useDismissStaleModel,
} from "@/hooks/useChats";
import { useHealth } from "@/hooks/useHealth";
import { useMessageAutoscroll } from "@/hooks/useMessageAutoscroll";
import {
  useState,
  useEffect,
  useLayoutEffect,
  useRef,
  useCallback,
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useSelectedModel } from "@/hooks/useSelectedModel";
import { useUser } from "@/hooks/useUser";
import { useHasVisionCapability } from "@/hooks/useModelCapabilities";
import { Message } from "@/gotypes";

export default function Chat({ chatId }: { chatId: string }) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const chatQuery = useChat(chatId === "new" ? "" : chatId);
  const chatErrorQuery = useChatError(chatId === "new" ? "" : chatId);
  const { selectedModel } = useSelectedModel(chatId);
  const { user } = useUser();
  const hasVisionCapability = useHasVisionCapability(selectedModel?.model);
  const shouldShowStaleDisplay = useShouldShowStaleDisplay(selectedModel);
  const dismissStaleModel = useDismissStaleModel();
  const { isHealthy } = useHealth();

  const [editingMessage, setEditingMessage] = useState<{
    content: string;
    index: number;
    originalMessage: Message;
  } | null>(null);
  const [prefill, setPrefill] = useState<{ text: string; nonce: number } | null>(
    null,
  );
  const [importOpen, setImportOpen] = useState(false);
  const prevChatIdRef = useRef<string>(chatId);

  const chatFormCallbackRef = useRef<
    | ((
        files: Array<{ filename: string; data: Uint8Array; type?: string }>,
        errors: Array<{ filename: string; error: string }>,
      ) => void)
    | null
  >(null);

  const handleFilesReceived = useCallback(
    (
      callback: (
        files: Array<{
          filename: string;
          data: Uint8Array;
          type?: string;
        }>,
        errors: Array<{ filename: string; error: string }>,
      ) => void,
    ) => {
      chatFormCallbackRef.current = callback;
    },
    [],
  );

  const handleFilesProcessed = useCallback(
    (
      files: Array<{ filename: string; data: Uint8Array; type?: string }>,
      errors: Array<{ filename: string; error: string }> = [],
    ) => {
      chatFormCallbackRef.current?.(files, errors);
    },
    [],
  );

  const allMessages = chatQuery?.data?.chat?.messages ?? [];
  // TODO(parthsareen): will need to consolidate when used with more tools with state
  const browserToolResult = chatQuery?.data?.chat?.browser_state;
  const chatError = chatErrorQuery.data;

  const messages = allMessages;
  const isStreaming = useIsStreaming(chatId);
  const isWaitingForLoad = useIsWaitingForLoad(chatId);
  const downloadProgress = useDownloadProgress(chatId);
  const isDownloadingModel = downloadProgress && !downloadProgress.done;
  const isDisabled = !isHealthy;

  // Clear editing state when navigating to a different chat
  useEffect(() => {
    setEditingMessage(null);
  }, [chatId]);

  const sendMessageMutation = useSendMessage(chatId);

  const { containerRef, handleNewUserMessage, spacerHeight } =
    useMessageAutoscroll({
      messages,
      isStreaming,
      chatId,
    });

  // Scroll to bottom only when switching to a different existing chat
  useLayoutEffect(() => {
    // Only scroll if the chatId actually changed (not just messages updating)
    if (
      prevChatIdRef.current !== chatId &&
      containerRef.current &&
      messages.length > 0 &&
      chatId !== "new"
    ) {
      // Always scroll to the bottom when opening a chat
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
    prevChatIdRef.current = chatId;
  }, [chatId, messages.length, containerRef]);

  // Simplified submit handler - ChatForm handles all the attachment logic
  const handleChatFormSubmit = (
    message: string,
    options: {
      attachments?: Array<{ filename: string; data: Uint8Array }>;
      index?: number;
      webSearch?: boolean;
      fileTools?: boolean;
      think?: boolean | string;
    },
  ) => {
    // Clear any existing errors when sending a new message
    sendMessageMutation.reset();
    if (chatError) {
      clearChatError();
    }

    // Prepare attachments for backend
    const allAttachments = (options.attachments || []).map((att) => ({
      filename: att.filename,
      data: att.data.length === 0 ? new Uint8Array(0) : att.data,
    }));

    sendMessageMutation.mutate({
      message,
      attachments: allAttachments,
      index: editingMessage ? editingMessage.index : options.index,
      webSearch: options.webSearch,
      fileTools: options.fileTools,
      think: options.think,
      onChatEvent: (event) => {
        if (event.eventName === "chat_created" && event.chatId) {
          navigate({
            to: "/c/$chatId",
            params: {
              chatId: event.chatId,
            },
          });
        }
      },
    });

    // Clear edit mode after submission
    setEditingMessage(null);
    handleNewUserMessage();
  };

  const handleEditMessage = (content: string, index: number) => {
    setEditingMessage({
      content,
      index,
      originalMessage: messages[index],
    });
  };

  const handleCancelEdit = () => {
    setEditingMessage(null);
    if (chatError) {
      clearChatError();
    }
  };

  const clearChatError = () => {
    queryClient.setQueryData(
      ["chatError", chatId === "new" ? "" : chatId],
      null,
    );
  };

  const isWindows = navigator.platform.toLowerCase().includes("win");

  return chatId === "new" || chatQuery ? (
    <FileUpload
      onFilesAdded={handleFilesProcessed}
      selectedModel={selectedModel}
      hasVisionCapability={hasVisionCapability}
    >
      {chatId === "new" ? (
        <div className="flex min-h-screen flex-col bg-neutral-50 dark:bg-neutral-950">
          <section className="mx-auto flex w-full max-w-5xl flex-1 flex-col justify-center px-5 pb-16 pt-10 sm:px-8">
            <div className="mx-auto w-full max-w-3xl text-center">
              <p className="text-xs font-medium uppercase tracking-[0.22em] text-neutral-400">Hades</p>
              <h1 className="mt-4 font-serif text-4xl font-medium tracking-tight text-neutral-900 dark:text-neutral-100 sm:text-5xl">O que posso fazer por você?</h1>
              <p className="mx-auto mt-4 max-w-xl text-sm leading-6 text-neutral-500 dark:text-neutral-400">Converse, pesquise, construa e execute com o runtime local-first. Para missões com plano, ferramentas e approvals, use o Agentic Console.</p>
              <div className="mt-8 text-left">
                <FirstModelCard />
              </div>
              <div className="mt-8 text-left">
                <ChatForm
                  hasMessages={false}
                  onSubmit={handleChatFormSubmit}
                  chatId={chatId}
                  autoFocus={true}
                  editingMessage={editingMessage}
                  onCancelEdit={handleCancelEdit}
                  isDownloadingModel={isDownloadingModel}
                  isDisabled={isDisabled}
                  onFilesReceived={handleFilesReceived}
                  prefill={prefill}
                />
              </div>

              <div className="mt-3 flex flex-wrap justify-center gap-2">
                {[['Criar slides', 'Crie uma apresentação profissional com roteiro, conteúdo e exportação.'], ['Criar site', 'Construa um site responsivo com preview e artifacts.'], ['Design', 'Proponha uma interface e um design system para meu produto.'], ['Criar jogos', 'Crie um jogo browser jogável e explique como executar.'], ['Pesquisar', 'Pesquise e sintetize as melhores opções para a minha tarefa, citando fontes.']].map(([label, objective]) => (
                  <button key={label} type="button" onClick={() => setPrefill({ text: objective, nonce: Date.now() })} className="rounded-full border border-neutral-200 bg-white px-4 py-2 text-xs font-medium text-neutral-600 transition hover:border-neutral-400 hover:text-neutral-900 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:border-neutral-600 dark:hover:text-white">{label}</button>
                ))}
              </div>

              <div className="mt-4 flex flex-wrap items-center justify-center gap-2 text-xs">
                <button
                  type="button"
                  onClick={() => setImportOpen(true)}
                  className="rounded-full border border-violet-200 bg-violet-50 px-4 py-2 font-medium text-violet-700 transition hover:border-violet-300 hover:bg-violet-100 dark:border-violet-900/60 dark:bg-violet-950/20 dark:text-violet-300 dark:hover:bg-violet-950/40"
                >
                  Importar projeto
                </button>
                <button
                  type="button"
                  onClick={() => window.location.assign("/agentic")}
                  className="rounded-full border border-neutral-200 bg-white px-4 py-2 font-medium text-neutral-600 transition hover:border-neutral-400 hover:text-neutral-900 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300 dark:hover:border-neutral-600 dark:hover:text-white"
                >
                  Modo missão (avançado)
                </button>
              </div>

              <div className="mt-8 text-left">
                <div className="mb-3 flex items-center justify-between"><h2 className="text-sm font-semibold text-neutral-800 dark:text-neutral-200">Sugestões para começar</h2></div>
                <div className="grid gap-3 md:grid-cols-3">
                  {[['Audite meu projeto', 'Inspecione o código, encontre riscos e proponha correções testáveis.'], ['Construa um MVP', 'Crie um produto completo, com backend, frontend, testes e preview.'], ['Pesquise o mercado', 'Compare soluções, cite fontes e entregue uma síntese com próximos passos.']].map(([title, objective]) => <button key={title} type="button" onClick={() => setPrefill({ text: objective, nonce: Date.now() })} className="min-h-28 rounded-2xl border border-neutral-200 bg-white p-4 text-left transition hover:-translate-y-0.5 hover:border-neutral-400 hover:shadow-sm dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-600"><span className="text-xs text-neutral-400">⌁</span><span className="mt-3 block text-sm font-medium leading-5 text-neutral-800 dark:text-neutral-200">{title}</span><span className="mt-1 block text-xs leading-5 text-neutral-500 dark:text-neutral-400">{objective}</span></button>)}
                </div>
              </div>
            </div>
          </section>
          <ImportProjectDialog
            open={importOpen}
            onClose={() => setImportOpen(false)}
            onImported={(project) => {
              setImportOpen(false);
              window.location.assign(
                `/agentic?project_id=${encodeURIComponent(project.id)}`,
              );
            }}
          />
        </div>
      ) : (
        <main className="flex h-screen w-full flex-col relative allow-context-menu">
          <section
            key={chatId} // This key forces React to recreate the element when chatId changes
            ref={containerRef}
            className={`flex-1 overflow-y-auto overscroll-contain relative min-h-0 ${isWindows ? "xl:pt-4" : "xl:pt-8"}`}
          >
            <MessageList
              messages={messages}
              spacerHeight={spacerHeight}
              isWaitingForLoad={isWaitingForLoad}
              isStreaming={isStreaming}
              downloadProgress={downloadProgress}
              onEditMessage={(content: string, index: number) => {
                handleEditMessage(content, index);
              }}
              editingMessageIndex={editingMessage?.index}
              error={chatError}
              browserToolResult={browserToolResult}
            />
          </section>

          <div className="flex-shrink-0 sticky bottom-0 z-20">
            {selectedModel && shouldShowStaleDisplay && (
              <div className="pb-2">
                <DisplayStale
                  model={selectedModel}
                  onDismiss={() =>
                    dismissStaleModel(selectedModel?.model || "")
                  }
                  chatId={chatId}
                  onScrollToBottom={() => {
                    if (containerRef.current) {
                      containerRef.current.scrollTo({
                        top: containerRef.current.scrollHeight,
                        behavior: "smooth",
                      });
                    }
                  }}
                />
              </div>
            )}
            {chatError && chatError.code === "usage_limit_upgrade" && (
              <div className="pb-2">
                <DisplayUpgrade
                  error={chatError}
                  onDismiss={clearChatError}
                  href={
                    user?.plan === "pro"
                      ? "https://ollama.com/settings/billing"
                      : "https://ollama.com/upgrade"
                  }
                />
              </div>
            )}
            {chatError && chatError.code === "cloud_unauthorized" && (
              <div className="pb-2">
                <DisplayLogin error={chatError} />
              </div>
            )}
            <ChatForm
              hasMessages={messages.length > 0}
              onSubmit={handleChatFormSubmit}
              chatId={chatId}
              autoFocus={true}
              editingMessage={editingMessage}
              onCancelEdit={handleCancelEdit}
              isDisabled={isDisabled}
              isDownloadingModel={isDownloadingModel}
              onFilesReceived={handleFilesReceived}
            />
          </div>
        </main>
      )}
    </FileUpload>
  ) : (
    <div>Loading...</div>
  );
}
