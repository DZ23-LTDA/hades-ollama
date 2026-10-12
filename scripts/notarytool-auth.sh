#!/bin/sh

# Autenticação do `notarytool` — arquivo FONTEÁVEL (sem efeito colateral no topo).
#
# POR QUE EXISTE
# -------------
# O projeto notarizava só por **Apple ID + senha de app** (`--apple-id --password --team-id`). O App Store Connect
# também aceita uma **chave de API** (`.p8`), que não depende de senha de app e é o método recomendado pela Apple:
# `--key <arquivo.p8> --key-id <KEYID> --issuer <ISSUER-UUID>`.
#
# A CHAVE PRIVADA NUNCA ENTRA NO REPOSITÓRIO. O que circula aqui é o **caminho** do arquivo (`APPLE_API_KEY`); o
# conteúdo é materializado em tempo de execução, num diretório temporário, a partir de um segredo do CI, com permissão
# restrita e nunca impresso no log.
#
# PRECEDÊNCIA
# -----------
# Se as três variáveis da chave de API estiverem preenchidas, elas ganham; senão, cai no fluxo por Apple ID (o que já
# existia, para não quebrar quem não configurou a chave).
#
# O QUE A CHAVE **NÃO** FAZ
# -------------------------
# Ela autentica a **submissão** ao serviço de notarização. Ela **não assina** o binário: assinar continua exigindo o
# certificado **Developer ID Application** (`APPLE_IDENTITY`). E notarização só roda em **macOS** (`xcrun notarytool`).

notarytool_auth_mode() {
	if [ -n "$APPLE_API_KEY" ] && [ -n "$APPLE_API_KEY_ID" ] && [ -n "$APPLE_API_ISSUER" ]; then
		echo "api-key"
		return 0
	fi
	echo "apple-id"
}

# notarytool_auth_args imprime os argumentos de autenticação, um por linha, para uso com expansão de palavra:
#
#   $(xcrun -f notarytool) submit arquivo.zip --wait $notary_args
#
# O caminho da chave não pode conter espaços (o CI usa o diretório temporário do runner, que não tem).
notarytool_auth_args() {
	if [ "$(notarytool_auth_mode)" = "api-key" ]; then
		printf '%s\n' --key "$APPLE_API_KEY" --key-id "$APPLE_API_KEY_ID" --issuer "$APPLE_API_ISSUER"
		return 0
	fi
	printf '%s\n' --apple-id "$APPLE_ID" --password "$APPLE_PASSWORD" --team-id "$APPLE_TEAM_ID"
}
