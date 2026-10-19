/**
 * FASE 28.8 do adendo V3: o painel de CREDENCIAIS dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL LIGA
 *
 * A E-1270 criou a rota `credentials.status` e provou-a com 7 testes. Mas **nenhum painel a
 * consumia** -- e o defeito recorrente desta missao ja' se reabriu nove vezes assim:
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * O QUE ESTE PAINEL MOSTRA, E O QUE ELE NAO DEIXA LER
 *
 * Mostra o inventario que o servidor mediu: ficheiros, quantos foram lidos, quantos nao, certificados,
 * expirados. Para os certificados publicos mostra o sujeito e a validade -- porque o servidor os leu.
 *
 * **Nao ha' campo para o valor de um segredo.** O tipo `CredencialPublica` nao o tem, o backend nao o
 * envia, e o painel nao tem onde o guardar:
 *
 *	*Um painel que tem um campo para o segredo acaba por o receber.*
 *
 * AS TRES DECISOES QUE ESTE PAINEL TORNA VISIVEIS
 *
 *  1. **"Nao configurado" nao e' erro.** O kit e' opcional por desenho (adendo 28.13: vive fora do
 *     repositorio). Aparece uma mensagem propria, nao um alerta vermelho.
 *
 *  2. **Nao lido nao e' falha.** `conteudo_lido: false` num `.p12` e' o comportamento CORRECTO: o
 *     servidor so' fez `os.Stat`. O painel di-lo com essas palavras.
 *
 *  3. **O aviso de fronteira vem do backend e e' mostrado por inteiro.** O painel nao o resume nem o
 *     reescreve: se o backend mudar o que promete, o utilizador ve' a promessa nova.
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 * Nao escolhe o caminho do kit (o servidor e' a autoridade), nao valida cadeia, nao assina, nao
 * publica. A rota que consome e' GET.
 *
 * ACESSIBILIDADE
 *
 * Regiao com nome, tabela com cabecalhos, `aria-live` no estado de carregamento e erro em
 * `role="alert"`.
 */

import { useCallback, useEffect, useState } from "react";

import {
  descreverTipoDeCredencial,
  eSegredo,
  obterInventarioDeCredenciais,
  resumirInventario,
  tamanhoLegivelDeCredencial,
  type CredencialPublica,
  type InventarioDeCredenciais,
} from "@/lib/credentialsClient";
import { mensagemDaFalhaCom } from "../lib/mensagemDeFalha";

type Estado = "a-carregar" | "pronto" | "erro";

/** Extrai a mensagem de uma falha sem inventar uma. */
function mensagemDeFalha(falha: unknown): string {
  return mensagemDaFalhaCom(falha, "nao foi possivel ler o inventario de credenciais");
}

/** Formata a data de validade que o backend ja' devolveu como texto. */
function formatarValidade(validade: string | undefined): string {
  if (validade === undefined || validade === "") {
    return "validade nao declarada";
  }
  return validade;
}

export function CredentialsPanel() {
  const [estado, setEstado] = useState<Estado>("a-carregar");
  const [inventario, setInventario] = useState<InventarioDeCredenciais | null>(null);
  const [erro, setErro] = useState<string>("");

  const carregar = useCallback(async () => {
    setEstado("a-carregar");
    setErro("");
    try {
      setInventario(await obterInventarioDeCredenciais());
      setEstado("pronto");
    } catch (falha) {
      setInventario(null);
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  return (
    <section
      aria-labelledby="credenciais-titulo"
      data-testid="credenciais"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="credenciais-titulo" className="text-lg font-semibold">
        Credenciais e certificados
      </h2>
      <p className="mt-1 text-sm text-neutral-400">
        Inventario do kit de credenciais medido pelo servidor. Somente leitura.
      </p>

      {estado === "a-carregar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="credenciais-a-carregar">
          A medir o kit de credenciais...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="credenciais-erro">
          {erro}
        </p>
      ) : null}

      {/* "Nao configurado" nao e' erro: e' um estado do produto. */}
      {estado === "pronto" && inventario !== null && !inventario.configurado ? (
        <p className="mt-3 text-sm text-neutral-300" data-testid="credenciais-nao-configurado">
          {inventario.aviso}
        </p>
      ) : null}

      {estado === "pronto" && inventario !== null && inventario.configurado ? (
        <div className="mt-3">
          <p className="text-sm" data-testid="credenciais-resumo">
            {resumirInventario(inventario)}
          </p>

          {inventario.credenciais.length === 0 ? (
            <p className="mt-2 text-sm text-neutral-400" data-testid="credenciais-vazio">
              Kit configurado, mas nenhuma credencial encontrada.
            </p>
          ) : (
            <table className="mt-3 w-full text-left text-sm">
              <caption className="sr-only">
                Credenciais do kit, o tipo de cada uma e se o conteudo foi lido
              </caption>
              <thead>
                <tr className="border-b border-neutral-700">
                  <th scope="col" className="py-1 pr-3">
                    Ficheiro
                  </th>
                  <th scope="col" className="py-1 pr-3">
                    Tipo
                  </th>
                  <th scope="col" className="py-1 pr-3">
                    Tamanho
                  </th>
                  <th scope="col" className="py-1 pr-3">
                    Lido
                  </th>
                  <th scope="col" className="py-1">
                    Detalhe
                  </th>
                </tr>
              </thead>
              <tbody>
                {inventario.credenciais.map((credencial) => (
                  <LinhaDeCredencial key={credencial.caminho_relativo} credencial={credencial} />
                ))}
              </tbody>
            </table>
          )}

          {/* O aviso vem do backend, por inteiro: o painel nao o resume. */}
          <p
            className="mt-3 text-xs text-neutral-400"
            data-testid="credenciais-aviso-fronteira"
          >
            {inventario.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}

/** Uma linha da tabela. O detalhe distingue "lido" de "nao lido" com palavras. */
function LinhaDeCredencial({ credencial }: { credencial: CredencialPublica }) {
  const segredo = eSegredo(credencial.tipo);
  return (
    <tr className="border-b border-neutral-800 align-top">
      <td className="py-1 pr-3 font-mono text-xs">{credencial.caminho_relativo}</td>
      <td className="py-1 pr-3">{descreverTipoDeCredencial(credencial.tipo)}</td>
      <td className="py-1 pr-3">{tamanhoLegivelDeCredencial(credencial.tamanho_bytes)}</td>
      <td className="py-1 pr-3">
        {credencial.conteudo_lido ? "sim" : "nao"}
      </td>
      <td className="py-1 text-xs">
        <DetalheDaCredencial credencial={credencial} segredo={segredo} />
      </td>
    </tr>
  );
}

/**
 * O detalhe de uma credencial.
 *
 * Para um segredo, diz que o conteudo NAO foi lido -- e isso e' informacao, nao falha. Para um
 * certificado publico, mostra o sujeito e a validade que o servidor mediu.
 */
function DetalheDaCredencial({
  credencial,
  segredo,
}: {
  credencial: CredencialPublica;
  segredo: boolean;
}) {
  if (segredo) {
    return (
      <span data-testid={`credenciais-segredo-${credencial.caminho_relativo}`}>
        Segredo: conteudo NAO lido. Apenas tamanho e data foram medidos.
      </span>
    );
  }

  const partes: string[] = [];
  if (credencial.sujeito !== undefined && credencial.sujeito !== "") {
    partes.push(`sujeito: ${credencial.sujeito}`);
  }
  if (credencial.emissor !== undefined && credencial.emissor !== "") {
    partes.push(`emissor: ${credencial.emissor}`);
  }
  partes.push(formatarValidade(credencial.valido_ate));
  if (credencial.expirado) {
    partes.push("EXPIRADO");
  }

  return <span>{partes.join(" · ")}</span>;
}
