# Blup Cloud: conta, sincronização e planos

Implementação conferida em 02/11/2026, após os handoffs de contrato da rodada 4. Este documento descreve funções existentes, substituindo o plano inicial. Fontes: [cliente Go](../../internal/blupcloud/client.go), [coordenação](../../internal/app/blup_cloud.go), [criptografia](../../internal/cloudcrypt/cloudcrypt.go), [destino de sync](../../internal/blupcloud/remote.go), [API](../../cloud/worker/index.ts), [sync](../../cloud/worker/sync.ts) e [planos](../../cloud/worker/plans.ts).

## Conta e autorização

`cloud/` contém o painel SvelteKit estático e uma API Hono em Cloudflare Workers. Better Auth usa D1 pelo adaptador Drizzle. `auth-options.ts` configura código de seis dígitos por e-mail, válido por dez minutos, cinco tentativas e armazenamento do código por hash; passkeys; Turnstile; autorização de aparelho; Bearer; e Stripe. Login com senha da conta está desativado. A senha das notas é outro mecanismo, executado no computador.

Uma conta pode nascer só com passkey. O servidor cria um endereço técnico no domínio `.invalid`; não envia mensagens a esse endereço. Vincular e-mail exige confirmar um código. Assinar exige e-mail real; trocar um e-mail já vinculado passa pelo suporte no fluxo atual.

`StartBlupLogin` obtém uma autorização de aparelho para o cliente `blup-studio`; a pessoa confere o código em `/aparelho`, no navegador, e aprova. `FinishBlupLogin` aguarda e guarda a sessão em `credentials.toml`. O Studio usa `Authorization: Bearer`, com identificação da instalação no User-Agent. O painel lista e encerra sessões de aparelhos. O limite de aparelhos é verificado na aprovação; não é uma contagem de cofres.

`LoadBlupAccount` retorna plano, uso e estado das notas: `new`, `locked` ou `ready`. Sair tenta enviar uma lista vazia de lembretes antes de encerrar a sessão e remove `blup_token` e `notes_key` das credenciais. Se o cancelamento não alcançar o servidor, conserva temporariamente a sessão em `signout-pending`, modo `0600`, para repetir o cancelamento no início ou no próximo login. Não apaga as notas no disco. A retenção desse token de trabalho é distinta da sessão usada normalmente pela interface.

O `id` de `/me` identifica a conta para suporte e pode ser copiado nas Configurações do Studio; não é token de acesso. O cliente valida seu formato opaco antes de expor a conta pelos bindings.

No painel Cloud, Perfil mostra ID da conta, copiar e retorno Copiado. A staff vê o ID em destaque no detalhe; na lista, uma conta só com passkey aparece como Sem e-mail (só passkey), com ID curto copiável. `AccountId.svelte` usa o ícone Lucide de copiar e copia o ID inteiro, mesmo quando o texto é abreviado. A ajuda é Em caso de ajuda, informe seu ID. O perfil sem e-mail oferece vinculação com um aviso discreto sobre recibos e recuperação, sem obrigar o vínculo para o trabalho local ou alterar a recuperação cifrada das notas.

## Planos executados

Os limites vêm de `cloud/worker/plans.ts`. Os preços vêm do Stripe, pelos lookup keys mensais e anuais de Pro, Max e Ultra. `plans.ts` não contém valores ou moeda fixos. Quantidades de armazenamento e tamanho de arquivo são multiplicadas por `1024 * 1024` no servidor.

| Limite | Grátis | Pro | Max | Ultra |
| --- | --- | --- | --- | --- |
| Armazenamento em MiB | 50 | 1024 | 5120 | 10240 |
| Arquivo em MiB | 10 | 25 | 50 | 100 |
| Aparelhos | 1 | 2 | 3 | Sem limite (`-1`) |
| Cofres por conta, teto da API | 50 | 50 | 50 | 50 |
| Sync automático | Não | Sim | Sim | Sim |
| Histórico | Só versão atual | 30 dias | 365 dias | 365 dias |
| Sites hospedados pelo Blup | 0 | 1 | 3 | 10 |
| Multiplicador da cota de IA | 0 | 1 | 2 | 5 |
| E-mails de lembrete por mês | 0 | 0 | 50 | 100 |
| Campo `customDomain` | `false` | `false` | `true` | `true` |

`GET /api/plans` retorna objetos `{name, label, limits, perks, suggested, prices}`. `prices.monthly` e `prices.yearly` são `{unitAmount, currency}` ou `null`. `unitAmount` usa a unidade menor da moeda do Stripe, que não é sempre centavos. No Grátis os dois preços são `null`, e a interface mostra Grátis. Nos pagos, `null` indica período indisponível e desabilita a compra. Não existem os antigos campos `priceCents`, `yearlyCents` e `currency` no objeto do plano.

[pricing.ts](../../cloud/worker/pricing.ts) mantém catálogo por isolate por dez minutos, agrupa refreshes concorrentes e espera trinta segundos para repetir uma falha. Falha de transporte/lista mantém o último catálogo válido; sem catálogo anterior, não inventa valor. Uma resposta válida com preço ausente, produto arquivado ou formato não aceito torna aquele período indisponível. [price-schema.ts](../../cloud/worker/price-schema.ts) exige preço recorrente por unidade, intervalo mensal/anual correspondente e produto ativo. O painel formata a moeda com [money.ts](../../cloud/src/lib/money.ts).

O servidor também recusa upgrade de período indisponível com `503 PRICING_UNAVAILABLE`; referência de outra conta retorna 403. `GET /me` permanece com o mesmo contrato de plano/uso e seu `id`; Studio e CLI não consultam `/plans`. A cota de IA definida em código não é recalculada automaticamente quando o preço Stripe muda. O script `npm run stripe:setup` apenas confere o catálogo, sem criar produtos ou preços. Publicação e troca de preços estão no [checklist](../deploy-checklist.md).

`unitAmount` usa unidades menores da moeda Stripe. O painel e os valores financeiros da staff usam [money.ts](../../cloud/src/lib/money.ts), que respeita moedas sem decimais e as exceções ISK/UGX da API; não divide todo valor por 100 nem inventa moeda ausente.

Notas cifradas, manifestos, referências, histórico e arquivos de sites somam no armazenamento da conta. Max e Ultra permitem um domínio próprio por site, com prova TXT de propriedade e validação Cloudflare/SSL antes de servir. O endereço `nome.getblup.site` continua disponível. Os dois hosts servem os mesmos arquivos. Quando domínio e SSL estão ativos e há direito ou carência válida, o cliente o escolhe como baseURL e o Studio republica para ajustar URLs absolutas; não há redirecionamento entre os dois hosts. Veja [domínio Blup Cloud](publicacao.md#domínio-próprio-no-blup-cloud). Publicar por Cloudflare Pages ou GitHub Pages continua independente da assinatura.

`account.ts` considera assinaturas `active`, `trialing` e `past_due`; sem uma delas cai no Grátis. O proprietário configurado por secret recebe Ultra de cortesia somente com o e-mail correspondente verificado. Acrescentar alguém à staff não concede esse plano.

## Chaves e dados cifrados

`cloudcrypt.NewKeys` gera chave mestra aleatória de 32 bytes por conta e outra chave aleatória de 32 bytes para recuperação. A senha deriva uma chave de embrulho por Argon2id: três iterações, 64 MiB, quatro threads e salt aleatório de 16 bytes. A recuperação é mostrada uma vez, em grupos base32. Cada uma protege uma cópia da mesma chave mestra com XChaCha20-Poly1305.

Na criação inicial de `/sync/key`, somente quem inseriu a linha recebe 201; quem perde a corrida recebe `409 KEY_EXISTS`. Atualização do embrulho existente exige preservar `by_recovery`. A migração `0012` remove o backup `previous_key` que não tinha leitor; o servidor conserva o embrulho atual. Não existe fluxo de troca da senha das notas na CLI ou no Studio; a função sem consumidor `cloudcrypt.ChangePassword` foi removida.

O servidor guarda os embrulhos e parâmetros. O cliente abre a chave no computador e guarda sua representação base64 em `credentials.toml`, modo `0600`. Base64 não cifra esse arquivo local. HKDF-SHA256 deriva subchaves distintas para conteúdo, identificadores e manifestos. Sem senha, recuperação ou uma cópia local da chave, a aplicação não oferece recuperação do conteúdo pela staff.

Arquivos são divididos em blocos de até 4 MiB. DEFLATE é aplicado quando reduz o tamanho, seguido de XChaCha20-Poly1305 com nonce aleatório e dados associados. O ID é HMAC-SHA256 do bloco em claro com a subchave de identificação. Blocos iguais na mesma conta podem ser reutilizados; o servidor vê igualdade e tamanho cifrado, mas não recebe um hash público do texto original para testar dicionários de conteúdo.

O manifesto cifrado contém `files`, mapa de caminho para SHA de blob Git, tamanho, modo e lista de blocos. O nome do cofre também é cifrado. O servidor recebe separadamente referências de IDs de blocos em claro para limpeza. Não recebe nomes nem os tamanhos individuais dos arquivos contidos no manifesto, mas observa quantidade de blocos, tamanhos armazenados, versões, horários, conta e tráfego. Veja o [modelo de segurança](../seguranca.md).

## Persistência e protocolo

| Local | Conteúdo |
| --- | --- |
| D1 `sync_account` | Embrulhos da chave, embrulho anterior, uso e datas de sync/limpeza |
| D1 `vault` | Conta, ID, nome cifrado, versão atual e atualização |
| D1 `vault_version` | Cofre, número, chave R2, tamanho e data |
| D1 `sync_block` | Conta, ID HMAC, tamanho cifrado e data |
| D1 `sync_transfer`, `sync_transfer_block` | Ticket, lease, reserva e recibos de blocos preparados |
| D1 `vault_head_block`, `sync_garbage` | Referências atuais e fila de exclusão com bytes ainda contabilizados |
| R2 `u/<conta>/b/<id>-<único>` | Bloco cifrado com chave física em `sync_block.object_key`; a leitura também aceita o layout antigo `b/<id>` |
| R2 `u/<conta>/t/<ticket>/` | Manifesto e referências preparados, plano interno e blocos com chave única por tentativa |
| R2 `u/<conta>/v/<cofre>/<n>-<id>/m` | Manifesto cifrado |
| Mesmo prefixo com `/r` | Referências dos blocos, sem nomes de arquivos |

Rotas autenticadas de `cloud/worker/sync.ts`:

| Rota após `/api/sync` | Operação |
| --- | --- |
| `GET /key`, `PUT /key` | Ler ou criar/atualizar embrulhos; não recebe a senha em claro |
| `GET /vaults`, `POST /vaults` | Listar até 500 cofres ou criar com nome cifrado |
| `GET /vaults/:id` | Ler versão atual de um cofre da conta |
| `GET /vaults/:id/versions?before=N` | Histórico, até 50 versões por resposta |
| `GET /vaults/:id/versions/:n` | Receber manifesto cifrado |
| `POST /blocks/missing` | Consultar até 1000 IDs; consultas D1 em lotes menores |
| `PUT /blocks/:id`, `GET /blocks/:id` | Enviar ou baixar bloco da conta |
| `POST /vaults/:id/versions` | Protocolo anterior de manifesto e referências com versão pai, ainda aceito |
| `POST /vaults/:id/prepare` | Preparar manifesto, referências/tamanhos e reserva líquida; recebe multipart |
| `POST /vaults/:id/commit` | Confirmar `{ticket}`; devolve `{n}` |
| `DELETE /preparations/:ticket` | Cancelar preparo e liberar reserva de forma idempotente |
| `DELETE /vaults/:id/history` | Apagar histórico antigo desse cofre, preservando a versão atual |
| `GET /live` | WebSocket para planos com `live` |

O upload de bloco aceita até 4 MiB mais 64 bytes; manifesto até 32 MiB. O cliente limita arquivos pelo plano, corta os blocos e usa quatro operações concorrentes. Descompactar um valor cifrado tem limite local de 64 MiB. Uma resposta cifrada é autenticada antes do uso. O cliente valida a forma do manifesto, caminhos, SHA, tamanhos de até 100 MiB e quantidade de blocos antes de alterar estado. Durante a remontagem limita os bytes recebidos ao tamanho declarado, exige igualdade final e confere o SHA do arquivo. O limite lógico do plano também se aplica no envio; o servidor só enxerga blocos cifrados, não o tamanho em claro por arquivo.

`Remote.Commit` cifra o manifesto para `parent + 1`. O servidor exige o mesmo `head` e confirma o avanço por operação condicional no D1. Se outra máquina avançou antes, responde 409 `MOVED`; o adaptador converte para `notesync.ErrNotFastForward` e o motor compara novamente. A conciliação preserva versões conflitantes em arquivos separados, como no destino GitHub.

O adaptador implementa `notesync.Ordered`: depois de sincronizar uma versão, o cliente recusa `head` vazio, não numérico ou inferior ao número conhecido, com `ErrRolledBack`. Isso impede aceitar uma versão antiga ainda autenticamente cifrada naquele computador; sem base local anterior, não fornece uma âncora externa para identificar todo rollback. IDs de cofre têm validação de formato antes de compor rotas/pastas, e leituras HTTP limitam o corpo a 64 MiB.

## Armazenamento e preparação transacional

Fontes: [uso](../../cloud/worker/usage.ts), [storage](../../cloud/worker/sync-storage.ts), [preparo](../../cloud/worker/sync-transfers.ts), [purga de histórico](../../cloud/worker/sync-history.ts), [cliente](../../internal/blupcloud/prepare.go) e [adaptador](../../internal/blupcloud/remote.go).

O uso soma blocos cifrados, manifestos/referências, histórico e arquivos dos sites, inclusive bytes aguardando exclusão confirmada. Ao observar uma redução de plano com uso acima do novo limite, `observePlan` abre 14 dias e conserva o teto do plano anterior; não eleva esse teto ao uso já excedente nem prolonga prazo que já está aberto. Sites e storage compartilham `grace_until`; domínio tem prazo próprio. Depois da carência, volta ao teto novo, preservando dados excedentes. Exceder espaço não dispara purga automática; rareamento por `historyDays`, purga manual e exclusão de conta são outras operações.

| Campo de `GET /api/me`, em `usage` | Contrato |
| --- | --- |
| `storageBytes` | Total atual da conta, incluindo histórico |
| `storageLimitBytes` | Limite do plano atual em bytes |
| `storageGraceUntil` | Instante ISO UTC ou `null` |
| `storageGraceLimitBytes` | Teto anterior durante a carência ou `null` |
| `historyBytes` | Total de histórico da conta; `null` quando não pode ser apurado com segurança, por ocupação ou metadados indisponíveis |

O Go mantém `historyBytes` como `*int64`; bindings levam `number | null` ao Studio. Não transforma ausência em zero. A interface mostra tamanho indisponível e oferece atualizar. O total é da conta, sem estimativa por cofre.

O cliente continua o fluxo normal de envio. Só depois de `413 QUOTA` guarda blocos cifrados novos em memória e passa pelo preparo no commit. Acima do teto, a nuvem pode aceitar substituição que não aumenta o uso líquido. O crédito considera apenas versões removíveis pela política de histórico e blocos que nenhuma versão retida/cofre usa. Uma versão que aumenta o saldo líquido continua recusada, salvo a exceção limitada seguinte: sem blocos novos e já acima da cota, o manifesto cifrado pode ter até o tamanho real do manifesto do head no R2 mais 4096 bytes. Isso permite pequenas renomeações e remoções; ultrapassar essa folga retorna `413 QUOTA`. Head existente sem metadados recuperáveis retorna `503 SYNC_METADATA_UNAVAILABLE`. Os bytes efetivos continuam contabilizados. Fora dessa exceção, crescimento de manifesto também reserva armazenamento, mesmo sem blocos novos.

1. `POST /api/sync/vaults/:id/prepare` recebe multipart: `parent` inteiro em texto, `blocks` com JSON `[{id,size}]` de todos os blocos distintos da versão e `manifest` cifrado. Retorna `{ticket, expiresAt, missing}`. Usa o tamanho D1 de blocos existentes; `size: 0` só é válido se o bloco já existe. Um bloco ausente de tamanho zero é inválido.
2. `PUT /api/sync/blocks/:id` recebe o bloco cifrado com `x-blup-upload-ticket`. Exige tamanho igual ao declarado. O corpo permanece limitado a 4 MiB mais 64 bytes.
3. `POST /api/sync/vaults/:id/commit` recebe estritamente `{ticket}` e retorna `{n}`. Repetição de commit concluído devolve o mesmo número enquanto o recibo é mantido. Pai divergente retorna 409 `MOVED` com `head`; exclusividade de preparo/purga retorna 409 `SYNC_BUSY`.
4. `DELETE /api/sync/preparations/:ticket` cancela sem tocar numa versão já confirmada. Ticket cancelado/expirado do dono retorna 410 `TICKET_EXPIRED` ao usar; ticket desconhecido ou alheio retorna 404. Inatividade expira em 30 minutos, renovável até 24 horas desde a criação. Recibos concluídos têm limpeza posterior; não são idempotência perpétua.

O preparo mantém lease exclusiva da conta e reserva o saldo líquido. Uploads normais de blocos diferentes usam leases próprias, compatíveis com as quatro operações concorrentes do cliente, e não passam durante preparo/purga exclusivos. A confirmação D1 une troca de head, referências atuais, remoção lógica, registro de bytes e fila de descarte. Cada tentativa de PUT preparado usa chave R2 única, adotada por CAS `ready=0`; estado é relido depois de obter a lease. Um PUT atrasado não sobrescreve bytes já adotados. Confirmação D1 perdida não autoriza apagar uma chave que pode já estar em uso.

A remoção lógica entra em `sync_garbage`; os bytes só são descontados após R2 confirmar a exclusão. Retirar a fila e descontar ocorre atomicamente, sem reembolso duplo. O cron repete falhas. Não há transação distribuída D1/R2, e uma chamada pode falhar depois de alguma etapa persistir.

### Apagar histórico antigo de um cofre

`DELETE /api/sync/vaults/:id/history`, sem corpo, preserva o head desse cofre e blocos ainda usados por qualquer versão retida da conta. Devolve `{freedBytes, storageBytes}`: bytes efetivamente descontados nesta chamada e uso total da conta. Sem histórico removível, `freedBytes` é zero. Falha R2 deixa descarte pendente e bytes contabilizados; uma repetição não desconta duas vezes.

Exige sessão e propriedade do cofre; cofre alheio retorna 404, preparo/sync exclusivo retorna 409 `SYNC_BUSY`, abuso retorna 429. No Studio, `PurgeHistory` opera somente sobre `sync.vault` do site aberto e aparece quando `sync.remote = "blup"`. O tamanho mostrado é de toda a conta, mas a confirmação explica o escopo por cofre. Não apaga as notas locais, a versão atual ou automaticamente todo histórico da conta.

## Automação, histórico e outra máquina

`internal/app/blup_live.go` agenda sync dois segundos após a última alteração salva; uma alteração durante sync causa outra rodada. Faz sync inicial, ouve a sala da conta no Durable Object `SyncLive` e verifica novamente a cada cinco minutos. Reconecta WebSocket com espera crescente de cinco segundos até dois minutos. O canal transporta somente ID do cofre e número da versão; arquivos passam pela API cifrada. Trocar ou fechar o site encerra a coordenação local correspondente.

`cleanup.ts` e `history.ts` mantêm a versão atual, todas as versões do último dia, uma por hora na última semana e uma por dia depois, dentro do prazo do plano. Blocos sem referência com mais de um dia podem ser removidos. Versões antigas e blocos enviados ainda sem manifesto podem consumir espaço até a limpeza; não há armazenamento gratuito separado para histórico.

`NoteHistory` consulta até 200 versões relevantes de um arquivo; `NoteVersionText` só devolve prévia textual para UTF-8 válido. `RestoreNoteVersion` baixa a versão, sincroniza o estado atual e escreve o arquivo restaurado dentro das notas, validando caminho e links simbólicos. O próximo sync guarda essa restauração como alteração nova; não reduz o número da versão remota. `DeletedNotes` permite localizar arquivos de versões anteriores que ainda não foram restaurados localmente.

`BlupSites` decifra os nomes de cofres com versão publicada no sync e informa quais já estão associados localmente. `BringBlupSite` cria pastas disponíveis, baixa primeiro, aplica `.blup/site.toml` e registra o site. Isso recupera as notas sincronizadas, não os overrides do diretório técnico nem credenciais de outros provedores. O TOML portátil inclui `translations`, mas exclui `Deploy` e `Feedback`: recuperar as notas não importa destinos de publicação nem repositórios de comentários de outra máquina. Arquivos antigos sem `translations` preservam a lista local; lista presente e vazia a limpa.

O destino das notas é `[sync].remote = "blup"`; vazio ou `github` escolhe o destino GitHub das APIs do Studio. Trocar destino não apaga a origem anterior, e a próxima comparação não usa sua base como se pertencesse ao novo destino. O comando interativo `blup sync` ainda implementa o fluxo GitHub em `internal/app/sync.go`.

## IA, lembretes, publicação e cobrança

A IA da conta usa `/api/ai`, com a sessão do Studio. Narração aceita modelos e vozes validados pelo servidor; a estimativa usa o catálogo e a contabilização prefere o custo reportado pelo provedor quando disponível. Tradução monta o prompt no servidor e registra `usage.cost` quando disponível. A cota base é 660.000 milionésimos de dólar por mês, multiplicada pelo plano; o limite da janela móvel de sete dias é metade da cota mensal. O painel mostra percentuais. Antes de enviar texto remoto, o Go exige aceite local separado para `blup` ou `openrouter`, em `ai-consent.json`. Notas protegidas por senha são recusadas pela IA remota. Texto enviado a esses endpoints é legível pelo servidor e pelo provedor de IA, diferentemente do sync. Detalhes em [narração](narracao-e-voz.md) e [Studio](studio.md#idiomas-e-tradução).

Lembretes usam fila D1 e cron a cada cinco minutos, separados do sync cifrado. O cliente envia somente ID opaco e instante; a cota é 50 e-mails no Max e 100 no Ultra, por mês UTC. As condições de atraso, cancelamento e entrega estão em [calendário e lembretes](calendario-e-lembretes.md).

Sites publicados usam bucket `SITES`, com conteúdo público, e outro Worker. Os limites de publicação, cache, manifesto e slug estão em [publicação](publicacao.md#blup-cloud). Isso não publica automaticamente um cofre sincronizado.

Stripe é configurado pelo plugin Better Auth, com SDK 23.1.0 e API `2026-10-28.endive` fixada em [stripe-client.ts](../../cloud/worker/stripe-client.ts). O registro Customer é criado no primeiro checkout, sem criar clientes para todo cadastro grátis. O endpoint `/api/auth/stripe/webhook` é próprio, em `billing.ts`, e verifica assinatura e ambiente test/live. Usa IDs do evento para consultar o estado atual no Stripe; a leitura da subscription ocorre depois da lease por assinatura. Uma lease por conta protege a adoção de referência pendente compartilhada por assinaturas distintas, preservando ambas. Estado, evento financeiro e recibo por `event.id` entram no mesmo batch D1, condicionado às leases. O handler registra pagamentos, falhas, reembolsos, disputas e alertas de fraude. Os caminhos webhook/success do plugin estão desativados; o success próprio só redireciona para URL local, sem gravar direitos. A migração `0013` cria os recibos e índices. Há deduplicação e repetição após erro; recibos ficam elegíveis para remoção após 90 dias em lotes diários de até 500, podendo permanecer mais com acúmulo. O painel oferece checkout e portal por redirecionamento. A exclusão lógica bloqueia a conta, retira os sites da entrega e revoga sessões imediatamente; o cancelamento Stripe é tentado e repetido em segundo plano. A purga é agendada para depois de 30 dias, conforme a seção seguinte. Procedimentos operacionais, Access e secrets estão em [deploy](../deploy.md), e os fluxos completos em [arquitetura](../arquitetura.md).


## Reservas, concorrência e limites de requisição

[quota.ts](../../cloud/worker/quota.ts) mantém reservas D1 de armazenamento, sites, cofres, aparelhos e IA. O teto de criação de cofres é 50 por conta em `sync.ts`, independente de não haver um campo comercial de cofres no plano. Uma operação condicional considera o uso durável e as reservas em andamento antes de admitir trabalho. O prazo padrão da reserva é dez minutos; a confirmação aplica as escritas e libera a reserva em lote D1. Liberação repetida é idempotente, e o cron remove reservas expiradas. Isso fecha as reproduções anteriores de ultrapassar cotas por pedidos simultâneos. Não equivale a uma transação distribuída com Stripe, OpenRouter, e-mail ou R2.

Preparação, purga de histórico e escolha de site retido entram nas operações de controle. Operações de controle com custo têm fusível de 120 pedidos por minuto por conta e IP (`ABUSE_LIMIT`); transferências têm 500 por dez segundos por conta (`BULK_LIMIT`). A solicitação de OTP mantém seu limite próprio de cinco por minuto por IP. As rotas exatas são selecionadas em `index.ts`; uma rejeição usa 429 e `Retry-After`. Reservas e limites comerciais continuam necessários mesmo quando o fusível ainda permite tráfego.

Uploads de sites usam staging R2, hash incremental e reserva antes de receber o corpo. Com `Content-Length` válido, verificam os bytes efetivamente recebidos e aceitam até o teto do plano, no máximo 100 MiB; sem esse header, o caminho de buffer limita a 4 MiB ou ao teto do plano, se menor. [Publicação](publicacao.md#api-e-armazenamento-dos-sites) explica leases e limpeza de versões anteriores.

## Exclusão, retenção e mudança de plano

[lifecycle.ts](../../cloud/worker/lifecycle.ts) registra `deleted_at` e `purge_after` ao excluir a conta. Sessões são revogadas, aprovações de aparelho deixam de funcionar e lembretes pendentes são cancelados. APIs e Worker público conferem essa condição antes de servir dados ou cache. A staff pode restaurar antes do prazo e antes de a purga começar; a restauração não recria uma assinatura já cancelada nem recupera uma chave de notas perdida.

Após 30 dias, o cron diário seleciona lotes de até 25 contas elegíveis. A purga aguarda a confirmação do cancelamento financeiro e percorre notas R2, sites R2, domínios e registros de produto/autenticação no D1. O Custom Hostname é removido na Cloudflare antes de apagar seus identificadores locais. Falha externa conserva os dados necessários para repetir; o prazo de 30 dias habilita a purga, sem prometer remoção no segundo exato. Cópias locais, conteúdo baixado por visitantes e retenções do próprio provedor de pagamento não são apagados por esse percurso.

Uma redução que deixa sites ou storage acima do novo direito abre carência de 14 dias, observada e persistida sem estender prazo já aberto. Durante o prazo, os sites existentes continuam servidos. Depois, Grátis mantém zero sites; Pro mantém um; outros planos mantêm a quantidade permitida. Arquivos e registros excedentes ficam preservados, e uma ampliação de direitos pode voltar a permiti-los. Exclusão/bloqueio da conta prevalece sobre carência.

No downgrade para Pro, `GET /api/sites` acrescenta `downgrade: null | {limit, deadline, selectedSiteId, retainedSiteIds, selectionOpen}`. `deadline` é ISO UTC. `PUT /api/sites/retained` aceita estritamente `{siteId}` e devolve `{ok: true}`; só aceita site do dono, plano com um site e prazo aberto, podendo trocar a escolha até o fim. Sem escolha, retém o mais antigo por `created_at`, com desempate por ID. Após o prazo retorna 409 `SELECTION_CLOSED`; site alheio retorna 404. `published` indica que há deploy guardado; `live` considera o direito de entrega atual.

Sair de Max/Ultra para Pro ou Grátis dá também 14 dias ao domínio próprio existente. `DomainInfo.disableAt` informa ISO UTC ou `null`; durante a carência ele pode permanecer `active` com SSL ativo. Voltar a Max/Ultra antes do desligamento limpa esse prazo. Ao vencer, o gateway bloqueia o hostname e o cron desativa/remove o Custom Hostname, preservando IDs em caso de falha externa. GET passa a `disabled`, SSL `pending`, registros vazios e hostname/prazo preservados. Depois de desligado, voltar ao plano elegível exige POST de reconfiguração e nova prova. O [contrato de domínio e URL oficial](publicacao.md#domínio-próprio-no-blup-cloud) detalha a republicação do Studio.

## Contratos de tipos

As rotas Hono encadeadas exportam `AppType`. O gerador `worker/export-rpc.mjs` produz `src/lib/rpc.d.ts`, usado por `hc` e `InferResponseType` no painel; o build não importa bindings secretos ou tipos do ambiente Worker para o navegador. Entradas externas ainda precisam de validação em execução. Zod verifica JSON, parâmetros e queries; números de cursores/versões exigem inteiros seguros. Upload binário e multipart usam limites de corpo e validadores próprios, em vez de tratar bytes como JSON.

Erros das APIs de produto têm `code`; validação pode acrescentar `fields`, e os erros específicos de IA mantêm também `error: {code, message}`. Uma `HTTPException` com resposta JSON preserva essa resposta. Exceções textuais são convertidas para JSON: 400 `INVALID`, 413 `TOO_LARGE` e demais `REQUEST_FAILED`; no caminho de IA, a exceção textual usa `INVALID` com `error: {code, message}` e mantém o status HTTP. Better Auth mantém seus próprios contratos. JSON e multipart malformados têm regressões normais em `security.test.ts`, sem `it.fails`. Veja [tipagem e contratos](tipagem-e-contratos.md) para os limites do cliente Go.
