---
title: Como este site foi montado
date: {{date-3}}
description: Por dentro do modelo que vem com o Blup, pasta por pasta.
tags: [Bastidores]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
Este post foi "escrito" pelo **Blup**, o mascote. A foto e o nome vêm da nota dele na pasta **Autores**, e a propriedade `authors: [Blup]` liga os dois.

## Uma pasta para cada seção

Na raiz da pasta do site, cada pasta com uma nota **_index** e a propriedade **blup** vira uma seção:

| Pasta | Propriedade | Vira |
| --- | --- | --- |
| Documentação | `blup: docs` | Documentação com menu lateral |
| Roadmap | `blup: roadmap` | Quadro com três colunas |
| Changelog | `blup: changelog` | Lista de versões |
| Ajuda | `blup: help` | Central de ajuda |
| Blog | `blup: blog` | Este blog |
| Autores | `blup: authors` | Perfis (sem página) |

## A página inicial

A nota **_index** da pasta do site é a página inicial: o título, a descrição e o texto aparecem no topo, e as seções aparecem em cartões, na ordem do **weight** de cada uma.

## As imagens

As capas estão na pasta **Anexos** e entram nos posts pela propriedade `image`. O Blup gera versões menores e em WebP de cada uma.

## E o resto?

Busca, tema claro e escuro, cartões para as redes sociais, feed RSS e mapa do site vêm prontos, sem configurar nada.
