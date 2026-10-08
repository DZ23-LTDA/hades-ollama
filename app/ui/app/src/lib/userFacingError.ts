export type UserFacingError = {
  message: string;
  action: "retry" | "configure" | "sign_in" | "resolve_conflict" | "none";
  technical: string;
};

function rawMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  return "Falha desconhecida";
}

function redactTechnical(value: string): string {
  return value
    .replace(/(authorization\s*[:=]\s*bearer\s+)[^\s,;]+/gi, "$1[redigido]")
    .replace(/((?:token|api[_-]?key|secret|password)\s*[:=]\s*)[^\s,;]+/gi, "$1[redigido]")
    .slice(0, 300);
}

export function humanizeApiError(error: unknown, fallback = "Não foi possível concluir a operação."): UserFacingError {
  const technical = redactTechnical(rawMessage(error));
  const normalized = technical.toLowerCase();
  const status = technical.match(/\b(401|403|404|409|429|500|502|503|504)\b/)?.[1];

  if (normalized.includes("failed to fetch") || normalized.includes("networkerror") || normalized.includes("econnrefused") || normalized.includes("network request failed")) {
    return { message: "O backend está offline ou inacessível. Tente novamente quando ele estiver disponível.", action: "retry", technical };
  }
  if (status === "401") return { message: "Sua sessão expirou. Entre novamente para continuar.", action: "sign_in", technical };
  if (status === "403") return { message: "Esta ação não está autorizada para sua organização ou conta.", action: "configure", technical };
  if (status === "409") return { message: "O recurso mudou em outro lugar. Atualize os dados e tente novamente.", action: "resolve_conflict", technical };
  if (status === "429") return { message: "Há muitas tentativas neste momento. Aguarde um pouco e tente novamente.", action: "retry", technical };
  if (normalized.includes("blocked_external") || normalized.includes("blocked external")) return { message: "Essa ação depende de uma conta ou serviço externo ainda não conectado.", action: "configure", technical };
  if (normalized.includes("not_configured") || normalized.includes("not configured")) return { message: "Esse recurso ainda não está configurado. Abra as configurações para conectá-lo.", action: "configure", technical };
  if (normalized.includes("cors")) return { message: "O backend recusou esta origem. Verifique a configuração de acesso e tente novamente.", action: "configure", technical };
  if (status === "500" || status === "502" || status === "503" || status === "504") return { message: "O serviço encontrou um problema temporário. Tente novamente; seu rascunho foi preservado.", action: "retry", technical };

  return { message: `${fallback} Detalhe: ${technical}`, action: "none", technical };
}
