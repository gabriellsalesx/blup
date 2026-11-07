# Especificação técnica do Blup

Esta especificação descreve a implementação do repositório `github.com/gabriellsalesx/blup`, conferida em 02/11/2026, incluindo os contratos da rodada 4 de pré-deploy. A leitura do código prevalece sobre mensagens de commit e comentários antigos. Os caminhos citados são relativos à raiz. Exemplos são fictícios; integração implementada não comprova provisionamento de infraestrutura de produção.

Para a visão entre componentes, consulte [Arquitetura](../arquitetura.md). A operação está em [Deploy de produção](../deploy.md) e no [checklist em ordem](../deploy-checklist.md), e as fronteiras de confiança, em [Segurança](../seguranca.md). O [guia de instalação](../instalacao.md) explica a distribuição do Studio, inclusive a abertura sem assinatura no Windows e no macOS.

## Índice

| Documento | Conteúdo |
| --- | --- |
| [CLI](cli.md) | Comandos em português e inglês, menu, criação, geração, prévia, publicação, conversão, sync, personalização e atualização |
| [Configuração](configuracao.md) | `blup.toml`, arquivos globais, preferências do Studio e propriedades das notas |
| [Temas Hugo](temas-hugo.md) | Mochi, Nori, Bento, calendário do changelog, ícones Lucide, renderização, busca, PWA, proteção, feedback e consentimento |
| [Quadro de Feedback](feedback.md) | Seção Bento do Studio, exemplos, pedidos, votos, status, `/posts`, limites e tradução |
| [Podcast dos blogs](podcast.md) | Seção do Studio nos blogs, episódios das narrações, player, tela de bloqueio, feed para Spotify e Apple |
| [Scaffold e build](scaffold-e-build.md) | Arquivos gerados, mounts, ferramentas, pipeline e mapa da prévia |
| [Notas e sync](notas-e-sync.md) | Notas Markdown, wikilinks, sincronização, conflitos e recuperação em outra máquina |
| [Blup Cloud e sync cifrado](blup-cloud-sync.md) | Conta, planos, dispositivos, cifra de ponta a ponta, histórico, IA e cobrança |
| [Blup Pulse](blup-pulse.md) | Fases 1 a 4 com gates locais fechados: ingestão, monitores, páginas, avisos, cobrança e domínio; staging, aceite Studio e liberação comercial pendentes |
| [Blup Deals](blup-deals.md) | F1 local: canais, afiliados, ofertas/jogos, links curtos, cliques e painel; preços aprovados no sandbox, 100 posts/dia e evolução F2/F3 |
| [Calendário e lembretes](calendario-e-lembretes.md) | Eventos Markdown, recorrência, notificações locais e e-mails Max 50 / Ultra 100 |
| [Publicação](publicacao.md) | Sites em `getblup.site`, Cloudflare Workers e GitHub Pages; domínio próprio só no Blup Cloud |
| [Narração e voz](narracao-e-voz.md) | Idioma pela pasta da nota, voz por idioma, OpenRouter, Blup Cloud, MP3 e Kokoro |
| [Studio](studio.md) | Wails, Svelte, editor, Feedback, tradução, reinício da prévia, painéis, terminal e integração desktop |
| [Tipagem e contratos](tipagem-e-contratos.md) | Hono RPC/Zod, bindings/eventos Wails, null/opcionais, arquivos externos e limites residuais |
| [Testes e build](testes-e-build.md) | Validação, compilação dos dois módulos, empacotamento e release |

## O que é o Blup

O Blup transforma uma pasta de notas Markdown em um site estático Hugo. O Studio trabalha com cofres próprios, recortes de pastas ou uma pasta comum. As notas continuam sendo arquivos no disco. O Blup monta essa origem no Hugo em vez de manter uma segunda cópia editável do conteúdo.

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
| `internal/deploy/github`, `internal/deploy/cfworkers` | Publicação por HTTP, sem `git` nem `wrangler` |
| `internal/notesync` | Comparação de base, disco e remoto; transporte GitHub ou adaptador cifrado Blup Cloud |
| `internal/blupcloud`, `internal/cloudcrypt` | Cliente Blup Cloud, blocos e manifestos cifrados, histórico e publicação |
| `internal/narrate`, `internal/kokoro` | Texto falável, geração de voz remota ou local e MP3 |
| `internal/update` | Atualização e rollback do executável da CLI |
| `studio/`, `studio/frontend/` | Aplicativo desktop, bindings, editor, prévia e painéis |
| `cloud/` | Conta, painel SvelteKit, API Hono/Better Auth, D1/R2, Stripe e Worker dos sites |

O fluxo de publicação é: notas e configurações alimentam o scaffold; Hugo gera `public/`; o Blup atualiza os dados de feedback e pode repetir Hugo; incorpora anexos locais pequenos nas regiões protegidas e as cifra; poda mídias sem referência pública; cria busca e PWA; envia `public/` ao provedor. O sync das notas é uma operação separada da publicação.

O [quadro de Feedback](feedback.md) é acrescentado em Seções pelo Studio. A nota marcada `blup: feedback` guarda exemplos; os pedidos reais são issues criadas pelos visitantes e lidas no navegador. O dono altera seus status no painel Feedback, sem republicar. Interface, exemplos e conversas têm contratos de idioma diferentes: o i18n traduz os controles, a tradução da nota cobre os exemplos e as conversas reais mantêm o texto de origem, compartilhado entre as versões do site.

O Studio também abre cofres `kind = "notes"`, com diário, mapa de notas, canvas, desenhos, tabelas e calendário. Eventos são notas; avisos locais dependem do aplicativo aberto, e e-mails dependem de uma lista de horários enviada à nuvem. O calendário do changelog do Bento é outro recurso, gerado estaticamente.

A troca de idioma do Studio atualiza os textos sem recarregar a janela. Nos sites traduzidos, a publicação escolhe uma tradução disponível conforme o navegador e usa inglês como fallback quando essa página existe. Nori compartilha posts com os controles comuns do Mochi. Narração escolhe idioma pela pasta da nota e permite modelo/voz Blup Cloud, com reserva de cota e custo reportado pelo provedor quando disponível, senão estimativa de catálogo. Sem escolha remota explícita, não troca voz local por nuvem. IA remota exige aceite por destino e recusa notas com senha. Veja [Studio](studio.md), [Temas Hugo](temas-hugo.md) e [Narração](narracao-e-voz.md).

O Podcast organiza narrações públicas em episódios e feed. O reinício após mudanças de seções trata o servidor local de prévia, sem reiniciar o desktop. Notas podem sincronizar pelo GitHub ou pelo Blup Cloud; só o segundo transporte cifra blocos, nomes e manifestos no cliente. Conteúdo enviado à IA e saída de sites públicos têm outras fronteiras de privacidade, descritas em [Segurança](../seguranca.md).

## Contratos atuais após a rodada 4

A [tradução do painel Cloud](blup-cloud-sync.md#idioma-da-conta-e-do-painel) oferece interface e e-mails em `pt/en/es`, com preferência na conta e migração `0014` antes do Worker novo. A Lupa confirmou os gates finais, o ensaio D1 local e o teste de navegador; o [registro da rodada](../auditoria/rodada4-documentacao.md#fechamento-posterior-da-tradução-cloud) distingue essa validação da aplicação em produção.

- Blup Cloud: Max permite 5 GiB de armazenamento e 3 sites; Ultra, 10 GiB e 10 sites. Arquivo máximo de 10/25/50/100 MiB para Grátis/Pro/Max/Ultra; reservas D1 e leases protegem admissão/publicação. Excesso após redução abre 14 dias com teto de storage anterior, contando histórico; depois volta ao novo limite sem apagar dados só por excesso.
- Sites: depois da carência Pro retém um site escolhido até o prazo, ou o mais antigo por data/ID; Grátis retém zero. Arquivos dos demais permanecem guardados. Domínio próprio Max/Ultra tem carência própria de 14 dias e `disableAt`/`disabled`.
- URL oficial: domínio e SSL ativos, com plano ou carência válida, podem ser baseURL. O Studio detecta mudança e republica em segundo plano, uma vez a cada dez minutos por site, para ajustar canonical/feed/sitemap; ambos os hosts servem o deploy sem redirects entre eles.
- Sync: após 413, o cliente pode preparar uma versão, guardar blocos novos em memória e confirmar por ticket/reserva líquida. Purga manual de histórico é por cofre; o tamanho mostrado é da conta e pode ser `null`, sem virar zero. Versão atual e blocos compartilhados necessários são preservados.
- Entrega: gateway sem cache consulta D1 em cada pedido e chama Workers Cache por site/deploy. `SITE_CACHE` executa purga persistente no contexto `CachedSite`; Early Hints e ganhos de custo reais exigem validação no ambiente Cloudflare.
- Studio/tema: home Bento exibe seção escolhida ou única em `/`, com paginação em `/page/2/`; `logo_link` vale nos três temas, com validação e mesma aba. Canvas preserva cartões de tipos desconhecidos. Guias `INSTRUCTIONS.md` por modelo/cofre ficam na pasta das notas com três arquivos de entrada, preservando texto editado pela pessoa.
- Exclusão de conta continua com bloqueio/revogação imediatos, retenção de 30 dias e purga diária em lotes/repetição. Autoatualização CLI usa Ed25519 dos checksums e SHA-256; instalação inicial permanece só SHA-256. Builds `production` exigem versão `x.y.z`.
- Sync portátil inclui `translations`, `home` e `logo_link`; `Deploy`, `Feedback`, credenciais e senha geral ficam fora. Tipos RPC/Wails e os limites residuais estão em [tipagem e contratos](tipagem-e-contratos.md).

A rodada 4 acrescenta preços Stripe anuláveis em `/plans`, webhook próprio assinado com reconciliação do estado atual e recibos, migrações até `0013`, validações de DTOs Go e mensagens da prévia/tradução, ID de conta copiável e aviso de e-mail ausente nos lembretes. Pacotes Linux e limites de validação estão em [testes e build](testes-e-build.md); sequência operacional e requisitos de cada canal estão no [checklist](../deploy-checklist.md).

Os relatórios das [rodada 2](../auditoria/rodada2-documentacao.md), [rodada 3](../auditoria/rodada3-documentacao.md) e [rodada 4](../auditoria/rodada4-documentacao.md) registram o estado e os checks de cada conferência. O histórico da rodada anterior não substitui os contratos atuais acima.

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
- A prévia mostra rascunhos, notas `publish: false`/`private: true` e o corpo de posts protegidos. Usa proxy loopback com Host/Origin conferidos. A publicação passa por criptografia. Rodar somente Hugo com `HUGO_BLUP_MODE=publish` não executa essa etapa Go.
- `hidden: true` mantém a página acessível pela URL e a retira das listagens que usam o filtro do tema. Não equivale a senha ou exclusão do build.
- O sync de notas não envia automaticamente os overrides do diretório técnico, os tokens ou a senha geral do site. Senhas literais presentes nas próprias notas viajam como conteúdo dessas notas.
- A instalação automática do Kokoro está completa na tabela de plataformas somente para Linux amd64 com CGO. Outras entradas não têm pacote de ferramentas configurado.
- O workflow publica a CLI e contém jobs para os pacotes Linux do Studio com SHA-256 e Ed25519; MSIX fica como artefato de execução, condicionado à identidade Partner Center. Configuração de CI não prova execução ou publicação já concluída. Linux usa AppImage, AUR, `.flatpak` no GitHub e `.deb`; Flathub está fora deste lançamento.

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

A seção abaixo é o histórico da primeira conferência, não uma lista de pendências atuais. Versão de produção, índice entre discos e exclusão do destino Blup Cloud já mudaram desde aquele parecer. Na revisão da conferência, os achados de código C1 a C5 foram documentados sem correção de código por este trabalho. Ela registrou o rename da busca que podia falhar entre sistemas de arquivos; as limitações de carregar mais e feedback; a versão `dev` dos pacotes do Studio; os clientes interativos de domínio e exclusão, sem presumir seu efeito numa janela sem terminal; e os comentários desatualizados. A revisão de 01/11/2026 atualiza os contratos que mudaram desde então, inclusive o fallback de cópia do índice da prévia e a descoberta de seções só nas propriedades. A tabela de instalação de voz e o workflow de release prevalecem sobre o comentário que atribui a geração das ferramentas ao workflow.

## Divergências com a conferência

1. **M2, divergência parcial sobre ausência da variável.** `BLUP_NO_SEARCH` já constava da tabela de variáveis de configuração e do pipeline em scaffold e build na versão conferida. A implementação em `internal/app/hugo.go` confirma que ela pula Pagefind no build; `internal/app/preview.go` não a consulta. A revisão ampliou a explicação na CLI e esclareceu os avisos de busca e PWA. A sugestão de esclarecer o comportamento foi acolhida; a afirmação de ausência em toda a spec não procede.
2. **B8, divergência parcial sobre caminhos obrigatoriamente preenchidos.** `CreateSite`, em `internal/app/create_api.go`, chama `expandHome` antes de testar se `Vault` e `Dir` estão vazios. Em `internal/app/new.go`, `expandHome` usa `filepath.Abs`, que normalmente converte a string vazia no diretório atual. Portanto a API exige título não vazio, mas não rejeita necessariamente entradas originalmente vazias de cofre e diretório por esse teste. Outras validações, inclusive `CheckSiteDir`, ainda podem rejeitar o resultado. A spec descreve essa ordem efetiva, além das demais regras aceitas do item.
3. **M12, divergência parcial sobre sempre retirar o site da lista.** `DeleteSite`, em `internal/app/delete_site.go`, chama `ForgetSite` no final, mas retorna antes se falhar `LoadSite` ou o envio de uma pasta local à lixeira. Problemas remotos são acumulados e permitem continuar até essa chamada. A spec descreve a retirada da lista quando a execução alcança o final, além da exclusão local e remota solicitada pelo parecer.
