---
title: Comandos
description: Um menu para tudo, e um comando para cada ação, em português ou inglês.
weight: 1
---
Rodar só `blup` abre o **menu**: prévia, publicar, personalizar, sincronizar, trocar de site, criar outro e idioma. É o jeito mais fácil.

![O Blup no terminal](../../Anexos/blup-terminal.jpg)

## Cada ação tem um comando

| Português | Inglês | O que faz |
| --- | --- | --- |
| `blup novo` | `blup new` | Cria um site novo |
| `blup ver` | `blup preview` | Abre a prévia no navegador |
| `blup publicar` | `blup publish` | Coloca o site no ar |
| `blup personalizar` | `blup customize` | Menu, seções, redes, logo, favicon, autor e senha geral |
| `blup sincronizar` | `blup sync` | Guarda as notas num repositório privado no GitHub |
| `blup converter` | `blup convert` | Troca `[[wikilinks]]` por links padrão |
| `blup gerar` | `blup build` | Gera o site em `public/`, sem perguntas |
| `blup idioma` | `blup language` | Troca o idioma do Blup |
| `blup atualizar` | `blup update` | Atualiza o Blup |
| `blup version` | `blup version` | Mostra a versão |

## Mais de um site

O Blup lembra de todos os sites que você criou. No menu, **Trocar de site** escolhe outro. Nos comandos, passe a pasta do site: `blup publicar ~/Sites/meu-site`.

## Sem perguntas

`blup gerar` não pergunta nada: bom para scripts e automações. O site sai na pasta `public` da pasta do site.
