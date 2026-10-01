# Hades — kit de marca (dois modos + vetor)

Emblema vetorizado a partir da line-art limpa. **Fonte da verdade = SVG** (nítido em
qualquer tamanho). Os PNGs são derivados para quem precisa de bitmap pronto.

## Cores
| Token            | Hex       | Uso                                   |
|------------------|-----------|---------------------------------------|
| Navy (tile/bg)   | `#0B0F17` | fundo escuro, tile do ícone, theme    |
| Verde            | `#36D696` | destaque, início do gradiente         |
| Azul             | `#1E66E0` | fim do gradiente                      |

## Logo (`logo/`)
- `hades-emblem.svg` — **canônico**. Usa `fill: currentColor`: recolore pelo tema via CSS
  (`color:#0B0F17` no claro, `color:#fff` no escuro). É o arquivo que você usa na UI.
- `hades-emblem-gradient.svg` — verde→azul, para peças de divulgação sobre fundo escuro.
- `hades-emblem-black.svg` / `hades-emblem-white.svg` — sólidos prontos (claro / escuro).
- `*-1024.png` — rasters 1024px transparentes (preto, branco, gradiente).

## Dois modos (claro / escuro)
- **Modo claro** → `hades-emblem.svg` com `color:#0B0F17` (ou `hades-emblem-black.svg`).
- **Modo escuro** → `hades-emblem.svg` com `color:#fff` (ou `hades-emblem-white.svg` / gradiente).
- **Ícone do app** é único (emblema no tile navy) e funciona nos dois temas do SO.

## Ícones (`icons/`)
`hades-icon-{16,32,48,64,128,180,192,256,512}.png` (tile navy arredondado),
`apple-touch-icon.png` (180, sem transparência), `hades-maskable-512.png` (PWA full-bleed).
Nos tamanhos ≤48 o traço é engrossado para manter legibilidade.

## Favicon (`favicon/`)
- `favicon.svg` — **preferido** (vetor, nítido). Emblema gradiente no tile navy.
- `favicon.ico` — 16/32/48 com bitmap próprio por tamanho (traço reforçado).

## `<head>` sugerido
```html
<link rel="icon" href="/favicon/favicon.svg" type="image/svg+xml">
<link rel="icon" href="/favicon/favicon.ico" sizes="16x16 32x32 48x48">
<link rel="apple-touch-icon" href="/icons/apple-touch-icon.png">
<link rel="manifest" href="/manifest.json">
<meta name="theme-color" content="#0B0F17">
<meta property="og:image" content="/social/hades-og-1200x630.png">
<meta property="og:title" content="Hades">
<meta property="og:description" content="Agente de IA local — paridade Manus.">
```

## Social (`social/`)
`hades-og-1200x630.png` — card Open Graph (Twitter/LinkedIn/WhatsApp).

> Nota honesta: em 16px um emblema com tanto detalhe fica naturalmente macio; por isso o
> `favicon.svg` (vetor) é o preferido e o `.ico` 16px usa traço reforçado. Para recolorir por
> tema use `hades-emblem.svg` (currentColor) — um único arquivo cobre claro e escuro.
