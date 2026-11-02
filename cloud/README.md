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

A conta pode nascer com passkey sem e-mail; um endereço técnico `.invalid` atende ao formato exigido pelo auth. E-mail real vinculado é necessário para assinar. Cadastro grátis não cria Customer no Stripe. O Studio entra por aprovação de aparelho em `/aparelho`; os limites contam sessões marcadas pelo servidor ao resgatar uma aprovação de aparelho; o User-Agent serve apenas como nome de exibição.

Planos, preços e lookup keys ficam em `worker/plans.ts`. Grátis permite 50 MiB, um aparelho e sync pelo botão. Pro, Max e Ultra têm sync ao vivo, histórico, IA e, respectivamente, um, cinco e dez sites. Notas, histórico e sites compartilham o armazenamento. Max tem 50 e Ultra 100 lembretes por e-mail por mês UTC. A [tabela completa](../docs/spec/blup-cloud-sync.md#planos-executados) detalha os limites. Free/Pro permitem arquivos de 10/25 MiB e Max/Ultra de 100 MiB; o texto comercial usa MB. Cada conta tem no máximo 50 cofres. Max/Ultra podem vincular um domínio próprio por site, junto com o endereço getblup.site.

O sync recebe nomes, blocos e manifestos cifrados pelo computador. A conta guarda embrulhos da chave e metadados, sem a senha de notas. Conteúdo enviado à IA é outro fluxo: precisa ser legível para processamento remoto. Lembretes recebem somente ID opaco e instante; o e-mail é genérico. Sites publicados são públicos e são servidos em `nome.getblup.site` pelo segundo Worker.

## Staff e operação

Em produção, `cloud.getblup.io` atende a conta e `staff.getblup.io` atende a equipe com Access no host inteiro. A API revalida JWT e papel. `STAFF_OWNER_EMAIL` é secret e faz o bootstrap do admin; a cortesia Ultra requer o e-mail correspondente verificado. `STAFF_DEV_EMAIL` só serve ao ambiente local e não deve ser cadastrado em produção.

Cobrança usa preços por lookup key, checkout/portal hospedados pelo Stripe e webhook assinado em `/api/auth/stripe/webhook`. `scripts/stripe-setup.ts` prepara produtos/preços no ambiente escolhido e avisa sobre divergências. A conta de operação configura sandbox e produção separadamente, com os eventos e permissões descritos em [deploy](../docs/deploy.md#6-stripe-de-produção). Não há segredo Stripe no painel estático.

Cron de cinco minutos processa até 100 lembretes, reservas expiradas, limpeza adiada de publicação, cancelamento de cobrança e avisos de downgrade. E-mail com falha pode ser tentado até três vezes, em rodadas posteriores; só o envio aceito pelo serviço entra na cota de 50/100 por mês UTC. Isso não garante entrega na caixa de entrada. O cron diário às 04:00 UTC rareia histórico, limpa blocos e registros antigos de lembretes em lotes e executa purgas vencidas de contas.

## Checks e mudanças de esquema

```sh
npx tsc -p worker/tsconfig.json --noEmit
npx vitest run
npm run check
npm run build
```

Mudança nos plugins de auth pode exigir regenerar `worker/db/auth-schema.ts` com `npm run db:auth-schema`; alterações de esquema geram migração por `npm run db:generate`. Revise os arquivos antes de aplicá-los. Produção usa `--env production` tanto nas migrações quanto na publicação dos dois Workers; os scripts genéricos não o acrescentam. Os passos completos e os nomes de secrets, sem valores, estão em [deploy](../docs/deploy.md).

## Contratos da rodada 2

As cotas de criação de sites/cofres, armazenamento compartilhado e IA usam reservas atômicas no D1 antes da operação externa. Falhas devolvem a reserva; reservas abandonadas expiram. O registro de IA usa `usage.cost` finito e não negativo, inclusive em resposta incompleta, antes de validar a tradução. Sem custo informado, usa a estimativa. Tradução tem teto de saída calculado a partir da entrada. Áudio considera o preço do modelo escolhido e o custo informado em cabeçalho, quando houver.

Uma publicação por site obtém uma trava com expiração; a segunda recebe `409 DEPLOY_BUSY`. Manifestos anteriores e arquivos substituídos ficam por pelo menos dez minutos. A limpeza do cron é paginada, então pode terminar depois desse mínimo. Arquivos mantidos por um manifesto ainda válido ficam protegidos. Objetos temporários e arquivos sem registro também são varridos após a espera.

Excluir a conta revoga sessões, bloqueia login e tira os sites do ar imediatamente. Os dados cifrados e o histórico ficam preservados para recuperação por suporte durante 30 dias, por `POST /api/staff/users/:id/restore`. Isso não reinicia cobrança nem restaura sessões antigas. O cancelamento da assinatura é tentado imediatamente e nas rodadas do cron se o Stripe falhar. A partir de 30 dias, o cron diário apaga os prefixos R2, os Custom Hostnames externos e as linhas D1. Falha na limpeza externa mantém a identificação necessária para repetir a operação. Contas cuja purga já começou não podem ser restauradas.

Ao passar de pago para Grátis, os sites continuam 14 dias e é enviado um aviso genérico com a marca Blup. Depois ficam fora do ar, com os arquivos preservados. Voltar a pagar religa o endereço getblup.site. A redução Max/Ultra para Pro preserva os sites existentes, bloqueia criar além do limite novo e deixa domínio próprio inativo enquanto o plano não o inclui. O prazo de 14 dias aplica-se à passagem para Grátis. A exclusão de conta prevalece sobre qualquer prazo.

### Domínio próprio

Rotas com a sessão do dono: `GET`, `POST` e `DELETE /api/sites/:id/domain`. POST aceita somente `{hostname}` e exige Max/Ultra. GET retorna o objeto abaixo ou `null`; DELETE retorna `{ok:true}`. GET/DELETE continuam disponíveis após downgrade.

```ts
type DomainInfo = {
  hostname: string;
  status: 'pending_ownership' | 'pending' | 'active' | 'error';
  ssl: 'pending' | 'active' | 'error';
  records: {type: 'TXT' | 'CNAME'; name: string; value: string}[];
};
```

No máximo dez registros, com valor de até 1024 caracteres. Hostname é normalizado por IDNA e não aceita esquema, caminho, porta, IP, localhost, .local, .internal nem getblup.io/getblup.site e seus subdomínios. Antes de criar o Custom Hostname na Cloudflare, o dono publica o TXT aleatório `_blup.<hostname>` retornado pelo servidor. Um pedido sem prova não reserva exclusividade sobre o hostname: só a posse comprovada recebe a restrição única do D1. Isso evita reservar domínios de terceiros sem controlar o DNS. Consultas DNS usam um resolvedor HTTPS fixo, sem buscar URLs do visitante. O conteúdo só é servido quando a posse, a ativação e o SSL estiverem confirmados.

O botão Verificar usa GET. Poll de 60 segundos com a tela aberta é suficiente; o servidor limita consultas de verificação. O CNAME vem na resposta, não deve ser fixado na interface. Apex exige flattening/ALIAS no provedor DNS, sem promessa de IP dedicado. A resolução de domínio tem cache de até 30 segundos; a situação da conta e do plano é conferida antes de servir conteúdo. O redirecionamento www só atua em requisições que chegam ao Worker; ele não cria um segundo certificado/hostname por conta própria.

### Limites de abuso e de corpos

`ABUSE_LIMIT`: 120/min por conta e por IP para IA, criação de site/cofre, domínio, check de slug, lembretes PUT e remoção de site. `BULK_LIMIT`: 500/10 s por conta para uploads, missing, deploys, blocos/manifestos e downloads da sync. Ao negar, ambos retornam `429 TOO_MANY_REQUESTS` e `Retry-After` de 60 ou 10 segundos, respectivamente. A Cloudflare aplica esses limitadores por localização; as cotas financeiras e de armazenamento continuam no D1, que é a autoridade global.

JSON geral: 96 KiB. Metadados de publicação: 8 MiB. Multipart de versão: 34 MiB, com manifesto cifrado de até 32 MiB. Bloco: 4 MiB mais 64 bytes. Arquivo de site: até o limite do plano, no máximo 100 MiB. O servidor verifica Content-Length antes de receber e conta os bytes durante a leitura. Upload de arquivo maior que 4 MiB precisa informar Content-Length. Arquivos grandes são processados em fluxo, com digest incremental e staging R2; a validação não carrega o arquivo todo no isolate de 128 MiB. Staging e cópia final acrescentam operações R2, consideradas no orçamento.

O painel usa `hc<AppType>` e `InferResponseType`. `worker/export-rpc.mjs` emite `src/lib/rpc.d.ts` diretamente da cadeia completa de rotas Hono, antes do check/build. Não edite a declaração gerada. Isso mantém os tipos JSON sem importar os globais do runtime Worker para o DOM do painel.

## Antes de publicar

- Aplicar de 0000 até `0010_cotas_retencao_dominios.sql`, gerada com drizzle-kit, antes do novo Worker. As migrações anteriores permanecem intactas. Em atualização de instalação já usada, revogar as sessões antigas do Studio que ainda não receberam marca de aparelho e pedir novo login.
- Substituir o ID de D1 fictício nas duas configurações, preparar R2, Durable Object e crons e conferir todos os segredos do [guia de deploy](../docs/deploy.md). Configurar os bindings `OTP_LIMIT`, `ABUSE_LIMIT` e `BULK_LIMIT` no ambiente de produção.
- Configurar Access no host inteiro da staff, Turnstile real, Email Service com remetente permitido, Stripe/webhook de produção e chaves de IA. `EMAIL_MODE` de produção deve ser send. Não copiar `.dev.vars` local para produção.
- Para domínio próprio, habilitar Cloudflare for SaaS na zona dos sites. Cadastrar o secret `CF_API_TOKEN` com permissão SSL and Certificates Write apenas nessa zona e a var `CF_ZONE_ID`. Informar `SITES_CNAME_TARGET` com o CNAME público do serviço. Sem isso, a API retorna `DOMAIN_UNAVAILABLE`.
- Preparar o fallback origin e o CNAME target com DNS proxied. `wrangler.sites.jsonc` usa a rota `*/*` na zona getblup.site para alcançar também Custom Hostnames, conforme a [documentação Cloudflare](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/start/advanced-settings/worker-as-origin/). Conferir se algum outro serviço dessa zona precisa de rota de exclusão. Isso não altera a zona getblup.io do painel.
- Se usar a cortesia do dono sem assinatura Stripe, cadastrar `STAFF_OWNER_EMAIL` como secret também no Worker dos sites. Ambos precisam reconhecer a mesma conta verificada. Nenhum valor de segredo fica neste repositório.
- Criar alerta de gasto da conta no painel Cloudflare, em Manage Account > Billing > Billable Usage > Create budget alert, escolhendo valor e destinatário aprovados pelo dono. Também é possível abrir esse fluxo pelo resumo de gastos de Workers, D1 e R2. É um aviso por e-mail, sem bloqueio automático de gastos. A [documentação de orçamento](https://developers.cloudflare.com/billing/manage/budget-alerts/) distingue alerta financeiro e notificações por métrica. Acompanhar linhas lidas/escritas D1, operações e armazenamento R2, requests/CPU Workers, e-mails e hostnames provisionados.
- Rodar check, testes e build; testar o fluxo completo em staging com serviços simulados ou sandbox antes de habilitar produção. Esta rodada não publicou nada e não consultou credenciais, DNS ou contas de produção para marcar esses itens como feitos.
