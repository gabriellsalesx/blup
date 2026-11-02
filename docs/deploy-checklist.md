# Checklist de deploy

Siga a ordem abaixo. **BLOQUEANTE** significa concluir antes de liberar a função ou o canal indicado. Marque somente depois de conferir no ambiente real. Este documento não comprova que contas, recursos, assinaturas ou publicações já foram configurados.

O [guia técnico](deploy.md) detalha os contratos; o [guia de instalação](instalacao.md) é para quem vai abrir o Studio. Revisão: 02/11/2026, rodada 4.

## 1. Separar lançamento e ambiente de teste

- [ ] **BLOQUEANTE:** escolher a revisão e a versão `x.y.z` que serão distribuídas. Guardar a revisão anterior dos Workers e um backup privado do D1 existente.
- [ ] Confirmar acesso às contas Cloudflare, Stripe e GitHub do projeto, com autenticação de dois fatores e recuperação guardada fora do repositório.
- [ ] Preparar um checkout local de lançamento com os lockfiles revisados. Instalar as dependências pelo lockfile em `cloud/` e `studio/frontend/`. Conferir Go pelos dois `go.mod` e Node/Wrangler pelos arquivos do projeto.
- [ ] **BLOQUEANTE:** concluir os testes abaixo, o `npm run check` e `npm run build` de cloud e os testes/build do frontend Studio. `go test` da raiz não atravessa o módulo `studio/`.

Na raiz:

```sh
go vet ./...
go test ./...
(cd studio && go vet . && go test ./...)
(cd cloud && npx tsc -p worker/tsconfig.json --noEmit && npx vitest run)
(cd studio/frontend && npx svelte-check)
```

- [ ] Homologar checkout, webhook, portal, limites, carências e exclusão em sandbox ou testes isolados. Usar IA e e-mail simulados nos testes automatizados. Não avançar a data nem forçar purgas em contas reais.

## 2. Ativar contas, domínios e controle de gasto

- [ ] **BLOQUEANTE para a nuvem:** confirmar as zonas `getblup.io` e `getblup.site` ativas no DNS Cloudflare, R2 habilitado e conta com Workers Paid para envio de e-mail a destinatários arbitrários. [Email Service: planos](https://developers.cloudflare.com/email-service/platform/pricing/).
- [ ] **BLOQUEANTE para cobrança:** ativar a conta comercial Stripe e conferir o ambiente de produção. Sandbox e produção têm chaves, catálogo, portal e webhook próprios.
- [ ] Criar alerta de gasto em **Manage Account > Billing > Billable Usage > Create budget alert**, com limite e destinatário operacional. Ele avisa e não interrompe gastos. Acompanhar Workers/CPU, D1, R2, Email Service e Custom Hostnames. [Orçamento Cloudflare](https://developers.cloudflare.com/billing/manage/budget-alerts/).
- [ ] Para oferecer IA, preparar uma chave OpenRouter com limite de gasto e saldo controlados. Manter recarga automática desligada no lançamento inicial; limite externo e cota do aplicativo são controles distintos. [Limites de chave](https://openrouter.ai/docs/api/api-reference/api-keys/create-keys), [recarga](https://openrouter.ai/support).

## 3. Criar D1 e os dois buckets privados

Daqui até a publicação dos Workers, os comandos pressupõem `cloud/` e a sessão Cloudflare da conta correta.

Criar somente os recursos que ainda não existem:

```sh
npx wrangler d1 create blup-cloud
npx wrangler r2 bucket create blup-cloud-notes
npx wrangler r2 bucket create blup-cloud-sites
```

- [ ] **BLOQUEANTE:** substituir o placeholder `database_id` no `env.production.d1_databases` de [wrangler.jsonc](../cloud/wrangler.jsonc) e [wrangler.sites.jsonc](../cloud/wrangler.sites.jsonc), no checkout reservado ao lançamento. Os dois devem apontar ao mesmo D1. Não publicar identificadores operacionais ou segredos no Git.
- [ ] Conferir `NOTES -> blup-cloud-notes` e `SITES -> blup-cloud-sites`. O Worker dos sites usa somente `DB` e `SITES`; os buckets continuam privados, sem acesso público direto. [Bindings por ambiente](https://developers.cloudflare.com/workers/wrangler/environments/).
- [ ] Conferir `SYNC_LIVE` e sua migração Workers `v1`, classe SQLite `SyncLive`. Essa migração não substitui o SQL do D1.

## 4. Planejar a migração e aplicar o SQL

- [ ] **BLOQUEANTE:** conferir backup e revisar a sequência [migrations/](../cloud/migrations). Nesta rodada ela vai de `0000` até `0013_webhook_stripe_atual.sql`.
- [ ] Em instalação nova, aplicar toda a sequência antes dos Workers. Em instalação existente, **não aplicar `0012` com o código antigo atendendo tráfego ou cron**: ele ainda escrevia os campos removidos. Preparar e ensaiar uma atualização em etapas compatíveis ou uma janela de manutenção, suspendendo tráfego/gravações e crons antigos, aplicando SQL e publicando os dois Workers novos antes de reabrir. A sequência abaixo pressupõe essa condição; rollback de código antigo pode ser incompatível com o schema novo.

```sh
npx wrangler d1 migrations list DB --remote --env production
npx wrangler d1 migrations apply DB --remote --env production
npx wrangler d1 migrations list DB --remote --env production
```

- [ ] Confirmar que não ficaram migrações pendentes. `0011` prepara carências e sync transacional; `0012` remove `sync_account.previous_key` e `site_domain.disabled_at`, sem consumidor; `0013` cria recibos de webhook e índices. As migrações anteriores permanecem intactas.
- [ ] Em atualização de instalação anterior à marca de aparelho, planejar revogar sessões Studio antigas e solicitar novo login, conforme [cloud/README.md](../cloud/README.md).

Aplicar SQL versionado é o fluxo de lançamento. Não gerar outro schema Better Auth ou Drizzle durante esse passo. [Migrações D1](https://developers.cloudflare.com/d1/reference/migrations/).

## 5. Preparar Email Service, Turnstile e Access

- [ ] **BLOQUEANTE para login por e-mail:** em **Compute > Email Service > Email Sending**, fazer **Onboard Domain** de `getblup.io`. Conferir os registros SPF, DKIM, DMARC e de retorno propostos, sua verificação e o remetente `no-reply@getblup.io`. Preservar o serviço de e-mail já usado pelo domínio. [Configuração oficial](https://developers.cloudflare.com/email-service/get-started/send-emails/).
- [ ] Conferir o binding `EMAIL` de produção com `allowed_sender_addresses` limitado ao remetente do produto, `MAIL_FROM` correspondente e `EMAIL_MODE=send`.
- [ ] **BLOQUEANTE para login público:** criar widget Turnstile para `cloud.getblup.io`, trocar a site key de teste e guardar o secret correspondente. [Widget e validação](https://developers.cloudflare.com/turnstile/get-started/).
- [ ] **BLOQUEANTE para staff:** criar aplicação Access para o hostname inteiro `staff.getblup.io`, incluindo assets e `/api/staff/*`. Permitir somente identidades da equipe. Guardar URL HTTPS do time e audience em `ACCESS_TEAM_DOMAIN` e `ACCESS_AUD`. [JWT Access](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/).
- [ ] Definir `STAFF_OWNER_EMAIL` como secret para o administrador inicial. Conferir que sua identidade no Access corresponde; o valor não vai para os docs ou vars públicas.

## 6. Configurar catálogo, portal e webhook Stripe

- [ ] **BLOQUEANTE para vender planos:** criar produtos ativos Pro, Max e Ultra no Dashboard de produção. Escolher valor e moeda e criar preço recorrente por unidade, mensal ou anual, com intervalo de uma unidade e uso `licensed`. [Produtos e preços](https://docs.stripe.com/products-prices/manage-prices).

| Plano | Lookup mensal | Lookup anual |
| --- | --- | --- |
| Pro | `blup_pro_mensal` | `blup_pro_anual` |
| Max | `blup_max_mensal` | `blup_max_anual` |
| Ultra | `blup_ultra_mensal` | `blup_ultra_anual` |

- [ ] Atribuir os lookup keys exatos. `npm run stripe:setup` agora **só confere** o catálogo; não cria produtos/preços nem define valor comercial. Só executá-lo com a chave do ambiente escolhido, recebida pelo gerenciador privado de segredos.
- [ ] Configurar `metadata.blup_plan` no produto ou Price como `pro`, `max` ou `ultra`. Isso permite reconhecer assinaturas no Price antigo depois de transferir o lookup key; não deduzir o plano do nome comercial.
- [ ] Conferir `/api/plans`: cada período pago disponível precisa trazer `prices.monthly` ou `prices.yearly` com `unitAmount` e `currency`. `null` desabilita a compra daquele período; Grátis continua gratuito. O cache é de dez minutos por isolate e mantém o último catálogo válido em falha.
- [ ] Para mudar valor/moeda, criar outro Price e transferir o lookup key com `transfer_lookup_key`. Assinaturas existentes continuam no Price anterior até migração explícita. Conferir painel e Checkout depois de atualizar o catálogo.
- [ ] **BLOQUEANTE:** ativar o Customer Portal de produção, conferir faturas, pagamento e cancelamento. Manter quantidade de assinatura sem edição para estes planos; revisar mudança de plano e prorrata conforme a política comercial definida. [Portal](https://docs.stripe.com/customer-management/configure-portal).
- [ ] **BLOQUEANTE:** criar destino webhook HTTPS em `https://cloud.getblup.io/api/auth/stripe/webhook`, compatível com a API `2026-10-28.endive` usada pelo SDK do projeto. Guardar o signing secret desse endpoint em `STRIPE_WEBHOOK_SECRET`, separado da API key e do listener local.

Selecionar os eventos usados pela reconciliação própria de assinatura e pelo [registro financeiro](../cloud/worker/billing.ts). O plugin continua preparando checkout/portal, com seus handlers webhook/success desativados:

```text
checkout.session.completed
customer.subscription.created
customer.subscription.updated
customer.subscription.deleted
invoice.paid
invoice.payment_failed
charge.refunded
charge.dispute.created
radar.early_fraud_warning.created
```

- [ ] **BLOQUEANTE para cobrar:** conferir assinatura, duplicação, ordem invertida e falha/repetição no sandbox. O handler próprio [billing.ts](../cloud/worker/billing.ts) busca o estado atual no Stripe sob lease por assinatura, grava estado e recibo no mesmo batch D1 e deduplica `event.id`. Uma lease por conta protege a adoção de referência pendente compartilhada entre assinaturas distintas. Os caminhos webhook/success do plugin estão desativados; success só redireciona para URL local, sem conceder plano. Em produção, monitorar as primeiras entregas no Workbench. [Webhooks Stripe](https://docs.stripe.com/webhooks).
- [ ] Conferir as permissões da chave server-side: leitura de Prices, Products, Invoices, Charges, Disputes e Early Fraud Warnings, mais as operações usadas de Customers, Checkout Sessions, Portal e Subscriptions. A reconciliação consulta objetos atuais, além de validar a assinatura. Preferir chave restrita; nunca colocar chave server-side no frontend. [Chaves Stripe](https://docs.stripe.com/keys).

## 7. Conferir DNS e Cloudflare for SaaS

| Host ou zona | Destino esperado |
| --- | --- |
| `getblup.io` | Institucional de outro repositório, preservado neste deploy |
| `cloud.getblup.io` | Custom Domain do Worker `blup-cloud-production` |
| `staff.getblup.io` | Mesmo Worker, sempre protegido pelo Access |
| `*.getblup.site` | DNS com proxy e rota da zona para `blup-sites-production` |

- [ ] **BLOQUEANTE para sites:** preparar o DNS curinga de `getblup.site`, com registro originless proxied, como AAAA `100::`. Conferir o certificado para `nome.getblup.site`; não presumir cobertura de `www.nome.getblup.site` pelo mesmo wildcard.
- [ ] **BLOQUEANTE para domínio próprio Max/Ultra:** ativar Cloudflare for SaaS na zona `getblup.site`. Criar fallback origin com DNS originless e configurá-lo no SaaS. Definir e conferir o CNAME target do serviço. [Configuração SaaS](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/start/getting-started/).
- [ ] Conferir a rota `*/*` dessa zona para o Worker dos sites, como no arquivo Wrangler. Ela inclui Custom Hostnames; uma rota apenas para `*.getblup.site/*` é insuficiente. [Worker como origin](https://developers.cloudflare.com/cloudflare-for-platforms/cloudflare-for-saas/start/advanced-settings/worker-as-origin/).
- [ ] Preparar token Cloudflare limitado à zona SaaS, com permissão `SSL and Certificates Write` para ler, criar e apagar Custom Hostnames. Informar `CF_ZONE_ID` e `SITES_CNAME_TARGET`; guardar somente o token em `CF_API_TOKEN`.
- [ ] Testar um domínio controlado pelo projeto: criar a prova TXT `_blup.<hostname>`, o CNAME e os registros de validação retornados. Só considerar pronto com propriedade comprovada, `status=active` e `ssl=active`. Para apex, conferir flattening/ALIAS no provedor DNS. Não há redirecionamento entre domínio próprio e endereço `getblup.site`.
- [ ] Solicitar inclusão de `getblup.site` na seção PRIVATE da Public Suffix List pelo responsável do domínio e acompanhar aceitação/propagação. A lista consultada em 02/11/2026 não continha a entrada. Isso é uma pendência externa; não declarar isolamento de cookies baseado na PSL como pronto. [Procedimento PSL](https://publicsuffix.org/submit/).

## 8. Revisar vars e preparar secrets

- [ ] **BLOQUEANTE:** conferir os valores não secretos em `env.production`, sem herdar por engano os padrões locais:

| Var | Valor ou origem |
| --- | --- |
| `APP_URL` | `https://cloud.getblup.io` |
| `PASSKEY_RP_ID` | `getblup.io` |
| `STAFF_HOST` | `staff.getblup.io` |
| `SITES_DOMAIN` | `getblup.site`, nos dois Workers |
| `MAIL_FROM` | Nome Blup e remetente `no-reply@getblup.io` |
| `EMAIL_MODE` | `send` |
| `TURNSTILE_SITE_KEY` | Widget real de produção |
| `CF_ZONE_ID` | Zona SaaS configurada, não um placeholder |
| `SITES_CNAME_TARGET` | Host do CNAME do serviço, sem esquema |
| `OPENROUTER_URL` | Vazio para usar o provedor padrão |

- [ ] Preparar os secrets abaixo no gerenciador privado do ambiente, sem copiar valores para Git, docs ou argumentos de terminal:

```text
BETTER_AUTH_SECRET
TURNSTILE_SECRET_KEY
STRIPE_SECRET_KEY
STRIPE_WEBHOOK_SECRET
OPENROUTER_API_KEY
ACCESS_TEAM_DOMAIN
ACCESS_AUD
STAFF_OWNER_EMAIL
CF_API_TOKEN
```

- [ ] O Worker dos sites recebe `STAFF_OWNER_EMAIL` se a cortesia Ultra do proprietário for usada. Ele não recebe os secrets de auth, Stripe, IA ou `CF_API_TOKEN`.
- [ ] Conferir `OTP_LIMIT` cinco/minuto, `ABUSE_LIMIT` 120/minuto e `BULK_LIMIT` 500/dez segundos; `SYNC_LIVE`, `EMAIL` e `SITE_CACHE` também precisam existir no ambiente de produção. Não configurar `STAFF_DEV_EMAIL`.

No primeiro lançamento, preparar um arquivo JSON de secrets fora do repositório permite enviá-los junto com o código. Manter um arquivo separado para cada Worker. O Wrangler instalado suporta `--secrets-file`; JSON evita expansão de variáveis do formato dotenv. `wrangler secret put` também existe, mas cria e publica uma versão imediatamente. [Secrets e deploy](https://developers.cloudflare.com/workers/configuration/secrets/).

## 9. Publicar primeiro sites, depois conta

- [ ] **BLOQUEANTE:** conferir migrações, vars, secrets e Access antes de executar a publicação.

```sh
npm run build
npx wrangler deploy -c wrangler.sites.jsonc --env production --secrets-file '<arquivo-privado-dos-sites.json>'
npx wrangler deploy --env production --secrets-file '<arquivo-privado-da-conta.json>'
```

Se os secrets dos sites não forem necessários ou já estiverem cadastrados, omitir esse `--secrets-file`. Os comandos acima são para o operador executar no lançamento; não foram executados por esta revisão.

- [ ] Confirmar `SITE_CACHE -> blup-sites-production`, entrypoint `CachedSite`. O Worker alvo precisa existir antes da conta. Não expor esse entrypoint por rota pública.
- [ ] Conferir Custom Domains de cloud/staff, TLS, `workers_dev=false` e `preview_urls=false`. O script genérico `npm run deploy` não escolhe produção sozinho.
- [ ] Manter o gateway sem cache e somente `CachedSite` com cache habilitado. Ativar Early Hints na zona; verificar headers e 103 real depois, sem deduzir funcionamento externo do teste local. [Cache e limites](https://developers.cloudflare.com/workers/cache/).
- [ ] Guardar IDs das versões publicadas. Rollback do Worker não desfaz migrações, arquivos R2 ou eventos Stripe.

## 10. Conferir cron e fazer o primeiro teste em produção

- [ ] **BLOQUEANTE para automações:** confirmar os dois Cron Triggers no Worker da conta: `0 4 * * *` e `*/5 * * * *`, ambos UTC. Conferir execuções e falhas em observabilidade; propagação de cron pode demorar. [Cron Triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/).
- [ ] Abrir `/api/config` e `/api/plans`; conferir Turnstile e valores/moedas reais ou indisponibilidade explícita, sem preço inventado.
- [ ] Usar uma conta interna, preferencialmente com destino de e-mail verificado no Email Service: entrar por OTP, cadastrar/usar passkey, aprovar aparelho do Studio e conferir o ID da conta/copiar. Verificar recebimento, não apenas log de envio.
- [ ] Confirmar que staff exige Access e papel válido e que suas rotas não abrem pelo host público.
- [ ] Criar uma pasta de notas de teste, configurar senha das notas, sincronizar, baixar em outra instalação de teste e conferir edição/conflito. Guardar a recuperação fora de logs e capturas.
- [ ] Publicar um site de teste em `getblup.site`; conferir HTML, arquivos, busca e isolamento entre dois sites no mesmo caminho. Conferir retirada da entrega após excluir o site de teste.
- [ ] Se oferecer domínio próprio, conferir prova, SSL, baseURL após publicação e canonical/feed/sitemap. O Studio republica em segundo plano ao observar mudança de domínio, com intervalo mínimo de dez minutos por site; não há troca instantânea de URLs absolutas sem build.
- [ ] Abrir o Checkout de uma conta interna que não use a cortesia do proprietário e conferir plano, período, valor e moeda; cancelar antes de pagar neste teste de abertura. Ciclos completos de cobrança continuam homologados no sandbox.
- [ ] Conferir as execuções de lembretes e limpeza. Max permite 50 e Ultra 100 e-mails/mês; carências de 14 dias, retenção de conta por 30 dias e arquivos de até 100 MiB precisam corresponder ao produto anunciado.
- [ ] Em ambiente isolado, conferir `historyBytes:null`, purga por cofre, prepare/ticket/commit e a folga de manifesto de 4096 bytes sobre o head. Não simular expiração forçada em produção.

## 11. Preparar release da CLI e do Studio

- [ ] **BLOQUEANTE para a CLI:** confirmar a chave Ed25519 já existente fora do repositório e a pública confiada em [keys.go](../internal/update/keys.go). Só gerar par novo se ainda não existir, com `go run ./scripts/sign-release -genkey`. Guardar backup privado.
- [ ] Cadastrar o secret GitHub Actions `BLUP_RELEASE_KEY` por entrada privada. Exemplo, na raiz, sem imprimir seu conteúdo:

```sh
gh secret set BLUP_RELEASE_KEY < "$HOME/.config/blup-release/release.key"
```

- [ ] Conferir [release.yml](../.github/workflows/release.yml) e [GoReleaser](../.goreleaser.yaml). A tag de versão aciona release da CLI com `checksums.txt` e `checksums.txt.sig`; o assinador recusa chave fora da lista confiada. Publicação da tag cabe ao responsável, após revisão.
- [ ] **BLOQUEANTE para o Studio:** compilar a versão real com tag `production` e ldflags; conferir também metadados de pacote. Windows/macOS serão distribuídos **sem assinatura de desenvolvedor por enquanto**, por decisão do dono. As tarefas de assinatura existentes são opcionais para uma etapa futura.
- [ ] Publicar somente artefatos testados em cada sistema. No Windows sem assinatura, usar o formato EXE/NSIS previsto; não prometer instalação de MSIX sem certificado. No macOS, diferenciar assinatura ad hoc de montagem de Developer ID/notarização.
- [ ] Incluir nas notas de release os passos de primeira abertura: Windows, **Mais informações > Executar assim mesmo**; macOS, tentar abrir e ir a **Ajustes do Sistema > Privacidade e Segurança > Abrir mesmo assim**. O [guia de instalação](instalacao.md) contém os passos oficiais completos.
- [ ] Linux: preparar AppImage, tarball usado pelo AUR, bundle `.flatpak` no GitHub e `.deb`. Os jobs Studio do workflow geram artefatos separados da CLI; a receita AUR tem publicação própria. Flathub está fora deste lançamento.
- [ ] Conferir os checksums dos pacotes distribuídos. Assinatura Ed25519 da autoatualização CLI não é Authenticode, Developer ID ou assinatura nativa do Studio. Os instaladores iniciais CLI seguem só com SHA-256.

## 12. Publicar os canais Linux

Decisão atual: **AppImage, AUR, `.flatpak` no GitHub e `.deb`**. Não há envio ao Flathub nesta rodada, devido à [política sobre IA](https://docs.flathub.org/docs/for-app-authors/requirements#generative-ai-policy). Um bundle Flatpak do projeto é outro canal.

- [ ] **BLOQUEANTE para publicar os artefatos:** conferir os jobs `studio-linux`/`studio-publish` de [release.yml](../.github/workflows/release.yml), a arquitetura e a versão. Validar AppImage, tarball, `.deb` e `.flatpak` antes de expor seus links. A configuração do CI não comprova execução bem-sucedida em GitHub Actions.
- [ ] Conferir `studio-checksums.txt` e `studio-checksums.txt.sig` junto dos arquivos. A assinatura Ed25519 usa a mesma chave confiada da release; não equivale à assinatura nativa de cada plataforma.
- [ ] Antes do primeiro release, confirmar o app ID atual `io.github.gabriellsalesx.blupstudio`. A alternativa `io.getblup.Studio` depende de decisão e alteração coordenada dos manifests; ainda não foi adotada.
- [ ] Para o bundle, executar [bundle.sh](../studio/build/linux/flatpak/bundle.sh) a partir de `studio/`, depois do build, com versão e caminho de saída. Ele usa `org.gnome.Platform//51` e app ID `io.github.gabriellsalesx.blupstudio`. Testar instalação/abertura com Flatpak e acesso às pastas pelas permissões reais. O runtime pode vir do Flathub; isso não publica o app lá. [Bundles Flatpak](https://docs.flatpak.org/en/latest/single-file-bundles.html).
- [ ] **BLOQUEANTE para AUR:** conferir o tarball já publicado; atualizar [PKGBUILD](../studio/build/linux/aur/PKGBUILD) e [.SRCINFO](../studio/build/linux/aur/.SRCINFO) com [update.sh](../studio/build/linux/aur/update.sh), passando a versão real. Recusar checksum placeholder. Construir/testar em Arch limpo e revisar fontes, dependências e licença.
- [ ] Conferir se `blup-studio-bin` está disponível; cadastrar chave SSH da conta AUR e preparar o repositório AUR separado. Enviar somente receita/auxiliares, não o binário. Recriar `.SRCINFO` com `makepkg --printsrcinfo > .SRCINFO` quando os metadados mudarem. Commit/push cabem ao mantenedor no lançamento. Não foi encontrado veto específico a IA nas diretrizes lidas; a responsabilidade pela receita continua humana. [Regras AUR](https://wiki.archlinux.org/title/AUR_submission_guidelines).
- [ ] **BLOQUEANTE para `.deb`:** conferir [nfpm.yaml](../studio/build/linux/nfpm/nfpm.yaml), `VERSION`, `GOARCH` e contato operacional `DEB_MAINTAINER`. Definir a variável GitHub como `Blup <help@getblup.io>`, usando o contato escolhido pelo dono. Conferir `Depends`, ABI, permissões, desktop entry, ícones, MetaInfo e licença. Não é submissão ao Debian/Ubuntu nem pacote aprovado por essas distribuições.
- [ ] Inspecionar o `.deb` com `dpkg-deb --info`/`--contents` e `lintian` quando disponível. Testar instalação, abertura, atualização e remoção nas versões Debian/Ubuntu alvo. O pacote fica em `/usr/bin` e o Studio deixa atualizações para esse canal: sem repositório APT, baixar e instalar o `.deb` novo com `sudo apt install ./arquivo.deb`. Não ativar autoatualização interna no pacote. [Boas práticas e fontes](auditoria/politicas-distribuicao.md#debian-e-ubuntu-arquivo-deb-fora-do-repositório).

No handoff do Prumo, AppImage, MetaInfo/desktop entry, receita AUR e bundle Flatpak foram conferidos localmente. Houve inspeção do `.deb`, sem lintian/dpkg no ambiente usado. Lintian e instalações Ubuntu 24.04/Debian 13 estão previstos no CI, ainda não executado; resolução de bibliotecas não substitui abertura gráfica do app.

## 13. Preparar eventual publicação na Microsoft Store

A distribuição direta Windows/macOS sem assinatura continua sendo um caminho separado. As [políticas verificadas](auditoria/politicas-distribuicao.md) detalham as exigências da loja.

- [ ] Conferir o tipo de conta Partner Center exigido para a atividade comercial, reservar o nome e obter a identidade oficial do produto. Nunca usar os placeholders do template Wails como identidade da loja.
- [ ] Configurar `MSIX_IDENTITY_NAME`, `MSIX_PUBLISHER` e `MSIX_PUBLISHER_DISPLAY_NAME` nas variáveis GitHub, conforme Product identity. O job `studio-msix` prepara MSIX sem assinatura para revisão; não publica automaticamente na Store ou como instalador direto. A Store pode reassinar MSIX na certificação.
- [ ] Justificar `runFullTrust` em **Submission options**, conferir manifest/arquitetura/versão e WebView2 Runtime e testar com o Windows App Certification Kit. [Capacidades](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/app-capability-declarations), [WebView2](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution).
- [ ] **BLOQUEANTE documental para Store até o institucional estar no ar:** publicar e conferir `https://getblup.io/privacidade`, `https://getblup.io/termos` e `https://getblup.io/feedback`, todas **a publicar no site institucional** em Hugo, no repositório privado do Confete. A política do produto não é o modelo gerado para sites de usuários. Cadastrar `/privacidade` no Partner Center, conforme 10.5.1.
- [ ] Declarar tradução/narração por IA na ficha e na submissão, conforme 11.16, e indicar `/feedback` como canal de relato, inclusive de resultado inadequado de IA. Contato: `help@getblup.io`. Conferir recebimento e responsável pelo tratamento. Por decisão do dono, não haverá botão de denúncia de IA no Studio. [Políticas da Store](https://learn.microsoft.com/en-us/windows/apps/publish/store-policies).
- [ ] Descrever Studio e Cloud como produtos próprios, informar serviços pagos/opcionais e cobrança de terceiros e preparar capturas/idiomas/notas de certificação. Realizar a primeira submissão pelo Partner Center; assistência de IA não dispensa revisão ou aprovação. [Guia Microsoft com IA](https://learn.microsoft.com/en-us/windows/apps/develop/ai-assisted/quickstart).
