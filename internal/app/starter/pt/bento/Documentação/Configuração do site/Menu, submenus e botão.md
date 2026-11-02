---
title: Menu, submenus e botão
description: Escolha os itens do menu do topo, agrupe em submenus e destaque um botão.
weight: 2
---
## Menu automático

Sem um menu escolhido, o menu do topo mostra as seções do site, na ordem do **weight** de cada uma.

## Pelo Blup Studio

Na tela **Site**, bloco **Menu do topo**:

1. **Adicionar página** escolhe uma pasta ou nota do site.
2. **Adicionar link** cria um item para outro endereço: um nome e o `https://`.
3. O nome de cada item é livre. Clique para editar.
4. **Arraste pela alça** para mudar a ordem.

### Submenus

- **Arraste um item para cima de outro**: ele entra no submenu do de cima. O item de cima passa a só abrir o submenu, sem link.
- **Novo submenu** cria uma entrada vazia com um nome. Arraste itens para dentro dela.
- Os botões de cada item fazem o mesmo sem mouse: subir, descer, **Pôr no submenu de cima**, **Tirar do submenu** e remover.
- Há um nível só: um submenu não entra em outro.

### Botão do topo

No bloco **Botão do topo**, escreva o **Texto do botão** e **Para onde leva**. Ele aparece à direita do menu, em destaque. Deixe o texto vazio para não ter botão.

## Pelo Obsidian

Na nota **_index** da pasta do site, um item por linha:

```yaml
menu:
  - Documentação
  - Ajuda | Central de ajuda
  - Recursos |
  - + Novidades | Changelog
  - + Roadmap
  - Preços | https://acme.com/precos
button_text: Criar conta
button_link: https://app.acme.com/cadastro
```

| Linha | Vira |
| --- | --- |
| `Documentação` | Link para a pasta (ou nota) com esse nome ou título. |
| `Ajuda \| Central de ajuda` | O texto "Ajuda", levando à pasta "Central de ajuda". |
| `Recursos \|` | Uma entrada que só abre o submenu. |
| `+ Novidades \| Changelog` | Com `+` na frente, o item entra no submenu da linha de cima. |
| `Preços \| https://...` | Link para outro site. |
| `Início` | A página inicial. |
| `Blog \| /blog/` | Um endereço do próprio site. |

O item da página atual fica marcado.

## Pela linha de comando

`blup personalizar` > menu monta o menu escolhendo pastas, tags e links, e pergunta o botão. Submenus não são refeitos pelo terminal: ele avisa e deixa você editar no Studio ou no Obsidian.
