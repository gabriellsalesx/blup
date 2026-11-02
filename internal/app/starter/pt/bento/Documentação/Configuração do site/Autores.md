---
title: Autores
description: Um perfil por pessoa, com foto, cargo, redes e bio.
weight: 4
---
Cada autor ganha uma página com foto, cargo, bio, redes e os posts dele. Nos posts, a foto e o nome aparecem junto do título.

![Os autores](../../Anexos/blup-autores.jpg)

## Pelo Blup Studio

Clique em **Autores** na faixa lateral. A tela lista as pessoas, com o número de posts de cada uma, e marca o **Autor do site**.

- **Novo autor** abre o formulário: **Nome** (como aparece nos posts), foto, **Cargo**, **Bio** e **Redes sociais**.
- Para a foto, escolha um dos avatares prontos ou **Enviar foto própria**.
- **Editar** muda tudo. Trocar o nome atualiza os posts daquela pessoa.
- **Apagar autor**: os posts continuam com o nome, sem foto. Dá para restaurar pela Lixeira.

Nos posts, escolha os autores no campo **Autores** das [Propriedades da nota](../Blup%20Studio/Propriedades%20da%20nota.md).

## Pelo Obsidian

Os perfis ficam numa pasta cuja **_index** tem `blup: authors`, aqui chamada **Autores**. Essa pasta não vira página: as notas dela são só os dados.

Uma nota com o nome da pessoa, como `Ana Lima.md`:

```yaml
---
photo: "[[ana.jpg]]"
role: Designer de produto
social:
  - https://github.com/ana
---
Ana desenha o produto e escreve sobre como ele é feito.
```

- **photo**: a foto. Sem ela, aparecem as iniciais (ou o mascote, para o autor do site).
- **role**: o cargo.
- **social**: os perfis, com ícones automáticos.
- O **texto** da nota é a bio.

Nos posts:

```yaml
authors: [Ana Lima, Bruno Costa]
```

Sem **authors**, o post é do autor do site. Com mais de um, as fotos aparecem lado a lado e os nomes juntos ("Ana Lima e Bruno Costa").

## Pela linha de comando

`blup personalizar` > **Nome, descrição e autor** edita nome, cargo, bio e foto do autor do site.

> [!tip] Perfis em outros idiomas
> Num site em vários idiomas, o cargo e a bio também são traduzidos, numa pasta do idioma dentro de Autores. Veja [Sites em vários idiomas](Sites%20em%20vários%20idiomas.md).
