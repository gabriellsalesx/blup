---
title: Autores
description: Uma nota por pessoa, com foto, cargo, redes e bio.
weight: 3
---
Os perfis ficam numa pasta cuja **_index** tem `blup: authors`, aqui chamada **Autores**. Essa pasta não vira página: as notas dela são só os dados dos perfis.

## O perfil

Uma nota com o nome da pessoa, como `Ana Lima.md`:

```yaml
---
photo: "[[ana.jpg]]"
role: Designer de produto
social:
  - https://github.com/ana
  - https://x.com/ana
---
Ana desenha o produto e escreve sobre como ele é feito.
```

- **photo**: a foto. Sem ela, aparecem as iniciais (ou o mascote, para o autor do site).
- **role**: o cargo.
- **social**: os perfis, com ícones automáticos.
- O **texto** da nota é a bio.

O assistente de site novo já cria a sua nota. `blup personalizar` > **Nome, descrição e autor** edita nome, cargo, bio e foto.

## Nos posts

```yaml
authors: [Ana Lima]
authors: [Ana Lima, Bruno Costa]
```

Sem **authors**, o post é do autor do site. Com mais de um, as fotos aparecem lado a lado e os nomes juntos ("Ana Lima e Bruno Costa").

Cada autor ganha uma página com foto, cargo, bio, redes e os posts dele.
