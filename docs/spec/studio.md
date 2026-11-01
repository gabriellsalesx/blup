# Blup Studio

Fontes: [entrada desktop](../../studio/main.go), [serviço principal](../../studio/studio.go), [aplicação Svelte](../../studio/frontend/src/App.svelte), [workspace](../../studio/frontend/src/Workspace.svelte), `studio/frontend/package.json` e os arquivos de serviço em `studio/`.

## Arquitetura

O Studio é um aplicativo Wails v3 com backend Go e frontend Svelte 5, TypeScript e Vite. `studio/go.mod` declara Wails `v3.0.0` e usa o módulo principal por `replace => ../`. Não mantém outra implementação completa da CLI: chama os fluxos e APIs de `internal/app` para criar, configurar, publicar, sincronizar e narrar.

`studio/main.go` embute `all:frontend/dist` no binário e registra uma instância de `Studio` como serviço Wails. O runtime serve os assets do frontend e faz a ponte de chamadas. A janela inicial mede 1360 por 860, com mínimo 900 por 560. O serviço mantém um site e um processo de prévia por vez.

O frontend usa runes Svelte como `$state`, `$derived`, `$effect` e `$props`. A entrada `src/main.ts` carrega idioma, tema e mecanismo de undo antes de montar `App`. As traduções da janela ficam em `src/i18n.ts`; as mensagens de trabalho Go continuam em `internal/i18n`.

O idioma inicial segue `config.Global.Lang`, senão o prefixo `pt` de `LANG`, senão inglês. `SetLanguage`, em `studio/about.go`, aceita `pt` ou `en`, converte outros valores para `en` e salva o idioma global. As Configurações recarregam a interface após a troca; as boas-vindas aplicam a tradução sem recarregar. Os rótulos da bandeja são montados no início em `studio/main.go` e só mudam ao reiniciar. Isso não troca o idioma dos sites existentes.

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
| `authors.go`, `attachments.go` | CRUD de perfis e gestão dos anexos |
| `publish.go` | Contas, destino, publicação, sync e domínio |
| `cloud.go` | Descoberta e recuperação GitHub, senha geral entre sites |
| `narration.go` | Modelos, vozes, amostras, instalação local, chave remota e geração |
| `media.go` | Detecção local de reprodução MP3 por `canPlayMP3` |
| `terminal.go` | Agentes instalados, PTY, entrada/saída, tamanho, snapshots e undo |
| `layout.go` | Organização da home e preferências |
| `desktop.go`, `hyprland.go`, `about.go` | Bandeja, autostart, atalhos, alteração de idioma por `SetLanguage` e versão |

| Evento | Payload e finalidade |
| --- | --- |
| `progress` | `{kind, text}`; etapas, avisos e resultados de publicação/sync |
| `previewChanged` | Notifica rebuild Hugo para recarregar a página da prévia |
| `notesChanged` | Caminhos relativos separados por quebra de linha; alterações externas |
| `term` | `{id, data, exit}`; bytes do terminal em base64 ou término |
| `narrate` | `{note, done, total}`; progresso de narração |
| `voiceInstall` | `{done, total, step}`; download do motor local |

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

## Canvas e desenhos (cofres de anotações)

Num cofre de anotações, a árvore lista também canvases (`.canvas`) e desenhos (`.draw`), com ícone próprio, e a lateral e o menu de pasta criam os dois na hora, sem pedir nome (`CreateCanvas`, `CreateDrawing`): "Desenho 2026-11-01 14.05.33" na pasta Desenhos (ou "Canvas …" na pasta Canvas), ou na pasta cujo menu foi usado, e já abre. Os dois são do próprio Studio, feitos em Svelte, sem biblioteca de terceiros; não precisam abrir no Obsidian (só as notas em Markdown precisam). `boards.go` lê e grava (`ReadBoard`, `SaveBoard`, só essas extensões), guarda imagens coladas na pasta de anexos (`SaveBoardImage`) e entrega imagens do cofre como `data:` (`BoardFile`). Renomear e copiar mantêm a extensão (`fileExt`). A sincronização e a lixeira tratam esses arquivos como qualquer outro.

`CanvasView.svelte` (lógica em `canvas.ts`) grava no formato aberto JSON Canvas: cartões de texto em Markdown, notas do cofre (`CanvasFile.svelte` mostra o texto, ou a imagem), links da web e grupos, ligados por setas com texto e cor. Mover a tela arrastando, zoom com Ctrl e a roda, dois cliques criam um cartão, arrastar do ponto lateral cria uma seta, notas arrastadas da árvore viram cartões, e há desfazer e refazer.

`DrawingView.svelte` (lógica em `draw.ts`) é o editor de desenho: retângulo, losango, elipse, seta, linha, desenho livre, texto (solto ou dentro da forma), borracha e imagens, com cor, fundo, espessura, tracejado e tamanho de texto, seleção por caixa, redimensionar, duplicar, ordem, atalhos de teclado, desfazer e exportar PNG para os anexos. O arquivo é JSON com um elemento por linha. As duas telas têm modo tela cheia (botão nos controles ou F; Esc sai), em que ocupam a janela inteira, sem lateral, barra ou menu. As duas telas salvam sozinhas e recarregam quando o arquivo muda fora do editor (evento `notesChanged`), se não houver mudança própria pendente.

## Prévia

`StartPreview` encontra ou baixa Hugo, gera scaffold e inicia servidor restrito a `127.0.0.1`, em memória, com rascunhos e `HUGO_BLUP_MODE=preview`. Desativa live reload nativo e observa log `Total in` para emitir `previewChanged`. Não executa o preparo Pagefind da CLI.

`Preview.svelte` usa dois iframes. O próximo carrega oculto, recebe a posição de scroll e assume o lugar quando sinaliza pronto; há fallback de quatro segundos. A comunicação usa `postMessage` e scripts que o tema injeta somente em prévia. O parâmetro `_blup` identifica os frames. Não é scroll sincronizado por linha de Markdown: conserva a posição da página e segue relações nota/URL.

`PreviewPage` e `NoteAt` leem `blup-map.json` no servidor local. Editar outra nota leva à sua página; navegar em link de página com nota abre essa nota no editor. Taxonomias sem arquivo podem continuar sem correspondente.

Em Linux, `canPlayMP3` procura `libgstaudioparsers.so` nos diretórios GStreamer habituais e em `GST_PLUGIN_PATH`. Se faltar, o servidor de prévia recebe `HUGO_BLUP_NO_MP3=1` e o tema omite o player nessa prévia. É uma detecção por arquivo, não um teste real de decodificação de todos os formatos.

## Painéis do site

| Tela | Operações atuais |
| --- | --- |
| Aparência, `LookPanel.svelte` | Nome, descrição, autor padrão, logo, favicon, menu, botão e redes |
| Autores, `AuthorsPanel.svelte` | Lista, perfil, foto/avatar, cargo, redes, bio, criação, renomeação e remoção |
| Anexos, `AttachmentsPanel.svelte` | Lista por tipo, tamanho e usos, adição, renomeação, exclusão e acesso às notas que usam |
| Seções, `SectionsPanel.svelte` | Somente Bento; acrescenta exemplos e remove seções para lixeira |
| Comentários, `CommentsPanel.svelte` | Ativa/desativa feedback, prepara repositório e orienta instalação do App |
| Cookies, `CookiesPanel.svelte` | Ferramentas, IDs, URLs próprias, banner e política |
| Senha, `PasswordPanel.svelte` | Senha geral específica do site e estado da global |
| Publicação, `PublishPanel.svelte` | Conectar contas, destino, publicar, acompanhar log, domínio e sync |
| Lixeira, `TrashPanel.svelte` | Restaurar, excluir item e esvaziar |

Perfis usam `internal/app/authors_api.go`, com avatares SVG embutidos em `internal/app/avatars`. Renomear autor ajusta as propriedades de posts conforme as rotinas implementadas. A pasta de perfis é identificada por `blup: authors`.

`internal/app/attachments_api.go` lista arquivos diretos da pasta de anexos, mais recentes primeiro, com categoria e notas que os mencionam. Renomeação ajusta referências encontradas nas notas. É uma gestão da pasta de anexos do site, não um catálogo de todos os arquivos em qualquer ponto do cofre.

A lixeira do Blup fica em `<site>/.blup/trash`, fora das notas, com diretório datado e origem para restauração. Restaurar recusa sobrescrever um destino já existente. Excluir definitivamente ou esvaziar remove esses dados. A lixeira de sync tem diretório e finalidade próprios.

`DeleteSite.svelte` permite escolher partes locais e remotas. As pastas locais selecionadas vão à lixeira do sistema por rotinas de Linux, macOS ou Windows; os destinos remotos são apagados por API. `ForgetSite` apenas retira da lista. `NotesOwned` é uma inferência de caminho sob a pasta padrão `Documents/Blup`, não um registro de criação infalível.

## Configurações globais

`SettingsModal.svelte` contém abas geral, aparência, editor, contas, senha geral, IA, narração, terminal e atalhos. Controla idioma, preferências visuais e de editor, prévia ao abrir, contas, senha entre sites, OpenRouter, motor local, escolha/amostra de voz por idioma, tamanho do terminal e agentes encontrados. O link de ajuda é definido em `DOCS` nesse arquivo.

Os atalhos padrão de `src/keys.ts` são:

| Ação | Teclas |
| --- | --- |
| Nova nota | `Ctrl+N` |
| Busca de notas | `Ctrl+Shift+F` |
| Painel lateral | `Ctrl+Alt+S` |
| Prévia | `Ctrl+Alt+P` |
| Terminal | Ctrl com acento grave |
| Configurações | `Ctrl+,` |

O usuário pode alterar ou desativar cada um. Ao escolher uma combinação já usada por outra ação, a outra perde aquela combinação. O atalho global de mostrar/ocultar é outro mecanismo, com padrão `Alt+Space`.

A aba Atalhos também mostra a lista fixa de comandos do editor, `k.editorKeys`. Em Hyprland sem configuração automática, apresenta as instruções do binding e um botão para copiar a linha.

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
