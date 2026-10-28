---
title: Instalação
description: Um comando no terminal e o Blup está pronto.
weight: 1
---
## Requisitos

- O [Obsidian](https://obsidian.md) com um cofre (vault).
- macOS, Linux ou Windows.
- Só isso. O Blup baixa sozinho o que precisa para gerar o site.

## macOS e Linux

Abra o Terminal e cole:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

## Windows

Abra o PowerShell e cole:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

## Conferir

Rode `blup`. Na primeira vez ele pergunta o idioma (português ou inglês) e abre o assistente de site novo.

> [!info] Atualizações
> O Blup confere uma vez por dia se há versão nova e sempre pergunta antes de atualizar. Veja [Atualizar](../CLI/Atualizar.md).
