---
title: Domínio próprio
description: O site no seu endereço, como meusite.com.br, com HTTPS. Pelo Blup Cloud, nos planos Max e Ultra.
weight: 3
---
Em vez de `seusite.getblup.site`, o site pode responder num endereço seu, como `meusite.com.br` ou `blog.meusite.com.br`. Você precisa de um domínio comprado (Registro.br, GoDaddy, Cloudflare...).

> [!info] Só pelo Blup Cloud
> O domínio próprio é ligado pelo **Blup Cloud**, nos planos **Max** e **Ultra**. Os sites publicados pelo **Cloudflare Workers** ou pelo **GitHub Pages** ficam no endereço grátis do serviço (`nome.suaconta.workers.dev` ou `seunome.github.io`). Para usar um domínio seu, publique o site pelo Blup Cloud: veja [Sites pelo Blup Cloud](../Blup%20Cloud/Sites%20pelo%20Blup%20Cloud.md).

## Pelo Blup Studio

O bloco **Domínio próprio** aparece na tela **Publicar** depois que o site está no ar pelo Blup Cloud.

1. Escreva só o domínio, sem `https://`, como `meusite.com.br`, e clique em **Vincular**.
2. O Studio mostra os **registros de DNS** que faltam (Tipo, Nome e Valor, com **Copiar**): um TXT que prova que o domínio é seu e o CNAME que aponta para o Blup Cloud. Crie esses registros no site onde você comprou o domínio.
3. Clique em **Verificar**. Enquanto a tela estiver aberta, o Studio confere de novo a cada minuto. O DNS pode levar de minutos a algumas horas.
4. Quando aparecer **Ativo** e **HTTPS ativo**, o Studio publica o site de novo sozinho, para os links usarem o novo endereço.

O endereço `seusite.getblup.site` continua funcionando. **Remover** desliga o domínio deste site, que volta a ficar só em `seusite.getblup.site`.

> [!tip] Domínio raiz
> Um domínio raiz (`meusite.com.br`, sem `www` ou `blog` na frente) precisa de um DNS que aceite CNAME na raiz (CNAME flattening ou ALIAS). Se o seu não aceitar, use `www.meusite.com.br`.

> [!warning] Mudou de plano?
> Saindo do Max ou do Ultra, o domínio continua funcionando por 14 dias. Voltando a um desses planos antes disso, nada muda. Depois do prazo, o site fica só em `seusite.getblup.site`, e para usar o domínio de novo é preciso vinculá-lo outra vez.

> [!note] Podcast precisa do endereço completo
> O feed do podcast dos blogs só funciona nos apps com o endereço completo do site. Com domínio próprio, o feed usa o seu domínio. Veja [Podcast dos blogs](../Recursos%20com%20IA/Podcast%20dos%20blogs.md).
