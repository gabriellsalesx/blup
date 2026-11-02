---
title: Domínio próprio
description: O site no seu endereço, como meusite.com.br, com HTTPS.
weight: 3
---
Em vez do endereço grátis (`site.pages.dev` ou `seunome.github.io`), o site pode responder num endereço seu, como `meusite.com.br` ou `blog.meusite.com.br`. Você precisa de um domínio comprado (Registro.br, GoDaddy, Cloudflare...).

> [!info] Onde já funciona
> Pelo **Cloudflare Pages** e pelo **GitHub Pages**, o domínio próprio já funciona. Nos sites publicados pelo **Blup Cloud**, ele vem nos planos Max e Ultra.

## Pelo Blup Studio

O bloco **Domínio próprio** aparece na tela **Publicar** depois que o site está no ar pelo Cloudflare ou pelo GitHub.

1. Escreva só o endereço, como `meusite.com.br`, e clique em **Ligar domínio**.
2. O Studio liga o domínio onde o site está publicado.
3. Se faltar um passo, ele mostra o **registro de DNS** exato (Tipo, Nome e Valor, com **Copiar**). Crie esse registro no site onde você comprou o domínio.
4. Clique em **Verificar**. Pode levar de minutos a algumas horas.
5. Quando aparecer **Funcionando**, publique de novo para os links do site usarem o novo endereço.

**Trocar** muda de domínio. **Voltar ao endereço grátis** desliga o domínio deste site (ele continua seu).

## Pela linha de comando

Rode `blup personalizar` e escolha **Domínio próprio**. O Blup pergunta o domínio, liga onde o site está publicado, mostra o registro de DNS se faltar e liga o HTTPS. Para conferir, volte em **Domínio próprio** e escolha **Ver se já está funcionando**; cada `blup publicar` também confere.

> [!tip] Domínio raiz no Cloudflare Pages
> Um domínio raiz (`meusite.com.br`, sem `www` ou `blog` na frente) no Cloudflare Pages precisa estar na sua conta Cloudflare, o que é grátis: adicione o domínio em dash.cloudflare.com e, onde você o comprou, troque os servidores de nome pelos que o Cloudflare mostrar. Se preferir não mudar isso, use `www.meusite.com.br`.

> [!note] Podcast precisa de domínio
> O feed do podcast dos blogs só funciona nos apps com o endereço completo do site. Veja [Podcast dos blogs](../Recursos%20com%20IA/Podcast%20dos%20blogs.md).
