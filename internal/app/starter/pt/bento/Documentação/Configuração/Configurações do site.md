---
title: Configurações do site
description: Nome, descrição, logo, favicon e redes ficam na nota _index da pasta do site.
weight: 1
---
As configurações do site são **propriedades da nota _index** da pasta do site. Mudou no Obsidian, mudou no site. No terminal, `blup personalizar` faz as mesmas perguntas e grava nas mesmas propriedades.

| Propriedade | O que faz |
| --- | --- |
| `site_name` | Nome do site: aba do navegador, rodapé e Google. |
| `site_description` | Uma frase sobre o site, para o Google e as redes. |
| `logo` | Imagem do logo no topo, como `"[[logo.svg]]"`. |
| `favicon` | Ícone da aba do navegador, como `"[[favicon.png]]"`. |
| `menu` | Itens do menu do topo. Veja [Menu e botão](Menu%20e%20botão.md). |
| `button_text` e `button_link` | O botão no canto superior direito. |
| `social` | Links dos seus perfis, com ícone no rodapé. |

## Exemplo

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
  - https://www.linkedin.com/company/acme
  - mailto:oi@acme.com
---
```

## Redes sociais

O Blup reconhece a rede pelo endereço e põe o ícone certo: Instagram, X, LinkedIn, YouTube, GitHub, Facebook, WhatsApp (`wa.me`) e e-mail (`mailto:`). Outros endereços ganham um ícone de link.

## Logo e favicon

- Em **SVG**, ficam nítidos em qualquer tela.
- Em **PNG**, o Blup gera os tamanhos que navegadores e celulares pedem.
- Sem logo, o site mostra o mascote. Sem favicon, também.

> [!note] E o blup.toml?
> Na pasta do site no computador, o **blup.toml** guarda o que não é conteúdo: onde publicar, o domínio próprio, a senha geral e o repositório da sincronização. Tudo isso se muda pelo `blup personalizar`, sem abrir o arquivo. O nome e a descrição que você deu no assistente também ficam lá, como reserva, mas o que vale é a nota _index.
