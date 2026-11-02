# Deploy de produção

Procedimento preparado em 02/11/2026 a partir de [wrangler.jsonc](../cloud/wrangler.jsonc), [wrangler.sites.jsonc](../cloud/wrangler.sites.jsonc), [package.json](../cloud/package.json) e código do Worker. Este trabalho documenta a operação: não criou recursos, aplicou migrações remotas, cobrou, enviou e-mail ou publicou aplicações. Exemplos não contêm credenciais. [Arquitetura](arquitetura.md) e [segurança](seguranca.md) explicam as fronteiras.

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

No checkout privado de lançamento, preencha o `database_id` devolvido no bloco `env.production.d1_databases` dos dois arquivos Wrangler. Ambos devem apontar ao mesmo banco. Confirme `migrations_dir = "migrations"`. Os buckets são privados, acessados por bindings; não habilite acesso público direto ao bucket das notas. O Worker da conta precisa de `DB`, `NOTES`, `SITES`, `SYNC_LIVE`, `EMAIL` e `OTP_LIMIT`; o dos sites precisa de `DB` e `SITES`. Bindings e vars precisam ser definidos por ambiente, conforme [Wrangler environments](https://developers.cloudflare.com/workers/wrangler/environments/). A criação de buckets segue a [referência R2](https://developers.cloudflare.com/r2/buckets/create-buckets/).

As migrações versionadas vão de `0000_inicial.sql` a `0009_lembretes.sql` nesta revisão. Revise o SQL e faça backup do banco existente antes de aplicá-las. Gere esquema Better Auth ou Drizzle somente quando houver alteração de esquema a desenvolver/revisar; um deploy normal aplica os arquivos já versionados. Com Drizzle, gerar esquema não aplica automaticamente o D1. Veja [Better Auth com Drizzle](https://better-auth.com/docs/adapters/drizzle) e [migrações D1](https://developers.cloudflare.com/d1/reference/migrations/).

```sh
npx wrangler d1 migrations list DB --remote --env production
npx wrangler d1 migrations apply DB --remote --env production
```

O script `npm run db:migrate:remote` não acrescenta `--env production` sozinho. Se usá-lo, passe o ambiente explicitamente: `npm run db:migrate:remote -- --env production`. Não aplique migrações outra vez por meio de um gerador de esquema independente.

Mantenha a migração Durable Object `v1` com `new_sqlite_classes: ["SyncLive"]`, exportada por `worker/index.ts`. Ela é da infraestrutura Workers, separada das migrações SQL do D1.

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
| `OPENROUTER_URL` | Vazio usa o endpoint padrão; não deixar endereço de teste |

Cadastre os segredos por prompt ou pelo gerenciador do ambiente. Nunca cole valores neste guia, em `.toml`, scripts versionados ou histórico de comandos com argumentos contendo a credencial. Cada secret é do ambiente correto, como documentado em [Workers secrets](https://developers.cloudflare.com/workers/configuration/secrets/).

```sh
npx wrangler secret put BETTER_AUTH_SECRET --env production
npx wrangler secret put TURNSTILE_SECRET_KEY --env production
npx wrangler secret put STRIPE_SECRET_KEY --env production
npx wrangler secret put STRIPE_WEBHOOK_SECRET --env production
npx wrangler secret put OPENROUTER_API_KEY --env production
npx wrangler secret put ACCESS_TEAM_DOMAIN --env production
npx wrangler secret put ACCESS_AUD --env production
npx wrangler secret put STAFF_OWNER_EMAIL --env production
```

`BETTER_AUTH_SECRET` deve ser forte e persistente entre publicações. A lista `trustedOrigins` do código contém somente `APP_URL`; não amplie para todos os sites dos usuários. `STAFF_DEV_EMAIL` não deve ser configurado em produção. O Worker dos sites não precisa receber os secrets de login, Stripe, IA ou e-mail. O uso de origem explícita e cookies por host segue [Better Auth security](https://better-auth.com/docs/reference/security). Passkeys de `localhost` não se tornam credenciais de `getblup.io`; confira [rpID e origin](https://better-auth.com/docs/plugins/passkey).

## 4. E-mail e cron

Na Cloudflare, acesse Compute, Email Service, Email Sending e faça Onboard Domain de `getblup.io`. Confira os registros indicados pelo serviço para bounce, SPF, DKIM e DMARC; não invente uma política ou chave DKIM copiando outro domínio. Aguarde a verificação do domínio. O binding `EMAIL` em produção permite `allowed_sender_addresses: ["no-reply@getblup.io"]`. O código usa a API de envio com objeto de remetente, destinatário, assunto, texto e HTML. Referência: [Workers Email Sending API](https://developers.cloudflare.com/email-service/api/send-emails/workers-api/).

Há dois crons em `env.production.triggers`: `0 4 * * *` para limpeza diária às 04:00 UTC e `*/5 * * * *` para lembretes. Ambos invocam `scheduled`; o código escolhe o trabalho pelo texto de `controller.cron`. Verifique que os dois foram instalados. A configuração e o teste local seguem a [referência Cron Triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/).

Para simular sem e-mail real, use `EMAIL_MODE=log`, credenciais de teste e persistência isolada. As portas do time reservadas não devem ser utilizadas. Exemplo de sessão própria, com a pasta de dados escolhida pelo operador:

```sh
npx wrangler dev --port 18820 --persist-to '<pasta-da-sessao>' --test-scheduled
```

Na versão atual da documentação Cloudflare, a rota local é `/cdn-cgi/local/scheduled`, com query `cron`; versões anteriores podem expor outro caminho. Confirme a rota na versão do Wrangler instalada. Para invocar o cron de lembretes nessa sessão:

```sh
curl --get --data-urlencode 'cron=*/5 * * * *' \
  'http://localhost:18820/cdn-cgi/local/scheduled'
```

Cron e log bem-sucedidos não provam entrega: o código atual marca a fila antes do envio e captura falhas do Email Service. O [contrato de lembretes](spec/calendario-e-lembretes.md) explica tentativas, cotas e retenção. Não use envio pago real como teste automatizado.

## 5. Turnstile, login e Access da equipe

Crie o widget Turnstile para o hostname de login de produção. Substitua a site key pública de teste e grave o secret correspondente. O backend valida o token ao pedir OTP e no cadastro inicial com passkey, além de limitar solicitações por IP. A documentação distingue [widget e validação no servidor](https://developers.cloudflare.com/turnstile/get-started/); uma chave de teste que sempre aprova não é a proteção de produção.

No Cloudflare Zero Trust, crie aplicação Access para o hostname inteiro `staff.getblup.io`, cobrindo assets e `/api/staff/*`. Cadastre só as identidades da equipe. Grave a URL HTTPS do time em `ACCESS_TEAM_DOMAIN` e o audience dessa aplicação em `ACCESS_AUD`. `staff.ts` verifica assinatura por JWKS, issuer, audience e e-mail, e depois aplica a tabela de papéis ou o bootstrap de proprietário. Não basta enviar um header de e-mail. Veja [validação Access](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/).

O suporte pode consultar conta/uso e revogar sessões/passkeys; admin também bloqueia, exclui e gerencia equipe. A área pública rejeita rotas de staff quando `STAFF_HOST` está configurado. Não use a conta de cortesia do proprietário como evidência de funcionamento do checkout pago.

## 6. Stripe de produção

Prepare e valide o fluxo antes em um sandbox. Produção tem recursos, preços, portal e segredo de webhook próprios; não reutilize IDs ou signing secret do listener local. O Worker usa chave server-side restrita às operações necessárias. A separação de ambientes e permissões segue [Stripe API keys](https://docs.stripe.com/keys).

1. Confirme os planos de `worker/plans.ts`, incluindo preços mensais/anuais e lookup keys.
2. Crie produtos e preços equivalentes no ambiente de produção. `scripts/stripe-setup.ts` é a automação disponível, exige escrita de Products/Prices e não altera um preço divergente: apenas avisa. Sua execução em produção é uma operação do responsável pelo lançamento, não foi realizada nesta revisão.
3. Configure o Customer Portal no ambiente real, com cartão, faturas e cancelamento. A aplicação redireciona para ele; não incorpora campos de cartão.
4. Crie o webhook HTTPS `https://cloud.getblup.io/api/auth/stripe/webhook`. Use eventos de checkout e assinatura exigidos pelo [plugin Stripe do Better Auth](https://better-auth.com/docs/plugins/stripe): `checkout.session.completed`, `customer.subscription.created`, `.updated` e `.deleted`; inclua o sucesso assíncrono de checkout conforme os meios habilitados.
5. Inclua os eventos que `billing.ts` registra: `invoice.paid`, `invoice.payment_failed`, `charge.refunded`, `charge.dispute.created` e `radar.early_fraud_warning.created`. Grave o signing secret do endpoint em `STRIPE_WEBHOOK_SECRET`.
6. Verifique entrega e resposta ao webhook no ambiente correspondente. Stripe pode repetir eventos, e o registro financeiro usa ID único. O recebimento e a verificação de assinatura seguem [Stripe webhooks](https://docs.stripe.com/webhooks).

A chave de execução precisa de Customers, Checkout Sessions, Customer Portal e Subscriptions conforme as operações usadas, leitura de Prices/Products e leitura de Charges para disputas. O script de setup precisa de permissões mais amplas que o Worker para criar preços. Políticas comerciais, impostos, recibos e tentativas de recuperação de pagamento são escolhas operacionais no Stripe; este documento não afirma que estejam ativadas.

## 7. DNS, TLS e separação dos sites

O Worker da conta tem Custom Domains para `cloud.getblup.io` e `staff.getblup.io`. O institucional no apex `getblup.io` continua independente. O Worker dos sites usa a rota `*.getblup.site/*` da zona `getblup.site`; precisa de DNS curinga com proxy habilitado. A configuração do projeto prevê `AAAA *` com destino `100::` para não depender de um origin externo. Confira a zona e ausência de registros conflitantes. Rotas precisam de DNS proxied, conforme [Workers Routes](https://developers.cloudflare.com/workers/configuration/routing/routes/).

O contrato público é `nome.getblup.site`, sempre HTTPS e sem `www`. O código redireciona HTTP e tenta normalizar `www.nome.getblup.site`, mas TLS acontece antes do Worker. Universal SSL cobre apex e primeiro nível, não garante certificado para esse segundo nível `www.nome`; não anuncie essa variante como funcional sem certificado adicional. Referência: [limitações Universal SSL](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/limitations/).

O bucket `SITES` não deve ser oferecido diretamente por domínio público próprio do bucket: o Worker precisa selecionar o manifesto e aplicar cache/headers. O bucket `NOTES` permanece privado e só é acessado por APIs autenticadas. Domínio próprio de cliente no Blup Cloud ainda é pendência de implementação, mesmo com `customDomain: true` nos planos.

Solicite a inclusão de `getblup.site` na seção PRIVATE da Public Suffix List pelo responsável pelo domínio, seguindo [as regras oficiais da PSL](https://publicsuffix.org/submit/). Isso ajuda os navegadores a tratar cada subdomínio como site registrável distinto, inclusive nas fronteiras de cookies. O PR precisa ser aprovado e a lista precisa chegar aos navegadores; criar um registro DNS não altera a PSL. Não há comprovante de inclusão e propagação no repositório, e o lançamento não deve declarar esse processo concluído sem conferência operacional. PSL não é selo de confiança nem substitui isolamento por origem.

## 8. Gerar e publicar os dois Workers

Depois de recursos, configurações, checks e migrações conferidos pelo responsável:

```sh
npm run build
npx wrangler deploy --env production
npx wrangler deploy -c wrangler.sites.jsonc --env production
```

Não use `npm run deploy` sem argumentos esperando que selecione produção: o script não adiciona esse ambiente. Com ambientes nomeados, o nome remoto normalmente recebe o sufixo `-production`; confira o nome apresentado pelo Wrangler, as rotas e os bindings dos dois Workers. `workers_dev` e preview URLs estão desativados em produção no projeto.

Verifique no ambiente apropriado: `/api/config`, login/passkey/device flow, acesso staff bloqueado fora da equipe, migrações, dois crons, limites dos planos, reserva de slug e publicação de um site de teste. Depois confirme revalidação do HTML e feed, cache de arquivos com fingerprint e atualização do manifesto. Essas verificações operacionais não foram feitas contra produção neste trabalho.

Guarde o identificador da versão anterior dos Workers e um backup pré-migração. Reverter código não reverte D1, objetos ou eventos Stripe. O servidor dos sites conserva apenas a publicação corrente, portanto não oferece histórico de deploys como substituto de backup do conteúdo. Retenha artefatos e plano de restauração em local privado.

## 9. Lançamento do Studio e da CLI

O workflow [release.yml](../.github/workflows/release.yml) publica a CLI por GoReleaser e executa testes com Hugo. Ele não monta, assina ou distribui o Studio. As tarefas desktop estão em [studio/Taskfile.yml](../studio/Taskfile.yml) e [testes e build](spec/testes-e-build.md).

1. Compile o frontend, gere bindings e compile o Studio com a toolchain de `studio/go.mod`. Em Linux, a pilha padrão é GTK4/WebKitGTK 6.0 com GStreamer para MP3. Não execute `run` ou abra uma segunda janela do dono para testar artefatos.
2. Defina a versão real do produto na preparação da release. As flags nativas atuais não injetam `app.Version`, e o nfpm ainda contém `0.1.0`; empacotar sem revisar isso pode produzir identificação incorreta.
3. Use tarefas específicas: Linux AppImage/nfpm; macOS `.app`/DMG e universal quando necessário; Windows NSIS ou MSIX. O frontend precisa ser embutido no binário correspondente.
4. Para assinatura, use `wails3 signing` e o armazenamento de credenciais fora do repositório. As tarefas disponíveis incluem `darwin:sign:notarize`, `windows:sign`, `windows:sign:installer`, `linux:sign:deb` e `linux:sign:rpm`. Confira [Wails CLI](https://v3.wails.io/reference/cli/), [macOS](https://v3.wails.io/guides/build/macos/), [Windows](https://v3.wails.io/guides/build/windows/) e [Linux](https://v3.wails.io/guides/build/linux/).
5. A assinatura ad hoc usada na montagem local de `.app` não substitui Developer ID e notarização para distribuição macOS. Em Windows, assine o executável e o instalador com identidade adequada; em Linux, valide a assinatura dos formatos escolhidos e publique checksums.
6. Faça verificação dos artefatos em máquinas de teste, preservando as sessões do usuário. Revise notificações, MP3, instalação/atualização e login na URL de produção. Só o responsável pelo lançamento publica os artefatos após revisão.

As referências Cloudflare e Better Auth foram consultadas pelo MCP Context7 e pelos sites oficiais em 02/11/2026. Também foram consultadas as documentações oficiais de Go, Svelte/SvelteKit, Wails, Hono, Drizzle, Zod e TypeScript listadas em [arquitetura](arquitetura.md#referências-oficiais-consultadas). Nenhum secret, conta pessoal ou certificado de assinatura faz parte deste guia.
