# Tipagem e contratos de dados

Auditoria das rodadas 2, 3 e 4, conferida no código em 02/11/2026. Tipos de compilação descrevem valores esperados; dados vindos de HTTP, arquivos, armazenamento do navegador e sistema operacional exigem verificação em execução. Esta spec registra o estado implementado, incluindo limites residuais.

## Hono, Zod e painel da conta

[index.ts](../../cloud/worker/index.ts) encadeia as rotas de conta, staff, sync, sites, lembretes e IA e exporta `AppType`. [export-rpc.mjs](../../cloud/worker/export-rpc.mjs) usa a API do compilador TypeScript para emitir o contrato de transporte em `cloud/src/lib/rpc.d.ts`; erros de tipos interrompem a geração. O artefato exclui bindings de ambiente e sessão do servidor. `npm run check` e `npm run build` o geram antes de conferir ou compilar o painel.

[api.ts](../../cloud/src/lib/api.ts) usa `hc<AppType>` e `InferResponseType` para derivar conta, plano, limites, aparelhos e sites; `staff.ts` aplica o mesmo contrato às operações da equipe. Não repete esses DTOs principais à mão. Datas serializadas em JSON são strings, sem fingir que o navegador recebeu objetos `Date`. Respostas de erro são analisadas antes de tratar o retorno como sucesso.

[validation.ts](../../cloud/worker/validation.ts) centraliza validação Zod e limites de corpo. IDs opacos, hashes, parâmetros e queries são validados; cursores e versões exigem inteiros seguros, sem aceitar infinito, frações ou conversões ambíguas. Multipart verifica campos como valores de texto e tamanho inteiro, sem converter `File` em string ou campo ausente em zero. Upload binário é um contrato de bytes com tamanho/hash, não um objeto JSON.

| Fronteira | Comportamento |
| --- | --- |
| JSON de IA | Schema Zod para texto, idioma, modelo, voz e formato antes do provedor |
| Idioma da conta | Enum Zod `pt/en/es` compartilhado; `PUT /api/me/language` aceita só `{language}`, com sessão; `GET /api/me` inclui o campo pelo contrato RPC |
| Sites/domínios | Params validados; hostname estrito, normalizado e verificado; listas de hashes e manifestos limitadas |
| Sync | Parâmetros, cursores, multipart, referências e tamanhos validados antes de persistir |
| Staff/aparelhos | IDs e queries validados inclusive no unban; autorização e papel são verificações separadas |
| Catálogo e uso externos | Schemas/ranges para respostas consumidas; preços e custos finitos e não negativos |
| Plano-base e variante | `/me.plan` mantém `name` base, `limits`, `billingName`, `billingInterval: month/year/null` e `pulse:{enabled,limits}`; `/plans` tem seis variantes e preços anuláveis |
| Cobrança própria | Bodies estritos e schemas Zod para Prices, assinatura, prévia e ticket; estado alterado invalida a prévia; `keep` aceita IDs do dono dentro dos limites |
| Pulse e domínio | DTO público separado de eventos privados; POST de domínio só `{hostname}`; TXT e reivindicação atômica de hostname compartilhada com sites |
| D1 tipado por Drizzle | Tipos ajudam no código; parâmetros genéricos e enums TypeScript não validam por si só valores persistidos |

A validação retorna 400 com `code: INVALID`, podendo incluir `fields`. APIs de produto usam `code`; os handlers IA e as rejeições comuns de origem, headers, limite de tráfego e corpo também mantêm `error: {code, message}` no caminho de IA. Fora dele, o tratamento global preserva `HTTPException` com resposta JSON; texto vira JSON com `INVALID` no 400, `TOO_LARGE` no 413 ou `REQUEST_FAILED` nos demais. No caminho IA, resposta JSON com `code` recebe o envelope próprio e exceção textual usa `INVALID`, conservando o status HTTP. Erros internos registram o nome da falha, sem corpo privado do provedor. Better Auth continua com seu contrato próprio de erros. Não há promessa de um único DTO para respostas binárias, WebSocket, autenticação e APIs de produto.

`security.test.ts` verifica JSON/multipart malformados por status, Content-Type e `code`, além de caso específico de `/ai/translate`. As regressões resolvidas são testes normais, sem `it.fails`.

## Go e Wails até o Svelte

Os bindings em `studio/frontend/bindings/` são gerados de structs e métodos Go. Alterar assinatura ou tag JSON exige regenerá-los pelo fluxo de [build](testes-e-build.md#frontend-e-studio), sem editar os arquivos gerados manualmente.

| Origem Go/JSON | Consequência no frontend |
| --- | --- |
| `*T` ou retorno que admite ausência | Tipo gerado pode incluir `null`; a tela precisa tratar a ausência |
| Slice/map nulo | Pode chegar como `null`; usar fallback antes de iterar ou consultar |
| Campo com `omitempty` | Pode faltar; não trocar ausência por presença garantida por um cast |
| `int`, `int64`, contagens e datas numéricas | JSON chega como `number`; fronteiras de rede verificam inteiro seguro/range quando necessário |
| `time.Time` | JSON textual; converter explicitamente quando precisar de data no navegador |
| `map[string]any` de propriedades YAML | Conteúdo dinâmico; ler por guards/helpers, sem assumir texto, lista ou número |

O frontend em `studio/frontend/src` não usa `any` para escapar da checagem; usa tipos das bibliotecas e estreitamento de `unknown`/propriedades dinâmicas. Isso não torna qualquer dado JSON válido por compilação, nem exige remover `any` legitimamente gerado a partir de `any` Go.

[events.go](../../studio/events.go) registra `previewChanged` como string e `calendarOpen` como `{dir, path, date}`; os demais serviços registram seus eventos com o tipo correspondente. `republish` também é registrado como estrutura Go com diretório, estado, URL e erro. `main.ts` importa os bindings de eventos, que tipam `Events.On` e seu `data`. O runtime verifica o tipo na emissão. Dados de notificação retornados pelo sistema são convertidos por `calendarOpenData`, aceitando somente campos de texto. Registro tipado não é um schema de qualquer mensagem `postMessage` recebida do iframe: essa ponte também confere origem/frame e forma dos dados.

## Arquivos locais e sincronizados

| Fonte | Validação/normalização implementada |
| --- | --- |
| `studio/layout.go` | Decodifica Prefs numa cópia; JSON danificado devolve os padrões completos; `SavePrefs` normaliza enums e faixas, preservando recentes |
| `studio/calendar.go` | Registro de lembretes sempre retorna mapa não nulo e descarta instantes inválidos |
| `frontend/src/canvas.ts` | Valida objeto, tipos, IDs e geometria finita; até 20.000 nós/40.000 arestas; remove duplicatas e ligações sem nós válidos; tipos desconhecidos e arestas preservadas são guardados sem renderizar e recolocados ao salvar |
| `frontend/src/draw.ts` | Valida tipos, limites de geometria e pontos; até 20.000 elementos/pontos nos limites definidos; descarta elementos inválidos e normaliza estilos |
| `frontend/src/graph.ts` | Normaliza preferências de mapa do navegador em vez de confiar no JSON |
| `internal/app/database.go` | Normaliza colunas, enums, IDs e referências de views do `.table`, preservando conteúdo válido |
| `internal/app/ai_privacy.go` | JSON inválido não concede aceite para IA, inclusive erro depois de um campo parcialmente decodificado |
| `internal/notesync/state.go` | Rejeita base incompatível, caminhos/SHA/tamanhos inválidos; recomeça sem deduzir exclusões de uma base corrompida |
| `internal/blupcloud/validate.go` | Valida manifesto decifrado, caminhos, SHA, modos, tamanhos e número de blocos antes de alterar estado |
| `internal/blupcloud/client.go` e `live.go` | Verificam presença/ranges de versões, cursor decrescente, identidade/plano e mensagens de versão |
| `studio/studio.go` | Mapa da prévia tem status HTTP, teto de 16 MiB e validação de caminhos/URLs locais, inclusive limites após symlinks |
| `internal/narrate/models.go` | Limita resposta do catálogo e valida preços/ranges antes de oferecer ou estimar modelos |

Canvas e desenho preservam chaves extras compatíveis; normalização pode descartar só os elementos inválidos em vez de recusar o arquivo inteiro. Isso não significa rejeição universal de versões futuras de todo formato. A cifra autentica a origem dos bytes, mas não substitui validar a forma de um manifesto produzido por outro cliente com acesso à chave.

O manifesto aceita arquivos até `notesync.MaxFileSize`, 100 MiB, e quantidade de blocos compatível com blocos de 4 MiB. A remontagem limita bytes recebidos ao tamanho declarado, exige tamanho final igual e verifica SHA. A lista de versões é limitada a 50 por página, exige números positivos em ordem decrescente e datas presentes; um head mais antigo que a base conhecida é recusado. Veja [sync cifrado](blup-cloud-sync.md).

## Contratos acrescentados na rodada 3

| Fronteira | Tipo e validação |
| --- | --- |
| `me.usage` | `storageLimitBytes: number`, `storageGraceUntil: string\|null`, `storageGraceLimitBytes: number\|null`, `historyBytes: number\|null` |
| Go e Wails para histórico | Cliente e `app.BlupAccount` mantêm `HistoryBytes *int64`; o TS gerado conserva `number \| null`, e ausência não vira zero |
| Carência Go | Datas inválidas ou limite ausente/negativo descartam a carência; o Studio recebe `graceUntil`/`graceLimit` e o limite atual |
| Domínio | `disabled` entra no enum e `disableAt` é ISO UTC ou `null`; estado ativo/SSL ativo e direito ou prazo futuro determinam elegibilidade oficial |
| Sites | `GET /api/sites` inclui `downgrade` nullable e distinção `published`/`live`; PUT retido estrito `{siteId}` com propriedade/prazo |
| Preparo | Multipart com pai inteiro, lista de IDs únicos/tamanhos e File cifrado; resposta `{ticket, expiresAt, missing}` |
| Cliente de preparo | Valida ticket antes de caminho/header, faltantes pertencentes à lista enviada e confirmação `n = parent + 1` |
| Purga | Retorno `{freedBytes, storageBytes}` não negativo; escopo de cofre validado no cliente e autorização no servidor |
| Logo | Go/TS validam `logo_link`; entrada portátil inválida é ignorada e o scaffold não a ativa |

O cliente guarda `expiresAt` como texto, sem validar a data para controlar expiração local. O servidor controla a lease e os erros 409/410. Os tipos RPC continuam gerados da cadeia Hono. Veja [protocolo e limites](blup-cloud-sync.md#armazenamento-e-preparação-transacional).

As duas corridas reportadas na revisão foram fechadas: domínio depende do claim retornado pelo D1, e PUT preparado usa estado lido após lease, chave física única por tentativa e CAS de adoção. Confirmação D1 perdida não deve apagar chave possivelmente adotada. Testes com barreiras verificam os cenários, sem afirmar que tipagem estática sozinha resolve concorrência.

## Contratos conferidos na rodada 4

O cliente Go verifica presença separadamente de valor zero. Em `internal/blupcloud/sites.go`, listar sites exige `limit >= -1` e a lista `sites`; cada site precisa de ID opaco, slug válido, URL HTTP(S) com host e sem usuário e contagens não negativas. Criar exige também o slug solicitado. A verificação de nome aceita somente `free`, `yours`, `taken`, `reserved` ou `invalid`; descarta sugestões inválidas e normaliza ausência para lista vazia. A confirmação de deploy exige URL válida.

Em `client.go`, `missing` precisa existir, conter somente IDs do lote e não repeti-los, tanto para sync quanto para publicação. `vaults` ausente é erro, evitando interpretar uma resposta incompleta como ausência de cofres. O embrulho da chave precisa de `by_password`, `by_recovery` e `params.salt` não vazios. Purga exige `freedBytes` e `storageBytes` presentes e não negativos. `reminders.go` exige `limit` e `used` presentes e não negativos; `used` pode superar o limite após downgrade. Resposta inválida não confirma `pushMail`, que tenta novamente após cinco minutos. O ID de `/me` também precisa corresponder ao formato opaco de 1 a 128 letras, dígitos, `_` ou `-`. Zero e listas vazias legítimos continuam aceitos; essas verificações não são um schema obrigatório para todos os campos futuros ou ignorados de cada DTO.

`Preview.svelte` recebe `MessageEvent<unknown>` e usa [previewurl.ts](../../studio/frontend/src/previewurl.ts): aceita `loaded`, `scroll` ou `ready`, caminho local até 2048 caracteres, query vazia ou iniciada em `?` até 4096 e posição finita, com negativo normalizado para zero. Confere origem, `contentWindow` e a marca `_blup` daquele frame antes de alterar estado. Reinício em outra porta recarrega mesmo se o caminho for igual.

`localtranslate.ts` usa [translatereply.ts](../../studio/frontend/src/translatereply.ts): ID inteiro positivo, erro textual ou lista de textos. Exige um texto de resposta para cada item enviado. [fieldvalue.ts](../../studio/frontend/src/fieldvalue.ts) mantém campos numéricos como número finito, ignora entrada inválida e não grava sem mudança. Os testes desses três arquivos cobrem parsers e funções puras; não montam os componentes Svelte nem executam o worker Bergamot completo em Node.

O endpoint público `/config` passou pelo cliente RPC e trata erro HTTP. `/plans` deriva de `AppType` com preços mensais/anuais anuláveis, conforme [Blup Cloud](blup-cloud-sync.md#planos-executados). O catálogo Stripe externo é conferido com Zod. O webhook próprio confere assinatura pelo SDK Stripe e valida por Zod os campos externos usados ao consultar o estado atual; webhook/success do plugin estão desativados. Better Auth mantém os contratos das demais rotas que implementa. `GET /reminders` também retorna `emailMissing: boolean`; o cliente Go aceita ausência como `false` para compatibilidade, sem relaxar presença/ranges de `limit` e `used`. O `check` de cloud inclui o script Node de conferência de preços, além do painel e Worker.

## Limites residuais conhecidos

Os wrappers genéricos de resposta em [GitHub](../../internal/deploy/github/client.go) e [Cloudflare Workers](../../internal/deploy/cfworkers/client.go) usam DTOs Go e `json.Unmarshal`, sem schema semântico estrito para cada resposta. JSON `null` ou campo ausente pode resultar em zero value. Continuam existindo classificação de erro HTTP, checagem de permissões e verificações posteriores nos fluxos. A equipe manteve essa fronteira para outra rodada para não reabrir a integração entregue; não se afirma validação completa de todo JSON de terceiros.

## Referências oficiais

O contrato segue [Hono RPC](https://hono.dev/docs/guides/rpc) e [validação Hono](https://hono.dev/docs/guides/validation), [Zod](https://zod.dev/basics), [TypeScript e narrowing](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html), [JSON Go](https://pkg.go.dev/encoding/json#Unmarshal) e os limites dos tipos [Drizzle SQLite](https://orm.drizzle.team/docs/sqlite/column-types). A ponte desktop foi conferida com [modelos Wails](https://v3.wails.io/features/bindings/models/), [métodos](https://v3.wails.io/features/bindings/methods/) e [eventos](https://v3.wails.io/guides/events-reference/). Também foram consultados os MCPs Svelte, Wails e Context7 para Cloudflare/Better Auth; veja [arquitetura](../arquitetura.md#referências-oficiais-consultadas).
