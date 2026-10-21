// FASE 26.13 / 28.4: providers de deploy configurados -- a superfície de `/deployments`.
//
// Quarta das 93 rotas do E-1319 a chegar ao produto.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// O CONTRATO, lido do servidor (`internal/agent/deploy.go:25`):
//
//	type DeployConfig struct {
//	    ID             string `json:"id"`
//	    OrganizationID string `json:"organization_id,omitempty"`
//	    Provider       string `json:"provider"`
//	    BaseURL        string `json:"base_url"`
//	    TokenEnv       string `json:"token_env,omitempty"`   // <-- NOME, nao valor
//	    ProjectID      string `json:"project_id,omitempty"`
//	    AccountID      string `json:"account_id,omitempty"`
//	    TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
//	}
//
// GET /deployments -> 200 {providers: []DeployConfig}
//
// DUAS ARMADILHAS DE SEGURANÇA NESTE CONTRATO:
//
//  1. `TokenEnv` é o NOME da variável de ambiente que contém o token. O painel
//     tem de o mostrar como REFERÊNCIA e dizer isso -- um utilizador que vê
//     "HADES_DEPLOY_TOKEN" tem de saber que está a ver o nome, não o segredo.
//
//		*Um painel que tem um campo para o segredo acaba por o receber.*
//
//  2. `BaseURL` pode conter credenciais em userinfo (`https://user:senha@host/`).
//     Um painel que mostra a URL crua mostra o segredo. A função `redigirBaseURL`
//     remove o userinfo e DIZ que o removeu, em vez de a esconder em silêncio.
//
//		*Uma credencial escondida sem aviso é uma credencial que ninguém sabe que
//		existe.*
import { agentFetch } from "@/lib/agenticClient";

export interface ProviderDeDeploy {
  id: string;
  organization_id?: string;
  provider: string;
  base_url: string;
  token_env?: string;
  project_id?: string;
  account_id?: string;
  timeout_seconds?: number;
}

export interface RespostaDeDeployments {
  providers: ProviderDeDeploy[] | null;
}

export interface URLRedigida {
  /** A URL sem userinfo, ou o valor original quando não havia nada a remover. */
  url: string;
  /** `true` quando a URL trazia credenciais e elas foram removidas. */
  removeuCredenciais: boolean;
}

/**
 * Remove `user:senha@` de uma URL e diz que o fez.
 *
 * Regressa o valor original quando não consegue analisar -- um painel que
 * apaga tudo o que não entende esconde o problema em vez de o mostrar.
 */
export function redigirBaseURL(bruta: string | undefined): URLRedigida {
  const valor = (bruta ?? "").trim();
  if (valor === "") return { url: "", removeuCredenciais: false };
  // Só URLs com esquema têm userinfo; um caminho simples não tem o que redigir.
  if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(valor)) {
    return { url: valor, removeuCredenciais: false };
  }
  try {
    const url = new URL(valor);
    const temCredenciais = url.username !== "" || url.password !== "";
    if (!temCredenciais) return { url: valor, removeuCredenciais: false };
    url.username = "";
    url.password = "";
    return { url: url.toString(), removeuCredenciais: true };
  } catch {
    // Não é analisável: não invento uma versão "limpa" que possa diferir da real.
    return { url: valor, removeuCredenciais: false };
  }
}

/**
 * Diz o que é o `token_env`, sem ambiguidade.
 *
 *	*Um campo chamado "token" que mostra o nome da variável é lido como o token
 *	por quem tem pressa -- e a pressa é o estado normal de quem opera.*
 */
export function descreverTokenEnv(tokenEnv: string | undefined): string {
  const nome = (tokenEnv ?? "").trim();
  if (nome === "") return "sem token configurado";
  return `token lido de ${nome} (nome da variável, não o valor)`;
}

/** Diz o timeout na unidade certa, ou que o provider usa a omissão do servidor. */
export function descreverTimeout(segundos: number | undefined): string {
  if (segundos === undefined || segundos === null || segundos <= 0) {
    return "timeout por omissão";
  }
  if (segundos < 60) return `${segundos} s`;
  const minutos = segundos / 60;
  return `${minutos.toFixed(minutos % 1 === 0 ? 0 : 1)} min`;
}

/** O prefixo `/api/agent/v1` é explícito por causa do gate de contrato (E-1321). */
export async function listarProvidersDeDeploy(
  init?: RequestInit,
): Promise<RespostaDeDeployments> {
  return agentFetch<RespostaDeDeployments>("/api/agent/v1/deployments", init);
}
