# Especificação técnica do Blup

Esta especificação descreve a implementação encontrada no repositório `github.com/gabriellsalesx/blup`, com referência inicial no commit `975554b`, em 30/10/2026. A revisão de 01/11/2026 cobre os 14 commits de `c0514f3` a `dc6345c`, conferidos com `git show` e leitura do código, e as rotinas relacionadas de narração e idiomas. Os caminhos citados são relativos à raiz do repositório. Os exemplos usam dados fictícios. Comportamentos de serviços externos são descritos conforme as chamadas e condições implementadas, sem presumir que uma conta remota esteja configurada.

## Índice

| Documento | Conteúdo |
| --- | --- |
| [CLI](cli.md) | Comandos em português e inglês, menu, criação, geração, prévia, publicação, conversão, sync, personalização e atualização |
| [Configuração](configuracao.md) | `blup.toml`, arquivos globais, preferências do Studio e propriedades das notas |
| [Temas Hugo](temas-hugo.md) | Mochi, Nori, Bento, calendário do changelog, ícones Lucide, renderização, busca, PWA, proteção, feedback e consentimento |
| [Quadro de Feedback](feedback.md) | Seção Bento do Studio, exemplos, pedidos, votos, status, `/posts`, limites e tradução |
| [Podcast dos blogs](podcast.md) | Seção do Studio nos blogs, episódios das narrações, player, tela de bloqueio, feed para Spotify e Apple |
| [Scaffold e build](scaffold-e-build.md) | Arquivos gerados, mounts, ferramentas, pipeline e mapa da prévia |
| [Notas e sync](notas-e-sync.md) | Obsidian, wikilinks, sincronização, conflitos e recuperação em outra máquina |
| [Publicação](publicacao.md) | Cloudflare Pages, GitHub Pages, autenticação e domínio próprio |
| [Narração e voz](narracao-e-voz.md) | Idioma pela pasta da nota, voz por idioma, OpenRouter, Blup Cloud, MP3 e Kokoro |
| [Studio](studio.md) | Wails, Svelte, editor, Feedback, tradução, reinício da prévia, painéis, terminal e integração desktop |
| [Testes e build](testes-e-build.md) | Validação, compilação dos dois módulos, empacotamento e release |

## O que é o Blup

O Blup transforma uma pasta de notas Markdown em um site estático Hugo. A pasta pode ser um cofre do Obsidian, parte dele ou, pela API usada pelo Studio, uma pasta comum. As notas continuam sendo arquivos no disco. O Blup monta essa origem no Hugo em vez de manter uma segunda cópia editável do conteúdo.

Há dois executáveis. A CLI entra por `cmd/blup/main.go` e oferece fluxos interativos ou um comando de build. O Blup Studio entra por `studio/main.go` e expõe serviços Go a uma interface Svelte. Ambos usam os pacotes do módulo principal, sobretudo `internal/app`. O Studio é um módulo Go próprio, com `replace github.com/gabriellsalesx/blup => ../` em `studio/go.mod`.

O site publicado é um conjunto de arquivos estáticos. Busca usa Pagefind no navegador. Comentários, reações, votos e pedidos do quadro de Feedback usam issues do GitHub; o serviço de login e leitura correspondente pertence ao repositório irmão `blup-api`. A narração pode usar OpenRouter, Blup Cloud ou um motor local opcional, com a voz escolhida pelo idioma da nota. Nenhum desses recursos transforma o site em um servidor Go.

## Peças e ligações

| Peça | Responsabilidade e chamadas principais |
| --- | --- |
| `cmd/blup`, `internal/ui`, `internal/i18n` | Entrada, perguntas, progresso, mensagens e idioma da CLI |
| `internal/app` | Coordena criação, configurações, build, publicação, sync e APIs reutilizadas pelo Studio |
| `internal/config` | Estado global, credenciais e configuração por site |
| `internal/obsidian`, `internal/wikilinks` | Descoberta dos cofres, propriedades YAML e conversão explícita de links |
| `internal/scaffold`, `embed.go` | Geram configurações Hugo e extraem o tema embutido |
| `internal/hugo` | Localiza ou baixa Hugo e cria seus processos |
| `theme/` | Templates, CSS, JavaScript, fontes, traduções e imagens padrão do site |
| `internal/lock` | Substitui regiões protegidas por conteúdo criptografado após Hugo |
| `internal/pagefind`, `internal/pwa` | Índice estático, ícones e service worker depois do build |
| `internal/deploy/github`, `internal/deploy/cfpages` | Publicação por HTTP, sem `git` nem `wrangler` |
| `internal/notesync` | Sync bidirecional por blobs, árvores, commits e referências do GitHub |
| `internal/narrate`, `internal/kokoro` | Texto falável, geração de voz remota ou local e MP3 |
| `internal/update` | Atualização e rollback do executável da CLI |
| `studio/`, `studio/frontend/` | Aplicativo desktop, bindings, editor, prévia e painéis |

O fluxo de publicação é: notas e configurações alimentam o scaffold; Hugo gera `public/`; o Blup atualiza os dados de feedback e pode repetir Hugo; criptografa regiões protegidas; cria busca e PWA; envia `public/` ao provedor. O sync das notas é uma operação separada da publicação.

O [quadro de Feedback](feedback.md) é acrescentado em Seções pelo Studio. A nota marcada `blup: feedback` guarda exemplos; os pedidos reais são issues criadas pelos visitantes e lidas no navegador. O dono altera seus status no painel Feedback, sem republicar. Interface, exemplos e conversas têm contratos de idioma diferentes: o i18n traduz os controles, a tradução da nota cobre os exemplos e as conversas reais mantêm o texto de origem, compartilhado entre as versões do site.

As demais mudanças dos 14 commits estão descritas em [Studio](studio.md#idiomas-e-tradução) e [Temas Hugo](temas-hugo.md): tradução de `linkTitle` e perfis de autores, comentários com indicação de idioma, selo de IA no quadro, calendário do changelog e disposição adaptável dos controles de idiomas e de Quem traduz. O reinício após mudanças de seções é do servidor local de prévia; não reinicia o aplicativo desktop.

## Diretórios que não devem ser confundidos

| Local | Papel |
| --- | --- |
| `site.Vault` e `site.Folder` | Origem das notas; `ContentDir()` junta os dois |
| `site.Dir()` | Diretório técnico do site, que contém `blup.toml`, configurações Hugo, tema e saída |
| `<site>/layouts`, `assets`, `static`, `i18n`, `data` | Extensões locais preservadas pelo scaffold, embora `data/blup_feedback.json` seja gerenciado pelo feedback |
| `<site>/themes/blup` | Cópia descartável do tema embutido, substituída pelo scaffold |
| `<site>/public` | Resultado de publicação, regenerado pelo build |
| `<site>/backups/` | ZIPs das notas afetadas pela conversão de wikilinks, criados antes da reescrita |
| `<notas>/.blup/site.toml` | Configuração portátil enviada com as notas |
| `<site>/.blup/` | Estado local de sync, prévia, lixeira e snapshots |
| `os.UserConfigDir()/blup` | Lista de sites, credenciais e preferências globais |
| `os.UserCacheDir()/blup` | Hugo, Pagefind, voz opcional, amostras de voz e consulta de atualizações |
| `os.UserCacheDir()/blup/kokoro` | Instalação opcional do motor de voz local |
| `os.UserCacheDir()/blup/voice-samples` | Amostras sintetizadas pelo OpenRouter |
| `os.UserCacheDir()/blup/update.json` | Cache da consulta de atualização da CLI |

Fontes centrais: [config](../../internal/config/config.go), [scaffold](../../internal/scaffold/scaffold.go), [build](../../internal/app/hugo.go), [serviço Studio](../../studio/studio.go).

## Limites confirmados na implementação

- Existem três modelos válidos: `mochi`, `nori` e `bento`. O assistente da CLI em `internal/app/new.go` oferece somente Mochi e Bento. O Studio oferece os três.
- A prévia mostra rascunhos e o corpo de posts protegidos. A publicação passa por criptografia. Rodar somente Hugo com `HUGO_BLUP_MODE=publish` não executa essa etapa Go.
- `hidden: true` mantém a página acessível pela URL e a retira das listagens que usam o filtro do tema. Não equivale a senha ou exclusão do build.
- O sync de notas não envia automaticamente os overrides do diretório técnico, os tokens ou a senha geral do site. Senhas literais presentes nas próprias notas viajam como conteúdo dessas notas.
- A instalação automática do Kokoro está completa na tabela de plataformas somente para Linux amd64 com CGO. Outras entradas não têm pacote de ferramentas configurado.
- O workflow de release versionado publica a CLI. Os arquivos de empacotamento do Studio existem, mas não há publicação do Studio nesse workflow.

Comentários antigos do código ainda usam expressões como feedback exclusivo do Bento ou apenas dois modelos. Esta spec segue as condições executadas, os templates e as telas atuais.

## Revisão da conferência

O [parecer do Prumo](../auditoria/conferencia-spec.md) foi conferido item por item contra a implementação. Dos 81 itens, 80 receberam correções ou complementos nos documentos abaixo. O item baixo 18 foi confirmado sem correção: `Feedback.On()` e `feedbackOn`, em `internal/config/config.go` e `internal/scaffold/scaffold.go`, não restringem o recurso ao Bento. Nesta tabela, `B` identifica os itens de baixa gravidade numerados de 1 a 64 no parecer.

| Documento | Altos | Médios | Baixos |
| --- | --- | --- | --- |
| Este README | Nenhum | Nenhum | B1, B2 |
| [CLI](cli.md) | Nenhum | M1, M2 | B3, B4, B5, B6, B7, B8, B9, B10, B11, B12, B13, B14, B15 |
| [Configuração](configuracao.md) | Nenhum | M3 | B16, B17; B18 confirmado sem correção |
| [Temas Hugo](temas-hugo.md) | A4 | M7, M8, M9 | B26, B27, B28, B29, B30, B31, B32, B33, B34, B35, B36 |
| [Scaffold e build](scaffold-e-build.md) | Nenhum | Nenhum | B37, B38 |
| [Notas e sync](notas-e-sync.md) | A1 | M10 | B39, B40, B41, B42, B43, B44, B45 |
| [Publicação](publicacao.md) | Nenhum | M11, M12 | B46, B47, B48, B49, B50, B51 |
| [Narração e voz](narracao-e-voz.md) | Nenhum | M13 | B52, B53, B54, B55 |
| [Studio](studio.md) | A2, A3 | M5, M6 | B56, B57, B58, B59, B60, B61, B62, B63, B64 |
| [Testes e build](testes-e-build.md) | Nenhum | M4 | B19, B20, B21, B22, B23, B24, B25 |

O script de rolagem do item M9 também foi acrescentado à comparação entre prévia e publicação em scaffold e build. A observação final sobre empacotamento ganhou um link no documento do Studio.

Na revisão da conferência, os achados de código C1 a C5 foram documentados sem correção de código por este trabalho. Ela registrou o rename da busca que podia falhar entre sistemas de arquivos; as limitações de carregar mais e feedback; a versão `dev` dos pacotes do Studio; os clientes interativos de domínio e exclusão, sem presumir seu efeito numa janela sem terminal; e os comentários desatualizados. A revisão de 01/11/2026 atualiza os contratos que mudaram desde então, inclusive o fallback de cópia do índice da prévia e a descoberta de seções só nas propriedades. A tabela de instalação de voz e o workflow de release prevalecem sobre o comentário que atribui a geração das ferramentas ao workflow.

## Divergências com a conferência

1. **M2, divergência parcial sobre ausência da variável.** `BLUP_NO_SEARCH` já constava da tabela de variáveis de configuração e do pipeline em scaffold e build na versão conferida. A implementação em `internal/app/hugo.go` confirma que ela pula Pagefind no build; `internal/app/preview.go` não a consulta. A revisão ampliou a explicação na CLI e esclareceu os avisos de busca e PWA. A sugestão de esclarecer o comportamento foi acolhida; a afirmação de ausência em toda a spec não procede.
2. **B8, divergência parcial sobre caminhos obrigatoriamente preenchidos.** `CreateSite`, em `internal/app/create_api.go`, chama `expandHome` antes de testar se `Vault` e `Dir` estão vazios. Em `internal/app/new.go`, `expandHome` usa `filepath.Abs`, que normalmente converte a string vazia no diretório atual. Portanto a API exige título não vazio, mas não rejeita necessariamente entradas originalmente vazias de cofre e diretório por esse teste. Outras validações, inclusive `CheckSiteDir`, ainda podem rejeitar o resultado. A spec descreve essa ordem efetiva, além das demais regras aceitas do item.
3. **M12, divergência parcial sobre sempre retirar o site da lista.** `DeleteSite`, em `internal/app/delete_site.go`, chama `ForgetSite` no final, mas retorna antes se falhar `LoadSite` ou o envio de uma pasta local à lixeira. Problemas remotos são acumulados e permitem continuar até essa chamada. A spec descreve a retirada da lista quando a execução alcança o final, além da exclusão local e remota solicitada pelo parecer.
