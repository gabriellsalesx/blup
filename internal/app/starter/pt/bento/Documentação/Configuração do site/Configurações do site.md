---
title: Configurações do site
description: Nome, descrição, autor, logo e favicon, na tela Site ou na nota _index.
weight: 1
---
As configurações do site ficam guardadas como **propriedades da nota _index** da pasta do site. Há três jeitos de mudar, e todos gravam no mesmo lugar.

## Pelo Blup Studio: a tela Site

Clique em **Site** na faixa lateral. A tela mostra como o site se apresenta, em blocos:

1. **Nome e descrição**
   - **Nome do site**: aparece no topo, na aba do navegador e no Google.
   - **Descrição**: uma frase. Aparece no topo, no Google e ao compartilhar.
   - **Autor**: quem assina os posts que não dizem o autor.
2. **Logo e favicon**
   - **Logo**: no topo do site. Sem logo, aparece o nome.
   - **Mostrar o nome ao lado do logo**: desligado, o topo do site mostra só o logo.
   - **Favicon**: o ícone da aba e do app. Quadrado, de preferência 512 pixels ou SVG.
3. **Menu do topo** e **Botão do topo**. Veja [Menu, submenus e botão](Menu,%20submenus%20e%20botão.md).
4. **Redes sociais**. Veja [Redes sociais](Redes%20sociais.md).

Cada mudança aparece na prévia ao lado na hora, sem recarregar.

![O Blup configurando](../../Anexos/blup-configurar.jpg)

## Pelo Obsidian: a nota _index

| Propriedade | O que faz |
| --- | --- |
| `site_name` | Nome do site: aba do navegador, rodapé e Google. |
| `site_description` | Uma frase sobre o site, para o Google e as redes. |
| `logo` | Imagem do logo no topo, como `"[[logo.svg]]"`. |
| `header_name` | `false` mostra só o logo no topo, sem o nome. |
| `favicon` | Ícone da aba do navegador, como `"[[favicon.png]]"`. |
| `menu` | Itens do menu do topo. |
| `button_text` e `button_link` | O botão no canto superior direito. |
| `social` | Links dos seus perfis, com ícone no rodapé. |

```yaml
---
title: Acme
description: O jeito simples de fazer X.
site_name: Acme
site_description: O jeito simples de fazer X.
logo: "[[acme-logo.svg]]"
favicon: "[[acme-icone.png]]"
social:
  - https://github.com/acme
  - mailto:oi@acme.com
---
```

## Pela linha de comando

`blup personalizar` faz as mesmas perguntas (nome, descrição, autor, logo, favicon, menu, redes) e grava nas mesmas propriedades.

## Logo e favicon

- Em **SVG**, ficam nítidos em qualquer tela.
- Em **PNG**, o Blup gera os tamanhos que navegadores e celulares pedem.
- Sem logo, o topo mostra o nome. Sem favicon, o site usa o mascote.

## As outras telas do site no Studio

| Tela | O que configura |
| --- | --- |
| Seções | As partes do Bento. Veja [Ligar e desligar seções](../Seções%20do%20Bento/Ligar%20e%20desligar%20seções.md) |
| Autores | Os perfis. Veja [Autores](Autores.md) |
| Idiomas | O site em outros idiomas. Veja [Sites em vários idiomas](Sites%20em%20vários%20idiomas.md) |
| Comentários | Votos, reações e comentários. Veja [Comentários e reações](../Seções%20do%20Bento/Comentários%20e%20reações.md) |
| Cookies e estatísticas | Ferramentas e banner. Veja [Cookies e estatísticas](Cookies%20e%20estatísticas.md) |
| Podcast | Só nos blogs. Veja [Podcast dos blogs](../Recursos%20com%20IA/Podcast%20dos%20blogs.md) |
| Publicar | Onde e como o site vai ao ar. Veja [Publicar pelo Studio](../Publicação/Publicar%20pelo%20Studio.md) |

> [!note] E o blup.toml?
> Na pasta do site no computador, o **blup.toml** guarda o que não é conteúdo: onde publicar, o domínio próprio, os idiomas, a senha do site e o repositório da sincronização. O Studio e o `blup personalizar` mudam tudo isso sem você abrir o arquivo.
