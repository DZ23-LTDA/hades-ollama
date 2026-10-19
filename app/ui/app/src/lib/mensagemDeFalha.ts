// # PORQUE ESTE FICHEIRO EXISTE
//
// Nove paineis do Build & Release tinham, cada um, a sua propria copia de `mensagemDeFalha`. As copias
// liam `falha.message` e mais nada -- e por isso o painel mostrava `agent_api_unavailable` (o campo
// `error` do corpo) quando o servidor tinha enviado, ao lado, o `reason` com o motivo real e o `note` a
// dizer que o Agent Center nao foi montado.
//
//	*O servidor escreveu o motivo; a interface apagou-o antes de o utilizador poder ler.* -- E-1416
//	*Nove copias da mesma funcao nao sao nove paineis: sao um painel escrito nove vezes.* -- E-1417
//
// Este modulo e' a unica copia. A precedencia e' explicita:
//
//	1. `reason` do corpo da API  -- o motivo que o servidor deu
//	2. `note` do corpo da API    -- contexto adicional, quando existe
//	3. `message` da excepcao     -- o que o cliente sabe
//	4. a alternativa do painel   -- o que sobra quando nao ha' nada
//
// O codigo (`error`) NUNCA e' o escolhido quando ha' `reason`: um codigo identifica, nao explica.

/** Devolve o motivo legivel que o corpo da API trouxe, ou "" quando nao houver nenhum. */
export function motivoDoCorpoDaApi(falha: unknown): string {
  const corpo = (falha as { corpo?: unknown } | null | undefined)?.corpo;
  if (typeof corpo !== "object" || corpo === null) return "";
  const campos = corpo as Record<string, unknown>;
  for (const chave of ["reason", "note"]) {
    const valor = campos[chave];
    if (typeof valor === "string" && valor.trim() !== "") return valor.trim();
  }
  return "";
}

/**
 * Mensagem para mostrar ao utilizador quando uma accao falha.
 *
 * @param falha       o que foi apanhado no `catch`
 * @param alternativa o texto do painel quando nao ha' mais nada a dizer
 */
export function mensagemDaFalhaCom(falha: unknown, alternativa: string): string {
  const motivo = motivoDoCorpoDaApi(falha);
  if (motivo !== "") return motivo;
  if (falha instanceof Error && falha.message.trim() !== "") return falha.message.trim();
  return alternativa;
}
