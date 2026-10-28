---
title: Configurar o site pelo Obsidian
date: {{date-6}}
description: Nome, descrição, menu, botão, redes sociais, logo e favicon ficam na nota _index.
tags: [Personalização]
image: "[[blup-configurar.jpg]]"
---
As configurações do site são **propriedades da nota _index** da pasta do site. Mudou no Obsidian, mudou no site. Se preferir o terminal, `blup personalizar` faz as mesmas perguntas e grava nas mesmas propriedades.

## Nome e descrição

```yaml
site_name: Meu Blog
site_description: Anotações sobre escrita, viagens e livros.
```

O nome aparece na aba do navegador, no rodapé (© ano e nome) e no Google. A descrição aparece no Google e quando alguém compartilha o site.

## Menu do topo

Um item por linha:

```yaml
menu:
  - Início
  - Escrevendo
  - Leituras | Livros
  - Loja | https://minhaloja.com
```

- Um nome sozinho procura uma **pasta ou nota** com esse nome: `Escrevendo` leva à seção Escrevendo.
- `Nome que aparece | pasta ou nota` troca o texto do item: `Leituras | Livros` mostra "Leituras" e leva à pasta Livros.
- `Nome | https://...` vira link para outro site.
- `Início` é a página inicial.
- Sem a propriedade, o menu é automático: as pastas da raiz do site.

## Botão no topo

```yaml
button_text: Assinar
button_link: https://minhanewsletter.com
```

## Redes sociais

```yaml
social:
  - https://instagram.com/seunome
  - https://github.com/seunome
  - mailto:voce@exemplo.com
```

Os ícones aparecem no rodapé. O Blup reconhece a rede pelo endereço: Instagram, X, LinkedIn, YouTube, GitHub, Facebook, WhatsApp e e-mail.

## Logo e favicon

Coloque as imagens no cofre (por exemplo na pasta **Anexos**) e aponte para elas:

```yaml
logo: "[[logo.svg]]"
favicon: "[[favicon.png]]"
```

O favicon é o ícone da aba do navegador. Em SVG, fica nítido em qualquer tamanho; em PNG, o Blup gera os tamanhos que os navegadores e o iPhone pedem.

> [!tip] Quer ver na prática?
> Abra a nota **_index** desta pasta do site: o menu deste blog está lá.
