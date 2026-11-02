---
title: Instalação
description: Baixe o Blup Studio em getblup.io. A linha de comando é opcional.
weight: 1
---
Há dois jeitos de usar o Blup, e eles convivem bem:

- **Blup Studio** (recomendado): um app para o computador, com editor, prévia ao vivo, telas de configuração e publicação com um botão.
- **Linha de comando** `blup` (opcional): para quem escreve no Obsidian e prefere o terminal.

## Requisitos

- Linux, macOS ou Windows.
- Só isso. O Blup baixa sozinho o que precisa para gerar o site.
- O Obsidian é opcional. Se você já tem um cofre, o Blup encontra e usa.

## Instalar o Blup Studio

1. Abra **getblup.io** no navegador.
2. Baixe a versão do seu sistema (Linux, macOS ou Windows).
3. Instale como qualquer outro app e abra o **Blup Studio**.

Na primeira vez, o Studio dá as boas-vindas, pergunta o idioma e oferece entrar na sua conta do Blup Cloud. A conta é **opcional**: tudo funciona neste computador sem ela. Depois é só seguir para [Criar o primeiro site](Criar%20o%20primeiro%20site.md).

> [!info] Atualizações do Studio
> Em **Configurações > Sobre**, o Studio procura versão nova uma vez por dia (ou quando você clica em **Procurar atualização**). Quando há, aparece um ponto azul no rodapé das configurações e o botão **Baixar**.

## Instalar a linha de comando (opcional)

Só precisa disto se for usar o `blup` no terminal. O Studio não depende dele.

### macOS e Linux

Abra o Terminal e cole:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

### Windows

Abra o PowerShell e cole:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

### Conferir

Rode `blup`. Na primeira vez ele pergunta o idioma (português, inglês ou espanhol) e abre o assistente de site novo.

> [!info] Atualizações da linha de comando
> O `blup` confere uma vez por dia se há versão nova e sempre pergunta antes de atualizar. Veja [Atualizar](../Linha%20de%20comando/Atualizar.md).
