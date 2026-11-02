---
title: Criar o primeiro site
description: Um passo de cada vez, no Studio ou no terminal. Dá para mudar tudo depois.
weight: 2
---
## Pelo Blup Studio

Na tela inicial, clique em **Criar site novo**. O Studio mostra uma pergunta por vez, com **Continuar** e **Voltar** embaixo.

1. **Que tipo de site você quer criar?**
   - **Blog pessoal**: blog, diário, newsletter ou portfólio de textos.
   - **Site de produto**: para apresentar um app, SaaS, API ou projeto, com documentação e novidades.
2. **Qual tema combina com o seu site?**
   - Blog: **Mochi** (post em destaque e grade de posts) ou **Nori** (uma coluna só, com o texto em primeiro plano).
   - Produto: **Bento**, este aqui.
3. **Quais partes o site vai ter?** (só no Bento): Documentação, Roadmap, Changelog, Central de ajuda e Blog. Cada parte vem com notas de exemplo que ensinam a usar. Escolha pelo menos uma.
4. **Como o site se chama?** O nome, uma descrição opcional, o seu nome (assina os posts) e o **idioma do site**.
5. **Onde ficam as notas?**
   - **Numa pasta nova**: o jeito mais simples. Dá para abrir no Obsidian depois.
   - **No meu cofre do Obsidian**: o site usa uma pasta do cofre que você já tem, ou o cofre inteiro.
   - **Numa pasta que eu escolher**: notas do VS Code, do Foam ou de qualquer editor.
6. **Tudo pronto?** Confira o resumo e clique em **Criar o site**.

Pronto: clique em **Abrir o site**. O editor abre com a prévia ao lado. Veja [A tela do Studio](../Blup%20Studio/A%20tela%20do%20Studio.md).

> [!tip] Ajustar o Obsidian
> Se as notas estão num cofre do Obsidian, o Studio oferece **Ajustar o Obsidian para os links funcionarem no site**. Assim os links novos entre notas são salvos no formato que a web entende. O Blup guarda uma cópia das configurações antes.

> [!note] Onde ficam os arquivos do site
> Em **Onde guardar os arquivos do site**, você escolhe a pasta das configurações e das páginas geradas. Ela fica fora das notas e você não precisa abrir.

### Já tem um site do Blup?

Na tela inicial, **Já tenho um site do Blup** abre uma pasta que já é um site, mostra os **Abertos recentemente** e, com a conta do GitHub conectada, oferece **Trazer** os sites guardados lá. Com a conta do Blup Cloud, o Studio também lista os seus sites guardados na nuvem.

## Pela linha de comando

Rode `blup novo` (ou só `blup`, na primeira vez). Cada pergunta aparece sozinha e já vem com uma resposta sugerida. Na dúvida, aperte Enter.

1. **Cofre do Obsidian**: o Blup lista os cofres que encontrou.
2. **Pasta**: o cofre inteiro ou só uma pasta, como "Site".
3. **Tema**: **Mochi** (blog) ou **Bento** (site de produto).
4. **Seções** (só no Bento): marque as que quiser.
5. **Nome do site**, **uma frase sobre ele** e **seu nome**.
6. **Idioma do site** e **onde salvar** os arquivos do site no computador.
7. **Resumo** para confirmar.
8. **Ajuste no Obsidian**: links Markdown com caminho relativo.
9. **Guardar as notas no GitHub**: opcional. Veja [Sincronizar](../Linha%20de%20comando/Sincronizar.md).
10. **Abrir a prévia**.

A linha de comando também cria a pasta **Templates** com modelos de nota para o Obsidian.

## O que é criado

- **Nas notas**: este site de exemplo (se a pasta estiver vazia), a sua nota de autor e as pastas das seções escolhidas.
- **Numa pasta à parte**: o **blup.toml** e os arquivos que o gerador do site (Hugo) usa.

> [!tip] Já tinha notas na pasta?
> Nada é apagado nem sobrescrito. O Blup só cria a nota **_index** se faltar e as pastas das seções escolhidas.
