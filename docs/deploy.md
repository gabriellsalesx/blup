# Deploy de produção

Procedimento revisado com os contratos da rodada 4 em 02/11/2026 a partir de [wrangler.jsonc](../cloud/wrangler.jsonc), [wrangler.sites.jsonc](../cloud/wrangler.sites.jsonc), [package.json](../cloud/package.json) e código do Worker. Siga o [checklist em ordem](deploy-checklist.md) para executar o lançamento. Este trabalho documenta a operação: não criou recursos, aplicou migrações remotas, cobrou, enviou e-mail ou publicou aplicações. Exemplos não contêm credenciais. [Arquitetura](arquitetura.md) e [segurança](seguranca.md) explicam as fronteiras.

## 1. Definir os destinos e preparar o lançamento

| Endereço | Aplicação |
| --- | --- |
| `getblup.io` | Site institucional, em outro repositório; não é produzido por `cloud/` |
| `cloud.getblup.io` | Painel da conta, login, aparelho e API pública autenticada |
| `staff.getblup.io` | Painel e API da equipe, protegidos por Access no domínio inteiro |
| `nome.getblup.site` | Site público do usuário, servido pelo Worker separado |

As duas zonas DNS devem estar ativas na conta Cloudflare. O envio para destinatários arbitrários pelo Email Service exige Workers Paid e domínio remetente habilitado; é diferente de encaminhar e-mail recebido. Veja [Email Service](https://developers.cloudflare.com/email-service/) e [onboarding de envio](https://developers.cloudflare.com/email-service/get-started/send-emails/).

Use o checkout e lockfiles revisados. Confirme as versões de Go, Node, Wrangler e dependências de `cloud/package.json`. As dependências gráficas do Studio pertencem à máquina de build, não ao Worker. A documentação atual de Better Auth foi consultada junto da configuração do projeto; exemplos novos de adaptador não autorizam trocar imports ou dependências durante um lançamento. O projeto usa `better-auth/adapters/drizzle`, conforme seu código e lock.

Antes de preparar artefatos, os checks do repositório são:

```sh
go vet ./...
go test ./...
(cd studio && go vet . && go test ./...)
(cd cloud && npx tsc -p worker/tsconfig.json --noEmit && npx vitest run)
(cd studio/frontend && npx svelte-check)
```

Os comandos seguintes pressupõem a pasta `cloud/`, exceto onde indicado. Em checkout novo, instale pelo lockfile com `npm ci`. O repositório não traz configuração de produção já provisionada: os UUIDs D1 são placeholders e a site key Turnstile ainda precisa ser substituída pela de produção.

## 2. Criar D1, R2 e preparar migrações

Com a sessão Cloudflare de operação autenticada:

```sh
npx wrangler d1 create blup-cloud
npx wrangler r2 bucket create blup-cloud-notes
npx wrangler r2 bucket create blup-cloud-sites
```

No checkout privado de lançamento, preencha o `database_id` devolvido no bloco `env.production.d1_databases` dos dois arquivos Wrangler. Ambos devem apontar ao mesmo banco. Confirme `migrations_dir = "migrations"`. Os buckets são privados, acessados por bindings; não habilite acesso público direto ao bucket das notas. O Worker da conta precisa de `DB`, `NOTES`, `SITES`, `SYNC_LIVE`, `EMAIL`, `OTP_LIMIT`, `ABUSE_LIMIT`, `BULK_LIMIT` e o service binding `SITE_CACHE`; o dos sites precisa de `DB` e `SITES`. Bindings e vars precisam ser definidos por ambiente, conforme [Wrangler environments](https://developers.cloudflare.com/workers/wrangler/environments/). A criação de buckets segue a [referência R2](https://developers.cloudflare.com/r2/buckets/create-buckets/).

As migrações versionadas vão de `0000_inicial.sql` a `0013_webhook_stripe_atual.sql` nesta revisão. Revise o SQL e faça backup do banco existente antes de aplicá-las. Em instalação nova, aplicar tudo antes do código. Em atualização existente, `0012` remove colunas que o Worker anterior ainda escrevia: não aplicar a sequência com esse código atendendo tráfego ou cron. Ensaiar etapas compatíveis ou usar janela de manutenção, suspender gravações/crons antigos, aplicar SQL e publicar os dois Workers novos antes de reabrir. Rollback para código que exige as colunas removidas não é seguro sem plano de schema/dados. Gere esquema Better Auth ou Drizzle somente quando houver alteração de esquema a desenvolver/revisar; um deploy normal aplica os arquivos já versionados. Com Drizzle, gerar esquema não aplica automaticamente o D1. Veja [Better Auth com Drizzle](https://better-auth.com/docs/adapters/drizzle) e [migrações D1](https://developers.cloudflare.com/d1/reference/migrations/).

```sh
npx wrangler d1 migrations list DB --remote --env production
npx wrangler d1 migrations apply DB --remote --env production
```

O script `npm run db:migrate:remote` não acrescenta `--env production` sozinho. Se usá-lo, passe o ambiente explicitamente: `npm run db:migrate:remote -- --env production`. Não aplique migrações outra vez por meio de um gerador de esquema independente.

A migração `0010` acrescenta reservas/cotas, leases, manifestos retirados, domínios verificados, retenção de conta, identificação de sessão de aparelho e tentativas/leases de lembretes. Deve estar aplicada antes de publicar código que consulta essas tabelas.

A migração `0011`, gerada pelo Drizzle, acrescenta carências/seleção de site, prazo de domínio, referências atuais, tickets/recibos, chaves físicas R2, fila de descarte e fila de purga do cache, com índices de expiração. Aplique a sequência versionada antes dos novos Workers. A Lupa validou migrações do zero no D1 local por Wrangler e planos EXPLAIN com os novos índices; isso não comprova aplicação remota. Em atualização de instalação existente, revogue as sessões Studio anteriores à marca de aparelho e solicite novo login, como orienta `cloud/README.md`.

Mantenha a migração Durable Object `v1` com `new_sqlite_classes: ["SyncLive"]`, exportada por `worker/index.ts`. Ela é da infraestrutura Workers, separada das migrações SQL do D1.

`0012_remover_estado_sem_uso.sql`, gerada pelo Drizzle na rodada 4, remove `sync_account.previous_key` e `site_domain.disabled_at`. O primeiro guardava um embrulho antigo nunca lido; o segundo não tinha consumidor. Não remove `disable_at`, que continua controlando a carência de domínio. A corrida da criação inicial de `/sync/key` agora responde `409 KEY_EXISTS` a quem perdeu, em vez de confirmar criação inexistente.

`0013_webhook_stripe_atual.sql` cria recibos `stripe_webhook_event`, índices por data/conta e índice de `subscription.stripe_subscription_id`. É bloqueante para o webhook próprio desta rodada. Os recibos ficam elegíveis para limpeza após 90 dias, em lotes diários de até 500; acúmulo pode atrasar a remoção. A purga de conta também remove seus recibos.

## 3. Configurar vars e secrets

Confira os valores não secretos da produção:

| Nome | Contrato |
| --- | --- |
| `APP_URL` | `https://cloud.getblup.io`, origem e base do Better Auth |
| `PASSKEY_RP_ID` | `getblup.io`; o origin da cerimônia continua sendo `APP_URL` |
| `STAFF_HOST` | `staff.getblup.io`; ativa a separação dos hosts no Worker |
| `SITES_DOMAIN` | `getblup.site` nos dois Workers |
| `MAIL_FROM` | Nome Blup e remetente `no-reply@getblup.io` |
| `EMAIL_MODE` | `send` em produção; `log` apenas em ambiente simulado |
| `TURNSTILE_SITE_KEY` | Site key pública do widget de produção |
| `CF_ZONE_ID` | ID da zona SaaS, 32 caracteres hexadecimais; usado só pelo Worker da conta para Custom Hostnames |
| `SITES_CNAME_TARGET` | Host de destino CNAME configurado no SaaS, sem esquema; exibido ao cliente |
| `OPENROUTER_URL` | Vazio usa o endpoint padrão; não deixar endereço de teste |

Cadastre os segredos por prompt ou pelo gerenciador do ambiente. Nunca cole valores neste guia, em `.toml`, scripts versionados ou histórico de comandos com argumentos contendo a credencial. Cada secret é do ambiente correto, como documentado em [Workers secrets](https://developers.cloudflare.com/workers/configuration/secrets/).

`wrangler secret put` cria e publica uma nova versão imediatamente. No primeiro lançamento, o operador pode preparar JSON privado fora do repositório e enviar código e secrets juntos com `wrangler deploy --secrets-file '<arquivo-privado.json>' --env production`. O arquivo dos sites contém somente os secrets necessários a esse Worker. O [checklist](deploy-checklist.md#8-revisar-vars-e-preparar-secrets) usa essa sequência para evitar configurações parciais; nenhuma dessas operações foi executada nesta revisão.

```sh
npx wrangler secret put BETTER_AUTH_SECRET --env production
npx wrangler secret put TURNSTILE_SECRET_KEY --env production
npx wrangler secret put STRIPE_SECRET_KEY --env production
npx wrangler secret put STRIPE_WEBHOOK_SECRET --env production
npx wrangler secret put OPENROUTER_API_KEY --env production
npx wrangler secret put CF_API_TOKEN --env production
npx wrangler secret put ACCESS_TEAM_DOMAIN --env production
npx wrangler secret put ACCESS_AUD --env production
npx wrangler secret put STAFF_OWNER_EMAIL --env production
```

`BETTER_AUTH_SECRET` deve ser forte e persistente entre publicações. A lista `trustedOrigins` do código contém somente `APP_URL`; não amplie para todos os sites dos usuários. `STAFF_DEV_EMAIL` não deve ser configurado em produção. O Worker dos sites não precisa dos secrets de login, Stripe, IA, e-mail ou `CF_API_TOKEN`. Se a cortesia Ultra do proprietário for usada, ele também precisa de `STAFF_OWNER_EMAIL`, pois calcula o plano na entrega: cadastre esse secret com `-c wrangler.sites.jsonc --env production`, sem publicar seu valor. O token Cloudflare deve ficar limitado à zona SaaS e às operações de Custom Hostnames, incluindo leitura, criação e exclusão; a API de criação documenta a permissão `SSL and Certificates Write`, conforme a [referência oficial](https://developers.cloudflare.com/api/resources/custom_hostnames/methods/create/). O uso de origem explícita e cookies por host segue [Better Auth security](https://better-auth.com/docs/reference/security). Passkeys de `localhost` não se tornam credenciais de `getblup.io`; confira [rpID e origin](https://better-auth.com/docs/plugins/passkey).

## 4. E-mail e cron

Na Cloudflare, acesse Compute, Email Service, Email Sending e faça Onboard Domain de `getblup.io`. Confira os registros indicados pelo serviço para bounce, SPF, DKIM e DMARC; não invente uma política ou chave DKIM copiando outro domínio. Aguarde a verificação do domínio. O binding `EMAIL` em produção permite `allowed_sender_addresses: ["no-reply@getblup.io"]`. O código usa a API de envio com objeto de remetente, destinatário, assunto, texto e HTML. Referência: [Workers Email Sending API](https://developers.cloudflare.com/email-service/api/send-emails/workers-api/).

Há dois crons em `env.production.triggers`: `0 4 * * *` para limpeza diária às 04:00 UTC e `*/5 * * * *` para trabalho periódico. O diário limpa histórico/blocos e fila antiga, tenta cancelamentos financeiros e purga contas elegíveis após 30 dias. O de cinco minutos envia lembretes, expira reservas/tickets, limpa sites, descarta objetos de sync, repete cancelamentos financeiros, avisa downgrades, desativa domínios vencidos e depois drena purgas do cache por `SITE_CACHE`. Ambos invocam `scheduled`; o código escolhe o trabalho pelo texto de `controller.cron`. Verifique que os dois foram instalados. A configuração e o teste local seguem a [referência Cron Triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/).

Para simular sem e-mail real, use `EMAIL_MODE=log`, credenciais de teste e persistência isolada. As portas do time reservadas não devem ser utilizadas. Exemplo de sessão própria, com a pasta de dados escolhida pelo operador:

```sh
npx wrangler dev --port 18820 --persist-to '<pasta-da-sessao>' --test-scheduled
```

Na versão atual da documentação Cloudflare, a rota local é `/cdn-cgi/local/scheduled`, com query `cron`; versões anteriores podem expor outro caminho. Confirme a rota na versão do Wrangler instalada. Para invocar o cron de lembretes nessa sessão:

```sh
curl --get --data-urlencode 'cron=*/5 * * * *' \
  'http://localhost:18820/cdn-cgi/local/scheduled'
```

Cron e log bem-sucedidos não provam entrega na caixa de entrada. A fila agora reserva cota e reivindica por lease, conta apenas aceitação do provedor e repete falhas até três tentativas; uma queda entre enviar e confirmar pode duplicar a mensagem. O [contrato de lembretes](spec/calendario-e-lembretes.md) explica tentativas, cotas e retenção. Não use envio pago real como teste automatizado.

## 5. Turnstile, login e Access da equipe

Crie o widget Turnstile para o hostname de login de produção. Substitua a site key pública de teste e grave o secret correspondente. O backend valida o token ao pedir OTP e no cadastro inicial com passkey, além de limitar solicitações por IP. A documentação distingue [widget e validação no servidor](https://developers.cloudflare.com/turnstile/get-started/); uma chave de teste que sempre aprova não é a proteção de produção.

No Cloudflare Zero Trust, crie aplicação Access para o hostname inteiro `staff.getblup.io`, cobrindo assets e `/api/staff/*`. Cadastre só as identidades da equipe. Grave a URL HTTPS do time em `ACCESS_TEAM_DOMAIN` e o audience dessa aplicação em `ACCESS_AUD`. `staff.ts` verifica assinatura por JWKS, issuer, audience e e-mail, e depois aplica a tabela de papéis ou o bootstrap de proprietário. Não basta enviar um header de e-mail. Veja [validação Access](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/).

O suporte pode consultar conta/uso, revogar sessões/passkeys e restaurar conta excluída antes do prazo e do início da purga; admin também bloqueia, exclui e gerencia equipe. A área pública rejeita rotas de staff quando `STAFF_HOST` está configurado. Não use a conta de cortesia do proprietário como evidência de funcionamento do checkout pago.

## 6. Stripe de produção

Prepare e valide o fluxo antes em um sandbox. Produção tem recursos, preços, portal e segredo de webhook próprios; não reutilize IDs ou signing secret do listener local. O Worker usa chave server-side restrita às operações necessárias. A separação de ambientes e permissões segue [Stripe API keys](https://docs.stripe.com/keys).

1. Confirme limites e lookup keys de `worker/plans.ts`. Valores e moedas dos preços não ficam mais nesse arquivo.
2. Crie produtos e preços recorrentes por unidade no Dashboard de produção, com os seis lookup keys mantidos no código. `npm run stripe:setup` agora apenas confere o catálogo, com leitura de Prices/Products; não cria nem corrige preço. `pricing.ts` valida o catálogo Stripe e fornece períodos mensais/anuais anuláveis a `/api/plans`. Ausência desabilita a compra, inclusive no servidor com `503 PRICING_UNAVAILABLE`. O cache por isolate dura dez minutos; falha conserva o último catálogo válido e usa backoff de trinta segundos. Para alterar valor/moeda, crie outro Price e transfira o lookup key; assinaturas existentes conservam o Price antigo até migração explícita. Antes dessa troca, configure `metadata.blup_plan` no Product ou Price com `pro`, `max` ou `ultra`, para o reconciliador identificar o plano dos preços antigos sem lookup key.
3. Configure o Customer Portal no ambiente real, com cartão, faturas e cancelamento. A aplicação redireciona para ele; não incorpora campos de cartão.
4. Crie o webhook HTTPS `https://cloud.getblup.io/api/auth/stripe/webhook`, conferindo compatibilidade com a API `2026-10-28.endive` fixada em `stripe-client.ts` para SDK 23.1.0. O endpoint é atendido pelo reconciliador próprio [billing.ts](../cloud/worker/billing.ts), antes do handler Better Auth. Configure `checkout.session.completed`, `customer.subscription.created`, `.updated` e `.deleted`; confira os eventos adicionais necessários antes de habilitar outros meios de pagamento.
5. Inclua os eventos que `billing.ts` registra: `invoice.paid`, `invoice.payment_failed`, `charge.refunded`, `charge.dispute.created` e `radar.early_fraud_warning.created`. Grave o signing secret do endpoint em `STRIPE_WEBHOOK_SECRET`.
6. Teste assinatura, duplicação, ordem invertida e repetição após falha em sandbox. O handler lê os objetos atuais no Stripe; toma lease por evento e por assinatura antes de ler a subscription; grava estado, evento financeiro e recibo no mesmo batch D1, conferindo as leases. Uma lease por conta também serializa a adoção de referência pendente compartilhada por assinaturas distintas, preservando ambas. `event.id` deduplica a entrega. Os caminhos `/stripe/webhook` e `/subscription/success` do plugin estão desativados. O success próprio só faz redirecionamento local validado; não grava assinatura nem concede plano. A verificação segue [Stripe webhooks](https://docs.stripe.com/webhooks).

A chave de execução precisa de Customers, Checkout Sessions, Customer Portal e Subscriptions conforme as operações usadas, leitura de Prices/Products, Invoices, Charges, Disputes e Early Fraud Warnings, além da leitura de Checkout Sessions e Subscriptions usada pela reconciliação. O script de conferência não precisa criar produtos/preços. Políticas comerciais, impostos, recibos e tentativas de recuperação de pagamento são escolhas operacionais no Stripe; este documento não afirma que estejam ativadas.

## 7. DNS, TLS e separação dos sites

O Worker da conta tem Custom Domains para `cloud.getblup.io` e `staff.getblup.io`. O institucional no apex `getblup.io` continua independente. O Worker dos sites usa a rota de zona `*/*` em `getblup.site`, necessária também aos Custom Hostnames; precisa de DNS curinga com proxy habilitado. A configuração do projeto prevê `AAAA *` com destino `100::` para não depender de um origin externo. Confira a zona e ausência de registros conflitantes. Rotas precisam de DNS proxied, conforme [Workers Routes](https://developers.cloudflare.com/workers/configuration/routing/routes/).

O contrato público é `nome.getblup.site`, sempre HTTPS e sem `www`. O código redireciona HTTP e tenta normalizar `www.nome.getblup.site`, mas TLS acontece antes do Worker. Universal SSL cobre apex e primeiro nível, não garante certificado para esse segundo nível `www.nome`; não anuncie essa variante como funcional sem certificado adicional. Referência: [limitações Universal SSL](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/limitations/).

O bucket `SITES` não deve ser oferecido diretamente por domínio público próprio do bucket: o Worker precisa selecionar o manifesto e aplicar cache/headers. O bucket `NOTES` permanece privado e só é acessado por APIs autenticadas. Domínios próprios Max/Ultra têm API e entrega implementadas; dependem de Cloudflare for SaaS configurado nos passos abaixo.

### Cloudflare for SaaS para domínios dos clientes

1. Habilite Cloudflare for SaaS na zona `getblup.site` e confira os direitos/custos da conta. Crie um hostname de fallback na zona com AAAA `100::` proxied, selecione-o como fallback origin e aguarde ficar ativo. Referência: [configuração inicial](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/start/getting-started/).
2. Defina o destino CNAME do serviço para esse origin. Grave o host em `SITES_CNAME_TARGET` e o ID da zona em `CF_ZONE_ID`, no Worker da conta. Não aponte esse destino ao bucket R2.
3. Confirme que a rota de zona `*/*` aponta ao Worker dos sites, como em `wrangler.sites.jsonc`. A rota somente `*.getblup.site/*` não captura por si todos os Custom Hostnames. Revise possíveis rotas de outros serviços nessa zona, conforme [Worker como fallback origin](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/start/advanced-settings/worker-as-origin/).
4. Com o código publicado pelo operador, o cliente Max/Ultra vincula um hostname pela API. O Blup primeiro confere a prova TXT aleatória em `_blup.<hostname>`; só então cria o Custom Hostname com SSL DV, validação TXT e TLS mínimo 1.2. Exiba também os registros de validação devolvidos pela Cloudflare. Prova de vínculo Blup e validação do certificado são etapas distintas, conforme [hostname validation](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/domain-support/hostname-validation/).
5. No DNS do cliente, crie a prova TXT e o CNAME indicados. Na raiz, o provedor precisa de CNAME flattening/ALIAS ou de uma solução de apex proxying habilitada; não prometa um CNAME convencional ou endereço IP genérico em qualquer registrador. Só anuncie o domínio quando propriedade, hostname e SSL estiverem ativos. Remover domínio/site ou purgar conta também exige permissão para DELETE do Custom Hostname.

A aplicação oferece um hostname por site; criação/reconfiguração exige Max/Ultra. Após perda desse direito, o domínio existente pode continuar por 14 dias, com `disableAt`, e GET/DELETE seguem permitidos. Domínio e SSL ativos, com direito ou carência futura, podem ser a baseURL oficial. O Studio republica ao observar mudança de base, no máximo uma vez a cada dez minutos por site; canonical, feed e sitemap só mudam na publicação concluída. Não há redirecionamento entre domínio e `getblup.site`. O contrato está em [publicação](spec/publicacao.md#domínio-próprio-no-blup-cloud). Nenhum provisionamento SaaS ou teste com DNS de cliente foi executado nesta revisão.

### Workers Cache, purga e Early Hints

Mantenha a entrada default sem cache em `wrangler.sites.jsonc`; somente `CachedSite` tem `cache.enabled = true` no mapa `exports`. A sintaxe por entrypoint exige Wrangler 4.107.0 ou superior; o projeto declara uma versão compatível. O gateway consulta autorização no D1 a cada pedido antes de chamar o cache com props confiadas de site/deploy. Não exponha `CachedSite` diretamente por rota pública. Confira [configuração Workers Cache](https://developers.cloudflare.com/workers/cache/) e [chaves](https://developers.cloudflare.com/workers/cache/cache-keys/).

O Worker da conta precisa de `SITE_CACHE` ligado ao entrypoint `CachedSite` do Worker dos sites. Produção usa `blup-sites-production`, conforme `wrangler.jsonc`; desenvolvimento usa `blup-sites`. Preserve a entrada `services` no ambiente correspondente. Publique/provisione o Worker dos sites antes do Worker da conta que depende desse binding. A chamada de purga precisa executar no contexto do entrypoint dono do cache; o cron usa `CachedSite.purgePending`, com fila D1 por site/geração. Há até cem tags por tentativa, no máximo uma tentativa por minuto, com execução efetiva pelo cron de cinco minutos e repetição em falhas. Referência: [purga por tags e escopo](https://developers.cloudflare.com/workers/cache/purge/).

O TTL interno é 600 segundos. A política de navegador continua separada; um hit não pula o gateway/D1 e não elimina a cobrança de requests, incluindo o loopback. Meça requests, hits, CPU, linhas D1 e operações R2 antes de afirmar redução de custo total, conforme os [limites do cache](https://developers.cloudflare.com/workers/cache/limitations/).

Ative Early Hints em Speed, Content Optimization da zona dos sites. O Worker devolve Link preload para CSS/fonte conhecidos no manifesto, e novos Custom Hostnames pedem `ssl.settings.early_hints = "on"`. Valide 103 real no ambiente Cloudflare; o teste local só confere configuração/header, pois emissão depende do serviço e do navegador. Referência: [103 Early Hints](https://developers.cloudflare.com/workers/examples/103-early-hints/).

Solicite a inclusão de `getblup.site` na seção PRIVATE da Public Suffix List pelo responsável pelo domínio, seguindo [as regras oficiais da PSL](https://publicsuffix.org/submit/). Isso ajuda os navegadores a tratar cada subdomínio como site registrável distinto, inclusive nas fronteiras de cookies. O PR precisa ser aprovado e a lista precisa chegar aos navegadores; criar um registro DNS não altera a PSL. Não há comprovante de inclusão e propagação no repositório, e o lançamento não deve declarar esse processo concluído sem conferência operacional. PSL não é selo de confiança nem substitui isolamento por origem.

## 8. Gerar e publicar os dois Workers

Depois de recursos, configurações, checks e migrações conferidos pelo responsável:

```sh
npm run build
npx wrangler deploy -c wrangler.sites.jsonc --env production
npx wrangler deploy --env production
```

Não use `npm run deploy` sem argumentos esperando que selecione produção: o script não adiciona esse ambiente. Com ambientes nomeados, o nome remoto normalmente recebe o sufixo `-production`; confira o nome apresentado pelo Wrangler, as rotas e os bindings dos dois Workers. `workers_dev` e preview URLs estão desativados em produção no projeto.

Verifique no ambiente apropriado: `/api/config`, login/passkey/device flow, acesso staff bloqueado fora da equipe, migrações, dois crons, limites dos planos, reserva de slug e publicação de um site de teste. Confira os bindings de rate limit: OTP cinco/minuto, controle 120/minuto por conta/IP e transferências 500/dez segundos por conta. Simule mudanças de plano, revogação, retenção e purga no ambiente isolado, incluindo falha ao remover Custom Hostname; não faça esses testes em contas reais. Depois confirme revalidação do HTML e feed, cache de arquivos com fingerprint e atualização do manifesto. Essas verificações operacionais não foram feitas contra produção neste trabalho.

Na validação de staging, confira também a carência com teto anterior e uso total incluindo histórico; escolha/troca de site retido até o prazo; `historyBytes: null`; purga por cofre com falha R2 sem desconto antecipado; prepare/PUT com ticket/commit idempotente/cancelamento; quatro uploads normais em paralelo; e upgrade concorrente com desligamento de domínio. As regressões locais usam simulações. Confirme externamente hits e isolamento do cache, purga por `SITE_CACHE`, restauração da política do navegador e 103, sem deduzir êxito de produção apenas de build verde.

Crie um budget alert com valor/destinatário aprovados pelo responsável em Manage Account, Billing, Billable Usage, Create budget alert. Ele avisa por e-mail e não interrompe gastos. Acompanhe Workers requests/CPU, D1, R2, e-mails e hostnames, conforme [alertas de orçamento Cloudflare](https://developers.cloudflare.com/billing/manage/budget-alerts/). Não há configuração desse alerta comprovada no repositório.

Guarde o identificador da versão anterior dos Workers e um backup pré-migração. Reverter código não reverte D1, objetos ou eventos Stripe. O servidor conserva manifestos substituídos por pelo menos dez minutos para limpeza segura, sem oferecer histórico de deploys como substituto de backup do conteúdo. A conta excluída fica bloqueada imediatamente e elegível à purga depois de 30 dias; falha externa adia e repete, não justifica apagar seus identificadores manualmente. Reduções com excesso de sites/storage abrem carência de 14 dias; Pro escolhe um site, Grátis fica com zero depois dela, e o domínio tem prazo próprio de 14 dias. Os excedentes permanecem guardados. A purga manual de histórico é por cofre e preserva sua versão atual. O [contrato de retenção](spec/blup-cloud-sync.md#exclusão-retenção-e-mudança-de-plano) explica restauração e limites. Retenha artefatos e plano de restauração em local privado.

## 9. Lançamento do Studio e da CLI

O workflow [release.yml](../.github/workflows/release.yml) publica a CLI por GoReleaser e executa testes com Hugo. Os jobs `studio-linux` e `studio-publish` geram, verificam e publicam os quatro arquivos Linux com checksums assinados. `studio-msix` prepara x64/arm64 como artefatos da execução, condicionado às variáveis Partner Center, sem publicar na Store. Esses jobs estão configurados, mas não foram executados nesta conferência. Windows direto e macOS continuam com preparação própria pelas [tarefas Studio](../studio/Taskfile.yml).

1. Compile o frontend, gere bindings e compile o Studio com a toolchain de `studio/go.mod`. Em Linux, a pilha padrão é GTK4/WebKitGTK 6.0 com GStreamer para MP3. Não execute `run` ou abra uma segunda janela do dono para testar artefatos.
2. Defina a versão real do produto na preparação da release. As Taskfiles injetam `app.Version` por ldflags com `VERSION` padrão `1.0.0`; o CI passa a versão da tag ao nfpm, ao MetaInfo e aos nomes de arquivo. Confira também recursos e metadados de plataforma na preparação. Build com tag `production` recusa iniciar sem versão `x.y.z`, inclusive `dev`.
3. Os canais definidos pelo dono são Windows/macOS sem assinatura de desenvolvedor por enquanto e Linux em AppImage, AUR, `.flatpak` no GitHub e `.deb`, sem Flathub. As tarefas nfpm/NSIS/DMG existentes são capacidades de empacotamento, não prova de publicação nesses canais. O frontend precisa ser embutido no binário correspondente.
4. Publicar Windows em EXE/NSIS compatível com esse plano; MSIX não deve ser apresentado como instalável sem certificado. No macOS, a assinatura ad hoc da montagem local não é Developer ID nem notarização. Nas notas de release, incluir **Mais informações > Executar assim mesmo** no Windows e **Ajustes do Sistema > Privacidade e Segurança > Abrir mesmo assim** no macOS, conforme o [guia de instalação](instalacao.md).
5. As tarefas de assinatura ficam disponíveis para uma fase futura: `darwin:sign:notarize`, `windows:sign`, `windows:sign:installer`, `linux:sign:deb` e `linux:sign:rpm`, com credenciais fora do repositório. Não são bloqueio adotado para esta distribuição sem assinatura. Consulte [Wails CLI](https://v3.wails.io/reference/cli/) e as tarefas do projeto. Preparação e envio dos canais Linux estão no [checklist](deploy-checklist.md#12-publicar-os-canais-linux).

A documentação de privacidade, termos e o canal de relatos, inclusive de IA, ficam no novo institucional Hugo privado, não no painel nem neste repositório. `https://getblup.io/privacidade`, `https://getblup.io/termos` e `https://getblup.io/feedback` estão **a publicar no site institucional**, com contato `help@getblup.io`. Para a Store, conferir publicação dessas páginas, cadastrar a política no Partner Center e declarar IA na ficha. Não haverá botão de denúncia de IA no Studio. Consulte as [políticas verificadas](auditoria/politicas-distribuicao.md).
6. Faça verificação dos artefatos em máquinas de teste, preservando as sessões do usuário. Revise notificações, MP3, instalação/atualização e login na URL de produção. Só o responsável pelo lançamento publica os artefatos após revisão.

### Assinatura da autoatualização CLI

GoReleaser publica `checksums.txt` e `checksums.txt.sig`. A assinatura Ed25519 cobre os bytes exatos do primeiro arquivo; não reformatar nem alterar quebras de linha depois de assinar. `internal/update/keys.go` contém somente chaves públicas. O workflow recebe a chave privada pelo secret CI `BLUP_RELEASE_KEY`, nunca por arquivo versionado. A ferramenta usa a biblioteca padrão [crypto/ed25519](https://pkg.go.dev/crypto/ed25519).

Para preparar uma chave nova ou assinar manualmente, o operador usa, na raiz:

```sh
go run ./scripts/sign-release -genkey
go run ./scripts/sign-release -in dist/checksums.txt
```

`-genkey` é uma preparação única fora do repositório: cria `~/.config/blup-release/release.key` com modo `0600`, diretório `0700`, e a contraparte pública; não sobrescreve um par existente. Cadastre somente a chave pública confiada no código por revisão própria. A assinatura falha se a chave não corresponde à lista confiada. Não gere outro par durante um deploy normal, não imprima a privada e não copie valores para este guia.

O atualizador exige HTTPS, assinatura válida e SHA-256 correspondente, e recusa versão igual ou anterior. A lista pública admite rotação, mas clientes antigos não aprendem uma chave nova sozinhos: publique antes uma versão de transição assinada pela chave antiga que já confie na nova. Clientes que não passaram pela transição podem precisar de uma instalação intermediária verificada ou instalação manual. Revogação e incidente de chave exigem procedimento operacional próprio.

A instalação inicial `install.sh`/`install.ps1` permanece só com SHA-256 do `checksums.txt` baixado da release; não possui essa verificação Ed25519. Os arquivos Linux do Studio têm `studio-checksums.txt` e `studio-checksums.txt.sig`, assinados com a mesma chave. A aba Sobre consulta atualizações na instalação direta e no AppImage, sem substituir seu executável; MSIX/Store, Flatpak e pacotes sob `/usr` encaminham ao próprio canal e não procuram releases. Assinatura Ed25519 da CLI, assinatura nativa dos pacotes e notarização macOS são verificações diferentes.

As referências Cloudflare e Better Auth foram consultadas pelo MCP Context7 e pelos sites oficiais em 02/11/2026. Também foram consultadas as documentações oficiais de Go, Svelte/SvelteKit, Wails, Hono, Drizzle, Zod e TypeScript listadas em [arquitetura](arquitetura.md#referências-oficiais-consultadas). Nenhum secret, conta pessoal ou certificado de assinatura faz parte deste guia.
