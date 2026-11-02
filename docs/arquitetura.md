# Arquitetura do Blup

Retrato do código em 02/11/2026, incluindo os contratos da rodada 4. Consulte o [índice da spec](spec/README.md), o [guia de produção](deploy.md), o [checklist operacional](deploy-checklist.md) e o [modelo de segurança](seguranca.md). Código e configuração versionados prevalecem sobre mensagens de commit e textos de exemplo.

## Componentes e fronteiras

```mermaid
flowchart LR
  Pessoa[Pessoa] --> Studio[Blup Studio: Wails e Svelte]
  Pessoa --> CLI[CLI Go]
  Studio --> App[internal/app]
  CLI --> App
  App <--> Notas[Markdown e anexos no disco]
  App --> Hugo[Scaffold e Hugo]
  Hugo --> Saida[Site estatico]
  Saida --> Pages[Cloudflare Pages ou GitHub Pages]
  Saida --> Sites[Worker blup-sites e R2 SITES]
  App <--> Cloud[Worker Blup Cloud]
  Cloud <--> D1[D1 e Drizzle]
  Cloud <--> Notes[R2 NOTES: dados cifrados]
  Cloud <--> DO[Durable Object SyncLive]
  Cloud --> Email[Email Service]
  Cloud <--> Stripe[Stripe]
  Cloud --> IA[OpenRouter]
  App --> Local[Kokoro ou traducao local]
  App --> IA
  Visitante[Visitante] --> Sites
  Visitante --> Pages
```

| Parte | Fonte e responsabilidade |
| --- | --- |
| CLI | [cmd/blup/main.go](../cmd/blup/main.go), [internal/app/app.go](../internal/app/app.go): comandos, perguntas e coordenação Go |
| Studio | [studio/main.go](../studio/main.go), [studio/studio.go](../studio/studio.go), [frontend](../studio/frontend/src/App.svelte): Wails v3, Svelte 5, CodeMirror, bindings Go/TypeScript e eventos |
| Regras locais | `internal/app`, `internal/config`, `internal/obsidian`, `internal/notesync`: persistência, arquivos, criação, sync e provedores |
| Tema | [theme](../theme): Hugo embutido no Go, modelos Mochi, Nori e Bento, CSS, JavaScript e i18n |
| Conta e API | [cloud/worker/index.ts](../cloud/worker/index.ts): Hono, middleware de sessão, validação Zod e APIs |
| Identidade e cobrança | [auth-options.ts](../cloud/worker/auth-options.ts): Better Auth, passkeys, OTP, aparelho, Bearer e plugin Stripe |
| Persistência cloud | [schema](../cloud/worker/db/schema.ts): Drizzle sobre D1; R2 guarda objetos; não há servidor Node persistente |
| Painel cloud | [cloud/vite.config.ts](../cloud/vite.config.ts): SvelteKit com adapter-static e fallback SPA, assets do Worker, Tailwind e Lucide |
| Sites públicos | [sites-entry.ts](../cloud/worker/sites-entry.ts), [serve-site.ts](../cloud/worker/serve-site.ts): outro Worker, sem o login ou os assets do painel |
| Feedback público | [feedback.js](../theme/assets/js/feedback.js) e repositório irmão `blup-api`: issues do GitHub, login e leituras com cache |

Raiz e `studio/` são módulos Go separados, unidos por `replace` local. O Studio usa Wails `v3.1.0` em `studio/go.mod`; o tema é compilado no executável pelo `embed.go`. `cloud/package.json` e seu lockfile definem as versões da aplicação web, incluindo Better Auth `^1.7.14`. Atualizar documentação de uma dependência não significa atualizar automaticamente essa versão ou os imports do projeto.

## Estado e propriedade dos dados

| Local | O que guarda |
| --- | --- |
| Pasta das notas | Markdown, YAML, anexos, arquivos de canvas/desenho/banco e agenda; é a origem editável |
| Diretório técnico do site | `blup.toml`, configurações Hugo, overrides, `public/` e `.blup/` local |
| `<notas>/.blup/site.toml` | Subconjunto portátil das configurações, enviado pelo sync |
| Configuração da máquina | Lista de sites, preferências, credenciais e chave local de sync; não vai no site |
| D1 | Conta, sessões, assinatura, staff, IDs/uso de sync, sites, cota de IA e horários de lembretes |
| R2 `NOTES` | Blocos e manifestos cifrados, referências de blocos para limpeza |
| R2 `SITES` | Arquivos públicos, staging e manifestos; anteriores protegidos temporariamente para limpeza segura |

O arquivo local de notas é legível no disco. A cifra de ponta a ponta é a fronteira do sync Blup Cloud. Sites públicos, chamadas de IA e e-mails não usam essa mesma fronteira. [Segurança](seguranca.md) detalha os dados visíveis em cada fluxo.

## Criar e escrever

`Onboarding.svelte` reúne tema, seções, nome e origem das notas e chama `CreateSite`; `VaultOnboarding.svelte` cria ou associa um cofre `kind = "notes"`, sem publicação. A CLI tem seu próprio assistente. O serviço Go valida o diretório técnico, cria a configuração e escreve exemplos quando aplicável. O Studio cria suas notas e exemplos diretamente na pasta escolhida. Feedback Bento e Podcast de Mochi/Nori são acrescentados depois, por Seções.

O workspace lê corpo e propriedades separadamente, agenda salvamento e observa alterações externas. O editor grava arquivos Markdown na pasta escolhida, também acessível aos agentes no terminal. Renomear e mover corrigem os formatos de links reconhecidos pelo código; conflito de edição não é um merge semântico automático. A lixeira e snapshots ficam fora da pasta publicada. Cofres acrescentam nota do dia, agenda, mapa, backlinks, `.canvas`, `.draw` e `.table`.

A home Bento pode renderizar uma seção em `/`, com paginação em `/page/2/`, mantendo título/descrição do site e canonical unificado. `home` guarda o nome da pasta; `logo_link` controla o destino do logo em todos os modelos. Guias de agentes ficam em `ContentDir()`: `INSTRUCTIONS.md` contém o texto por modelo/cofre, e três arquivos de entrada apontam para ele; texto editado pela pessoa é preservado.

A interface troca de idioma sem remontar a aplicação: `i18n-live.svelte.ts` fornece estado reativo e `i18n.ts` atualiza os grupos existentes. `Studio.SetLanguage` salva a escolha e atualiza a bandeja. O idioma do site e o das notas traduzidas são estados separados.

## Prévia e publicação

```mermaid
sequenceDiagram
  participant E as Editor
  participant A as Go internal/app
  participant H as Hugo
  participant P as Provedor
  E->>A: Salvar Markdown e propriedades
  A->>H: Mounts das notas e tema embutido
  H-->>E: Previa local por iframe
  E->>A: Publicar
  A->>H: Build de producao na URL final
  H-->>A: HTML e manifestos
  A->>A: Feedback, cifra de posts, Pagefind e PWA
  A->>P: Enviar somente saida estatica
  P-->>E: URL publicada
```

O scaffold substitui a cópia do tema e as configurações geradas, preservando diretórios de overrides. Hugo monta as notas, sem duplicar a origem editável. Na prévia inclui rascunhos e conteúdo `publish: false`/`private: true`, revelando o corpo protegido com aviso. `PreviewGuard` expõe proxy loopback em porta aleatória, valida Host/Origin e mantém Hugo em outra porta interna. No publish, o pós-processamento Go incorpora anexos locais pequenos no corpo protegido antes de cifrá-lo, retira manifestos internos, poda mídias sem referência, indexa busca e gera PWA. Falha na proteção apaga a saída, inclusive quando há marcadores sem manifesto válido. Somente Hugo não executa todas essas etapas.

Adicionar, remover ou restaurar seções pode exigir reiniciar o processo Hugo da prévia: o watcher pode perder arquivos copiados antes de observar uma pasta nova. `sectionsChanged` trata esse caso; `previewFollows` trata mudança de idiomas/mounts. O Studio aberto não precisa reiniciar.

GitHub Pages recebe blobs/árvores e Cloudflare Pages recebe Direct Upload. Blup Cloud usa a sessão da conta, reserva slug e recebe arquivos pelo hash do conteúdo armazenado, compactando recursos apropriados. O manifesto é ativado no D1 depois dos uploads; um Worker separado autoriza cada pedido no D1 e chama um entrypoint com Workers Cache por site/deploy, que resolve o manifesto e lê R2 em miss. O site usa `https://nome.getblup.site/`; Max/Ultra também vinculam um hostname próprio, após prova TXT e validação Cloudflare/SSL. O cliente escolhe domínio ativo com SSL/direito ou carência válida como baseURL. O Studio republica em segundo plano quando essa base muda, sem redirecionar visitantes entre os dois hosts. Reservas D1 limitam admissão concorrente, e leases serializam publicação/limpeza, preservando manifestos substituídos por pelo menos dez minutos.

Publicar é uma ação distinta de sincronizar. Um cofre sincronizado não fica público por isso; publicar o site não é backup das notas. [Publicação](spec/publicacao.md) descreve cache, concorrência e limites dos provedores.

## Sincronização cifrada

```mermaid
sequenceDiagram
  participant S as Studio A
  participant K as Cifra local
  participant W as Worker da conta
  participant R as R2 e D1
  participant B as Studio B
  S->>K: Arquivos e chave aberta no computador
  K-->>S: Blocos cifrados, IDs HMAC e manifesto cifrado
  S->>W: Consultar faltantes e enviar blocos
  W->>R: Guardar por conta
  alt Upload recusado por cota
    S->>S: Guardar blocos cifrados novos em memoria
    S->>W: Prepare: manifesto e tamanhos dos blocos
    W->>R: Lease exclusiva e reserva liquida
    W-->>S: Ticket e blocos faltantes
    S->>W: Enviar faltantes sob ticket e confirmar
  else Upload normal
    S->>W: Commit com versao pai
  end
  W->>R: Head, referencias, uso e fila de descarte atomicos
  W-->>B: WebSocket: ID do cofre e nova versao
  B->>W: Baixar manifesto e blocos
  B->>B: Decifrar, comparar e gravar notas
```

O motor `notesync` compara base, disco e remoto. O adaptador GitHub trabalha com Git Data API; o adaptador Blup Cloud trabalha com blocos de 4 MiB e manifesto cifrado. A chave mestra é aleatória e fica embrulhada por senha e recuperação. A nuvem não recebe esses segredos em claro. Um commit concorrente causa 409 e nova comparação, preservando cópias em conflitos.

Planos pagos têm debounce de dois segundos, avisos por Durable Object com WebSocket Hibernation e verificação a cada cinco minutos. Grátis usa o botão. O histórico tem prazo e rareamento, e consome a mesma cota. A configuração portátil inclui `translations`, identidade, domínio, home Bento, link do logo e cookies; exclui tokens, chave, senha geral, `Deploy` e `Feedback`. O cliente valida a forma do manifesto decifrado, caminhos, tamanhos de até 100 MiB e quantidade de blocos, e limita os bytes ao remontar antes de conferir SHA. Consulte [Blup Cloud](spec/blup-cloud-sync.md) e [notas e sync](spec/notas-e-sync.md).

Storage usa o total da conta, incluindo histórico. Uma redução com excesso abre 14 dias com o teto anterior; depois vale o novo limite, sem apagar dados só por excesso. O cliente tenta o fluxo normal e só prepara após 413, mantendo blocos novos em memória até o commit. O servidor admite substituições sem crescimento líquido e a exceção de versão sem blocos novos; não cria armazenamento gratuito separado para manifestos. Purga manual conserva a versão atual do cofre escolhido e só desconta bytes após confirmação R2. `historyBytes: null` permanece desconhecido no Go e na interface.

## Narrar, ouvir e traduzir

`NoteOrigin` decide o idioma pela pasta imediata da nota traduzida. `Studio.Narrate` busca `Prefs.Voices[lang]`; `voiceFor` preserva a escolha explícita OpenRouter/Blup Cloud ou resolve uma voz local compatível. Sem voz local instalada, pede escolher/baixar, sem fallback automático para envio remoto. O texto falável retira sintaxe não narrável, é dividido em partes, vira MP3 e é unido localmente. A interface grava as propriedades de áudio. Nota com senha é recusada pela narração para não criar uma cópia pública falada.

Kokoro gera no computador. Pela chave própria, a requisição vai ao OpenRouter; pelo plano, vai a `/api/ai/audio/speech` e o Worker usa sua credencial do provedor. O modelo escolhido determina a estimativa por caractere; o servidor reserva cota antes da chamada e prefere custo reportado pelo provedor quando disponível; a cota é exibida em percentual. Esses textos precisam estar em claro para síntese remota. O Podcast agrupa narrações elegíveis em página e feed por idioma, com URLs do endereço de publicação, sem inscrição automática em diretórios de podcast.

O Go exige aceite independente para enviar texto ao OpenRouter ou Blup, em `ai-consent.json`. Narração e tradução remota recusam notas com senha. Tradução local usa Firefox Translations em worker da janela. A preferência explícita `translateAI` escolhe `local`, `openrouter` ou `blup`, sem inferir destino por saldo/chave. Tradução remota usa OpenRouter ou `/api/ai/translate`; os textos traduzidos voltam a arquivos na pasta do idioma. Título, corpo, propriedades textuais e exemplos de Feedback seguem o contrato da tradução. O selo de IA é retirado por revisão explícita. Conversas reais do GitHub são compartilhadas, sem tradução automática. O site publicado escolhe uma tradução existente para o idioma do visitante, com preferência manual lembrada e fallback inglês somente se aquela página tiver inglês.

## Agenda e e-mail

```mermaid
flowchart LR
  Nota[Nota do evento] --> Desktop[Studio verifica a cada 30 segundos]
  Desktop --> SO[Notificacao no computador]
  Desktop --> Lista[ID opaco e horario]
  Lista --> API[PUT api/reminders]
  API --> Fila[D1: pendentes e cota mensal]
  Cron[Cron a cada cinco minutos] --> Fila
  Fila --> Email[Email Service: aviso generico]
```

Eventos são notas em Agenda/Calendar, com repetição e antecedência. Avisos locais precisam do aplicativo em execução. O Studio pode deixar horários futuros na nuvem, que continua verificando com o app fechado. A fila não recebe título ou anotações; Max permite 50 e Ultra 100 e-mails por mês UTC. A preferência global controla as duas formas de aviso. O servidor reivindica itens por lease, reserva cota mensal e admite até três tentativas; só aceitação do provedor aumenta o contador. Desligar avisos ou sair cancela pendentes com repetição após falha de rede. Não há garantia de entrega ou exatamente uma mensagem por evento em várias instalações; veja [calendário](spec/calendario-e-lembretes.md).

## Cobrança e operação

```mermaid
sequenceDiagram
  participant U as Painel da conta
  participant A as API Hono e Better Auth
  participant W as Reconciliador do Worker
  participant T as Stripe
  participant D as D1
  U->>A: Consultar planos e precos disponiveis
  A->>T: Ler precos ativos por lookup key
  A-->>U: Valor e moeda ou periodo indisponivel
  U->>A: Pedir assinatura com e-mail vinculado
  A->>T: Criar cliente e Checkout por lookup key
  T-->>U: Pagina de pagamento
  T->>W: Webhook assinado com ID do evento
  W->>D: Lease por evento e assinatura
  W->>T: Consultar estado atual da assinatura
  W->>D: Batch de estado, evento financeiro e recibo
  W-->>T: Confirmar ou pedir repeticao com 503
  U->>A: Consultar plano e uso
```

O plano resulta da assinatura reconhecida, ou cortesia específica do proprietário, e é conferido nas APIs. Stripe cuida do checkout e portal em suas próprias páginas. O código não armazena números de cartão. `/api/plans` lê valor/moeda do catálogo Stripe, com cache por isolate de dez minutos e último catálogo válido em falha. Período indisponível é `null` e bloqueia upgrade também no servidor; não há valor comercial fixo no plano Go/TypeScript. O script `stripe:setup` apenas confere preços. A cota de IA continua definida separadamente em código. A staff usa Cloudflare Access, valida JWT novamente e aplica papéis `support`/`admin`; ações ficam no registro de auditoria. Não há impersonação do usuário ou interface de leitura de notas.

O webhook é atendido antes do handler Better Auth. Os caminhos webhook/success do plugin estão desativados; o success próprio só redireciona para uma URL local validada. A leitura atual da subscription ocorre sob lease, evitando aplicar um snapshot antigo entregue depois de um evento novo. Outra lease, por conta, serializa a adoção de referência pendente entre assinaturas distintas, sem sobrescrever uma com a outra. O batch D1 exige leases válidas e grava recibo único por `event.id`; falha permite repetição. A migração `0013` cria recibos e índices. A política dos preços antigos usa lookup key ou `metadata.blup_plan` no Price/Product; transferir lookup key não migra uma assinatura existente.

O Worker da conta executa limpeza diária às 04:00 UTC, incluindo purga de contas elegíveis após 30 dias e até 500 recibos de webhook com mais de 90 dias. Esse limite permite atraso na remoção quando há acúmulo. A cada cinco minutos processa lembretes, reservas/tickets expirados, limpeza de sites, descarte de sync, cancelamentos financeiros, avisos de downgrade, desligamento de domínios e a fila de purgas do cache. Publicação de código, migração D1, objetos R2, configuração de e-mail e lançamento do Studio são operações diferentes. As tarefas e os limites de rollback estão em [deploy](deploy.md).

## Domínio, retenção e mudanças de direitos

```mermaid
flowchart TD
  Vincular[Vincular hostname: Max ou Ultra] --> TXT[Gerar prova TXT aleatoria]
  TXT --> Provar[Conferir DNS em _blup.hostname]
  Provar --> CF[Cloudflare Custom Hostname e SSL]
  CF --> Ativo[Servir mesmos arquivos do site]
  Plano[Mudanca com excesso de sites ou storage] --> Prazo[14 dias sem ampliar prazo ja aberto]
  Prazo --> Gratis[Gratis: zero sites depois do prazo]
  Prazo --> Pro[Pro: site escolhido ou mais antigo]
  Prazo --> Espaco[Storage: limite anterior durante a carencia]
  Plano --> Dominio[Perda de dominio: prazo proprio de 14 dias]
  Dominio --> Desligar[Gateway bloqueia e cron remove hostname apos prazo]
  Apagar[Excluir conta] --> Bloquear[Bloquear entrega e revogar sessoes agora]
  Bloquear --> Espera[Reter por 30 dias e tentar cancelar Stripe]
  Espera --> Restaurar[Staff pode restaurar antes da purga]
  Espera --> Purga[Cron diario: lotes e repeticao em falhas]
  Purga --> Remover[Apagar Custom Hostnames antes dos identificadores D1]
```

[lifecycle.ts](../cloud/worker/lifecycle.ts) persiste carências e direitos. O prazo de 14 dias agora cobre reduções com excesso de sites ou storage; Pro escolhe o único site retido em `PUT /api/sites/retained` até o prazo, e Grátis retém zero após ele. Sem escolha, fica o mais antigo por data e ID. Sites excedentes permanecem guardados. O domínio tem `disableAt` e estado `disabled`; Max/Ultra pode cancelar o desligamento antes de ocorrer, e reconfigura depois dele. Storage considera histórico e o teto anterior durante a carência, compartilhando `grace_until` com os sites.
A exclusão lógica agenda purga após 30 dias e cancela sessões/lembretes imediatamente. A staff pode restaurar antes de vencer o prazo e antes de começar a purga, sem recriar assinatura cancelada. Falhas de Stripe, Cloudflare ou R2 mantêm estado para repetição; o cron não promete apagar no segundo exato. [Publicação](spec/publicacao.md) e [segurança](seguranca.md) detalham cópias de navegador e dados externos.

## Entrega pública com gateway e cache

```mermaid
flowchart LR
  Visitante[Visitante no hostname do site] --> Gateway[Default sem cache]
  Gateway --> D1[Conta, deploy, quantidade, dominio e prazo no D1]
  D1 --> Gateway
  Gateway --> Cache[CachedSite por siteId e deploy]
  Cache --> R2[R2 SITES somente no miss]
  Cache --> Resposta[Gateway restaura Cache-Control, ETag e HEAD]
  Resposta --> Visitante
  Eventos[Bloqueio, remocao ou prazo vencido] --> Fila[Fila D1 por site e geracao]
  Fila --> Cron[Cron por SITE_CACHE]
  Cron --> Purga[CachedSite.purgePending no contexto do cache]
  Purga --> Cache
```

O gateway executa em todo pedido e autoriza antes de chamar `ctx.exports.CachedSite({props:{siteId, deploy}})`. O cache interno tem TTL de 600 segundos e tags por site/deploy, sem cookies, Authorization ou query de visitante. O mapeamento de domínio pode estar em memória, mas host exato e prazo são revalidados no D1. A Cache API anterior foi removida. Purgas persistidas têm até cem tags por tentativa e limitador de uma por minuto, executadas pelo cron de cinco minutos no entrypoint dono do cache.

O HTML oferece Link preload dos recursos conhecidos do manifesto, e o provisionamento solicita Early Hints; 103 real depende da infraestrutura. Cache não elimina cobrança por request ou consulta D1 do gateway. [Publicação](spec/publicacao.md#workers-cache-chaves-e-purga) e [deploy](deploy.md) explicam isolamento, custo, operação e validação ainda externa.

## Contratos entre linguagens e release

O painel infere DTOs de `AppType` Hono por `hc`/`InferResponseType`; `worker/export-rpc.mjs` gera um contrato de transporte sem bindings privados do Worker. Zod valida entradas e contratos externos nos pontos consumidos. Wails gera modelos/métodos TS de Go, e os eventos registrados geram payloads tipados para `Events.On`. Arquivos locais/sincronizados recebem normalização e limites próprios. Os wrappers genéricos de GitHub/Cloudflare Pages ainda usam DTOs sem schema semântico estrito; os detalhes e limites de null/opcionais/números estão em [tipagem e contratos](spec/tipagem-e-contratos.md).

A autoatualização CLI verifica Ed25519 dos bytes exatos de `checksums.txt`, depois SHA-256 do pacote, e recusa versão igual/anterior. O gerador `scripts/sign-release` usa chave fora do repositório. Os instaladores iniciais permanecem só com SHA-256. Builds `production` exigem `app.Version` em `x.y.z`; assinatura nativa/notarização e distribuição do Studio têm fluxo separado. Veja [build](spec/testes-e-build.md) e [deploy](deploy.md).

## Referências oficiais consultadas

A separação de módulos segue a [documentação Go](https://go.dev/doc/modules/managing-dependencies); a ponte desktop e distribuição são descritas pelo [Wails v3](https://v3.wails.io/getting-started/installation/). A reatividade segue [Svelte 5](https://svelte.dev/docs/svelte/$state) e o painel estático usa o [adapter-static do SvelteKit](https://svelte.dev/docs/kit/adapter-static). O Worker usa o ambiente documentado por [Hono](https://hono.dev/docs/getting-started/cloudflare-workers), [Drizzle D1](https://orm.drizzle.team/docs/get-started/d1-new), [Zod](https://zod.dev/basics) e [TypeScript](https://www.typescriptlang.org/docs/handbook/intro.html). A configuração concreta da autenticação é a do código, conferida com as referências atuais de [Better Auth](https://better-auth.com/docs/reference/security). Cloudflare e Better Auth também foram consultados pelo MCP Context7 com suas fontes oficiais; este guia não pressupõe que recursos externos já estejam provisionados.


Na rodada 2 também foram conferidos [Hono RPC](https://hono.dev/docs/guides/rpc), [validação Hono](https://hono.dev/docs/guides/validation), [tipagem Better Auth](https://better-auth.com/docs/concepts/typescript), [Ed25519 Go](https://pkg.go.dev/crypto/ed25519), [eventos Wails](https://v3.wails.io/guides/events-reference/) e [tipos SvelteKit](https://svelte.dev/docs/kit/types). MCP Svelte forneceu a documentação de TypeScript/estado; MCP Wails inspecionou o projeto sem executá-lo. Context7 forneceu referências oficiais atuais de Cloudflare for SaaS e Better Auth, complementadas pelos links oficiais de [deploy](deploy.md).


Na rodada 3, Workers Cache foi conferido pelo MCP Context7 e pelas referências oficiais de [chaves](https://developers.cloudflare.com/workers/cache/cache-keys/), [purga](https://developers.cloudflare.com/workers/cache/purge/) e [limites](https://developers.cloudflare.com/workers/cache/limitations/). Early Hints segue a [documentação oficial](https://developers.cloudflare.com/workers/examples/103-early-hints/). As referências descrevem a API, sem comprovar provisionamento ou ganho de custo neste projeto.
