"""Empacota a extensao HADES Browser Bridge: .zip nao assinado + SHA256SUMS + LEIA-ME.

Porque um script e nao 'Compress-Archive' nem ZipFile.CreateFromDirectory: no
Windows o .NET escreve as entradas com '\\' (medido: 8 de 9 entradas), e a spec do
ZIP -- e o Chrome/Edge -- exige '/'. Um zip com 'icons\\hades-icon-192.png' NAO
resolve 'icons/hades-icon-192.png' do manifest: parece bem (9 entradas, 84 KB,
Test-Path verde) e NAO CARREGA. A unica verificacao que apanha isto e' abrir o zip
e testar cada caminho que o manifest referencia.

Uso:
    python -X utf8 scripts/hades_browser_bridge_package.py            # empacota e verifica
    python -X utf8 scripts/hades_browser_bridge_package.py --check    # so' verifica

Exit 0 = pacote integro e verificavel. Exit 1 = defeito (imprime o que falhou).
"""
import hashlib
import io
import json
import os
import subprocess
import sys
import zipfile
from pathlib import Path

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SRC = os.path.join(RAIZ, 'apps', 'hades-browser-bridge')
OUT = os.path.join(RAIZ, 'dist', 'browser-bridge')

# So' o que o navegador carrega. `test/` fica de fora de proposito: um pacote de
# distribuicao nao leva o arnes de teste.
INCLUIR = ['manifest.json', 'src', 'popup', 'icons']

def _source_sha() -> str:
    """O SHA do commit de onde este pacote saiu. *Sem ele, o pacote nao diz de que fonte veio.*"""
    try:
        r = subprocess.run(["git", "rev-parse", "--short=12", "HEAD"],
                           cwd=str(Path(__file__).resolve().parent.parent),
                           capture_output=True, text=True, timeout=20)
        if r.returncode == 0 and r.stdout.strip():
            return r.stdout.strip()
    except (OSError, subprocess.SubprocessError):
        pass
    return "INDISPONIVEL (git indisponivel no ambiente)"


LEIA_ME = """LEIA-ME -- HADES Browser Bridge (extensao de desenvolvimento, NAO PUBLICADA)

  Ficheiro : {zip}
  Versao   : {versao}
  SHA-256  : {sha}
  SOURCE_SHA: {source_sha}
  Assinado : NAO. Este pacote e' de DESENVOLVIMENTO e nao esta' assinado.
  Publicado: NAO. Nao foi enviado para a Chrome Web Store, Edge Add-ons,
             Firefox Add-ons nem para nenhuma loja. Nao distribuir.

O QUE E'
  Extensao WebExtension Manifest V3 que liga ao HADES apenas as abas que o
  utilizador autorizar. Reutiliza o contrato do runtime (`browser.operator`):
  os scopes `browser:navigate`, `browser:files` e `browser:takeover`, a
  exigencia de aprovacao e o `session_id` da missao.

O QUE **NAO** FAZ
  - nao le campos de senha, cartao ou MFA: nesses campos o controlo e' do utilizador;
  - nao decide aprovacoes: quem decide e' o HADES;
  - nao tem acesso ao sistema operativo nem shell;
  - nao aceita ligacoes de rede publica nem LAN: so' loopback (127.0.0.1, localhost);
  - nao funciona em modo anonimo/incognito (desligado no proprio manifest).

INSTALAR (Chrome ou Edge)
  1. Extraia este .zip para uma pasta sua (por exemplo: Documentos\\hades-bridge).
  2. Abra chrome://extensions  (ou  edge://extensions).
  3. Ligue o "Modo de programador" (canto superior direito).
  4. Clique "Carregar sem compactacao" e escolha a pasta extraida.
  5. A extensao aparece como "HADES Browser Bridge" versao {versao}.

USAR
  1. Abra a pagina que quer ligar e clique no icone da extensao.
  2. Clique "Conectar esta aba". O badge fica "ON" enquanto durar a concessao.
     A concessao expira sozinha em 30 minutos e e' revogada ao:
       - navegar para outra origem na mesma aba;
       - fechar a aba;
       - clicar "Desconectar" ou "Revogar todas as abas".
  3. "Revogar todas as abas" apaga todas as concessoes de uma vez.

REMOVER (sem deixar rasto)
  1. Clique "Revogar todas as abas" (apaga as concessoes).
  2. chrome://extensions -> HADES Browser Bridge -> "Remover".
  3. Apague a pasta que extraiu. Nada do HADES vive fora dela.
  A extensao NAO escreve em disco nem apaga projetos do utilizador.

LIMITES HONESTOS DESTE PACOTE
  - NAO HA PAIRING: o `native messaging` e o `HADES Local Companion` ainda nao
    existem. O popup mostra "host: - (native host nao instalado)" e os campos
    "missao" e "aprovacao" ficam vazios, porque nada os alimenta.
  - NAO foi carregada num Chrome/Edge real durante a construcao: o que esta
    provado sao 24 testes automaticos, nao um E2E de navegador.
  - Sem Firefox, sem Safari e sem .xpi.
  - As 9 entradas do .zip tem separador "/" (exigencia da spec do ZIP): um zip
    com "\\" parece igual e NAO carrega.

REPRODUZIR
  node --test apps/hades-browser-bridge/test/protocol.test.js
  node --test apps/hades-browser-bridge/test/carregavel.test.js
  python -X utf8 scripts/hades_browser_bridge_package.py
"""


def separadores_crus(zip_path):
    """Le os bytes CRUS do diretorio central do ZIP e devolve os nomes com '\\'.

    Porque `zipfile.ZipFile(...).namelist()` NORMALIZA '\\' para '/' ao ler: um
    zip com entradas 'icons\\x.png' chega ao namelist() como 'icons/x.png'. Logo a
    regra `'\\' in n` escrita sobre o namelist() NUNCA dispara -- foi medido nesta
    sessao. E' preciso ler o diretorio central a mao (assinatura PK\\x01\\x02).
    """
    dados = io.open(zip_path, 'rb').read()
    nomes = []
    pos = 0
    while True:
        pos = dados.find(b'PK\x01\x02', pos)
        if pos < 0:
            break
        # offset 28: tamanho do nome; offset 46: inicio do nome
        if pos + 46 > len(dados):
            break
        tam = int.from_bytes(dados[pos + 28:pos + 30], 'little')
        nome = dados[pos + 46:pos + 46 + tam]
        nomes.append(nome)
        pos += 46 + tam
    return [n for n in nomes if b'\\' in n]


def ficheiros():
    """Todos os ficheiros a empacotar, com caminho relativo e separador '/'."""
    for entrada in INCLUIR:
        p = os.path.join(SRC, entrada)
        if os.path.isfile(p):
            yield entrada, p
        else:
            for base, _dirs, nomes in os.walk(p):
                for nome in sorted(nomes):
                    full = os.path.join(base, nome)
                    yield os.path.relpath(full, SRC).replace(os.sep, '/'), full


def referenciados(manifest):
    """Cada caminho que o manifest referencia. Se um so' nao resolver, nao carrega."""
    refs = {manifest['background']['service_worker'],
            manifest['action']['default_popup']}
    for cs in manifest.get('content_scripts', []):
        refs.update(cs.get('js', []))
    refs.update(manifest.get('icons', {}).values())
    refs.update(manifest.get('action', {}).get('default_icon', {}).values())
    return refs


def empacotar():
    os.makedirs(OUT, exist_ok=True)
    manifest = json.load(io.open(os.path.join(SRC, 'manifest.json'), encoding='utf-8'))
    versao = manifest['version']
    zip_nome = 'hades-browser-bridge-%s-unsigned.zip' % versao
    zip_path = os.path.join(OUT, zip_nome)
    if os.path.exists(zip_path):
        os.remove(zip_path)

    itens = sorted(set(ficheiros()))
    if not itens:
        print('  FALHA: nada para empacotar')
        return 1
    with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for rel, full in itens:
            if '\\' in rel:
                print('  FALHA: separador invalido em %s' % rel)
                return 1
            z.write(full, arcname=rel)

    sha = hashlib.sha256(io.open(zip_path, 'rb').read()).hexdigest()
    with io.open(os.path.join(OUT, 'SHA256SUMS.txt'), 'w', encoding='ascii', newline='\n') as f:
        f.write('%s  %s\n' % (sha, zip_nome))

    leia = LEIA_ME.format(zip=zip_nome, versao=versao, sha=sha, source_sha=_source_sha())
    destino = os.path.join(OUT, 'LEIA-ME.txt')
    with io.open(destino, 'w', encoding='ascii', newline='\n') as f:
        f.write(leia)
    print('  empacotado: %s (%.1f KB)' % (zip_nome, os.path.getsize(zip_path) / 1024.0))
    print('  SHA-256   : %s' % sha)
    return 0


def verificar():
    """Verifica o pacote. Devolve o numero de falhas."""
    falhas = []
    manifest = json.load(io.open(os.path.join(SRC, 'manifest.json'), encoding='utf-8'))
    versao = manifest['version']
    zip_nome = 'hades-browser-bridge-%s-unsigned.zip' % versao
    zip_path = os.path.join(OUT, zip_nome)

    if not os.path.exists(zip_path):
        print('  FALHA: o .zip nao existe: %s' % zip_nome)
        return 1

    # 1) o SHA256SUMS tem de conferir com o ficheiro real
    sha_path = os.path.join(OUT, 'SHA256SUMS.txt')
    if not os.path.exists(sha_path):
        falhas.append('SHA256SUMS.txt ausente')
    else:
        linha = io.open(sha_path, encoding='ascii').read().strip()
        partes = linha.split('  ', 1)
        if len(partes) != 2:
            falhas.append('SHA256SUMS.txt malformado')
        else:
            esperado, ficheiro = partes
            real = hashlib.sha256(io.open(zip_path, 'rb').read()).hexdigest()
            if ficheiro != zip_nome:
                falhas.append('SHA256SUMS aponta para %s, esperado %s' % (ficheiro, zip_nome))
            if esperado != real:
                falhas.append('hash diverge: sums=%s real=%s' % (esperado[:16], real[:16]))

    with zipfile.ZipFile(zip_path) as z:
        nomes = z.namelist()
        if z.testzip() is not None:
            falhas.append('entrada corrompida no zip')

    # 2) A REGRA CENTRAL: a spec do ZIP exige '/'. O .NET no Windows escreve '\'.
    #    Lido dos BYTES CRUS -- o namelist() do zipfile normaliza e esconderia isto.
    crus = separadores_crus(zip_path)
    if crus:
        falhas.append('%d entradas com backslash nos bytes crus do diretorio central '
                      '(o navegador nao resolve): %s'
                      % (len(crus), [n.decode('utf-8', 'replace') for n in crus[:3]]))

    # 3) o pacote de distribuicao nao leva o arnes de teste
    testes = [n for n in nomes if n.startswith('test/')]
    if testes:
        falhas.append('o zip leva testes: %s' % testes)

    # 4) cada caminho do manifest tem de resolver
    refs = referenciados(manifest)
    faltam = sorted(refs - set(nomes))
    if faltam:
        falhas.append('o zip nao resolve: %s' % faltam)

    # 5) nada de vazio, e' um zip inutil
    with zipfile.ZipFile(zip_path) as z:
        vazios = [i.filename for i in z.infolist() if i.file_size == 0]
    if vazios:
        falhas.append('entradas vazias: %s' % vazios)

    # 6) o LEIA-ME tem de existir e ser ASCII (powershell 5.1 le sem BOM como cp1252)
    leia_path = os.path.join(OUT, 'LEIA-ME.txt')
    if not os.path.exists(leia_path):
        falhas.append('LEIA-ME.txt ausente')
    else:
        bruto = io.open(leia_path, 'rb').read()
        ruins = [b for b in bruto if b > 127]
        if ruins:
            falhas.append('LEIA-ME.txt tem %d bytes > 127' % len(ruins))
        if b'NAO PUBLICADA' not in bruto:
            falhas.append('LEIA-ME.txt nao declara que nao foi publicado')
        if b'Assinado : NAO' not in bruto:
            falhas.append('LEIA-ME.txt nao declara assinatura=false')

    print('  zip        : %s' % zip_nome)
    print('  entradas   : %d' % len(nomes))
    print('  referencias: %d resolvidas de %d' % (len(refs) - len(faltam), len(refs)))
    if falhas:
        for f in falhas:
            print('  FALHA: %s' % f)
        return len(falhas)
    print('  INTEGRO: separador /, sem testes, manifest resolve, SHA-256 confere, LEIA-ME ASCII')
    return 0


def main():
    so_verificar = '--check' in sys.argv
    if not so_verificar:
        rc = empacotar()
        if rc:
            return rc
    n = verificar()
    return 1 if n else 0


if __name__ == '__main__':
    sys.exit(main())
