---
title: Instalar e criar o site
date: {{date-1}}
description: Baixe o Blup Studio e crie o site respondendo uma pergunta de cada vez. Ou use a linha de comando.
tags: [Começando, Studio]
image: "[[blup-dicas.jpg]]"
---
## Instalar o Blup Studio

O **Blup Studio** se baixa em **getblup.io**, para Linux, macOS e Windows. Instale como qualquer app e abra.

Na primeira vez, o Studio dá boas-vindas e faz algumas perguntas rápidas: o idioma, se você quer conectar a sua conta do GitHub agora e uma senha geral para os posts com senha. Dá para pular o que quiser e voltar depois, nas **Configurações** (a engrenagem, embaixo do menu ao lado).

## Criar um site

Na tela inicial, clique em **Criar site novo**. O Studio faz uma pergunta de cada vez, e dá para mudar tudo depois.

1. **Que tipo de site**: blog pessoal ou site de produto.
2. **Qual tema**: Mochi (este aqui, com cards e capas) ou Nori, para o blog; Bento, para o site de produto.
3. **Como o site se chama**: nome, uma frase sobre ele, seu nome e o idioma do site.
4. **Onde ficam as notas**: numa pasta nova (o jeito mais simples), numa pasta que você escolher ou numa pasta de notas que você já usa.
5. **Tudo pronto?** Confira e crie o site.

Se a pasta estiver vazia, o Blup coloca o site de exemplo do tema: este tutorial.

> [!tip] Já tem uma pasta de notas?
> Se escolher uma pasta de notas que você já usa, deixe marcado **Ajustar a pasta para que os links funcionem no site**. O Blup ajusta as configurações das notas da pasta, e os links passam a ser gravados no formato que o site entende.

## A janela do site

- **Arquivos**, à esquerda: as notas e pastas. Use **Nova nota** e **Nova pasta** para criar, e a busca para achar uma nota.
- **O editor**, no meio: onde você escreve. Em cima dele fica o painel **Propriedades** da nota.
- **Prévia**, à direita: o site de verdade, atualizado enquanto você escreve.
- **O menu ao lado**: Escrever, Anexos, Autores, Site, Seções, Idiomas, Comentários, Cookies e estatísticas, Publicar e Lixeira, além do terminal.

Não existe botão de salvar: o Studio salva sozinho enquanto você escreve.

Clique com o botão direito numa nota para criar outra ali, fazer uma cópia, mover, copiar o link, renomear, exportar ou apagar. **Renomear** atualiza sozinho os links que apontam para a nota, e **Apagar** manda a nota para a **Lixeira**, de onde dá para restaurar.

## Pela linha de comando

Prefere o terminal? Instale a linha de comando `blup`.

No macOS ou Linux, abra o Terminal e cole:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

No Windows, abra o PowerShell e cole:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Depois, rode `blup`. Na primeira vez ele pergunta o idioma e abre o assistente de site novo: qual cofre do Obsidian, qual pasta dele, o tema (Mochi ou Bento), o nome do site e onde salvar. Cada pergunta já vem com uma resposta sugerida; na dúvida, aperte Enter.

| Comando | O que faz |
| --- | --- |
| `blup novo` | Cria um site novo |
| `blup ver` | Abre a prévia no navegador |
| `blup publicar` | Coloca o site no ar |
| `blup personalizar` | Menu, redes, logo, comentários e cookies |
| `blup sincronizar` | Guarda as notas num repositório privado no GitHub |
| `blup converter` | Troca `[[wikilinks]]` por links padrão |
| `blup atualizar` | Atualiza o Blup |

Os mesmos comandos funcionam em inglês: `new`, `preview`, `publish`, `customize`, `sync`, `convert` e `update`.

> [!info] Os dois juntos
> Um site criado no Studio abre na linha de comando, e o contrário também. As notas são as mesmas; só muda onde você clica.
