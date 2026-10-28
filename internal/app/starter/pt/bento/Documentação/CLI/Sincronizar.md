---
title: Sincronizar
description: As notas do site num repositório privado do GitHub, para usar em outro computador. Sem git.
weight: 4
---
`blup sincronizar` guarda uma cópia das notas da pasta do site num **repositório privado** do GitHub. Em outro computador, instale o Blup e sincronize: as notas chegam. Edite onde quiser e sincronize de novo.

![Dois computadores sincronizando](../../Anexos/blup-sincronizar.jpg)

## Na primeira vez

1. Rode `blup sincronizar`, escolha no menu ou responda "sim" no fim do assistente de site novo.
2. Use a mesma chave do GitHub da publicação.
3. Escolha o nome do repositório. Ele é criado **privado**. Se o repositório já existir e for público, o Blup avisa antes.

## Toda vez

O Blup compara e **mostra antes** quantas notas vai enviar, receber, apagar no GitHub e mover para a lixeira aqui. Só continua se você confirmar.

| Situação | O que acontece |
| --- | --- |
| Mudou só aqui | É enviada |
| Mudou só em outro computador | É recebida |
| Apagada de um lado | É apagada do outro |
| Apagada de um lado e editada do outro | A versão editada fica |
| Editada dos dois lados | As duas ficam: a sua e uma cópia com "conflict" no nome |

## Segurança

- Notas apagadas em outro computador vão para uma **lixeira fora do cofre**; o Blup mostra onde.
- Se a pasta do site aparecer vazia mas já tinha notas, o Blup **para** e não apaga nada no GitHub.
- Pastas que começam com ponto (como `.obsidian`) não são sincronizadas.
- Arquivos acima de 100 MB ficam de fora, com aviso.
