---
title: Sincronizar as notas
date: {{date-8}}
description: Suas notas num repositório privado do GitHub, para usar em outro computador. Sem git.
tags: [Personalização, Studio]
image: "[[blup-sincronizar.jpg]]"
---
O Blup guarda uma cópia das notas do site num **repositório privado** no GitHub. Em outro computador, abra o site no Studio e sincronize: as notas chegam. Edite onde quiser e sincronize de novo.

Você não precisa saber git nem instalar nada além do Studio.

## Na primeira vez

1. Clique em **Sincronizar**, no topo. Você cai na tela **Publicar**, em **Sincronizar as notas**.
2. Conecte a sua conta do GitHub, com a mesma chave usada para publicar.
3. O repositório é criado **privado**: só você vê.

## Toda vez

Clique em **Sincronizar**, no topo, ou em **Conferir**. O Blup compara as notas daqui com as do GitHub e **mostra o que vai fazer** antes de fazer:

- o que mudou aqui é enviado;
- o que mudou em outro computador é recebido;
- o que foi apagado de um lado é apagado do outro.

Se estiver tudo certo, clique em **Sincronizar agora**.

## Quando os dois lados mudaram

Se a mesma nota foi editada aqui e em outro computador, nada se perde: a sua versão fica, e a outra vira uma cópia com "conflict" no nome. Compare as duas e apague a que sobrar.

> [!info] Nada some de repente
> Notas apagadas em outro computador vão para uma lixeira fora da pasta das notas. Se a pasta do site aparecer vazia de repente (por exemplo, se ela mudou de lugar), o Blup para e não apaga nada no GitHub.
