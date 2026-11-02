# Blup Cloud: conta, sincronização e planos

Implementação conferida em 02/11/2026. Este documento descreve funções existentes, substituindo o plano inicial. Fontes: [cliente Go](../../internal/blupcloud/client.go), [coordenação](../../internal/app/blup_cloud.go), [criptografia](../../internal/cloudcrypt/cloudcrypt.go), [destino de sync](../../internal/blupcloud/remote.go), [API](../../cloud/worker/index.ts), [sync](../../cloud/worker/sync.ts) e [planos](../../cloud/worker/plans.ts).

## Conta e autorização

`cloud/` contém o painel SvelteKit estático e uma API Hono em Cloudflare Workers. Better Auth usa D1 pelo adaptador Drizzle. `auth-options.ts` configura código de seis dígitos por e-mail, válido por dez minutos, cinco tentativas e armazenamento do código por hash; passkeys; Turnstile; autorização de aparelho; Bearer; e Stripe. Login com senha da conta está desativado. A senha das notas é outro mecanismo, executado no computador.

Uma conta pode nascer só com passkey. O servidor cria um endereço técnico no domínio `.invalid`; não envia mensagens a esse endereço. Vincular e-mail exige confirmar um código. Assinar exige e-mail real; trocar um e-mail já vinculado passa pelo suporte no fluxo atual.

`StartBlupLogin` obtém uma autorização de aparelho para o cliente `blup-studio`; a pessoa confere o código em `/aparelho`, no navegador, e aprova. `FinishBlupLogin` aguarda e guarda a sessão em `credentials.toml`. O Studio usa `Authorization: Bearer`, com identificação da instalação no User-Agent. O painel lista e encerra sessões de aparelhos. O limite de aparelhos é verificado na aprovação; não é uma contagem de cofres.

`LoadBlupAccount` retorna plano, uso e estado das notas: `new`, `locked` ou `ready`. Sair encerra a sessão quando possível e apaga `blup_token` e `notes_key` locais. Não apaga os arquivos de notas já existentes no disco.

## Planos executados

Valores de `cloud/worker/plans.ts`, não uma promessa de preços externos. A API entrega os preços em centavos de real e limites; Stripe encontra os preços pelos lookup keys. Quantidades de armazenamento e tamanho de arquivo são multiplicadas por `1024 * 1024` no servidor.

| Limite | Grátis | Pro | Max | Ultra |
| --- | --- | --- | --- | --- |
| Preço mensal em BRL | 0 | 14,90 | 29,90 | 59,90 |
| Preço anual em BRL | 0 | 149,00 | 299,00 | 599,00 |
| Armazenamento em MiB | 50 | 1024 | 10240 | 51200 |
| Arquivo em MiB | 10 | 25 | 200 | 500 |
| Aparelhos | 1 | 2 | 3 | Sem limite (`-1`) |
| Quantidade de cofres na tabela de planos | Sem limite | Sem limite | Sem limite | Sem limite |
| Sync automático | Não | Sim | Sim | Sim |
| Histórico | Só versão atual | 30 dias | 365 dias | 365 dias |
| Sites hospedados pelo Blup | 0 | 1 | 5 | 10 |
| Multiplicador da cota de IA | 0 | 1 | 2 | 5 |
| E-mails de lembrete por mês | 0 | 0 | 50 | 100 |
| Campo `customDomain` | `false` | `false` | `true` | `true` |

Notas cifradas, manifestos, referências, histórico e arquivos de sites somam no armazenamento da conta. O campo `customDomain` nos planos Max e Ultra ainda não corresponde a uma API de domínio próprio dos sites da nuvem; hoje a publicação usa `nome.getblup.site`. Publicar por Cloudflare Pages ou GitHub Pages continua independente da assinatura.

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

O upload de bloco aceita até 4 MiB mais 64 bytes; manifesto até 32 MiB. O cliente limita arquivos pelo plano, corta os blocos e usa quatro operações concorrentes. Descompactar um valor cifrado tem limite local de 64 MiB. Uma resposta cifrada é autenticada antes do uso, e o arquivo remontado é comparado ao SHA esperado.

`Remote.Commit` cifra o manifesto para `parent + 1`. O servidor exige o mesmo `head` e confirma o avanço por operação condicional no D1. Se outra máquina avançou antes, responde 409 `MOVED`; o adaptador converte para `notesync.ErrNotFastForward` e o motor compara novamente. A conciliação preserva versões conflitantes em arquivos separados, como no destino GitHub.

O adaptador implementa `notesync.Ordered`: depois de sincronizar uma versão, o cliente recusa `head` vazio, não numérico ou inferior ao número conhecido, com `ErrRolledBack`. Isso impede aceitar uma versão antiga ainda autenticamente cifrada naquele computador; sem base local anterior, não fornece uma âncora externa para identificar todo rollback. IDs de cofre têm validação de formato antes de compor rotas/pastas, e leituras HTTP limitam o corpo a 64 MiB.

## Automação, histórico e outra máquina

`internal/app/blup_live.go` agenda sync dois segundos após a última alteração salva; uma alteração durante sync causa outra rodada. Faz sync inicial, ouve a sala da conta no Durable Object `SyncLive` e verifica novamente a cada cinco minutos. Reconecta WebSocket com espera crescente de cinco segundos até dois minutos. O canal transporta somente ID do cofre e número da versão; arquivos passam pela API cifrada. Trocar ou fechar o site encerra a coordenação local correspondente.

`cleanup.ts` e `history.ts` mantêm a versão atual, todas as versões do último dia, uma por hora na última semana e uma por dia depois, dentro do prazo do plano. Blocos sem referência com mais de um dia podem ser removidos. Versões antigas e blocos enviados ainda sem manifesto podem consumir espaço até a limpeza; não há armazenamento gratuito separado para histórico.

`NoteHistory` consulta até 200 versões relevantes de um arquivo; `NoteVersionText` só devolve prévia textual para UTF-8 válido. `RestoreNoteVersion` baixa a versão, sincroniza o estado atual e escreve o arquivo restaurado dentro das notas, validando caminho e links simbólicos. O próximo sync guarda essa restauração como alteração nova; não reduz o número da versão remota. `DeletedNotes` permite localizar arquivos de versões anteriores que ainda não foram restaurados localmente.

`BlupSites` decifra os nomes de cofres com versão publicada no sync e informa quais já estão associados localmente. `BringBlupSite` cria pastas disponíveis, baixa primeiro, aplica `.blup/site.toml` e registra o site. Isso recupera as notas sincronizadas, não os overrides do diretório técnico nem credenciais de outros provedores. A lista `translations` não consta de `sharedSite` hoje; não se deve prometer que todos os campos de `blup.toml` viajem nesse arquivo.

O destino das notas é `[sync].remote = "blup"`; vazio ou `github` escolhe o destino GitHub das APIs do Studio. Trocar destino não apaga a origem anterior, e a próxima comparação não usa sua base como se pertencesse ao novo destino. O comando interativo `blup sync` ainda implementa o fluxo GitHub em `internal/app/sync.go`.

## IA, lembretes, publicação e cobrança

A IA da conta usa `/api/ai`, com a sessão do Studio. Narração aceita modelos e vozes validados pelo servidor; o custo do catálogo do modelo determina a redução da cota. Tradução monta o prompt no servidor e registra `usage.cost` quando disponível. A cota base é 660.000 milionésimos de dólar por mês, multiplicada pelo plano; o limite da janela móvel de sete dias é metade da cota mensal. O painel mostra percentuais. Texto enviado a esses endpoints é legível pelo servidor e pelo provedor de IA, diferentemente do sync. Detalhes em [narração](narracao-e-voz.md) e [Studio](studio.md#idiomas-e-tradução).

Lembretes usam fila D1 e cron a cada cinco minutos, separados do sync cifrado. O cliente envia somente ID opaco e instante; a cota é 50 e-mails no Max e 100 no Ultra, por mês UTC. As condições de atraso, cancelamento e entrega estão em [calendário e lembretes](calendario-e-lembretes.md).

Sites publicados usam bucket `SITES`, com conteúdo público, e outro Worker. Os limites de publicação, cache, manifesto e slug estão em [publicação](publicacao.md#blup-cloud). Isso não publica automaticamente um cofre sincronizado.

Stripe é configurado pelo plugin Better Auth. O registro Customer é criado no primeiro checkout, sem criar clientes para todo cadastro grátis. O webhook `/api/auth/stripe/webhook` mantém assinaturas; `billing.ts` registra pagamentos, falhas, reembolsos, disputas e alertas de fraude com ID de evento único. O painel oferece checkout e portal por redirecionamento. Exclusão de conta tenta cancelar assinaturas antes de remover o usuário. Procedimentos operacionais, Access e secrets estão em [deploy](../deploy.md), e os fluxos completos em [arquitetura](../arquitetura.md).
