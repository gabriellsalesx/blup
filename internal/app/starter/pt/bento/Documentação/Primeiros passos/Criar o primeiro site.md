---
title: Criar o primeiro site
description: O assistente pergunta uma coisa de cada vez. Na dúvida, aperte Enter.
weight: 2
---
Rode `blup novo` (ou só `blup`, na primeira vez). Cada pergunta aparece sozinha e já vem com uma resposta sugerida.

## As perguntas

1. **Cofre do Obsidian**: o Blup lista os cofres que encontrou.
2. **Pasta**: o cofre inteiro ou só uma pasta, como "Site".
3. **Tema**:
   - **Mochi**: blog pessoal, com posts, tags e autores.
   - **Bento**: site de produto, este aqui.
4. **Seções** (só no Bento): Documentação, Roadmap, Changelog, Central de ajuda e Blog. Marque as que quiser; dá para adicionar depois.
5. **Nome do site**, **uma frase sobre ele** e **seu nome**.
6. **Idioma do site** e **onde salvar** os arquivos do site no computador.
7. **Resumo** para confirmar.
8. **Ajuste no Obsidian**: links Markdown com caminho relativo, o formato que sites entendem. O Blup guarda uma cópia das configurações antigas.
9. **Guardar as notas no GitHub**: opcional, para usar em outro computador. Veja [Sincronizar](../CLI/Sincronizar.md).
10. **Abrir a prévia**.

## O que o assistente cria

- No cofre: este tutorial (se a pasta estiver vazia), a sua nota de autor e a pasta **Templates** com modelos de nota.
- Numa pasta à parte: o **blup.toml** e os arquivos que o gerador do site (Hugo) usa. Você não precisa abrir essa pasta.

> [!tip] Já tinha notas na pasta?
> Nada é apagado nem sobrescrito. O Blup só cria a nota **_index** se faltar e as pastas das seções escolhidas.
