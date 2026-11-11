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

Para reproduzir o perfil sem CGO e a injeção de versão usados na release CLI, com `1.0.0` como exemplo:

```sh
CGO_ENABLED=0 go build -tags production -trimpath -ldflags="-s -w -X github.com/gabriellsalesx/blup/internal/app.Version=1.0.0" -o bin/blup ./cmd/blup
```

Sem tag `production` e sem `-X ...app.Version=...`, o binário mantém `dev`, e a oferta automática de atualização fica desativada. Com `production`, `internal/app/version_production.go` recusa iniciar sem versão `x.y.z`, evitando distribuir um executável `dev`. O comando de build cria o binário; sua execução usa o estado real do usuário, pode escrever configurações e pode baixar Hugo/Pagefind. Uma validação de documentação não precisa iniciar os fluxos interativos.

| Área | Testes presentes e finalidade |
| --- | --- |
| `internal/config` | Modelos aceitos, gravação/leitura da versão e rejeição de versão futura |
| `internal/i18n` | Correspondência dos catálogos e existência das chaves usadas |
| `internal/obsidian` | Propriedades e preservação de notas |
| `internal/wikilinks` | Resolução, conversão, âncoras e backups |
| `internal/notesync` | Comparação de versões, conflitos, exclusões, falhas e servidor GitHub falso |
| `internal/cloudcrypt`, `internal/blupcloud` | Cifra, embrulhos, limites de derivação, isolamento de blocos, histórico, sync e publicação com HTTP simulado |
| `internal/deploy/github` | Repositórios, publicação, Pages e autenticação com HTTP simulado |
| `internal/deploy/cfworkers` | Hashes, sessão de upload, buckets, publicação do Worker, subdomínio workers.dev e erros com API simulada |
| `internal/lock` | Criptografia e pós-processamento |
| `internal/pwa` | Ícones e worker |
| `internal/narrate` | Limpeza, partes, HTTP e MP3 |
| `internal/kokoro` | Tokens, fonemas, estilo, pausas e instalação simulada |
| `internal/update` | Versões, download, checksums, troca e rollback |
| `internal/app` | Criação, exemplos, configurações, domínio Blup Cloud, feedback, arquivos, autores, anexos, snapshots e recuperação |
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

O frontend declara Svelte 5, Vite 8, TypeScript, CodeMirror 6, runtime Wails 3, xterm e Hunspell WASM, com `package-lock.json` versionado. Há scripts de build, análise Svelte e testes Node (`node --test tests/*.test.ts`); estes últimos não são uma suíte completa de navegador.

Na pasta `studio/frontend`, este fluxo manual usa `npm ci` com o lockfile versionado:

```sh
npm ci
npm run check
npm test
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
go build -tags production -trimpath -buildvcs=false -ldflags="-w -s -X github.com/gabriellsalesx/blup/internal/app.Version=1.0.0" -o bin/blup-studio .
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
| DEB | `libgtk-4-1 (>= 4.10)`, `libwebkitgtk-6.0-4` | `gstreamer1.0-plugins-good` |
| RPM | `gtk4 >= 4.10`, `webkitgtk6.0` | `gstreamer1-plugins-good` |
| Arch | `gtk4`, `webkitgtk-6.0` | `gst-plugins-good` |

Esses são nomes de runtime dos arquivos de pacote, não uma lista universal de pacotes de desenvolvimento. `studio/media.go` procura o plugin de parsers de áudio e pode desativar o player na prévia quando ausente. Isso não substitui instalar os componentes de mídia da distribuição.

`studio/build/linux/Taskfile.yml` usa build nativo em Linux quando há compilador C e arquitetura compatível; caso contrário tenta Docker com imagem `wails-cross`. Build desktop nativo define `CGO_ENABLED=1`, `GOOS=linux` e arquitetura escolhida. Release usa tag `production`, `-trimpath`, `-buildvcs=false` e remoção de símbolos; desenvolvimento usa flags distintas.

## Empacotamento Studio

| Plataforma | Tarefas e arquivos |
| --- | --- |
| Linux | `studio/build/linux/Taskfile.yml`, nfpm, geração de `.desktop`, AppImage, DEB, RPM, pacote Arch e assinatura DEB/RPM |
| macOS | `studio/build/darwin/Taskfile.yml`, `.app`, build universal, DMG, assinatura e notarização |
| Windows | `studio/build/windows/Taskfile.yml`, recursos `.syso`, NSIS, MSIX e assinatura |

O [nfpm.yaml](../../studio/build/linux/nfpm/nfpm.yaml) recebe `VERSION`, `GOARCH` e `DEB_MAINTAINER`. O workflow usa como padrão o contato corporativo `Blup <no-reply@getblup.io>`; o responsável deve conferir que o contato operacional escolhido funciona. O binário vai a `/usr/bin/blup-studio`; desktop entry, MetaInfo, ícones de 128/256/512 e licença/avisos vão a locais sob `/usr/share`. Não há scripts de instalação. O `.deb` exige GTK 4.10 e WebKitGTK 6.0, com GStreamer Good recomendado; os alvos são Ubuntu 24.04+ e Debian 13+.

A tarefa Linux `package` chama AppImage, DEB, RPM e Arch. AppImage usa `wails3 generate appimage` com `LDAI_OUTPUT` absoluto para funcionar também com caminhos contendo espaços. Existe um script auxiliar `studio/build/linux/appimage/build.sh`; ele não é o comando chamado por essa tarefa. RPM e pacote Arch via nfpm são capacidades locais; a receita AUR publicada é outra, em [aur/PKGBUILD](../../studio/build/linux/aur/PKGBUILD), com [update.sh](../../studio/build/linux/aur/update.sh) e [.SRCINFO](../../studio/build/linux/aur/.SRCINFO).

As tarefas Linux `sign:deb`, `sign:rpm` e `sign:packages` chamam `wails3 tool sign`; podem usar a configuração de assinatura Wails ou `PGP_KEY`, e a tarefa DEB também passa `SIGN_ROLE` quando definido. A tarefa conjunta assina DEB e RPM, não todos os formatos Linux.

As tarefas macOS e Windows incluem arquivos e opções de assinatura específicos. Ter esses templates não demonstra que certificados, chaves ou ferramentas de assinatura estejam instalados. A decisão atual é distribuir Windows/macOS sem assinatura de desenvolvedor e usar AppImage, AUR, `.flatpak` no GitHub e `.deb` no Linux. A publicação no Flathub está [em preparação](../flathub.md). As tarefas de assinatura nativa continuam disponíveis para uma etapa futura. O [guia de instalação](../instalacao.md) explica SmartScreen e Gatekeeper; o [checklist](../deploy-checklist.md) separa geração de pacote de envio às lojas. `build/config.yml` informa produto, identificador e versão `1.0.0`. `Studio.Version()` retorna `app.Version`, injetada por ldflags nas Taskfiles Linux/macOS/Windows e nos builds cruzados, com `VERSION` padrão `1.0.0`. Para outra release, passe a versão real e confira os metadados de plataforma. O workflow já deriva da tag a versão do binário Linux, nfpm, MetaInfo e nomes dos arquivos.

Há tarefas adicionais `build:server`, `run:server`, `build:docker`, `run:docker` e `setup:docker`. O modo server usa tags `server` e `production` e pode construir sem GUI nativa. Isso é uma capacidade de build Wails presente no projeto, sem substituir as integrações desktop ou comprovar um serviço hospedado de Studio em produção. Também há assets/tarefas iOS de origem Wails; não há fluxo de release móvel do Blup no workflow atual.

## Release e instalação CLI

`.goreleaser.yaml` compila `./cmd/blup` com `CGO_ENABLED=0` para Linux, macOS e Windows, em amd64 e arm64. Usa tag `production`, `-trimpath`, remove símbolos e injeta `app.Version`. Os arquivos seguem `blup_<versão>_<os>_<arch>.tar.gz`, ou ZIP em Windows, e incluem README, licença, avisos de terceiros e licença de ícones.

O checksum é SHA-256 em `checksums.txt`, acompanhado de `checksums.txt.sig`, assinatura Ed25519 dos bytes exatos dos checksums. `internal/update`, `install.sh` e `install.ps1` dependem desses nomes. GoReleaser executa `go run ./scripts/sign-release` com a chave privada fornecida por `BLUP_RELEASE_KEY` no CI ou arquivo fora do repositório. O atualizador exige assinatura válida com a lista pública de `internal/update/keys.go` antes de conferir SHA-256 e substituir o executável; recusa versão igual/anterior e `dev`. Os instaladores iniciais continuam só com SHA-256. Rotação e preparação de chave estão em [deploy](../deploy.md#assinatura-da-autoatualização-cli). As notas de release são manuais; o changelog automático do GoReleaser está desativado.

`.github/workflows/release.yml` dispara em tags `v*.*.*`. O job `release` faz checkout completo, usa Go do módulo, instala Hugo, testa a raiz e chama GoReleaser v2. O workflow também inclui os jobs Studio descritos abaixo; não publica as ferramentas de voz.

### Pacotes e jobs do Studio

O app ID é `io.getblup.Studio`, licença `AGPL-3.0-only`, adotado em 11/11/2026 nos metadados e no empacotamento do Studio. A verificação do Flathub será pelo domínio `getblup.io`, conforme o [guia](../flathub.md#verificação-pelo-domínio). A identidade MSIX reservada no Partner Center continua vindo das variáveis de publicação, conforme as exigências da Microsoft.

`studio-linux`, após a CLI, usa Ubuntu 24.04, Go do módulo Studio, Node 24 e Wails 3.1.0. Constrói AppImage, tarball, `.deb` e bundle Flatpak; valida MetaInfo/desktop entry e prevê lintian, instalação do `.deb` no Ubuntu 24.04/Debian 13 e resolução de bibliotecas no Arch e Flatpak. Esses checks do CI estão configurados, sem execução dos jobs comprovada nesta revisão.

`studio-publish` baixa os artefatos Linux e publica na release os arquivos abaixo, com `studio-checksums.txt` e assinatura Ed25519 `studio-checksums.txt.sig`, usando `BLUP_RELEASE_KEY`:

- `blup-studio_X.Y.Z_linux_amd64.tar.gz`, usado pela receita `blup-studio-bin` do AUR.
- `blup-studio_X.Y.Z_x86_64.AppImage`.
- `blup-studio_X.Y.Z_amd64.deb`.
- `blup-studio_X.Y.Z_x86_64.flatpak`.

O [bundle.sh](../../studio/build/linux/flatpak/bundle.sh) usa `org.gnome.Platform//51` e exporta o app com permissões explícitas de rede, áudio, renderização, notificações, bandeja, Documentos e `~/Blup`, sem home inteira. O runtime pode vir do Flathub; não há submissão do aplicativo lá. O bundle não é um repositório de atualização próprio.

A receita AUR usa binário pré-compilado e dependências `gtk4`, `webkitgtk-6.0`, `libsoup3`, `glib2`, `glibc`, `pango`, `harfbuzz`, `cairo`, `gdk-pixbuf2`, `graphene`, `libx11` e `vulkan-icd-loader`; áudio e Hugo são opcionais. Antes do envio, `update.sh` recebe a versão publicada, lê o SHA-256 do tarball e regenera `.SRCINFO`. O script não verifica a assinatura Ed25519 da lista; o mantenedor deve conferir a procedência antes de usar o checksum. A receita com checksum placeholder não está pronta para submissão.

`studio-msix` usa as variáveis `MSIX_IDENTITY_NAME`, `MSIX_PUBLISHER` e `MSIX_PUBLISHER_DISPLAY_NAME` do Partner Center. O [gerador](../../scripts/msix-manifest/main.go) preenche o [template Store](../../studio/build/windows/msix/AppxManifest.store.xml) e a tarefa empacota x64/arm64 com `CHANNEL=msstore`. Sem Publisher, o job é pulado com aviso. Os MSIX sem assinatura ficam apenas como artefatos da execução, sem upload à Store ou à release. Falta execução em Windows/MakeAppx e teste do pacote; os testes Go do gerador não substituem essa etapa.

No handoff do Prumo, foram conferidos AppImage em caminho com espaços, MetaInfo também em modo pedantic, desktop entry, inspeção do `.deb`, receita AUR com makepkg/pacman e bundle instalado isoladamente com bibliotecas resolvidas. Lintian/instalação Debian e jobs GitHub ficaram para o CI; isso não comprova abertura gráfica em todos os sistemas. Preparação, requisitos da Store e publicação de receitas estão no [checklist](../deploy-checklist.md).

A atualização via GitHub é consultada no Studio direto/AppImage/macOS; o executável não é substituído automaticamente. `internal/app/channel.go` e `internal/app/studio_update.go` deixam Store/MSIX, Flatpak e pacotes sob `/usr` com seu próprio canal, e a tela Sobre explica por onde atualizar.

`install.sh` e `install.ps1` são instaladores da CLI. O download Hugo automático não oferece pacote extraível para macOS no código atual; um binário Hugo compatível já instalado permite usar o site ali. A CLI da release não oferece inferência local Kokoro por estar sem CGO.

## Blup Cloud e checks de pré-deploy

Na raiz, os checks exigidos pelo time são:

```sh
go vet ./...
go test ./...
(cd studio && go vet . && go test ./...)
(cd cloud && npx tsc -p worker/tsconfig.json --noEmit && npx vitest run)
(cd studio/frontend && npx svelte-check)
```

`cloud/package.json` gera `src/lib/rpc.d.ts` por `worker/export-rpc.mjs` antes de `check` e `build`. O contrato do painel vem de `AppType` e não deve ser refeito por DTOs manuais. Também oferece `check` para SvelteKit/Svelte/TypeScript e `build` para assets de produção. Não confunda analisar apenas `worker/tsconfig.json` com construir o painel. `npm ci` usa os lockfiles; não se deve acrescentar dependência para validar documentação.

Os testes Cloud usam D1/R2/auth simulados. `plans.test.ts`, `history.test.ts`, `reminders.test.ts` e `site-rules.test.ts` cobrem regras dos planos, limpeza, agenda e publicação; `security.test.ts`, `auth-security.test.ts` e `serve-site-security.test.ts` exercitam entradas forjadas e fronteiras de origem, autenticação e cache. As regressões de cotas concorrentes, contabilização de IA, limpeza de deploy e revogação agora são testes normais, sem `it.fails`. A rodada 2 acrescentou contratos RPC/erros, JSON e multipart malformados, domínios com prova TXT/SSL, retenção/purga e lembretes com tentativas. A suíte verde vale para os cenários simulados; veja [Segurança](../seguranca.md#cotas-concorrência-e-revogação).

No cliente, `internal/app/blup_reminders_test.go` confere payload opaco; `narration_blup_test.go` confere escolha de voz e preço desconhecido; `internal/cloudcrypt/attack_test.go` cobre parâmetros abusivos, troca de dados autenticados e recuperação. `studio/guard_test.go` e `paths_security_test.go` verificam chamadas Wails e caminhos de arquivos. Há também regressões para assinatura Ed25519/versão, consentimento de IA, arquivos privados, anexos cifrados, proxy Host/Origin, manifestos até 100 MiB, respostas externas e normalização de Prefs/agenda/canvas/desenho/bancos. `studio/frontend/tests/security.test.ts` verifica os tratamentos de HTML/link no frontend.

Testes HTTP locais usam portas efêmeras próprias. Se o sandbox impedir `httptest` de abrir sockets, é preciso executar o mesmo teste com a permissão apropriada, preservando portas e processos reservados. Para temporários grandes, use pasta da sessão e `TMPDIR`/`GOTMPDIR`; remova só os arquivos da própria execução. Testes de IA, Stripe e e-mail usam simulações; não comprovam entrega, cobrança ou publicação de produção.

## Regressões da rodada 3

| Arquivo | Cobertura |
| --- | --- |
| `cloud/worker/round3-security.test.ts` | Carências de domínio/storage, site retido, titularidade/prazo, preparação líquida, cancelamento/expiração, purga por cofre e R2 indisponível |
| `cloud/worker/site-cache.test.ts` | Props/site/deploy, autorização antes do hit, headers de navegador, fila/geração de purga, limites e Early Hints |
| `internal/blupcloud/prepare_test.go` | Multipart/ticket/header, blocos preparados, erros de contrato e sequência do commit |
| `internal/blupcloud/history_test.go` | Purga, campos nullable, retorno e compatibilidade |
| `internal/app/blup_base_test.go`, `studio/republish_test.go` | Domínio/SSL/plano/carência como base oficial, falha temporária e republicação controlada |
| `internal/app/bento_home_test.go`, `logo_link_test.go` | Home como seção, URLs/pager/canonical/sitemap, nomes de pasta e links do logo nos três temas |
| `internal/app/agents_test.go`, `agents_legacy_test.go` | Guias por modelo/cofre, preservação de edições e migração do texto antigo intacto |
| `studio/frontend/tests/boards.test.ts`, `cloudhistory.test.ts`, `logolink.test.ts` | Canvas desconhecido preservado, total de conta/nullable e validação do link do logo |

## Regressões da rodada 4

| Arquivo | Cobertura |
| --- | --- |
| `internal/blupcloud/sites_answer_test.go` | Presença, nulos, invariantes de sites, deploy, listas missing, cofres, embrulhos, lembretes, purga e ID da conta |
| `internal/app/notes_key_race_test.go` | Perder criação inicial da chave não conserva chave/recuperação local incorreta |
| `studio/watch_test.go` | Exclusão externa de pasta do topo solicita reinício da prévia |
| `studio/frontend/tests/previewurl.test.ts` | URL depois de reinício e parser de mensagens da prévia |
| `studio/frontend/tests/translatereply.test.ts`, `fieldvalue.test.ts` | Resposta tipada da tradução e valor numérico finito, vazio e ausência de mudança |
| `cloud/worker/pricing.test.ts`, `cloud/src/lib/api.test.ts` | Catálogo Stripe, cache/falha/indisponibilidade, configuração por RPC e formatação monetária |
| `cloud/scripts/stripe-setup.test.ts` | Conferência de catálogo faz somente leitura e não cria preços |
| `cloud/worker/security.test.ts`, `auth-tooling.test.ts` | Erros IA, embrulho atual, corrida da chave e integração de auth com serviços simulados |
| `studio/frontend/tests/accountid.test.ts`, `cloud/src/lib/account-id.test.ts`, `ui/AccountId.test.ts` | Textos aprovados, copiar ID e feedback, nos limites de cada teste |

Os testes frontend dessa rodada cobrem parsers/funções puras e contratos. Não constituem teste do `Preview.svelte` montado, do worker Bergamot completo ou da instalação do pacote em todos os sistemas.

As corridas reportadas pela documentação têm testes normais com barreiras: PUT duplicado suspenso antes da lease e retomado depois do commit deve conservar bytes; upgrade que limpa `disable_at` antes do claim deve impedir DELETE externo. Há proteção de confirmação D1 perdida e regressão dos uploads normais paralelos, preservando exclusividade de preparo/purga. Verificar somente o status HTTP não comprova essas propriedades de integridade.

A Lupa também informou aplicação das migrações `0000..0011` do zero por Wrangler no D1 local e EXPLAIN usando os novos índices. Esse resultado é evidência da validação da área Cloud, separada da suíte SQLite simulada e da configuração remota de produção. Não reaplique migrações remotas para validar Markdown.

## Alcance de validação desta documentação

Esta spec foi produzida por inspeção do código, fixtures, testes e tarefas versionadas. Não registra como executados builds multiplataforma, empacotamentos, login em provedores ou publicação remota. Os comandos acima documentam os fluxos implementados; validações locais da documentação devem conferir links, caminhos e consistência com as fontes. Mudanças somente em Markdown não exigem recriar binários ou disparar operações de conta.
