# Blup Cloud

Conta, sincronização cifrada de notas, histórico, IA, lembretes e publicação dos sites do Blup. Este diretório contém o painel e dois Workers: o da conta e o dos sites públicos. Implementação conferida em 02/11/2026; provisionamento de produção é uma etapa separada.

- [Arquitetura e fluxos](../docs/arquitetura.md).
- [Deploy de produção](../docs/deploy.md), com DNS, D1, R2, Email Service, Stripe, Access, Turnstile e PSL.
- [Modelo de segurança](../docs/seguranca.md).
- [Contrato Blup Cloud e planos](../docs/spec/blup-cloud-sync.md).
- [Publicação](../docs/spec/publicacao.md) e [calendário/lembretes](../docs/spec/calendario-e-lembretes.md).

## Pilha e pastas

| Parte | Implementação |
| --- | --- |
| API | Hono e Zod em `worker/index.ts`, prefixo `/api` |
| Conta | Better Auth, OTP por e-mail, passkey, device authorization e Bearer |
| Persistência | Drizzle/D1, migrações SQL em `migrations/` |
| Sync | `worker/sync.ts`, R2 `NOTES`, `SyncLive` com WebSocket Hibernation |
| Sites | `worker/sites.ts` recebe publicações; `sites-entry.ts`/`serve-site.ts` servem outro Worker, R2 `SITES` |
| IA | `worker/ai.ts`: narração, tradução e consumo da cota |
| Agenda | `worker/reminders.ts`, `reminder-rules.ts` e `email.ts` |
| Cobrança | Plugin Stripe do Better Auth e `worker/billing.ts` |
| Staff | `worker/staff.ts`, Access, papéis e auditoria |
| Painel | `src/`, SvelteKit SPA/adapter-static, Tailwind, Inter local e Lucide |

`wrangler.jsonc` configura conta/painel; `wrangler.sites.jsonc` configura os sites. Os ambientes de produção repetem bindings e vars. D1 usa o mesmo banco nos dois; `NOTES` e `SITES` são buckets diferentes. O institucional `getblup.io` está fora deste diretório.

## Desenvolvimento local

Instale pelo lockfile e prepare uma `.dev.vars` privada a partir de `.dev.vars.example`. Nunca use secrets de produção. Para uma sessão isolada, ajuste `APP_URL` para a origem local escolhida, `PASSKEY_RP_ID` para `localhost`, `EMAIL_MODE=log`, hosts locais e Turnstile de teste. Stripe deve ser sandbox; IA e e-mail pagos são simulados.

```sh
npm ci
npm run build
npx wrangler d1 migrations apply DB --local --persist-to '<pasta-da-sessao>'
npx wrangler dev --port 18820 --persist-to '<pasta-da-sessao>' --test-scheduled
```

Use a mesma pasta de persistência nas migrações e no servidor. Acesse por `localhost`, de acordo com a origem configurada do login. O proxy do Vite e algumas URLs locais do código têm portas padrão próprias; usar somente assets compilados evita depender do servidor Vite. O endereço retornado para sites em `SITES_DOMAIN=localhost` ainda tem a porta de desenvolvimento codificada em `siteURL`; não presuma que uma porta alternativa já seja refletida na resposta da API.

O cron local pode ser invocado pelo endpoint da versão do Wrangler instalada. A documentação Cloudflare atual usa `/cdn-cgi/local/scheduled`; o [guia de deploy](../docs/deploy.md#4-e-mail-e-cron) mostra a chamada. No modo log, OTP e destinatário aparecem no terminal: não compartilhe nem publique esse log. Ao encerrar, remova só os dados temporários da própria sessão.

## Conta, planos e dados

O ID da conta aparece com botão de copiar na tela Conta e no detalhe da staff; ele não é segredo. A ajuda é: “Em caso de ajuda, informe seu ID.” Na lista da staff, contas sem e-mail mostram o ID curto copiável. Uma conta só com passkey recebe o aviso discreto para ligar um e-mail, com acesso ao diálogo existente, sem obrigação.

A conta pode nascer com passkey sem e-mail; um endereço técnico `.invalid` atende ao formato exigido pelo auth. E-mail real vinculado é necessário para assinar. Cadastro grátis não cria Customer no Stripe. O Studio entra por aprovação de aparelho em `/aparelho`; os limites contam sessões marcadas pelo servidor ao resgatar uma aprovação de aparelho; o User-Agent serve apenas como nome de exibição.

Limites, benefícios e lookup keys ficam em `worker/plans.ts`. Os preços exibidos vêm dos Prices ativos do Stripe, inclusive os anuais, por `worker/pricing.ts`. Grátis permite 50 MiB, um aparelho e sync pelo botão. Pro, Max e Ultra têm sync ao vivo, histórico, IA e, respectivamente, um, cinco e dez sites. Notas, histórico e sites compartilham o armazenamento. Max tem 50 e Ultra 100 lembretes por e-mail por mês UTC. A [tabela completa](../docs/spec/blup-cloud-sync.md#planos-executados) detalha os limites. Free/Pro permitem arquivos de 10/25 MiB e Max/Ultra de 100 MiB; o texto comercial usa MB. Cada conta tem no máximo 50 cofres. Max/Ultra podem vincular um domínio próprio por site, junto com o endereço getblup.site.

O sync recebe nomes, blocos e manifestos cifrados pelo computador. A conta guarda embrulhos da chave e metadados, sem a senha de notas. Conteúdo enviado à IA é outro fluxo: precisa ser legível para processamento remoto. Lembretes recebem somente ID opaco e instante; o e-mail é genérico. Sites publicados são públicos e são servidos em `nome.getblup.site` pelo segundo Worker.

## Staff e operação

Em produção, `cloud.getblup.io` atende a conta e `staff.getblup.io` atende a equipe com Access no host inteiro. A API revalida JWT e papel. `STAFF_OWNER_EMAIL` é secret e faz o bootstrap do admin; a cortesia Ultra requer o e-mail correspondente verificado. `STAFF_DEV_EMAIL` só serve ao ambiente local e não deve ser cadastrado em produção.

Cobrança usa preços por lookup key, checkout/portal hospedados pelo Stripe e webhook assinado em `/api/auth/stripe/webhook`. `scripts/stripe-setup.ts` apenas confere, por leitura, os seis lookup keys no ambiente escolhido. Não cria nem altera produtos ou preços. A conta de operação configura sandbox e produção separadamente, com os eventos e permissões descritos em [deploy](../docs/deploy.md#6-stripe-de-produção). Não há segredo Stripe no painel estático.

Cron de cinco minutos processa até 100 lembretes, reservas expiradas, limpeza adiada de publicação, cancelamento de cobrança e avisos de downgrade. Sem e-mail real, o lembrete é marcado como pulado (`sent_at=0`), sem tentativa nem cota; `GET /api/reminders` informa `emailMissing:boolean`. E-mail com falha pode ser tentado até três vezes, em rodadas posteriores; só o envio aceito pelo serviço entra na cota de 50/100 por mês UTC. Isso não garante entrega na caixa de entrada. O cron diário às 04:00 UTC rareia histórico, limpa blocos e registros antigos de lembretes em lotes e executa purgas vencidas de contas.

## Checks e mudanças de esquema

```sh
npx tsc -p worker/tsconfig.json --noEmit
npx vitest run --maxWorkers 1 --no-file-parallelism
npm run check
npm run build
```

Mudança nos plugins de auth pode exigir regenerar `worker/db/auth-schema.ts` com `npm run db:auth-schema`; alterações de esquema geram migração por `npm run db:generate`. Revise os arquivos antes de aplicá-los. Produção usa `--env production` tanto nas migrações quanto na publicação dos dois Workers; os scripts genéricos não o acrescentam. Os passos completos e os nomes de secrets, sem valores, estão em [deploy](../docs/deploy.md).

## Contratos das rodadas 2 e 3

As cotas de criação de sites/cofres, armazenamento compartilhado e IA usam reservas atômicas no D1 antes da operação externa. Falhas devolvem a reserva; reservas abandonadas expiram. O registro de IA usa `usage.cost` finito e não negativo, inclusive em resposta incompleta, antes de validar a tradução. Sem custo informado, usa a estimativa. Tradução tem teto de saída calculado a partir da entrada. Áudio considera o preço do modelo escolhido e o custo informado em cabeçalho, quando houver.

Uma publicação por site obtém uma trava com expiração; a segunda recebe `409 DEPLOY_BUSY`. Manifestos anteriores e arquivos substituídos ficam por pelo menos dez minutos. A limpeza do cron é paginada, então pode terminar depois desse mínimo. Arquivos mantidos por um manifesto ainda válido ficam protegidos. Objetos temporários e arquivos sem registro também são varridos após a espera.

Excluir a conta revoga sessões, bloqueia login e tira os sites do ar imediatamente. Os dados cifrados e o histórico ficam preservados para recuperação por suporte durante 30 dias, por `POST /api/staff/users/:id/restore`. Isso não reinicia cobrança nem restaura sessões antigas. O cancelamento da assinatura é tentado imediatamente e nas rodadas do cron se o Stripe falhar. A partir de 30 dias, o cron diário apaga os prefixos R2, os Custom Hostnames externos e as linhas D1. Falha na limpeza externa mantém a identificação necessária para repetir a operação. Contas cuja purga já começou não podem ser restauradas.

Qualquer redução de plano que deixe sites ou armazenamento acima do novo limite inicia uma carência de 14 dias, sem prolongar um prazo já aberto. Durante a carência, os sites continuam e o armazenamento pode usar até o limite anterior. Depois, Grátis mantém zero sites; Pro mantém um. Os dados e arquivos excedentes são preservados. A exclusão ou o bloqueio de conta prevalecem sobre qualquer prazo.

No downgrade para Pro, o painel e o e-mail pedem qual site deve ficar. `GET /api/sites` inclui `downgrade: null | {limit,deadline,selectedSiteId,retainedSiteIds,selectionOpen}`. `deadline` é ISO UTC. `PUT /api/sites/retained` recebe somente `{siteId}` e responde `{ok:true}`; só o dono pode escolher um site dele, e pode mudar até o fim da carência. Sem escolha, fica o mais antigo, com desempate pelo ID. A política está isolada em `retainedSiteIds`. Fora do prazo retorna `409 SELECTION_CLOSED`; ID de outro dono retorna `404 NOT_FOUND`.

### Armazenamento e preparação da sync

O uso é o total de blocos, manifestos, histórico e sites. `GET /api/me` acrescenta em `usage`: `storageLimitBytes` (novo limite), `storageGraceUntil: string|null` (ISO UTC), `storageGraceLimitBytes: number|null` (limite durante a carência) e `historyBytes: number|null`. Histórico desconhecido por trava ativa ou metadados antigos indisponíveis retorna null, sem inventar zero. O aviso por e-mail informa usado, novo limite e data. Não apagamos dados por excesso de armazenamento.

Depois da carência, uma operação que não aumenta o saldo líquido continua permitida acima da cota. Acima do limite efetivo, a exceção para uma versão sem blocos novos exige que seu manifesto cifrado tenha no máximo o tamanho real do manifesto do head mais 4096 bytes. O tamanho vem do R2, sem confiar em um número do cliente. A folga de 4 KiB comporta pequenas mudanças de nomes e metadados, mas impede aumentar o uso sem limite por manifestos inflados. Uma exclusão que reduz o manifesto passa; exceder essa folga retorna `413 QUOTA`. Abaixo do limite e durante a carência vale a reserva normal, com o limite efetivo aplicável. Todos os bytes aceitos continuam registrados no uso real. Crédito de substituição só inclui versões que a política `historyDays` permite remover e blocos que nenhum cofre ou versão retida usa.

O protocolo antigo continua aceito. Para enviar uma substituição acima da cota:

1. `POST /api/sync/vaults/:id/prepare`, multipart com `parent` inteiro em texto, `blocks` JSON `[{id,size}]` com todos os IDs da versão, e `manifest` File cifrado. Retorna `{ticket,expiresAt,missing}`. IDs não podem repetir. Para blocos existentes, o servidor usa o tamanho do D1, ignorando o declarado; `size:0` é permitido só quando o bloco já existe. Bloco ausente com tamanho zero retorna `400 INVALID`.
2. Enviar os blocos de `missing` por `PUT /api/sync/blocks/:id`, binário, com `x-blup-upload-ticket`. O tamanho recebido precisa coincidir com o preparo.
3. `POST /api/sync/vaults/:id/commit` com somente `{ticket}` retorna `{n}`. Repetir o commit concluído devolve o mesmo número. Pai alterado retorna `409 MOVED` com `head`; outro sync na conta retorna `409 SYNC_BUSY`.
4. Em falha, `DELETE /api/sync/preparations/:ticket` cancela de forma idempotente. Ticket encerrado ou expirado do dono retorna `410 TICKET_EXPIRED` ao usar; desconhecido ou de outra conta retorna `404 NOT_FOUND`. A inatividade expira em 30 minutos, com limite total de 24 horas.

O preparo trava a conta e reserva o saldo líquido no D1. O commit une troca de head, referências atuais, remoção lógica e fila de descarte na mesma transação. Objetos têm chaves físicas únicas, para uma limpeza atrasada não apagar reenvios do mesmo hash. Os bytes só são descontados depois de o R2 confirmar a exclusão; a retirada da fila e o desconto são atômicos, impedindo reembolso duplo. O cron repete descartes com falha.

### Apagar histórico antigo

`DELETE /api/sync/vaults/:id/history`, sem corpo, preserva o head atual do cofre e todos os blocos ainda usados por qualquer versão retida. Retorna `{freedBytes,storageBytes}`, com bytes realmente descontados nesta chamada e uso total atualizado da conta. Segunda chamada sem histórico retorna zero. Só o dono pode usar; sem sessão: 401, cofre alheio: `404 NOT_FOUND`, sync/preparo ativo: `409 SYNC_BUSY`, abuso: `429 TOO_MANY_REQUESTS`. O escopo é por cofre. Falha no R2 mantém os bytes cobrados e o descarte pendente, sem duplicar o desconto na repetição.

### Domínio próprio

Rotas com a sessão do dono: `GET`, `POST` e `DELETE /api/sites/:id/domain`. POST aceita somente `{hostname}` e exige Max/Ultra. GET retorna o objeto abaixo ou `null`; DELETE retorna `{ok:true}`. GET/DELETE continuam disponíveis após downgrade.

```ts
type DomainInfo = {
  hostname: string;
  status: 'pending_ownership' | 'pending' | 'active' | 'error' | 'disabled';
  disableAt: string | null; // ISO UTC
  ssl: 'pending' | 'active' | 'error';
  records: {type: 'TXT' | 'CNAME'; name: string; value: string}[];
};
```

No máximo dez registros, com valor de até 1024 caracteres. Hostname é normalizado por IDNA e não aceita esquema, caminho, porta, IP, localhost, .local, .internal nem getblup.io/getblup.site e seus subdomínios. Antes de criar o Custom Hostname na Cloudflare, o dono publica o TXT aleatório `_blup.<hostname>` retornado pelo servidor. Um pedido sem prova não reserva exclusividade sobre o hostname: só a posse comprovada recebe a restrição única do D1. Isso evita reservar domínios de terceiros sem controlar o DNS. Consultas DNS usam um resolvedor HTTPS fixo, sem buscar URLs do visitante. O conteúdo só é servido quando a posse, a ativação e o SSL estiverem confirmados.

O botão Verificar usa GET. Poll de 60 segundos com a tela aberta é suficiente; o servidor limita consultas de verificação. O CNAME vem na resposta, não deve ser fixado na interface. Apex exige flattening/ALIAS no provedor DNS, sem promessa de IP dedicado. A resolução de domínio tem cache de até 30 segundos; conta, plano, host exato e prazo são conferidos no D1 antes de servir conteúdo. O redirecionamento www só atua em requisições que chegam ao Worker; ele não cria um segundo certificado/hostname por conta própria.

Sair de Max/Ultra para Pro ou Grátis também dá 14 dias de domínio próprio. Durante o prazo, o GET mantém `active`/SSL `active` com `disableAt` futuro, se a configuração já estava ativa. O cliente pode usá-lo como URL oficial apesar de o novo plano não incluir domínio. Voltar a Max/Ultra antes do fim limpa `disableAt`. Ao vencer, o domínio deixa de servir imediatamente pela verificação do gateway e o cron remove o Custom Hostname. Se a Cloudflare falhar, o servidor mantém os IDs e repete a exclusão; GET já informa `disabled`, SSL `pending`, `records:[]`, hostname e prazo preservados. Voltar a pagar depois de desligado exige POST para configurar de novo. O endereço getblup.site só continua nos sites que cabem no plano. Os dois hosts servem o mesmo deploy, sem redirecionamento para uma URL oficial.

### Workers Cache e Early Hints

`wrangler.sites.jsonc` desliga o cache da entrada default e liga somente `CachedSite`. O gateway resolve o host e consulta autorização no D1 em todo pedido; chama `ctx.exports.CachedSite({props:{siteId,deploy}}).fetch(...)`. Host não integra a chave do Workers Cache. `ctx.props` separa site e deploy; encoding é normalizado e usa `Vary: Accept-Encoding`. Query não altera arquivos estáticos e é removida da chave; redirecionamento de barra final recupera a query no gateway. Cookies e Authorization não são encaminhados ao cache público.

O cache interno usa 600 segundos, tags `site:<id>` e `deploy:<id>:<deploy>`. O navegador mantém Cache-Control e ETag anteriores, inclusive 304 e HEAD. Respostas inexistentes ou bloqueadas usam no-store. A Cache API antiga foi removida. Link preload do CSS Bento/Mochi e da fonte Inter presentes no manifesto acompanha o HTML. A criação de Custom Hostname pede `ssl.settings.early_hints: "on"`; ative Early Hints também na zona dos sites. O envio real de 103 depende da Cloudflare, do cache dos hints e do navegador, e só pode ser conferido em staging/produção.

Bloqueio, desbloqueio, exclusão, restauração, remoção de site, vencimento da carência e desligamento de domínio enfileiram purgas por site. O cron agrupa até 100 tags e faz no máximo uma tentativa por minuto, abaixo do limite Free de cinco por minuto. A chamada parte de `CachedSite.purgePending`, por service binding `SITE_CACHE`, pois a purga pertence ao entrypoint que possui o cache. Falhas e limite da Cloudflare mantêm a fila; eventos concorrentes não são perdidos. O cron atual roda a cada cinco minutos, então a purga pode esperar essa rodada ou mais em uma fila grande. A autorização do gateway impede servir conteúdo revogado durante a espera.

A [documentação Workers Cache](https://developers.cloudflare.com/workers/cache/), [chaves](https://developers.cloudflare.com/workers/cache/cache-keys/), [purga](https://developers.cloudflare.com/workers/cache/purge/) e [limites](https://developers.cloudflare.com/workers/cache/limitations/) confirma que o cache não elimina a cobrança por request. O loopback com cache também é uma invocação cobrada; só a CPU do entrypoint atendido pelo cache deixa de rodar. Gateway e consulta D1 continuam em todos os acessos. O ganho esperado é reduzir R2 e CPU do serving; medir hits, CPU, linhas lidas e custo por visita antes de afirmar economia total. A propagação global de purgas, o hit real da camada gerenciada e o 103 ficam para validação em ambiente Cloudflare. Os testes locais verificam contrato, isolamento e bloqueios com provedores simulados.

### Limites de abuso e de corpos

`ABUSE_LIMIT`: 120/min por conta e por IP para IA, criação de site/cofre, domínio, check de slug, lembretes PUT, escolha de site, preparação/purga de histórico e remoção de site. `BULK_LIMIT`: 500/10 s por conta para uploads, missing, deploys, blocos/manifestos e downloads da sync. Ao negar, ambos retornam `429 TOO_MANY_REQUESTS` e `Retry-After` de 60 ou 10 segundos, respectivamente. A Cloudflare aplica esses limitadores por localização; as cotas financeiras e de armazenamento continuam no D1, que é a autoridade global.

JSON geral: 96 KiB; webhook Stripe assinado: 1 MiB. Metadados de publicação: 8 MiB. Multipart de versão: 34 MiB, com manifesto cifrado de até 32 MiB. Bloco: 4 MiB mais 64 bytes. Arquivo de site: até o limite do plano, no máximo 100 MiB. O servidor verifica Content-Length antes de receber e conta os bytes durante a leitura. Upload de arquivo maior que 4 MiB precisa informar Content-Length. Arquivos grandes são processados em fluxo, com digest incremental e staging R2; a validação não carrega o arquivo todo no isolate de 128 MiB. Staging e cópia final acrescentam operações R2, consideradas no orçamento.

O painel usa `hc<AppType>` e `InferResponseType`. `worker/export-rpc.mjs` emite `src/lib/rpc.d.ts` diretamente da cadeia completa de rotas Hono, antes do check/build. Não edite a declaração gerada. Isso mantém os tipos JSON sem importar os globais do runtime Worker para o DOM do painel.

## Antes de publicar

- Em instalação nova, aplicar de 0000 até `0013_webhook_stripe_atual.sql` antes do novo Worker. As migrações 0012 e 0013 foram geradas com drizzle-kit, sem alterar as anteriores. A 0012 remove os campos sem uso `previous_key` e `disabled_at`; a 0013 cria recibos de webhook e índice da assinatura Stripe. Em instalação já usada, preparar primeiro as migrações aditivas e publicar o código que não usa os campos removidos antes de aplicar a 0012: o Worker antigo ainda os escrevia. Planejar essa atualização com janela de manutenção se a ordem padrão das migrações impedir separar as etapas. Em atualização de instalação já usada, revogar as sessões antigas do Studio que ainda não receberam marca de aparelho e pedir novo login.
- Substituir o ID de D1 fictício nas duas configurações, preparar R2, Durable Object e crons e conferir todos os segredos do [guia de deploy](../docs/deploy.md). Configurar os bindings `OTP_LIMIT`, `ABUSE_LIMIT` e `BULK_LIMIT` no ambiente de produção.
- Configurar Access no host inteiro da staff, Turnstile real, Email Service com Email Sending habilitado, domínio remetente verificado e conta Workers Paid para enviar a destinatários arbitrários, Stripe/webhook de produção e chaves de IA. `EMAIL_MODE` de produção deve ser send. Não copiar `.dev.vars` local para produção.
- Para domínio próprio, habilitar Cloudflare for SaaS na zona dos sites. Cadastrar o secret `CF_API_TOKEN` com permissão SSL and Certificates Write apenas nessa zona e a var `CF_ZONE_ID`. Informar `SITES_CNAME_TARGET` com o CNAME público do serviço. Sem isso, a API retorna `DOMAIN_UNAVAILABLE`.
- Preparar o fallback origin e o CNAME target com DNS proxied. `wrangler.sites.jsonc` usa a rota `*/*` na zona getblup.site para alcançar também Custom Hostnames, conforme a [documentação Cloudflare](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/start/advanced-settings/worker-as-origin/). Conferir se algum outro serviço dessa zona precisa de rota de exclusão. Isso não altera a zona getblup.io do painel.
- Se usar a cortesia do dono sem assinatura Stripe, cadastrar `STAFF_OWNER_EMAIL` como secret também no Worker dos sites. Ambos precisam reconhecer a mesma conta verificada. Nenhum valor de segredo fica neste repositório.
- Criar alerta de gasto da conta no painel Cloudflare, em Manage Account > Billing > Billable Usage > Create budget alert, escolhendo valor e destinatário aprovados pelo dono. Também é possível abrir esse fluxo pelo resumo de gastos de Workers, D1 e R2. É um aviso por e-mail, sem bloqueio automático de gastos. A [documentação de orçamento](https://developers.cloudflare.com/billing/manage/budget-alerts/) distingue alerta financeiro e notificações por métrica. Acompanhar linhas lidas/escritas D1, operações e armazenamento R2, requests/CPU Workers, e-mails e hostnames provisionados.
- Provisionar primeiro o Worker dos sites e o binding `SITE_CACHE` para o entrypoint `CachedSite` do Worker correspondente (produção: `blup-sites-production`). Não expor esse entrypoint diretamente por rota pública. Conferir Early Hints da zona, certificado, TXT, CNAME e as purgas em staging.
- Rodar check, testes e build; testar o fluxo completo em staging com serviços simulados ou sandbox antes de habilitar produção. Esta rodada não publicou nada e não consultou credenciais, DNS ou contas de produção para marcar esses itens como feitos.

## Preços administrados no Stripe

`GET /api/plans` retorna `prices.monthly` e `prices.yearly`, cada um como `{unitAmount,currency}` ou `null`. `unitAmount` segue as unidades menores do Stripe; o painel respeita moedas sem casas decimais e as exceções da API. O plano Grátis não depende de Price. Um período sem preço válido fica sem valor exibido e com compra desabilitada; a API também nega esse checkout com `503 PRICING_UNAVAILABLE`.

O catálogo consulta `prices.list` com `active`, os seis `lookup_keys` e expansão de `data.product`, com validação Zod do produto, moeda, valor inteiro não negativo, recorrência e período. Os valores ficam por dez minutos na memória de cada isolate, sem binding novo. Requisições simultâneas compartilham a atualização. Se o Stripe falhar ou devolver uma lista inválida, permanece o último catálogo bom daquele isolate e a próxima tentativa espera trinta segundos. Um isolate novo sem valor bom retorna preços indisponíveis. Um preço removido ou inválido numa lista válida fica indisponível, sem ressuscitar seu valor antigo. Não existe garantia de cache compartilhado entre regiões ou após reinício.

Para mudar um preço sem alterar código:

1. Comece no sandbox. Abra o Product correspondente a Pro, Max ou Ultra no Dashboard Stripe e configure `metadata.blup_plan` como `pro`, `max` ou `ultra`. Também pode usar essa metadata no Price. Ela é necessária para reconhecer Prices antigos após a transferência do lookup key; não a remova de produtos com assinaturas existentes.
2. Crie um novo Price ativo no mesmo Product, com valor, moeda e recorrência corretos: mês ou ano, intervalo 1, preço fixo por unidade. O valor de um Price existente é imutável. A mudança anual é independente da mensal.
3. Atribua ao novo Price o lookup key correspondente: `blup_pro_mensal`, `blup_pro_anual`, `blup_max_mensal`, `blup_max_anual`, `blup_ultra_mensal` ou `blup_ultra_anual`. Se a ferramenta do Dashboard não oferecer a transferência, use a API ou o Workbench oficial do Stripe ao criar/atualizar o Price com `lookup_key` e `transfer_lookup_key=true`. A transferência tira a key do Price antigo e a coloca no novo, sem gravar chave privada em comando compartilhado, log ou arquivo público.
4. Confira os produtos e preços permitidos na configuração do Customer Portal. Não arquive um Price antigo antes de avaliar o efeito sobre cobranças, Portal e migrações em andamento.
5. Rode `npm run stripe:setup` com a chave do sandbox fornecida pelo ambiente privado. O script só lê e lista disponibilidade, valor e moeda dos lookup keys. Aguarde até dez minutos por isolate e confira o painel e o Checkout no sandbox. Não existe desconto anual presumido no código.
6. Repita a configuração no ambiente de produção somente após aprovação e validação operacional. A mudança de preço não exige deploy do Blup Cloud. Alterar limites, benefícios ou acrescentar um plano exige mudar `worker/plans.ts`, acompanhar tipagem/testes e publicar código.

Assinaturas existentes continuam vinculadas ao Price ID antigo; transferir a lookup key não troca seu valor. Para migrá-las, o operador precisa atualizar explicitamente os itens da assinatura ou criar uma alteração agendada no Stripe. Escolha e comunique a data, as regras de proporcionalidade e a cobrança antes de migrar. O plugin usa `create_prorations` por padrão nas mudanças imediatas de plano; a configuração do Customer Portal e eventuais schedules também determinam quando a mudança produz efeito. Este script não migra clientes nem cobra valores. Referências: [gerenciar preços](https://docs.stripe.com/products-prices/manage-prices), [listar lookup keys](https://docs.stripe.com/api/prices/list), [alterar preços de assinaturas](https://docs.stripe.com/billing/subscriptions/change-price) e [proporcionalidade](https://docs.stripe.com/billing/subscriptions/prorations).

### Webhook de cobrança

O SDK instalado é Stripe 23.1.0, com versão de API fixada em `2026-10-28.endive`; Better Auth e seus plugins estão no lockfile em 1.7.14. Configure o endpoint `POST /api/auth/stripe/webhook` com assinatura e eventos snapshot dessa versão. O Worker verifica o corpo bruto com `constructEventAsync`, o tempo da assinatura e a separação sandbox/produção. Não confia no status, preço, cliente nem valores do snapshot: consulta o objeto atual no Stripe e, para eventos de assinatura, checkout e invoice, busca a assinatura atual com `subscriptions.retrieve`, expandindo `items.data.price.product`. `lookup_key` já pertence ao Price e não é um campo expansível separado.

Eventos de assinatura usam `customer.subscription.*`; checkout usa os eventos de conclusão, pagamento assíncrono e expiração necessários ao fluxo; invoices incluem pagamentos e falhas. Registre pelo menos `customer.subscription.created`, `customer.subscription.updated`, `customer.subscription.deleted`, `customer.subscription.paused`, `customer.subscription.resumed`, `checkout.session.completed`, `checkout.session.async_payment_succeeded`, `checkout.session.async_payment_failed`, `invoice.paid` e `invoice.payment_failed`. Para o histórico financeiro da staff, inclua também `charge.refunded`, `charge.dispute.created` e `radar.early_fraud_warning.created`. Só habilite os tipos realmente usados; configurar todos aumenta consultas e custo sem benefício.

Uma lease por assinatura serializa consulta e gravação, e outra por event.id evita processamento simultâneo da mesma entrega. Uma lease por conta também protege a adoção da linha pendente do Checkout quando duas assinaturas distintas apontam para a mesma referência; uma nunca sobrescreve a outra. Estado, registro financeiro e recibo são gravados na mesma transação D1, condicionada à posse das leases. Trava ocupada, estado inválido ou falha de API retorna 503 com `Retry-After: 1`; o Stripe pode reenviar. `event.created` não serve como ordenação. Evento velho apenas reaplica o estado atual, e uma assinatura cancelada não cancela outra da mesma conta. A identificação do dono cruza o Customer atual com a conta e as referências internas, sem ressuscitar conta apagada.

Os caminhos `/stripe/webhook` e `/subscription/success` do plugin foram desabilitados com `disabledPaths`. A rota pública do webhook é atendida pelo Worker, e a volta do Checkout apenas redireciona para uma URL da origem do painel, sem escrever estado. As demais operações de Checkout/Portal continuam no plugin. A confirmação de pagamento na interface aguarda a confirmação assíncrona, sem prometer plano ativo pelo parâmetro da URL.

Recibos de eventos tornam-se elegíveis para remoção após noventa dias, com limpeza diária indexada em lotes de até 500; uma fila grande pode atrasar a remoção. A purga de uma conta remove os recibos associados. O índice permite localizar os vencidos sem varrer todos os recibos. A unicidade do registro financeiro continua impedindo duplicação desse histórico. Depois da retenção, uma reentrega ainda consulta o estado atual. O endpoint não guarda payloads nem imprime dados de clientes em logs. A assinatura e os testes locais não substituem um teste de entrega real no sandbox antes de cobrar. Referências: [ordem, assinatura e reentrega](https://docs.stripe.com/webhooks), [consulta de assinatura](https://docs.stripe.com/api/subscriptions/retrieve) e [plugin Better Auth](https://better-auth.com/docs/plugins/stripe).

## Variáveis, secrets e bindings

Valores reais ficam fora do repositório. O mesmo nome pode ter valores diferentes em desenvolvimento e produção.

| Var pública ou operacional | Uso |
| --- | --- |
| `APP_URL` | Origem oficial do painel, URLs do auth, cookies e origens confiáveis |
| `PASSKEY_RP_ID` | RP ID das passkeys; precisa combinar com o domínio e a origem |
| `MAIL_FROM` | Nome e endereço remetente autorizados no Email Sending |
| `TURNSTILE_SITE_KEY` | Chave pública do widget; trocar a de teste em produção |
| `EMAIL_MODE` | `log` local ou `send` produção; log local pode conter OTP |
| `STAFF_HOST` | Host exclusivo da staff, protegido por Access |
| `OPENROUTER_URL` | Vazio usa o provedor oficial; substituição só em testes controlados |
| `SITES_DOMAIN` | Sufixo dos sites; normalmente getblup.site |
| `CF_ZONE_ID` | ID da zona Cloudflare for SaaS dos sites |
| `SITES_CNAME_TARGET` | Destino CNAME público informado para domínios próprios |

| Secret | Uso |
| --- | --- |
| `BETTER_AUTH_SECRET` | Assinatura e proteção dos dados de autenticação |
| `TURNSTILE_SECRET_KEY` | Verificação do captcha no servidor |
| `STRIPE_SECRET_KEY` | Leitura do catálogo/assinaturas e operações de Customer, Checkout, Portal e assinatura; separar sandbox/produção e limitar permissões |
| `STRIPE_WEBHOOK_SECRET` | Verificação da assinatura do endpoint daquele ambiente |
| `ACCESS_TEAM_DOMAIN`, `ACCESS_AUD` | Emissor e audience exigidos no JWT do Cloudflare Access da staff |
| `STAFF_OWNER_EMAIL` | Bootstrap de admin e cortesia Ultra para a conta com e-mail verificado; também no Worker dos sites |
| `STAFF_DEV_EMAIL` | Atalho somente local; não configurar em produção |
| `OPENROUTER_API_KEY` | Autorização das chamadas de IA |
| `CF_API_TOKEN` | API de Custom Hostnames, SSL and Certificates Write limitada à zona dos sites |

`CLOUDFLARE_API_TOKEN`, quando usado pelo processo de publicação, é uma credencial operacional do Wrangler e não é a credencial `CF_API_TOKEN` do Worker. O catálogo de preços precisa de leitura de Prices e Products; o webhook precisa de leitura de subscriptions, invoices, checkout sessions e dos objetos financeiros registrados. Uma chave restrita deve cobrir também as operações que Checkout/Portal e a exclusão de conta executam.

Bindings: `DB` é D1 compartilhado pelos Workers; `NOTES` e `SITES` são R2 separados; `EMAIL` é Email Sending; `OTP_LIMIT` limita OTP/cadastro; `ABUSE_LIMIT` e `BULK_LIMIT` limitam controle e transferência; `SYNC_LIVE` é o Durable Object de sync; `SITE_CACHE` alcança o entrypoint de cache dos sites; `ASSETS` serve o painel compilado. Não foi acrescentado KV nem outro binding para preços.

Enviar e-mails a destinatários arbitrários exige Email Sending habilitado, domínio autorizado e Workers Paid, conforme a [documentação do Email Service](https://developers.cloudflare.com/email-service/). Não confundir isso com encaminhamento do Email Routing para um destinatário verificado. Confirme cotas, aprovação do serviço e DNS antes de habilitar OTP ou lembretes em produção.
