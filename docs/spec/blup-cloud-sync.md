# Sincronização pelo Blup Cloud

Plano da sincronização das notas de um site (a pasta que o Studio edita e publica) pelo Blup Cloud, com criptografia de ponta a ponta. Convive com o sync pelo GitHub descrito em [notas-e-sync.md](notas-e-sync.md): cada site escolhe um destino.

## Decisões de produto

| Tema | Grátis | Pro | Plano superior (a definir) |
| --- | --- | --- | --- |
| Quando sincroniza | Botão "Sincronizar" | Tempo real (ao parar de digitar ~2 s, ao fechar) | Tempo real |
| Aviso aos outros computadores | Ao sincronizar pelo botão | Instantâneo (Durable Object por usuário, WebSocket Hibernation) | Instantâneo |
| Histórico de versões | Não | 1 mês | 1 ano |
| Armazenamento | 100 MB | 5 GB | A definir |

As versões antigas contam no armazenamento. Restaurar uma versão cria uma versão nova, então nada se perde. Conflito guarda as duas versões, como no sync do GitHub.

Rareamento do histórico: todas as versões do último dia; uma por hora na última semana; uma por dia depois, até o limite do plano.

## Criptografia

Tudo é compactado e cifrado no computador. A nuvem guarda blocos ilegíveis e não vê nomes de arquivos, pastas, tamanhos individuais de notas nem o conteúdo das versões. Bibliotecas: só a biblioteca padrão do Go e `golang.org/x/crypto`.

- **Chave mestra**: 32 bytes aleatórios, uma por conta, gerada no primeiro uso.
- **Senha das notas**: Argon2id (`x/crypto/argon2`, t=3, m=64 MiB, p=4, sal de 16 bytes) deriva a chave que embrulha a chave mestra.
- **Chave de recuperação**: 32 bytes aleatórios mostrados uma vez em grupos base32. Embrulha outra cópia da chave mestra.
- A nuvem guarda só as duas cópias embrulhadas (XChaCha20-Poly1305) e os parâmetros do Argon2id. Sem a senha e sem a chave de recuperação, ninguém recupera as notas, nem o Blup.
- **Subchaves** por HKDF-SHA256 a partir da chave mestra: uma para cifrar blocos, uma para identificar blocos, uma para cifrar manifestos.
- No computador, a chave mestra fica em `credentials.toml` (modo 0600), como os tokens.

## Formato

- **Bloco**: cada arquivo é cortado em pedaços de até 4 MiB. Cada pedaço é compactado (DEFLATE da biblioteca padrão, pulado quando não reduz) e cifrado com XChaCha20-Poly1305, nonce aleatório e o id do bloco como dado associado. O id do bloco é o HMAC-SHA256 do pedaço com a subchave de identificação: pedaços iguais viram o mesmo bloco (deduplicação) sem que a nuvem consiga confirmar um conteúdo conhecido.
- **Manifesto**: um por versão do cofre, JSON compactado e cifrado: `{path: {sha, size, mode, blocks: [ids]}}`. O `sha` é o SHA de blob do Git do conteúdo em claro, o mesmo que o motor de três versões já usa.
- **Cofre**: um por site, com nome cifrado.

## Servidor (Worker)

Tabelas D1:

| Tabela | Conteúdo |
| --- | --- |
| `sync_key` | `user_id`, parâmetros do Argon2id, chave mestra embrulhada pela senha e pela recuperação |
| `vault` | `id`, `user_id`, nome cifrado, `head` (número da versão atual) |
| `vault_version` | `vault_id`, `n`, tamanho do manifesto, `created_at`; uma linha por versão guardada |
| `sync_block` | `user_id`, `id`, `size`, `created_at`; base da cobrança de armazenamento |

R2: `u/<user>/b/<bloco>`, `u/<user>/v/<cofre>/<n>` (manifesto) e `u/<user>/r/<cofre>/<n>` (lista em claro dos ids de blocos da versão, para a limpeza).

Rotas (autenticadas pela sessão do Studio, obtida pelo login com o navegador):

- `GET/PUT /api/sync/key`: buscar e criar as chaves embrulhadas; trocar a senha reescreve só o embrulho.
- `GET/POST /api/sync/vaults`: listar e criar cofres.
- `GET /api/sync/vaults/:id`: versão atual.
- `GET /api/sync/vaults/:id/versions/:n`: manifesto.
- `POST /api/sync/blocks/missing`: quais ids ainda não existem.
- `PUT/GET /api/sync/blocks/:id`: enviar e receber bloco. O envio confere a cota do plano.
- `POST /api/sync/vaults/:id/versions`: nova versão com `parent`. Só avança se `head = parent` (`UPDATE ... WHERE head = ?`); senão 409 e o Studio refaz o plano, como já faz com o GitHub.
- `GET /api/sync/live` (Pro): WebSocket com o Durable Object do usuário, que avisa "cofre X foi para a versão N".

Limpeza (Cron Trigger diário): aplica o rareamento e o limite do plano a `vault_version`, apaga manifestos e listas descartados e remove blocos sem referência criados há mais de um dia.

## Studio

1. **Motor**: `internal/notesync` passa a falar com uma interface de destino (ler a versão atual, enviar e receber conteúdo, gravar nova versão com avanço só para frente). O GitHub vira uma implementação; o Blup Cloud, outra. As regras de três versões, conflitos, lixeira e estado continuam iguais.
2. **Entrar com Blup**: login pelo navegador (device authorization do Better Auth).
3. **Senha das notas**: criação com a chave de recuperação; desbloqueio em outro computador; troca de senha.
4. **Sincronizar**: botão para todos; no Pro, envio automático com espera de ~2 s e conexão ao vivo.
5. **Versões anteriores** e **Notas apagadas** (Pro).
6. Uso real (espaço e sites) no painel.

## Ordem de entrega

1. Pacote de criptografia em Go com testes.
2. Interface de destino no `notesync`, com o GitHub migrado e os testes atuais passando.
3. Tabelas, rotas e cotas no Worker, com testes.
4. Login do Studio pelo navegador e senha das notas.
5. Destino Blup Cloud no Studio, botão Sincronizar e tempo real do Pro.
6. Histórico, notas apagadas e limpeza.
7. Uso real no painel e textos dos planos.
