# Blup Cloud: conta, sincronização e planos

Implementação conferida em 02/11/2026. Este documento descreve funções existentes, substituindo o plano inicial. Fontes: [cliente Go](../../internal/blupcloud/client.go), [coordenação](../../internal/app/blup_cloud.go), [criptografia](../../internal/cloudcrypt/cloudcrypt.go), [destino de sync](../../internal/blupcloud/remote.go), [API](../../cloud/worker/index.ts), [sync](../../cloud/worker/sync.ts) e [planos](../../cloud/worker/plans.ts).

## Conta e autorização

`cloud/` contém o painel SvelteKit estático e uma API Hono em Cloudflare Workers. Better Auth usa D1 pelo adaptador Drizzle. `auth-options.ts` configura código de seis dígitos por e-mail, válido por dez minutos, cinco tentativas e armazenamento do código por hash; passkeys; Turnstile; autorização de aparelho; Bearer; e Stripe. Login com senha da conta está desativado. A senha das notas é outro mecanismo, executado no computador.

Uma conta pode nascer só com passkey. O servidor cria um endereço técnico no domínio `.invalid`; não envia mensagens a esse endereço. Vincular e-mail exige confirmar um código. Assinar exige e-mail real; trocar um e-mail já vinculado passa pelo suporte no fluxo atual.

`StartBlupLogin` obtém uma autorização de aparelho para o cliente `blup-studio`; a pessoa confere o código em `/aparelho`, no navegador, e aprova. `FinishBlupLogin` aguarda e guarda a sessão em `credentials.toml`. O Studio usa `Authorization: Bearer`, com identificação da instalação no User-Agent. O painel lista e encerra sessões de aparelhos. O limite de aparelhos é verificado na aprovação; não é uma contagem de cofres.

`LoadBlupAccount` retorna plano, uso e estado das notas: `new`, `locked` ou `ready`. Sair tenta enviar uma lista vazia de lembretes antes de encerrar a sessão e remove `blup_token` e `notes_key` das credenciais. Se o cancelamento não alcançar o servidor, conserva temporariamente a sessão em `signout-pending`, modo `0600`, para repetir o cancelamento no início ou no próximo login. Não apaga as notas no disco. A retenção desse token de trabalho é distinta da sessão usada normalmente pela interface.

## Planos executados

Valores de `cloud/worker/plans.ts`, não uma promessa de preços externos. A API entrega os preços em centavos de real e limites; Stripe encontra os preços pelos lookup keys. Quantidades de armazenamento e tamanho de arquivo são multiplicadas por `1024 * 1024` no servidor.

| Limite | Grátis | Pro | Max | Ultra |
| --- | --- | --- | --- | --- |
| Preço mensal em BRL | 0 | 14,90 | 29,90 | 59,90 |
| Preço anual em BRL | 0 | 149,00 | 299,00 | 599,00 |
| Armazenamento em MiB | 50 | 1024 | 10240 | 51200 |
| Arquivo em MiB | 10 | 25 | 100 | 100 |
| Aparelhos | 1 | 2 | 3 | Sem limite (`-1`) |
| Cofres por conta, teto da API | 50 | 50 | 50 | 50 |
| Sync automático | Não | Sim | Sim | Sim |
| Histórico | Só versão atual | 30 dias | 365 dias | 365 dias |
| Sites hospedados pelo Blup | 0 | 1 | 5 | 10 |
| Multiplicador da cota de IA | 0 | 1 | 2 | 5 |
| E-mails de lembrete por mês | 0 | 0 | 50 | 100 |
| Campo `customDomain` | `false` | `false` | `true` | `true` |

Notas cifradas, manifestos, referências, histórico e arquivos de sites somam no armazenamento da conta. Max e Ultra permitem um domínio próprio por site, com prova TXT de propriedade e validação Cloudflare/SSL antes de servir. O endereço `nome.getblup.site` continua disponível. O domínio serve os mesmos arquivos; o build ainda usa o endereço `getblup.site` nas URLs absolutas. Veja [domínio Blup Cloud](publicacao.md#domínio-próprio-no-blup-cloud). Publicar por Cloudflare Pages ou GitHub Pages continua independente da assinatura.

`account.ts` considera assinaturas `active`, `trialing` e `past_due`; sem uma delas cai no Grátis. O proprietário configurado por secret recebe Ultra de cortesia somente com o e-mail correspondente verificado. Acrescentar alguém à staff não concede esse plano.

## Chaves e dados cifrados

`cloudcrypt.NewKeys` gera chave mestra aleatória de 32 bytes por conta e outra chave aleatória de 32 bytes para recuperação. A senha deriva uma chave de embrulho por Argon2id: três iterações, 64 MiB, quatro threads e salt aleatório de 16 bytes. A recuperação é mostrada uma vez, em grupos base32. Cada uma protege uma cópia da mesma chave mestra com XChaCha20-Poly1305.

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
| R2 `u/<conta>/b/<id>` | Bloco cifrado |
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
| `POST /vaults/:id/versions` | Gravar manifesto e referências com versão pai |
| `GET /live` | WebSocket para planos com `live` |

O upload de bloco aceita até 4 MiB mais 64 bytes; manifesto até 32 MiB. O cliente limita arquivos pelo plano, corta os blocos e usa quatro operações concorrentes. Descompactar um valor cifrado tem limite local de 64 MiB. Uma resposta cifrada é autenticada antes do uso. O cliente valida a forma do manifesto, caminhos, SHA, tamanhos de até 100 MiB e quantidade de blocos antes de alterar estado. Durante a remontagem limita os bytes recebidos ao tamanho declarado, exige igualdade final e confere o SHA do arquivo. O limite lógico do plano também se aplica no envio; o servidor só enxerga blocos cifrados, não o tamanho em claro por arquivo.

`Remote.Commit` cifra o manifesto para `parent + 1`. O servidor exige o mesmo `head` e confirma o avanço por operação condicional no D1. Se outra máquina avançou antes, responde 409 `MOVED`; o adaptador converte para `notesync.ErrNotFastForward` e o motor compara novamente. A conciliação preserva versões conflitantes em arquivos separados, como no destino GitHub.

O adaptador implementa `notesync.Ordered`: depois de sincronizar uma versão, o cliente recusa `head` vazio, não numérico ou inferior ao número conhecido, com `ErrRolledBack`. Isso impede aceitar uma versão antiga ainda autenticamente cifrada naquele computador; sem base local anterior, não fornece uma âncora externa para identificar todo rollback. IDs de cofre têm validação de formato antes de compor rotas/pastas, e leituras HTTP limitam o corpo a 64 MiB.

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

Stripe é configurado pelo plugin Better Auth. O registro Customer é criado no primeiro checkout, sem criar clientes para todo cadastro grátis. O webhook `/api/auth/stripe/webhook` mantém assinaturas; `billing.ts` registra pagamentos, falhas, reembolsos, disputas e alertas de fraude com ID de evento único. O painel oferece checkout e portal por redirecionamento. A exclusão lógica bloqueia a conta, retira os sites da entrega e revoga sessões imediatamente; o cancelamento Stripe é tentado e repetido em segundo plano. A purga é agendada para depois de 30 dias, conforme a seção seguinte. Procedimentos operacionais, Access e secrets estão em [deploy](../deploy.md), e os fluxos completos em [arquitetura](../arquitetura.md).


## Reservas, concorrência e limites de requisição

[quota.ts](../../cloud/worker/quota.ts) mantém reservas D1 de armazenamento, sites, cofres, aparelhos e IA. O teto de criação de cofres é 50 por conta em `sync.ts`, independente de não haver um campo comercial de cofres no plano. Uma operação condicional considera o uso durável e as reservas em andamento antes de admitir trabalho. O prazo padrão da reserva é dez minutos; a confirmação aplica as escritas e libera a reserva em lote D1. Liberação repetida é idempotente, e o cron remove reservas expiradas. Isso fecha as reproduções anteriores de ultrapassar cotas por pedidos simultâneos. Não equivale a uma transação distribuída com Stripe, OpenRouter, e-mail ou R2.

Operações de controle com custo têm fusível de 120 pedidos por minuto por conta e IP (`ABUSE_LIMIT`); transferências têm 500 por dez segundos por conta (`BULK_LIMIT`). A solicitação de OTP mantém seu limite próprio de cinco por minuto por IP. As rotas exatas são selecionadas em `index.ts`; uma rejeição usa 429 e `Retry-After`. Reservas e limites comerciais continuam necessários mesmo quando o fusível ainda permite tráfego.

Uploads de sites usam staging R2, hash incremental e reserva antes de receber o corpo. Com `Content-Length` válido, verificam os bytes efetivamente recebidos e aceitam até o teto do plano, no máximo 100 MiB; sem esse header, o caminho de buffer limita a 4 MiB ou ao teto do plano, se menor. [Publicação](publicacao.md#api-e-armazenamento-dos-sites) explica leases e limpeza de versões anteriores.

## Exclusão, retenção e mudança de plano

[lifecycle.ts](../../cloud/worker/lifecycle.ts) registra `deleted_at` e `purge_after` ao excluir a conta. Sessões são revogadas, aprovações de aparelho deixam de funcionar e lembretes pendentes são cancelados. APIs e Worker público conferem essa condição antes de servir dados ou cache. A staff pode restaurar antes do prazo e antes de a purga começar; a restauração não recria uma assinatura já cancelada nem recupera uma chave de notas perdida.

Após 30 dias, o cron diário seleciona lotes de até 25 contas elegíveis. A purga aguarda a confirmação do cancelamento financeiro e percorre notas R2, sites R2, domínios e registros de produto/autenticação no D1. O Custom Hostname é removido na Cloudflare antes de apagar seus identificadores locais. Falha externa conserva os dados necessários para repetir; o prazo de 30 dias habilita a purga, sem prometer remoção no segundo exato. Cópias locais, conteúdo baixado por visitantes e retenções do próprio provedor de pagamento não são apagados por esse percurso.

Ao passar de pago para Grátis, sites existentes têm prazo de 14 dias e aviso por e-mail. O prazo é persistido quando o sistema observa essa mudança, não recalculado a cada visita. Após expirar, o Worker deixa de servir os sites. Voltar a um plano pago remove esse bloqueio. Max/Ultra para Pro mantém o plano Pro e os sites existentes, com criação e operações sujeitas aos direitos atuais; não dispara o prazo de 14 dias. Domínio próprio deixa de ser servido fora de Max/Ultra, inclusive durante a tolerância do site no endereço `getblup.site`.

## Contratos de tipos

As rotas Hono encadeadas exportam `AppType`. O gerador `worker/export-rpc.mjs` produz `src/lib/rpc.d.ts`, usado por `hc` e `InferResponseType` no painel; o build não importa bindings secretos ou tipos do ambiente Worker para o navegador. Entradas externas ainda precisam de validação em execução. Zod verifica JSON, parâmetros e queries; números de cursores/versões exigem inteiros seguros. Upload binário e multipart usam limites de corpo e validadores próprios, em vez de tratar bytes como JSON.

Erros das APIs de produto têm `code`; validação pode acrescentar `fields`, e os erros específicos de IA mantêm também `error: {code, message}`. Uma `HTTPException` com resposta JSON preserva essa resposta. Exceções textuais são convertidas para JSON: 400 `INVALID`, 413 `TOO_LARGE` e demais `REQUEST_FAILED`; no caminho de IA, a exceção textual usa `INVALID` com `error: {code, message}` e mantém o status HTTP. Better Auth mantém seus próprios contratos. JSON e multipart malformados têm regressões normais em `security.test.ts`, sem `it.fails`. Veja [tipagem e contratos](tipagem-e-contratos.md) para os limites do cliente Go.
