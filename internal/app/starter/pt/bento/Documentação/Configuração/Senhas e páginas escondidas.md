---
title: Senhas e páginas escondidas
description: Proteja páginas com senha e publique páginas que só quem tem o link encontra.
weight: 4
---
## Senha

Qualquer página ou post pode ter senha:

```yaml
password: minha-senha
```

Quem abrir vê o título e um campo para a senha. O conteúdo é **criptografado** quando o site é gerado (AES-256-GCM): sem a senha, ninguém lê, nem pelo código da página. Não há servidor.

Para usar a mesma senha em várias páginas, defina uma **senha geral** em `blup personalizar` e escreva:

```yaml
password: sim
```

A senha geral fica só no seu computador, no blup.toml.

> [!warning] O que continua visível
> Título, descrição e capa aparecem nas listas. Imagens dentro da página não são criptografadas.

## Páginas escondidas

```yaml
hidden: true
```

A página é publicada, mas sai das listas, das tags e da busca do site, e pede ao Google para não aparecer. Quem tem o link abre normalmente. Combina bem com senha.

## Rascunhos

```yaml
draft: true
```

O rascunho aparece só na prévia (`blup ver`) e nunca é publicado.
