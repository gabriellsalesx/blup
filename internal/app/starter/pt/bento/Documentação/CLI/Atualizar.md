---
title: Atualizar
description: O Blup avisa quando há versão nova e só atualiza se você quiser.
weight: 5
---
Uma vez por dia, ao abrir o menu, o Blup confere se há versão nova. Se houver, mostra o que mudou e pergunta:

- **Atualizar agora**
- **Agora não**
- **Não avisar desta versão**

Para atualizar a qualquer momento: `blup atualizar`.

## Voltar atrás

```sh
blup atualizar --rollback
```

Volta para a versão anterior, guardada na atualização.

## Os temas

Os temas vêm dentro do Blup: depois de atualizar, os sites recebem o tema novo na próxima prévia ou publicação. Suas notas não mudam.

> [!info] Sem internet ou com pressa?
> A verificação desiste em 2 segundos e nunca impede o Blup de abrir.
