---
title: Menu e botão
description: Escolha os itens do menu do topo e o botão de destaque.
weight: 2
---
## Menu automático

Sem a propriedade **menu**, o menu do topo mostra as seções do site, na ordem do **weight** de cada uma.

## Menu escolhido

Na nota **_index** da pasta do site, um item por linha:

```yaml
menu:
  - Documentação
  - Ajuda | Central de ajuda
  - Novidades | Changelog
  - Preços | https://acme.com/precos
```

| Linha | Vira |
| --- | --- |
| `Documentação` | Link para a pasta (ou nota) com esse nome ou título. |
| `Ajuda \| Central de ajuda` | O texto "Ajuda", levando à pasta "Central de ajuda". |
| `Preços \| https://...` | Link para outro site. |
| `Início` | A página inicial. |
| `Blog \| /blog/` | Um endereço do próprio site. |

O item da página atual fica marcado.

## O botão

```yaml
button_text: Criar conta
button_link: https://app.acme.com/cadastro
```

Ele aparece no canto superior direito, em destaque. Sem as duas propriedades, não há botão.
