---
title: Prévia e publicação
date: {{date-2}}
description: Veja o site enquanto escreve e publique pelo Blup Cloud, pelo Cloudflare Pages ou pelo GitHub Pages.
tags: [Começando, Studio]
image: "[[blup-publicar.jpg]]"
---
## A prévia

No Studio, clique em **Prévia**, no topo. O site aparece ao lado do editor e **atualiza sozinho** enquanto você escreve. Trocou de nota, a prévia vai para a página dela; clicou num link da prévia, o editor abre a nota daquela página.

A prévia mostra também os rascunhos, para você conferir antes de publicar.

> [!info] Pela linha de comando
> Rode `blup ver`. O site abre no navegador e atualiza a cada nota salva. Para parar, aperte Ctrl+C no terminal.

## Onde publicar

Clique em **Publicar**, no topo. Na primeira vez, escolha onde o site vai ficar:

- **Blup Cloud**: endereço como `seusite.getblup.site`, sem configurar nada. Vem nos planos pagos, a partir do Pro.
- **Cloudflare Pages**: grátis, com endereço como `seusite.pages.dev`. Precisa de uma conta no Cloudflare.
- **GitHub Pages**: grátis, com endereço como `seunome.github.io/seusite`. Precisa de uma conta no GitHub.

### Pelo Blup Cloud

Entre com a sua conta do Blup e digite o endereço que quer. Enquanto você digita, o Studio confere se o nome está livre e, se não estiver, sugere outros parecidos. O endereço fica fixo depois da primeira publicação.

Cada plano tem um número de sites: o Pro tem 1, o Max tem 3 e o Ultra tem 10. Domínio próprio nos sites do Blup Cloud está chegando, nos planos Max e Ultra.

### Pelo Cloudflare ou pelo GitHub

Conecte a conta uma vez:

1. Clique em **Conectar a conta** e depois em **Abrir no navegador**. O formulário da chave abre já com as permissões certas.
2. Crie a chave e copie.
3. Cole a chave no Studio e clique em **Conectar**. Ela fica guardada só no seu computador.

Pronto: clique em **Publicar agora**. Quando terminar, o Studio mostra o endereço do site.

> [!info] Pela linha de comando
> Rode `blup publicar`. Na primeira vez ele pergunta onde publicar (Cloudflare Pages ou GitHub Pages), abre a página da chave e pede para colar no terminal.

Das próximas vezes, só o que mudou é enviado.

> [!tip] Publicar é sempre igual
> Escreveu, conferiu na prévia, clicou em Publicar. Não existe servidor para manter nem banco de dados.

## Domínio próprio

Quer um endereço seu, como `meusite.com.br`? Com o site no ar pelo Cloudflare ou pelo GitHub, vá à tela **Publicar**, em **Domínio próprio**, e digite o endereço. O Blup liga o domínio onde o site está publicado e, se faltar algo, mostra o registro de DNS para criar onde você comprou o domínio. Pela linha de comando, o caminho é `blup personalizar` e **Domínio próprio**.

## O que vai para o ar

O site é público: qualquer pessoa com o endereço pode ler. Por isso só vai o que você publicou:

- Todas as notas da pasta do site, menos as marcadas como **Rascunho** (`draft: true`).
- Os comentários entre `%%` ficam de fora.
- Só os anexos que as notas usam, otimizados.
- Posts com senha saem cifrados do seu computador.
- A busca, o feed RSS, o mapa do site e o cartão de compartilhamento de cada página.
