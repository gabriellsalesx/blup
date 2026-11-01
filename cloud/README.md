# Blup Cloud

A conta do Blup na nuvem: cadastro, login, planos e (em breve) a sincronização das notas e a publicação automática dos sites. Roda num único Cloudflare Worker.

## Pilha

| Parte | Escolha |
|---|---|
| API | [Hono](https://hono.dev) no Worker, validação com Zod |
| Login | [Better Auth](https://www.better-auth.com): código por e-mail (`emailOTP`, cria a conta) e passkey (`@better-auth/passkey`), que também cria conta sozinha. Sem senha |
| Proteção | Turnstile no pedido de código e Rate Limiting dos Workers |
| Dados | D1 com Drizzle ORM; R2 para as notas (cifradas no computador da pessoa) |
| E-mail | Email Service da Cloudflare (precisa do plano Workers Paid) |
| Painel | SvelteKit em modo SPA (adapter-static), Tailwind, Inter local e ícones Lucide, servido como assets do mesmo Worker |

## Pastas

- `worker/`: a API. `index.ts` (rotas), `auth.ts` (Better Auth com o `env` da Cloudflare), `auth-options.ts` (as opções, também usadas pela CLI do Better Auth), `email.ts`, `db/` (esquema Drizzle).
- `src/`: o painel (rotas `/entrar` e `/conta`).
- `migrations/`: SQL do D1, gerado pelo `drizzle-kit`.
- `static/`: fonte Inter e o Blup.

## Rodar no computador

```sh
npm install
cp .dev.vars.example .dev.vars   # e troque BETTER_AUTH_SECRET
npm run db:migrate:local
npm run dev                      # http://localhost:8787
```

Com `EMAIL_MODE=log` (o padrão no `wrangler.jsonc`), o código de login aparece no terminal em vez de ir por e-mail. Em produção os e-mails saem de `no-reply@getblup.io`: o domínio `getblup.io` precisa estar em **Email Service → Email Sending** na Cloudflare ("Onboard Domain", que cria os registros SPF, DKIM e DMARC), e o binding só aceita esse remetente. As chaves do Turnstile no repositório são as de **teste** da Cloudflare, que sempre aprovam.

Abra pelo endereço `http://localhost:8787` (não `127.0.0.1`): a passkey e a origem confiável do login usam `localhost`.

## Conta só com passkey

- "Criar conta com passkey" cria a conta sem e-mail (o Better Auth exige um, então ela ganha um endereço em `sem-email.blup.invalid`, domínio que nunca recebe e-mail). A conta só é criada depois que a passkey é confirmada, com Turnstile e limite por IP.
- No Perfil, "Vincular" manda um código para o e-mail novo e liga ele à conta. Depois disso a pessoa entra tanto pela passkey quanto pelo código.
- Sem e-mail não dá para assinar (recibos e recuperação da conta). Trocar um e-mail já vinculado é pelo suporte, por enquanto.

## Planos e Stripe

- Os planos ficam em `worker/plans.ts` (nome, preço, limites). O preço pago é achado no Stripe pelo *lookup key*, então nenhum ID do Stripe fica no repositório.
- Use um **sandbox** só do Blup e uma **chave restrita** (`rk_`), como o Stripe recomenda. Permissões da chave do Worker: *Customers* (escrita), *Checkout Sessions* (escrita), *Customer portal* (escrita), *Subscriptions* (escrita, para cancelar ao excluir a conta), *Prices* e *Products* (leitura), *Charges* (leitura, para os alertas de contestação e fraude). O `npm run stripe:setup` cria produto e preço, então roda com uma chave que tenha *Products* e *Prices* em escrita (só no seu computador).
- `npm run stripe:setup` cria o produto e o preço de cada plano pago. Rodar de novo não muda nada.
- **Portal do cliente:** a configuração padrão só se cria no painel do Stripe (Configurações > Billing > Portal do cliente): ligar histórico de faturas, troca de cartão e cancelamento no fim do período, e salvar. Faça no sandbox e de novo no modo real.
- **Webhook** em `/api/auth/stripe/webhook`. No computador: `stripe listen --forward-to localhost:8787/api/auth/stripe/webhook` e o `whsec_...` vai para `STRIPE_WEBHOOK_SECRET`. Eventos: `checkout.session.completed`, `checkout.session.async_payment_succeeded`, `customer.subscription.created`, `.updated` e `.deleted` (a assinatura, pelo Better Auth) e `invoice.paid`, `invoice.payment_failed`, `charge.refunded`, `charge.dispute.created`, `radar.early_fraud_warning.created` (guardados em `billing_event` e mostrados na staff).
- No painel do Stripe, ligar a **recuperação automática** (Smart Retries e os e-mails de pagamento falho) e o **monitoramento de impostos** (Stripe Tax), conforme o plano de integração do Stripe.
- **Segurança:** CSP gerado pelo SvelteKit (`vite.config.ts`) e os demais cabeçalhos em `static/_headers` e no Worker. Trava de commit com chave: `git config core.hooksPath .githooks` (uma vez por clone).
- O cliente do Stripe só é criado no primeiro pagamento: contas grátis nunca chegam ao Stripe.

## Staff (suporte e administração)

- Painel em `/staff` e API em `/api/staff`. Em produção, crie no Cloudflare Access uma aplicação para esses dois caminhos e libere só os e-mails da equipe. O Worker confere o token do Access de novo (`ACCESS_TEAM_DOMAIN` e `ACCESS_AUD`) e o papel na tabela `staff`.
- O primeiro admin é o `STAFF_OWNER_EMAIL` (secret). Os outros entram pela tela Equipe.
- O `STAFF_OWNER_EMAIL` também ganha o plano Ultra de cortesia (sem Stripe), desde que o e-mail da conta seja o mesmo e esteja confirmado. Só ele: a staff adicionada pela tela Equipe não ganha.
- Papéis: **Suporte** vê contas, encerra sessões e remove passkeys perdidas. **Admin** também bloqueia, exclui contas (cancelando no Stripe) e cuida da equipe.
- Toda ação fica na tabela `staff_log`. A staff nunca lê notas (são cifradas no computador da pessoa) e nunca entra como o usuário.
- No computador, `STAFF_DEV_EMAIL` em `.dev.vars` deixa entrar sem o Access, só pelo `localhost`.

## IDs

- Tabelas do Better Auth (conta, sessão, passkey, assinatura): **UUIDv7** (RFC 9562) em texto, gerado pelo pacote `uuid` (contador dentro do milissegundo, já que o Worker congela o `Date.now()` durante o pedido, e aleatório de `crypto.getRandomValues`). Ordenado no índice do D1 e impossível de adivinhar.
- Tabelas internas de volume alto (registro da staff e, depois, a sincronização): `INTEGER PRIMARY KEY`, o próprio rowid do SQLite, sem índice extra nem `AUTOINCREMENT`.

## Paginação

- Toda lista que pode crescer usa paginação por cursor (`src/lib/pager.svelte.ts` com o componente `Pager.svelte`), nunca `OFFSET`: o D1 cobra cada linha lida, e o `OFFSET` lê (e cobra) todas as linhas que pula.
- A API responde `{ items, nextCursor }`. O cursor é a chave do último item: os IDs UUIDv7 e os inteiros do registro seguem a ordem de criação, então a próxima página é um intervalo na chave primária.
- Toda consulta passa por um índice (conferido com `EXPLAIN QUERY PLAN`, que deve dizer `SEARCH`, nunca `SCAN` numa tabela grande). A busca da staff pelo começo do e-mail é um intervalo no índice do e-mail (o Better Auth guarda e-mails em minúsculas), não um `LIKE`. Os índices que o Better Auth não cria ficam em `migrations/0001_indices.sql`.
- Contagens leem a tabela inteira: o resumo da staff fica 5 minutos no cache do Worker.

## Banco

- Mudou plugin do Better Auth: `npm run db:auth-schema` (gera `worker/db/auth-schema.ts`).
- Mudou o esquema: `npm run db:generate` e depois `npm run db:migrate:local`.

## Segredos

Este repositório é público. **Nenhum segredo entra no git**: em desenvolvimento eles ficam em `.dev.vars` (ignorado); em produção, `npx wrangler secret put` para `BETTER_AUTH_SECRET`, `TURNSTILE_SECRET_KEY`, `STRIPE_SECRET_KEY` e `STRIPE_WEBHOOK_SECRET`. Os IDs do D1 e do R2 no `wrangler.jsonc` são de exemplo.

## Endereços

Um Worker só, com dois domínios (`env.production` no `wrangler.jsonc`):

- **cloud.getblup.io**: o painel do usuário (`/entrar`, `/conta`) e a API (`/api`).
- **staff.getblup.io**: a staff, na raiz (`/`, `/contas/:id`, `/equipe`, `/registro`), com o Cloudflare Access na frente do domínio inteiro. O `reroute` do SvelteKit (`src/hooks.ts`) mostra as telas de `routes/staff` ali.
- Com `STAFF_HOST` definido, cada domínio responde 404 para as telas e a API do outro. No computador (`STAFF_HOST` vazio) tudo fica em `localhost:8787`, com a staff em `/staff`.
- O site institucional (getblup.io) é outro repositório.

## Antes de publicar

- Publicar com `npx wrangler deploy --env production` (os domínios já estão no `wrangler.jsonc`). Passkeys valem em `getblup.io` e subdomínios; as de teste do `localhost` não passam para lá.
- Criar o D1 (`npx wrangler d1 create blup-cloud`) e o R2, e aplicar as migrações com `npm run db:migrate:remote`.
- Trocar as chaves de teste do Turnstile pelas reais e ligar o envio de e-mail (`EMAIL_MODE` diferente de `log`).
