# Conferência da spec do Blup

Conferente: Prumo. Data: 30/10/2026. Referência: commit `975554b`, árvore limpa.

Escopo: os 10 arquivos de `docs/spec/`, conferidos contra `cmd/`, `internal/`, `theme/`, `studio/`, `.github/`, `.goreleaser.yaml` e os Taskfiles. Nada foi alterado na spec nem no código.

## Resumo

A spec é fiel ao código na grande maioria das afirmações. Foram conferidas cerca de 700 afirmações concretas, e cerca de 610 estavam corretas como escritas. Entre elas estão nomes e aliases de comandos, flags, chaves TOML, valores padrão, versões (Hugo 0.169.0, Pagefind 1.5.2, Wails v3.0.0, Go 1.27.3), criptografia, PWA, protocolo do Cloudflare, API do GitHub, OpenRouter, Kokoro e a estrutura do Studio.

Os problemas são poucos erros de fato e muitas omissões e imprecisões.

| Gravidade | Itens |
| --- | --- |
| ALTA | 4 |
| MÉDIA | 13 |
| BAIXA | 64 |
| Total | 81 |

Critério:
- **ALTA**: afirmação errada ou inventada, que levaria alguém a usar ou implementar errado.
- **MÉDIA**: desatualizado, impreciso ou incompleto de forma relevante.
- **BAIXA**: detalhe menor, nome de função, omissão pequena ou redação.

Os itens marcados como "a confirmar" vêm da leitura do código e não foram executados. Os quatro itens ALTA e vários MÉDIA foram verificados por amostragem, lendo o código de novo.

No fim há uma seção à parte com problemas no próprio código, achados durante a conferência.

## ALTA

### A1. Escopo do sync em relação ao que é publicado
- **Spec**: `notas-e-sync.md:53`, "O sync considera os arquivos elegíveis das notas, inclusive anexos não usados no site. Seu escopo é maior que o dos assets efetivamente publicados."
- **Código**: o sync usa `Dir: site.ContentDir()` (`internal/app/sync.go:73`, `internal/app/deploy_api.go:266`), que é `Vault + Folder` (`internal/config/config.go:242`). O scaffold monta anexos do cofre inteiro em `assets/vault` (`internal/scaffold/scaffold.go:183`).
- **Consequência**: com `folder` preenchido, anexos de outras pastas do cofre usados pelo site são publicados mas não vão para o repositório de sync. Na recuperação em outra máquina (`BringSite`), esses anexos não voltam.
- **Correção**: dizer que o escopo difere nos dois sentidos. O sync inclui anexos não usados da pasta do site, mas exclui anexos de fora da pasta que o site publica, e esses se perdem na recuperação.

### A2. Cópia de nota não recebe nome livre
- **Spec**: `studio.md:71`, "cópia recebe nome livre e exclusão vai à lixeira".
- **Código**: `CopyNote(rel string)` não recebe nome (`studio/files.go:34`). `app.CopyNote` cria a cópia ao lado com nome numerado automático, `Nome 2.md`, `Nome 3.md` (`internal/app/files_api.go:116`).
- **Correção**: "a cópia é criada ao lado com nome numerado automático (`Nome 2.md`); exclusão vai à lixeira".

### A3. Não existe diálogo de URL no editor
- **Spec**: `studio.md:77`, "Há seletor de nota e diálogo de URL."
- **Código**: o seletor existe (`NotePicker`, `studio/frontend/src/Editor.svelte:498`). Já o link externo não abre diálogo: `link()` insere `[texto](https://)` e seleciona o endereço para ser digitado no texto (`studio/frontend/src/mdcommands.ts:64-71`). Vale para o botão e para Ctrl/Cmd+K (`Editor.svelte:360`, `:385`).
- **Correção**: "Há seletor de nota; o link externo insere `[texto](https://)` com o endereço selecionado (também Ctrl/Cmd+K)."

### A4. `blup:added` não reinicializa o feedback
- **Spec**: `temas-hugo.md:35`, "Novos elementos geram `blup:added` para inicializadores de proteção e feedback."
- **Código**: só `theme/assets/js/lock.js:74` escuta o evento. `feedback.js` roda uma vez sobre os `[data-fb-issue]` presentes no carregamento (`feedback.js:21`, `:507`). O carregar mais de `blup.js:62-78` só trata `.post-grid` e `.changelog`, então no Nori (`.nori-list`) ele navega normalmente.
- **Correção**: "Novos elementos geram `blup:added`; hoje só a proteção reage a ele. Cards trazidos pelo carregar mais não recebem contagens de feedback." Ver também o item C2.

## MÉDIA

### M1. O build pode fazer pergunta (senha geral)
- **Spec**: `cli.md:11`, "criação, personalização, escolha de múltiplos sites e publicação pela CLI ainda podem exigir perguntas"; e `cli.md:69`.
- **Código**: `encryptLocked` (`internal/app/hugo.go:109-130`). Quando uma nota usa a senha geral e não há senha salva, com terminal pede a senha com `ui.AskSecret` e grava com `SetGeneralPassword` (`internal/app/settings.go:316`), valendo para todos os sites. Sem terminal, falha com `lock.no_global`. Isso afeta builds em CI.
- **Correção**: incluir o build na lista da linha 11 e descrever o caso na seção Gerar.

### M2. Busca e PWA não bloqueiam o build; `BLUP_NO_SEARCH`
- **Spec**: `cli.md:69`, "inclusive feedback, criptografia, Pagefind e PWA".
- **Código**: `BLUP_NO_SEARCH` pula o Pagefind (`internal/app/hugo.go:145-147`). Falhas do Pagefind e do PWA só geram aviso (`hugo.go:156-158`, `:99-101`). A variável não aparece em nenhuma parte da spec.
- **Correção**: citar `BLUP_NO_SEARCH` (também na tabela de variáveis de `configuracao.md`) e dizer que busca e PWA não interrompem o build.

### M3. Arquivos de configuração ajustados pela CLI faltam na tabela de locais
- **Spec**: `configuracao.md:9-19`, tabela "Locais e responsabilidades".
- **Código**: a CLI ajusta os arquivos de configuração da origem: `.obsidian/app.json`, com backup `app.json.blup-backup` (`internal/obsidian/obsidian.go:184-197`), e também `.obsidian/templates.json` e `core-plugins.json` (`obsidian.go:213-240`). Também faltam na tabela:
  - `<site>/hugo.toml` e `hugo.preview.toml` (`internal/scaffold/scaffold.go:43`, `:49`, `:59`);
  - `<cache>/blup/update.json` (`internal/update/update.go:303-308`);
  - `~/.config/hypr/bindings.lua` ou `.conf` (`studio/hyprland.go:49-55`, `:136`), que é montado a partir de `UserHomeDir` e ignora `XDG_CONFIG_HOME`;
  - `$XDG_CONFIG_HOME/autostart/blup-studio.desktop` (`studio/desktop.go:150-158`).
- **Correção**: acrescentar essas linhas, ou um parágrafo "outros arquivos tocados" com links para os documentos que já os detalham.

### M4. O Studio empacotado sempre reporta versão `dev`
- **Spec**: `testes-e-build.md:128`, "`Studio.Version()` retorna `app.Version`, que fica `dev` se não for injetado no build".
- **Código**: nenhuma tarefa do Studio injeta `-X ...app.Version`. Os `BUILD_FLAGS` têm só `-ldflags="-w -s"` (`studio/build/linux/Taskfile.yml:62`, `studio/build/Taskfile.yml:265`).
- **Correção**: afirmar que as tarefas atuais não injetam a versão, então todo Studio empacotado mostra `dev`. Ver C3.

### M5. Boas-vindas: o que dá para pular está vago
- **Spec**: `studio.md:51`, "há caminhos para pular etapas".
- **Código** (`studio/frontend/src/Welcome.svelte`):
  - O botão "Pular" só existe nas etapas GitHub e senha, discreto no topo ao lado dos pontos de progresso (`:93-97`, commits `3dc788b` e `975554b`).
  - A etapa de idioma exige escolher pt ou en (`:82-88`).
  - Em GitHub, "Continuar" fica desativado sem conexão; em senha, "Salvar e continuar" exige senha (`:160-163`).
  - A etapa "encontrados" só aparece se houver sites (`:31`).
  - `finish` grava `welcomed` e também `shortcutTip=true` (`:74-78`).
- **Correção**: listar as etapas (idioma, GitHub, encontrados quando houver, senha, janela, pronto) e as regras acima.

### M6. Boas-vindas: onde os sites do GitHub são trazidos
- **Spec**: `studio.md:51` e `:55`, "os resultados alimentam o fluxo de escolha e recuperação" e "trazer site GitHub".
- **Código**:
  - A etapa "encontrados" só lista os sites, sem ação (`Welcome.svelte:117-125`); o texto manda trazer pelo Criar site (`studio/frontend/src/i18n.ts:59`, `foundLead`).
  - O botão "Trazer" fica no `Onboarding`, seção "No seu GitHub", que só aparece com GitHub conectado e filtra sites já com pasta local (`Onboarding.svelte:110-117`, `:184-199`).
  - Sem sites, a Home cai direto no Onboarding (`Home.svelte:61`).
- **Correção**: explicitar esse fluxo e apontar para `notas-e-sync.md` (`FindCloudSites`, `BringSite`).

### M7. Retirar consentimento recarrega a página
- **Spec**: `temas-hugo.md:134`, "A revogação atualiza o estado, mas não deve ser descrita como desinstalação retroativa".
- **Código**: retirar um "sim" dado antes recarrega a página (`theme/assets/js/consent.js:116-117`), para que as ferramentas recusadas não carreguem de novo.
- **Correção**: "Retirar um consentimento já dado grava a escolha e recarrega a página sem as ferramentas recusadas."

### M8. O que o service worker pré-carrega
- **Spec**: `temas-hugo.md:146`, "Pré-carrega páginas, estilos, scripts, fontes e ícones selecionados."
- **Código** (`internal/pwa/pwa.go:224-257`, `:319`):
  - Pré-carrega todo `index.html` do site.
  - Pré-carrega todo `.css`, `.js`, `.woff2`, `.svg`, `.ico` e `.json` fora de `pagefind/` e `vault/`, além de `icons/`. Imagens raster ficam de fora.
  - Sem rede, a navegação cai para a home em cache.
  - O cache `blup-seen` é cache-first e nunca é limpo.
- **Correção**: dizer que entram todas as páginas, o que pesa na instalação de sites grandes. Citar também o fallback para a home e o risco de conteúdo antigo em `blup-seen` para arquivos sem fingerprint.

### M9. Script de rolagem da prévia não documentado
- **Spec**: omissão em `temas-hugo.md` ("Player e instalação") e na tabela da prévia de `scaffold-e-build.md:75`.
- **Código**: com `HUGO_BLUP_MODE=preview`, `theme/layouts/_partials/mochi/scripts.html:19-21` injeta um script que, dentro de iframe, envia `postMessage` (`loaded`, `scroll`, `ready`) e obedece a `scrollTo`. É o que permite ao Studio trocar o iframe na mesma posição.
- **Correção**: acrescentar "Studio: sincronização de rolagem via postMessage, só em preview".

### M10. Relatório do sync no Studio é diferente da CLI
- **Spec**: `notas-e-sync.md:72` e `:78`, sobre o relatório trazer os nomes dos conflitos e os itens ignorados.
- **Código**: `SyncInfo` não tem campo para `Skipped` nem `SkipReasons` (`internal/app/deploy_api.go:227-236`, `:314-317`). `SyncStatus` devolve em `Conflicts` os nomes originais; `SyncNow` devolve os nomes das cópias (`rep.ConflictCopies`, `:316`). A CLI mostra ignorados e conflitos (`internal/app/sync.go:108-120`).
- **Correção**: separar o que a CLI mostra do que o Studio recebe.

### M11. APIs do Studio que usam clientes interativos da CLI (a confirmar)
- **Spec**: `publicacao.md:9` e `:64-83` não mencionam isso.
- **Código**: `PublishNow` usa clientes que não fazem perguntas. Já `CheckDomain`, `SetDomain`, `RemoveDomain` (`internal/app/domain_api.go:77`, `:109`, `:166`, `:176`) e `DeleteSite` (`internal/app/delete_site.go:98`, `:132`) usam `githubClient` e `cloudflareClient`. Esses clientes podem chamar `ui.AskSecret` e `ui.Confirm` (formulários do terminal, `internal/ui/ui.go:134-147`) quando a chave falta ou é inválida.
- **A confirmar**: o efeito na janela do Studio, sem terminal (provavelmente erro ou espera).
- **Correção**: documentar a diferença, ou tratar no código (C4).

### M12. Exclusão de site remove mais que as partes remotas
- **Spec**: `publicacao.md:85-89`.
- **Código**: `DeleteSite` também pode mover para a lixeira do sistema a pasta de notas (`Notes`) e o diretório técnico (`Site`). Sempre chama `ForgetSite` no fim (`internal/app/delete_site.go:36-43`, `:117-127`). `NotesOwned` distingue um cofre que já era do usuário (`:30`).
- **Correção**: descrever as partes locais e o `ForgetSite`.

### M13. `BLUP_VOICE_TOOLS` não serve para experimentar um build
- **Spec**: `narracao-e-voz.md:90`, "permite substituir endereço do pacote [...] para experimentar um build. A verificação SHA-256 continua obrigatória".
- **Código**: só a URL é trocada; SHA-256 e tamanho continuam os da tabela fixa (`internal/kokoro/install.go:163-165`, `:55`, `:274`). Logo, qualquer build diferente falha na verificação. A variável é lida uma vez, no início do processo (`:94`).
- **Correção**: "serve para baixar o mesmo arquivo de outro lugar (espelho ou cópia local); testar um build novo exige atualizar hash e tamanho na tabela". O comentário do código tem a mesma contradição.

## BAIXA

### README.md
1. **`README.md:48-60`**: a tabela de diretórios não cita `<site>/backups/`, onde `convert` grava ZIPs (`internal/app/links.go:63-67`). Acrescentar.
2. **`README.md:60`**: correto, mas genérico. Explicitar:
   - `os.UserCacheDir()/blup/kokoro` (`internal/app/narration_api.go:48-53`);
   - `blup/voice-samples`;
   - `blup/update.json` (`internal/update/update.go:303-309`).

### cli.md
3. **`cli.md:30`**: "`prévia`, `preview` em português por extenso e `personalização` não são aliases" está confuso. Reescrever ou remover.
4. **`cli.md:32`**: "primeiro argumento restante como diretório absoluto". Na verdade aceita caminho relativo e converte com `filepath.Abs` (`internal/app/app.go:186-187`).
5. **`cli.md:32`**: antes de abrir a criação, mostra o aviso `no_site` (`app.go:194-195`).
6. **`cli.md:47`**: se `ensureHugo` falhar, a criação retorna erro e pula feedback, sync e prévia (`internal/app/new.go:176-178`). Falha na conversão só avisa (`:173-175`).
7. **`cli.md:51`**: `Slug` não "mantém separadores `-`". Cada sequência fora de `a-z0-9`, incluindo `-`, vira um único `-`, e as pontas são aparadas, inclusive após o corte em 50 (`app.go:253-266`).
8. **`cli.md:53`**: `CreateSite` (`internal/app/create_api.go:78-98`, `:135`):
   - um modelo desconhecido vira `mochi` sem erro, e as seções são descartadas fora do Bento;
   - exige título, `vault` e `dir`, e recusa `..` em `folder`;
   - cria a pasta de notas se ela não existir;
   - ignora em silêncio a falha dos templates.
9. **`cli.md:67`**: flags desconhecidas não são rejeitadas. Viram `rest`, e a primeira vira o caminho do site (`internal/app/build_cmd.go:40-42`, `app.go:186-187`).
10. **`cli.md:69`**: se a criptografia falhar, o build apaga `public/` inteiro (`internal/app/hugo.go:92-96`).
11. **`cli.md:75-77`**: a frase "não é a mesma invocação do servidor" é vaga. A construção temporária da busca roda Hugo com `--buildDrafts --baseURL http://localhost/ --quiet`, sem `hugo.preview.toml`. O índice vai para `<site>/.blup/preview/pagefind`, e a falha só avisa (`internal/app/preview.go:79-109`). A prévia não respeita `BLUP_NO_SEARCH`.
12. **`cli.md:83`**: com plano em dia e sem conflitos, encerra sem perguntar (`internal/app/sync.go:83-86`). Nome sugerido do repositório: `Slug(título)` + `-notas` ou `-notes` (`sync.go:38`).
13. **`cli.md:102`**: "Falhas são silenciosas" vale só para a consulta. Se o usuário escolhe atualizar e falha, aparece `ui.Error` (`internal/app/update.go:63-66`).
14. **`cli.md:104`**: em `dev`, `--rollback`/`--voltar` também é recusado (`update.go:108-125`).
15. **`cli.md:7-11`**: falta uma lista das variáveis de ambiente que mudam o comportamento da CLI (`BLUP_NO_SEARCH`, `BLUP_NO_UPDATE_CHECK`).

### configuracao.md
16. **`configuracao.md:61`**: `privacy` só é passado ao tema com `banner = true` (`internal/scaffold/scaffold.go:275-279`, teste `TestCookieParams`). O rodapé sempre usa a nota `blup: privacy` (`theme/layouts/_partials/mochi/footer.html:21`). Ajustar o texto.
17. **`configuracao.md:179`**: além de detectar Hyprland, o Studio lê `~/.config/hypr/hyprland.lua|conf` para detectar Omarchy e grava o atalho global nos bindings (`studio/hyprland.go:39-46`).
18. **`configuracao.md:51`**: `[feedback].enabled` vale para todos os modelos (`scaffold.go:299`). A spec está certa; o comentário de `internal/config/config.go:114` ("Bento model only") é que está desatualizado.

### testes-e-build.md
19. **`testes-e-build.md:23`**: o build CLI de release também usa `-ldflags "-s -w -X github.com/gabriellsalesx/blup/internal/app.Version=..."` (`.goreleaser.yaml:14-15`). Sem isso, `Version` fica `dev` e a oferta de atualização some (`internal/app/update.go:29`).
20. **`testes-e-build.md:75-78`**: `go build -o bin/blup-studio .` gera build sem a tag `production`. As tarefas usam `-tags production -trimpath -buildvcs=false` (`studio/build/linux/Taskfile.yml:62`).
21. **`testes-e-build.md:65-66`, `:96`**: as tarefas rodam `npm install`, não `npm ci` (`studio/build/Taskfile.yml:30`), e respeitam `PACKAGE_MANAGER` (`studio/Taskfile.yml:6`).
22. **`testes-e-build.md:96`, `:120`, `:130`**: faltam várias tarefas e variáveis.
    - Tarefas de assinatura: `sign:deb`, `sign:rpm` e `sign:packages` (`studio/build/linux/Taskfile.yml:185-220`). A tabela só fala de assinatura em macOS e Windows.
    - Outras tarefas: `codesign:adhoc`, `codesign:skip`, `update:build-assets`, `generate:icons`, `dev:frontend` e `install:msix:tools`.
    - Variáveis: `DEV`, `OBFUSCATED` e `EXTRA_TAGS`.
23. **`testes-e-build.md:102`**: a tag `gtk3` troca a pilha GTK4/WebKitGTK 6.0.
24. **`testes-e-build.md:124`**: no nfpm, `maintainer` depende de `GIT_COMMITTER_NAME`/`EMAIL` (`studio/build/linux/nfpm/nfpm.yaml:12`), e a versão `0.1.0` é fixa, sem ligação com a tag.
25. **`testes-e-build.md:55-57`**: existem dois `TestSpeak`. Só o de `internal/kokoro/kokoro_test.go:156-160` é pulado sem `BLUP_KOKORO_DIR`.

### temas-hugo.md
26. **`temas-hugo.md:11-23`**: a tabela não cita `theme/assets/favicon.svg`, `theme/assets/defaults/` (`apple-touch-icon.png`, `avatar.png`, `avatar.svg`) nem `theme/assets/og/`.
27. **`temas-hugo.md:25`**: o fingerprint do CSS também é só em produção (`head/css.html:4`). O JS é sempre minificado e sempre recebe fingerprint.
28. **`temas-hugo.md:29`**: a descrição só aparece quando a home não tem texto (`mochi/home.html:15-19`).
29. **`temas-hugo.md:31`**: o conjunto de posts não vale para todos.
    - `term.html` usa `blog-only.html` + `visible.html` (`mochi/term.html:2`).
    - No Mochi, `pills.html` conta tags por `site.Taxonomies.tags`, incluindo posts escondidos (`pills.html:33`).
    - Só no Bento o conjunto é passado (`mochi/home.html:43`).
30. **`temas-hugo.md:33`**: no Bento, os relacionados ficam limitados à mesma seção (`mochi/page.html:68`). A confirmar: se o `related` padrão do Hugo 0.169 de fato prioriza `tags`, já que o scaffold não configura.
31. **`temas-hugo.md:41`**: as abas de feedback só aparecem se houver issue com reações **e** issue com comentários (`feedback.js:470`).
32. **`temas-hugo.md:90`**: no Bento, toda página regular vira `TechArticle`, inclusive posts e itens de roadmap (`head/jsonld.html:8`).
33. **`temas-hugo.md:98`**: faltam os atalhos de busca Ctrl/Cmd+K e `/` (`blup.js:88-94`).
34. **`temas-hugo.md:126-128`**: `feedback.js` lê só as 100 primeiras issues e os 100 primeiros comentários, sem paginação (`feedback.js:414`, `:425`).
35. **`temas-hugo.md:136`**: `cookies.privacy` só vale no link do banner (`consent.html:19`); os rodapés sempre usam `site/privacy.html`. A busca `where ... "Params.blup" "privacy"` diferencia maiúsculas (`site/privacy.html:32`), enquanto `page.html` usa `lower`.
36. **`temas-hugo.md:144`**: `manifest.json` é gerado em qualquer modo (`scaffold.go:202`), mas o link e o `blup-icon` só existem em publish (`head/meta.html:11`, `:34-37`). Sem favicon, usa `theme/assets/favicon.svg` (`pwa-source.html:4`).

### scaffold-e-build.md
37. **`scaffold-e-build.md:46`**: o Pagefind também usa `pagefind.exe` no Windows (`internal/pagefind/pagefind.go:104-107`).
38. **`scaffold-e-build.md:56`**: sem chave do GitHub e sem terminal, o sync de feedback é pulado com aviso `fb.no_token` (`internal/app/feedback.go:254-264`).

### notas-e-sync.md
39. **`notas-e-sync.md:17`**: `SetProperties` reserializa o front matter inteiro com yaml.v3, indentação 2, o que pode mudar a formatação das outras chaves. Valor `nil` remove a chave, e a nota é criada se não existir (`internal/obsidian/props.go:83-88`, `:110-114`, `:126-137`).
40. **`notas-e-sync.md:19`**: `Sections` lê só `<pasta>/_index.md`, e o regex `(?m)^blup:` casa em qualquer linha, não só no front matter (`internal/obsidian/obsidian.go:297`, `:313`).
41. **`notas-e-sync.md:23`**: a conversão de links também roda na criação do site pela CLI, com confirmação (`internal/app/new.go:173`, `internal/app/links.go:26-31`). O Studio não tem esse passo.
42. **`notas-e-sync.md:49`**: nem tudo é Git data.
    - A criação do repositório usa `POST /user/repos` (`internal/deploy/github/repo.go:143`).
    - A inicialização usa `PUT /contents` (`publish.go:440-457`).
    - Tópicos e a leitura de `site.toml` usam `/topics` e `/contents` (`topics.go:52-70`).
43. **`notas-e-sync.md:76`**: "alterações durante a execução são ignoradas" vale para o lado que recebe. No envio, `uploadOne` relê o arquivo e envia o conteúdo novo (`internal/notesync/notesync.go`).
44. **`notas-e-sync.md:97`**: `FindCloudSites` tem limites.
    - `OwnRepos` usa `affiliation=owner` e lê até 1.000 repositórios (`topics.go:24-38`).
    - `FileContent` lê até 1 MiB.
    - Repositórios com `site.toml` ilegível, sem título ou de versão futura somem da lista sem aviso (`internal/app/cloud_sites.go:143-149`).
45. **`notas-e-sync.md:99`**: em `BringSite`, se o `owner/repo` já tem site local, só carrega e não sincroniza (`cloud_sites.go:192-194`). Se o sync falhar, o `blup.toml` já foi salvo, mas o site não entra na lista (`:220-230`).

### publicacao.md
46. **`publicacao.md:11`**: a função é `LoadConnections`; `Connections` é o struct (`internal/app/deploy_api.go:24-45`).
47. **`publicacao.md:36`**: falta a consequência do `Unchanged` (`internal/app/publish.go:165-176`). Se `EnsurePages` falhou antes, republicar sem mudanças nunca habilita o Pages.
48. **`publicacao.md:53`**: a falha em `upsert-hashes` é ignorada (`internal/deploy/cfpages/deploy.go:290-292`).
49. **`publicacao.md:62`**: a URL mostrada é `live`, calculada de `EnsureProject` e `siteURL`, e não `res.URL` (`internal/app/cloudflare.go:116-131`).
50. **`publicacao.md:66`**: "retira esquema" remove só `https://` e `http://`. Também passa para minúsculas antes do IDNA e ignora erro de `ToASCII` (`internal/app/domain.go:27-33`).
51. **`publicacao.md:68`**: se a desconexão falhar, `RemoveDomain` retorna erro e mantém `site.Domain` (`internal/app/domain_api.go:57-61`).

### narracao-e-voz.md
52. **`narracao-e-voz.md:7`**: a regra é `!isIndex && (kind === "" || kind === "blog") && !props.blup` (`studio/frontend/src/Props.svelte:50`). Notas do Bento fora de pastas marcadas também podem ser narradas (`studio/notes.go:154-166`).
53. **`narracao-e-voz.md:21`**: `Speakable` também faz outras limpezas (`internal/narrate/text.go:24-26`, `:66`, `:112-115`).
    - Remove tags HTML, mantendo o texto.
    - Remove atributos `{.x}` e `{#id}`.
    - Remove referências `[^n]`.
    - Das definições de nota de rodapé, só a primeira linha sai.
54. **`narracao-e-voz.md:58`**: `PreviewVoice` só sintetiza via OpenRouter e sempre exige a chave. Quem decide usar uma amostra gravada é o frontend (`internal/app/narration_api.go:244-275`).
55. **`narracao-e-voz.md:96`**: os dicionários são `pt_dict` e `en_dict`, mas o script copia a pasta `lang/` inteira do eSpeak (`studio/build/voice/build-tools.sh:70-71`).

### studio.md
56. **`studio.md:13`**: `SetLanguage` fica em `studio/about.go:16-31` e aceita só `pt` ou `en`; qualquer outro valor vira `en`. A recarga só acontece pelas Configurações (`SettingsModal.svelte:100-103`). Nas boas-vindas, o idioma é aplicado sem recarregar (`Welcome.svelte:36-39`). Os rótulos da bandeja só mudam ao reiniciar (`studio/main.go:65-68`).
57. **`studio.md:21-34`**: a tabela de serviços não lista `studio/media.go` (`canPlayMP3`). O idioma fica dividido entre `studio.go:352` (`Language`) e `about.go` (`SetLanguage`).
58. **`studio.md:45`**: o hook não é restaurado; é zerado com `ui.Report = nil` (`studio/publish.go:34-35`).
59. **`studio.md:67`, `:150`**: a spec não diz onde ficam as preferências. São gravadas em `<config>/blup/studio-prefs.json` (`studio/layout.go:94-123`), e a senha geral fica em `credentials.toml` (`internal/app/settings.go:309-323`). Citar os dois e linkar `configuracao.md`.
60. **`studio.md:61`**: faltam elementos do workspace (`Workspace.svelte`).
    - Barra superior com Prévia, Sincronizar e Publicar (`:463-465`).
    - Troca e exclusão de site no rodapé lateral (`:444-453`).
    - Botões de ajuda e de engrenagem (`:440-441`).
    - Modal de dica do atalho global, com variante para Hyprland sem Omarchy (`:179-187`, `:512-526`).
    - Aba Seções, só no Bento (`:199`).
61. **`studio.md:75`, `:81`**: faltam detalhes do editor e das propriedades.
    - O editor tem botão de alinhamento (`Editor.svelte:330-364`).
    - A narração não aparece em `_index.md` nem em nota com `blup:` (`Props.svelte:50`).
    - Os campos reais são `audio`, `audio_hash` e `audio_seconds`; "revisão" é o hash (`Props.svelte:86-88`).
    - A ordem do Bento é a propriedade `weight` (`Props.svelte:31`).
62. **`studio.md:119-130`**: a aba Atalhos também mostra a lista fixa do editor (`k.editorKeys`) e as instruções de Hyprland, com botão de copiar (`SettingsModal.svelte:426-468`).
63. **`studio.md:132-142`**: o terminal tem abas e uma tela de revisão das mudanças da IA, com `AIChanges` e `UndoAI` (`TerminalPanel.svelte:28`, `:139-144`).
64. **`studio.md:136`**: os guias de agente são escritos ao abrir qualquer terminal, inclusive shell sem agente. Só o snapshot depende do agente (`studio/terminal.go:100`, `:115-118`).

Observação: o empacotamento do Studio (AppImage, nfpm com `gstreamer1.0-plugins-good`, dmg, nsis/msix) já está em `testes-e-build.md`. Em `studio.md` basta um link para lá.

## Problemas no próprio código (fora do escopo da spec, para o dono decidir)

- **C1. Busca da prévia provavelmente quebrada em Linux com `/tmp` em tmpfs (a confirmar por execução).** `previewSearch` cria o temporário com `os.MkdirTemp("", ...)` e move com `os.Rename` para `<site>/.blup/preview/pagefind` (`internal/app/preview.go:80`, `:105`). Nesta máquina, `/tmp` é tmpfs e `/home` fica em outro sistema de arquivos. O rename entre os dois falha com EXDEV, e o resultado é só o aviso `build.search_failed`. Correção possível: criar o temporário dentro de `<site>/.blup/`.
- **C2.** O carregar mais não reinicializa o feedback e não trata `.nori-list` (`theme/assets/js/blup.js:62-78`, `feedback.js`). Ligado ao item A4.
- **C3.** O Studio empacotado não injeta `app.Version` e mostra `dev` (item M4).
- **C4.** As APIs de domínio e de exclusão de site usam clientes que podem abrir formulários de terminal dentro do Studio (item M11).
- **C5.** Há comentários desatualizados no código.
  - `internal/config/config.go:114` diz que o feedback é só para Bento.
  - `internal/app/create_api.go:24` diz "mochi or bento", sem citar o Nori.
  - `internal/kokoro/install.go:57-58` diz que as ferramentas de voz são feitas pelo workflow de release, mas `release.yml` não as gera.
  - `install.go` diz que `BLUP_VOICE_TOOLS` serve para testar um build, o que o hash fixo impede (item M13).
