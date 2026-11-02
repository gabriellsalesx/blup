---
title: Senhas e páginas escondidas
description: Proteja páginas com senha, esconda páginas das listas e guarde rascunhos.
weight: 7
---
![O Blup com um cadeado](../../Anexos/blup-senha.jpg)

## Senha

Qualquer página ou post pode pedir senha. Quem abrir vê o título e um campo para a senha. O conteúdo é **criptografado no seu computador**, quando o site é gerado (AES-256-GCM): sem a senha, ninguém lê, nem pelo código da página.

### Pelo Blup Studio

Nas **Propriedades** da nota, em **Senha**, escolha:

- **Senha geral**: a mesma senha para vários posts. Ela fica em **Configurações > Senha geral** e vale para todos os seus sites.
- **Senha própria**: uma senha só deste post, em **Senha deste post**.

### Direto na nota

```yaml
password: minha-senha
```

Para usar a senha geral, escreva:

```yaml
password: sim
```

Na linha de comando, `blup personalizar` também define uma senha geral só para aquele site. As senhas gerais ficam só no seu computador, nunca no site.

> [!warning] O que continua visível
> Título, descrição e capa aparecem nas listas. Imagens dentro da página não são criptografadas. Posts com senha também não são narrados nem entram no podcast, porque o áudio entregaria o texto.

## Páginas escondidas

No Studio, ligue **Escondida** nas propriedades. Direto na nota:

```yaml
hidden: true
```

A página é publicada, mas sai das listas, das tags e da busca do site, e pede ao Google para não aparecer. Quem tem o link abre normalmente. Combina bem com senha.

## Rascunhos

No Studio, ligue **Rascunho**. Direto na nota:

```yaml
draft: true
```

O rascunho aparece só na prévia e nunca é publicado.

## Trechos que só você vê

Trechos entre `%%` também nunca vão para o site. Veja [Fórmulas, código e comentários](../Escrevendo/Fórmulas,%20código%20e%20comentários.md).
