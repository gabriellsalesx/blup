# CLI e coordenação da aplicação

Fontes: [entrada](../../cmd/blup/main.go), [dispatcher e menu](../../internal/app/app.go), `internal/ui/ui.go`, `internal/i18n/i18n.go` e `internal/i18n/messages.go`.

## Entrada, idioma e retorno

`main` chama `app.Run(context.Background(), os.Args[1:])`. Erros reconhecidos como cancelamento pela UI encerram com código 130; os demais são apresentados pela UI e encerram com código 1. Sucesso encerra normalmente.

`Run` carrega primeiro o estado global, inclusive para versão e ajuda. `Version` começa como `dev` e pode ser injetado com `-ldflags`. A interface tem `pt`, `en` e `es`; a tradução cai para inglês e depois para a própria chave quando falta uma mensagem. Os aliases dos comandos continuam sendo os da tabela, sem uma terceira lista de comandos em espanhol.

Para `build`/`gerar`, ou quando não há terminal interativo, o dispatcher seleciona a linguagem salva ou inglês e evita a pergunta inicial de idioma. Isso não torna todos os fluxos não interativos: criação, personalização, escolha de múltiplos sites, publicação e o próprio build ainda podem exigir perguntas. O build pode pedir a senha geral de posts protegidos, como detalhado em Gerar. `help` usa o fluxo de escolha de idioma; `version` apenas imprime a versão.

Variáveis que mudam esses fluxos: `BLUP_NO_SEARCH` não vazio pula Pagefind no build de publicação; `BLUP_NO_UPDATE_CHECK` não vazio pula a oferta automática de atualização. A indexação inicial da prévia da CLI não consulta `BLUP_NO_SEARCH`. A lista de variáveis e seus escopos está em [configuração](configuracao.md).

## Comandos reconhecidos

| Inglês | Português ou alternativas | Efeito |
| --- | --- | --- |
| Sem argumento | Sem argumento | Abre o menu, ou a criação se não houver site existente |
| `new` | `novo` | Assistente de criação |
| `preview [site]` | `ver [site]` | Prévia Hugo local |
| `publish [site]` | `publicar [site]` | Gera e publica no destino configurado |
| `build [site]` | `gerar [site]` | Build em `public/`, com opções de URL e cópia |
| `convert [site]` | `converter [site]` | Conversão assistida de wikilinks |
| `customize [site]` | `personalizar [site]` | Menu de personalização |
| `sync [site]` | `sincronizar [site]` | Sincroniza as notas com GitHub |
| `language` | `idioma` | Escolhe e salva o idioma da interface |
| `update` | `atualizar` | Atualiza o executável da CLI |
| `version` | `--version`, `-v` | Imprime `blup <versão>` |
| `help` | `--help`, `-h` | Imprime ajuda traduzida |

Os nomes são comparados literalmente. Não há parser geral de subcomandos ou flags globais. Comando desconhecido imprime ajuda e devolve erro. Para a prévia, os nomes aceitos são `preview` e `ver`; `prévia` e `personalização` não são comandos reconhecidos.

`withSite` resolve o alvo nesta ordem: primeiro argumento restante como caminho, relativo ou absoluto, convertido por `filepath.Abs`; `blup.toml` no diretório atual; único site global existente; escolha entre vários sites. Se não houver site, mostra o aviso traduzido `no_site` e encaminha para criação. O diretório atual não é procurado subindo até os pais. Após carregar um site, ele é lembrado como o mais recente no estado global.

O menu usa o primeiro site existente como atual. Oferece prévia, publicação, personalização, sync, conversão, troca de site quando há vários, criação, idioma e saída. A maior parte dos erros de uma ação é exibida e devolve o usuário ao menu.

## Criar site

`internal/app/new.go` implementa perguntas com retorno por Esc. O fluxo é:

1. Escolher um cofre descoberto em registros do Obsidian ou informar outro caminho. A seleção manual da CLI exige uma pasta com `.obsidian`.
2. Publicar o cofre inteiro ou uma pasta. Lista pastas até dois níveis e permite criar uma pasta nova.
3. Escolher Mochi ou Bento. No Bento, explicar e selecionar pelo menos uma seção entre `docs`, `roadmap`, `changelog`, `help` e `blog`.
4. Informar título, descrição, autor, idioma do site e diretório técnico vazio ou inexistente. A sugestão usa o diretório de sites traduzido e `Slug(title)`.
5. Confirmar o resumo e, quando necessário, oferecer o ajuste dos links do Obsidian.
6. Criar exemplos se não houver notas. Se já houver, garantir `_index.md` e adicionar as seções selecionadas. Criar propriedades do site e a nota do autor.
7. Acrescentar templates do Obsidian, gerar scaffold, salvar `blup.toml` e lembrar o site globalmente.
8. Oferecer conversão dos wikilinks encontrados, garantir Hugo, oferecer feedback, sync privado e abertura da prévia.

Templates são um complemento: falha nessa etapa gera aviso na CLI. Na conversão opcional, erro que não seja cancelamento também só gera aviso. Se `ensureHugo` falhar, a criação retorna erro e não oferece feedback, sync ou prévia. A pasta técnica é criada antes das últimas etapas; não há transação geral que desfaça tudo quando um passo posterior falha.

`Slug` remove acentos, converte para minúsculas, mantém `a-z` e `0-9` e troca cada sequência dos demais caracteres, inclusive hífens, por um único `-`. Apara hífens nas pontas, limita a 50 caracteres, apara de novo após o corte e usa `site` para resultado vazio. O Cloudflare tem sanitização própria para nomes de projeto.

A API `CreateSite`, em `internal/app/create_api.go`, recebe `NewSiteOptions` com `vault`, `folder`, `dir`, `model`, `sections`, `title`, `description`, `author`, `lang` e `adjustLinks`. Ela é usada pelo Studio, aceita pasta comum e Nori. Modelo diferente de `bento` ou `nori` vira `mochi`, sem erro; fora do Bento, descarta `sections`. Exige título não vazio, valida os tipos e a seleção não vazia de seções Bento, recusa `..` em `folder` e normaliza idioma diferente de `pt` e `es` para `en`.

`vault` e `dir` passam por `expandHome` antes da verificação de vazio: se `filepath.Abs` funcionar, uma entrada vazia vira o diretório atual, portanto essa verificação não garante rejeitar caminho vazio de entrada. O diretório técnico ainda precisa passar por `CheckSiteDir`. A API cria as pastas de notas e técnica quando ausentes e registra o site. Não cria a pasta de modelos do Obsidian (Templates/Modelos/Plantillas): o Studio cria as notas ele mesmo e a pasta só confundia a árvore; só a CLI a cria. Não executa as perguntas opcionais, o download Hugo ou a conversão do fluxo da CLI.

Os exemplos estão embutidos em `internal/app/starter/{pt,en,es}/{mochi,nori,bento}` e em `starter/covers`. `starter.go` copia arquivos ausentes, preenche dados do site e permite acrescentar uma seção sem sobrescrever notas existentes. `templates.go` fornece templates de post para Mochi/Nori e de documentação, roadmap e changelog para Bento.

Feedback tem modelo de pasta nos três idiomas, mas fica em `studioKinds`, separado de `docsKinds`. Os seletores da CLI e a validação de seções de `CreateSite` continuam usando somente `docsKinds`; o Studio acrescenta o quadro depois da criação pelo painel Seções. Isso é diferente de habilitar comentários e reações, opção de feedback que a CLI já oferece. Consulte o [quadro de Feedback](feedback.md). O [Podcast dos blogs](podcast.md) segue a mesma regra: fica em `blogKinds`, a CLI não o oferece, e um blog novo nunca recebe a pasta `Podcast`.

## Gerar

Fontes: `internal/app/build_cmd.go` e `internal/app/hugo.go`.

```sh
blup gerar /caminho/do/site
blup build /caminho/do/site --url https://exemplo.test/blog/ --out /tmp/site-pronto
blup gerar --endereco=https://exemplo.test/ --saida=/tmp/site-pronto /caminho/do/site
```

Flags aceitas: `--url`/`--endereco` e `--out`/`--saida`, com `=` ou valor no próximo argumento. A URL deve ter esquema HTTP ou HTTPS e host; seu caminho recebe `/` final. Sem URL explícita, usa o domínio próprio ou `/`, sem calcular automaticamente a URL do provedor salvo. O parser não rejeita flags desconhecidas: elas ficam em `rest`, e o primeiro argumento restante é interpretado como caminho do site, inclusive se começar com `--`.

O build passa pelo pipeline de publicação local, com feedback, criptografia e tentativas de gerar Pagefind e PWA. `BLUP_NO_SEARCH` não vazio pula Pagefind; falhas de busca ou PWA só geram aviso e não interrompem o build. Não envia os arquivos ao provedor. Mesmo sem deploy, feedback ativo e credenciais disponíveis podem provocar criação ou atualização de issues.

Se uma nota pede senha geral e não há senha resolvida para o site, `encryptLocked` pergunta a senha quando há terminal interativo e a grava em `credentials.toml` por `SetGeneralPassword`, valendo para todos os sites sem senha própria. Sem interatividade, retorna `lock.no_global`, inclusive em CI. Falha na etapa de proteção apaga todo `<site>/public` e encerra o build, antes de qualquer cópia por `--out`.

`--out` copia `public/` para o destino e grava `.blup-build`. A cópia só pode substituir uma pasta inexistente, vazia ou já marcada por Blup. Pasta com conteúdo sem marcador é recusada. O diretório técnico continua com sua própria saída em `public/`.

## Prévia

`internal/app/preview.go` atualiza o scaffold, prepara uma cópia temporária para indexar com Pagefind e executa `hugo server` com `hugo.preview.toml`, `--renderToMemory`, `--buildDrafts`, `--navigateToChanged`, bind em `127.0.0.1` e base local. Define `HUGO_BLUP_MODE=preview`.

O servidor fica atrás de `PreviewGuard`, proxy loopback em porta aleatória com Host/Origin conferidos; Hugo usa uma segunda porta interna. A URL do proxy é aberta no navegador. Ctrl+C cancela o processo e volta ao menu. Notas protegidas aparecem legíveis com aviso, sem gravar sua página em `public/`. O índice da busca reflete o conteúdo no início da prévia; não é refeito a cada salvamento.

Para esse índice, `previewSearch` cria uma saída temporária com `hugo --buildDrafts --baseURL http://localhost/ --destination <temporário> --quiet`, sem selecionar `hugo.preview.toml`, e move `pagefind/` para `<site>/.blup/preview/pagefind`. `placeSearch` usa `moveDir`, com tentativa de rename e fallback de cópia quando necessário. Falhas só geram aviso; `BLUP_NO_SEARCH` não pula essa etapa. O índice é inicial, sem atualização a cada salvamento.

## Publicar e sincronizar

`internal/app/publish.go` pergunta o provedor somente quando não foi definido. A ordem de opções começa por GitHub. Credenciais são solicitadas e verificadas, destino é preparado, o site é gerado com a URL correta e os arquivos são enviados. O estado é salvo após sucesso; a CLI conecta ou verifica o domínio e oferece abrir a URL. Detalhes em [publicação](publicacao.md).

O seletor da CLI oferece GitHub e Cloudflare. Um site já configurado com `deploy.provider = "blup"` é encaminhado a `deployBlup`, usando a sessão Blup Cloud salva pelo Studio. Não há comando próprio da CLI para login Blup Cloud, agenda, tradução, narração ou criação de Podcast/Feedback. Seus arquivos e seções já existentes ainda podem ser processados pelo build compartilhado.

`internal/app/sync.go` prepara o repositório privado, orientando criação/inclusão em `github.com/new` quando a chave restrita não permite criá-lo, sugerindo `Slug(title)` mais `-notas` ou `-notes` conforme o idioma da interface, escreve os dados portáteis e consulta o plano. Se estiver em dia e sem conflitos, encerra sem perguntar; caso contrário pede confirmação antes de aplicar alterações. Um repositório já público exige confirmação adicional na CLI. O Studio recusa esse caso na API compartilhada. A CLI mostra conflitos, itens ignorados e lixeira; a resposta do Studio é mais restrita. Detalhes em [notas e sync](notas-e-sync.md).

Esse comando ainda prepara explicitamente o transporte GitHub: não é uma interface da CLI para o sync cifrado do Blup Cloud. O Studio escolhe o adaptador de acordo com `sync.remote`; o contrato da nuvem está em [Blup Cloud](blup-cloud-sync.md).

`convert` faz simulação, mostra exemplos, pede confirmação, cria ZIP das notas afetadas em `<site>/backups/notes-AAAA-MM-DD-HHMMSS.zip` e somente então reescreve as notas. Não é uma etapa automática de cada build.

## Personalizar

`internal/app/customize.go` repete o menu e salva `blup.toml` e o scaffold após uma mudança bem-sucedida. Inclui:

- Menu superior com seleção de pastas e tags, links externos e botão com `button_text` e `button_link`. Um menu com submenus não é refeito pelo terminal, que avisa (`custom.menu_sub`) e pergunta só o botão; os submenus se editam no Studio ou no Obsidian.
- Inclusão de seções no Bento. O fluxo CLI de seções adiciona exemplos; a remoção é feita nas notas. O Studio tem remoção para a lixeira.
- Feedback com issues e instalação do GitHub App; cookies, ferramentas e política de privacidade.
- Redes sociais, logo, favicon, nome, descrição e autor/perfil.
- Domínio próprio, consulta de estado e remoção.
- Senha geral específica do site.

Nome, descrição, menu, redes e imagens ficam também nas propriedades da nota inicial; veja [configuração](configuracao.md). Essas opções não publicam imediatamente as alterações.

## Atualização da CLI

`internal/app/update.go` consulta atualizações ao abrir o menu, exceto em `dev`, sem terminal interativo ou com `BLUP_NO_UPDATE_CHECK` definido. A consulta tem timeout de dois segundos e cache de 24 horas. Falhas da consulta automática são silenciosas. O usuário pode atualizar, adiar ou ignorar aquela versão; se escolher atualizar e a instalação falhar, o erro é exibido por `ui.Error`.

`blup update` faz a consulta explícita; `--rollback` ou `--voltar` restaura o executável anterior. Builds `dev` recusam o comando antes de analisar essas flags, portanto também recusam rollback. `internal/update/update.go` lê GitHub Releases, seleciona arquivo para o sistema e arquitetura, verifica assinatura Ed25519 de `checksums.txt` e depois SHA-256 do pacote, substitui o executável e mantém backup. A nova versão roda na próxima inicialização. Isso não é um atualizador implementado para o aplicativo Studio.

O cliente exige HTTPS para pacote, `checksums.txt` e `checksums.txt.sig`, recusando downgrade para HTTP em redirecionamentos. Verifica Ed25519 sobre os bytes exatos dos checksums com as chaves públicas de `internal/update/keys.go`; recusa assinatura ausente/inválida, checksum divergente e versão que não seja mais nova (`ErrNotNewer`, inclusive igual e `dev`). A base HTTP substituível serve aos testes.

A assinatura é produzida por `scripts/sign-release`, com `BLUP_RELEASE_KEY` no CI ou arquivo privado fora do repositório. A lista de chaves públicas permite rotação preparada em releases intermediárias. Os instaladores iniciais `install.sh` e `install.ps1` continuam conferindo somente SHA-256; não aplicam a verificação Ed25519 da autoatualização. Assinatura nativa do Studio é outro fluxo, descrito em [deploy](../deploy.md#9-lançamento-do-studio-e-da-cli).
