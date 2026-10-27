# blup

Transforma um vault do Obsidian (ou uma pasta dele) em um site Hugo rápido, com SEO e imagem OG, e publica no GitHub Pages ou na Cloudflare. Tudo guiado por perguntas, em português ou inglês.

```sh
go build -o bin/blup ./cmd/blup
./bin/blup          # abre o menu
```

## Como funciona

- O site **lê o vault direto** (mounts do Hugo): nada é copiado. Cada nota vira página e cada subpasta vira seção.
- O assistente ajusta o Obsidian para links Markdown relativos e converte os `[[wikilinks]]` antigos, com backup.
- O Hugo é baixado sozinho na primeira vez (versão fixada em `internal/hugo`, com conferência de SHA-256).
- Dois modelos: `blog` e `docs`, no mesmo tema (`theme/`, embutido no binário).

## Estrutura

| Pasta | O quê |
| --- | --- |
| `cmd/blup` | ponto de entrada |
| `internal/app` | fluxos: menu, assistente, prévia, publicação |
| `internal/i18n` | todos os textos (EN/PT) |
| `internal/obsidian` | vaults, configurações do Obsidian |
| `internal/wikilinks` | conversão de `[[wikilinks]]` |
| `internal/scaffold` | gera `hugo.toml` e copia o tema; teste de build real |
| `internal/hugo` | encontra ou baixa o Hugo |
| `internal/deploy/github` | GitHub Pages via API REST (sem git) |
| `internal/deploy/cfworkers` | Cloudflare Workers Static Assets via API (sem wrangler) |
| `theme/` | tema Hugo (layouts, hooks do Obsidian, SEO, OG) |

## Testes

```sh
go test ./...
```

O teste de `internal/scaffold` gera e compila um site real com o vault de `internal/scaffold/testdata/vault` (precisa do Hugo instalado).
