---
title: Ligar e desligar seções
description: Todas as seções são opcionais. Adicione pelo Blup, tire apagando a pasta.
weight: 6
---
## Adicionar

Rode `blup personalizar` e escolha **Seções do site**. O Blup mostra as que faltam e cria a pasta de cada uma, com notas de exemplo.

Também dá para criar à mão: uma pasta na raiz do site com uma nota **_index** que tenha a propriedade **blup**:

| Seção | Propriedade |
| --- | --- |
| Documentação | `blup: docs` |
| Roadmap | `blup: roadmap` |
| Changelog | `blup: changelog` |
| Central de ajuda | `blup: help` |
| Blog | `blup: blog` |
| Perfis dos autores | `blup: authors` |

## Tirar

Apague a pasta no Obsidian. Pronto: a seção sai do site e do menu.

## Renomear e reordenar

- O nome da pasta é livre: "Docs", "Guia", "Manual"... o que vale é a propriedade **blup**.
- O **title** da _index é o nome no menu e no topo da seção.
- O **weight** da _index ordena as seções no menu automático e nos cartões da página inicial.
