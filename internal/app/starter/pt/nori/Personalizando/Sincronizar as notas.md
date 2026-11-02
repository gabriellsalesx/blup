---
title: Sincronizar as notas
date: {{date-8}}
description: As mesmas notas em mais de um computador, pelo Blup Cloud com criptografia de ponta a ponta ou num repositório privado do GitHub.
tags: [Personalização, Studio]
image: "[[blup-sincronizar.jpg]]"
---
Sincronizar guarda uma cópia das notas na nuvem. Em outro computador, abra o Studio e sincronize: as notas chegam. Edite onde quiser e sincronize de novo.

No Studio, clique em **Sincronizar**, no topo. Na primeira vez, escolha onde guardar as notas:

- **Blup Cloud** (recomendado): criptografado de ponta a ponta. Nem o Blup consegue ler.
- **GitHub**: num repositório privado da sua conta do GitHub. Sem precisar saber git.

## Pelo Blup Cloud

1. Clique em **Entrar com Blup**. O Blup Cloud abre no navegador: confira se o código é o mesmo e aprove.
2. Crie a **senha das notas**. Ela embaralha as notas antes de saírem do computador.
3. Guarde a **chave de recuperação** que aparece. Ela abre as notas se você esquecer a senha, e não aparece de novo.

Em outro computador, entre com a mesma conta e digite a senha das notas (ou a chave de recuperação).

> [!warning] Ninguém mais tem a sua senha
> Nem o Blup. Se perder a senha e a chave de recuperação, as notas na nuvem não voltam. Guarde a chave num lugar seguro, fora do computador.

| Plano | Por mês | Por ano | Notas | Como sincroniza | Computadores | Versões anteriores |
| --- | --- | --- | --- | --- | --- | --- |
| Grátis | R$ 0 | R$ 0 | 50 MB | Pelo botão | 1 | Não |
| Pro | R$ 14,90 | R$ 149 | 1 GB | Em tempo real | 2 | 1 mês |
| Max | R$ 29,90 | R$ 299 | 10 GB | Em tempo real | 3 | 12 meses |
| Ultra | R$ 59,90 | R$ 599 | 50 GB | Em tempo real | Sem limite | 12 meses |

Em tempo real, as notas sincronizam sozinhas enquanto você escreve. Nos planos pagos, o botão direito numa nota mostra as **Versões anteriores**: escolha uma e traga de volta. Os planos pagos também publicam sites pelo Blup Cloud e incluem IA para narrar e traduzir.

## Pelo GitHub

1. Conecte a sua conta do GitHub, com a mesma chave usada para publicar.
2. O Blup cria um repositório **privado**: só você vê.
3. Clique em **Conferir**. O Blup compara as notas daqui com as do GitHub e **mostra o que vai fazer** antes de fazer: o que mudou aqui é enviado, o que mudou em outro computador é recebido, e o que foi apagado de um lado é apagado do outro.
4. Se estiver tudo certo, clique em **Sincronizar agora**.

> [!info] Pelo Obsidian
> Rode `blup sincronizar`. A linha de comando sincroniza pelo GitHub, do mesmo jeito.

## Quando os dois lados mudaram

Se a mesma nota foi editada aqui e em outro computador, nada se perde: a sua versão fica, e a outra vira uma cópia com "conflict" no nome. Compare as duas e apague a que sobrar.

> [!info] Nada some de repente
> Notas apagadas em outro computador vão para uma lixeira fora da pasta das notas. Se a pasta do site aparecer vazia de repente (por exemplo, se ela mudou de lugar), o Blup para e não apaga nada na nuvem.
