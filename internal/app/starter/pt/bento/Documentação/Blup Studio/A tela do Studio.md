---
title: A tela do Studio
description: Faixa lateral, árvore de notas, editor, prévia, terminal e seletor de site.
weight: 1
---
Ao abrir um site, o Studio mostra a área de trabalho. Da esquerda para a direita:

1. **Faixa lateral**: os ícones das telas.
2. **Árvore de notas**: as pastas e notas do site.
3. **Editor**: a nota aberta.
4. **Prévia**: o site de verdade, ao lado.
5. **Terminal** (opcional): embaixo, quando você abre.

No topo ficam os botões **Prévia**, **Sincronizar** e **Publicar**.

![O Blup no terminal](../../Anexos/blup-terminal.jpg)

## A faixa lateral

Passe o mouse num ícone para ver o nome. Prefere ver sempre os nomes? Em **Configurações > Aparência**, ligue **Mostrar nomes no menu lateral**.

Num **site**, as telas vêm em grupos:

| Grupo | Telas |
| --- | --- |
| Conteúdo | Escrever, Anexos, Autores |
| Site | Site, Seções (só no Bento), Idiomas |
| Público | Comentários, Feedback (no Bento, com a seção) e Podcast (nos blogs, com a seção) |
| Privacidade | Cookies e estatísticas |
| Embaixo | Publicar, Lixeira e o terminal |

Num **cofre de anotações**, a faixa muda: Escrever, Mapa de notas e Anexos; depois Novo canvas, Novo desenho, Novo banco de dados e Nota do dia; embaixo, Sincronizar, Lixeira e o terminal. Veja [O que é um cofre](../Cofres%20de%20notas/O%20que%20é%20um%20cofre.md).

No rodapé da faixa ficam a **Ajuda** (abre esta documentação), a **engrenagem** das [Configurações do Studio](Configurações%20do%20Studio.md) e o **seletor de site**.

## O seletor de site

Clique nele para trocar de site ou cofre sem voltar à tela inicial. Ele separa **Sites** e **Cofres**, mostra primeiro os abertos há pouco (até cinco de cada) e tem **Ver todos** e **Tela inicial**.

## A árvore de notas

- **Arquivos** mostra as pastas e notas. **Buscar** procura no nome e no texto das notas, sem ligar para acentos e maiúsculas, e mostra as linhas encontradas.
- **Nova nota** e **Nova pasta** criam no lugar: aparece um campo de nome dentro da pasta escolhida; Enter cria, Esc desiste.
- Clique numa pasta para abrir ou fechar. Ela passa a ser o lugar das notas novas.
- **Expandir tudo** e **Recolher tudo** abrem e fecham todas as pastas.

O botão direito numa nota abre o menu:

| Opção | O que faz |
| --- | --- |
| Nova nota aqui, Nova pasta aqui | Cria dentro da pasta |
| Renomear | Troca o nome. Os links das outras notas são corrigidos |
| Mover para… | Escolhe outra pasta. Os links também são corrigidos |
| Fazer uma cópia | Cria "Nome 2" ao lado |
| Copiar link da nota | Para colar em outra nota |
| Mostrar na pasta do sistema | Abre o gerenciador de arquivos |
| Exportar como HTML | Uma página só, com as imagens dentro |
| Imprimir ou salvar em PDF | Abre a impressão do sistema |
| Versões anteriores | Com o Blup Cloud, mostra o histórico da nota |
| Apagar | Vai para a Lixeira, e dá para restaurar |

Pastas de autores e de anexos não aparecem na árvore: elas têm telas próprias, **Autores** e **Anexos**.

## Salvamento

O Studio salva sozinho enquanto você escreve, e antes de trocar de nota ou sair. O estado aparece na tela: salvo, salvando ou não salvo. `Ctrl+S` também salva.

Se uma nota mudar fora do Studio (no Obsidian ou por uma IA no terminal), a árvore se atualiza. Se você estava editando a mesma nota, o Studio avisa e deixa escolher entre **Carregar a nova** e **Manter a minha**.

## Atalhos principais

| Ação | Teclas |
| --- | --- |
| Nova nota | `Ctrl+N` |
| Buscar notas | `Ctrl+Shift+F` |
| Mostrar ou esconder o painel de arquivos | `Ctrl+Alt+S` |
| Ligar ou desligar a prévia | `Ctrl+Alt+P` |
| Abrir ou fechar o terminal | Ctrl e a tecla do acento grave |
| Configurações | `Ctrl+,` |

Todos podem ser trocados ou desligados em **Configurações > Atalhos**.

> [!tip] O Studio de qualquer lugar
> O atalho global `Alt+Espaço` mostra e esconde o Blup Studio mesmo com outro programa na frente. Dá para trocar nas configurações.
