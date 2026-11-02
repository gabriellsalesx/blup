---
title: Sincronizar as notas
description: As mesmas notas em vários computadores, criptografadas de ponta a ponta. Ou num repositório do GitHub.
weight: 2
---
Sincronizar guarda uma cópia das notas na nuvem. Em outro computador, as notas chegam; edite onde quiser e elas se encontram.

![Dois computadores sincronizando](../../Anexos/blup-sincronizar.jpg)

## Escolher onde guardar

Clique em **Sincronizar**, no topo (num site) ou na faixa lateral (num cofre). Em **Onde guardar as notas**:

- **Blup Cloud** (recomendado): criptografado de ponta a ponta. Nem o Blup consegue ler.
- **GitHub**: num repositório privado da sua conta do GitHub. Sem git.

## Pelo Blup Cloud

### Na primeira vez

1. Entre na sua conta (veja [Conta e planos](Conta%20e%20planos.md)).
2. **Crie a senha das notas**, com pelo menos 8 caracteres. Ela embaralha as notas antes de saírem do computador.
3. **Guarde a chave de recuperação**. Ela aparece uma vez só: copie e guarde num lugar seguro, fora deste computador. Marque **Guardei a chave num lugar seguro** e clique em **Continuar**.

> [!danger] Ninguém mais tem essa senha
> Nem o Blup. Se você perder a senha **e** a chave de recuperação, as notas na nuvem não voltam. As notas no seu computador continuam lá.

### Em outro computador

1. Instale o Studio e entre na mesma conta.
2. O Studio pede **Digite a senha das notas**. Use a senha ou a chave de recuperação e clique em **Abrir as notas**.
3. Escolha os sites e cofres para **Trazer**.

### Quando sincroniza

- **Grátis**: quando você toca em **Sincronizar**.
- **Pro, Max e Ultra**: sozinho, enquanto você escreve. Os outros computadores ficam sabendo na hora.

### Como funciona a criptografia

Tudo é compactado e cifrado no seu computador, antes de sair. A nuvem guarda só pedaços ilegíveis: não vê o texto, nem os nomes das notas e pastas. A chave que abre as notas é protegida pela sua senha e, numa segunda cópia, pela chave de recuperação.

Arquivos maiores que o limite do seu plano não sincronizam, e o Studio avisa quais.

## Pelo GitHub

1. Em **Onde guardar as notas**, escolha **GitHub** e conecte a sua conta (uma chave, como na publicação).
2. O Blup cria um repositório **privado** para as notas.
3. **Conferir** mostra quantas notas vão ser enviadas e recebidas. **Sincronizar agora** faz.

Em outro computador, na tela inicial, **Já tenho um site do Blup** lista os sites guardados no seu GitHub para **Trazer**. Pela linha de comando, veja [Sincronizar](../Linha%20de%20comando/Sincronizar.md).

## Conflitos

Se a mesma nota foi editada em dois computadores antes de sincronizar, o Blup **guarda as duas**: a sua, com o nome normal, e a outra, com "conflict" e a data no nome. Compare, junte o que quiser e apague a cópia.

| Situação | O que acontece |
| --- | --- |
| Mudou só aqui | É enviada |
| Mudou só em outro computador | É recebida |
| Apagada de um lado | É apagada do outro |
| Apagada de um lado e editada do outro | A versão editada fica |
| Editada dos dois lados | As duas ficam |

> [!tip] O que não sincroniza
> Pastas que começam com ponto ficam de fora. As configurações do site que precisam viajar (nome, tema, destino da publicação) vão junto; senhas e chaves de acesso, não.
