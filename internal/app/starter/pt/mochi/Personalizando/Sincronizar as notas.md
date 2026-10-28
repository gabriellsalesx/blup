---
title: Sincronizar as notas
date: {{date-8}}
description: Suas notas num repositório privado do GitHub, para usar em outro computador. Sem git.
tags: [Personalização, CLI]
image: "[[blup-sincronizar.jpg]]"
---
O `blup sincronizar` guarda uma cópia das notas do site num **repositório privado** no GitHub. Em outro computador, instale o Blup e sincronize: as notas chegam. Edite onde quiser e sincronize de novo.

Você não precisa saber git nem instalar nada além do Blup.

## Na primeira vez

1. Rode `blup sincronizar` (ou escolha no menu, ou responda "sim" no fim do assistente de site novo).
2. Entre com a mesma chave do GitHub usada para publicar.
3. Escolha o nome do repositório. Ele é criado **privado**: só você vê.

## Toda vez

O Blup compara as notas daqui com as do GitHub e **mostra o que vai fazer** antes de fazer:

- o que mudou aqui é enviado;
- o que mudou em outro computador é recebido;
- o que foi apagado de um lado é apagado do outro.

## Quando os dois lados mudaram

Se a mesma nota foi editada aqui e em outro computador, nada se perde: a sua versão fica, e a outra vira uma cópia com "conflict" no nome. Compare as duas e apague a que sobrar.

> [!info] Nada some de repente
> Notas apagadas em outro computador vão para uma lixeira fora do cofre, e o Blup mostra onde ela fica. Se a pasta do site aparecer vazia de repente (por exemplo, se o cofre mudou de lugar), o Blup para e não apaga nada no GitHub.
