---
title: Comandos
description: Um menu para tudo, e um comando para cada ação, em português ou inglês.
weight: 1
---
A linha de comando é o caminho para quem escreve no **Obsidian** e quer gerar e publicar o site sem o Blup Studio. Instale seguindo a [Instalação](../Primeiros%20passos/Instalação.md).

Rodar só `blup` abre o **menu**: prévia, publicar, personalizar, sincronizar, converter links, trocar de site, criar outro e idioma. É o jeito mais fácil.

![O Blup no terminal](../../Anexos/blup-terminal.jpg)

## Cada ação tem um comando

| Português | Inglês | O que faz |
| --- | --- | --- |
| `blup novo` | `blup new` | Cria um site novo |
| `blup ver` | `blup preview` | Abre a prévia no navegador |
| `blup publicar` | `blup publish` | Coloca o site no ar |
| `blup personalizar` | `blup customize` | Menu, seções, redes, logo, favicon, autor, senha geral e comentários |
| `blup sincronizar` | `blup sync` | Guarda as notas num repositório privado no GitHub |
| `blup converter` | `blup convert` | Troca `[[wikilinks]]` por links padrão |
| `blup gerar` | `blup build` | Gera o site em `public/`, sem perguntas |
| `blup idioma` | `blup language` | Troca o idioma do Blup |
| `blup atualizar` | `blup update` | Atualiza o Blup |
| `blup version` | `blup version` | Mostra a versão |
| `blup --help` | `blup help` | Mostra a ajuda |

## Mais de um site

O Blup lembra de todos os sites que você criou. No menu, **Trocar de site** escolhe outro. Nos comandos, passe a pasta do site: `blup publicar ~/Sites/meu-site`.

## Sem perguntas

`blup gerar` não pergunta nada: bom para scripts e automações. O site sai na pasta `public` da pasta do site.

## Site dentro de uma subpasta

Quer a documentação em `meusite.com.br/bento/`, junto de uma landing page na raiz? Gere o site para esse endereço e copie para a pasta da landing:

```sh
blup gerar --url https://meusite.com.br/bento/ --saida ../minha-landing/bento
```

- `--url` é o endereço final do site, e pode ser uma subpasta. Links, imagens e a busca passam a funcionar dentro dela.
- `--saida` copia o site pronto para outra pasta, trocando o que o Blup tinha posto lá antes. Por segurança, o Blup nunca apaga uma pasta que não foi ele que preencheu.
- Depois, publique a pasta da landing, por exemplo num serviço ligado a um repositório do GitHub: cada `git push` atualiza o site.

> [!tip] Em inglês também
> `blup build --url … --out …` faz o mesmo.

## O que fica só no Studio

Algumas coisas não têm comando: os cofres de anotações, a tradução, a narração, o podcast, o quadro de Feedback, os submenus, a publicação e a sincronização pelo Blup Cloud. Um site que já usa esses recursos continua gerando e publicando normalmente pelo terminal. Veja [O Blup Studio](../Primeiros%20passos/O%20Blup%20Studio.md).
