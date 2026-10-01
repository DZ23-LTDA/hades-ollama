# Large Files / Projetos grandes — Ollama Full

Arquitetura interna para upload/ingestão de arquivos e projetos grandes (§17).
SIP/PBX e provedores externos não estão envolvidos aqui.

## Componentes (internal/agent)

### `SafeExtractZip` (`archive.go`)
Extração de ZIP com defesa em profundidade e **relatório por-arquivo** (sem skip
silencioso): status `extracted`/`skipped`/`error` + motivo + bytes.

Proteções (falha-fechada — entrada perigosa aborta a extração):
- **zip-slip**: rejeita qualquer segmento `..` e caminho absoluto (unix/windows);
- **symlink**: rejeitado;
- **limite de contagem** de entradas;
- **tamanho descomprimido** por-arquivo e **total**;
- **razão de compressão** (guarda contra zip bomb);
- cópia com **teto real** (não confia no `UncompressedSize` declarado).

### `UploadManager` / `UploadSession` (`upload_session.go`)
Upload grande **resumable** por tenant (org-scoped):
- `StartUpload`: valida filename (sem traversal), tamanho, **quota por organização**;
- `AppendChunk`: **sequencial/resumable** (o cliente retoma consultando
  `ReceivedBytes`), com limites e quota, `WriteAt` no arquivo temporário;
- `CancelUpload`: remove o temporário;
- `FinalizeUpload`: exige completude, verifica **SHA-256** e faz **rename atômico**
  para o arquivo final;
- `GetUploadForOrganization`: leitura org-scoped.

## Benchmarks — MEDIDOS (não prometidos)

Medido em Linux (WSL2, tmpfs `/tmp`), Go 1.26, chunks de 8 MiB, com verificação
de SHA-256 e finalize atômico. Reproduza com:

```bash
UPLOAD_BENCH_MB=1024 go test ./internal/agent -run x \
  -bench BenchmarkUploadThroughput -benchtime=1x
```

| Tamanho | Resultado | Vazão medida | Tempo |
|---|---|---|---|
| 100 MB | ✅ passou (hash ok) | **81,4 MB/s** | ~1,29 s |
| 500 MB | ✅ passou (hash ok) | **72,4 MB/s** | ~7,24 s |
| 1 GB   | ✅ passou (hash ok) | **69,4 MB/s** | ~15,5 s |

> Estes números são de uma máquina de desenvolvimento; servem de baseline
> reprodutível, não de SLA. Um teste de tamanho médio (16 MiB) roda no CI
> (`TestUploadSessionHandlesMediumFile`); os tamanhos maiores são benchmarks
> sob demanda (não pesam o CI por padrão).

## Pendências honestas (próximas etapas)
- Integrar `UploadManager`/`SafeExtractZip` a uma rota HTTP do server (o modelo e
  os testes existem; falta o handler + wiring).
- Ingestão incremental com status por-arquivo persistido além do relatório de
  extração em memória.
- `.deb`/`.rpm`/AppImage e quotas por-projeto (além de por-organização) quando
  fizer sentido.
