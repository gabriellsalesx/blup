---
title: Escrever no editor
description: Comandos com /, links com [[, menções com @, imagens coladas e corretor.
weight: 2
---
O editor do Studio escreve Markdown, mas mostra o texto já desenhado: negrito aparece em negrito, títulos aparecem grandes, callouts mostram o ícone. As marcas (como `**` ou `#`) voltam só enquanto o cursor está nelas, para você editar.

![O Blup escrevendo](../../Anexos/blup-escrever.jpg)

## A barra de formatação

Selecione um texto e use a barra: negrito, itálico, riscado, marca-texto, código, títulos, listas, tarefas, citação, callout, link, imagem, tabela, linha divisória e comentário. O botão de alinhamento centraliza, alinha à direita ou justifica.

## Comandos com /

No começo da linha (ou depois de um espaço), digite `/` e escolha com as setas e Enter (ou Tab):

- Título 1, Título 2 e Título 3
- Lista, Lista numerada e Lista de tarefas
- Citação e Destaque (callout)
- Tabela, Bloco de código e Linha divisória
- Link para outra nota (abre a busca de notas) e Link externo
- Imagem do computador
- Data de hoje

Num cofre de anotações, o `/` também tem **Novo desenho** e **Novo canvas**: o Studio cria o arquivo e põe o link na nota.

## Links com [[

Digite `[[` e comece a escrever o nome de uma nota. Escolha na lista e o Studio grava um link Markdown relativo, o formato que o site entende.

No editor, o link aparece só pelo texto, em azul e sublinhado. Um clique abre a nota. Para links da web, use `Ctrl` e clique: o clique simples continua editando.

O botão de link (ou `Ctrl+K`) insere um link externo e já seleciona o endereço para você colar.

## Menções com @

No começo da linha ou depois de um espaço, digite `@`:

- **Notas**: escolha uma nota e o Studio põe um link para ela, com o `@` no texto.
- **Datas**: hoje, amanhã, ontem e os próximos dias da semana pelo nome. Vira a data escrita, como 01/11/2026.

Num cofre, a nota mencionada mostra quem a citou no painel **Mencionada em**.

## Imagens

Cole (`Ctrl+V`) ou arraste uma imagem para o editor. O Studio salva o arquivo em **Anexos** e põe a imagem no texto. Ela aparece no lugar, dentro do editor.

Imagens da internet esperam um clique em "Mostrar imagem de…", porque carregar a imagem avisa o servidor dela que a nota foi aberta.

## Tabelas

Crie pelo `/` ou pela barra. Clique com o botão direito na tabela para inserir ou apagar linhas e colunas, ou para **Editar como texto (Markdown)**.

## Títulos que recolhem

Passe o mouse à esquerda de um título: a setinha esconde tudo até o próximo título do mesmo nível. Atalhos: `Ctrl+Shift+[` recolhe, `Ctrl+Shift+]` mostra, `Ctrl+Alt+[` recolhe todos e `Ctrl+Alt+]` mostra todos.

## Corretor ortográfico

Palavras com erro ganham um sublinhado. Clique com o botão direito para ver as sugestões ou para adicionar a palavra ao dicionário. O corretor roda no seu computador: o texto não vai para nenhum serviço.

Em **Configurações > Editor** você liga ou desliga o corretor, escolhe o idioma do dicionário (português, inglês ou espanhol) e vê as palavras adicionadas.

## Colar sem formatação

O botão direito tem **Colar sem formatação**, útil quando o texto vem de um site com estilos.

## Atalhos do editor

| Ação | Teclas |
| --- | --- |
| Salvar | `Ctrl+S` |
| Desfazer e refazer | `Ctrl+Z` e `Ctrl+Shift+Z` |
| Negrito e itálico | `Ctrl+B` e `Ctrl+I` |
| Link | `Ctrl+K` |
| Aceitar a sugestão | Tab |

> [!tip] Comentários que não vão para o site
> Texto entre `%%` e `%%` fica só nas notas. Use para lembretes e rascunhos. Veja [Fórmulas, código e comentários](../Escrevendo/Fórmulas,%20código%20e%20comentários.md).
