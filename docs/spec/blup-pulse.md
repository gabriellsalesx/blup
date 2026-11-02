# Blup Pulse

Especificação de produto e proposta técnica, pesquisada e atualizada em 02/11/2026. Nome, preços e decisões comerciais refletem a atualização aprovada pelo dono. O produto ainda está em implementação por fases; este documento não afirma que os contratos já estejam disponíveis em produção. As definições técnicas permanecem como propostas para implementar e validar. Exemplos usam marcadores, sem credenciais reais.

## 1. Produto e MVP

O Blup Pulse reúne erros de aplicações, monitores HTTP, páginas públicas de status e avisos por Discord e Telegram. Serve também para acompanhar Studio, Cloud, sites e futuros aplicativos móveis do próprio Blup.

O primeiro lançamento oferece quatro telas em `pulse.getblup.io`: Projetos, Monitores, Página de status e Avisos. Usa o mesmo login Better Auth e o backend Hono do Blup Cloud. A identidade visual usa o Blup verde-menta `turma-2` e favicon verde, reaproveitando os assets existentes.

A integração de erros usa SDKs oficiais do Sentry com um DSN do Blup. A promessa inicial é compatibilidade com envio de erros nas versões homologadas. Não significa reproduzir toda a plataforma Sentry, seus algoritmos de agrupamento ou sua API administrativa. Tracing, performance, replay, profiling, source maps, arquivos de símbolos, minidumps e anexos ficam fora do MVP. Também ficam fora avisos por e-mail/SMS, SDK próprio, equipes e consultas arbitrárias de logs.

## 2. Planos e limites

O Pulse não é vendido sozinho. O switch **+ Blup Pulse** escolhe uma variante completa do plano, com uma única assinatura por pessoa.

| Recurso | Max + Pulse | Ultra + Pulse |
| --- | --- | --- |
| Nome interno | `max_pulse` | `ultra_pulse` |
| Preço mensal aprovado, BRL | R$ 49,90 | R$ 129,90 |
| Preço anual aprovado, BRL | R$ 499 | R$ 1.299 |
| Direitos do Blup Cloud | Todos do Max | Todos do Ultra |
| Projetos | 3 | 10 |
| Erros aceitos por mês, total da conta | 50.000 | 250.000 |
| Retenção máxima de eventos e contadores | 30 dias | 90 dias |
| Monitores, total da conta | 10 | 50 |
| Menor intervalo de checagem | 5 minutos | 2 minutos |
| Páginas públicas de status | 1 | 3 |
| Domínio próprio nas páginas de status | Não incluído na proposta inicial | Incluído |
| Avisos | Discord e Telegram | Discord e Telegram |

As cotas do Pulse e o mês civil UTC seguem a proposta mantida pelo dono. O anual custa o equivalente a dez mensalidades, cobrado uma vez por ano. Grátis, Pro, Max e Ultra sem o switch não incluem Pulse. Max continua com 5 GiB de notas, arquivos de até 50 MiB, 3 sites, 3 computadores, histórico de 365 dias, tempo real, domínio próprio, IA 2x e 50 lembretes por e-mail. Ultra continua com 10 GiB, arquivos de até 100 MiB e 10 sites, além dos demais direitos atuais. A [spec Cloud](blup-cloud-sync.md#planos-executados) detalha esses direitos.

O armazenamento de eventos é separado da cota de notas e sites. Página de status não consome um site Hugo do plano. O contrato usa mês civil UTC, sem acumular saldo, sem cobrança automática por excedente e sem zerar uso ao trocar de plano ou recriar projeto. Um novo `event_id` aceito consome uma unidade, mesmo quando pertence a um problema já conhecido. Reenvio com o mesmo ID, dentro da janela de deduplicação, não consome outra unidade.

### Proteções técnicas propostas

| Proteção | Proposta inicial |
| --- | --- |
| Evento após descompressão, antes de tratar dados | Até 200 KiB, ou 204.800 bytes |
| Envelope inteiro | Até 1 MiB recebido e 1 MiB descomprimido |
| Itens por envelope | Até 100, com no máximo um `event` |
| Linha de cabeçalho do envelope/item | Até 8 KiB |
| Estrutura JSON | Até 32 níveis; strings/listas e quantidade de chaves limitadas |
| Ocorrências completas por problema | No máximo as últimas 100, já tratadas para privacidade |
| Banco por projeto, incluindo índices | Proposta: 256 MiB no Max + Pulse; 1 GiB no Ultra + Pulse |
| Pico de erros por projeto | Proposta: 60/minuto no Max + Pulse; 300/minuto no Ultra + Pulse |
| Pico de ingestão por IP | Fusível adicional, configurável e independente da cota paga |
| Histórico de checagens detalhadas | 7 dias; depois só resumo diário dentro da retenção do plano |

Esses são limites do Blup, menores que os do Sentry. Precisam constar nos exemplos de integração e passar na matriz de SDKs. Ao alcançar o teto de armazenamento, remover primeiro payloads antigos, preservando contadores dentro do prazo. Se não houver espaço seguro para novos índices/contadores, recusar a admissão e informar capacidade esgotada. Não prometer 100 amostras ou todos os eventos completos quando o teto por projeto já foi alcançado.

## 3. Arquitetura e fronteiras

```mermaid
flowchart LR
  SDK[SDKs Sentry] --> Ingest[ingest.getblup.io]
  Pessoa[Pessoa autenticada] --> Painel[pulse.getblup.io]
  Painel --> API[Hono e Better Auth]
  Ingest --> API
  API --> Indice[D1: indice e admissao]
  API --> Projeto[DO SQLite por projeto]
  Cron[Cron por minuto] --> Indice
  Cron --> HTTP[Checagem HTTP no Worker]
  HTTP --> Projeto
  Cron --> Avisos[Discord ou Telegram]
  Publico[Visitante] --> Pagina[Pagina publica de status]
  Pagina --> Indice
```

A organização prevista pela Lupa mantém API em `cloud/worker/pulse` e painel estático próprio em `cloud/pulse`. O mesmo backend não exige servir o mesmo HTML nos três hosts. O roteamento por hostname precisa separar painel Pulse, painel Cloud e ingestão. O host de ingestão não serve páginas de login nem aceita cookies como autorização. Assets e fallback SPA devem corresponder ao painel pedido, sem capturar as rotas Sentry ou devolver `index.html` em uma falha de API.

D1 guarda identidade, configuração, índices de agendamento e metadados de admissão/entrega. Não recebe o JSON bruto dos erros ou stack traces. Cada projeto tem um Durable Object SQLite, identificado por mapeamento do servidor, que serializa eventos, agrupamento, deduplicação, limites locais e histórico de checagens. Não criar um DO por ocorrência nem por problema.

As esperas de rede dos monitores ocorrem no Worker, fora do DO. O resultado chega ao objeto em lote por projeto. Isso evita manter o DO ativo durante cada timeout HTTP. Alarmes cuidam de expiração e trabalho curto; notificações externas usam uma fila persistente de metadados e o despachante do cron.

SQLite em DO tem teto de 10 GB por objeto no Workers Paid; os tetos propostos do produto ficam abaixo disso. Usar uma nova classe SQLite e uma migração Workers `new_sqlite_classes`, sem alterar a classe `SyncLive`. O schema interno de cada objeto tem versão própria e migrações idempotentes. [Limites oficiais de DO](https://developers.cloudflare.com/durable-objects/platform/limits/), [API SQLite](https://developers.cloudflare.com/durable-objects/api/sqlite-storage-api/).

### Login entre subdomínios

O alinhamento de implementação da Lupa mantém o login central em `cloud.getblup.io`, com cookie restrito a esse host. O painel `pulse.getblup.io` chama a API Cloud com `credentials: "include"`; o navegador envia o cookie ao host Cloud de destino, sem precisar de cookie no domínio pai. Configurar as duas origens exatas em `trustedOrigins`, preservando `HttpOnly`, `Secure` e SameSite apropriado. A configuração está em [auth-options.ts](../../cloud/worker/auth-options.ts) e exige validação por fase antes de ser tratada como entregue.

A API deve responder ao painel Pulse com CORS de origem exata e `Access-Control-Allow-Credentials: true`, nunca origem `*`. As passkeys continuam sendo criadas/usadas na origem Cloud, preservando o `rpID`. Retornos de login para Pulse exigem destino explicitamente permitido, sem aceitar redirects arbitrários. Sair, revogar sessão, bloquear e excluir a conta devem valer nos dois painéis.

Não habilitar `crossSubDomainCookies` para esta integração nem criar um segundo cookie de sessão em Pulse. O host de ingestão continua sem autenticação por cookie, e as páginas públicas ficam em `pulse.getblup.site/<nome>`, separadas do domínio da conta. Testar login central, requisições autenticadas entre origens e logout, incluindo bloqueio de origem não autorizada. [Cookies Better Auth](https://better-auth.com/docs/concepts/cookies), [segurança](https://better-auth.com/docs/reference/security), [passkeys](https://better-auth.com/docs/plugins/passkey).

## 4. Ingestão compatível com Sentry

### DSN e autenticação

Formato de apresentação, com marcadores:

```text
https://<CHAVE_PUBLICA>@ingest.getblup.io/<ID_NUMERICO_DO_PROJETO>
```

O identificador público é uma string decimal positiva, única, limitada a um intervalo compatível com os SDKs homologados. Não usar UUID na posição que SDKs possam exigir como ID numérico. A chave aleatória identifica a permissão de ingestão de um projeto, pode ser revogada/rotacionada e não autoriza ler problemas ou administrar a conta. Não incluir parte secreta no DSN.

Aceitar as três formas documentadas: `X-Sentry-Auth: Sentry sentry_version=7, sentry_key=<CHAVE_PUBLICA>`, parâmetros de query equivalentes e `dsn` no cabeçalho do envelope. Se houver mais de uma, conferir que chave, projeto e destino combinam. Chave ausente, desativada ou incompatível retorna 403, sem revelar se outro projeto existe. O protocolo é versão 7; o campo legado `sentry_secret` não concede poder adicional. [Autenticação Sentry](https://develop.sentry.dev/sdk/foundations/transport/authentication/).

O DSN de um navegador é público por natureza. Ele permite que outra pessoa tente enviar eventos e gastar a cota do projeto. Uma lista de origens ajuda o navegador, mas não autentica clientes nativos e não impede um atacante de forjar `Origin`. A defesa real combina revogação de chave, limites e admissão no servidor. Antes de admitir eventos, consultar o vínculo ativo e o direito da conta no D1; cache ou estado antigo no DO não pode autorizar projeto revogado, conta bloqueada ou assinatura encerrada. Falha nessa consulta não libera ingestão por padrão.

### Endpoints e parser

| Método e caminho em `ingest.getblup.io` | Corpo e resultado |
| --- | --- |
| `OPTIONS /api/:project/envelope/` e `/store/` | Preflight sem exigir sessão da conta |
| `POST /api/:project/envelope/` | Envelope Sentry; sucesso `200` com `{id}` para um evento aceito |
| `POST /api/:project/store/` | Um evento JSON, com o mesmo tratamento e agrupamento |

Envelope é uma sequência de bytes: uma linha JSON de cabeçalho e itens com cabeçalho próprio e payload. O parser deve respeitar `length` em bytes, consumir exatamente o payload e tolerar a ausência da última quebra de linha. Sem `length`, seguir a delimitação por linha definida pelo protocolo. Não usar `split("\n")` para interpretar payload binário, nem tratar envelope como um único JSON. Conferir IDs do envelope/evento e comprimento real, incluindo UTF-8 multibyte. Aceitar ID hexadecimal de 32 caracteres ou UUID com separadores, normalizando para 32; se ambos os locais omitirem o ID, gerar um. IDs presentes e conflitantes retornam 400. [Formato de envelopes](https://develop.sentry.dev/sdk/foundations/envelopes/).

Aceitar `application/x-sentry-envelope` ou ausência de Content-Type. Para compatibilidade do navegador, aceitar também os Content-Types simples documentados, como `text/plain`, sem trocar a interpretação dos bytes por um parser de formulário. `store` aceita o evento JSON, inclusive quando transportado comprimido. Variantes legadas fora dessa matriz só entram depois de uma fixture oficial e um teste de compatibilidade.

| Tipo de item | MVP |
| --- | --- |
| `event` de erro/mensagem | Aceitar campos reconhecidos e validar antes de persistir |
| `session`, `sessions`, `client_report` | Consumir e ignorar; não entram na cota de erros |
| `transaction`, `span`, perfis e métricas | Consumir e ignorar; sem performance ou tracing |
| `replay_event`, `replay_recording` | Consumir e ignorar; nunca armazenar gravações |
| `attachment`, inclusive payload nativo | Consumir e ignorar; sem anexos/minidumps |
| `user_report`, `feedback`, `check_in`, logs e tipos futuros | Ignorar se o enquadramento for válido; não criar recurso implícito |

Tipos de item não são as mesmas categorias usadas nos headers de rate limit. Dentro de `event`, o MVP aceita `type` ausente, `default` ou `error`; outros tipos de telemetria não viram erros só por estarem nesse item. Um envelope só com itens ignorados recebe `200 {}` depois de validado e limitado em bytes, sem incrementar contadores. Item ignorado continua sujeito ao teto do envelope. Enquadramento inválido retorna 400; não responder sucesso para bytes que não foram interpretados com segurança. A API não faz download de source maps, anexos ou URLs presentes no evento. [Tipos e restrições dos itens](https://develop.sentry.dev/sdk/foundations/envelopes/envelope-items/).

### Compressão, CORS e respostas

Suportar primeiro corpo sem compressão, `Content-Encoding: gzip` e `deflate` com estrutura zlib. A descompressão tem limite de saída e prazo antes de montar JSON. Medir bytes recebidos e descomprimidos mesmo sem `Content-Length`. Compressão desconhecida retorna 415; stream inválido retorna 400; excedente retorna 413. Não usar o tamanho comprimido como teto do evento.

O Sentry também documenta `br` e `zstd`. Workers oferece gzip/deflate nas Compression Streams e Brotli em `node:zlib`; isso não comprova suporte a todo decoder/SDK no ambiente escolhido. Se alguma versão homologada emitir outra compressão por padrão, suportá-la com API disponível ou retirar essa versão da promessa de trocar só o DSN. Não acrescentar biblioteca de compressão pouco conhecida para encobrir a diferença. [Compressão Sentry](https://develop.sentry.dev/sdk/foundations/transport/compression/), [Web APIs Workers](https://developers.cloudflare.com/workers/runtime-apis/web-standards/), [zlib Workers](https://developers.cloudflare.com/workers/runtime-apis/nodejs/zlib/).

Na ingestão pública, CORS pode permitir `*` sem credenciais. Expor `Retry-After`, `X-Sentry-Rate-Limits` e `X-Sentry-Error`; permitir os headers usados pelos SDKs, incluindo `Content-Type`, `Content-Encoding` e `X-Sentry-Auth`. Aplicar esses headers também em erros e 429. Separar essa política do CORS autenticado do painel.

| HTTP | Significado |
| --- | --- |
| 200 | Evento persistido, duplicado já conhecido ou apenas itens deliberadamente ignorados |
| 400 | Envelope, JSON, metadados ou autenticação em formato inválido |
| 403 | Chave inválida/revogada ou projeto sem direito de ingestão |
| 413 | Tamanho excedido |
| 415 | Compressão/formato não suportado |
| 429 | Pico, cota mensal ou capacidade de admissão esgotada; incluir espera |
| 503 | Falha transitória antes da confirmação de armazenamento |

Sucesso de erro devolve ID hexadecimal normalizado. Só confirmar depois da transação SQLite; se a resposta se perder, deduplicar o reenvio por projeto e `event_id`. JSON de falha pode incluir `detail` e `X-Sentry-Error` sanitizados, sem ecoar corpo, DSN ou detalhes internos. Os SDKs não garantem reenviar todo evento após falha; não anunciar entrega garantida. [Transporte e respostas Sentry](https://develop.sentry.dev/sdk/foundations/transport/).

### Rate limit que os SDKs entendem

Usar segundos em `Retry-After`, calculados até a liberação do pico ou o próximo mês UTC. Emitir também `X-Sentry-Rate-Limits`, por exemplo:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
X-Sentry-Rate-Limits: 60:error;default:project:blup_burst
```

O formato é `espera:categorias:escopo:motivo`, podendo ter campos adicionais. Categorias vazias significam todas; usar `error;default` para a cota de erros, preservando o entendimento de SDKs que classificam mensagens como `default`. SDKs dão preferência a `X-Sentry-Rate-Limits`; se ele não existir, usam `Retry-After` no 429. O header também pode antecipar bloqueio numa resposta 200. Validar o comportamento por SDK, sem confundir esses headers com os limites da API administrativa do Sentry. [Rate limiting de ingestão](https://develop.sentry.dev/sdk/foundations/transport/rate-limiting/).

Workers Rate Limiting é um fusível rápido, aproximado e por localização Cloudflare. Não serve como contador exato da assinatura. O DO aplica o pico global do projeto e sua admissão local. A cota mensal total precisa de coordenação entre projetos. [Limitação oficial da API de rate limit](https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/).

Proposta para evitar um UPDATE D1 por evento: reservar pequenos lotes, inicialmente 32 unidades, do orçamento mensal da conta. D1 grava concessões idempotentes por projeto/mês, com soma nunca superior ao plano; cada DO importa uma concessão uma só vez e consome os slots em transação com o evento. Lotes ainda não consumidos permanecem reservados até o fim do mês, inclusive se o projeto for apagado. Mostrar aceitos e reservados separadamente. Não liberar uma concessão após timeout sem comprovar que o DO não a adotou. Isso evita duplicar saldo em falhas entre D1 e DO.

## 5. Agrupamento e ciclo dos problemas

O campo Sentry `fingerprint` é uma lista de strings. Valores iguais agrupam ocorrências; `{{ default }}` insere a identidade calculada pelo servidor. O Blup deve aceitar esse mecanismo, mas seu agrupamento padrão será próprio e versionado. Não prometer que os mesmos eventos gerarão os mesmos grupos do serviço Sentry. [Fingerprinting oficial](https://docs.sentry.io/platforms/javascript/enriching-events/fingerprinting/).

Proposta `groupingVersion = 1`:

1. Tratar dados sensíveis antes de construir qualquer chave ou título.
2. Calcular uma base com plataforma, tipo da exceção e frames relevantes normalizados; sem stack, usar mensagem normalizada e limitada. Retirar caminhos pessoais, IDs variáveis e parâmetros de URL.
3. Com fingerprint explícito, preservar ordem e fronteiras dos elementos, expandindo `{{ default }}` com essa base. Limitar a 32 elementos e 256 caracteres por elemento.
4. Aplicar SHA-256 a uma representação canônica que inclua a versão do algoritmo. O DO do projeto separa os grupos entre clientes; ambiente e release são filtros, sem dividir o grupo por padrão.

O problema guarda primeiro/último recebimento, contagem, severidade, título tratado, ambientes/releases resumidos e estado `open`, `resolved` ou `ignored`. Resolver não apaga eventos. Um evento novo reabre um resolvido e pode gerar aviso de regressão; um ID duplicado não reabre. Ignorar mantém contadores, mas suprime avisos até a pessoa reabrir. Não fazer merge automático entre versões do algoritmo.

As últimas 100 ocorrências completas são o evento permitido pela política de privacidade, não uma cópia crua do pedido. Ocorrências adicionais continuam contando; amostras mais antigas saem. Preservar IDs de deduplicação dentro da retenção mesmo quando o payload foi descartado. Essa combinação é limitada também pelo espaço do projeto.

## 6. Modelo de dados proposto

As tabelas abaixo são desenho de schema, não migrações existentes. A próxima numeração D1 deve ser escolhida quando a implementação for feita, preservando a migração de idioma e o histórico atual.

| D1 geral | Campos e índices principais |
| --- | --- |
| `pulse_project` | ID público decimal, `owner_id`, nome, plataforma, ID interno do DO, ativo, versão de configuração, datas; índice por dono |
| `pulse_project_key` | Projeto, hash da chave, prefixo para exibição, ativo/revogado, datas; chave única e vínculo ao projeto |
| `pulse_usage_month`, `pulse_quota_grant` | Conta/mês, limite, reservado; concessões por projeto com ID único e quantidade; admissão condicional |
| `pulse_monitor` | Dono, projeto, nome público separado, URL privada, intervalo, ativo, próximo horário, claim e estado atual; índice por vencimento |
| `pulse_page`, `pulse_page_component` | Dono, slug único, título, componentes, projetos/monitores autorizados, visibilidade e domínio |
| `pulse_incident`, `pulse_incident_update` | Página, texto público, estado, impacto e datas; índice por página/data |
| `pulse_channel` | Dono, tipo, dados de destino protegidos, configuração, estado e último erro sanitizado |
| `pulse_channel_project`, `pulse_alert_outbox` | Vínculos e IDs de problema/monitor; deduplicação, próximo envio, tentativas e estado; índice da fila |
| `pulse_telegram_link` | Hash do código, dono, expiração, chat candidato, consumido; sem token do bot |
| `pulse_page_domain` | Página, hostname, prova de posse, estado/SSL e identificador do provedor; unicidade de hostname verificado |

Credenciais Discord não ficam em texto puro: cifrar com AES-GCM usando chave de serviço em Worker secret, nonce único, identificação de versão da chave e dados associados ao dono/canal. O servidor pode decifrar para enviar; isso não é cifra de ponta a ponta. Token do bot Telegram e segredo de verificação do webhook são secrets globais. DSNs são públicos para envio; guardar hash da chave e mostrar o valor completo na criação/rotação, com recuperação por rotação.

| SQLite de cada DO de projeto | Conteúdo |
| --- | --- |
| `meta` | Versão do schema, configuração, período e revisões aplicadas |
| `quota_grant`, `usage_month` | Concessões importadas e consumo local |
| `issue` | Fingerprint, versão, estado, título tratado, primeiro/último evento e contagem |
| `event_seen` | `event_id` único, grupo e recebimento, para deduplicar sem reter todo payload |
| `event_sample` | Grupo, evento tratado, tamanho e expiração; índice por grupo/data e por expiração |
| `issue_day` | Contadores diários, sem corpo do evento |
| `monitor_check` | Monitor, horário previsto/real, resultado e duração; índice por monitor/data |
| `monitor_day` | Sucessos, falhas, desconhecidos e resumo de latência por dia |
| `alert_pending` | Transições a entregar ao índice de avisos, com ID idempotente |

Contador, deduplicação, grupo e amostra entram na mesma transação SQLite. Não há transação distribuída entre D1 e DO: criação/exclusão usam estados intermediários recuperáveis e concessões idempotentes. Só habilitar ingestão depois do vínculo completo. Exclusão revoga primeiro no D1, depois apaga o objeto e os vínculos com retomada após falha.

### Retenção e limpeza

Usar o horário de recebimento no servidor para expiração, sem permitir que um timestamp futuro do SDK estenda o prazo. Amostras, IDs, contadores, incidentes e resumos operacionais têm expiração compatível com 30/90 dias; não manter histórico de ocorrências em contadores eternos. A contagem exibida representa a janela vigente e desconta os buckets diários expirados. Configuração ativa de projeto/monitor é outra classe de dado, mantida enquanto o recurso existir.

Checagens completas expiram em 7 dias, depois de contabilizadas no resumo diário. Alarmes processam lotes por índice de expiração; o próximo alarme é o menor prazo pendente. Há apenas um alarme por DO e ele pode executar mais de uma vez, então o trabalho deve ser idempotente. O cron diário revisita projetos para recuperar limpeza interrompida. [Alarmes oficiais](https://developers.cloudflare.com/durable-objects/api/alarms/).

Expirados deixam de aparecer nas consultas imediatamente. Atraso na remoção física é uma falha operacional a medir e corrigir, não uma extensão comercial da retenção. O teto de 90 dias não autoriza restaurar dados vencidos por backup: após restauração, reaplicar expiração antes de servir. O dono precisa fechar como o teto será apresentado diante de backups/recuperação do provedor; não afirmar destruição instantânea de todas as cópias.

## 7. Monitores HTTP e páginas de status

Proposta de monitor MVP: `GET` HTTPS em porta 443, nome, projeto, URL, intervalo do plano e ativo/pausado. Resposta 2xx é sucesso. Erro DNS/TLS, timeout e demais status são falha. Não seguir redirects no MVP; a pessoa informa a URL final. Não aceitar cookies, Authorization, headers arbitrários ou corpo de pedido nesta fase. Não armazenar corpo de resposta.

Aplicar timeout de 10 segundos, cancelar leitura e limitar bytes processados. Bloquear usuário/senha na URL, IP literal, localhost, redes privadas, link-local e destinos de metadados. A proteção deve valer na resolução e na conexão efetiva, incluindo DNS rebinding; só uma expressão regular de hostname não resolve SSRF. Homologar a estratégia de saída no runtime real antes de abrir monitores arbitrários, sem invocar produção durante os testes. [Fetch no Workers](https://developers.cloudflare.com/workers/runtime-apis/fetch/).

Um cron `* * * * *` seleciona monitores vencidos; Max fica elegível a cada cinco minutos e Ultra a cada dois. Os horários são UTC. Índice por vencimento e claim com lease impedem execução concorrente do mesmo monitor/horário. Processar em lotes e com concorrência limitada, sem criar uma promessa de checagem no segundo exato. Não repetir milhares de checagens atrasadas para recuperar cron perdido: registrar lacuna e retomar o próximo horário. [Cron Triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/).

Estado definido para a implementação: começar `unknown`, marcar `down` após duas falhas consecutivas e `up` após duas respostas válidas. Avisar apenas transições confirmadas. Atraso da própria infraestrutura do Blup é `unknown`, não prova de indisponibilidade do destino. Mostrar horário da última checagem, lacunas e que o teste parte da infraestrutura Cloudflare, sem prometer múltiplas regiões ou SLA. Disponibilidade usa falhas/sucessos observados, exibindo também cobertura e intervalos sem observação.

A página pública mostra componentes escolhidos, estado atual, resumos diários e incidentes escritos manualmente. Estados do incidente: `investigating`, `identified`, `monitoring`, `resolved`; impacto público: operacional, degradado, interrupção parcial ou indisponível. Erros privados nunca viram incidentes públicos automaticamente. URL monitorada, DSN, stack trace, nomes privados e credenciais não entram no HTML/JSON público.

O endereço público aprovado é `https://pulse.getblup.site/<nome>`, com nome único por página e sem cookie da conta. Domínio próprio Ultra exige prova de posse, unicidade e TLS ativo pelo fluxo Cloudflare for SaaS; reutilizar o desenho de [domínio Cloud](publicacao.md#domínio-próprio-no-blup-cloud), com vínculo para página Pulse e sem consumir sites Hugo. A zona base é `getblup.site`; o hostname público `pulse.getblup.site` deve ser reservado para esse serviço, sem ser oferecido como slug de site Hugo. A configuração de fallback origin/CNAME, prova e certificados fica no procedimento de deploy. Escapar texto de incidentes, sem HTML arbitrário, e limitar cache público para não esconder transições indefinidamente.

## 8. Avisos Discord e Telegram

Disparos iniciais: novo problema, regressão e mudança confirmada de monitor. Incidente manual pode ser anunciado por ação explícita. Repetições do mesmo problema são contadas, com cooldown proposto de 15 minutos por destino; não enviar uma mensagem por ocorrência. Proposta MVP: um Discord e um chat Telegram por projeto, sem automações arbitrárias.

`pulse_alert_outbox` registra apenas referência e estado de entrega. O despachante carrega um resumo já tratado, tenta no máximo cinco vezes com espera crescente e jitter, obedece à espera do provedor e expira avisos antigos. Proposta de validade: 24 horas. Se a mensagem foi aceita e a confirmação local se perdeu, pode haver duplicata; a interface não deve prometer entrega exatamente uma vez ou leitura pelo destinatário.

### Discord

Aceitar só webhook HTTPS de entrada em host/caminho oficial permitido; não fazer fetch em URL arbitrária colada como canal. Enviar com `wait=true`, `content` limitado a 2.000 caracteres e `allowed_mentions: {parse: []}`. Desativar menções e não anexar eventos crus. A URL contém uma credencial e aparece mascarada depois de salva.

Não fixar uma regra universal como cinco mensagens a cada dois segundos. Respeitar `X-RateLimit-Bucket`, `Remaining` e `Reset-After`; no 429, respeitar `Retry-After`/`retry_after`, inclusive valores fracionários e limites globais. Em 401/403/404 persistente, desativar o destino e pedir correção, evitando loops. [Execute Webhook](https://docs.discord.com/developers/resources/webhook#execute-webhook), [rate limits Discord](https://docs.discord.com/developers/topics/rate-limits).

### Telegram

O usuário precisa abrir/iniciar a conversa com o bot. A tela cria código aleatório base64url, de uso único, ligado à conta e com validade proposta de dez minutos; persiste apenas seu hash. O link `https://t.me/BlupPulseBot?start=<CODIGO>` faz chegar `/start <CODIGO>`. O parâmetro aceita até 64 caracteres. O nome definido para o bot é `@BlupPulseBot`; a criação, disponibilidade do nome e configuração no BotFather são tarefas operacionais, não confirmação de um bot já ativo. Vincular primeiro um chat privado; grupos/canais exigem comprovação de permissão e ficam para uma fase posterior. [Deep linking oficial](https://core.telegram.org/bots/features#deep-linking).

O webhook do bot exige `X-Telegram-Bot-Api-Secret-Token` e validação do update. Deduplicar `update_id`; conferir expiração, uso único e remetente/chat; apresentar o chat candidato no painel e exigir confirmação autenticada antes de ativar. IDs de chat são inteiros seguros no protocolo, mas ficam como string decimal no banco/DTO de destino, preservando sinal e evitando conversões descuidadas.

Enviar por `sendMessage`, texto simples de até 4.096 caracteres, sem `parse_mode` para interpretar conteúdo do erro. Validar `ok`, resultado e parâmetros de erro. No 429, usar `parameters.retry_after`; não habilitar broadcasts pagos. Planejar menos de uma mensagem por segundo por chat e cerca de 30 por segundo por bot; grupos têm referência de 20 por minuto. Manter folga e seguir as respostas, pois os limites podem mudar. O token do bot fica em secret e nunca aparece em logs de URL. [Bot API](https://core.telegram.org/bots/api#sendmessage), [webhook](https://core.telegram.org/bots/api#setwebhook), [limites oficiais](https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this).

O limite global do bot precisa de coordenação entre contas. Proposta MVP: um despachante Telegram serializado por lease no D1, com contador global e envio conservador, separado dos DOs dos projetos. Se o volume exigir outro mecanismo, evoluir a fila antes de aumentar os limites comerciais.

## 9. API Hono tipada

Rotas propostas sob `/api/pulse`. Todas exigem sessão, dono e direito do plano, exceto leitura pública e webhooks com sua autenticação própria. IDs, query, body e cursores passam por Zod, com JSON de configuração estrito. O parser binário Sentry é separado do middleware JSON geral de 96 KiB do Cloud; precisa dos limites próprios desta spec. Metadados extras de SDK desconhecidos podem ser descartados com segurança, sem exigir que SDKs só emitam campos que o painel conhece.

Exportar rotas encadeadas como `PulseAppType`/parte de `AppType` e usar `hc` com respostas inferidas no painel. Reaproveitar a geração RPC existente, sem repetir DTOs manualmente. Exceções do transporte Sentry têm o formato compatível da seção 4; a API do painel mantém erro JSON com `code` estável e mensagem traduzida no cliente. [RPC Hono](https://hono.dev/docs/guides/rpc), [contratos do projeto](tipagem-e-contratos.md).

| Método e rota | Contrato resumido |
| --- | --- |
| `GET /api/pulse/usage` | Limites, aceitos/reservados do mês, espaço, recursos e datas; números finitos e inteiros quando aplicável |
| `GET/POST /api/pulse/projects` | Lista paginada; criação estrita `{name, platform}` e DSN na resposta de criação |
| `PATCH/DELETE /api/pulse/projects/:project` | Nome/ativo; exclusão revoga ingestão antes da purga |
| `POST /api/pulse/projects/:project/keys` | Rotaciona ou cria chave; valor completo só na emissão |
| `DELETE /api/pulse/projects/:project/keys/:key` | Revoga chave do projeto |
| `GET /api/pulse/projects/:project/issues` | Filtros `state`, `environment`, `release`, `q`, cursor e limite máximo 50 |
| `GET/PATCH /api/pulse/projects/:project/issues/:issue` | Detalhe; alteração estrita `{state}` |
| `GET /api/pulse/projects/:project/issues/:issue/events` | Amostras paginadas dentro da retenção |
| `GET/POST /api/pulse/monitors` | Lista; criação `{projectId, name, url, intervalMinutes}` |
| `PATCH/DELETE /api/pulse/monitors/:monitor` | Configuração permitida e pausa; índice recuperável |
| `GET /api/pulse/monitors/:monitor/checks` | Janela limitada e cursor; detalhe ou resumo diário conforme idade |
| `GET/POST /api/pulse/pages` | Lista; criação `{title, slug, componentIds}` com recursos do mesmo dono |
| `GET/PATCH/DELETE /api/pulse/pages/:page` | Configuração, publicação/retirada e componentes selecionados |
| `GET/POST /api/pulse/pages/:page/incidents` | Histórico limitado e criação de incidente manual |
| `POST /api/pulse/pages/:page/incidents/:incident/updates` | Atualização pública de texto e estado |
| `GET/POST/DELETE /api/pulse/pages/:page/domain` | Contrato de domínio no plano elegível; DELETE não tem body |
| `GET/POST /api/pulse/channels` | Lista mascarada; cadastro Discord com destino validado |
| `DELETE /api/pulse/channels/:channel` | Desliga canal e cancela pendências ainda não enviadas |
| `POST /api/pulse/channels/:channel/test` | Teste explícito, limitado, sem conteúdo privado |
| `PUT /api/pulse/projects/:project/channels` | Substitui lista estrita de IDs do mesmo dono |
| `POST /api/pulse/telegram/link` | Emite código/link temporário |
| `GET /api/pulse/telegram/link/:link` | Consulta estado do vínculo, sem expor código de novo |
| `POST /api/pulse/telegram/link/:link/confirm` | Confirma o chat proposto e cria o canal |
| `POST /api/pulse/telegram/webhook` | Update do Telegram autenticado por secret, sem sessão do painel |
| `GET /api/pulse/public/:slug` | Só DTO público, usado em `pulse.getblup.site/:slug` |

Datas saem em ISO UTC; ausência explícita usa `null`, listas vazias usam `[]`. Um recurso de outra conta responde como não encontrado. Reservar slugs, limitar nomes/textos, validar cada vínculo de componentes/canais e usar idempotência nas operações que podem ser repetidas. Não aceitar owner, plano, limite ou nome de DO fornecidos pelo cliente como autoridade.

## 10. Telas do painel

| Tela | Conteúdo do MVP |
| --- | --- |
| Projetos | Uso/cota, criar projeto, copiar DSN, rotacionar/revogar chave e exemplos de SDK homologado; lista de problemas, filtros, amostras tratadas e resolver/ignorar |
| Monitores | URL privada, intervalo, estado, última checagem, lacunas e histórico; criar, pausar e apagar |
| Página de status | Título, endereço, componentes públicos, prévia, incidentes manuais e domínio próprio Ultra |
| Avisos | Discord mascarado, vínculo Telegram por código, projetos associados, teste explícito e resultado das entregas |

Sem Pulse na assinatura, mostrar explicação e link para a página de planos Cloud; não criar uma segunda área de cobrança. Reaproveitar idioma da conta/navegador, temas claro/escuro e componentes acessíveis do Cloud. Nenhum erro privado aparece na página pública sem ação explícita. O teste de canal nunca é disparado ao apenas abrir a tela.

## 11. Switch e cobrança Stripe

Catálogo com preços aprovados pelo dono, a criar/conferir no Dashboard Stripe. Os totais são R$ 49,90/mês ou R$ 499/ano no Max + Pulse e R$ 129,90/mês ou R$ 1.299/ano no Ultra + Pulse. Lookup keys seguem a convenção proposta:

| Variante | Lookup key mensal | Lookup key anual |
| --- | --- | --- |
| `max_pulse` | `blup_max_pulse_mensal` | `blup_max_pulse_anual` |
| `ultra_pulse` | `blup_ultra_pulse_mensal` | `blup_ultra_pulse_anual` |

O switch altera a variante no seletor Max/Ultra e exibe o preço total daquela assinatura, não um adicional avulso com cobrança paralela. `max_pulse` herda todos os limites Max; `ultra_pulse`, todos Ultra. Modelar separadamente `basePlan` e direitos `pulse`, mesmo que o identificador comercial seja único. O domínio próprio dos sites Cloud e o das páginas Pulse são direitos distintos: `limits.customDomain` do Max não libera automaticamente `pulse.customDomain`. Adicionar os nomes aos enums, catálogos de idioma, validação de preços, metadados `blup_plan`, reconciliação e cálculos de downgrade, sem deixar variante desconhecida cair no Grátis.

As variantes precisam estar consistentes em [plans.ts](../../cloud/worker/plans.ts), na reconciliação de lookup keys/metadados de [billing.ts](../../cloud/worker/billing.ts) e no [cliente Stripe](../../cloud/worker/stripe-client.ts), cuja API de referência é `2026-10-28.endive`. Esses pontos devem ser validados juntos na implementação. `/plans` continua localizando textos e obtendo preços reais do Stripe; ausência de preço mantém a compra desabilitada. Não copiar valores de Max/Ultra para inventar o total com Pulse.

Ativar Pulse numa assinatura existente deve substituir o item recorrente da própria assinatura, com idempotência e serialização por conta. Fazer prévia da proporcionalidade com o mesmo instante aplicado na atualização, mostrar crédito/débito e exigir confirmação do total. Proposta: `proration_behavior=always_invoice` e `payment_behavior=pending_if_incomplete`, liberando Pulse somente depois de o Stripe aplicar a mudança paga. Tratar autenticação de pagamento e pendência, sem confiar no retorno visual de sucesso. [Proporcionalidade Stripe](https://docs.stripe.com/billing/subscriptions/prorations), [pending updates](https://docs.stripe.com/billing/subscriptions/pending-updates).

O plugin Better Auth oferece troca proporcional e agendamento no fim do período, mas seus padrões não comprovam a política de liberação condicionada a pagamento. Validar integração com o reconciliador próprio do Blup, mantendo os handlers webhook/success concorrentes do plugin desabilitados. A confirmação vem do webhook assinado e retrieve atual sob lease, como no Cloud. Impedir criar outro Checkout quando já existe assinatura aplicável ou troca pendente. [Stripe no Better Auth](https://better-auth.com/docs/plugins/stripe).

Desligar o switch mantém o Pulse até o fim do período pago, conforme decisão do dono. Reduzir Ultra + Pulse para Max + Pulse segue a proposta de agendar a mudança para o fim do período, preservando o que foi pago. Mudança de plano não reinicia o consumo mensal. Antes de reduzir, mostrar projetos/monitores/páginas excedentes e pedir os recursos retidos; não escolher silenciosamente o projeto com os erros mais recentes.

Ao terminar o período pago sem Pulse, pausar monitores e ingestão, sem novos gastos ou avisos. Os dados privados ficam só para leitura por 14 dias e então são apagados, preservando expirações anteriores e o teto de 90 dias: a carência não renova a retenção dos eventos. Páginas públicas mostram indisponibilidade do serviço, sem estado operacional antigo. Reativar dentro desses 14 dias pode retomar os dados ainda válidos, sem ressuscitar expirados. Essa carência do Pulse é independente dos 30 dias de retenção de conta excluída no Cloud. Exclusão de conta continua bloqueando tudo imediatamente; a purga inclui DOs, canais e domínios Pulse.

A cortesia do dono inclui Pulse com os limites de Ultra + Pulse. Modelar esse direito explicitamente ao resolver o plano de cortesia, sem exigir uma assinatura cobrada e sem criar cota ilimitada. O monitoramento dos próprios produtos continua sujeito ao mesmo aceite e à mesma política de privacidade.

## 12. Privacidade e aceite no Studio

Eventos do Pulse não são notas sincronizadas com cifra de ponta a ponta. O servidor precisa ler erro, frames tratados, versão e contadores para agrupar e avisar; Discord/Telegram recebem o resumo que for enviado. Separar essa explicação do [modelo de segurança do sync](../seguranca.md).

No Studio, o envio fica desligado por padrão. Exigir aceite específico, revogável, antes de iniciar SDK, instalar capturadores automáticos ou enviar qualquer evento. Não reaproveitar o aceite de IA. O monitoramento operacional do backend Cloud é outro fluxo, descrito na política de privacidade institucional.

O Studio usa lista fechada de campos: código estável de falha, componente, versão/build, sistema operacional e frames que não contenham caminhos pessoais. Não transmitir corpo/título de notas, nomes de arquivos/cofres, anexos, texto selecionado, prompts, terminal, URLs privadas, credenciais ou mensagens de erro que possam carregar esses valores. Não anexar minidump, screenshot, log ou estado inteiro do aplicativo. No Go, transformar falhas em códigos antes de capturá-las; `beforeSend` e filtro de breadcrumbs são defesa adicional, não autorização para enviar o erro cru.

Na ingestão geral, remover cookies, Authorization, tokens, senhas, query strings sensíveis, IP, dados de usuário e corpos HTTP por padrão. Limitar e tratar recursivamente contexto, breadcrumbs, tags e mensagens. Remover caminhos pessoais de stack e nunca buscar código remoto. SDK configurado com envio padrão de PII desligado ainda exige revisão dos demais campos; scrubbing genérico não detecta todo segredo em texto livre. Testes do Studio devem injetar notas/segredos sentinela e comprovar ausência antes da rede e depois do armazenamento. [Dados coletados em React Native](https://docs.sentry.io/platforms/react-native/data-management/data-collected/), [Flutter](https://docs.sentry.io/platforms/dart/guides/flutter/data-management/data-collected/).

Não registrar bodies de ingestão, URLs de webhook ou token do bot nos logs do Worker. A infraestrutura ainda pode observar IP e metadados de acesso; minimizar retenção e separar métricas agregadas de payload. Não enviar erros internos do próprio pipeline de ingestão para ele de forma recursiva.

## 13. SDKs e critérios de compatibilidade

No lançamento, a promessa comercial cobre sites, servidores e aplicativo de computador. Integração com celular aparece como **em teste**, sem promessa de suporte móvel homologado. Nenhum SDK está homologado só pela pesquisa desta spec. Na implementação, fixar versões e publicar a matriz real, com versão atual estável e pelo menos uma versão anterior que se pretenda suportar.

| SDK oficial | Primeiro teste | Limite da promessa |
| --- | --- | --- |
| [JavaScript browser](https://docs.sentry.io/platforms/javascript/) | `captureException`, mensagem, erro não tratado, CORS, fetch/Beacon e headers de bloqueio | Stack de bundle minificado continua minificada, sem source maps |
| [Node.js](https://docs.sentry.io/platforms/javascript/guides/node/) | Exceção, rejeição, mensagem, gzip e flush no encerramento | Sem tracing, logs ou perfis como produto implícito |
| [Go](https://docs.sentry.io/platforms/go/) | Erro/panic recuperado, mensagem, flush e filtragem antes do envio | Não prometer capturar todo encerramento fatal ou falha nativa |
| [React Native](https://docs.sentry.io/platforms/react-native/), em teste | Erro JavaScript em Android/iOS reais ou simulados e transporte offline | Sem symbolication nativa, source maps, anexos ou minidumps; provar o caminho JSON antes de anunciar |
| [Flutter](https://docs.sentry.io/platforms/dart/guides/flutter/), em teste | Exceção Dart e transporte em Android/iOS, inclusive após reconexão | Sem símbolos/obfuscação resolvidos ou processamento de crash nativo fora do JSON suportado |

No conjunto comum, testar fingerprint explícito e `{{ default }}`, `event_id` repetido, autenticação por header/query/envelope, itens extras ignorados, comprimentos binários, UTF-8, evento acima de 200 KiB, envelope acima do teto, gzip/deflate corrompido e bomba de descompressão. Confirmar com tráfego capturado localmente que 429 e os dois headers fazem o SDK suspender a categoria, inclusive no navegador.

Matriz também deve testar SDK sem tracing/replay e com defaults reais: ignorar um item não pode corromper o erro do mesmo envelope. Ferramentas de build que tentam enviar source maps/símbolos para APIs administrativas do Sentry não se tornam compatíveis por trocar o DSN; desabilitar esses uploads e explicar o limite. Não mostrar um selo geral de suporte mobile antes desses testes.

## 14. Custos e margem estimados

Modelo de 30 dias, com todas as cotas usadas. Os preços de venda abaixo foram aprovados em BRL; as despesas de infraestrutura são estimadas em USD. Os limites incluídos são da conta Cloudflare inteira, compartilhados com Blup Cloud. A simulação não é medição de produção.

Tarifas oficiais consultadas: Workers Paid tem base de US$5/mês, 10 milhões de requests e 30 milhões de ms de CPU incluídos; excedentes de US$0,30/milhão de requests e US$0,02/milhão de ms. Subrequests HTTP de checagem não são requests de entrada cobradas por unidade, mas continuam consumindo CPU e limites de execução. [Workers pricing](https://developers.cloudflare.com/workers/platform/pricing/).

DO Paid inclui 1 milhão de requests, 400.000 GB-s, 25 bilhões de linhas lidas, 50 milhões escritas e 5 GB-mês; excedentes de US$0,15/milhão de requests, US$12,50/milhão de GB-s, US$0,001/milhão de leituras, US$1/milhão de escritas e US$0,20/GB-mês. D1 tem franquias próprias de 25 bilhões de leituras, 50 milhões de escritas e 5 GB; excedentes SQL equivalentes, com armazenamento D1 a US$0,75/GB-mês. Índices e exclusões também pesam nas escritas. [DO pricing](https://developers.cloudflare.com/durable-objects/platform/pricing/), [D1 pricing](https://developers.cloudflare.com/d1/platform/pricing/).

Premissas: um evento por envelope; concessões de 32 unidades, duas escritas D1 por concessão; duas escritas D1 e três SQLite por checagem; oito escritas SQLite por erro, incluindo manutenção média. Resultados são agrupados por projeto/horário, sem chamar um DO por monitor individual. Essas quantidades precisam ser medidas, incluindo índices e poda.

```text
checagens = monitores * 30 * 24 * 60 / intervalo_em_minutos
requests_DO = erros + projetos * 30 * 24 * 60 / intervalo_em_minutos
escritas_D1 = 2 * checagens + 2 * teto(erros / 32)
escritas_DO = 8 * erros + 3 * checagens
custo_USD = (0,30 * erros + 0,15 * requests_DO + escritas_D1 + escritas_DO) / 1.000.000
```

| Carga em 30 dias por conta | Max + Pulse | Ultra + Pulse |
| --- | --- | --- |
| Intervalo mínimo | 5 minutos | 2 minutos |
| Erros recebidos | 50.000 | 250.000 |
| Checagens HTTP | 86.400 | 1.080.000 |
| Requests DO aproximadas, sem painel/alarmes | 75.920 | 466.000 |
| Escritas D1 aproximadas | 175.926 | 2.175.626 |
| Escritas SQLite DO aproximadas | 659.200 | 5.240.000 |
| Custo variável de requests e escritas, sem descontar franquias | US$0,86 | US$7,56 |

O cron continua executando a cada minuto; só seleciona os monitores Ultra vencidos a cada dois minutos. O custo bruto modelado do Ultra é US$7,560526, e o do Max, US$0,861514. Não estão incluídos CPU, armazenamento, leituras, alarmes, avisos, acessos públicos ou extras do painel. Os primeiros clientes podem caber nas franquias compartilhadas; a escala consome excedentes.

### Conversão e tarifas de pagamento

Usar como referência a PTAX de venda de 30/10/2026, R$ 5,4212 por US$1, consultada em 02/11/2026. Ela converte o subtotal modelado para R$ 4,67 no Max e R$ 40,99 no Ultra por mês. A cotação efetiva do pagamento Cloudflare, impostos sobre a operação e spread podem diferir. [Consulta oficial PTAX](https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata/CotacaoDolarDia%28dataCotacao=@dataCotacao%29?%40dataCotacao=%2710-30-2026%27&%24format=json).

Para a simulação Stripe, assumir cartão nacional com tarifa padrão de 3,99% + R$ 0,39 por pagamento e Billing por consumo de 0,7% do volume, totalizando 4,69% mais o valor fixo. No anual, há um pagamento e o fixo é distribuído por 12 meses. Contrato comercial, cartão internacional ou outro método alteram essa conta. [Payments Brasil](https://stripe.com/br/pricing), [Billing Brasil](https://stripe.com/br/billing/pricing).

| Pacote e cobrança | Receita equivalente/mês | Stripe estimado/mês | Requests/escritas Pulse/mês | Saldo parcial/mês |
| --- | --- | --- | --- | --- |
| Max + Pulse mensal | R$ 49,90 | R$ 2,73 | R$ 4,67 | R$ 42,50 |
| Max + Pulse anual, R$ 499 pagos por ano | R$ 41,58 | R$ 1,98 | R$ 4,67 | R$ 34,93 |
| Ultra + Pulse mensal | R$ 129,90 | R$ 6,48 | R$ 40,99 | R$ 82,43 |
| Ultra + Pulse anual, R$ 1.299 pagos por ano | R$ 108,25 | R$ 5,11 | R$ 40,99 | R$ 62,15 |

Esse saldo ainda precisa pagar o plano-base inteiro, os custos Pulse não modelados, impostos e operação. O anual equivale a dez mensalidades, com receita distribuída pelos 12 meses de serviço. Os valores exibidos foram arredondados depois do cálculo.

### Margem incremental do adicional

Para isolar a contribuição do Pulse, a referência de preço-base usada é a do catálogo institucional conferido nesta revisão: Max R$ 29,90/mês ou R$ 299/ano; Ultra R$ 59,90/mês ou R$ 599/ano. Recalcular se os preços-base efetivos do Stripe mudarem. Como há uma única assinatura, a parcela fixa de R$ 0,39 já existe no pagamento do plano-base; o acréscimo de tarifa vem dos 4,69% sobre a diferença de preço.

| Cobrança | Receita adicional/mês | Tarifa adicional/mês | Requests/escritas Pulse/mês | Contribuição parcial do Pulse/mês |
| --- | --- | --- | --- | --- |
| Max mensal | R$ 20,00 | R$ 0,94 | R$ 4,67 | R$ 14,39 |
| Max anual | R$ 16,67 | R$ 0,78 | R$ 4,67 | R$ 11,21 |
| Ultra mensal | R$ 70,00 | R$ 3,28 | R$ 40,99 | R$ 25,73 |
| Ultra anual | R$ 58,33 | R$ 2,74 | R$ 40,99 | R$ 14,61 |

A contribuição parcial incremental corresponde a cerca de 72,0%/67,3% do acréscimo de receita Max mensal/anual e 36,8%/25,0% no Ultra. Ainda faltam CPU, espaço, avisos, páginas públicas e operação. Em cenário de câmbio a R$6 por dólar, sem mudar as outras premissas, os saldos incrementais caem para R$13,89/R$10,72 no Max e R$21,35/R$10,23 no Ultra. Essa sensibilidade serve para acompanhar a margem aprovada, sem criar novos preços por conta própria.

Com eventos únicos de 5 KiB e retenção completa, o corpo sozinho representa cerca de 0,256 GB no Max e 3,84 GB no Ultra após três meses de carga, antes dos índices. No pior caso de 200 KiB, o Ultra poderia receber 153,6 GB em 90 dias se nada fosse podado. As últimas 100 amostras por problema não evitam isso quando cada evento cria um grupo diferente. Tetos por projeto, retenção e poda são obrigatórios.

Com os tetos de banco mantidos na proposta, o total máximo é 768 MiB no Max e 10 GiB no Ultra, antes de recuperação/backups do provedor. Medir os itens ausentes antes de ampliar o piloto. Definir alertas de orçamento, limite de contas, volume da fila e corte operacional para abuso. Discord/Telegram comuns não exigem broadcasts pagos; manter recursos pagos do bot desativados. A carga varia com o número de dias do mês; custos e tarifas reais substituem a simulação.

## 15. Riscos e fases

| Gravidade | Risco | Condição de liberação |
| --- | --- | --- |
| Alto | Conteúdo de notas/credenciais escapar em erro, contexto ou breadcrumb | Aceite separado e teste de ausência de sentinelas antes da rede; tratamento no servidor |
| Alto | Monitores usados para SSRF ou atingir serviços internos | Saída restrita, defesa verificada contra DNS rebinding, timeout e limites |
| Alto | Variante Stripe cair no Grátis, cobrar duas assinaturas ou liberar antes do pagamento | Migração conjunta dos enums/reconciliador, lease por conta e testes de proporcionalidade/pendência |
| Alto | Chave pública ser usada para drenar cota ou criar grupos sem limite | Fusíveis, pico no DO, admissão mensal e teto físico por projeto |
| Médio | CORS ou retorno de login permitir uso indevido da sessão central | Cookie restrito ao host Cloud, origens exatas e testes de sessão/logout/CSRF |
| Médio | Falso alerta por atraso do cron ou mensagem duplicada | Estado desconhecido, confirmação por sequência, deduplicação e limites de retentativa |
| Médio | Promessa ampla de mobile/compressão sem suporte real | Matriz de SDKs com defaults e limites visíveis |
| Médio | Retenção nominal esconder atraso de limpeza ou backups | Expiração nas consultas, alarmes, reconciliação e política de recuperação explícita |

Fase 0: preparar o catálogo e os hosts aprovados, fechar apenas os detalhes operacionais pendentes e produzir fixtures dos SDKs; ensaiar custo/SSRF sem dados reais.

Fase 1, MVP para piloto: ingestão de erros browser/Node/Go, DO/D1 e quotas, Projetos com resolver/ignorar, monitores HTTP, uma implementação simples das páginas e incidentes manuais, Discord/Telegram, aceite Studio e switch com uma assinatura. Os limites de cada variante continuam sendo os da tabela; o piloto pode limitar o número de contas para medir capacidade.

Fase 2: homologar os caminhos JSON de React Native/Flutter, que no lançamento ficam como em teste. O domínio próprio faz parte da oferta Ultra; pode ser validado em etapa própria do piloto, com sua disponibilidade real claramente informada antes de cobrar por ele.

Performance, tracing, replay e source maps não entram automaticamente numa fase posterior: exigem nova decisão de produto, orçamento e spec.

Antes de produção, exigir testes de concorrência de quotas/concessões, deduplicação com resposta perdida, poda e exclusão retomável, alarme repetido, cron sobreposto, fila com 429, vínculo Telegram roubado/expirado, isolamento entre donos e ausência de dados privados no DTO público. Cobrança só em sandbox, provedores de aviso simulados nos testes. Rodar os gates de Cloud e dos módulos Go afetados; esta entrega documental não executa uma homologação inexistente.

## 16. Decisões aprovadas e pendências restantes

As definições abaixo incorporam a atualização do dono e o alinhamento técnico de implementação da Lupa.

| Assunto | Definição fechada |
| --- | --- |
| Produto e planos | Blup Pulse; variantes `max_pulse` e `ultra_pulse`, sempre com todos os direitos do plano-base e uma assinatura por pessoa |
| Hosts | Painel `pulse.getblup.io`, ingestão `ingest.getblup.io`, páginas públicas `pulse.getblup.site/<nome>`; Ultra com domínio próprio via Cloudflare for SaaS |
| Preços | Max + Pulse R$ 49,90/mês ou R$ 499/ano; Ultra + Pulse R$ 129,90/mês ou R$ 1.299/ano; anual equivalente a dez mensalidades |
| Monitoramento | Max a cada 5 minutos; Ultra a cada 2; marcar indisponível após 2 falhas consecutivas |
| Desligamento | Direito mantido até o fim do período pago; depois 14 dias só para leitura e apagamento, sem renovar retenção de eventos |
| Cortesia | O dono recebe Pulse com os limites de Ultra + Pulse |
| Telegram | Nome adotado `@BlupPulseBot`, sujeito à criação/configuração operacional |
| Celular | No lançamento, em teste; promessa comercial cobre sites, servidores e aplicativo de computador |
| Outras definições | Cotas e limites técnicos mantidos como propostos, mês civil UTC, retenção de 30/90 dias e aceite/privacidade do Studio |
| Login e organização | Login central Cloud com cookie restrito ao host, CORS exato, API em `cloud/worker/pulse`, DO por projeto e painel próprio em `cloud/pulse` |

As pendências restantes são operacionais e de homologação:

1. Definir tamanho do piloto, orçamento global e gatilhos de corte/alerta, usando medições de CPU, SQLite, avisos e acessos públicos para completar a margem.
2. Criar e conferir os Prices/lookup keys, configurar DNS, fallback origin/CNAME, Cloudflare for SaaS, certificados e reserva do hostname público. O endereço e os preços estão definidos; o provisionamento precisa ser executado.
3. Criar/configurar o bot adotado e guardar suas credenciais e o segredo de webhook fora do repositório, com responsável operacional e canal de suporte.
4. Finalizar e publicar o texto do aceite e as políticas institucionais, inclusive como backups/recuperação do provedor se relacionam com retenção e apagamento.
5. Publicar a matriz de versões SDK realmente homologadas e validar o fluxo de pagamento, SSO, monitores e exclusão em cada fase. A classificação mobile continua em teste até essa homologação.

Essas pendências não reabrem nome, preços, intervalo Ultra, cortesia ou política de desligamento. O produto continua em implementação por fases; os handoffs de cada fase devem atualizar o contrato efetivamente entregue.
