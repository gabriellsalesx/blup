---
title: Configurar o site
date: {{date-6}}
description: Nome, descrição, logo, menu com submenus, botão e redes sociais, na tela Site do Studio ou na nota _index.
tags: [Personalização, Studio]
image: "[[blup-configurar.jpg]]"
---
As configurações do site ficam na tela **Site**, no menu ao lado do Studio. Mudou lá, mudou no site: a prévia mostra na hora.

Tudo isso fica guardado nas propriedades da nota **_index** da pasta do site. Por isso também dá para editar essa nota direto, em qualquer editor de texto. No fim de cada parte há o jeito de escrever na nota.

## Nome e descrição

O **Nome do site** aparece no topo, na aba do navegador, no rodapé (© ano e nome) e no Google. A **Descrição** é uma frase sobre o site: aparece no Google e quando alguém compartilha o site.

Em **Autor**, escolha quem assina os posts que não dizem o autor.

```yaml
site_name: Meu Blog
site_description: Anotações sobre escrita, viagens e livros.
```

## Logo e favicon

Em **Logo e favicon**, clique em **Escolher**. A imagem vai para a pasta **Anexos** sozinha.

- A **Logo** aparece no topo do site. Sem logo, aparece o nome.
- **Mostrar o nome ao lado do logo**: desligado, o topo do site mostra só o logo. Bom quando o logo já traz o nome.
- O **Favicon** é o ícone da aba do navegador e do app. Em SVG, fica nítido em qualquer tamanho; em PNG, o Blup gera os tamanhos que os navegadores e o iPhone pedem. Quadrado, de preferência com 512 pixels.

```yaml
logo: "[[logo.svg]]"
favicon: "[[favicon.png]]"
header_name: false
```

## Menu do topo

As páginas do menu, na ordem:

- **Adicionar página** coloca uma pasta ou nota do site. Uma pasta, como **Escrevendo**, leva à seção inteira.
- **Adicionar link** coloca um link para outro site, como a sua loja.
- O nome de cada item é livre: escreva o texto que deve aparecer.
- Arraste pela alça para mudar a ordem.

**Submenus**: arraste um item para cima de outro. O item de cima passa a abrir uma lista com os de baixo, sem link próprio. **Novo submenu** cria uma entrada vazia para você arrastar itens para dentro. É um nível só.

Sem nenhum item, o menu é automático: as pastas da raiz do site.

Na nota, um item por linha. `Nome | destino` troca o texto do item, um endereço vira link para fora, e `+ ` na frente põe o item no submenu da linha de cima:

```yaml
menu:
  - Início
  - Escrevendo
  - Leituras |
  - + Livros | Livros
  - + Revistas | Revistas
  - Loja | https://minhaloja.com
```

## Botão do topo

Um botão de destaque à direita do menu, como "Assinar". Preencha o **Texto do botão** e **Para onde leva**. Deixe o texto vazio para não ter botão.

```yaml
button_text: Assinar
button_link: https://minhanewsletter.com
```

## Redes sociais

Em **Redes sociais**, clique na rede (Instagram, X, LinkedIn, YouTube, Discord, GitHub, Facebook, WhatsApp ou e-mail): o Studio já coloca o começo do endereço, e você completa com o seu perfil. Os ícones aparecem no rodapé.

```yaml
social:
  - https://instagram.com/seunome
  - https://github.com/seunome
  - mailto:voce@exemplo.com
```

> [!tip] Quer ver na prática?
> Abra a nota **_index** da pasta do site: o menu deste blog está lá. Pela linha de comando, `blup personalizar` faz as mesmas perguntas e grava nas mesmas propriedades.
