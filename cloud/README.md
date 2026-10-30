# Blup Cloud

A conta do Blup na nuvem: cadastro, login, planos e (em breve) a sincronização das notas e a publicação automática dos sites. Roda num único Cloudflare Worker.

## Pilha

| Parte | Escolha |
|---|---|
| API | [Hono](https://hono.dev) no Worker, validação com Zod |
| Login | [Better Auth](https://www.better-auth.com): código por e-mail (`emailOTP`, cria a conta) e passkey (`@better-auth/passkey`). Sem senha |
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

Com `EMAIL_MODE=log` (o padrão no `wrangler.jsonc`), o código de login aparece no terminal em vez de ir por e-mail. As chaves do Turnstile no repositório são as de **teste** da Cloudflare, que sempre aprovam.

Abra pelo endereço `http://localhost:8787` (não `127.0.0.1`): a passkey e a origem confiável do login usam `localhost`.

## Planos e Stripe

- Os planos ficam em `worker/plans.ts` (nome, preço, limites). O preço pago é achado no Stripe pelo *lookup key*, então nenhum ID do Stripe fica no repositório.
- Coloque a chave de **teste** em `.dev.vars` (`STRIPE_SECRET_KEY`) e rode `npm run stripe:setup`: ele cria o produto e o preço de cada plano pago. Rodar de novo não muda nada.
- Webhook no computador: `stripe listen --forward-to localhost:8787/api/auth/stripe/webhook` e copie o `whsec_...` para `STRIPE_WEBHOOK_SECRET`. Em produção, o webhook aponta para `https://<domínio>/api/auth/stripe/webhook` com os eventos `checkout.session.completed` e `customer.subscription.created`, `.updated` e `.deleted`.
- O cliente do Stripe só é criado no primeiro pagamento: contas grátis nunca chegam ao Stripe.

## Banco

- Mudou plugin do Better Auth: `npm run db:auth-schema` (gera `worker/db/auth-schema.ts`).
- Mudou o esquema: `npm run db:generate` e depois `npm run db:migrate:local`.

## Segredos

Este repositório é público. **Nenhum segredo entra no git**: em desenvolvimento eles ficam em `.dev.vars` (ignorado); em produção, `npx wrangler secret put` para `BETTER_AUTH_SECRET`, `TURNSTILE_SECRET_KEY`, `STRIPE_SECRET_KEY` e `STRIPE_WEBHOOK_SECRET`. Os IDs do D1 e do R2 no `wrangler.jsonc` são de exemplo.

## Antes de publicar

- Domínio: trocar `APP_URL`, `PASSKEY_RP_ID` e `MAIL_FROM` no `wrangler.jsonc`. Passkeys criadas em outro domínio deixam de valer.
- Criar o D1 (`npx wrangler d1 create blup-cloud`) e o R2, e aplicar as migrações com `npm run db:migrate:remote`.
- Trocar as chaves de teste do Turnstile pelas reais e ligar o envio de e-mail (`EMAIL_MODE` diferente de `log`).
