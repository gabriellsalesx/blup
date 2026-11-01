# Blup Studio

Fontes: [entrada desktop](../../studio/main.go), [serviço principal](../../studio/studio.go), [aplicação Svelte](../../studio/frontend/src/App.svelte), [workspace](../../studio/frontend/src/Workspace.svelte), `studio/frontend/package.json` e os arquivos de serviço em `studio/`.

## Arquitetura

O Studio é um aplicativo Wails v3 com backend Go e frontend Svelte 5, TypeScript e Vite. `studio/go.mod` declara Wails `v3.1.0` e usa o módulo principal por `replace => ../`. Não mantém outra implementação completa da CLI: chama os fluxos e APIs de `internal/app` para criar, configurar, publicar, sincronizar e narrar.

`studio/main.go` embute `all:frontend/dist` no binário e registra uma instância de `Studio` como serviço Wails. O runtime serve os assets do frontend e faz a ponte de chamadas. A janela inicial mede 1360 por 860, com mínimo 900 por 560. O serviço mantém um site e um processo de prévia por vez.

O frontend usa runes Svelte como `$state`, `$derived`, `$effect` e `$props`. A entrada `src/main.ts` carrega idioma, tema e mecanismo de undo antes de montar `App`. As traduções da janela ficam em `src/i18n.ts`; as mensagens de trabalho Go continuam em `internal/i18n`.

O idioma inicial segue `config.Global.Lang`, senão `i18n.System()`, em `studio/main.go`. `SetLanguage`, em `studio/about.go`, aceita `pt`, `en` ou `es`, converte outros valores para `en` e salva o idioma global. As Configurações recarregam a interface após a troca. Os rótulos da bandeja são montados no início em `studio/main.go` e só mudam ao reiniciar. Isso não troca o idioma dos sites existentes.

## Bindings e eventos

Os bindings TypeScript ficam em `studio/frontend/bindings/github.com/gabriellsalesx/blup/`, especialmente `studio/studio.ts` e `studio/models.ts`, com modelos de `internal/app`, `config` e `deploy/cfpages`. A interface importa essas funções em vez de construir endpoints HTTP próprios para cada operação local.

`studio/build/Taskfile.yml` gera bindings com `wails3 generate bindings`, flags de build, limpeza do destino, TypeScript e interfaces. `vite.config.ts` aplica o plugin Wails ao diretório de bindings. Eles são artefatos de integração derivados das assinaturas Go e precisam acompanhar as alterações de serviço.

| Serviço Go | Grupos de operações expostas |
| --- | --- |
| `studio.go` | Sites, abrir, árvore, leitura/escrita simples, prévia, mapa nota/página e consulta `Language` |
| `create.go` | Descoberta de cofre/pastas, sugestões, validação, criação, abrir existente, recentes, retirada e exclusão de site |
| `notes.go` | Corpo/propriedades, criar nota/pasta, capa, imagens coladas e autores disponíveis |
| `files.go` | Renomear, mover, copiar, lixeira, link Markdown, abrir no Obsidian e gerenciador de arquivos |
| `settings.go` | Identidade, menu, redes, imagens, seções, cookies, senhas, feedback e lixeira |
| `feedback_board.go` | `FeedbackPosts` e `SetFeedbackStatus`, para os pedidos reais do quadro Bento |
| `translate.go`, `localtr.go` | Idiomas de notas e site, tradução remota/local, revisão, textos de propriedades e atualização da prévia |
| `authors.go`, `attachments.go` | CRUD de perfis e gestão dos anexos |
| `publish.go` | Contas, destino, publicação, sync e domínio |
| `cloud.go` | Descoberta e recuperação GitHub, senha geral entre sites |
| `narration.go` | Modelos, vozes, amostras, instalação local, chave remota e geração |
| `media.go` | Detecção local de reprodução MP3 por `canPlayMP3` |
| `terminal.go` | Agentes instalados, PTY, entrada/saída, tamanho, snapshots e undo |
| `layout.go` | Organização da home e preferências |
| `boards.go`, `graph.go` | Canvas, desenho e `.table` como texto, imagens dos quadros e mapa de notas (cofres) |
| `database.go`, `tags.go` | Bancos de dados, células (`SetProps`), nota do dia e tags conhecidas (cofres) |
| `desktop.go`, `hyprland.go`, `about.go` | Bandeja, autostart, atalhos, alteração de idioma por `SetLanguage` e versão |

| Evento | Payload e finalidade |
| --- | --- |
| `progress` | `{kind, text}`; etapas, avisos e resultados de publicação/sync |
| `previewChanged` | Notifica rebuild Hugo para recarregar a página da prévia |
| `previewRestarted` | Nova URL quando o processo de prévia reiniciado muda de endereço |
| `notesChanged` | Caminhos relativos separados por quebra de linha; alterações externas |
| `term` | `{id, data, exit}`; bytes do terminal em base64 ou término |
| `narrate` | `{note, done, total}`; progresso de narração |
| `voiceInstall` | `{done, total, step}`; download do motor local |
| `translate` | `{note, lang, done, total}`; partes da tradução remota de uma nota |
| `translateSite` | `{lang, done, total, note}`; progresso da tradução do site |

`running` em `publish.go` usa mutex e devolve `busy` se já houver publicação/sync em execução. Durante a operação encaminha `ui.Report` a eventos da janela e atribui `nil` ao hook ao terminar, sem restaurar um valor anterior. Esse mutex não é um agendador geral para todos os serviços.

## Componentes de interface

Telas novas usam os componentes de `frontend/src/ui/` e as classes globais de `app.css`, para todo campo, opção e botão terem a mesma aparência:

| Componente | Uso |
| --- | --- |
| `TextField` | Campo de texto com rótulo em cima e ajuda ou erro embaixo; `large` para o nome de um site ou cofre novo |
| `Choice` | Uma opção entre poucas, em cartão com título, linha de ajuda e o anel azul quando escolhida |
| `StepScreen` | Tela inteira centralizada das boas-vindas e da criação: imagem, título, linha de apoio, conteúdo e os botões embaixo |

Classes globais: `.btn` (`.ghost`, `.big`), `.card`, `.field`, `.choices`, `.link`, `.actions`, `.row`, `.muted`, `.bad`, `.ok`, `.badge`.

## Telas de entrada

`App.svelte` escolhe boas-vindas, home, criação ou workspace. Se `welcomed` ainda é falso e não há sites conhecidos, exibe `Welcome.svelte`. Se já houver sites, marca boas-vindas como vistas. Pode reabrir `lastSite` desde que continue entre os sites válidos; o workspace tenta reabrir a última nota.

`Welcome.svelte` segue as etapas `lang`, `github`, `found` somente quando encontra sites, `password`, `window` e `done`. Primeiro exige escolher português ou inglês. O botão de pular existe somente em GitHub e senha. Para continuar pela ação principal do GitHub, precisa haver conta conectada e a busca deve ter terminado; salvar a senha e continuar exige valor não vazio. A etapa de janela permite continuar sem alterar as opções. As escolhas são salvas conforme feitas; concluir grava `welcomed = true` e `shortcutTip = true` nas preferências.

A etapa `found` apenas lista os sites descobertos. A recuperação acontece no bloco "No seu GitHub" de `Onboarding.svelte`: aparece com conta GitHub conectada, consulta `CloudSites`, filtra os resultados que já têm `dir` local e oferece a ação que chama `BringSite`. A descoberta e a recuperação estão detalhadas em [notas e sync](notas-e-sync.md).

`Home.svelte` lista sites locais, permite arrastar para ordenar ou criar grupos e oferece menu com ações equivalentes. Grupos e ordem ficam em `studio.json`. Retirar da lista permite desfazer por toast. Não apaga automaticamente os diretórios. Sem sites, inclusive ao carregar a lista vazia, encaminha diretamente ao Onboarding.

`Onboarding.svelte` apresenta categoria, tema, seções Bento, nome/dados, origem das notas e revisão. Blog oferece Nori e Mochi; produto oferece Bento. Permite usar uma nova pasta, pasta escolhida ou cofre/pasta dele. Consulta `SuggestPlaces`, `VaultFolders`, `NeedsLinkAdjust` e `CheckSiteDir`, depois chama `CreateSite` com as respostas. Também permite abrir um site encontrado, restaurar recente ou trazer site GitHub.

`FindSite`, em `internal/app/find_site.go`, pode associar uma pasta escolhida ao diretório técnico: procura nela, em pais próximos, lugares usuais e sites conhecidos, com profundidade limitada. Prefere correspondência exata da pasta de notas, depois pasta dentro dela e depois contêiner. Isso é mais amplo que a resolução de diretório atual feita pelo comando CLI.

## Workspace e salvamento

O workspace tem faixa lateral de ações, explorador de notas, editor, painel de prévia e terminal opcional na parte inferior. A barra superior oferece Prévia, Sincronizar e Publicar. O rodapé lateral inclui ajuda, engrenagem de configurações e menu de troca e exclusão de site; a ação Seções só aparece no Bento. O explorador alterna entre árvore e busca. A busca local filtra nomes de notas e limita a 80 resultados; não é a busca de conteúdo Pagefind do site.

Quando existe atalho global e `shortcutTip` ainda é falso, o workspace exibe uma dica. Em Hyprland sem configuração automática Omarchy, mostra a variante com instruções para cadastrar o binding; a escolha de outro atalho pode abrir a aba Atalhos. A conclusão das boas-vindas já marca essa dica como vista.

`Studio.Open` valida o site, interrompe a prévia anterior, fecha terminais, muda o site atual, inicia watcher e retorna árvore. A árvore ordena pastas antes de notas, por nome sem distinção de caixa. Exclui itens ocultos, guias de agentes na raiz e as pastas de autores/anexos que têm painéis próprios. Só apresenta `.md` como notas.

`ReadNote` separa YAML, corpo e tipo Bento da nota. `SaveNote` conserva propriedades não editadas pelo formulário e salva corpo mais alterações específicas. As operações recusam caminhos lexicalmente fora da raiz das notas. Essa verificação de caminho não é apresentada pelo código como isolamento completo contra todos os links simbólicos existentes no disco.

`Workspace.svelte` agenda salvamento 300 ms após mudanças e salva antes de trocar nota, sair ou executar ações relevantes. Ctrl/Cmd+S no editor também solicita salvamento. Mantém estados salvo, salvando e não salvo e apresenta falhas. Preferências lembram nota por site e larguras de painéis em `<config>/blup/studio-prefs.json`, gerido por `studio/layout.go`. A senha geral entre sites fica em `credentials.toml`. Veja os campos e a resolução de diretórios em [configuração](configuracao.md).

O watcher recebe alterações de agentes ou outros editores. A árvore é atualizada; uma nota sem edições pendentes é recarregada. Se houver edição local, a tela mostra que houve mudança externa e permite recarregar, evitando descrever isso como merge automático de documentos concorrentes.

O explorador tem criação de nota/pasta, expansão, recolhimento e menu de contexto. `internal/app/files_api.go` implementa renomear e mover com correção textual de links Markdown e wikilinks nas notas. `CopyNote` recebe apenas o caminho de origem e cria uma cópia ao lado com nome automático `Nome 2.md`, `Nome 3.md` e assim por diante, usando o primeiro número disponível; não pede um nome. A exclusão vai à lixeira. A correção de links usa os formatos reconhecidos por regex, sem parser semântico completo de todo conteúdo Markdown.

## Editor e propriedades

`Editor.svelte` usa CodeMirror 6 com Markdown, cores, undo/redo, sugestões, tabelas e comandos de formatação. Fornece negrito, itálico, tachado, destaque, código, headings, listas, tarefas, citações, callouts, imagens, links, tabelas, regra e comentários Obsidian. O botão de alinhamento oferece esquerda, centro, direita e justificado, usando atributos como `{.text-center}` nos casos aplicáveis. Menu de contexto inclui copiar, recortar, colar e colar como texto sem formatação.

Digitar `[[` oferece notas, mas a seleção insere um link Markdown relativo compatível com o site. Há seletor de nota. O botão de link e Ctrl/Cmd+K chamam `mdcommands.link`, que insere `[texto](https://)` e seleciona o endereço para edição no próprio documento, sem diálogo de URL. Colar ou arrastar imagens grava anexo e insere referência. `mdcommands.ts`, `table.ts`, `callouts.ts` e `mdextras.ts` concentram comportamentos de edição.

O corretor usa Hunspell em WebAssembly dentro de `spell.worker.ts`. Carrega dicionários empacotados de português, inglês ou espanhol e palavras personalizadas. Oferece sublinhado, sugestões e conclusão. O idioma acompanha a interface quando não há override. Não envia o texto do corretor a um serviço remoto.

`Props.svelte` expõe título, descrição, data, tags, autores, imagem, rascunho, escondido e senha. Para Bento também usa ordem pela propriedade `weight`, status e categoria quando aplicáveis. Propriedades adicionais desconhecidas pelo formulário ficam no arquivo. A narração aparece apenas fora de `_index.md`, sem `blup` preenchido e com tipo vazio ou `blog`; atualiza `audio`, `audio_hash` e `audio_seconds` e avisa quando o hash do texto difere. As condições e os motores estão em [narração e voz](narracao-e-voz.md).

## Cofres de anotações

Um cofre de anotações é um site com `kind = "notes"` (`site.IsNotes()` no Go, `notesVault` em `Workspace.svelte`): só notas, sem tema nem publicação. As seções abaixo descrevem o que o Studio acrescenta nesse modo; num site comum essas ações não aparecem e os atalhos delas não fazem nada.

### Faixa lateral e árvore

A faixa da esquerda (`.ribbon`) mostra só ícones, com o nome no hover. No cofre ela tem Escrever, Mapa de notas, Anexos, Sincronizar e Lixeira; depois um separador com Novo canvas, Novo desenho, Novo banco de dados e Nota do dia; embaixo, o terminal. Sem botão de Prévia nem de Publicar na barra superior.

A árvore (`Tree.svelte`) lista notas, pastas, canvases (`.canvas`), desenhos (`.draw`) e bancos de dados (`.table`), cada um com ícone próprio; `Studio.Open` só inclui essas três extensões quando o site é um cofre. Clicar numa pasta abre ou fecha e a torna o lugar das notas novas.

- Criar no lugar: Nova nota e Nova pasta (lateral, menu da pasta, `Ctrl+N`) abrem a pasta de destino e põem um campo de nome no topo dela; Enter ou clicar fora cria, Esc desiste. Canvas e desenho não pedem nome (veja abaixo).
- Renomear no lugar: Renomear, no menu de contexto, troca o nome por um campo com tudo selecionado; Enter ou clicar fora salva, Esc desiste. A extensão é mantida por `fileExt` (`.draw`, `.canvas`, `.table`, `.md` ou a de `path.Ext`).
- Arrastar: no cofre, nota ou pasta solta sobre uma pasta vai para dentro dela; solta sobre uma nota vai para a pasta dessa nota; solta no espaço vazio embaixo da árvore vai para a raiz. Uma pasta mantida sob o item arrastado por 700 ms se abre. Não aceita soltar dentro de si mesma, numa subpasta sua ou onde já está. O movimento usa `MoveItem`, que corrige os links como o "Mover para…". Notas da árvore também podem ser arrastadas para um canvas aberto, onde viram cartões.

### Página da nota e propriedades

A nota do cofre começa pelo nome do arquivo em letra grande (`.inline-title`): editar e sair do campo, ou Enter, renomeia o arquivo; Esc devolve o nome. Abaixo vem `NoteProps.svelte`, no lugar de `Props.svelte`, com apenas criado, atualizado e tags, e o texto em seguida. Ao pé da nota fica o painel "Mencionada em".

Nota nova num cofre nasce com `created` e `updated` do dia e sem `title` (`createFile`, em `studio/notes.go`). Ao salvar (`Workspace.save`), se `created` falta ele é preenchido; se o texto mudou, `updated` não foi editado à mão e ainda não é de hoje, recebe a data de hoje. Os dois campos são `input type="date"` e podem ser alterados.

As tags aceitam Enter ou vírgula para confirmar, Backspace no campo vazio tira a última, um `#` inicial é descartado e espaços viram hífen. As sugestões vêm de `Tags` (`studio/tags.go`): todas as tags das propriedades das notas, das mais usadas para as menos, lidas como lista ou texto separado por vírgulas; mostra até 8.

Se a nota é registro de um banco de dados, os campos do banco aparecem depois das tags (veja Bancos de dados).

### Tags herdadas da pasta

`FolderTags` (`internal/app/quicknotes.go`) decide as tags de uma nota nova na pasta: se a pasta é a pasta de registros de um banco, a tag do banco; senão, as tags presentes em pelo menos metade das notas `.md` diretas da pasta, só quando há duas notas ou mais. Vale para Nova nota (`createFile`) e para a nota do dia.

### Nota do dia

O botão Nota do dia (`Ctrl+Alt+D`) chama `DailyNote(pasta, tag)` com os valores da tradução: pasta `Diário` e tag `diario` em português, `Daily` e `daily` em inglês. A nota é `Diário/AAAA-MM-DD.md`. Na primeira vez do dia é criada com `created` e `updated` e com as tags de `FolderTags` mais a tag do diário na frente, se ainda não estiver entre elas. Toda chamada, nova ou não, acrescenta ao fim do arquivo um título `## HH:MM` com a hora atual. O workspace salva a nota aberta antes, abre a nota do dia e põe o cursor no fim do texto (`atEnd`), pronto para escrever; se ela já estava aberta, recarrega o conteúdo.

## Editor: links, títulos, `@` e `/`

`livemd.ts` aproxima o editor da página lida:

- Links aparecem só pelo texto, em azul (`--blue`) e sublinhados: `[[Nota]]` mostra o nome, `[[Nota|texto]]` mostra o texto, `[texto](nota.md)` e `[texto](https://…)` mostram o texto. A forma escrita volta enquanto o cursor ou a seleção toca o link (digitando, pelas setas ou com o clique direito). Tudo que está dentro de código fica como escrito. Um clique num link para outra nota a abre; um endereço da web abre no navegador só com Ctrl/Cmd e clique, para o clique simples continuar editando.
- O resto do Markdown também aparece desenhado (`livepreview.ts`), nos dois editores (site e notas). A marca volta enquanto o cursor está nela (marcas de texto) ou na linha (blocos):
  - `**`, `*`, `~~`, `==` e a crase do código em linha somem e fica só o efeito; os `#` dos títulos também;
  - citação (`>`) vira uma barra azul à esquerda, sem o `>`;
  - callout (`> [!tip] Título`) mostra o ícone do tipo e o título, ou o nome do tipo quando não há título;
  - bloco de código ganha fundo, a linguagem e o botão Copiar; as cercas ```` ``` ```` só aparecem com o cursor dentro do bloco, e o botão continua lá;
  - `---` vira uma linha (o front matter do topo fica como escrito);
  - item de lista com `-`, `*` ou `+` mostra um ponto; tarefa (`- [ ]`) mostra uma caixa que marca com um clique, e a feita fica riscada;
  - imagem (`![](caminho)` relativo à nota, ou `![[foto.png]]`) aparece no lugar, lida por `NoteImage` como `data:` URL. Imagem da web (`https://…`) espera um clique em "Mostrar imagem de …", porque carregar avisa o servidor dela que a nota foi aberta.
- Títulos recolhem como no Obsidian: uma seta à esquerda de cada `#` a `######` que tenha conteúdo recolhível, visível no hover ou sempre quando recolhido, esconde tudo até o próximo título do mesmo nível ou maior. Os atalhos de dobra vêm do `basicSetup` do CodeMirror: `Ctrl+Shift+[` recolhe, `Ctrl+Shift+]` mostra, `Ctrl+Alt+[` recolhe todos e `Ctrl+Alt+]` mostra todos.

`Editor.svelte` tem dois menus de digitação, ativados só no começo da linha ou depois de um espaço (assim um e-mail não abre o `@`) e fora de código; Enter ou Tab escolhem.

- `@` lista as notas (menos a aberta e os `_index`), na seção Notas, e datas, na seção Datas: hoje, amanhã, ontem e os próximos dias da semana pelo nome. Nota vira link Markdown relativo; data vira texto no formato local (`01/11/2026` em português).
- `/` lista as ações: Título 1 a 3, Lista, Lista numerada, Lista de tarefas, Citação, Destaque (callout), Tabela, Bloco de código, Linha divisória, Link para outra nota (abre o seletor), Link externo, Imagem do computador e Data de hoje. No cofre, também Novo desenho e Novo canvas: criam o arquivo como os botões da faixa e inserem o link para ele na nota.

`[[` continua listando as notas e inserindo link Markdown, como em "Editor e propriedades".

## Mapa de notas e "Mencionada em"

`app.NoteGraph` (`internal/app/graph.go`) percorre a pasta de notas, pulando pastas ocultas e os `_index.md`, e devolve:

- nós: cada `.md` (`note`), `.canvas` (`canvas`) e `.draw` (`drawing`), com caminho, nome sem extensão, pasta, tags e `created` (a propriedade, cortada em 10 caracteres, senão a data de modificação do arquivo);
- arestas, sem repetição e sem laço: links Markdown relativos ou com `/` inicial (ignora endereços com `://`, `mailto:` e âncoras puras, tira `#…` e `?…` e decodifica `%20`), `[[wikilinks]]` (pelo caminho, senão pelo nome, preferindo a mesma pasta) e, num canvas, cada nó `file`. O texto dentro de código (crases e blocos) é ignorado. Desenhos não são lidos em busca de links. Como o arquivo inteiro é lido, links nas propriedades também contam, inclusive os campos Ligação com nota dos bancos de dados.

`Backlinks.svelte` ("Mencionada em") fica sob cada nota do cofre: lista as notas (e canvas) que têm link para a nota aberta, com nome e pasta, e abre a escolhida. Sob cada uma vêm até três linhas onde ela menciona a nota (`Mentions`): o texto da linha sem marcas de Markdown, links pelo texto, e a menção em azul. A menção fica registrada na nota que é citada, não na que cita. Uma menção feita com `@` grava `[@Nome](caminho.md)`: o `@` faz parte do texto do link, então aparece azul no editor e no site. Recolhido ou aberto, lembra a última escolha em `localStorage` (`blup.backlinks`). A lista é calculada ao abrir a nota.

`GraphView.svelte` é a tela Mapa de notas (`Ctrl+Alt+G` alterna com Escrever). Cada nota é um ponto, cada link uma linha; o raio cresce com a raiz do número de links (`graph.ts`). O layout é uma simulação de forças própria, sem biblioteca: repulsão entre pares até 420 px, molas nos links, gravidade para o centro, amortecimento e `alpha` que decai até parar; começa em espiral. Passar o mouse acende a nota e os vizinhos; clique abre; arrastar um ponto o move (fixo enquanto arrasta); arrastar o vazio move o mapa; a roda dá zoom de 0,15 a 4. Os botões Ver tudo (enquadra) e Ler as notas de novo ficam na tela; o mapa também se atualiza sozinho quando notas mudam no disco (evento `notesChanged`).

O painel do canto (Ajustes do mapa) tem quatro partes que abrem e fecham, botão Voltar ao padrão e Fechar (vira um ícone). Tudo é salvo em `localStorage` (`blup.graph`).

| Parte | Controles |
| --- | --- |
| Filtros | Busca (só acende os nomes que batem, não esconde), Pasta (inclui subpastas), Tag, Tags como pontos (um ponto por tag ligado às suas notas), Notas soltas (com a contagem; desligado esconde nós sem links), Canvas e desenhos, Em volta da nota aberta com Profundidade de 1 a 4 links; mostra "N notas, M links" |
| Grupos | Novo grupo: pasta (com subpastas) ou tag, com cor que troca no clique; vale o primeiro grupo que casa |
| Exibição | Setas, Limite para mostrar os nomes (zoom mínimo para os rótulos), Tamanho das notas, Espessura das linhas e Animar: as notas surgem na ordem de `created`, em 3 a 12 s, e as linhas quando as duas pontas já surgiram |
| Forças | Força do centro, Força de repulsão, Força dos links e Distância dos links |

## Canvas e desenhos

Canvases (`.canvas`) e desenhos (`.draw`) são do próprio Studio, feitos em Svelte, sem biblioteca de terceiros; não precisam do Obsidian. A faixa lateral (`Ctrl+Alt+C` e `Ctrl+Alt+W`), o menu de pasta e o menu `/` os criam na hora, sem pedir nome (`CreateCanvas`, `CreateDrawing`): "Desenho 2026-11-01 14.05.33" na pasta `Desenhos` ou "Canvas …" na pasta `Canvas`, ou na pasta cujo menu foi usado, e já abre. Arquivo vazio: `{"nodes":[],"edges":[]}` e `{"version":1,"elements":[]}`.

`studio/boards.go` lê e grava o texto (`ReadBoard`, `SaveBoard`, só `.canvas`, `.draw` e `.table`), guarda imagens coladas e PNGs exportados na pasta de anexos (`SaveBoardImage`, via `app.PasteImage`, devolvendo o caminho no cofre) e entrega imagens do cofre como `data:` (`BoardFile`, só `image/*`). Renomear e copiar mantêm a extensão (`fileExt`). Sincronização e lixeira tratam esses arquivos como qualquer outro.

`CanvasView.svelte` (lógica em `canvas.ts`) grava no formato aberto JSON Canvas: cartões de texto em Markdown, notas do cofre (`CanvasFile.svelte` mostra o texto, ou a imagem), links da web e grupos, ligados por setas com texto e cor. Arrastar o vazio ou a roda move a tela; Ctrl e a roda, ou pinça, dão zoom; dois cliques criam um cartão; arrastar do ponto lateral de um cartão cria uma seta; notas arrastadas da árvore viram cartões. Teclado: Enter edita o selecionado, Delete ou Backspace apaga, `Ctrl+A` seleciona tudo, `Ctrl+Z`, `Ctrl+Shift+Z` e `Ctrl+Y` desfazem e refazem.

`DrawingView.svelte` (lógica em `draw.ts`) desenha retângulo, losango, elipse, seta, linha, desenho livre, texto (solto ou dentro da forma), borracha e imagens, com cor, fundo (suave ou sólido), espessura, tracejado e tamanho de texto; seleção por caixa, redimensionar, duplicar, ordem e exportar PNG para os anexos. Cada ferramenta tem uma letra: H mover a tela, V selecionar, R retângulo, D losango, O elipse, A seta, L linha, P desenho livre, T texto, E borracha. Também: Espaço segurado move a tela, setas movem o selecionado (Shift, de 10 em 10), Enter escreve na forma ou no texto, `Ctrl+D` duplica, `Ctrl+=`, `Ctrl+-` e `Ctrl+0` aproximam, afastam e enquadram. O arquivo é JSON com um elemento por linha; cada elemento tem caixa `x, y, w, h`, e linhas, setas e traços guardam os pontos relativos a ela. Chaves desconhecidas no topo do arquivo são preservadas.

As duas telas têm tela cheia (botão nos controles, F dentro da tela, ou o atalho `F11` do Studio): ocupam a janela inteira, sem faixa, árvore ou barra. Esc sai quando nada está selecionado (no desenho, também com a ferramenta de seleção). Salvam sozinhas e recarregam quando o arquivo muda fora do editor (evento `notesChanged`), se não houver mudança própria pendente.

## Bancos de dados

Um banco de dados do cofre é uma lista de coisas (clientes, tarefas, leituras) em que cada item é uma nota: tem espaço para escrever, links de e para outras notas e aparece na busca e no mapa. Código em `internal/app/database.go`, `studio/database.go`, `DatabaseView.svelte`, `DatabaseNew.svelte`, `FieldInput.svelte` e `db.ts`.

```text
Bancos de dados/Clientes.table   colunas e formas de ver (JSON)
Bancos de dados/Clientes/        uma nota por registro, campos nas propriedades
```

### Arquivo `.table`

JSON do próprio Studio, indentado com tabulação:

| Campo | Conteúdo |
| --- | --- |
| `version` | Sempre `1` |
| `records` | Nome da pasta de registros, ao lado do `.table`; se faltar, o nome do arquivo sem extensão |
| `tag` | Tag de todo registro; ao criar, o nome em minúsculas com hífens no lugar de espaços |
| `columns` | `{key, name, type, options}`; `key` é a propriedade nas notas, `options` são `{value, color}` |
| `views` | `{id, name, type, groupBy, sort, filter, match}`; `type` é `table` ou `board` |

Tipos de coluna (`db.ts`, rótulo em português): `text` Texto, `number` Número, `select` Escolha única, `multi` Várias escolhas, `date` Data, `phone` Telefone, `email` E-mail, `url` Link, `check` Sim/Não, `link` Ligação com nota. Cores das opções: `gray`, `red`, `orange`, `yellow`, `green`, `blue`, `pink`. Nas formas de ver, `sort` é a chave da coluna (vazio para o nome), com `-` na frente para ordem decrescente; `groupBy` é a coluna de escolha única que dá as colunas do quadro; `filter` e `match` guardam o filtro (coluna e opção).

`keyOf` gera a chave a partir do nome da coluna: minúsculas, sem acentos, não alfanuméricos viram hífen, `campo` se ficar vazio, e sufixo `-2`, `-3` quando já existe ou quando colide com `tags`, `created`, `updated` ou `title`.

### Criar

Novo banco de dados (faixa lateral ou `Ctrl+Alt+B`) abre `DatabaseNew.svelte`: escolher um modelo e o nome (o do modelo, até o usuário digitar outro). `CreateDatabase` grava na pasta de bancos do idioma do Studio (`Bancos de dados/` ou `Databases/`), cria a pasta de registros e o `.table`; nome tomado, no arquivo ou na pasta, ganha número ("Clientes 2").

| Modelo | Colunas | Quadro por |
| --- | --- | --- |
| Clientes | Empresa, Telefone, E-mail, Temperatura (Quente, Morno, Frio), Último contato | Temperatura |
| Tarefas | Situação (A fazer, Fazendo, Feito), Prioridade (Alta, Média, Baixa), Prazo, Responsável | Situação |
| Leituras | Autor, Situação (Quero ler, Lendo, Lido), Nota (número), Link | Situação |
| Do zero | Situação (A fazer, Feito); nome padrão "Minha lista" | Situação |

Todo modelo vem com as formas de ver Tabela e Quadro.

### Registros

`ReadDatabase` lê os `.md` diretos da pasta de registros (sem ocultos), em ordem de nome sem distinção de caixa, com as propriedades (datas como `AAAA-MM-DD`) e os primeiros 2000 bytes do texto, usados pela busca. `AddRecord` cria `Nome.md` (ou `Nome 2.md`) com `created` e `updated` de hoje, a tag do banco e as propriedades não vazias recebidas. `SetProps` grava uma célula na nota; valor nulo, vazio, falso ou zero remove a propriedade (a regra de `clean`).

`DatabaseOf` reconhece um registro: a nota está numa pasta `X` e existe `X.table` ao lado dessa pasta com `records` igual a `X`. Nesse caso `ReadNote` devolve `table` e `database`, e a página da nota mostra os campos em Propriedades, com o mesmo `FieldInput` da tabela; criar uma opção ali grava a opção no `.table`. Nota nova criada na pasta de registros pela árvore herda a tag do banco (`FolderTags`).

### Tela do banco

`DatabaseView.svelte` abre o `.table`. A barra tem as abas das formas de ver (e "+ Quadro" quando não há quadro e existe coluna de escolha única), Buscar em tudo…, Filtrar (Tudo ou uma opção de uma coluna de escolha única), Colunas por… no quadro quando há mais de uma coluna de escolha única, e Novo.

- Tabela: cada célula é editável como planilha; clicar no nome abre a nota; a lixeira da linha apaga o registro para a Lixeira, com Desfazer no aviso. O cabeçalho de uma coluna abre o menu dela: nome, tipo, opções (com cor, renomear, tirar, Adicionar opção), Ordenar A → Z, Ordenar Z → A e Apagar coluna; o cabeçalho Nome só ordena. O "+" do fim acrescenta "Nova coluna" do tipo Texto e abre o menu.
- Quadro: uma coluna por opção da coluna de agrupamento, mais "Sem valor"; arrastar um cartão para outra coluna troca a escolha (para "Sem valor", remove a propriedade). O cartão mostra o nome e até três outros campos preenchidos; clique abre a nota; cada coluna tem seu Novo.
- Busca: todas as palavras precisam aparecer no nome, em algum campo ou no começo do texto da nota.
- Novo registro: nome e Enter numa linha no topo (ou no topo da coluna do quadro). No quadro recebe a opção da coluna; na tabela com filtro, recebe a opção filtrada.
- Campos (`FieldInput.svelte`): escolha única e várias escolhas abrem uma lista com "Buscar ou criar…" e Limpar; Telefone tem Ligar (`tel:`) e Abrir no WhatsApp (`wa.me`); E-mail tem Escrever e-mail; Link tem Abrir o link (prefixa `https://` se faltar); Data usa o seletor de data; Sim/Não é uma caixa; Ligação com nota abre o seletor de notas e grava `[[Nome]]`, que aparece no mapa e no "Mencionada em" da nota ligada.

Colunas e formas de ver são gravadas 300 ms depois da mudança (`SaveDatabaseSchema`, que preserva `records` e, se vier vazio, `tag`). Apagar uma coluna não apaga a propriedade das notas. A tela relê os registros quando algo muda na pasta de registros e oferece recarregar quando o próprio `.table` muda fora dela.

## Prévia

`StartPreview` encontra ou baixa Hugo, gera scaffold e inicia servidor restrito a `127.0.0.1`, em memória, com rascunhos e `HUGO_BLUP_MODE=preview`. Desativa live reload nativo e observa log `Total in` para emitir `previewChanged`. Não executa o preparo Pagefind da CLI.

`AddSection`, `RemoveSection` e `RestoreSection`, em [studio/settings.go](../../studio/settings.go), chamam `sectionsChanged` depois de uma operação bem-sucedida. Em [studio/translate.go](../../studio/translate.go), essa rotina solicita o reinício de uma prévia em execução. O motivo é a descoberta de pastas pelo watcher do Hugo: uma seção pode ser copiada com todas as notas antes de o watcher passar a observar a pasta, deixando páginas ausentes até outra alteração. Não basta recarregar o iframe para Hugo descobrir essas notas.

`restartPreview` interrompe o servidor de prévia e inicia outro; se a URL continua igual, emite `previewChanged`, e se mudou, emite `previewRestarted` com o novo endereço. O workspace recebe o endereço e segue a página da nota aberta. A operação trata o servidor local da prévia, sem reiniciar o aplicativo desktop. O caminho de tradução também pode solicitar reinício por `previewFollows` quando mudam os mounts ou os idiomas da configuração gerada.

Os reinícios são serializados pelo mutex `restarting`; a rotina retorna se a prévia já estiver parada. Ao aceitar um processo recém-iniciado, `adopt`, em `studio/studio.go`, confere o site atual e a existência de outro processo já adotado. Descarta o processo que pertence a um site anterior ou que perdeu a disputa, mantendo só uma prévia registrada.

`Preview.svelte` usa dois iframes. O próximo carrega oculto, recebe a posição de scroll e assume o lugar quando sinaliza pronto; há fallback de quatro segundos. A comunicação usa `postMessage` e scripts que o tema injeta somente em prévia. O parâmetro `_blup` identifica os frames. Não é scroll sincronizado por linha de Markdown: conserva a posição da página e segue relações nota/URL.

`PreviewPage` e `NoteAt` leem `blup-map.json` no servidor local. Editar outra nota leva à sua página; navegar em link de página com nota abre essa nota no editor. Taxonomias sem arquivo podem continuar sem correspondente.

Em Linux, `canPlayMP3` procura `libgstaudioparsers.so` nos diretórios GStreamer habituais e em `GST_PLUGIN_PATH`. Se faltar, o servidor de prévia recebe `HUGO_BLUP_NO_MP3=1` e o tema omite o player nessa prévia. É uma detecção por arquivo, não um teste real de decodificação de todos os formatos.

## Painéis do site

| Tela | Operações atuais |
| --- | --- |
| Aparência, `LookPanel.svelte` | Nome, descrição, autor padrão, logo, favicon, menu, botão e redes |
| Autores, `AuthorsPanel.svelte` | Lista, perfil, foto/avatar, cargo, redes, bio, criação, renomeação e remoção |
| Anexos, `AttachmentsPanel.svelte` | Lista por tipo, tamanho e usos, adição, renomeação, exclusão e acesso às notas que usam |
| Seções, `SectionsPanel.svelte` | Somente Bento; acrescenta exemplos, inclusive Feedback, remove para a lixeira e oferece restaurar a seção removida |
| Comentários, `CommentsPanel.svelte` | Ativa/desativa feedback, prepara repositório e orienta instalação do App |
| Feedback, `FeedbackPanel.svelte` | Só no Bento com a seção Feedback: lista issues de pedidos via `FeedbackPosts`, filtra por categoria, abre no GitHub e muda o status via `SetFeedbackStatus` |
| Cookies, `CookiesPanel.svelte` | Ferramentas, IDs, URLs próprias, banner e política |
| Senha, `PasswordPanel.svelte` | Senha geral específica do site e estado da global |
| Publicação, `PublishPanel.svelte` | Conectar contas, destino, publicar, acompanhar log, domínio e sync |
| Lixeira, `TrashPanel.svelte` | Restaurar, excluir item e esvaziar |

Perfis usam `internal/app/authors_api.go`, com avatares SVG embutidos em `internal/app/avatars`. Renomear autor ajusta as propriedades de posts conforme as rotinas implementadas. A pasta de perfis é identificada por `blup: authors`.

### Seções e painel Feedback

`internal/app/starter.go` separa `docsKinds` (`docs`, `roadmap`, `changelog`, `help`, `blog`) de `studioKinds` (`feedback`). `LoadSettings` oferece as duas listas no painel Seções. A criação continua usando as cinco seções de `docsKinds`; para ter o quadro, o dono acrescenta Feedback depois. A pasta modelo nos três idiomas tem `blup: feedback` e `examples`, reconhecida pela propriedade em `obsidian.Sections`, sem depender do nome `Feedback`.

`Workspace.svelte` só oferece o painel Feedback quando `site.model` é `bento` e `settings.sections` contém `feedback` presente. Essa condição não exige comentários ligados; o serviço informa erro se faltam o repositório habilitado ou a credencial GitHub do dono. Os controles de ativação dos comentários e instalação do App permanecem no painel Comentários.

[FeedbackPanel.svelte](../../studio/frontend/src/FeedbackPanel.svelte) carrega `FeedbackPosts` ao abrir e no botão Atualizar. O backend em `studio/feedback_board.go` obtém o site atual e delega a `internal/app/feedback_board.go`. A leitura consulta até três páginas de 100 issues, das mais novas para as mais antigas, e seleciona as marcas de pedido. O painel ordena por votos decrescentes, com número da issue como desempate, e oferece filtro de categoria com contagens. Mostra tipo, título, autor, comentários, votos e código de idioma; clicar no título abre a issue no navegador.

O seletor de cada pedido chama `SetFeedbackStatus(number, status)`, com atualização otimista, indicação de salvamento e restauração do valor anterior em caso de erro. Os valores fixos são `new`, `planned`, `progress`, `done` e `declined`; Feito e Recusado fecham a issue, os outros reabrem. Categorias e opções de status não podem ser editadas por esse painel. Os exemplos da nota não entram na lista de issues nem recebem status por essa API. O [contrato do Feedback](feedback.md#issues-marcas-e-status) descreve marcas, etiquetas e limites de atualização no site, sem prometer refresh automático de uma aba já aberta.

`internal/app/attachments_api.go` lista arquivos diretos da pasta de anexos, mais recentes primeiro, com categoria e notas que os mencionam. Renomeação ajusta referências encontradas nas notas. É uma gestão da pasta de anexos do site, não um catálogo de todos os arquivos em qualquer ponto do cofre.

A lixeira do Blup fica em `<site>/.blup/trash`, fora das notas, com diretório datado e origem para restauração. Restaurar recusa sobrescrever um destino já existente. Excluir definitivamente ou esvaziar remove esses dados. A lixeira de sync tem diretório e finalidade próprios.

`DeleteSite.svelte` permite escolher partes locais e remotas. As pastas locais selecionadas vão à lixeira do sistema por rotinas de Linux, macOS ou Windows; os destinos remotos são apagados por API. `ForgetSite` apenas retira da lista. `NotesOwned` é uma inferência de caminho sob a pasta padrão `Documents/Blup`, não um registro de criação infalível.

## Configurações globais

`SettingsModal.svelte` organiza as abas em grupos na lateral:

- no topo, o cartão da conta do Blup Cloud (e-mail e plano, ou "Entrar no Blup Cloud"), que abre a aba Blup Cloud. Ela vale para todos os cofres e sites: entrar e sair (`CloudAccount.svelte` com `brief`), senha das notas e o uso do plano em barras: armazenamento, IA do mês (`BlupAIUsage`: percentual usado, data em que renova e quanto dá para usar agora pelo teto semanal), sites publicados e computadores, com Ver planos e Gerenciar conta;
- Studio: geral (idioma, autostart, bandeja), aparência, editor, atalhos e terminal;
- Sites: publicação (GitHub e Cloudflare) e senha geral dos posts;
- IA e voz: IA (a IA do Blup Cloud, a voz no computador e o OpenRouter) e narração;
- no rodapé, `Blup Studio v<versão>`, que abre a aba Sobre (versão, novidades, documentação e código).

O link de ajuda é definido em `DOCS` nesse arquivo. A lista de modelos do OpenRouter não traz o Kokoro, que já roda grátis no computador.

Nas escolhas em segmento de `SettingsModal.svelte`, os rótulos de cada botão permanecem numa linha (`white-space: nowrap`). Em Quem traduz, exibido quando a tradução local está instalada, a linha usa quebra flexível: as opções ficam ao lado da explicação enquanto couberem e passam para baixo quando falta largura. Isso altera a disposição visual, mantendo a preferência `translateAI` e as opções No computador e IA na nuvem.

Os atalhos padrão de `src/keys.ts` são (os marcados com cofre só agem num cofre de anotações; a tela cheia só com um canvas ou desenho aberto):

| Ação | Teclas |
| --- | --- |
| Nova nota | `Ctrl+N` |
| Nota do dia (cofre) | `Ctrl+Alt+D` |
| Mapa de notas (cofre) | `Ctrl+Alt+G` |
| Novo canvas (cofre) | `Ctrl+Alt+C` |
| Novo desenho (cofre) | `Ctrl+Alt+W` |
| Novo banco de dados (cofre) | `Ctrl+Alt+B` |
| Tela cheia no canvas e no desenho | `F11` |
| Busca de notas | `Ctrl+Shift+F` |
| Painel lateral | `Ctrl+Alt+S` |
| Prévia | `Ctrl+Alt+P` |
| Terminal | Ctrl com acento grave |
| Configurações | `Ctrl+,` |

`comboOf` monta a combinação pela tecla física das letras e números (`e.code`), então o layout do teclado e o Shift não mudam a letra. As escolhas do usuário ficam em `keys` nas preferências; ausência da ação usa o padrão e valor vazio desliga.

O usuário pode alterar ou desativar cada um. Ao escolher uma combinação já usada por outra ação, a outra perde aquela combinação. O atalho global de mostrar/ocultar é outro mecanismo, com padrão `Alt+Space`.

A aba Atalhos também mostra a lista fixa de comandos do editor, `k.editorKeys`: `Ctrl+S` salvar, `Ctrl+Z` e `Ctrl+Shift+Z` desfazer e refazer, `Ctrl+B` negrito, `Ctrl+I` itálico, `Ctrl+K` link, Tab aceita a sugestão, `@` menciona nota ou data, `/` abre as ações e os quatro atalhos de recolher títulos. Em Hyprland sem configuração automática, apresenta as instruções do binding e um botão para copiar a linha.

Também em Configurações: a aba Narração tem uma voz padrão para cada idioma (Português, English, Español), para todos os sites; Geral tem a cópia diária das notas (`BackupDir` nas preferências: um `.zip` por dia em `internal/app/backup.go`, feito um minuto depois de abrir o site, guardando 7 cópias, nunca dentro das notas); Sobre procura versão nova (`CheckUpdate`, uma vez por dia ou no botão) e o rodapé mostra um ponto azul quando há.

## Idiomas e tradução

Um site é escrito no idioma de `lang` (`pt`, `en` ou `es`) e pode ser publicado nos outros, listados em `translations` no `blup.toml`. A tradução de `Pasta/Nota.md` fica numa pasta com o código do idioma, em minúsculas, com o nome traduzido (`Pasta/es/Nota traducida.md`). Original e traduções compartilham a propriedade `translationKey` (posta na original na primeira tradução, com o caminho dela), que o Hugo usa para ligar as páginas; uma tradução com o mesmo nome do arquivo original também vale. Quando uma tradução muda a configuração (um idioma ou uma pasta de idioma nova, que o Hugo só lê ao iniciar), a prévia reinicia sozinha e volta na página da nota aberta. O `hugo.toml` gerado (`internal/scaffold`) declara os idiomas e monta cada pasta de idioma no lugar da pasta mãe só naquele idioma (`sites.matrix`), então a página traduzida é a mesma página no outro idioma: o Hugo liga as duas, o tema mostra o seletor de idioma (`site/langs.html`) e `hreflang`, e um link para uma nota sem tradução cai na original.

- Painel Idiomas na nota (`NoteLangs.svelte`): a original e cada idioma, com Abrir, Traduzir, Traduzir de novo e revisão. Aparece em toda nota de um site, inclusive nas notas de seção (`_index.md` com `blup:`, como Podcast e Feedback). O bloco ocupa as duas colunas das propriedades por `grid-column: 1 / -1`; nomes e ações usam quebra flexível, colocando botões abaixo do nome quando falta espaço. A tradução guarda `translation_hash`, a impressão da original (versão do tradutor, título, descrição, texto e textos dos exemplos, quando presentes); quando essa impressão muda, uma tradução com hash salvo aparece desatualizada.
- Tela Idiomas do site (`LangsPanel.svelte`): quantas notas estão traduzidas, desatualizadas e faltando em cada idioma, e Traduzir o que falta de uma vez.
- Quem traduz (`internal/translate`): a IA do plano no Blup Cloud (`/api/ai/translate`, que monta o pedido no servidor e desconta o custo real) quando o plano inclui IA, senão a chave do OpenRouter (Claude Haiku 5.5). A nota vai em partes de até 12 KB, cortadas em linhas em branco fora de blocos de código. Código, links, imagens e o tipo dos callouts não mudam.
- A tradução também traduz o que o site mostra fora do texto: na nota inicial, `site_description`, `button_text` e os rótulos do menu (`Nome | destino`: o nome; um item só com o nome de uma pasta usa o título traduzido da seção), e em qualquer nota `linkTitle` (o nome curto dos cartões e do menu), `category` e `tags` (`TextProps`). Quando o que a tradução cobre muda, `translateVersion` entra na impressão da original e as traduções antigas aparecem desatualizadas. A nota de uma seção (`_index.md`) mantém o nome na pasta do idioma, e é ela que dá o nome traduzido da seção, do blog, do roadmap, da documentação ou do changelog.
- Na nota da seção Feedback (Bento), `exampleTexts` e `walkExamples`, em `internal/app/translate.go`, percorrem `examples[].title`, `body` e `replies[].text`, para a tradução local e remota. Votos, tipo, status, autores e `days` ficam. Esses textos entram na impressão só das notas com `examples`, sem invalidar as outras por essa ampliação; mudar somente votos ou status não altera o hash. O cabeçalho do quadro inclui `site/ai-badge.html`, com selo quando há marca de IA e original ligada. Interface e categorias usam o i18n do tema; os exemplos dependem da tradução da nota; pedidos e comentários reais mantêm seus textos e idioma de origem, sem tradução automática. Veja [Feedback](feedback.md#idiomas-e-tradução).
- Os perfis dos autores também são traduzidos (o cargo, `role`, e a bio), na pasta do idioma dentro da pasta de autores e com o mesmo nome (o nome é o autor). Cada idioma do site lê o perfil no seu idioma; sem tradução, o original. Como a pasta de autores, essas traduções nunca viram páginas.
- Comentários, reações e votos: todas as versões de uma página usam a conversa da original (a mesma issue do GitHub, `bento/feedback-key.html`), então votos e reações somam os leitores de todos os idiomas. Um comentário guarda o idioma em que foi escrito (uma marca invisível no texto) e, visto em outra versão, mostra "comentado em English".
- Uma tradução feita pela IA leva `ai_translated: true`, e o site mostra junto do título um selo "Traduzido com IA" que abre um balão com o idioma original e o link para ele (`site/ai-badge.html`, nos três temas). No painel Idiomas, "Marcar como revisada" tira a marca e o selo.
- A narração consulta `NoteOrigin(site, rel)`: a pasta imediata `en`, `es` ou `pt` determina o idioma de uma tradução; fora dela usa o idioma principal. `Studio.Narrate` lê `Prefs.Voices[lang]`, e `app.Narrate` resolve a escolha por `voiceFor` naquele idioma, com fallback local ou remoto. O arquivo novo de uma tradução leva o idioma no nome (`Nota (en).mp3`), salvo quando uma referência existente a MP3 é reutilizada. As regras estão em [narração e voz](narracao-e-voz.md#idioma-da-nota-e-escolha-de-voz).
- Tradução no computador (`internal/localtr`, `localtranslate.ts`, `public/bergamot`): o motor e os modelos do Firefox Translations (Bergamot 4.0 e modelos 3.0 de inglês com português e espanhol), baixados da Mozilla sob demanda (cerca de 100 MB, 145 MB no disco, cada arquivo conferido por SHA-256 antes e depois de descompactar) e servidos à janela em `/local-translate/`. Roda num worker; português e espanhol passam pelo inglês. Cada linha do Markdown vira HTML que o motor preserva: código, endereços, alvos de wikilinks, imagens e matemática vão em marcações com o próprio texto (só para contexto) e voltam exatamente como eram, com os espaços e a caixa baixa do original. A escolha de quem traduz fica em `translateAI` (vazio: no computador quando instalado).

## Busca, exportar e desempenho

- A busca da lateral procura no nome e no texto das notas (`SearchNotes`, sem diferenciar maiúsculas nem acentos), mostrando as linhas encontradas.
- O botão direito numa nota exporta como HTML (imagens embutidas, `export.ts`) ou abre a impressão do sistema, que também salva em PDF.
- Leveza: terminal, mapa, canvas, desenho, banco de dados e os painéis do site carregam só quando abertos (`lazy.ts`); o mapa de notas não desenha parado; o mapa e o "Mencionada em" releem só as notas que mudaram (cache por data e tamanho em `internal/app/graph.go`); mudanças só de permissão nos arquivos não recarregam nada. No Linux o motor de mídia do WebKit (GStreamer) liga na abertura (`audio.ts`), porque a primeira vez que um som toca ele segura a janela por meio segundo.

## Terminal e agentes locais

`terminal.go` usa `go-pty` e `TerminalPanel.svelte` usa xterm.js com addon de ajuste e abas para vários terminais. Inicia shell na pasta das notas: PowerShell em Windows, `$SHELL -l` nos demais ou `/bin/bash -l`. Define ambiente de terminal de 256 cores. Dados binários de saída são transmitidos em base64.

O seletor reconhece Claude Code (`claude`), Codex (`codex`), OpenCode (`opencode`), Gemini CLI (`gemini`) e Antigravity (`agy`), verificando o PATH. Não instala esses programas. Ao iniciar um agente, dispara o comando dentro do shell; sair do agente deixa o shell aberto.

`OpenTerminal` chama `WriteAgentGuides` ao abrir qualquer terminal, inclusive um shell sem agente. A função cria guias nas notas conforme suas regras de preservação. O scaffold exclui os três nomes de guia na raiz para que não virem páginas.

Antes de iniciar agente, `TakeSnapshot` copia arquivos elegíveis para `<site>/.blup/snapshots`; guarda os cinco snapshots mais recentes, limita cada arquivo a 20 MiB e o conjunto a 300 MiB e ignora ocultos. `AIChanges` compara com o snapshot e `UndoAI` restaura o estado capturado, levando criações à lixeira segundo a rotina de undo. Os limites significam que não é backup integral garantido de todos os anexos grandes.

O painel oferece revisão das mudanças da IA por aba: consulta `AIChanges` com o ID do terminal e permite desfazer por `UndoAI`. O snapshot depende da escolha de um agente; abrir somente o shell não cria essa captura.

O watcher fsnotify acompanha subpastas não ocultas, acrescenta novas pastas e agrupa eventos em janelas de 500 ms. Terminais continuam vivos quando seu painel é ocultado. Trocar site fecha os terminais anteriores; shutdown também os fecha.

## Bandeja, instância única e autostart

`main.go` usa instância única com ID `dev.blup.studio`. Nova abertura mostra e foca a janela existente. Se a segunda invocação tem `--toggle`, alterna mostrar e ocultar/minimizar. `--hidden` só inicia oculto quando a preferência de bandeja está ativa.

`desktop.go` cria bandeja com abrir e sair. Quando habilitada, fechar ou minimizar a janela a oculta e mantém os terminais em execução. Sair pelo menu encerra a aplicação. Sem bandeja, aplica o comportamento de janela correspondente. O atalho global fora do Hyprland usa `GlobalShortcut` do Wails.

Autostart usa a API Wails com identificador `blup-studio` e argumento `--hidden`. Em Linux o Studio também reconhece `~/.config/autostart/blup-studio.desktop`, respeitando `XDG_CONFIG_HOME`, e pode corrigir o caminho do executável após mudança de instalação. A detecção de autostart não fica no JSON de preferências.

## Hyprland e Omarchy

`hyprland.go` detecta Linux com `HYPRLAND_INSTANCE_SIGNATURE`. Em Hyprland, o atalho chama o próprio executável com `--toggle`, para a instância única receber a solicitação.

A detecção de Omarchy procura a palavra `omarchy` em `~/.config/hypr/hyprland.lua` ou `.conf`. Se existir a configuração Lua, usa `bindings.lua`; senão `bindings.conf`. No Omarchy escreve linha marcada `Blup Studio`, substitui a linha anterior e executa `hyprctl reload`. A sintaxe Lua usa `o.bind`; a configuração tradicional usa `bindd`.

Fora do Omarchy, salva a preferência e mostra linha em sintaxe Hyprland para o usuário colar. Não escreve automaticamente uma configuração arbitrária de outro ambiente. No início, `fixBinding` pode reparar a linha marcada no Omarchy se estiver ausente ou apontar para executável antigo, limitado ao executável Studio reconhecido.

Essas descrições registram efeitos do produto. A criação desta spec não executa essas rotinas nem altera configurações do desktop.

Os comandos de compilação, dependências nativas e pacotes AppImage, nfpm, DMG, NSIS e MSIX estão em [testes e build](testes-e-build.md).

### Renomear, mover e copiar bancos

Renomear ou mover um `.table` leva a pasta de registros junto (com o novo nome) e corrige os links para os registros (`followDatabase` em `internal/app/database.go`); renomear só a pasta de registros grava o novo nome no `.table`. `DatabaseOf` acha o banco pelo `.table` de mesmo nome ou por qualquer `.table` ao lado que nomeie a pasta. Não há "Fazer uma cópia" para `.table` (as duas cópias dividiriam a mesma pasta). Renomear uma opção de escolha renomeia o valor nos registros; num cofre, o número 0 fica gravado (só nos formulários de site 0 significa "sem valor").

## Segurança do conteúdo

- O site: o Hugo roda com o goldmark sem `unsafe`, então HTML escrito na nota (`<script>`, `<iframe>`, `onerror=`) sai como `<!-- raw HTML omitted -->`, e um link `javascript:` vira `#ZgotmplZ` pelo escape do template.
- O Studio: o editor trata a nota como texto; o que ele desenha entra por `textContent`. Os cartões do canvas escapam o texto antes de virar HTML.
- `OpenURL` só abre `http`, `https`, `mailto` e `tel`; outro endereço (`file:///…`, um esquema que abre um programa) é ignorado.
- Caminhos: `notesPath`, `notePath` e `ImageData` exigem que o caminho, seguido pelos links simbólicos (`app.Within`), fique dentro da pasta das notas. Um link para nada também é recusado, porque escreveria onde ele aponta.
- A prévia roda num iframe com `sandbox` (scripts e a própria origem, sem navegar a janela do Studio), e as mensagens dela só valem vindas da origem da prévia.
