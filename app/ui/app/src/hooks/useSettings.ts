import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Settings } from "@/gotypes";
import { getSettings, updateSettings } from "@/api";
import { useMemo, useCallback, useSyncExternalStore } from "react";

// TODO(hoyyeva): remove turboEnabled when we remove Migration logic in useSelectedModel.ts
interface SettingsState {
  turboEnabled: boolean;
  webSearchEnabled: boolean;
  selectedModel: string;
  sidebarOpen: boolean;
  lastHomeView: string;
  onboardingVersion: number;
  thinkEnabled: boolean;
  thinkLevel: string;
}

// Type for partial settings updates
type SettingsUpdate = Partial<{
  TurboEnabled: boolean;
  WebSearchEnabled: boolean;
  ThinkEnabled: boolean;
  ThinkLevel: string;
  SelectedModel: string;
  SidebarOpen: boolean;
  LastHomeView: string;
  OnboardingVersion: number;
}>;

// Estabiliza a chave de uma escrita: a mesma intenção produz sempre a mesma
// string, independente da ordem das propriedades.
export function settingsWriteKey(updates: SettingsUpdate): string | null {
  const keys = Object.keys(updates).sort() as (keyof SettingsUpdate)[];
  if (keys.length === 0) return null;
  return keys.map((key) => `${key}=${JSON.stringify(updates[key])}`).join("&");
}

// Fotografia estável do que o backend devolveu; usada para detectar se uma
// escrita foi ou não refletida na leitura seguinte.
export function settingsSnapshot(settings: Settings): string {
  return JSON.stringify(settings);
}

export interface SettingsWriteGuard {
  key: string;
  snapshot: string;
}

// shouldSkipSettingsWrite evita repetir uma escrita idêntica enquanto o estado
// observado não mudou. Pure, para teste.
//
// Motivo (bug real): quando o /api/v1/settings aceita a escrita mas devolve um
// payload sem o campo gravado (parcial, proxy ou endpoint somente leitura),
// todo efeito "garanta X" — trocar LastHomeView para "chat", escolher o modelo
// padrão — reexecutava a cada invalidação da query: grava -> invalida ->
// refetch -> mesmo valor ausente -> grava de novo. Medido na home com payload
// vazio: 2.387 requisições a /api/v1/settings em 3s (~1.200 POST/s + ~1.180
// GET/s), saturando o renderer. Enquanto o snapshot observado não muda, a
// repetição é inútil por definição; assim que o backend reflete qualquer
// mudança, uma nova escrita é permitida.
export function shouldSkipSettingsWrite(
  guard: SettingsWriteGuard | null,
  key: string | null,
  snapshot: string,
): boolean {
  if (key === null || guard === null) return false;
  return guard.key === key && guard.snapshot === snapshot;
}

// 🚨 E-783: ***O SERVIDOR PODE NAO GUARDAR - E ISSO NAO PODE APAGAR A ESCOLHA DO UTILIZADOR.***
// *O `POST /api/v1/settings` do servidor local responde **501** (`E-560`): a escolha do modelo era
// descartada e o chat **nunca** enviava - o `E-781` mediu **12x 501** e **zero** `/api/chat`.*
// ⇒ *a leitura passa a ser **servidor + sobreposicao local**, e `settingsNotPersisted` diz a verdade.*
// *A 1.a tentativa (`E-782`) falhou - **42 651** POST - porque a sobreposicao mudava de identidade
// a cada escrita; esta versao corrige isso (**estavel**) e mantem a guarda armada no erro.*
export function applySettingsOverlay(
  base: Settings | undefined,
  overlay: SettingsUpdate,
): Settings {
  return new Settings({ ...(base ?? {}), ...overlay });
}

/**
 * 🚨 E-919: ***A SOBREPOSICAO DO `E-783` ERA POR INSTANCIA - E POR ISSO NAO CHEGAVA AOS OUTROS.***
 *
 * **O defeito, medido**: `useSettings` e' um **hook simples**, sem `createContext` (*confirmado:
 * `grep 'createContext|Provider'` em `useSettings.ts` devolve **nada***), e tinha **cinco**
 * consumidores - `ChatForm.tsx:171`, `useSelectedModel.ts:97`, `FirstModelCard.tsx:19`,
 * `ModelsPanel.tsx:19` e `c.$chatId.tsx:34`. *Cada um chamava `useState` e criava a **sua** copia de
 * `localOverlay`, `persistFailed` e `writeGuardRef`.*
 *
 * **A consequencia, exactamente**: o `ChatForm` **escrevia** no seu overlay e **mostrava** o modelo
 * escolhido (e o botao `Enviar` ficava `enabled`), mas o `useSelectedModel` - *que e' de onde o
 * `useSendMessage` le'* - via um overlay **vazio**, logo `settings.selectedModel` ficava com o valor
 * do **servidor** (vazio, porque o `POST /api/v1/settings` responde **501**), e o `mutationFn` fazia
 * **`throw new Error("No model selected")`** (`useChats.ts:302-304`) **antes** de qualquer `/api/chat`.
 * *Foi isso que produziu `0` pedidos a `/api/chat` e `12` respostas **501**; e cada uma das cinco
 * copias tinha a sua propria guarda de escrita, o que explica os **11** POST repetidos.*
 *
 * **A cura**: uma unica store, fora do React, vista por **todas** as instancias. *`useSyncExternalStore`
 * e' a API oficial para ler estado externo em React sem quebrar a estabilidade de referencia* - e a
 * estabilidade importa: o `E-782` mediu **42 651** POST porque cada escrita criava um objecto novo.
 * *Aqui a referencia so' muda quando o valor muda de facto* (`if (proximo === estado.overlay) return`).
 *
 * **Porque' nao um `SettingsProvider`**: *seria a alternativa idiomatica, mas obrigaria a mexer na
 * arvore de componentes (`__root.tsx` e os testes que a montam) para resolver um problema que e' de
 * **partilha de estado**, nao de composicao.* O §00.2 do prompt pede reutilizar em vez de reescrever;
 * esta cura fica **num** ficheiro e nao toca em nenhuma superficie.
 */
type EstadoPartilhadoDeSettings = {
  overlay: SettingsUpdate;
  persistFailed: boolean;
};

let estadoPartilhado: EstadoPartilhadoDeSettings = {
  overlay: {},
  persistFailed: false,
};

/** Guarda de escrita **unica** - a trava que impede a repeticao identica (`E-782`). */
let guardaDeEscritaPartilhada: SettingsWriteGuard | null = null;

const subscritores = new Set<() => void>();

function notificarSubscritores(): void {
  for (const s of subscritores) s();
}

/** Leitura para `useSyncExternalStore`. *Devolve a **mesma** referencia enquanto nada muda.* */
function lerEstadoPartilhado(): EstadoPartilhadoDeSettings {
  return estadoPartilhado;
}

/** **So' para testes**: le' a store partilhada sem a alterar. */
export function lerEstadoPartilhadoDeSettings(): EstadoPartilhadoDeSettings {
  return lerEstadoPartilhado();
}

function subscreverEstadoPartilhado(aoMudar: () => void): () => void {
  subscritores.add(aoMudar);
  return () => {
    subscritores.delete(aoMudar);
  };
}

/**
 * Aplica uma escrita a' sobreposicao partilhada. *Se nada mudar, **nao** cria referencia nova* -
 * e' o que impede os efeitos de re-executarem (a licao do `E-782`).
 */
export function sobreporSettings(
  atualizar: (anterior: SettingsUpdate) => SettingsUpdate,
): void {
  const proximo = atualizar(estadoPartilhado.overlay);
  if (proximo === estadoPartilhado.overlay) return;
  estadoPartilhado = { ...estadoPartilhado, overlay: proximo };
  notificarSubscritores();
}

export function marcarSettingsNaoPersistidas(falhou: boolean): void {
  if (estadoPartilhado.persistFailed === falhou) return;
  estadoPartilhado = { ...estadoPartilhado, persistFailed: falhou };
  notificarSubscritores();
}

/** O servidor guardou: a sobreposicao deixa de ser necessaria e a leitura manda. */
export function limparSobreposicaoDeSettings(): void {
  guardaDeEscritaPartilhada = null;
  if (
    Object.keys(estadoPartilhado.overlay).length === 0 &&
    !estadoPartilhado.persistFailed
  ) {
    return; // nada a limpar: *nao notificar evita renderizacoes inuteis*
  }
  estadoPartilhado = { overlay: {}, persistFailed: false };
  notificarSubscritores();
}

/** Acesso a' guarda singleton (para o `setSettings`). */
export function guardaDeEscritaDeSettings(): SettingsWriteGuard | null {
  return guardaDeEscritaPartilhada;
}

export function definirGuardaDeEscritaDeSettings(
  guarda: SettingsWriteGuard | null,
): void {
  guardaDeEscritaPartilhada = guarda;
}

/** **So' para testes**: devolve a store ao estado inicial. *Nao usar em producao.* */
export function __reiniciarEstadoPartilhadoDeSettings(): void {
  estadoPartilhado = { overlay: {}, persistFailed: false };
  guardaDeEscritaPartilhada = null;
  notificarSubscritores();
}

export function useSettings({
  refetchInterval,
}: { refetchInterval?: number } = {}) {
  const queryClient = useQueryClient();

  // 🚨 E-919: **uma** sobreposicao, **uma** bandeira e **uma** guarda - partilhadas por todos os
  // consumidores. *Antes disto cada instancia tinha as suas, e o chat nao enviava (ver acima).*
  const estado = useSyncExternalStore(
    subscreverEstadoPartilhado,
    lerEstadoPartilhado,
    lerEstadoPartilhado,
  );
  const localOverlay = estado.overlay;
  const persistFailed = estado.persistFailed;

  // Compatibilidade com o codigo que ja' existia neste ficheiro - *os nomes mantem-se, o
  // armazenamento e' que deixou de ser local ao hook.*
  const setLocalOverlay = sobreporSettings;
  const writeGuardRef = {
    get current() {
      return guardaDeEscritaDeSettings();
    },
    set current(valor: SettingsWriteGuard | null) {
      definirGuardaDeEscritaDeSettings(valor);
    },
  };

  // Fetch settings with useQuery
  const { data: settingsData, error } = useQuery({
    queryKey: ["settings"],
    queryFn: getSettings,
    refetchInterval,
  });

  // Update settings with useMutation
  const updateSettingsMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: () => {
      // E-932: AQUI **NAO** SE LIMPA NADA. *Era aqui que o loop nascia.*
      //
      // O `E-919` chamava `limparSobreposicaoDeSettings()` neste `onSuccess`, e essa funcao
      // apagava a **guarda de escrita** (`useSettings.ts:167`) **incondicionalmente** - mesmo
      // quando o servidor devolvia um payload **sem** os campos gravados.
      //
      // O ciclo, medido pelo `settings-write-loop.spec.ts`:
      //   escrita -> 200 (mas sem o campo) -> `onSuccess` -> **guarda limpa** ->
      //   `invalidateQueries` -> a query volta com o payload parcial -> o efeito "garanta X"
      //   re-executa -> a guarda esta' **vazia** -> **escreve outra vez** -> ...
      //   => **2 226** POST `/api/v1/settings` em 3s (contra os **16** da versao corrigida).
      //
      // O `onError` logo abaixo ja' fazia o **certo** (*mantem a guarda*) e o commentario dele
      // explica porquê: *«a guarda existe para travar a **repeticao identica**»*. O `onSuccess`
      // desfazia exactamente isso.
      //
      // A limpeza passou para o `setSettings`, **depois** de comparar a resposta do servidor
      // com o que foi pedido - *so' limpa quando o servidor **confirmou** os campos.*
      queryClient.invalidateQueries({ queryKey: ["settings"] });
    },
    onError: () => {
      // 🚨 E-783: ***A GUARDA NAO E' DESARMADA.*** *No `E-782` esta linha limpava a
      // guarda, e como a sobreposicao ganhava identidade nova a cada escrita os efeitos
      // re-executavam => **42 651** POST *identicos*. A guarda existe para travar a
      // **repeticao identica**; desarma-la aqui era desarma-la exactamente quando o ciclo e'
      // infinito. Uma intencao **nova** tem chave nova e continua livre.*
      marcarSettingsNaoPersistidas(true);
    },
  });

  // Extract settings with defaults, sobrepostos pelo que o servidor nao guardou (E-783).
  const servidor = useMemo(
    () => applySettingsOverlay(settingsData?.settings, localOverlay),
    [settingsData?.settings, localOverlay],
  );

  const settings: SettingsState = useMemo(
    () => ({
      turboEnabled: servidor.TurboEnabled ?? false,
      webSearchEnabled: servidor.WebSearchEnabled ?? false,
      thinkEnabled: servidor.ThinkEnabled ?? false,
      thinkLevel: servidor.ThinkLevel ?? "none",
      selectedModel: servidor.SelectedModel ?? "",
      sidebarOpen: servidor.SidebarOpen ?? false,
      lastHomeView: servidor.LastHomeView ?? "chat",
      onboardingVersion: servidor.OnboardingVersion ?? 0,
    }),
    [servidor],
  );

  // Single function to update most settings
  const setSettings = useCallback(
    /**
     * E-933: `opcoes.forcar` distingue **intencao explicita do utilizador** de **re-execucao
     * automatica de um efeito** - *e essa distincao e' o que resolve o conflito medido.*
     *
     *  - **Sem `forcar`**: a guarda (`:278`) trava uma escrita **identica** a ultima. E' o que
     *    mata o loop do `settings-write-loop.spec.ts` (*2226 POST -> converge*).
     *  - **Com `forcar`**: a guarda e' ignorada e a escrita sai. E' o que o **retry explicito**
     *    precisa - *o `OnboardingRoute.test.tsx:216` exige `toHaveBeenCalledTimes(2)` e
     *    `:217` exige que o 2.o corpo seja **igual** ao 1.o.*
     *
     * *O commentario do `onError` (`:241`) ja' dizia «uma intencao **nova** tem chave nova e
     * continua livre» - mas o retry tem chave **igual**, logo aquela regra nao o cobria.*
     */
    async (updates: SettingsUpdate, opcoes?: { forcar?: boolean }) => {
      if (!settingsData?.settings) return;

      const key = settingsWriteKey(updates);
      const snapshot = settingsSnapshot(settingsData.settings);
      if (!opcoes?.forcar && shouldSkipSettingsWrite(writeGuardRef.current, key, snapshot)) {
        return;
      }
      writeGuardRef.current = key === null ? null : { key, snapshot };

      const updatedSettings = new Settings({
        ...settingsData.settings,
        ...updates,
      });

      // 🚨 E-783: a intencao vale **ja'**, mas a sobreposicao e' **ESTAVEL**: se nada
      // mudou, devolve-se a **mesma** referencia - foi o que faltou no `E-782`, onde cada
      // escrita criava um objecto novo e re-executava os efeitos que empurram escritas.
      setLocalOverlay((anterior) => {
        const proximo: SettingsUpdate = { ...anterior, ...updates };
        const chaves = Object.keys(proximo) as (keyof SettingsUpdate)[];
        const mudou = chaves.some((chave) => anterior[chave] !== proximo[chave]);
        return mudou ? proximo : anterior;
      });

      /**
       * E-932: **SO' LIMPA SE O SERVIDOR CONFIRMOU.**
       *
       * O `updateSettings` (`api.ts:487-505`) devolve `{ settings }` com o **payload do
       * servidor**. Compara-se cada campo que foi **pedido** com o que **voltou**:
       *   - se todos coincidem, o servidor guardou => a sobreposicao (e a guarda) ja' nao fazem
       *     falta e a leitura manda;
       *   - se algum **diferir** (payload parcial, proxy, endpoint so'-leitura), a sobreposicao
       *     **fica** e a guarda **fica armada** => o proximo `setSettings` **identico** e'
       *     travado e o loop nao acontece.
       *
       * *Antes desta cura o `onSuccess` limpava a guarda **sempre**, e o ciclo fechava-se.*
       */
      const guardado = await updateSettingsMutation.mutateAsync(updatedSettings);
      const pedido = updates as unknown as Record<string, unknown>;
      const devolvido = (guardado?.settings ?? {}) as unknown as Record<string, unknown>;
      const servidorConfirmou = Object.keys(pedido).every(
        (campo) => devolvido[campo] === pedido[campo],
      );
      if (servidorConfirmou) {
        limparSobreposicaoDeSettings();
      }
    },
    [settingsData?.settings, updateSettingsMutation],
  );

  return useMemo(
    () => ({
      settings,
      settingsData: settingsData?.settings,
      error,
      setSettings,
      // 🚨 E-783: `true` quando a ultima escrita NAO foi aceite pelo servidor.
      // *A UI tem de o dizer: a escolha funciona, mas **nao** fica guardada.*
      settingsNotPersisted: persistFailed,
    }),
    [settings, settingsData?.settings, error, setSettings, persistFailed],
  );
}
