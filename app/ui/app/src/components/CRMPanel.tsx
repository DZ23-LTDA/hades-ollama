// Painel de CRM -- contactos e funil, com PII mascarada por omissão.
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - NÃO mostra telefone nem email completos. A lista mascara por omissão e diz
//     que mascarou; há um botão para mostrar, e ele diz que está a mostrar.
//   - NÃO mostra centavos como reais.
//   - NÃO trata CRM ausente (503) como lista vazia.
//   - NÃO inventa o estado de consentimento: mostra o `blocked_channels` como o
//     servidor o devolve, e diz quando não veio.
//   - NÃO cria, edita nem concede/revoga consentimento: só lê.
import { useCallback, useEffect, useState } from "react";
import {
  CRMAusenteError,
  descreverCanaisBloqueados,
  descreverEtapa,
  descreverValor,
  funilDeCRM,
  listarContactos,
  mascararEmail,
  mascararTelefone,
  type ContactoDeCRM,
  type FunilDeCRM,
} from "@/lib/crmClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-crm" | "erro";

export function CRMPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [contactos, setContactos] = useState<ContactoDeCRM[]>([]);
  const [funil, setFunil] = useState<FunilDeCRM | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [filtro, setFiltro] = useState("");
  const [mostrarPII, setMostrarPII] = useState(false);

  const carregar = useCallback(async (consulta: string) => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [rContactos, rFunil] = await Promise.all([
        listarContactos(consulta),
        funilDeCRM(),
      ]);
      setContactos(rContactos.contacts ?? []);
      setFunil(rFunil);
      setEstado("pronta");
    } catch (causa) {
      setContactos([]);
      setFunil(null);
      if (causa instanceof CRMAusenteError) {
        setErro(causa.message);
        setEstado("sem-crm");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível ler o CRM.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar("");
  }, [carregar]);

  return (
    <section
      data-testid="crm"
      aria-labelledby="crm-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="crm-titulo" className="text-sm font-semibold text-neutral-100">
          CRM
        </h3>
        <button
          type="button"
          data-testid="crm-actualizar"
          onClick={() => void carregar(filtro)}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      <div className="mb-3 flex flex-wrap items-end gap-2">
        <label className="flex flex-1 flex-col gap-1 text-xs text-neutral-300">
          Procurar contacto
          <input
            type="text"
            data-testid="crm-filtro"
            value={filtro}
            onChange={(evento) => setFiltro(evento.target.value)}
            placeholder="nome, telefone ou email"
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>
        <button
          type="button"
          data-testid="crm-procurar"
          onClick={() => void carregar(filtro)}
          className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800"
        >
          Procurar
        </button>
      </div>

      {estado === "sem-crm" && (
        <div data-testid="crm-sem-crm" className="text-xs text-amber-300">
          <p className="font-semibold">O CRM não está ligado neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há contactos»: é «não há CRM». Uma lista vazia aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="crm-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="crm-a-carregar" className="text-xs text-neutral-400">
          A ler o CRM...
        </p>
      )}

      {estado === "pronta" && funil && (
        <>
          <p data-testid="crm-funil" className="mb-2 text-xs text-neutral-300">
            {funil.open_count} oportunidade(s) aberta(s) ·{" "}
            <span className="text-emerald-400">
              {descreverValor(funil.open_value_cents, funil.currency)}
            </span>{" "}
            em aberto · {descreverValor(funil.won_value_cents, funil.currency)} ganho ·{" "}
            {funil.lost_count} perdida(s)
          </p>

          {funil.stages && funil.stages.length > 0 && (
            <p data-testid="crm-etapas" className="mb-3 text-[11px] text-neutral-400">
              {funil.stages
                .map(
                  (etapa) =>
                    `${descreverEtapa(etapa.stage)}: ${etapa.count} (${descreverValor(etapa.total_cents, funil.currency)})`,
                )
                .join(" · ")}
            </p>
          )}

          <div className="mb-2 flex items-center justify-between gap-2">
            <h4 className="text-xs font-semibold text-neutral-300">
              Contactos ({contactos.length})
            </h4>
            <label className="flex items-center gap-2 text-[11px] text-neutral-400">
              <input
                type="checkbox"
                data-testid="crm-mostrar-pii"
                checked={mostrarPII}
                onChange={(evento) => setMostrarPII(evento.target.checked)}
              />
              Mostrar dados pessoais
            </label>
          </div>

          {!mostrarPII && (
            <p data-testid="crm-aviso-pii" className="mb-2 text-[11px] text-amber-300">
              Telefone e email estão mascarados. Uma lista de contactos que mostra os contactos todos
              não é uma lista: é um relatório que se pode fotografar.
            </p>
          )}

          {contactos.length === 0 ? (
            <p data-testid="crm-sem-contactos" className="text-xs text-neutral-400">
              {filtro.trim() !== ""
                ? `Nenhum contacto corresponde a «${filtro.trim()}».`
                : "O CRM está ligado e não há contactos nesta organização."}
            </p>
          ) : (
            <table data-testid="crm-contactos" className="w-full text-left text-xs text-neutral-200">
              <thead className="text-neutral-400">
                <tr>
                  <th scope="col" className="py-1 pr-2">nome</th>
                  <th scope="col" className="py-1 pr-2">telefone</th>
                  <th scope="col" className="py-1 pr-2">email</th>
                </tr>
              </thead>
              <tbody>
                {contactos.map((contacto) => (
                  <tr
                    key={contacto.id}
                    data-testid={`crm-contacto-${contacto.id}`}
                    className="border-t border-neutral-800"
                  >
                    <td className="py-1 pr-2">{contacto.name}</td>
                    <td className="py-1 pr-2 font-mono text-[11px] text-neutral-400">
                      {mostrarPII
                        ? contacto.phone || "—"
                        : mascararTelefone(contacto.phone)}
                    </td>
                    <td className="py-1 pr-2 font-mono text-[11px] text-neutral-400">
                      {mostrarPII
                        ? contacto.email || "—"
                        : mascararEmail(contacto.email)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          <p data-testid="crm-consentimento" className="mt-3 text-[11px] text-neutral-500">
            O estado de consentimento por canal vem do registro de consentimento, não do contacto:{" "}
            {descreverCanaisBloqueados(undefined)}.
          </p>
        </>
      )}

      <p data-testid="crm-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê contactos e o funil. Criar contacto, criar oportunidade, mudar de etapa e
        conceder ou revogar consentimento continuam a ser ações separadas — e os valores vêm em reais
        a partir dos cêntimos que o servidor envia.
      </p>
    </section>
  );
}
