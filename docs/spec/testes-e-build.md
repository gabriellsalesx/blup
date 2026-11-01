# Testes, compilação e distribuição

Fontes: [módulo principal](../../go.mod), [módulo Studio](../../studio/go.mod), [GoReleaser](../../.goreleaser.yaml), [workflow](../../.github/workflows/release.yml), [tarefas Studio](../../studio/Taskfile.yml), `studio/build/` e `studio/frontend/package.json`.

## Dois módulos Go

O módulo principal é `github.com/gabriellsalesx/blup`; o Studio é `github.com/gabriellsalesx/blup/studio`. Ambos declaram Go `1.27.3`. O `replace` do Studio aponta para a raiz local, portanto o build depende do checkout conjunto. `go test ./...` na raiz não atravessa automaticamente o módulo aninhado do Studio.

O tema é incorporado por `embed.go`. O frontend do Studio é incorporado por `studio/main.go`, exigindo `studio/frontend/dist` antes de compilar esse módulo. Alterações nos assets embutidos exigem novo build para distribuição.

## Testes do módulo principal

Na raiz:

```sh
go test ./...
go build -o bin/blup ./cmd/blup
```

Para reproduzir o perfil sem CGO e a injeção de versão usados na release CLI, com `0.1.0` como exemplo:

```sh
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/gabriellsalesx/blup/internal/app.Version=0.1.0" -o bin/blup ./cmd/blup
```

Sem `-X ...app.Version=...`, o binário mantém `dev`, e a oferta automática de atualização fica desativada. O comando de build cria o binário; sua execução usa o estado real do usuário, pode escrever configurações e pode baixar Hugo/Pagefind. Uma validação de documentação não precisa iniciar os fluxos interativos.

| Área | Testes presentes e finalidade |
| --- | --- |
| `internal/config` | Modelos aceitos, gravação/leitura da versão e rejeição de versão futura |
| `internal/i18n` | Correspondência dos catálogos e existência das chaves usadas |
| `internal/obsidian` | Propriedades e preservação de notas |
| `internal/wikilinks` | Resolução, conversão, âncoras e backups |
| `internal/notesync` | Comparação de versões, conflitos, exclusões, falhas e servidor GitHub falso |
| `internal/deploy/github` | Repositórios, publicação, Pages, domínios e autenticação com HTTP simulado |
| `internal/deploy/cfpages` | Hashes, upload, erros e domínios com API simulada |
| `internal/lock` | Criptografia e pós-processamento |
| `internal/pwa` | Ícones e worker |
| `internal/narrate` | Limpeza, partes, HTTP e MP3 |
| `internal/kokoro` | Tokens, fonemas, estilo, pausas e instalação simulada |
| `internal/update` | Versões, download, checksums, troca e rollback |
| `internal/app` | Criação, exemplos, configurações, domínios, feedback, arquivos, autores, anexos, snapshots e recuperação |
| `internal/scaffold` | Montagem e integração real com Hugo |

`internal/scaffold/scaffold_test.go` chama `hugo.Find`; se não houver Hugo compatível, pula a integração. `TestBuild` exercita Mochi e Bento com o cofre de `testdata/vault`, valida links, callouts, MathML, WebP, título, imagens sociais, proteção e notas escondidas. O teste de Hugo sem modo Blup verifica que não publica o corpo protegido em claro. Esse arquivo não inclui Nori no loop principal.

### Feedback, tradução e changelog

Os recursos da revisão de 01/11/2026 têm os seguintes testes de regressão no código:

| Teste e arquivo | O que verifica |
| --- | --- |
| `TestSections`, `internal/obsidian/sections_test.go` | Identifica `blup: feedback` mesmo quando a pasta se chama de outra forma |
| `TestFeedbackBoard`, `internal/scaffold/scaffold_test.go` | HTML do quadro, categorias, filtro próprio, formulário, exemplos e detalhes, inclusive com comentários desligados |
| `TestBoardPosts`, `internal/app/feedback_board_test.go` | Filtra as marcas de pedido, lê idioma, votos, comentários e status, com fallback de tipo e de issue fechada |
| `TestTranslateFeedbackExamples`, `internal/app/translate_test.go` | Traduz título, corpo e comentários de exemplo, preserva dados não textuais e marca a tradução como desatualizada quando o texto do exemplo muda |
| `TestTranslateHomeAndSection`, `internal/app/translate_test.go` | Traduz rótulos, `linkTitle` e perfil do autor, mantendo os nomes de arquivo de índice e autor |
| `TestBuildTranslated`, `internal/scaffold/scaffold_test.go` | Montagem dos idiomas, perfil traduzido como recurso e mesma issue para original e tradução |
| `TestChangelogMap`, `internal/scaffold/scaffold_test.go` | Calendário antes da lista, cores, prioridade num dia com várias versões, link e exclusão de versões antigas do mapa |
| `TestVoiceFor`, `internal/app/narration_api_test.go` | Escolhas locais/remotas e fallback quando uma voz não fala o idioma solicitado |
| `TestSetFeedbackStatusAttacks` e `TestBoardStatusRejectsUnmanagedLabel`, `internal/app/feedback_board_test.go` | Recusa número, status e issue inválidos; altera pedido antigo sem substituir etiquetas de terceiros; ignora status desconhecido no backend |
| `TestFeedbackTranslationRejectsMalformedTextsBeforeWriting`, `internal/app/translate_test.go` | Recusa lista de textos de exemplo inválida antes de gravar arquivos |
| `TestPlaceSearchAcrossDisks`, `internal/app/preview_test.go` | Instala o índice da prévia com cópia quando rename devolve EXDEV |
| `TestAdoptPreview`, `studio/preview_test.go` | Descarta processo duplicado ou de site anterior sem substituir a prévia já adotada |
| `TestFeedbackBrowserAttacks`, `internal/app/feedback_security_test.go` | Exercita o script em Chromium headless contra persistência de autorização, classes, HTML ativo e metadados forjados; pula se Chromium não estiver instalado |
| `TestFeedbackExamplesEscapeAndClasses`, `internal/app/feedback_security_test.go` | Confere escape dos textos de exemplo e fallback de categoria/status inválidos no Hugo |

Para rodar os pacotes internos e o pacote desktop, separadamente:

```sh
go test ./internal/...
(cd studio && go test .)
```

O teste de Chromium cobre as entradas de segurança descritas na tabela, sem uma suíte funcional completa das interações. Cartão clicável, votação dos exemplos, Carregar mais, menu de status, quebras de layout em `NoteLangs`/`SettingsModal` e preservação visual após reinício ainda precisam de verificação de interface quando o código dessas interações muda. A checagem de documentação verifica links locais, ausência de travessões e ausência de credenciais reais, sem abrir ou reiniciar o Studio.

O workflow instala Hugo `0.169.0` antes de `go test ./...`, para não depender desse skip. Os testes HTTP dos provedores usam respostas simuladas e não provam disponibilidade das contas ou APIs reais em produção. O teste de modelos em `internal/config/config_test.go` também ainda não inclui Nori entre os casos positivos, embora `LoadSite` já o aceite.

Variáveis de apoio encontradas nos testes:

| Variável | Uso |
| --- | --- |
| `BLUP_KEEP` | Preserva saída dos testes scaffold no diretório informado |
| `BLUP_STARTER_OUT` | Escreve sites de exemplo do teste para inspeção |
| `BLUP_KOKORO_DIR` | Habilita teste de fala com instalação local real |

Sem `BLUP_KOKORO_DIR` apontando para instalação reconhecida, `internal/kokoro.TestSpeak` é pulado. O teste homônimo `internal/narrate.TestSpeak` usa servidor HTTP simulado e não depende dessa variável. Criar instalação real pode exigir rede e bibliotecas; isso é distinto dos testes de transformação de texto e instalação simulada.

## Frontend e Studio

O frontend declara Svelte 5, Vite 8, TypeScript, CodeMirror 6, runtime Wails 3, xterm e Hunspell WASM, com `package-lock.json` versionado. O projeto tem scripts de build e análise Svelte, mas não declara um script de suíte automatizada de navegador.

Na pasta `studio/frontend`, este fluxo manual usa `npm ci` com o lockfile versionado:

```sh
npm ci
npm run check
npm run build
```

`check` executa `svelte-check --tsconfig ./tsconfig.json`. `build` executa Vite em produção; `build:dev` usa modo development sem minificação. O servidor de desenvolvimento usa `127.0.0.1:9245`, ou `WAILS_VITE_PORT`, com porta estrita.

As tarefas Wails usam `npm install`, não `npm ci`, quando `PACKAGE_MANAGER=npm`, que é o padrão. A variável também aceita `bun`, `pnpm` e `yarn`, acionando as rotinas de instalação e execução correspondentes em `studio/build/Taskfile.yml`.

Com `frontend/dist` existente, na pasta `studio`:

```sh
go test ./...
go build -o bin/blup-studio .
```

O `go build` simples acima não seleciona a tag `production`. Para reproduzir as flags do build nativo de produção do Studio, use, ainda em `studio`:

```sh
go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" -o bin/blup-studio .
```

O módulo contém testes específicos de Hyprland em `studio/hyprland_test.go`. Compilar o Studio desktop em Linux exige toolchain nativa e bibliotecas da pilha gráfica; não basta copiar o binário da CLI.

As tarefas do projeto usam o CLI `wails3`. A dependência Wails do módulo está fixada em `v3.1.0`. A instalação do comando correspondente pode ser feita com:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
```

Dentro de `studio`, os fluxos previstos são:

```sh
wails3 build
wails3 task dev
wails3 task package
```

`Taskfile.yml` despacha `build`, `run` e `package` para a plataforma. `dev` chama `wails3 dev` com `build/config.yml`. As tarefas comuns instalam dependências frontend, geram bindings e ícones e compilam frontend. Algumas executam `go mod tidy`; são operações que podem alterar arquivos e gerar assets, não simples verificações somente de leitura.

Para regenerar a ponte explicitamente, a tarefa `common:generate:bindings` usa `wails3 generate bindings` com flags, limpeza, TypeScript e interfaces. Mudanças de métodos Go devem ser conferidas também nos imports TypeScript e tipos serializados.

Outras tarefas presentes: `common:generate:icons` gera ícones de plataforma; `common:dev:frontend` inicia Vite; `common:update:build-assets` atualiza os assets de empacotamento a partir de `build/config.yml`; `install:msix:tools` encaminha à preparação das ferramentas MSIX Windows. No macOS, `codesign:adhoc` e `codesign:skip` são tarefas internas de assinatura ad hoc ou de sua omissão no cross-build.

As variáveis de build incluem `DEV=true`, que escolhe perfil de desenvolvimento; `OBFUSCATED=true`, que usa `garble` e tag `wails_obfuscated`; e `EXTRA_TAGS`, lista adicional de tags separadas por vírgula. `GARBLE_ARGS` permite argumentos do garble. `GOOS`, `ARCH`, `OUTPUT` e `PACKAGE_MANAGER` também participam dos caminhos de tarefas conforme a plataforma. A presença de uma variável na tarefa não instala suas ferramentas dependentes.

## Dependências Linux

A implementação e os pacotes usam GTK4 e WebKitGTK 6.0, padrão Wails 3 deste projeto. A compilação nativa requer compilador C, pkg-config e headers/bibliotecas de desenvolvimento correspondentes da distribuição. O runtime do WebKitGTK usa GStreamer para mídia.

A dependência Wails `v3.1.0` também oferece a tag `gtk3`: seus arquivos `pkg/application/linux_cgo_gtk3.go` e `linux_cgo.go` selecionam respectivamente `gtk+-3.0 webkit2gtk-4.1 gdk-3.0` ou `gtk4 webkitgtk-6.0` pelo pkg-config. `EXTRA_TAGS=gtk3` altera a pilha do build; os nomes de dependências no nfpm deste repositório continuam fixos para GTK4/WebKitGTK 6.0 e não se ajustam automaticamente à tag.

Os nomes de dependências declarados pelo nfpm são:

| Formato | GTK e WebKit | MP3 |
| --- | --- | --- |
| DEB | `libgtk-4-1`, `libwebkitgtk-6.0-4` | `gstreamer1.0-plugins-good` |
| RPM | `gtk4`, `webkitgtk6.0` | `gstreamer1-plugins-good` |
| Arch | `gtk4`, `webkitgtk-6.0` | `gst-plugins-good` |

Esses são nomes de runtime dos arquivos de pacote, não uma lista universal de pacotes de desenvolvimento. `studio/media.go` procura o plugin de parsers de áudio e pode desativar o player na prévia quando ausente. Isso não substitui instalar os componentes de mídia da distribuição.

`studio/build/linux/Taskfile.yml` usa build nativo em Linux quando há compilador C e arquitetura compatível; caso contrário tenta Docker com imagem `wails-cross`. Build desktop nativo define `CGO_ENABLED=1`, `GOOS=linux` e arquitetura escolhida. Release usa tag `production`, `-trimpath`, `-buildvcs=false` e remoção de símbolos; desenvolvimento usa flags distintas.

## Empacotamento Studio

| Plataforma | Tarefas e arquivos |
| --- | --- |
| Linux | `studio/build/linux/Taskfile.yml`, nfpm, geração de `.desktop`, AppImage, DEB, RPM, pacote Arch e assinatura DEB/RPM |
| macOS | `studio/build/darwin/Taskfile.yml`, `.app`, build universal, DMG, assinatura e notarização |
| Windows | `studio/build/windows/Taskfile.yml`, recursos `.syso`, NSIS, MSIX e assinatura |

O pacote nfpm em `studio/build/linux/nfpm/nfpm.yaml` identifica `blup-studio` com versão fixa `0.1.0`, sem derivação automática da tag Git, e usa `GIT_COMMITTER_NAME` e `GIT_COMMITTER_EMAIL` no campo `maintainer`. Instala o executável em `/usr/local/bin/blup-studio`, ícone em `/usr/share/icons/hicolor/128x128/apps/` e desktop entry em `/usr/share/applications/`. Seu hook ativo é `postinstall`; hooks adicionais existem como arquivos ou exemplos comentados e não devem ser presumidos ativos.

A tarefa Linux `package` chama AppImage, DEB, RPM e Arch. A criação de AppImage usada pelo Taskfile chama `wails3 generate appimage`. Existe também `studio/build/linux/appimage/build.sh`, script auxiliar com linuxdeploy, mas sua existência não significa que todas as tarefas usem esse script diretamente.

As tarefas Linux `sign:deb`, `sign:rpm` e `sign:packages` chamam `wails3 tool sign`; podem usar a configuração de assinatura Wails ou `PGP_KEY`, e a tarefa DEB também passa `SIGN_ROLE` quando definido. A tarefa conjunta assina DEB e RPM, não todos os formatos Linux.

As tarefas macOS e Windows incluem arquivos e opções de assinatura específicos. Ter esses templates não demonstra que certificados, chaves ou ferramentas de assinatura estejam instalados. `build/config.yml` informa produto, identificador e versão `0.1.0`. Já `Studio.Version()` retorna `app.Version`: nenhuma tarefa atual do Studio injeta esse símbolo com `-X`, então o binário produzido por essas tarefas reporta `dev` pela API, inclusive quando empacotado. Os metadados de pacote e a versão retornada pela aplicação são fontes distintas; injetar a versão exigiria um build customizado.

Há tarefas adicionais `build:server`, `run:server`, `build:docker`, `run:docker` e `setup:docker`. O modo server usa tags `server` e `production` e pode construir sem GUI nativa. Isso é uma capacidade de build Wails presente no projeto, sem substituir as integrações desktop ou comprovar um serviço hospedado de Studio em produção. Também há assets/tarefas iOS de origem Wails; não há fluxo de release móvel do Blup no workflow atual.

## Release e instalação CLI

`.goreleaser.yaml` compila `./cmd/blup` com `CGO_ENABLED=0` para Linux, macOS e Windows, em amd64 e arm64. Usa `-trimpath`, remove símbolos e injeta `app.Version`. Os arquivos seguem `blup_<versão>_<os>_<arch>.tar.gz`, ou ZIP em Windows, e incluem README, licença, avisos de terceiros e licença de ícones.

O checksum é SHA-256 em `checksums.txt`. `internal/update`, `install.sh` e `install.ps1` dependem desses nomes. As notas de release são manuais; o changelog automático do GoReleaser está desativado.

`.github/workflows/release.yml` dispara em tags `v*.*.*`, faz checkout completo, usa a versão Go do módulo, instala Hugo, roda testes da raiz e chama GoReleaser v2. Não compila Studio ou seus pacotes e não publica as ferramentas de voz nesse workflow.

`install.sh` e `install.ps1` são instaladores da CLI. O download Hugo automático não oferece pacote extraível para macOS no código atual; um binário Hugo compatível já instalado permite usar o site ali. A CLI da release não oferece inferência local Kokoro por estar sem CGO.

## Alcance de validação desta documentação

Esta spec foi produzida por inspeção do código, fixtures, testes e tarefas versionadas. Não registra como executados builds multiplataforma, empacotamentos, login em provedores ou publicação remota. Os comandos acima documentam os fluxos implementados; validações locais da documentação devem conferir links, caminhos e consistência com as fontes. Mudanças somente em Markdown não exigem recriar binários ou disparar operações de conta.
