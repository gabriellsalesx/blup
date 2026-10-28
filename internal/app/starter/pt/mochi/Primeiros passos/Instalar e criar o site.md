---
title: Instalar e criar o site
date: {{date-1}}
description: Um comando para instalar e um assistente que pergunta uma coisa de cada vez.
tags: [Começando, CLI]
image: "[[blup-terminal.jpg]]"
---
O Blup é um programa de terminal, mas você não precisa decorar nada: ele mostra um menu e faz uma pergunta de cada vez.

## Instalar

No macOS ou Linux, abra o Terminal e cole:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

No Windows, abra o PowerShell e cole:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Depois, rode `blup`. Na primeira vez ele pergunta o idioma e abre o assistente de site novo.

## O assistente

Cada pergunta já vem com uma resposta sugerida; se estiver em dúvida, aperte Enter.

1. **Qual cofre do Obsidian** vira site.
2. **Qual pasta** dele: o cofre inteiro ou só uma pasta, como "Blog".
3. **Tema**: Mochi (blog pessoal, este aqui) ou Bento (site de produto, com documentação, roadmap, changelog, central de ajuda e blog).
4. **Nome do site**, **uma frase sobre ele** e **seu nome**.
5. **Idioma do site** e **onde salvar** os arquivos do site no computador.
6. **Ajustes no Obsidian**: o Blup pede para usar links Markdown com caminho relativo, o formato que sites entendem.
7. **Guardar as notas no GitHub**, se quiser usar em outro computador.
8. **Abrir a prévia**.

> [!info] Onde fica cada coisa
> As notas ficam no seu cofre, como sempre. Numa pasta à parte, o Blup guarda o arquivo **blup.toml** (onde publicar, senha geral) e os arquivos que o Hugo usa para gerar o site. Você não precisa abrir essa pasta.

## Os comandos

Rodar só `blup` abre o menu com tudo. Se preferir, cada ação tem um comando, em português ou inglês:

| Comando | O que faz |
| --- | --- |
| `blup novo` | Cria um site novo |
| `blup ver` | Abre a prévia no navegador |
| `blup publicar` | Coloca o site no ar |
| `blup personalizar` | Menu, redes, logo, favicon e autor |
| `blup sincronizar` | Guarda as notas num repositório privado no GitHub |
| `blup converter` | Troca `[[wikilinks]]` por links padrão |
| `blup gerar` | Gera o site na pasta public, sem perguntas |
| `blup idioma` | Troca o idioma do Blup |
| `blup atualizar` | Atualiza o Blup |

Os mesmos comandos funcionam em inglês: `new`, `preview`, `publish`, `customize`, `sync`, `convert`, `build`, `language` e `update`.

## Atualizações

O Blup confere uma vez por dia se há versão nova e **sempre pergunta** antes de atualizar. Se algo der errado, `blup atualizar --rollback` volta para a versão anterior.
