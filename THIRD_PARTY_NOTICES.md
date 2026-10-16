# THIRD_PARTY_NOTICES

Documento **gerado** por `scripts/hades_third_party_notices.py`. Nao editar a mao.
**100% offline**: nada foi descarregado nem instalado.

Fontes: `go list -m all` + cache `GOPATH/pkg/mod` (Go); `package-lock.json` e, quando o
lockfile nao traz `license`, o `package.json` do proprio pacote (npm).

## 0. Resumo por familia (TRIAGEM, nao parecer juridico)

A familia sai de **frases e siglas com limite de palavra** no texto da licenca (Go:
ficheiro inteiro; npm: campo `license`). **Nao** decide compatibilidade com a MIT
deste projecto: **nomeia o que exige revisao humana**.

| familia | Go | npm |
|---|---|---|
| COPYLEFT_FRACO | 0 | 14 |
| PERMISSIVA | 136 | 780 |

### Fora do permissivo ou nao classificada, no Go (0)


### Forte/nao-livre/desconhecida, no npm (0)


## 1. Dependencias Go

| modulo | versao | ficheiro | primeira linha | familia | bytes | sha256 |
|---|---|---|---|---|---|---|
| `cel.dev/expr` | `v0.25.2` | `(nenhum)` | - | - | 0 | `` |
| `cloud.google.com/go/auth` | `v0.18.2` | `(nenhum)` | - | - | 0 | `` |
| `cloud.google.com/go/compute/metadata` | `v0.9.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp` | `v1.33.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/TheTitanrain/w32` | `v0.0.0-20180517000239-4f5cfb03fabf` | `LICENSE` | Copyright (c) 2010-2012 The w32 Authors. All rights reserved. | PERMISSIVA | 1404 | `cc30914e1e928b27...` |
| `github.com/agnivade/levenshtein` | `v1.2.1` | `License.txt` | The MIT License (MIT) | PERMISSIVA | 1083 | `911d0fb30ff3345e...` |
| `github.com/apapsch/go-jsonmerge/v2` | `v2.0.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/arbovm/levenshtein` | `v0.0.0-20160628152529-48b4e1c0c4d0` | `LICENSE` | Copyright (c) 2015, Arbo von Monkiewitsch All rights reserved. | PERMISSIVA | 1492 | `04fd712a0cbe2772...` |
| `github.com/aymanbagabas/go-osc52/v2` | `v2.0.1` | `LICENSE` | MIT License | PERMISSIVA | 1070 | `11319e34d9dac2f4...` |
| `github.com/aymanbagabas/go-udiff` | `v0.2.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/bahlo/generic-list-go` | `v0.2.0` | `LICENSE` | Copyright (c) 2009 The Go Authors. All rights reserved. | PERMISSIVA | 1479 | `2d36597f7117c38b...` |
| `github.com/beevik/etree` | `v1.6.0` | `LICENSE` | Copyright 2015-2024 Brett Vickers. All rights reserved. | PERMISSIVA | 1293 | `2dfed8b3afa7acbc...` |
| `github.com/bits-and-blooms/bitset` | `v1.22.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/buger/jsonparser` | `v1.1.2` | `LICENSE` | MIT License | PERMISSIVA | 1070 | `632b59709152e1b5...` |
| `github.com/bytedance/gopkg` | `v0.1.4` | `(nenhum)` | - | - | 0 | `` |
| `github.com/bytedance/sonic` | `v1.15.2` | `(nenhum)` | - | - | 0 | `` |
| `github.com/bytedance/sonic/loader` | `v0.5.1` | `(nenhum)` | - | - | 0 | `` |
| `github.com/cenkalti/backoff/v5` | `v5.0.3` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1077 | `5c0476add4c38b55...` |
| `github.com/cespare/xxhash/v2` | `v2.3.0` | `LICENSE.txt` | Copyright (c) 2016 Caleb Spare | PERMISSIVA | 1068 | `f566a9f97bacdaf0...` |
| `github.com/charmbracelet/bubbletea` | `v1.3.10` | `LICENSE` | MIT License | PERMISSIVA | 1080 | `69793670485e4ae8...` |
| `github.com/charmbracelet/colorprofile` | `v0.2.3-0.20250311203215-f60798e515dc` | `LICENSE` | MIT License | PERMISSIVA | 1080 | `57d6eeeb336030b8...` |
| `github.com/charmbracelet/lipgloss` | `v1.1.0` | `LICENSE` | MIT License | PERMISSIVA | 1080 | `699cffa88b62691e...` |
| `github.com/charmbracelet/x/ansi` | `v0.10.1` | `LICENSE` | MIT License | PERMISSIVA | 1076 | `0baa4a5942a801d9...` |
| `github.com/charmbracelet/x/cellbuf` | `v0.0.13-0.20250311204145-2c3ea96c31dd` | `LICENSE` | MIT License | PERMISSIVA | 1076 | `0baa4a5942a801d9...` |
| `github.com/charmbracelet/x/exp/golden` | `v0.0.0-20240806155701-69247e0abc2a` | `(nenhum)` | - | - | 0 | `` |
| `github.com/charmbracelet/x/term` | `v0.2.1` | `LICENSE` | MIT License | PERMISSIVA | 1076 | `0baa4a5942a801d9...` |
| `github.com/clipperhouse/displaywidth` | `v0.10.0` | `LICENSE` | MIT License | PERMISSIVA | 1069 | `fcdcc889a11c39c7...` |
| `github.com/clipperhouse/stringish` | `v0.1.1` | `(nenhum)` | - | - | 0 | `` |
| `github.com/clipperhouse/uax29/v2` | `v2.6.0` | `LICENSE` | MIT License | PERMISSIVA | 1069 | `07e9e3daaccc707c...` |
| `github.com/cloudwego/base64x` | `v0.1.7` | `(nenhum)` | - | - | 0 | `` |
| `github.com/cncf/xds/go` | `v0.0.0-20260202195803-dba9d589def2` | `(nenhum)` | - | - | 0 | `` |
| `github.com/containerd/console` | `v1.0.5` | `LICENSE` | Apache License | PERMISSIVA | 10765 | `4bbe3b885e8cd190...` |
| `github.com/cpuguy83/go-md2man/v2` | `v2.0.6` | `(nenhum)` | - | - | 0 | `` |
| `github.com/crewjam/saml` | `v0.5.1` | `LICENSE` | Copyright (c) 2015, Ross Kinder | PERMISSIVA | 1290 | `a335d6bb6742808e...` |
| `github.com/davecgh/go-spew` | `v1.1.1` | `LICENSE` | ISC License | PERMISSIVA | 766 | `1b93a317849ee09d...` |
| `github.com/dgryski/trifles` | `v0.0.0-20230903005119-f50d829f2e54` | `(nenhum)` | - | - | 0 | `` |
| `github.com/emirpasic/gods/v2` | `v2.0.0-alpha` | `LICENSE` | Copyright (c) 2015, Emir Pasic | PERMISSIVA | 2148 | `871b440e9b348d6b...` |
| `github.com/envoyproxy/go-control-plane` | `v0.14.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/envoyproxy/go-control-plane/envoy` | `v1.37.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/envoyproxy/go-control-plane/ratelimit` | `v0.1.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/envoyproxy/protoc-gen-validate` | `v1.3.3` | `(nenhum)` | - | - | 0 | `` |
| `github.com/erikgeiser/coninput` | `v0.0.0-20211004153227-1c3628e74d0f` | `LICENSE` | MIT License | PERMISSIVA | 1064 | `0a00deeeb36bb7b8...` |
| `github.com/fatih/color` | `v1.18.0` | `LICENSE.md` | The MIT License (MIT) | PERMISSIVA | 1079 | `de95ed802d2f90e3...` |
| `github.com/felixge/httpsnoop` | `v1.1.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/gabriel-vasile/mimetype` | `v1.4.13` | `LICENSE` | MIT License | PERMISSIVA | 1071 | `b5a867655833aca9...` |
| `github.com/gin-contrib/cors` | `v1.7.9` | `LICENSE` | MIT License | PERMISSIVA | 1066 | `bafe84dafda45b69...` |
| `github.com/gin-contrib/sse` | `v1.1.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1091 | `03458b6d5828e1be...` |
| `github.com/gin-gonic/gin` | `v1.12.0` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1091 | `03458b6d5828e1be...` |
| `github.com/go-jose/go-jose/v4` | `v4.1.4` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-logr/logr` | `v1.4.4` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `b40930bbcf80744c...` |
| `github.com/go-logr/stdr` | `v1.2.2` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `github.com/go-openapi/analysis` | `v0.25.5` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/errors` | `v0.22.8` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/jsonpointer` | `v1.0.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/jsonreference` | `v1.0.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/loads` | `v0.25.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/runtime` | `v0.33.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/runtime/server-middleware` | `v0.30.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/spec` | `v0.22.9` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/strfmt` | `v0.27.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/cmdutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/conv` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/fileutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/jsonutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/loading` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/mangling` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/netutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/pools` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/stringutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/typeutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/swag/yamlutils` | `v0.28.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-openapi/validate` | `v0.26.1` | `(nenhum)` | - | - | 0 | `` |
| `github.com/go-playground/assert/v2` | `v2.2.0` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1077 | `9e321f6a1db81616...` |
| `github.com/go-playground/locales` | `v0.14.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1079 | `a1a5ca382f9e8c64...` |
| `github.com/go-playground/universal-translator` | `v0.18.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `a55d0c6e1a35ee6c...` |
| `github.com/go-playground/validator/v10` | `v10.30.3` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1077 | `9e321f6a1db81616...` |
| `github.com/go-viper/mapstructure/v2` | `v2.5.0` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1085 | `22adc4abdece712a...` |
| `github.com/goccy/go-json` | `v0.10.6` | `(nenhum)` | - | - | 0 | `` |
| `github.com/goccy/go-yaml` | `v1.19.2` | `LICENSE` | MIT License | PERMISSIVA | 1072 | `43fe703ea9ebe27f...` |
| `github.com/golang-jwt/jwt/v4` | `v4.5.2` | `LICENSE` | Copyright (c) 2012 Dave Grijalva | PERMISSIVA | 1100 | `fe26ca41577b9b2b...` |
| `github.com/golang/glog` | `v1.2.5` | `(nenhum)` | - | - | 0 | `` |
| `github.com/golang/protobuf` | `v1.5.4` | `LICENSE` | Copyright 2010 The Go Authors.  All rights reserved. | PERMISSIVA | 1480 | `8778a9fc1eaffb03...` |
| `github.com/google/go-cmp` | `v0.7.0` | `LICENSE` | Copyright (c) 2017 The Go Authors. All rights reserved. | PERMISSIVA | 1479 | `17b5d209ba8f9684...` |
| `github.com/google/gofuzz` | `v1.0.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/google/s2a-go` | `v0.1.9` | `(nenhum)` | - | - | 0 | `` |
| `github.com/google/uuid` | `v1.6.0` | `LICENSE` | Copyright (c) 2009,2014 Google Inc. All rights reserved. | PERMISSIVA | 1480 | `0a8d61ed3cbfd531...` |
| `github.com/googleapis/enterprise-certificate-proxy` | `v0.3.11` | `(nenhum)` | - | - | 0 | `` |
| `github.com/googleapis/gax-go/v2` | `v2.17.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/gorilla/websocket` | `v1.5.3` | `LICENSE` | Copyright (c) 2013 The Gorilla WebSocket Authors. All rights reserved. | PERMISSIVA | 1312 | `2be1b548b0387ca8...` |
| `github.com/grpc-ecosystem/grpc-gateway/v2` | `v2.30.0` | `LICENSE` | Copyright (c) 2015, Gengo, Inc. | PERMISSIVA | 1511 | `a15b1d1b168954c9...` |
| `github.com/inconshreveable/mousetrap` | `v1.1.0` | `LICENSE` | Apache License | PERMISSIVA | 11360 | `4819582701f150b2...` |
| `github.com/jackc/pgpassfile` | `v1.0.0` | `LICENSE` | Copyright (c) 2019 Jack Christensen | PERMISSIVA | 1073 | `adb1663fda031df8...` |
| `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | `LICENSE` | Copyright (c) 2020 Jack Christensen | PERMISSIVA | 1073 | `fc505773403fe869...` |
| `github.com/jackc/pgx/v5` | `v5.11.0` | `LICENSE` | Copyright (c) 2013-2021 Jack Christensen | PERMISSIVA | 1078 | `467f95e074fe2307...` |
| `github.com/jackc/puddle/v2` | `v2.2.2` | `LICENSE` | Copyright (c) 2018 Jack Christensen | PERMISSIVA | 1073 | `2d50e98a4900b4d6...` |
| `github.com/jonboulle/clockwork` | `v0.5.0` | `LICENSE` | Apache License | PERMISSIVA | 11324 | `a8ad31b1c3f40dca...` |
| `github.com/jordanlewis/gcassert` | `v0.0.0-20250430164644-389ef753e22e` | `(nenhum)` | - | - | 0 | `` |
| `github.com/josharian/intern` | `v1.0.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/json-iterator/go` | `v1.1.12` | `LICENSE` | MIT License | PERMISSIVA | 1070 | `3247931083f058b0...` |
| `github.com/klauspost/compress` | `v1.20.1` | `LICENSE` | Copyright (c) 2012 The Go Authors. All rights reserved. | PERMISSIVA | 16733 | `0d9e582ee4bff57b...` |
| `github.com/klauspost/cpuid/v2` | `v2.4.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/kr/pretty` | `v0.3.1` | `License` | Copyright 2012 Keith Rarick | PERMISSIVA | 1052 | `283c28b781bc7487...` |
| `github.com/kr/text` | `v0.2.0` | `License` | Copyright 2012 Keith Rarick | PERMISSIVA | 1052 | `283c28b781bc7487...` |
| `github.com/ledongthuc/pdf` | `v0.0.0-20250511090121-5959a4027728` | `LICENSE` | Copyright (c) 2009 The Go Authors. All rights reserved. | PERMISSIVA | 1479 | `2d36597f7117c38b...` |
| `github.com/leodido/go-urn` | `v1.4.0` | `LICENSE` | MIT License | PERMISSIVA | 1075 | `fe81a9072afbfe0f...` |
| `github.com/lucasb-eyer/go-colorful` | `v1.2.0` | `LICENSE` | Copyright (c) 2013 Lucas Beyer | PERMISSIVA | 1055 | `1e4cf2a117516ec1...` |
| `github.com/mailru/easyjson` | `v0.7.7` | `LICENSE` | Copyright (c) 2016 Mail.Ru Group | PERMISSIVA | 1057 | `87baa92c3100cf37...` |
| `github.com/mattermost/xml-roundtrip-validator` | `v0.1.0` | `LICENSE.txt` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `github.com/mattn/go-colorable` | `v0.1.14` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1085 | `88a2379b3ca34bf5...` |
| `github.com/mattn/go-isatty` | `v0.0.23` | `LICENSE` | Copyright (c) Yasuhiro MATSUMOTO <mattn.jp@gmail.com> | PERMISSIVA | 1099 | `08eab1118c80885f...` |
| `github.com/mattn/go-localereader` | `v0.0.2-0.20220822084737-6bae6c923850` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1085 | `dceede03550621e6...` |
| `github.com/mattn/go-pointer` | `v0.0.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1085 | `7228b277bcb9df4b...` |
| `github.com/mattn/go-runewidth` | `v0.0.30` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1085 | `88a2379b3ca34bf5...` |
| `github.com/mattn/go-sqlite3` | `v1.14.52` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1085 | `afa48e5e64dc6102...` |
| `github.com/modern-go/concurrent` | `v0.0.0-20180306012644-bacd9c7ef1dd` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `github.com/modern-go/reflect2` | `v1.0.2` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `github.com/muesli/ansi` | `v0.0.0-20230316100256-276c6243b2f6` | `LICENSE` | MIT License | PERMISSIVA | 1079 | `43f65e00e8919fa4...` |
| `github.com/muesli/cancelreader` | `v0.2.2` | `LICENSE` | MIT License | PERMISSIVA | 1095 | `8c139dbbc83b4fb0...` |
| `github.com/muesli/termenv` | `v0.16.0` | `LICENSE` | MIT License | PERMISSIVA | 1079 | `233e8cf9a812999c...` |
| `github.com/oapi-codegen/runtime` | `v1.6.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/oklog/ulid/v2` | `v2.1.1` | `(nenhum)` | - | - | 0 | `` |
| `github.com/olekukonko/cat` | `v0.0.0-20250911104152-50322a0618f6` | `LICENSE` | MIT License | PERMISSIVA | 1068 | `6b2a3fd39696e823...` |
| `github.com/olekukonko/errors` | `v1.2.0` | `LICENSE` | MIT License | PERMISSIVA | 1068 | `6b2a3fd39696e823...` |
| `github.com/olekukonko/ll` | `v0.1.6` | `LICENSE` | MIT License | PERMISSIVA | 1068 | `6b2a3fd39696e823...` |
| `github.com/olekukonko/tablewriter` | `v1.1.5` | `LICENSE.md` | Copyright (C) 2014 by Oleku Konko | PERMISSIVA | 1058 | `0583acca2b32a7e4...` |
| `github.com/olekukonko/ts` | `v0.0.0-20171002115256-78ecb04241c0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/pelletier/go-toml/v2` | `v2.4.3` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1101 | `26844e4b53c5adec...` |
| `github.com/pkg/browser` | `v0.0.0-20240102092130-5ac0b6a4141c` | `LICENSE` | Copyright (c) 2014, Dave Cheney <dave@cheney.net> | PERMISSIVA | 1312 | `4801bb0c5d70b980...` |
| `github.com/pkg/errors` | `v0.9.1` | `LICENSE` | Copyright (c) 2015, Dave Cheney <dave@cheney.net> | PERMISSIVA | 1312 | `8d427fd87bc9579e...` |
| `github.com/planetscale/vtprotobuf` | `v0.6.1-0.20240319094008-0393e58bdf10` | `(nenhum)` | - | - | 0 | `` |
| `github.com/pmezard/go-difflib` | `v1.0.0` | `LICENSE` | Copyright (c) 2013, Patrick Mezard | PERMISSIVA | 1445 | `2eb550be6801c1ea...` |
| `github.com/quic-go/go-ossfuzz-seeds` | `v0.1.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/quic-go/qpack` | `v0.6.0` | `LICENSE.md` | Copyright 2019 Marten Seemann | PERMISSIVA | 1054 | `1b6a897efd39b20b...` |
| `github.com/quic-go/quic-go` | `v0.60.0` | `LICENSE` | MIT License | PERMISSIVA | 1091 | `77d0b7b53e8abb84...` |
| `github.com/rivo/uniseg` | `v0.4.7` | `LICENSE.txt` | MIT License | PERMISSIVA | 1072 | `a59885f5f0f3b3c0...` |
| `github.com/rogpeppe/fastuuid` | `v1.2.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/rogpeppe/go-internal` | `v1.14.1` | `LICENSE` | Copyright (c) 2018 The Go Authors. All rights reserved. | PERMISSIVA | 1479 | `4835612df0098ca9...` |
| `github.com/russellhaering/goxmldsig` | `v1.6.0` | `LICENSE` | Apache License | PERMISSIVA | 10142 | `09e8a9bcec806710...` |
| `github.com/russross/blackfriday/v2` | `v2.1.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/spf13/cobra` | `v1.10.2` | `LICENSE.txt` | Apache License | PERMISSIVA | 10140 | `5e3400b93bbb099e...` |
| `github.com/spf13/pflag` | `v1.0.9` | `LICENSE` | Copyright (c) 2012 Alex Ogier. All rights reserved. | PERMISSIVA | 1531 | `b8514c577c1c4b46...` |
| `github.com/spiffe/go-spiffe/v2` | `v2.7.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/stretchr/objx` | `v0.5.3` | `LICENSE` | The MIT License | PERMISSIVA | 1117 | `b2663894033a05fd...` |
| `github.com/stretchr/testify` | `v1.12.1` | `LICENSE` | MIT License | PERMISSIVA | 1103 | `f8e536c1c7b69581...` |
| `github.com/tkrajina/go-reflector` | `v0.5.5` | `LICENSE.txt` | Apache License | PERMISSIVA | 11348 | `3f1fc64a3fded0d8...` |
| `github.com/tkrajina/typescriptify-golang-structs` | `v0.2.0` | `LICENSE.txt` | Apache License | PERMISSIVA | 11348 | `e4794d2db458f14f...` |
| `github.com/tree-sitter/go-tree-sitter` | `v0.25.0` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1101 | `0eea8dc45e89deeb...` |
| `github.com/tree-sitter/tree-sitter-c` | `v0.23.4` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `2e0110e07abef7c2...` |
| `github.com/tree-sitter/tree-sitter-cpp` | `v0.23.4` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `2e0110e07abef7c2...` |
| `github.com/tree-sitter/tree-sitter-embedded-template` | `v0.23.2` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `c5cfb43042b6b720...` |
| `github.com/tree-sitter/tree-sitter-go` | `v0.23.4` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `2e0110e07abef7c2...` |
| `github.com/tree-sitter/tree-sitter-html` | `v0.23.2` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `2e0110e07abef7c2...` |
| `github.com/tree-sitter/tree-sitter-java` | `v0.23.5` | `LICENSE` | MIT License | PERMISSIVA | 1069 | `52ed137b039cd9c4...` |
| `github.com/tree-sitter/tree-sitter-javascript` | `v0.23.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `2e0110e07abef7c2...` |
| `github.com/tree-sitter/tree-sitter-json` | `v0.24.8` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `2e0110e07abef7c2...` |
| `github.com/tree-sitter/tree-sitter-php` | `v0.23.11` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1166 | `664764fabf40c0b3...` |
| `github.com/tree-sitter/tree-sitter-python` | `v0.23.6` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `d724405ce238a22c...` |
| `github.com/tree-sitter/tree-sitter-ruby` | `v0.23.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1075 | `ee006f02a3d856df...` |
| `github.com/tree-sitter/tree-sitter-rust` | `v0.23.2` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1080 | `31d5b6f4243d5c7c...` |
| `github.com/twitchyliquid64/golang-asm` | `v0.15.1` | `LICENSE` | Copyright (c) 2009 The Go Authors. All rights reserved. | PERMISSIVA | 1479 | `2d36597f7117c38b...` |
| `github.com/ugorji/go/codec` | `v1.3.1` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1106 | `c8a84b926246b4ac...` |
| `github.com/wk8/go-ordered-map/v2` | `v2.1.8` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `b40930bbcf80744c...` |
| `github.com/xdg-go/pbkdf2` | `v1.0.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/xdg-go/scram` | `v1.2.0` | `(nenhum)` | - | - | 0 | `` |
| `github.com/xdg-go/stringprep` | `v1.0.4` | `(nenhum)` | - | - | 0 | `` |
| `github.com/xo/terminfo` | `v0.0.0-20220910002029-abceb7e1c41e` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1078 | `1f5f51151ba96f04...` |
| `github.com/youmark/pkcs8` | `v0.0.0-20240726163527-a2c0da244d78` | `(nenhum)` | - | - | 0 | `` |
| `go.mongodb.org/mongo-driver/v2` | `v2.8.0` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `go.opentelemetry.io/auto/sdk` | `v1.2.1` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `go.opentelemetry.io/contrib/detectors/gcp` | `v1.44.0` | `(nenhum)` | - | - | 0 | `` |
| `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` | `v0.70.0` | `(nenhum)` | - | - | 0 | `` |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | `v0.70.0` | `(nenhum)` | - | - | 0 | `` |
| `go.opentelemetry.io/otel` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/otel/exporters/stdout/stdouttrace` | `v1.45.0` | `(nenhum)` | - | - | 0 | `` |
| `go.opentelemetry.io/otel/log` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/otel/metric` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/otel/sdk` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/otel/sdk/metric` | `v1.47.0` | `(nenhum)` | - | - | 0 | `` |
| `go.opentelemetry.io/otel/trace` | `v1.47.0` | `LICENSE` | Apache License | PERMISSIVA | 12892 | `1ae07514be1d7bb3...` |
| `go.opentelemetry.io/proto/otlp` | `v1.11.0` | `LICENSE` | Apache License | PERMISSIVA | 11357 | `c71d239df91726fc...` |
| `go.uber.org/goleak` | `v1.3.0` | `LICENSE` | The MIT License (MIT) | PERMISSIVA | 1090 | `cea390bdf643a06f...` |
| `go.uber.org/mock` | `v0.6.0` | `(nenhum)` | - | - | 0 | `` |
| `go.yaml.in/yaml/v3` | `v3.0.5` | `LICENSE` | This project is covered by two different licenses: MIT and Apache. | PERMISSIVA | 2151 | `d18f6323b71b0b76...` |
| `go.yaml.in/yaml/v3` | `v3.0.5` | `NOTICE` | Copyright 2011-2016 Canonical Ltd. | PERMISSIVA | 560 | `f6c2dd3a67b576ea...` |
| `golang.org/x/arch` | `v0.29.0` | `(nenhum)` | - | - | 0 | `` |
| `golang.org/x/crypto` | `v0.57.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/exp` | `v0.0.0-20250218142911-aa4b98e5adaa` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/image` | `v0.46.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/mod` | `v0.41.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/net` | `v0.60.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/oauth2` | `v0.36.0` | `(nenhum)` | - | - | 0 | `` |
| `golang.org/x/sync` | `v0.24.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/sys` | `v0.49.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/term` | `v0.46.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/text` | `v0.42.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `golang.org/x/tools` | `v0.49.0` | `LICENSE` | Copyright 2009 The Go Authors. | PERMISSIVA | 1453 | `911f8f5782931320...` |
| `gonum.org/v1/gonum` | `v0.17.0` | `LICENSE` | Copyright ?2013 The Gonum Authors. All rights reserved. | PERMISSIVA | 1530 | `b44d9e394ba3efc1...` |
| `google.golang.org/genproto/googleapis/api` | `v0.0.0-20260928230214-8a89bd6388cc` | `LICENSE` | Apache License | PERMISSIVA | 11358 | `cfc7749b96f63bd3...` |
| `google.golang.org/genproto/googleapis/rpc` | `v0.0.0-20260928230214-8a89bd6388cc` | `LICENSE` | Apache License | PERMISSIVA | 11358 | `cfc7749b96f63bd3...` |
| `google.golang.org/grpc` | `v1.83.2` | `LICENSE` | Apache License | PERMISSIVA | 11358 | `cfc7749b96f63bd3...` |
| `google.golang.org/grpc` | `v1.83.2` | `NOTICE.txt` | Copyright 2014 gRPC authors. | PERMISSIVA | 554 | `693ff28ec216d511...` |
| `google.golang.org/protobuf` | `v1.36.12` | `LICENSE` | Copyright (c) 2018 The Go Authors. All rights reserved. | PERMISSIVA | 1479 | `4835612df0098ca9...` |
| `gopkg.in/check.v1` | `v1.0.0-20201130134442-10cb98267c6c` | `LICENSE` | Gocheck - A rich testing framework for Go | PERMISSIVA | 1377 | `69ce77f2b1c9c608...` |
| `gopkg.in/yaml.v3` | `v3.0.1` | `LICENSE` | This project is covered by two different licenses: MIT and Apache. | PERMISSIVA | 2151 | `d18f6323b71b0b76...` |
| `gopkg.in/yaml.v3` | `v3.0.1` | `NOTICE` | Copyright 2011-2016 Canonical Ltd. | PERMISSIVA | 560 | `f6c2dd3a67b576ea...` |
| `gotest.tools` | `v2.2.0+incompatible` | `LICENSE` | Copyright 2018 gotest.tools authors | PERMISSIVA | 561 | `4911436829de6a54...` |
| `rsc.io/pdf` | `v0.1.1` | `(nenhum)` | - | - | 0 | `` |

### Go: no grafo mas NAO extraidos na cache local (74)

*A cache contem os modulos efectivamente compilados neste host.*

- `cel.dev/expr`
- `cloud.google.com/go/auth`
- `cloud.google.com/go/compute/metadata`
- `github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp`
- `github.com/apapsch/go-jsonmerge/v2`
- `github.com/aymanbagabas/go-udiff`
- `github.com/bits-and-blooms/bitset`
- `github.com/bytedance/gopkg`
- `github.com/bytedance/sonic`
- `github.com/bytedance/sonic/loader`
- `github.com/charmbracelet/x/exp/golden`
- `github.com/clipperhouse/stringish`
- `github.com/cloudwego/base64x`
- `github.com/cncf/xds/go`
- `github.com/cpuguy83/go-md2man/v2`
- `github.com/dgryski/trifles`
- `github.com/envoyproxy/go-control-plane`
- `github.com/envoyproxy/go-control-plane/envoy`
- `github.com/envoyproxy/go-control-plane/ratelimit`
- `github.com/envoyproxy/protoc-gen-validate`
- `github.com/felixge/httpsnoop`
- `github.com/go-jose/go-jose/v4`
- `github.com/go-openapi/analysis`
- `github.com/go-openapi/errors`
- `github.com/go-openapi/jsonpointer`
- `github.com/go-openapi/jsonreference`
- `github.com/go-openapi/loads`
- `github.com/go-openapi/runtime`
- `github.com/go-openapi/runtime/server-middleware`
- `github.com/go-openapi/spec`
- `github.com/go-openapi/strfmt`
- `github.com/go-openapi/swag`
- `github.com/go-openapi/swag/cmdutils`
- `github.com/go-openapi/swag/conv`
- `github.com/go-openapi/swag/fileutils`
- `github.com/go-openapi/swag/jsonutils`
- `github.com/go-openapi/swag/loading`
- `github.com/go-openapi/swag/mangling`
- `github.com/go-openapi/swag/netutils`
- `github.com/go-openapi/swag/pools`
- `github.com/go-openapi/swag/stringutils`
- `github.com/go-openapi/swag/typeutils`
- `github.com/go-openapi/swag/yamlutils`
- `github.com/go-openapi/validate`
- `github.com/goccy/go-json`
- `github.com/golang/glog`
- `github.com/google/gofuzz`
- `github.com/google/s2a-go`
- `github.com/googleapis/enterprise-certificate-proxy`
- `github.com/googleapis/gax-go/v2`
- `github.com/jordanlewis/gcassert`
- `github.com/josharian/intern`
- `github.com/klauspost/cpuid/v2`
- `github.com/oapi-codegen/runtime`
- `github.com/oklog/ulid/v2`
- `github.com/olekukonko/ts`
- `github.com/planetscale/vtprotobuf`
- `github.com/quic-go/go-ossfuzz-seeds`
- `github.com/rogpeppe/fastuuid`
- `github.com/russross/blackfriday/v2`
- `github.com/spiffe/go-spiffe/v2`
- `github.com/xdg-go/pbkdf2`
- `github.com/xdg-go/scram`
- `github.com/xdg-go/stringprep`
- `github.com/youmark/pkcs8`
- `go.opentelemetry.io/contrib/detectors/gcp`
- `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`
- `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- `go.opentelemetry.io/otel/exporters/stdout/stdouttrace`
- `go.opentelemetry.io/otel/sdk/metric`
- `go.uber.org/mock`
- `golang.org/x/arch`
- `golang.org/x/oauth2`
- `rsc.io/pdf`

### Go: extraidos mas SEM ficheiro de licenca (0)


## 2. Dependencias npm (`app/ui/app`)

A licenca vem do **campo `license` do lockfile**; quando ele falta, do **campo `license` do `package.json` do pacote** (18 pacotes obtidos por essa segunda via). O `sha256` so' aparece quando existe ficheiro de licenca em `node_modules`.

| pacote | versao | licenca | familia | ficheiro | bytes | sha256 |
|---|---|---|---|---|---|---|
| `@adobe/css-tools` | `4.4.3` | MIT | PERMISSIVA | `LICENSE` | 1159 | `326ff38500622860...` |
| `@ampproject/remapping` | `2.3.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11358 | `cfc7749b96f63bd3...` |
| `@axe-core/playwright` | `4.13.0` | MPL-2.0 | COPYLEFT_FRACO | `LICENSE` | 15920 | `812e9d96e900a093...` |
| `@babel/code-frame` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/compat-data` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/core` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/generator` | `7.29.8` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-compilation-targets` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-globals` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-module-imports` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-module-transforms` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-plugin-utils` | `7.27.1` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-string-parser` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-validator-identifier` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helper-validator-option` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/helpers` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1189 | `4be9d87b56a30629...` |
| `@babel/parser` | `7.29.9` | MIT | PERMISSIVA | `LICENSE` | 1086 | `2e97627cb278aa75...` |
| `@babel/plugin-transform-react-jsx-self` | `7.27.1` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/plugin-transform-react-jsx-source` | `7.27.1` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/runtime` | `7.27.6` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/template` | `7.29.7` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/traverse` | `7.29.8` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@babel/types` | `7.29.8` | MIT | PERMISSIVA | `LICENSE` | 1106 | `117da2af0d4ce0fe...` |
| `@bcoe/v8-coverage` | `1.0.2` | MIT | PERMISSIVA | `LICENSE.md` | 1088 | `98d226d560dfbb0a...` |
| `@chromatic-com/storybook` | `4.1.3` | MIT | PERMISSIVA | `LICENSE` | 1079 | `d4d772bf058070f7...` |
| `@csstools/cascade-layer-name-parser` | `2.0.5` | MIT | PERMISSIVA | `LICENSE.md` | 1111 | `d00d032f517721b4...` |
| `@csstools/color-helpers` | `5.1.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/css-calc` | `2.1.4` | MIT | PERMISSIVA | `LICENSE.md` | 1111 | `d00d032f517721b4...` |
| `@csstools/css-color-parser` | `3.1.0` | MIT | PERMISSIVA | `LICENSE.md` | 1111 | `d00d032f517721b4...` |
| `@csstools/css-parser-algorithms` | `3.0.5` | MIT | PERMISSIVA | `LICENSE.md` | 1111 | `d00d032f517721b4...` |
| `@csstools/css-tokenizer` | `3.0.4` | MIT | PERMISSIVA | `LICENSE.md` | 1111 | `d00d032f517721b4...` |
| `@csstools/media-query-list-parser` | `4.0.3` | MIT | PERMISSIVA | `LICENSE.md` | 1111 | `d00d032f517721b4...` |
| `@csstools/postcss-alpha-function` | `1.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-cascade-layers` | `5.0.2` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/selector-specificity` | `5.0.0` | MIT-0 | PERMISSIVA | `(nenhum)` | 0 | `` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `@csstools/postcss-color-function` | `4.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-color-function-display-p3-linear` | `1.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-color-mix-function` | `3.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-color-mix-variadic-function-arguments` | `1.0.2` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-content-alt-text` | `2.0.8` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-contrast-color-function` | `2.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-exponential-functions` | `2.0.9` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-font-format-keywords` | `4.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-gamut-mapping` | `2.0.11` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-gradients-interpolation-method` | `5.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-hwb-function` | `4.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-ic-unit` | `4.0.4` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-initial` | `2.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-is-pseudo-class` | `5.0.3` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/selector-specificity` | `5.0.0` | MIT-0 | PERMISSIVA | `(nenhum)` | 0 | `` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `@csstools/postcss-light-dark-function` | `2.0.11` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-logical-float-and-clear` | `3.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-logical-overflow` | `2.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-logical-overscroll-behavior` | `2.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-logical-resize` | `3.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-logical-viewport-units` | `3.0.4` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-media-minmax` | `2.0.9` | MIT | PERMISSIVA | `LICENSE.md` | 1124 | `8d4ea4f4698e9f0d...` |
| `@csstools/postcss-media-queries-aspect-ratio-number-values` | `3.0.5` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-nested-calc` | `4.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-normalize-display-values` | `4.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-oklab-function` | `4.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-position-area-property` | `1.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-progressive-custom-properties` | `4.2.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-property-rule-prelude-list` | `1.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-random-function` | `2.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-relative-color-syntax` | `3.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-scope-pseudo-class` | `4.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `@csstools/postcss-sign-functions` | `1.1.4` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-stepped-value-functions` | `4.0.9` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-syntax-descriptor-syntax-production` | `1.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-system-ui-font-family` | `1.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-text-decoration-shorthand` | `4.0.3` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-trigonometric-functions` | `4.0.9` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/postcss-unset-value` | `4.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/utilities` | `2.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@esbuild/aix-ppc64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/android-arm` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/android-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/android-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/darwin-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/darwin-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/freebsd-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/freebsd-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-arm` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-ia32` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-loong64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-mips64el` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-ppc64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-riscv64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-s390x` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/linux-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/netbsd-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/netbsd-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/openbsd-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/openbsd-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/sunos-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/win32-arm64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/win32-ia32` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@esbuild/win32-x64` | `0.25.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@eslint-community/eslint-utils` | `4.10.1` | MIT | PERMISSIVA | `LICENSE` | 1071 | `fcf6eabf68ca9698...` |
| `eslint-visitor-keys` | `3.4.3` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11337 | `d8bf34ff6d190640...` |
| `@eslint-community/regexpp` | `4.12.2` | MIT | PERMISSIVA | `LICENSE` | 1071 | `fcf6eabf68ca9698...` |
| `@eslint/config-array` | `0.20.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@eslint/config-helpers` | `0.2.2` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@eslint/core` | `0.14.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@eslint/eslintrc` | `3.3.1` | MIT | PERMISSIVA | `LICENSE` | 1094 | `3b6be04f862a077a...` |
| `globals` | `14.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `@eslint/js` | `9.39.5` | MIT | PERMISSIVA | `LICENSE` | 1094 | `3b6be04f862a077a...` |
| `@eslint/object-schema` | `2.1.6` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@eslint/plugin-kit` | `0.3.5` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@eslint/core` | `0.15.2` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@floating-ui/core` | `1.7.1` | MIT | PERMISSIVA | `LICENSE` | 1089 | `0e4c9a9b6c71019c...` |
| `@floating-ui/dom` | `1.7.1` | MIT | PERMISSIVA | `LICENSE` | 1089 | `0e4c9a9b6c71019c...` |
| `@floating-ui/react` | `0.26.28` | MIT | PERMISSIVA | `LICENSE` | 1089 | `0e4c9a9b6c71019c...` |
| `@floating-ui/react-dom` | `2.1.3` | MIT | PERMISSIVA | `LICENSE` | 1089 | `0e4c9a9b6c71019c...` |
| `@floating-ui/utils` | `0.2.9` | MIT | PERMISSIVA | `LICENSE` | 1089 | `0e4c9a9b6c71019c...` |
| `@headlessui/react` | `2.2.10` | MIT | PERMISSIVA | `LICENSE` | 1069 | `c0812e5b3b584876...` |
| `@heroicons/react` | `2.2.0` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `@humanfs/core` | `0.19.2` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@humanfs/node` | `0.16.8` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@humanfs/types` | `0.15.0` | Apache-2.0 | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@humanwhocodes/module-importer` | `1.0.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@humanwhocodes/retry` | `0.4.3` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@isaacs/cliui` | `8.0.2` | ISC | PERMISSIVA | `LICENSE.txt` | 731 | `2dc0465729366c3a...` |
| `@istanbuljs/schema` | `0.1.3` | MIT | PERMISSIVA | `LICENSE` | 1068 | `5a547aafe7b171c8...` |
| `@joshwooding/vite-plugin-react-docgen-typescript` | `0.6.1` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@jridgewell/gen-mapping` | `0.3.13` | MIT | PERMISSIVA | `LICENSE` | 1080 | `769d154fbde32a91...` |
| `@jridgewell/remapping` | `2.3.5` | MIT | PERMISSIVA | `LICENSE` | 1080 | `769d154fbde32a91...` |
| `@jridgewell/resolve-uri` | `3.1.2` | MIT | PERMISSIVA | `LICENSE` | 1079 | `b8778b155bfde5a2...` |
| `@jridgewell/sourcemap-codec` | `1.6.0` | MIT | PERMISSIVA | `LICENSE` | 1080 | `769d154fbde32a91...` |
| `@jridgewell/trace-mapping` | `0.3.31` | MIT | PERMISSIVA | `LICENSE` | 1080 | `769d154fbde32a91...` |
| `@mdx-js/react` | `3.1.0` | MIT | PERMISSIVA | `license` | 1094 | `c274d2be30a4def0...` |
| `@napi-rs/lzma-linux-x64-gnu` | `1.5.1` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@neoconfetti/react` | `1.0.0` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@pkgjs/parseargs` | `0.11.0` | MIT | PERMISSIVA | `LICENSE` | 11357 | `c71d239df91726fc...` |
| `@playwright/test` | `1.64.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11601 | `45873d00a0dd2435...` |
| `@polka/url` | `1.0.0-next.29` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@react-aria/focus` | `3.20.5` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@react-aria/interactions` | `3.25.3` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@react-aria/ssr` | `3.9.9` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@react-aria/utils` | `3.29.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@react-stately/flags` | `3.1.2` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@react-stately/utils` | `3.10.7` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@react-types/shared` | `3.30.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11335 | `7dfe6526888bac51...` |
| `@rolldown/pluginutils` | `1.0.0-beta.27` | MIT | PERMISSIVA | `LICENSE` | 1302 | `23ecfff35a5a2e80...` |
| `@rollup/pluginutils` | `5.2.0` | MIT | PERMISSIVA | `LICENSE` | 1151 | `7b8136f76c483d03...` |
| `@rollup/rollup-android-arm-eabi` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-android-arm64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-darwin-arm64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-darwin-x64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-freebsd-arm64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-freebsd-x64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-arm-gnueabihf` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-arm-musleabihf` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-arm64-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-arm64-musl` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-loong64-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-loong64-musl` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-ppc64-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-ppc64-musl` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-riscv64-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-riscv64-musl` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-s390x-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-x64-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-linux-x64-musl` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-openbsd-x64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-openharmony-arm64` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-win32-arm64-msvc` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-win32-ia32-msvc` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-win32-x64-gnu` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@rollup/rollup-win32-x64-msvc` | `4.63.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@shikijs/core` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `@shikijs/engine-javascript` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `@shikijs/engine-oniguruma` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `@shikijs/langs` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `@shikijs/themes` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `@shikijs/types` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `@shikijs/vscode-textmate` | `10.0.2` | MIT | PERMISSIVA | `LICENSE.md` | 1083 | `b5cb7ff7859e7d28...` |
| `@storybook/addon-a11y` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/addon-docs` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/addon-onboarding` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/addon-vitest` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/builder-vite` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/csf-plugin` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `unplugin` | `1.16.1` | MIT | PERMISSIVA | `LICENSE` | 1077 | `2ef2cbeeb84d5e7a...` |
| `@storybook/global` | `5.0.0` | MIT | PERMISSIVA | `LICENSE` | 1062 | `576f0541c45448ed...` |
| `@storybook/icons` | `1.4.0` | MIT | PERMISSIVA | `LICENSE.md` | 1080 | `a884e287f0e37d3e...` |
| `@storybook/react` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/react-dom-shim` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@storybook/react-vite` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `find-up` | `7.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `locate-path` | `7.2.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `p-limit` | `4.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `p-locate` | `6.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `path-exists` | `5.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `yocto-queue` | `1.2.1` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `@swc/helpers` | `0.5.17` | Apache-2.0 | PERMISSIVA | `LICENSE` | 10836 | `d96eba1f1881334d...` |
| `@tailwindcss/node` | `4.3.3` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `@tailwindcss/oxide` | `4.3.3` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `@tailwindcss/oxide-android-arm64` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-darwin-arm64` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-darwin-x64` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-freebsd-x64` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-linux-arm-gnueabihf` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-linux-arm64-gnu` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-linux-arm64-musl` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-linux-x64-gnu` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-linux-x64-musl` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-wasm32-wasi` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@emnapi/core` | `1.11.1` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@emnapi/runtime` | `1.11.1` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@emnapi/wasi-threads` | `1.2.2` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@napi-rs/wasm-runtime` | `1.1.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tybys/wasm-util` | `0.10.2` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `tslib` | `2.8.1` | 0BSD | PERMISSIVA | `LICENSE.txt` | 655 | `210b19e543130388...` |
| `@tailwindcss/oxide-win32-arm64-msvc` | `4.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@tailwindcss/oxide-win32-x64-msvc` | `4.3.3` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `@tailwindcss/typography` | `0.5.20` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `@tailwindcss/vite` | `4.3.3` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `@tanstack/history` | `1.162.5` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/query-core` | `5.104.1` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/react-query` | `5.104.1` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/react-router` | `1.170.42` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/react-store` | `0.11.2` | MIT | PERMISSIVA | `LICENSE` | 1071 | `cb4a7917af22bd56...` |
| `@tanstack/react-virtual` | `3.13.10` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/router-core` | `1.171.35` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/router-generator` | `1.167.40` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/history` | `1.162.4` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/router-core` | `1.171.34` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/router-plugin` | `1.168.42` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/history` | `1.162.4` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/router-core` | `1.171.34` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/router-utils` | `1.162.3` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/store` | `0.11.2` | MIT | PERMISSIVA | `LICENSE` | 1071 | `cb4a7917af22bd56...` |
| `@tanstack/virtual-core` | `3.13.10` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@tanstack/virtual-file-routes` | `1.162.0` | MIT | PERMISSIVA | `LICENSE` | 1079 | `a405ee70c632bb93...` |
| `@testing-library/dom` | `10.4.0` | MIT | PERMISSIVA | `LICENSE` | 1079 | `bf8fd38056b7606d...` |
| `@testing-library/jest-dom` | `6.6.3` | MIT | PERMISSIVA | `LICENSE` | 1079 | `bf8fd38056b7606d...` |
| `chalk` | `3.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `dom-accessibility-api` | `0.6.3` | MIT | PERMISSIVA | `LICENSE.md` | 1077 | `0ffe67fe630169de...` |
| `@testing-library/user-event` | `14.6.1` | MIT | PERMISSIVA | `LICENSE` | 1081 | `b5a4028c49c48f82...` |
| `@types/aria-query` | `5.0.4` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/babel__core` | `7.20.5` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/babel__generator` | `7.27.0` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/babel__template` | `7.4.4` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/babel__traverse` | `7.20.7` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/chai` | `5.2.2` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/debug` | `4.1.12` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/deep-eql` | `4.0.2` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/doctrine` | `0.0.9` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/estree` | `1.0.9` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/estree-jsx` | `1.0.5` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/hast` | `3.0.4` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/json-schema` | `7.0.15` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/katex` | `0.16.7` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/mdast` | `4.0.4` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/mdx` | `2.0.13` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/ms` | `2.1.0` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/node` | `24.19.1` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/prismjs` | `1.26.6` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/react` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/react-dom` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/react-test-renderer` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/resolve` | `1.20.6` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@types/unist` | `3.0.3` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `@typescript-eslint/eslint-plugin` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `ignore` | `7.0.12` | MIT | PERMISSIVA | `LICENSE-MIT` | 1095 | `9c94db23dc4b1e9a...` |
| `@typescript-eslint/parser` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/project-service` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `053a75a14b7508a8...` |
| `@typescript-eslint/scope-manager` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/tsconfig-utils` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `053a75a14b7508a8...` |
| `@typescript-eslint/type-utils` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `f3c51ace7f159b21...` |
| `@typescript-eslint/types` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/typescript-estree` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `balanced-match` | `4.0.4` | MIT | PERMISSIVA | `LICENSE.md` | 1096 | `a1bd5deadb6a06dd...` |
| `brace-expansion` | `5.0.12` | MIT | PERMISSIVA | `LICENSE` | 1096 | `68f12f6e2c336886...` |
| `minimatch` | `10.2.6` | BlueOak-1.0.0 | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `semver` | `7.8.5` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `@typescript-eslint/utils` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/visitor-keys` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `eslint-visitor-keys` | `5.0.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11337 | `d8bf34ff6d190640...` |
| `@ungap/structured-clone` | `1.3.0` | ISC | PERMISSIVA | `LICENSE` | 765 | `dc6d4961d8b6ee74...` |
| `@vitejs/plugin-react` | `4.7.0` | MIT | PERMISSIVA | `LICENSE` | 1103 | `29b68325fe026047...` |
| `@vitest/browser` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/pretty-format` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/utils` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/coverage-v8` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/expect` | `3.2.4` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/mocker` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/spy` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `estree-walker` | `3.0.3` | MIT | PERMISSIVA | `LICENSE` | 1126 | `8a6dcbabe7179f9c...` |
| `@vitest/pretty-format` | `3.2.4` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/runner` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/pretty-format` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/utils` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/snapshot` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/pretty-format` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/spy` | `3.2.4` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/ui` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/pretty-format` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/utils` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/utils` | `3.2.4` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `acorn` | `8.15.0` | MIT | PERMISSIVA | `LICENSE` | 1099 | `76a876cf886ff9be...` |
| `acorn-jsx` | `5.3.2` | MIT | PERMISSIVA | `LICENSE` | 1068 | `cfa72b62b9ae1730...` |
| `ajv` | `6.15.0` | MIT | PERMISSIVA | `LICENSE` | 1090 | `e85e131fa4ed2553...` |
| `ansi-regex` | `5.0.1` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `ansi-styles` | `4.3.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `ansis` | `4.4.0` | ISC | PERMISSIVA | `LICENSE` | 740 | `d80713a740590fd0...` |
| `argparse` | `2.0.1` | Python-2.0 | PERMISSIVA | `LICENSE` | 12775 | `de4d1f2d2ad5ad0c...` |
| `aria-query` | `5.3.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 10236 | `c8df456c7ccba74b...` |
| `assertion-error` | `2.0.1` | MIT | PERMISSIVA | `LICENSE` | 1108 | `2130216d5ab4c021...` |
| `ast-types` | `0.16.1` | MIT | PERMISSIVA | `LICENSE` | 1075 | `a374b737c41b30b1...` |
| `ast-v8-to-istanbul` | `0.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `estree-walker` | `3.0.3` | MIT | PERMISSIVA | `LICENSE` | 1126 | `8a6dcbabe7179f9c...` |
| `js-tokens` | `9.0.1` | MIT | PERMISSIVA | `LICENSE` | 1103 | `2213d91c606205c7...` |
| `autoprefixer` | `10.6.1` | MIT | PERMISSIVA | `LICENSE` | 1095 | `5be1f3465bba68a6...` |
| `axe-core` | `4.13.0` | MPL-2.0 | COPYLEFT_FRACO | `LICENSE` | 15921 | `af175b9d96ee93c2...` |
| `babel-dead-code-elimination` | `1.0.12` | MIT | PERMISSIVA | `LICENSE` | 1057 | `13d42724706a0703...` |
| `bail` | `2.0.2` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `balanced-match` | `1.0.2` | MIT | PERMISSIVA | `LICENSE.md` | 1096 | `a1bd5deadb6a06dd...` |
| `baseline-browser-mapping` | `2.11.25` | Apache-2.0 | PERMISSIVA | `LICENSE.txt` | 11357 | `c71d239df91726fc...` |
| `better-opn` | `3.0.2` | MIT | PERMISSIVA | `LICENSE` | 1077 | `914d187272780931...` |
| `brace-expansion` | `1.1.21` | MIT | PERMISSIVA | `LICENSE` | 1096 | `68f12f6e2c336886...` |
| `browserslist` | `4.29.0` | MIT | PERMISSIVA | `LICENSE` | 1118 | `21c2679a63d7699c...` |
| `cac` | `6.7.14` | MIT | PERMISSIVA | `LICENSE` | 1117 | `709131d7af352b81...` |
| `callsites` | `3.1.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `caniuse-lite` | `1.0.30001810` | CC-BY-4.0 | PERMISSIVA | `LICENSE` | 18651 | `fd3a263fe19ed8fa...` |
| `ccount` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `chai` | `5.2.0` | MIT | PERMISSIVA | `LICENSE` | 1082 | `b181da80336ff9dd...` |
| `chalk` | `4.1.2` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `character-entities` | `2.0.2` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `character-entities-html4` | `2.1.0` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `character-entities-legacy` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `character-reference-invalid` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `check-error` | `2.1.1` | MIT | PERMISSIVA | `LICENSE` | 1109 | `e5aee0714b6ecc41...` |
| `chokidar` | `5.0.0` | MIT | PERMISSIVA | `LICENSE` | 1124 | `bdfd5e0edb6089e6...` |
| `chromatic` | `13.3.5` | MIT | PERMISSIVA | `LICENSE` | 1072 | `ac44f6b695db4b7b...` |
| `clsx` | `2.1.1` | MIT | PERMISSIVA | `license` | 1104 | `9a9edad7baae5262...` |
| `color-convert` | `2.0.1` | MIT | PERMISSIVA | `LICENSE` | 1087 | `693866fc419c6f61...` |
| `color-name` | `1.1.4` | MIT | PERMISSIVA | `LICENSE` | 1085 | `c064f7a3e353bc1b...` |
| `comma-separated-tokens` | `2.0.3` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `commander` | `15.0.0` | MIT | PERMISSIVA | `LICENSE` | 1098 | `04512a63dce4d2d5...` |
| `concat-map` | `0.0.1` | MIT | PERMISSIVA | `LICENSE` | 1073 | `435a6722c786b0a5...` |
| `convert-source-map` | `2.0.0` | MIT | PERMISSIVA | `LICENSE` | 1078 | `1fa6ee8bb95a81ae...` |
| `cookie-es` | `3.1.1` | MIT | PERMISSIVA | `LICENSE` | 1431 | `f57270b85c392241...` |
| `cross-spawn` | `7.0.6` | MIT | PERMISSIVA | `LICENSE` | 1105 | `aaa78451b6fecd1b...` |
| `css-blank-pseudo` | `7.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `css-has-pseudo` | `7.0.3` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/selector-specificity` | `5.0.0` | MIT-0 | PERMISSIVA | `(nenhum)` | 0 | `` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `css-prefers-color-scheme` | `10.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `css.escape` | `1.5.1` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `cssdb` | `8.12.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 931 | `3083d752cad31edc...` |
| `cssesc` | `3.0.0` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `csstype` | `3.2.3` | MIT | PERMISSIVA | `LICENSE` | 1062 | `11d55bd4541c75ee...` |
| `debug` | `4.4.3` | MIT | PERMISSIVA | `LICENSE` | 1139 | `3a61c6c96caf5c1d...` |
| `decode-named-character-reference` | `1.1.0` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `deep-eql` | `5.0.2` | MIT | PERMISSIVA | `LICENSE` | 1109 | `e5aee0714b6ecc41...` |
| `deep-is` | `0.1.4` | MIT | PERMISSIVA | `LICENSE` | 1237 | `f2042f3634c4136d...` |
| `define-lazy-prop` | `2.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `dequal` | `2.0.3` | MIT | PERMISSIVA | `license` | 1114 | `306fa513e39b23a6...` |
| `detect-libc` | `2.0.4` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11357 | `b40930bbcf80744c...` |
| `devlop` | `1.1.0` | MIT | PERMISSIVA | `license` | 1099 | `6f296a5eccc1dc96...` |
| `diff` | `8.0.4` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1546 | `eda81a415869e678...` |
| `doctrine` | `3.0.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 9585 | `5c272e206669b79e...` |
| `dom-accessibility-api` | `0.5.16` | MIT | PERMISSIVA | `LICENSE.md` | 1077 | `0ffe67fe630169de...` |
| `eastasianwidth` | `0.2.0` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `electron-to-chromium` | `1.5.438` | ISC | PERMISSIVA | `LICENSE` | 728 | `25ba5c59dad3e0dd...` |
| `emoji-regex` | `9.2.2` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `enhanced-resolve` | `5.26.0` | MIT | PERMISSIVA | `LICENSE` | 1071 | `9068a8782d2fb4c6...` |
| `entities` | `6.0.1` | BSD-2-Clause | PERMISSIVA | `LICENSE` | 1260 | `cb992345949ccd6e...` |
| `es-module-lexer` | `1.7.0` | MIT | PERMISSIVA | `LICENSE` | 1095 | `fecc38b5e10d8896...` |
| `esbuild` | `0.25.5` | MIT | PERMISSIVA | `LICENSE.md` | 1069 | `b40ec5baec7bb34f...` |
| `esbuild-register` | `3.6.0` | MIT | PERMISSIVA | `LICENSE` | 1109 | `4a6ab1f41bbf070d...` |
| `escalade` | `3.2.0` | MIT | PERMISSIVA | `license` | 1104 | `9a9edad7baae5262...` |
| `escape-string-regexp` | `4.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `eslint` | `9.28.0` | MIT | PERMISSIVA | `LICENSE` | 1094 | `3b6be04f862a077a...` |
| `eslint-plugin-react-hooks` | `5.2.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `eslint-plugin-react-refresh` | `0.5.7` | MIT | PERMISSIVA | `LICENSE` | 1098 | `fba570176c68f716...` |
| `eslint-plugin-storybook` | `10.6.1` | MIT | PERMISSIVA | `LICENSE` | 1079 | `cd498eccb06619b3...` |
| `@typescript-eslint/project-service` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `053a75a14b7508a8...` |
| `@typescript-eslint/scope-manager` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/tsconfig-utils` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `053a75a14b7508a8...` |
| `@typescript-eslint/types` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/typescript-estree` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/utils` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `@typescript-eslint/visitor-keys` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `balanced-match` | `4.0.4` | MIT | PERMISSIVA | `LICENSE.md` | 1096 | `a1bd5deadb6a06dd...` |
| `brace-expansion` | `5.0.12` | MIT | PERMISSIVA | `LICENSE` | 1096 | `68f12f6e2c336886...` |
| `eslint-visitor-keys` | `5.0.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11337 | `d8bf34ff6d190640...` |
| `minimatch` | `10.2.6` | BlueOak-1.0.0 | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `semver` | `7.8.5` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `eslint-scope` | `8.4.0` | BSD-2-Clause | PERMISSIVA | `LICENSE` | 1390 | `d3a724e2ed749f17...` |
| `eslint-visitor-keys` | `4.2.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11337 | `d8bf34ff6d190640...` |
| `@eslint/js` | `9.28.0` | MIT | PERMISSIVA | `LICENSE` | 1094 | `3b6be04f862a077a...` |
| `espree` | `10.4.0` | BSD-2-Clause | PERMISSIVA | `LICENSE` | 1322 | `26c95937762a3dc1...` |
| `esprima` | `4.0.1` | BSD-2-Clause | PERMISSIVA | `LICENSE.BSD` | 1303 | `94bcb9959136723a...` |
| `esquery` | `1.6.0` | BSD-3-Clause | PERMISSIVA | `license.txt` | 1488 | `ac68116ae73740de...` |
| `esrecurse` | `4.3.0` | BSD-2-Clause | PERMISSIVA | `(nenhum)` | 0 | `` |
| `estraverse` | `5.3.0` | BSD-2-Clause | PERMISSIVA | `LICENSE.BSD` | 1231 | `0e74697a68cebdcd...` |
| `estree-util-is-identifier-name` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `estree-walker` | `2.0.2` | MIT | PERMISSIVA | `LICENSE` | 1126 | `8a6dcbabe7179f9c...` |
| `esutils` | `2.0.3` | BSD-2-Clause | PERMISSIVA | `LICENSE.BSD` | 1231 | `0e74697a68cebdcd...` |
| `expect-type` | `1.2.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 10760 | `7c6cc83c84eaa249...` |
| `extend` | `3.0.2` | MIT | PERMISSIVA | `LICENSE` | 1081 | `97af7428f93afbab...` |
| `fast-deep-equal` | `3.1.3` | MIT | PERMISSIVA | `LICENSE` | 1074 | `7bf9b2de73a6b356...` |
| `fast-json-stable-stringify` | `2.1.0` | MIT | PERMISSIVA | `LICENSE` | 1145 | `a833d366242c298c...` |
| `fast-levenshtein` | `2.0.6` | MIT | PERMISSIVA | `LICENSE.md` | 1100 | `942a98cb8846a635...` |
| `fflate` | `0.8.3` | MIT | PERMISSIVA | `LICENSE` | 1069 | `0a1df3a083d0c010...` |
| `file-entry-cache` | `8.0.0` | MIT | PERMISSIVA | `LICENSE` | 1086 | `a0e2f73971f663b4...` |
| `filesize` | `10.1.6` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1509 | `a1ac0a1c975fdadf...` |
| `find-up` | `5.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `flat-cache` | `4.0.1` | MIT | PERMISSIVA | `LICENSE` | 1088 | `a6a23fb76d91d986...` |
| `flatted` | `3.4.4` | ISC | PERMISSIVA | `LICENSE` | 770 | `148718606d34f467...` |
| `foreground-child` | `3.3.1` | ISC | PERMISSIVA | `LICENSE` | 775 | `93f5d82a8d59442f...` |
| `fraction.js` | `5.3.4` | MIT | PERMISSIVA | `LICENSE` | 1070 | `ea6916b674c353de...` |
| `framer-motion` | `12.43.0` | MIT | PERMISSIVA | `LICENSE.md` | 1078 | `5ff46083b8856382...` |
| `fsevents` | `2.3.3` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `function-bind` | `1.1.2` | MIT | PERMISSIVA | `LICENSE` | 1052 | `773e131a76847260...` |
| `gensync` | `1.0.0-beta.2` | MIT | PERMISSIVA | `LICENSE` | 1075 | `e3a956681ee067f9...` |
| `glob` | `10.5.0` | ISC | PERMISSIVA | `LICENSE` | 775 | `cdbc258d13806538...` |
| `glob-parent` | `6.0.2` | ISC | PERMISSIVA | `LICENSE` | 857 | `fc68eb8f1c8e1d6b...` |
| `brace-expansion` | `2.1.7` | MIT | PERMISSIVA | `LICENSE` | 1096 | `68f12f6e2c336886...` |
| `minimatch` | `9.0.9` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `globals` | `16.5.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `globrex` | `0.1.2` | MIT | PERMISSIVA | `license` | 1079 | `ade66f698fd417ad...` |
| `graceful-fs` | `4.2.11` | ISC | PERMISSIVA | `LICENSE` | 791 | `f65c5d9f22a317b2...` |
| `has-flag` | `4.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `hasown` | `2.0.2` | MIT | PERMISSIVA | `LICENSE` | 1083 | `bf9b0d665be2a689...` |
| `hast-util-from-dom` | `5.0.1` | ISC | PERMISSIVA | `license` | 760 | `c01b44788e1829e8...` |
| `hast-util-from-html` | `2.0.3` | MIT | PERMISSIVA | `license` | 1099 | `e620971c29d750df...` |
| `hast-util-from-html-isomorphic` | `2.0.0` | MIT | PERMISSIVA | `license` | 1101 | `fb5678944a6345b1...` |
| `hast-util-from-parse5` | `8.0.3` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `hast-util-is-element` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `hast-util-parse-selector` | `4.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `hast-util-raw` | `9.1.0` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `hast-util-sanitize` | `5.0.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `hast-util-to-html` | `9.0.5` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `hast-util-to-jsx-runtime` | `2.3.6` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `hast-util-to-parse5` | `8.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `property-information` | `6.5.0` | MIT | PERMISSIVA | `license` | 1101 | `5d0698266086f217...` |
| `hast-util-to-string` | `3.0.1` | MIT | PERMISSIVA | `license` | 1070 | `e18cf5173e928f80...` |
| `hast-util-to-text` | `4.0.2` | MIT | PERMISSIVA | `license` | 1099 | `691115fe0234b1f0...` |
| `hast-util-whitespace` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `hastscript` | `9.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `html-escaper` | `2.0.2` | MIT | PERMISSIVA | `LICENSE.txt` | 1089 | `27d06bbb2eb031be...` |
| `html-url-attributes` | `3.0.1` | MIT | PERMISSIVA | `license` | 1070 | `e18cf5173e928f80...` |
| `html-void-elements` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `ignore` | `5.3.2` | MIT | PERMISSIVA | `LICENSE-MIT` | 1095 | `9c94db23dc4b1e9a...` |
| `import-fresh` | `3.3.1` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `imurmurhash` | `0.1.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `indent-string` | `4.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `inline-style-parser` | `0.2.4` | MIT | PERMISSIVA | `LICENSE` | 1098 | `395c95f20598da65...` |
| `is-alphabetical` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `is-alphanumerical` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `is-core-module` | `2.16.1` | MIT | PERMISSIVA | `LICENSE` | 1078 | `5c496ce5ae47eb8e...` |
| `is-decimal` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `is-docker` | `2.2.1` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `is-extglob` | `2.1.1` | MIT | PERMISSIVA | `LICENSE` | 1087 | `73b5283588baa142...` |
| `is-fullwidth-code-point` | `3.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `is-glob` | `4.0.3` | MIT | PERMISSIVA | `LICENSE` | 1088 | `4cd903859549d4b2...` |
| `is-hexadecimal` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `is-plain-obj` | `4.1.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `is-wsl` | `2.2.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `isbot` | `5.2.2` | Unlicense | PERMISSIVA | `LICENSE` | 1224 | `472f8bb212250d2b...` |
| `isexe` | `2.0.0` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `istanbul-lib-coverage` | `3.2.2` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1488 | `e09f9b52c6ce5c17...` |
| `istanbul-lib-report` | `3.0.1` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1488 | `e09f9b52c6ce5c17...` |
| `istanbul-lib-source-maps` | `5.0.6` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1483 | `b661867c1d66a1b7...` |
| `istanbul-reports` | `3.1.7` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1488 | `e09f9b52c6ce5c17...` |
| `jackspeak` | `3.4.3` | BlueOak-1.0.0 | PERMISSIVA | `LICENSE.md` | 1550 | `2c7c5d22ed5a8ee9...` |
| `jiti` | `2.7.0` | MIT | PERMISSIVA | `LICENSE` | 1078 | `46231df5a7733c3f...` |
| `js-tokens` | `4.0.0` | MIT | PERMISSIVA | `LICENSE` | 1103 | `2213d91c606205c7...` |
| `js-yaml` | `4.3.2` | MIT | PERMISSIVA | `LICENSE` | 1084 | `a07bc24468b9654c...` |
| `jsesc` | `3.1.0` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `json-buffer` | `3.0.1` | MIT | PERMISSIVA | `LICENSE` | 1071 | `715f1f0f2eb7688e...` |
| `json-schema-traverse` | `0.4.1` | MIT | PERMISSIVA | `LICENSE` | 1074 | `7bf9b2de73a6b356...` |
| `json-stable-stringify-without-jsonify` | `1.0.1` | MIT | PERMISSIVA | `LICENSE` | 1073 | `435a6722c786b0a5...` |
| `json5` | `2.2.3` | MIT | PERMISSIVA | `LICENSE.md` | 1145 | `53e59feb13058722...` |
| `jsonfile` | `6.1.0` | MIT | PERMISSIVA | `LICENSE` | 1110 | `a9801fb52ba22ef8...` |
| `katex` | `0.19.0` | MIT | PERMISSIVA | `LICENSE` | 1107 | `766ccc1f306c885a...` |
| `keyv` | `4.5.4` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `kleur` | `3.0.3` | MIT | PERMISSIVA | `license` | 1114 | `306fa513e39b23a6...` |
| `levn` | `0.4.1` | MIT | PERMISSIVA | `LICENSE` | 1054 | `b9eb082c39fe245e...` |
| `lightningcss` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `LICENSE` | 15648 | `5eba353fe5076ac3...` |
| `lightningcss-android-arm64` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-darwin-arm64` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-darwin-x64` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-freebsd-x64` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-linux-arm-gnueabihf` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-linux-arm64-gnu` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-linux-arm64-musl` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-linux-x64-gnu` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-linux-x64-musl` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-win32-arm64-msvc` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `(nenhum)` | 0 | `` |
| `lightningcss-win32-x64-msvc` | `1.32.0` | MPL-2.0 | COPYLEFT_FRACO | `LICENSE` | 15648 | `5eba353fe5076ac3...` |
| `locate-path` | `6.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `lodash` | `4.18.1` | MIT | PERMISSIVA | `LICENSE` | 1952 | `f71e8ed126b46346...` |
| `lodash.merge` | `4.6.2` | MIT | PERMISSIVA | `LICENSE` | 1952 | `f71e8ed126b46346...` |
| `longest-streak` | `3.1.0` | MIT | PERMISSIVA | `license` | 1106 | `9f084fac69d8cf1e...` |
| `loupe` | `3.1.4` | MIT | PERMISSIVA | `LICENSE` | 1102 | `592141f7b861426c...` |
| `lru-cache` | `5.1.1` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `lz-string` | `1.5.0` | MIT | PERMISSIVA | `LICENSE` | 1064 | `433fc9dfe659dbfb...` |
| `magic-string` | `0.30.21` | MIT | PERMISSIVA | `LICENSE` | 1051 | `1cbe51b907662f6c...` |
| `magicast` | `0.3.5` | MIT | PERMISSIVA | `LICENSE` | 1120 | `7917b04272f3f7c1...` |
| `make-dir` | `4.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `semver` | `7.7.2` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `markdown-table` | `3.0.4` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `marked` | `17.0.6` | MIT | PERMISSIVA | `LICENSE.md` | 2942 | `8e3a3f82f59a6095...` |
| `mdast-util-find-and-replace` | `3.0.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `escape-string-regexp` | `5.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `mdast-util-from-markdown` | `2.0.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `mdast-util-gfm` | `3.1.0` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `mdast-util-gfm-footnote` | `2.1.0` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `mdast-util-gfm-strikethrough` | `2.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-gfm-table` | `2.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-gfm-task-list-item` | `2.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-gfm-autolink-literal` | `2.0.0` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `mdast-util-math` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-mdx-expression` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-mdx-jsx` | `3.2.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-mdxjs-esm` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `mdast-util-phrasing` | `4.1.0` | MIT | PERMISSIVA | `license` | 1150 | `b6d1a4e1831b177a...` |
| `mdast-util-to-hast` | `13.2.1` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `mdast-util-to-markdown` | `2.1.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `mdast-util-to-string` | `4.0.0` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `micromark` | `4.0.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-core-commonmark` | `2.0.3` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-extension-gfm` | `3.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `micromark-extension-gfm-autolink-literal` | `2.1.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `micromark-extension-gfm-footnote` | `2.1.0` | MIT | PERMISSIVA | `license` | 1099 | `807a88522a850092...` |
| `micromark-extension-gfm-strikethrough` | `2.1.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `micromark-extension-gfm-table` | `2.1.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-extension-gfm-tagfilter` | `2.0.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `micromark-extension-gfm-task-list-item` | `2.1.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `micromark-extension-llm-math` | `3.1.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `commander` | `8.3.0` | MIT | PERMISSIVA | `LICENSE` | 1098 | `04512a63dce4d2d5...` |
| `katex` | `0.16.47` | MIT | PERMISSIVA | `LICENSE` | 1107 | `766ccc1f306c885a...` |
| `micromark-extension-math` | `3.1.0` | MIT | PERMISSIVA | `license` | 1099 | `536804b3adbff17d...` |
| `commander` | `8.3.0` | MIT | PERMISSIVA | `LICENSE` | 1098 | `04512a63dce4d2d5...` |
| `katex` | `0.16.47` | MIT | PERMISSIVA | `LICENSE` | 1107 | `766ccc1f306c885a...` |
| `micromark-factory-destination` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-factory-label` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-factory-space` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-factory-title` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-factory-whitespace` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-character` | `2.1.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-chunked` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-classify-character` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-combine-extensions` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-decode-numeric-character-reference` | `2.0.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-decode-string` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-encode` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-html-tag-name` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-normalize-identifier` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-resolve-all` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-sanitize-uri` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-subtokenize` | `2.1.0` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-symbol` | `2.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `micromark-util-types` | `2.0.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `min-indent` | `1.0.1` | MIT | PERMISSIVA | `license` | 1173 | `9638fa87f845af6c...` |
| `minimatch` | `3.1.5` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `minimist` | `1.2.8` | MIT | PERMISSIVA | `LICENSE` | 1073 | `435a6722c786b0a5...` |
| `minipass` | `7.1.2` | ISC | PERMISSIVA | `LICENSE` | 787 | `833faa18ac4b83a6...` |
| `motion-dom` | `12.43.0` | MIT | PERMISSIVA | `LICENSE.md` | 1100 | `1bf0dc3f7727723e...` |
| `motion-utils` | `12.39.0` | MIT | PERMISSIVA | `LICENSE.md` | 1100 | `1bf0dc3f7727723e...` |
| `mrmime` | `2.0.1` | MIT | PERMISSIVA | `license` | 1122 | `ba573393f24555ac...` |
| `ms` | `2.1.3` | MIT | PERMISSIVA | `license.md` | 1079 | `1662fae9b5314d11...` |
| `nanoid` | `3.3.19` | MIT | PERMISSIVA | `LICENSE` | 1095 | `da4db1480d9beea3...` |
| `natural-compare` | `1.4.0` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `node-releases` | `2.0.57` | MIT | PERMISSIVA | `LICENSE` | 1107 | `3706296ed6118881...` |
| `ollama` | `0.6.4` | MIT | PERMISSIVA | `LICENSE` | 1061 | `6687182ea5e41cd2...` |
| `oniguruma-parser` | `0.12.2` | MIT | PERMISSIVA | `LICENSE` | 1077 | `5fe78c91b36c6147...` |
| `oniguruma-to-es` | `4.3.6` | MIT | PERMISSIVA | `LICENSE` | 1077 | `c1a5a1ebc88f3e0f...` |
| `open` | `8.4.2` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `optionator` | `0.9.4` | MIT | PERMISSIVA | `LICENSE` | 1054 | `b9eb082c39fe245e...` |
| `p-limit` | `3.1.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `p-locate` | `5.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `package-json-from-dist` | `1.0.1` | BlueOak-1.0.0 | PERMISSIVA | `LICENSE.md` | 1764 | `a49c9ba464796f65...` |
| `parent-module` | `1.0.1` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `parse-entities` | `4.0.2` | MIT | PERMISSIVA | `license` | 1101 | `5d0698266086f217...` |
| `@types/unist` | `2.0.11` | MIT | PERMISSIVA | `LICENSE` | 1141 | `c2cfccb812fe4821...` |
| `parse-numeric-range` | `1.3.0` | ISC | PERMISSIVA | `LICENSE` | 742 | `53a3985388c606d7...` |
| `parse5` | `7.3.0` | MIT | PERMISSIVA | `LICENSE` | 1109 | `8c535800331e1e44...` |
| `path-exists` | `4.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `path-key` | `3.1.1` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `path-parse` | `1.0.7` | MIT | PERMISSIVA | `LICENSE` | 1080 | `3f4609137f650705...` |
| `path-scurry` | `1.11.1` | BlueOak-1.0.0 | PERMISSIVA | `LICENSE.md` | 1552 | `8a1af140fdfbf5af...` |
| `lru-cache` | `10.4.3` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `pathe` | `2.0.3` | MIT | PERMISSIVA | `LICENSE` | 3355 | `52e92576851154ba...` |
| `pathval` | `2.0.1` | MIT | PERMISSIVA | `LICENSE` | 1096 | `5d45910e7ed82fec...` |
| `picocolors` | `1.1.1` | ISC | PERMISSIVA | `LICENSE` | 787 | `6582629e29794668...` |
| `picomatch` | `4.0.7` | MIT | PERMISSIVA | `LICENSE` | 1091 | `d0cd141b0c322fde...` |
| `playwright` | `1.64.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11601 | `45873d00a0dd2435...` |
| `playwright-core` | `1.64.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 11601 | `45873d00a0dd2435...` |
| `postcss` | `8.5.28` | MIT | PERMISSIVA | `LICENSE` | 1095 | `5be1f3465bba68a6...` |
| `postcss-attribute-case-insensitive` | `7.0.1` | MIT | PERMISSIVA | `LICENSE.md` | 1106 | `fa840a2d897165d2...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-clamp` | `4.1.0` | MIT | PERMISSIVA | `LICENSE` | 1105 | `f3c4f4b59a2222f3...` |
| `postcss-color-functional-notation` | `7.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-color-hex-alpha` | `10.0.0` | MIT | PERMISSIVA | `LICENSE.md` | 1070 | `111e97d86cfc6d19...` |
| `postcss-color-rebeccapurple` | `10.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-custom-media` | `11.0.6` | MIT | PERMISSIVA | `LICENSE.md` | 1070 | `111e97d86cfc6d19...` |
| `postcss-custom-properties` | `14.0.6` | MIT | PERMISSIVA | `LICENSE.md` | 1070 | `111e97d86cfc6d19...` |
| `postcss-custom-selectors` | `8.0.5` | MIT | PERMISSIVA | `LICENSE.md` | 1070 | `111e97d86cfc6d19...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-dir-pseudo-class` | `9.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-double-position-gradients` | `6.0.4` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-focus-visible` | `10.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-focus-within` | `9.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-font-variant` | `5.0.0` | MIT | PERMISSIVA | `LICENSE` | 1101 | `8e285253dc1c6277...` |
| `postcss-gap-properties` | `6.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-image-set-function` | `7.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-lab-function` | `7.0.12` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-logical` | `8.1.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-nesting` | `13.0.2` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `@csstools/selector-resolve-nested` | `3.1.0` | MIT-0 | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@csstools/selector-specificity` | `5.0.0` | MIT-0 | PERMISSIVA | `(nenhum)` | 0 | `` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-opacity-percentage` | `3.0.0` | MIT | PERMISSIVA | `LICENSE.md` | 1103 | `6c46d3c24b905d6a...` |
| `postcss-overflow-shorthand` | `6.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-page-break` | `3.0.4` | MIT | PERMISSIVA | `LICENSE` | 1089 | `64272a7b0e7c77de...` |
| `postcss-place` | `10.0.0` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-preset-env` | `10.6.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-pseudo-class-any-link` | `10.0.1` | MIT-0 | PERMISSIVA | `LICENSE.md` | 930 | `947e32047a166cd0...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-replace-overflow-wrap` | `4.0.0` | MIT | PERMISSIVA | `LICENSE` | 1135 | `f40a75eea4b81d45...` |
| `postcss-selector-not` | `8.0.1` | MIT | PERMISSIVA | `LICENSE.md` | 1082 | `120ebda8ae7e7156...` |
| `postcss-selector-parser` | `7.1.6` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-selector-parser` | `6.0.10` | MIT | PERMISSIVA | `LICENSE-MIT` | 1092 | `2998094b38f7ace2...` |
| `postcss-value-parser` | `4.2.0` | MIT | PERMISSIVA | `LICENSE` | 1074 | `3687447039151857...` |
| `prelude-ls` | `1.2.1` | MIT | PERMISSIVA | `LICENSE` | 1054 | `b9eb082c39fe245e...` |
| `prettier` | `3.9.10` | MIT | PERMISSIVA | `LICENSE` | 1065 | `b0f2417199889f1c...` |
| `pretty-format` | `27.5.1` | MIT | PERMISSIVA | `LICENSE` | 1086 | `52412d7bc7ce4157...` |
| `ansi-styles` | `5.2.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `prompts` | `2.4.2` | MIT | PERMISSIVA | `license` | 1079 | `ade66f698fd417ad...` |
| `property-information` | `7.1.0` | MIT | PERMISSIVA | `license` | 1101 | `5d0698266086f217...` |
| `punycode` | `2.3.1` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `react` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `react-docgen` | `8.0.0` | MIT | PERMISSIVA | `LICENSE` | 1086 | `52412d7bc7ce4157...` |
| `react-docgen-typescript` | `2.4.0` | MIT | PERMISSIVA | `LICENSE.md` | 1076 | `a61b261379cbd8e1...` |
| `react-dom` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `react-is` | `17.0.2` | MIT | PERMISSIVA | `LICENSE` | 1086 | `52412d7bc7ce4157...` |
| `react-markdown` | `10.1.0` | MIT | PERMISSIVA | `license` | 1079 | `f6196c64e144f9a6...` |
| `react-refresh` | `0.17.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `react-test-renderer` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `react-is` | `19.3.0` | MIT | PERMISSIVA | `LICENSE` | 1086 | `52412d7bc7ce4157...` |
| `readdirp` | `5.1.1` | MIT | PERMISSIVA | `LICENSE` | 1114 | `dffec71d93f273d2...` |
| `recast` | `0.23.11` | MIT | PERMISSIVA | `LICENSE` | 1075 | `6ca87d1dce5b8287...` |
| `redent` | `3.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `strip-indent` | `3.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `refractor` | `5.0.0` | MIT | PERMISSIVA | `license` | 1118 | `4b74d4b7b5ec11a2...` |
| `regex` | `6.1.0` | MIT | PERMISSIVA | `LICENSE` | 1072 | `3d19c2b6246b3072...` |
| `regex-recursion` | `6.0.2` | MIT | PERMISSIVA | `LICENSE` | 1072 | `3d19c2b6246b3072...` |
| `regex-utilities` | `2.3.0` | MIT | PERMISSIVA | `LICENSE` | 1072 | `8ca7af4f6c753e5a...` |
| `rehype-harden` | `1.1.8` | MIT | PERMISSIVA | `LICENSE.md` | 1068 | `19a348dadee07bc9...` |
| `rehype-katex` | `7.0.1` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `commander` | `8.3.0` | MIT | PERMISSIVA | `LICENSE` | 1098 | `04512a63dce4d2d5...` |
| `katex` | `0.16.47` | MIT | PERMISSIVA | `LICENSE` | 1107 | `766ccc1f306c885a...` |
| `rehype-parse` | `9.0.1` | MIT | PERMISSIVA | `license` | 1070 | `e18cf5173e928f80...` |
| `rehype-prism-plus` | `2.0.2` | MIT | PERMISSIVA | `LICENSE` | 1064 | `14125ed4d9663de7...` |
| `rehype-raw` | `7.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `rehype-sanitize` | `6.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `remark` | `15.0.1` | MIT | PERMISSIVA | `license` | 1099 | `413654b0a78b9866...` |
| `remark-gfm` | `4.0.1` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `remark-math` | `6.0.0` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `remark-parse` | `11.0.0` | MIT | PERMISSIVA | `license` | 1099 | `413654b0a78b9866...` |
| `remark-rehype` | `11.1.2` | MIT | PERMISSIVA | `license` | 1094 | `dd1081884a929528...` |
| `remark-stringify` | `11.0.0` | MIT | PERMISSIVA | `license` | 1099 | `413654b0a78b9866...` |
| `remend` | `1.3.1` | Apache-2.0 | PERMISSIVA | `LICENSE` | 552 | `b4f9adb7c5689048...` |
| `resolve` | `1.22.10` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60acd7711e6c138e...` |
| `resolve-from` | `4.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `rollup` | `4.63.4` | MIT | PERMISSIVA | `LICENSE.md` | 35231 | `fa1bd040c5bdeefe...` |
| `scheduler` | `0.28.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `semver` | `6.3.1` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `seroval` | `1.6.9` | MIT | PERMISSIVA | `LICENSE` | 1129 | `2bd8a2c61751fc47...` |
| `seroval-plugins` | `1.6.9` | MIT | PERMISSIVA | `LICENSE` | 1129 | `2bd8a2c61751fc47...` |
| `shebang-command` | `2.0.0` | MIT | PERMISSIVA | `license` | 1116 | `69dee148a2cc4705...` |
| `shebang-regex` | `3.0.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `shiki` | `3.23.0` | MIT | PERMISSIVA | `LICENSE` | 1121 | `7a9d8d01038aeacf...` |
| `siginfo` | `2.0.0` | ISC | PERMISSIVA | `LICENSE` | 744 | `3bdddf0b9b08aaa2...` |
| `signal-exit` | `4.1.0` | ISC | PERMISSIVA | `LICENSE.txt` | 790 | `b173e19b9a78df30...` |
| `simple-icons` | `16.34.0` | CC0-1.0 | PERMISSIVA | `LICENSE.md` | 6569 | `9046848b63a5c92b...` |
| `sirv` | `3.0.1` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `sisteransi` | `1.0.5` | MIT | PERMISSIVA | `license` | 1079 | `ade66f698fd417ad...` |
| `source-map` | `0.6.1` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1526 | `6cb0631f71c77497...` |
| `source-map-js` | `1.2.1` | BSD-3-Clause | PERMISSIVA | `LICENSE` | 1526 | `6cb0631f71c77497...` |
| `space-separated-tokens` | `2.0.2` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `stackback` | `0.0.2` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `std-env` | `3.9.0` | MIT | PERMISSIVA | `LICENCE` | 1079 | `a6f36438e46fb911...` |
| `storybook` | `9.1.20` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `@vitest/mocker` | `3.2.4` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `estree-walker` | `3.0.3` | MIT | PERMISSIVA | `LICENSE` | 1126 | `8a6dcbabe7179f9c...` |
| `semver` | `7.7.2` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `streamdown` | `2.6.0` | Apache-2.0 | PERMISSIVA | `LICENSE` | 552 | `b4f9adb7c5689048...` |
| `string-width` | `5.1.2` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `string-width-cjs` | `4.2.3` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `emoji-regex` | `8.0.0` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `strip-ansi` | `6.0.1` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `stringify-entities` | `4.0.4` | MIT | PERMISSIVA | `license` | 1106 | `9f084fac69d8cf1e...` |
| `strip-ansi` | `7.1.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `strip-ansi-cjs` | `6.0.1` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `ansi-regex` | `6.1.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `strip-bom` | `3.0.0` | MIT | PERMISSIVA | `license` | 1119 | `6fb9754611c20f66...` |
| `strip-indent` | `4.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `strip-json-comments` | `3.1.1` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `strip-literal` | `3.0.0` | MIT | PERMISSIVA | `LICENSE` | 1094 | `af32efdd008a371c...` |
| `js-tokens` | `9.0.1` | MIT | PERMISSIVA | `LICENSE` | 1103 | `2213d91c606205c7...` |
| `style-to-js` | `1.1.16` | MIT | PERMISSIVA | `LICENSE` | 1110 | `bbb380e5c1a71707...` |
| `style-to-object` | `1.0.8` | MIT | PERMISSIVA | `LICENSE` | 1110 | `896c9fd6b5fc48ac...` |
| `supports-color` | `7.2.0` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `supports-preserve-symlinks-flag` | `1.0.0` | MIT | PERMISSIVA | `LICENSE` | 1067 | `0f0ca96f50793990...` |
| `tabbable` | `6.2.0` | MIT | PERMISSIVA | `LICENSE` | 1079 | `8e714750725e75c8...` |
| `tailwind-merge` | `3.7.0` | MIT | PERMISSIVA | `LICENSE.md` | 1070 | `d4c70c7ce38cea87...` |
| `tailwindcss` | `4.3.3` | MIT | PERMISSIVA | `LICENSE` | 1071 | `60e0b68c0f35c078...` |
| `tapable` | `2.3.3` | MIT | PERMISSIVA | `LICENSE` | 1088 | `824ef6c33ae5ecbb...` |
| `test-exclude` | `7.0.1` | ISC | PERMISSIVA | `LICENSE.txt` | 731 | `365496ca1f56da40...` |
| `brace-expansion` | `2.1.7` | MIT | PERMISSIVA | `LICENSE` | 1096 | `68f12f6e2c336886...` |
| `minimatch` | `9.0.9` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `tiny-invariant` | `1.3.3` | MIT | PERMISSIVA | `LICENSE` | 1073 | `b1c501095de70182...` |
| `tinybench` | `2.9.0` | MIT | PERMISSIVA | `LICENSE` | 1065 | `cebc084d54e6dd99...` |
| `tinyexec` | `0.3.2` | MIT | PERMISSIVA | `LICENSE` | 1065 | `f95f668fe64081dd...` |
| `tinyglobby` | `0.2.17` | MIT | PERMISSIVA | `LICENSE` | 1076 | `22c68811e174cbbf...` |
| `fdir` | `6.5.0` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `tinypool` | `1.1.1` | MIT | PERMISSIVA | `LICENSE` | 1209 | `0d506ca1b2adc4ee...` |
| `tinyrainbow` | `2.0.0` | MIT | PERMISSIVA | `LICENCE` | 1065 | `cebc084d54e6dd99...` |
| `tinyspy` | `4.0.3` | MIT | PERMISSIVA | `LICENCE` | 1065 | `cebc084d54e6dd99...` |
| `totalist` | `3.0.1` | MIT | PERMISSIVA | `license` | 1114 | `306fa513e39b23a6...` |
| `trim-lines` | `3.0.1` | MIT | PERMISSIVA | `license` | 1106 | `9f084fac69d8cf1e...` |
| `trough` | `2.2.0` | MIT | PERMISSIVA | `license` | 1099 | `6c03fd41cfd7c92d...` |
| `ts-api-utils` | `2.5.0` | MIT | PERMISSIVA | `LICENSE.md` | 1038 | `76c9d5f5ea942054...` |
| `ts-dedent` | `2.2.0` | MIT | PERMISSIVA | `LICENSE` | 1073 | `eff13095d9d762c3...` |
| `tsconfck` | `3.1.6` | MIT | PERMISSIVA | `LICENSE` | 2360 | `483edf639466c35b...` |
| `tsconfig-paths` | `4.2.0` | MIT | PERMISSIVA | `LICENSE` | 1078 | `8850a2d482851411...` |
| `tslib` | `2.8.1` | 0BSD | PERMISSIVA | `LICENSE.txt` | 655 | `210b19e543130388...` |
| `type-check` | `0.4.0` | MIT | PERMISSIVA | `LICENSE` | 1054 | `b9eb082c39fe245e...` |
| `typescript` | `5.9.3` | Apache-2.0 | PERMISSIVA | `LICENSE.txt` | 9197 | `a7d00bfd54525bc6...` |
| `typescript-eslint` | `8.71.1` | MIT | PERMISSIVA | `LICENSE` | 1097 | `2eb5c7a0bba9deb7...` |
| `undici-types` | `7.24.6` | MIT | PERMISSIVA | `LICENSE` | 1090 | `a6db8096b2707bc0...` |
| `unicorn-magic` | `0.1.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `unified` | `11.0.5` | MIT | PERMISSIVA | `license` | 1099 | `05811400116ed61f...` |
| `unist-builder` | `4.0.0` | MIT | PERMISSIVA | `license` | 1082 | `3279c1ba2a32073f...` |
| `unist-util-filter` | `5.0.1` | MIT | PERMISSIVA | `license` | 1082 | `c6e4778fe285e5a5...` |
| `unist-util-find-after` | `5.0.0` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `unist-util-is` | `6.0.0` | MIT | PERMISSIVA | `license` | 1099 | `82974dbf2639d13e...` |
| `unist-util-parents` | `3.0.0` | MIT | PERMISSIVA | `license` | 1082 | `3279c1ba2a32073f...` |
| `unist-util-position` | `5.0.0` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `unist-util-remove-position` | `5.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `unist-util-stringify-position` | `4.0.0` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `unist-util-visit` | `5.1.0` | MIT | PERMISSIVA | `license` | 1099 | `63cb98b3f6abfb3c...` |
| `unist-util-visit-parents` | `6.0.2` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `universalify` | `2.0.1` | MIT | PERMISSIVA | `LICENSE` | 1100 | `3fda5977c0904e22...` |
| `unplugin` | `3.4.0` | MIT | PERMISSIVA | `LICENSE` | 1077 | `2ef2cbeeb84d5e7a...` |
| `update-browserslist-db` | `1.3.3` | MIT | PERMISSIVA | `LICENSE` | 1118 | `9877e90da14aa8c9...` |
| `uri-js` | `4.4.1` | BSD-2-Clause | PERMISSIVA | `LICENSE` | 1452 | `0af366eff4c01ec1...` |
| `use-sync-external-store` | `1.7.0` | MIT | PERMISSIVA | `LICENSE` | 1088 | `da6d3703ed11cbe4...` |
| `util-deprecate` | `1.0.2` | MIT | PERMISSIVA | `LICENSE` | 1102 | `0154425673db15cd...` |
| `vfile` | `6.0.3` | MIT | PERMISSIVA | `license` | 1099 | `05811400116ed61f...` |
| `vfile-location` | `5.0.3` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `vfile-message` | `4.0.2` | MIT | PERMISSIVA | `license` | 1099 | `d8fc337aaf98fc7a...` |
| `vite` | `6.4.4` | MIT | PERMISSIVA | `LICENSE.md` | 115624 | `008938a3441801ca...` |
| `vite-node` | `3.2.4` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `vite-tsconfig-paths` | `5.1.4` | MIT | PERMISSIVA | `LICENSE` | 1063 | `76822e5dea6731e1...` |
| `fdir` | `6.4.5` | MIT | PERMISSIVA | `(nenhum)` | 0 | `` |
| `vitest` | `3.2.7` | MIT | PERMISSIVA | `LICENSE.md` | 34974 | `6b861d7554fbc848...` |
| `@vitest/expect` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/pretty-format` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/spy` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `@vitest/utils` | `3.2.7` | MIT | PERMISSIVA | `LICENSE` | 1076 | `ddc110f3b89cc397...` |
| `web-namespaces` | `2.0.1` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |
| `webpack-virtual-modules` | `0.6.2` | MIT | PERMISSIVA | `LICENSE` | 1065 | `aa523c5b0d9c0223...` |
| `whatwg-fetch` | `3.6.20` | MIT | PERMISSIVA | `LICENSE` | 1061 | `c887d1c97797646d...` |
| `which` | `2.0.2` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `why-is-node-running` | `2.3.0` | MIT | PERMISSIVA | `LICENSE` | 1079 | `6a134e51aa31496c...` |
| `word-wrap` | `1.2.5` | MIT | PERMISSIVA | `LICENSE` | 1087 | `73b5283588baa142...` |
| `wrap-ansi` | `8.1.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `wrap-ansi-cjs` | `7.0.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `emoji-regex` | `8.0.0` | MIT | PERMISSIVA | `LICENSE-MIT.txt` | 1077 | `483acb265f182907...` |
| `string-width` | `4.2.3` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `strip-ansi` | `6.0.1` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `ansi-styles` | `6.2.1` | MIT | PERMISSIVA | `license` | 1109 | `48da2f39e100d408...` |
| `ws` | `8.21.3` | MIT | PERMISSIVA | `LICENSE` | 1183 | `2b29dcfe0d6471f7...` |
| `yallist` | `3.1.1` | ISC | PERMISSIVA | `LICENSE` | 765 | `4ec3d4c66cd87f5c...` |
| `yocto-queue` | `0.1.0` | MIT | PERMISSIVA | `license` | 1117 | `5c932d88256b4ab9...` |
| `zod` | `4.6.5` | MIT | PERMISSIVA | `LICENSE` | 1072 | `3f1189b28e3866e0...` |
| `zwitch` | `2.0.4` | MIT | PERMISSIVA | `license` | 1099 | `ca4662cb5d1b738f...` |

### npm: SEM licenca no lockfile, no package.json e SEM ficheiro no disco (0)


## Escopo declarado

* A classificacao Go usa o **texto inteiro** do ficheiro de licenca; a do npm usa o campo
  `license` do lockfile ou do `package.json`. **Nao** e' auditoria SPDX nem parecer juridico.
* **Nao** cobre licenca de **modelos**, de `assets/`, nem dos binarios do runner (ver `dist/`).
* A decisao final de **compatibilidade com a MIT deste projecto** continua **por fazer**:
  este documento diz **onde olhar**.

## Reproduzir

```
python scripts/hades_third_party_notices.py
```
