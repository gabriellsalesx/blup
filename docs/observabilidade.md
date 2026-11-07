# Observabilidade com o Blup Pulse

Todas as partes do Blup mandam os próprios erros para o **Blup Pulse**, pelo SDK oficial do Sentry. O Pulse guarda só erros: rastreio de desempenho, replay e logs ficam desligados nos SDKs. Os monitores de uptime do Pulse checam as rotas de saúde descritas abaixo.

Sem DSN configurada, nenhuma parte envia nada. É o caso desta máquina, a menos que você passe uma para testar.

## Projetos no Pulse

Crie um projeto por parte na sua conta do Pulse e guarde a DSN de cada um:

| Projeto | Plataforma | Onde a DSN entra |
|---|---|---|
| Blup Cloud (API, staff, Pulse, Deals, cron e Durable Objects) | `javascript` | `wrangler secret put SENTRY_DSN --env production` |
| Sites publicados (`*.getblup.site`) | `javascript` | `wrangler secret put SENTRY_DSN -c wrangler.sites.jsonc --env production` |
| Painéis no navegador (Cloud, staff, Pulse, Deals) | `javascript` | `VITE_SENTRY_DSN=... npm run build` (e no build de cada painel) |
| Ponte do WhatsApp | `go` | variável `SENTRY_DSN` do contêiner |
| Blup Studio (desktop) | `go` | `SENTRY_DSN=... task build` (vai para `-X main.studioSentryDSN`) |
| Blup Studio (celular) | `go` | `SENTRY_DSN=... task android:build` |

Os erros saem marcados com o ambiente:
- `production` no ar;
- `development` nesta máquina.

A versão também vai junto: o id da versão publicada no Cloudflare, `blup-bridge@<versão>` e `blup-studio@<versão>`.

Os erros do próprio Cloud entram direto na ingestão do Pulse, no mesmo pedido (`worker/observability.ts`). Eles não saem pela rede, então não há ciclo.

## Monitores de uptime

As rotas ficam fora de `/api`, que os monitores do Pulse não podem chamar. Elas nunca vão para cache e não mostram detalhes: respondem só `ok` ou `unavailable`.

| Monitor | Endereço | O que confere |
|---|---|---|
| Cloud | `https://cloud.getblup.io/_blup/ready` | o Worker responde e o D1 responde em até 3 s |
| Páginas de status | `https://pulse.getblup.site/_blup/ready` | idem |
| Páginas de ofertas | `https://deals.getblup.site/_blup/ready` | idem |
| Sites publicados | `https://<um-site>.getblup.site/_blup/ready` | o Worker dos sites e o D1 |
| Ponte do WhatsApp | `https://<endereço-da-ponte>/_blup/ready` | o processo e o banco das sessões |
| Painel do Pulse | `https://pulse.getblup.io/` | a página carrega |
| Painel do Deals | `https://deals.getblup.io/` | a página carrega |

Detalhes:
- `/_blup/health` só diz se o processo está vivo e não toca no banco. Use quando quiser separar "fora do ar" de "banco lento".
- O `ingest.getblup.io` não pode ser monitorado direto, porque o próprio Pulse bloqueia. Ele vive no mesmo Worker do Cloud, então o monitor do Cloud cobre.
- O staff fica atrás do Cloudflare Access, por isso não tem monitor.

## O que nunca é enviado

- **Usuário, IP, cookies, cabeçalhos, corpo e query** dos pedidos. Os SDKs usam `dataCollection` todo desligado, e o `beforeSend` deixa só o método e o caminho.
- **Linhas do console**, nos navegadores.
- **No Studio:**
  - o texto das notas;
  - a pasta pessoal, que vira `~` nos caminhos;
  - o nome do computador.

  Os relatórios só ligam depois do **aceite**. A pergunta aparece na primeira abertura, num modal no desktop e numa gaveta no celular, e a resposta pode mudar nas configurações. "Agora não" pergunta de novo na próxima vez.
- O próprio Pulse ainda tira o que sobrar ao receber (ver `docs/spec/blup-pulse.md`).

## Testar nesta máquina

- **Cloud:** suba o `wrangler dev` com `--var PULSE_INGEST_HOST:ingest.localhost --var SENTRY_DSN:http://<chave>@ingest.localhost:<porta>/<projeto>`, usando um projeto criado no Pulse local.
- **Painéis:** rode com `VITE_SENTRY_DSN=http://<chave>@ingest.localhost:<porta>/<projeto>`. O CSP libera essa origem sozinho (`shared/sentry-csp.ts`).
- **Ponte:** `go test ./...` manda um erro para um servidor de teste e confere que nada pessoal vai junto.
- **Studio:** `BLUP_SENTRY_DSN=...` liga o envio sem precisar recompilar (os relatórios ainda dependem do aceite).
