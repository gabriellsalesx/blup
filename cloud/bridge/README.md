# Ponte do Blup Deals (WhatsApp)

A Ponte mantém conectados os WhatsApp dos clientes e publica nos **Canais** (newsletter) deles quando o Blup Cloud pede. Ela só publica em Canal: grupos e conversas não existem para ela.

O caminho é **não oficial**. A API da Meta não publica em Canal, e a Ponte fala o protocolo do WhatsApp Web pela biblioteca [whatsmeow](https://github.com/tulir/whatsmeow). O número pode ser suspenso pelo WhatsApp. O cliente aceita esse risco ao conectar, e a landing page avisa.

## O que fica onde

| Onde | O quê |
|---|---|
| D1, com o Blup Cloud | tudo do negócio: canais, fila, posts, ritmo, cliques, contas |
| Ponte (`DATA_DIR`) | só as chaves das sessões do WhatsApp, a hora do último post de cada Canal e os posts já feitos (para um pedido repetido não publicar duas vezes) |

A Ponte não acessa o D1. Só o Blup Cloud fala com ela, e só ela fala com o WhatsApp. As imagens chegam prontas do Blup Cloud, que as baixa com a proteção contra endereços maliciosos; a Ponte não busca nada na internet além do WhatsApp. Nada fica salvo: a imagem vai ao WhatsApp e some da memória.

Se a pasta de dados se perder, ninguém perde o Canal: cada cliente só precisa ler o QR de novo.

## Modo segurança

No máximo **um post a cada 30 minutos por Canal**. O Blup Cloud já espaça a fila assim, e a Ponte confere de novo: mesmo que o Blup Cloud peça antes, ela responde `429 TOO_SOON` com quanto falta. `MIN_GAP` só aumenta esse intervalo, nunca diminui.

## Configuração

Tudo por variáveis de ambiente:

| Variável | Padrão | Para quê |
|---|---|---|
| `BRIDGE_SECRET` | (obrigatória) | assina os pedidos do Blup Cloud; 32 caracteres ou mais |
| `CALLBACK_URL` | (vazia) | endereço `https` do Blup Cloud para os avisos de sessão |
| `CALLBACK_SECRET` | (vazia) | assina os avisos; obrigatória com `CALLBACK_URL` |
| `HOST` | `127.0.0.1` | onde escuta; `0.0.0.0` quando o servidor exige |
| `PORT` | `8080` | a porta |
| `DATA_DIR` | `data` | a pasta do SQLite |
| `DATABASE_URL` | (vazia) | um Postgres no lugar do SQLite |
| `MIN_GAP` | `30m` | o intervalo mínimo por Canal; aceita `30m` ou mais |
| `DEVICE_NAME` | `Blup Deals` | o nome em "Aparelhos conectados" |

Gere as senhas com `openssl rand -hex 32`. Elas ficam só nas variáveis do servidor e nos secrets do Blup Cloud, nunca no repositório.

**Uma cópia só.** Cada sessão do WhatsApp só pode estar ligada em um lugar por vez, então a Ponte roda em uma cópia só. Por isso o SQLite basta; o Postgres é para quando o servidor já tiver um.

## Subir

Em qualquer servidor com Go:

```sh
cd cloud/bridge
CGO_ENABLED=0 go build -trimpath -o bridge .
BRIDGE_SECRET=... CALLBACK_URL=... CALLBACK_SECRET=... ./bridge
```

Ou com Docker (a pasta de dados vai num volume):

```sh
docker build -t blup-bridge cloud/bridge
docker run -d --restart unless-stopped -v blup-bridge-data:/data \
  -e BRIDGE_SECRET=... -e CALLBACK_URL=... -e CALLBACK_SECRET=... blup-bridge
```

**Numa VPS, a Ponte não fica aberta na internet.** Ela escuta em `127.0.0.1` e o acesso chega por um [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/) apontando para `http://127.0.0.1:8080`. Uma plataforma que já entrega HTTPS também serve; aí use `HOST=0.0.0.0`. Em qualquer caso, sem a assinatura nenhum pedido passa.

Os registros saem em JSON na saída padrão, sem texto de post, QR ou senha. `GET /health` responde `{"ok":true}` sem assinatura, para o servidor saber se a Ponte está viva. Ao receber `SIGTERM`, ela fecha as conexões e as sessões antes de sair.

## Contrato

Todos os pedidos, menos `/health`, levam três cabeçalhos:

| Cabeçalho | Valor |
|---|---|
| `X-Bridge-Timestamp` | segundos Unix; vale cinco minutos para mais ou para menos |
| `X-Bridge-Nonce` | 32 caracteres hexadecimais aleatórios, um por pedido |
| `X-Bridge-Signature` | HMAC-SHA256 em hexadecimal, com `BRIDGE_SECRET`, de `momento \n nonce \n MÉTODO \n caminho \n sha256(corpo) em hexadecimal` |

Um nonce já usado é recusado. Os avisos para `CALLBACK_URL` vão assinados do mesmo jeito, com `CALLBACK_SECRET`.

A sessão (`{id}`) é o ID que o Blup Cloud escolhe, de 8 a 64 caracteres `A-Z a-z 0-9 _ -`.

| Pedido | Resposta |
|---|---|
| `POST /v1/sessions/{id}` | abre o QR (ou devolve a sessão como está): `{state, qr, qrExpiresAt, phoneEnd}`; `qr` é uma imagem PNG em `data:` e só existe enquanto espera ser lido |
| `GET /v1/sessions/{id}` | o mesmo, para acompanhar a leitura do QR |
| `GET /v1/sessions/{id}/channels` | `{channels: [{id, name, role}]}`, só os Canais em que a sessão é `owner` ou `admin` |
| `POST /v1/sessions/{id}/posts` | publica `{channel, key, text, image?}`; `image` é JPEG ou PNG em base64, até 5 MB; `text` até 4096 caracteres; responde `{id, at}`. A mesma `key` não publica duas vezes |
| `DELETE /v1/sessions/{id}` | desconecta o aparelho e apaga a sessão (`204`) |

Estados: `pairing`, `connected`, `disconnected` (caiu e volta sozinha), `expired` (o QR venceu), `logged_out` (desconectado pelo celular), `banned`, `replaced` (a sessão abriu em outro lugar) e `outdated` (o whatsmeow precisa de atualização). O aviso em `CALLBACK_URL` é `{session, state, at}`, enviado quando a sessão conecta pela primeira vez ou entra num estado que exige ação.

Erros: `400 BAD_REQUEST`, `401 UNAUTHORIZED`, `403 NOT_ADMIN` (o Canal não é dela), `404 NOT_FOUND`, `409 NOT_CONNECTED`, `413 TOO_LARGE`, `429 TOO_SOON` (com `retryAfter` em segundos e `Retry-After`), `502 UPSTREAM` e `503 BUSY` (QRs demais esperando ao mesmo tempo).

## Testes

```sh
cd cloud/bridge && go test -race ./...
```
