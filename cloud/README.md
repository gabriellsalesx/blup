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

A conta pode nascer com passkey sem e-mail; um endereço técnico `.invalid` atende ao formato exigido pelo auth. E-mail real vinculado é necessário para assinar. Cadastro grátis não cria Customer no Stripe. O Studio entra por aprovação de aparelho em `/aparelho`; os limites contam sessões reconhecidas como instalações do Studio.

Planos, preços e lookup keys ficam em `worker/plans.ts`. Grátis permite 50 MiB, um aparelho e sync pelo botão. Pro, Max e Ultra têm sync ao vivo, histórico, IA e, respectivamente, um, cinco e dez sites. Notas, histórico e sites compartilham o armazenamento. Max tem 50 e Ultra 100 lembretes por e-mail por mês UTC. A [tabela completa](../docs/spec/blup-cloud-sync.md#planos-executados) distingue limite contratado e função ainda não implementada, como domínio próprio dos sites da nuvem.

O sync recebe nomes, blocos e manifestos cifrados pelo computador. A conta guarda embrulhos da chave e metadados, sem a senha de notas. Conteúdo enviado à IA é outro fluxo: precisa ser legível para processamento remoto. Lembretes recebem somente ID opaco e instante; o e-mail é genérico. Sites publicados são públicos e são servidos em `nome.getblup.site` pelo segundo Worker.

## Staff e operação

Em produção, `cloud.getblup.io` atende a conta e `staff.getblup.io` atende a equipe com Access no host inteiro. A API revalida JWT e papel. `STAFF_OWNER_EMAIL` é secret e faz o bootstrap do admin; a cortesia Ultra requer o e-mail correspondente verificado. `STAFF_DEV_EMAIL` só serve ao ambiente local e não deve ser cadastrado em produção.

Cobrança usa preços por lookup key, checkout/portal hospedados pelo Stripe e webhook assinado em `/api/auth/stripe/webhook`. `scripts/stripe-setup.ts` prepara produtos/preços no ambiente escolhido e avisa sobre divergências. A conta de operação configura sandbox e produção separadamente, com os eventos e permissões descritos em [deploy](../docs/deploy.md#6-stripe-de-produção). Não há segredo Stripe no painel estático.

Cron de cinco minutos processa lembretes; cron diário às 04:00 UTC rareia histórico, limpa blocos e registros antigos de lembretes. A marcação anterior ao envio não é garantia de entrega por e-mail. A remoção de objetos de conta e as políticas de retenção precisam ser verificadas explicitamente; apagar só uma linha de usuário não significa automaticamente apagar cada prefixo R2.

## Checks e mudanças de esquema

```sh
npx tsc -p worker/tsconfig.json --noEmit
npx vitest run
npm run check
npm run build
```

Mudança nos plugins de auth pode exigir regenerar `worker/db/auth-schema.ts` com `npm run db:auth-schema`; alterações de esquema geram migração por `npm run db:generate`. Revise os arquivos antes de aplicá-los. Produção usa `--env production` tanto nas migrações quanto na publicação dos dois Workers; os scripts genéricos não o acrescentam. Os passos completos e os nomes de secrets, sem valores, estão em [deploy](../docs/deploy.md).
