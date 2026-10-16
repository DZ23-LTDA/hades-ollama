// Testes da politica de ficheiros (`E-1040`).
//
// *Um teste que so' confirma o caminho feliz nao ve a fronteira.* Aqui testa-se sobretudo o
// que TEM de ser recusado, e cada veredicto e' discriminado pelo motivo.
import { test } from "node:test";
import assert from "node:assert/strict";

import {
  PROTOCOLO_ARQUIVOS, LIMITES_ARQUIVOS, EXECUTAVEIS, SENSIVEIS, INERTES, DIRECOES,
  extensaoDe, nomeSeguro, sha256De, classificar, decidir, resumoDaDecisao, normalizarBytes,
  impressaoDoPedido, fingerprintDe, provaValida, manifesto as montarManifesto,

} from "../src/arquivos.js";

const H64 = "a".repeat(64);

/** Bytes deterministicos, sem depender de crypto nos testes de forma. */
const bytes = (n) => new Uint8Array(n).fill(7);

test("1) o protocolo e' versionado e as listas sao as que dizem ser", () => {
  assert.equal(PROTOCOLO_ARQUIVOS, 1);
  assert.deepEqual([...DIRECOES], ["download", "upload"]);
  assert.ok(EXECUTAVEIS.includes("exe") && EXECUTAVEIS.includes("sh"));
  assert.ok(SENSIVEIS.includes("pem") && SENSIVEIS.includes("kdbx"));
  assert.ok(INERTES.includes("txt") && INERTES.includes("png"));
  // as tres listas nao podem partilhar extensoes -- senao a classificacao e' ambigua
  const inter = EXECUTAVEIS.filter((e) => SENSIVEIS.includes(e) || INERTES.includes(e));
  assert.deepEqual(inter, [], "listas sobrepostas: " + inter.join(","));
});

test("2) `extensaoDe` nao se engana com caminhos, pontos e maiusculas", () => {
  assert.equal(extensaoDe("a.txt"), "txt");
  assert.equal(extensaoDe("A.TXT"), "txt");
  assert.equal(extensaoDe("C:\\x\\y\\a.exe"), "exe");
  assert.equal(extensaoDe("/x/y/a.tar.gz"), "gz");
  assert.equal(extensaoDe("sem_extensao"), "");
  assert.equal(extensaoDe(".oculto"), "", "um ficheiro so' com ponto nao tem extensao");
  assert.equal(extensaoDe("termina."), "");
});

test("3) `nomeSeguro` RECUSA caminho -- nao normaliza", () => {
  assert.equal(nomeSeguro("a.txt").ok, true);
  // *Um caminho absoluto num manifesto e' um pedido para escrever fora do combinado.*
  assert.equal(nomeSeguro("C:\\Users\\x\\a.exe").motivo, "nome_tem_caminho");
  assert.equal(nomeSeguro("/etc/passwd").motivo, "nome_tem_caminho");
  assert.equal(nomeSeguro("../../a.txt").motivo, "nome_tem_caminho");
  assert.equal(nomeSeguro("..").motivo, "nome_travessia");
  assert.equal(nomeSeguro("").motivo, "nome_vazio");
  assert.equal(nomeSeguro("   ").motivo, "nome_vazio");
  assert.equal(nomeSeguro("a".repeat(300)).motivo, "nome_longo");
  assert.equal(nomeSeguro("CON.txt").motivo, "nome_reservado_windows");
  assert.equal(nomeSeguro("LPT1").motivo, "nome_reservado_windows");
  assert.equal(nomeSeguro("a\u0000b.txt").motivo, "nome_caracteres_invalidos");
  // `a:b.txt` NAO e' um caso de caracteres invalidos: no Windows e' o ficheiro `b.txt` na
  // unidade `A:`. *O teste estava errado, nao o codigo* -- e a ordem das regras e' que decide.
  assert.equal(nomeSeguro("a:b.txt").motivo, "nome_tem_unidade");
  assert.equal(nomeSeguro("a*b.txt").motivo, "nome_caracteres_invalidos");
  assert.equal(nomeSeguro("a?b.txt").motivo, "nome_caracteres_invalidos");
  assert.equal(nomeSeguro("a<b.txt").motivo, "nome_caracteres_invalidos");
});

test("4) `sha256De` mede os BYTES, e o mesmo conteudo da' o mesmo hash", async () => {
  const a = await sha256De(bytes(10));
  const b = await sha256De(bytes(10));
  assert.equal(a.ok, true);
  assert.equal(a.sha256, b.sha256, "conteudo igual, hash igual");
  assert.equal(a.bytes, 10);
  const c = await sha256De(bytes(11));
  assert.notEqual(a.sha256, c.sha256, "conteudo diferente, hash diferente");
  // e um byte diferente muda tudo
  const d1 = new Uint8Array([1, 2, 3]);
  const d2 = new Uint8Array([1, 2, 4]);
  assert.notEqual((await sha256De(d1)).sha256, (await sha256De(d2)).sha256);
  assert.match(a.sha256, /^[a-f0-9]{64}$/);
});

test("5) `sha256De` recusa o que nao pode medir", async () => {
  assert.equal((await sha256De(null)).motivo, "bytes_ausentes");
  assert.equal((await sha256De("texto")).motivo, "bytes_ausentes");
  assert.equal((await sha256De(123)).motivo, "bytes_ausentes");
  // `"texto"` nao e' bytes e nao passa. E um objecto com uma chave NAO numerica tambem nao:
  // *a representacao JSON do canal so' pode ter chaves "0","1","2",...*
  assert.equal((await sha256De({ byteLength: 5 })).motivo, "bytes_invalidos");
  assert.equal((await sha256De({ a: 1 })).motivo, "bytes_invalidos");
  assert.equal((await sha256De({})).motivo, "bytes_ausentes");
  assert.equal((await sha256De({ 0: 300 })).motivo, "bytes_invalidos", "300 nao e' um octeto");
  assert.equal((await sha256De({ 0: -1 })).motivo, "bytes_invalidos");
  assert.equal((await sha256De({ 0: 1.5 })).motivo, "bytes_invalidos");
  // O LIMITE e' aplicado sobre o tamanho REAL: nao se pode forcar `byteLength` num TypedArray
  // verdadeiro -- *um teste que mente sobre o tamanho nao mede o limite, mede a mentira.* O
  // caminho em que o limite e' accionavel e' o do tamanho DECLARADO (o do manifesto).
  assert.equal(classificar("a.txt", LIMITES_ARQUIVOS.ficheiroBytes + 1).motivo, "ficheiro_demasiado_grande");
  assert.equal(classificar("a.txt", LIMITES_ARQUIVOS.ficheiroBytes).ok, true, "o limite e' inclusivo");
});

test("6) `classificar` separa executavel de sensivel de inerte de desconhecido", () => {
  assert.equal(classificar("a.exe", 10).risco, "executavel");
  assert.equal(classificar("a.PEM", 10).risco, "sensivel");
  assert.equal(classificar("a.png", 10).risco, "inerte");
  assert.equal(classificar("a.zzz", 10).risco, "desconhecido");
  assert.equal(classificar("a.txt", 10).tamanho, 10);
  assert.equal(classificar("a.txt", null).motivo, "tamanho_ausente");
  assert.equal(classificar("a.txt", -1).motivo, "tamanho_ausente");
  assert.equal(classificar("a.txt", LIMITES_ARQUIVOS.ficheiroBytes + 1).motivo, "ficheiro_demasiado_grande");
});

test("7) SEM `sha256` nada e' aprovado -- *nao se aprova o que nao se mediu*", () => {
  const sem = decidir({ nome: "a.txt", direcao: "download", tamanho: 10 });
  assert.equal(sem.veredicto, "proibido");
  assert.equal(sem.motivo, "sha256_ausente_ou_invalido");
  const curto = decidir({ nome: "a.txt", direcao: "download", tamanho: 10, sha256: "abc" });
  assert.equal(curto.veredicto, "proibido");
  const naoHex = decidir({ nome: "a.txt", direcao: "download", tamanho: 10, sha256: "z".repeat(64) });
  assert.equal(naoHex.veredicto, "proibido");
});

test("8) direccao invalida e' proibida, e nao ha' terceira direccao", () => {
  assert.equal(decidir({ nome: "a.txt", direcao: "lateral", tamanho: 1, sha256: H64 }).motivo, "direccao_invalida");
  assert.equal(decidir({ nome: "a.txt", direcao: "", tamanho: 1, sha256: H64 }).motivo, "direccao_invalida");
  assert.equal(decidir({ direcao: "download", tamanho: 1, sha256: H64 }).motivo, "nome_vazio");
});

test("9) DOWNLOAD inerte e' permitido; o resto NAO", () => {
  const ok = decidir({ nome: "relatorio.csv", direcao: "download", tamanho: 100, sha256: H64 });
  assert.equal(ok.veredicto, "permitido");
  assert.equal(ok.motivo, "conteudo_inerte_em_download");
  assert.equal(ok.risco, "inerte");

  // *Um executavel nunca fica "permitido" -- nem com aprovacao.*
  const exe = decidir({ nome: "setup.exe", direcao: "download", tamanho: 100, sha256: H64 });
  assert.equal(exe.veredicto, "exige_aprovacao");
  assert.equal(exe.motivo, "executavel_exige_aprovacao");
  // *nao ha' `aprovado: false` a confirmar: nao ha' aprovacao nenhuma.*
  assert.equal(exe.aprovacao, "aprovacao_ausente");
  assert.equal("aprovado" in exe, false, "o campo booleano desapareceu do contrato");

  // `E-1041`: `{ aprovado: true }` JA' NAO BASTA -- *um booleano solto diz que alguem
  // afirmou; nao diz quem, nem para que pedido.*
  const solto = decidir({ nome: "setup.exe", direcao: "download", tamanho: 100, sha256: H64 }, { aprovado: true });
  assert.equal(solto.veredicto, "exige_aprovacao", "um booleano solto nao autoriza");
  assert.equal(solto.aprovacao, "booleano_solto_nao_e_aprovacao");
});

test("10) DOWNLOAD sensivel e desconhecido exigem aprovacao", () => {
  const pem = decidir({ nome: "chave.pem", direcao: "download", tamanho: 10, sha256: H64 });
  assert.equal(pem.veredicto, "exige_aprovacao");
  assert.equal(pem.motivo, "ficheiro_sensivel_exige_aprovacao");

  // *Nao ha' lista completa de perigo* -- o desconhecido nao passa por omissao.
  const zzz = decidir({ nome: "a.zzz", direcao: "download", tamanho: 10, sha256: H64 });
  assert.equal(zzz.veredicto, "exige_aprovacao");
  assert.equal(zzz.motivo, "extensao_desconhecida_exige_aprovacao");
});

test("11) UPLOAD de QUALQUER ficheiro exige aprovacao -- mesmo inerte", () => {
  // *Sair do disco e' sempre um acto.* Um `a.txt` inerte em download e' permitido; em upload
  // nao e'.
  const dl = decidir({ nome: "a.txt", direcao: "download", tamanho: 10, sha256: H64 });
  const up = decidir({ nome: "a.txt", direcao: "upload", tamanho: 10, sha256: H64 });
  assert.equal(dl.veredicto, "permitido");
  assert.equal(up.veredicto, "exige_aprovacao");
  assert.equal(up.motivo, "upload_exige_aprovacao");
  assert.notEqual(dl.veredicto, up.veredicto, "a direccao muda a decisao");
});

test("12) a ORDEM das regras e' a politica -- o executavel vem antes do upload", () => {
  const upExe = decidir({ nome: "a.exe", direcao: "upload", tamanho: 10, sha256: H64 });
  assert.equal(upExe.motivo, "executavel_exige_aprovacao", "o risco manda sobre a direccao");
});

test("13) `manifesto` monta a forma canonica -- com FINGERPRINT -- e recusa o que nao caiba", async () => {
  const m = await montarManifesto({ nome: "a.txt", direcao: "download", bytes: bytes(5), sha256: H64, missao: "M-1", origem: "https://x" });
  assert.equal(m.ok, true);
  assert.equal(m.manifesto.protocolo, 1);
  assert.equal(m.manifesto.tamanho, 5);
  assert.equal(m.manifesto.extensao, "txt");
  assert.equal(m.manifesto.missao, "M-1");
  assert.match(m.manifesto.fingerprint, /^[a-f0-9]{64}$/, "o manifesto traz o fingerprint");
  // *um manifesto sem hash MEDIDO nao e' um manifesto: e' uma descricao.*
  assert.equal((await montarManifesto({ nome: "a.txt", direcao: "download", bytes: bytes(5) })).motivo, "sha256_ausente_ou_invalido");
  assert.equal((await montarManifesto({ nome: "../a.txt", direcao: "download", bytes: bytes(5), sha256: H64 })).motivo, "nome_tem_caminho");
  // um objecto que nao e' uma sequencia de octetos e' recusado na normalizacao
  assert.equal((await montarManifesto({ nome: "a.txt", direcao: "download", bytes: { byteLength: 5 }, sha256: H64 })).motivo, "bytes_invalidos");
  assert.equal((await montarManifesto({ nome: "a.txt", direcao: "download", bytes: null, sha256: H64 })).motivo, "bytes_ausentes");
  // e o LIMITE e' verificado sobre o tamanho real dos bytes normalizados
  const grande = new Uint8Array(LIMITES_ARQUIVOS.ficheiroBytes + 1);
  assert.equal((await montarManifesto({ nome: "a.txt", direcao: "download", bytes: grande, sha256: H64 })).motivo, "ficheiro_demasiado_grande");
});

test("14) `resumoDaDecisao` e' curto e nao leva o hash", () => {
  const d = decidir({ nome: "a.exe", direcao: "download", tamanho: 10, sha256: H64 });
  const r = resumoDaDecisao(d);
  assert.equal(r, "exige_aprovacao / executavel / download / exe");
  assert.equal(r.includes(H64), false, "o resumo nao leva o hash");
  assert.equal(resumoDaDecisao(null), "sem decisao");
});

test("15) um manifesto HONESTO ponta a ponta: hash medido, decisao coerente", async () => {
  const conteudo = bytes(12);
  const h = await sha256De(conteudo);
  assert.equal(h.ok, true);
  const m = await montarManifesto({ nome: "dados.csv", direcao: "download", bytes: conteudo, sha256: h.sha256, missao: "M-9" });
  assert.equal(m.ok, true);
  const d = decidir(m.manifesto);
  assert.equal(d.veredicto, "permitido");
  assert.equal(d.sha256, h.sha256);
  assert.equal(d.tamanho, 12);

  // e se os BYTES mudarem, o hash declarado deixa de corresponder -- o que a decisao NAO
  // verifica sozinha, porque `decidir` nao tem os bytes. *Quem decide nao re-mede: quem pede
  // e' que mede, e quem aprova compara.*
  const outros = await sha256De(bytes(13));
  assert.notEqual(outros.sha256, h.sha256);
});

test("16) `normalizarBytes` aceita as TRES formas -- e recusa o que nao e' octeto", async () => {
  // *Medido no E2E: o `sendMessage` serializa em JSON, e um `Uint8Array` atravessa como
  // objecto de chaves numericas, SEM `byteLength`.* O canal NAO transporta binario -- logo a
  // medicao tem de acontecer onde os bytes existem.
  const t = new Uint8Array([1, 2, 3, 250]);
  const a = await sha256De(t);
  const b = await sha256De(t.buffer);
  const c = await sha256De({ 0: 1, 1: 2, 2: 3, 3: 250 });      // a forma do JSON
  const d = await sha256De([1, 2, 3, 250]);                     // array simples
  assert.equal(a.ok, true);
  assert.equal(a.sha256, b.sha256, "TypedArray e ArrayBuffer dao o mesmo");
  assert.equal(a.sha256, c.sha256, "a representacao JSON da' o mesmo hash");
  assert.equal(a.sha256, d.sha256, "o array simples da' o mesmo hash");
  assert.equal(a.bytes, 4);
  // e um byte diferente muda o hash, venha de onde vier
  assert.notEqual(a.sha256, (await sha256De([1, 2, 3, 251])).sha256);
  // um sub-array respeita o offset -- *nao hashear o buffer inteiro por engano*
  const grande = new Uint8Array([9, 9, 1, 2, 3, 250, 9]);
  const vista = grande.subarray(2, 6);
  assert.equal((await sha256De(vista)).sha256, a.sha256, "respeita byteOffset/byteLength");
});

test("17) o canal NAO transporta binario -- e isso e' uma propriedade, nao um defeito", () => {
  // Uma consequencia de seguranca que vale a pena fixar: o conteudo de um ficheiro nao passa
  // por `sendMessage`. O que atravessa sao METADADOS e a representacao dos bytes.
  const json = JSON.stringify(new Uint8Array([1, 2, 3]));
  assert.equal(json, '{"0":1,"1":2,"2":3}', "e' isto que atravessa");
  // e um Uint8Array reconstruido da forma JSON NAO e' um TypedArray
  const voltou = JSON.parse(json);
  assert.equal(voltou instanceof Uint8Array, false);
  assert.equal(voltou.byteLength, undefined, "sem byteLength: a primeira versao falhava aqui");
  assert.equal(Array.isArray(voltou), false);
});

test("18) `impressaoDoPedido` cobre o que a decisao decide", () => {
  // *Uma aprovacao para `a.csv` de 12 bytes nao pode servir para `b.exe` de 2 MB.*
  const a = impressaoDoPedido({ direcao: "download", nome: "a.csv", tamanho: 12, sha256: H64 });
  const b = impressaoDoPedido({ direcao: "download", nome: "b.csv", tamanho: 12, sha256: H64 });
  const c = impressaoDoPedido({ direcao: "download", nome: "a.csv", tamanho: 13, sha256: H64 });
  const d = impressaoDoPedido({ direcao: "download", nome: "a.csv", tamanho: 12, sha256: "b".repeat(64) });
  const e = impressaoDoPedido({ direcao: "upload", nome: "a.csv", tamanho: 12, sha256: H64 });
  const iguais = new Set([a, b, c, d, e]);
  assert.equal(iguais.size, 5, "cada campo muda a impressao");
  assert.equal(a, impressaoDoPedido({ direcao: "download", nome: "a.csv", tamanho: 12, sha256: H64 }),
    "e a mesma impressao para o mesmo pedido");
});

test("19) `provaValida` exige os TRES campos -- id, fingerprint e autor", () => {
  const fp = "a".repeat(64);
  assert.equal(provaValida({ aprovacaoId: "AP-1", fingerprint: fp, aprovadoPor: "LMPrado-DZ23" }).ok, true);
  assert.equal(provaValida(null).motivo, "aprovacao_ausente");
  assert.equal(provaValida({}).motivo, "aprovacao_sem_id");
  assert.equal(provaValida({ aprovacaoId: "AP-1" }).motivo, "aprovacao_sem_fingerprint");
  assert.equal(provaValida({ aprovacaoId: "AP-1", fingerprint: "abc" }).motivo, "aprovacao_sem_fingerprint");
  assert.equal(provaValida({ aprovacaoId: "AP-1", fingerprint: fp }).motivo, "aprovacao_sem_actor");
  assert.equal(provaValida({ aprovacaoId: "  ", fingerprint: fp, aprovadoPor: "x" }).motivo, "aprovacao_sem_id");
  // *uma aprovacao sem autor nao tem responsavel.*
  assert.equal(provaValida({ aprovacaoId: "AP-1", fingerprint: fp, aprovadoPor: " " }).motivo, "aprovacao_sem_actor");
});

test("20) a PROVA autoriza -- mas so' para o pedido EXACTO", async () => {
  const pedido = { nome: "setup.exe", direcao: "download", tamanho: 100, sha256: H64 };
  const fp = await fingerprintDe(pedido);
  assert.equal(fp.ok, true);
  const manifesto = { ...pedido, fingerprint: fp.fingerprint };

  // sem prova nenhuma
  const sem = decidir(manifesto);
  assert.equal(sem.veredicto, "exige_aprovacao");
  assert.equal(sem.aprovacao, "aprovacao_ausente");

  // prova do pedido CERTO
  const boa = decidir(manifesto, { prova: { aprovacaoId: "AP-1", fingerprint: fp.fingerprint, aprovadoPor: "humano" } });
  assert.equal(boa.veredicto, "autorizado", JSON.stringify(boa));
  assert.equal(boa.motivo, "executavel_exige_aprovacao_com_aprovacao");
  assert.equal(boa.aprovacaoId, "AP-1");
  assert.equal(boa.aprovadoPor, "humano");

  // prova de OUTRO pedido -- o fingerprint nao casa
  const outro = await fingerprintDe({ nome: "outro.csv", direcao: "download", tamanho: 1, sha256: H64 });
  const cruzada = decidir(manifesto, { prova: { aprovacaoId: "AP-2", fingerprint: outro.fingerprint, aprovadoPor: "humano" } });
  assert.equal(cruzada.veredicto, "exige_aprovacao", "uma aprovacao para outro pedido nao serve");
  assert.equal(cruzada.aprovacao, "fingerprint_nao_corresponde");

  // prova sem fingerprint do pedido
  const semFp = decidir({ ...pedido }, { prova: { aprovacaoId: "AP-3", fingerprint: fp.fingerprint, aprovadoPor: "h" } });
  assert.equal(semFp.veredicto, "exige_aprovacao");
  assert.equal(semFp.aprovacao, "fingerprint_do_pedido_ausente");
});

test("21) a prova tambem autoriza upload e ficheiro sensivel -- com o fingerprint certo", async () => {
  for (const caso of [
    { nome: "a.txt", direcao: "upload", tamanho: 10 },
    { nome: "chave.pem", direcao: "download", tamanho: 10 },
    { nome: "a.zzz", direcao: "download", tamanho: 10 },
  ]) {
    const pedido = { ...caso, sha256: H64 };
    const fp = await fingerprintDe(pedido);
    const manifesto = { ...pedido, fingerprint: fp.fingerprint };
    assert.equal(decidir(manifesto).veredicto, "exige_aprovacao", caso.nome + " sem prova");
    const com = decidir(manifesto, { prova: { aprovacaoId: "AP-9", fingerprint: fp.fingerprint, aprovadoPor: "h" } });
    assert.equal(com.veredicto, "autorizado", caso.nome + " com prova: " + JSON.stringify(com));
  }
});

test("22) inerte em download e' PERMITIDO e nao precisa de prova nenhuma", async () => {
  const pedido = { nome: "a.csv", direcao: "download", tamanho: 10, sha256: H64 };
  const fp = await fingerprintDe(pedido);
  const d = decidir({ ...pedido, fingerprint: fp.fingerprint });
  assert.equal(d.veredicto, "permitido");
  // e uma prova INVALIDA nao estraga um caso que ja' era permitido -- mas tambem nao o muda
  const d2 = decidir({ ...pedido, fingerprint: fp.fingerprint }, { prova: { aprovacaoId: "" } });
  assert.equal(d2.veredicto, "permitido");
});

test("23) `fingerprintDe` e' deterministico e sensivel ao pedido", async () => {
  const a = await fingerprintDe({ direcao: "download", nome: "a.csv", tamanho: 1, sha256: H64 });
  const b = await fingerprintDe({ direcao: "download", nome: "a.csv", tamanho: 1, sha256: H64 });
  const c = await fingerprintDe({ direcao: "download", nome: "a.csv", tamanho: 2, sha256: H64 });
  assert.equal(a.fingerprint, b.fingerprint);
  assert.notEqual(a.fingerprint, c.fingerprint);
  assert.match(a.fingerprint, /^[a-f0-9]{64}$/);
  assert.equal(typeof a.impressao, "string");
  assert.ok(a.impressao.startsWith("v1|download|a.csv|1|"));
});
