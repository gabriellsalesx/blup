# Modelo de segurança

Escopo: código local, tema e Blup Cloud observados em 02/11/2026. Este guia descreve proteções e limites implementados; não é certificação nem afirma que a configuração remota já foi auditada. [Arquitetura](arquitetura.md), [produção](deploy.md) e [spec](spec/README.md) têm os contratos complementares.

## Ativos e fronteiras

| Ativo | Fronteira e armazenamento |
| --- | --- |
| Texto e anexos das notas | Legíveis no disco da pessoa; cifrados no transporte lógico do sync Blup Cloud |
| Chave mestra de sync | Aberta no cliente; cópia local base64 em `credentials.toml`; embrulhos no servidor |
| Senha das notas e recuperação | Abrem a chave no computador; não são o login Better Auth |
| Sessão Blup, chaves GitHub/Cloudflare/OpenRouter | Arquivo de credenciais local `0600`; secrets do Worker são outra configuração |
| Site publicado e anexos referenciados | Conteúdo público no provedor; não recebem a cifra de ponta a ponta do sync |
| Senha de post | Protege o corpo pós-processado; não restringe todo o diretório publicado |
| Conta e cobrança | D1, Better Auth e Stripe; números de cartão ficam no fluxo hospedado pelo Stripe |
| Agenda | Notas locais; por e-mail saem apenas IDs opacos e horários |

As três senhas/credenciais não devem ser confundidas: login da conta por OTP/passkey, senha que embrulha a chave das notas e senha de leitura de post. Trocar uma não equivale a trocar as outras.

## Ameaças consideradas

| Ator ou falha | Defesa existente | Limite |
| --- | --- | --- |
| Leitura indevida do armazenamento cloud | Blocos, nomes de cofres e manifestos cifrados no computador | Metadados e disponibilidade continuam visíveis/controláveis pelo serviço |
| Senha de notas adivinhada | Argon2id com salt e custo de memória/tempo | Senha fraca continua vulnerável a tentativa offline dos embrulhos |
| Conta cloud comprometida | Chave das notas exige senha/recuperação ou cópia local aberta | A sessão ainda autoriza outras operações e não garante disponibilidade dos dados |
| Computador comprometido | Permissões de arquivo e limites de caminho | Arquivos locais e chave aberta não ficam protegidos contra o usuário local ou malware com seu acesso |
| Outro usuário tentando acessar um cofre/site | Sessão e consultas por `userId`, validação de IDs e caminhos | A segurança depende das APIs em execução e de sua configuração, não da aparência opaca dos IDs |
| Site de cliente contendo scripts | Sites hospedados em `getblup.site`, separados do domínio da conta `getblup.io`, com origem por hostname | Cookies de domínio pai e conceitos de site dependem também da PSL; não há sandbox de HTML arbitrário no Worker público |
| Requisição cross-site com cookie ambiente | Better Auth valida origem; o Worker filtra mutações por Origin/Fetch Metadata | Webhook Stripe tem outra autenticação, pela assinatura; Bearer do Studio é outro fluxo |
| HTML malicioso em notas ou comentários | Goldmark, escape de templates e sanitização explícita do feedback | Overrides de tema e JS fornecidos por quem publica são código confiado por essa pessoa |
| Falha entre salvar e sincronizar | Comparação entre base/disco/remoto e cópias de conflito | Sync não substitui backup, nem garante preservar toda edição concorrente em qualquer formato |

As rotas estão em [index.ts](../cloud/worker/index.ts), [sync.ts](../cloud/worker/sync.ts), [sites.ts](../cloud/worker/sites.ts) e [staff.ts](../cloud/worker/staff.ts). A política de mutações está em [security.ts](../cloud/worker/security.ts). O filtro considera origem da própria requisição, permite o caminho de webhook e preserva o uso de Bearer pelo desktop; não substitui validar sessão ou assinatura.

Respostas da API recebem `Cache-Control: no-store` quando não definem outra política. O retorno de login `volta` é aceito apenas como caminho da própria origem, sem controles ou barras invertidas, em `cloud/src/lib/auth.ts`. Logs de falhas de IA não incluem o corpo devolvido pelo provedor; falhas de e-mail conservam apenas códigos de formato permitido. `EMAIL_MODE=log` continua sendo um modo de desenvolvimento que registra destinatário e OTP, e não deve operar em produção.

## O que é cifrado no sync

[internal/cloudcrypt](../internal/cloudcrypt/cloudcrypt.go) gera 32 bytes aleatórios de chave mestra por conta e 32 bytes de recuperação. A senha passa por Argon2id com parâmetros padrão de três iterações, 64 MiB, quatro threads e salt de 16 bytes. Os parâmetros acompanham os embrulhos e têm limites locais antes de executar a derivação. HKDF-SHA256 separa chaves de conteúdo, IDs e manifesto.

Cada bloco de até 4 MiB é compactado quando vale a pena e protegido por XChaCha20-Poly1305. Seu ID HMAC-SHA256 depende de uma chave secreta, evitando que o servidor teste conteúdo conhecido usando um hash público simples. Manifestos incluem caminhos, tamanhos, SHA do arquivo e blocos; são cifrados e vinculados ao cofre e número da versão por dados associados. O nome do cofre é cifrado separadamente. Depois de baixar, o cliente autentica os blocos e confere o arquivo remontado.

O servidor vê conta, ID do cofre, versões e horários, quantidades e tamanhos dos objetos cifrados, IDs HMAC e referências de blocos por versão. Deduplicação revela reutilização de blocos dentro da conta. A cifra não oculta IP, padrão de acesso ou frequência de alterações, nem promete impedir exclusão, indisponibilidade ou toda forma de rollback por um serviço malicioso.

Trocar a senha reembrulha a mesma chave mestra, mantendo o embrulho de recuperação. Isso não revoga uma chave já copiada de uma máquina nem recifra todos os objetos com outra chave mestra. O código guarda um embrulho anterior no D1. A recuperação depende de uma cópia da senha, da recuperação ou da chave local; não existe um botão da staff que decifre as notas.

`credentials.toml` não usa keyring do sistema: `notes_key` é base64, não uma cifra do arquivo local. Notas, anexos e backups ZIP locais continuam legíveis no disco. O sync GitHub usa repositório privado, sem a camada E2E do Blup Cloud. Uma senha literal escrita numa nota viaja como conteúdo dessa nota em qualquer sync; no destino GitHub o provedor consegue lê-la.

## Dados visíveis fora do sync

| Operação | Dados que o serviço pode ler |
| --- | --- |
| Login e conta | E-mail quando vinculado, passkeys públicas, sessões, User-Agent e dados de uso |
| Sync cifrado | Metadados e bytes cifrados descritos acima |
| Publicação | HTML, CSS, JS, imagens, MP3 e manifesto públicos; nome/slug/título do site |
| Narração cloud | Texto falável e voz/modelo, lidos pelo Worker e OpenRouter |
| Tradução cloud | Texto das partes e instruções, lidos pelo Worker e OpenRouter |
| IA com chave própria | Texto vai diretamente ao OpenRouter a partir do cliente |
| Kokoro e tradução local | Texto processado no computador; instalação inicial baixa ferramentas/modelos |
| Lembretes por e-mail | Conta, ID opaco, instante, contador e destinatário, sem título ou corpo do evento |
| Comentários e Feedback | Conteúdo público e identidade GitHub nas issues; não são notas privadas cifradas |
| Podcast | Feed, endereço de contato configurado, narração e capas públicos |

IA remota não é processamento cifrado de ponta a ponta: o texto precisa ser fornecido ao serviço para ser narrado ou traduzido. O fato de a mesma nota estar cifrada no bucket de sync não muda isso. A seleção da voz e o custo estão em [narração](spec/narracao-e-voz.md); o campo de contato público do feed está em [podcast](spec/podcast.md).

## Publicação, senha e navegador

[internal/lock](../internal/lock/lock.go) protege regiões de corpo dos posts depois do Hugo, com AES-256-GCM e PBKDF2-SHA256 de 600.000 iterações. `lock.js` abre o corpo no navegador e guarda senhas bem-sucedidas somente na sessão da aba. Isso é diferente de cifrar notas no sync: título, autores, capa, URL e outros metadados permanecem públicos, e anexos são arquivos separados.

`hidden` retira de listagens que aplicam o filtro, mas mantém URL acessível. `noindex` é instrução a indexadores, não autorização. Rascunhos entram na prévia e não no build normal. A prévia mostra texto protegido e deve permanecer local, não exposta como site público. Narrar uma nota com senha é recusado; aplicar senha depois não apaga automaticamente um MP3 já existente. O feed de podcast exclui conteúdo protegido, escondido ou rascunho.

O tema usa escape e Goldmark sem HTML arbitrário habilitado. O feedback recebido passa por `markdown` e `safeURL` em [feedback.js](../theme/assets/js/feedback.js), que copiam elementos permitidos e filtram endereços. Por escolha de produto, a autorização GitHub do visitante fica em `localStorage` para persistir entre visitas; sair remove a chave. Estado temporário do retorno OAuth e caches usam `sessionStorage`. Essa persistência exige evitar scripts ativos vindos de comentários e não cifra conversas públicas.

No Studio, [guard.go](../studio/guard.go) exige POST e `x-wails-client-id` para `/wails/runtime` e recusa metadados de origem cruzada nas rotas Wails. A prévia fica em iframe de outra origem local com sandbox; o receptor confere origem e identifica o frame. `OpenURL` permite somente HTTP, HTTPS, e-mail e telefone. As APIs de arquivos usam `app.Within` para verificar caminhos após links simbólicos. Isso reduz caminhos de ataque de notas e HTML da prévia; não é proteção contra código local com acesso à conta do sistema.

O painel da conta tem CSP gerado pelo SvelteKit e headers em `static/_headers`; o Worker aplica headers também às próprias respostas. Os sites de usuários têm outro Worker e outros headers. A memória/cache do servidor, o service worker do site e o cache do navegador têm contratos distintos; excluir ou substituir uma publicação não é promessa de apagar cópias já baixadas por visitantes.

## Staff, autenticação e cobrança

Better Auth valida a origem configurada e usa OTP armazenado por hash, passkeys e sessões. O cadastro com passkey sem sessão também passa por Turnstile e limite por IP. O device flow aceita somente o identificador do Studio e requer aprovação no navegador. A configuração concreta é [auth-options.ts](../cloud/worker/auth-options.ts); a referência de proteção de origem é [Better Auth security](https://better-auth.com/docs/reference/security).

Cloudflare Access protege o host da equipe. A API valida novamente o JWT, issuer e audience com JWKS, e só então consulta `support`/`admin` ou o proprietário de bootstrap. A staff pode ver metadados de conta, planos, uso e eventos financeiros, revogar acesso e executar operações autorizadas, mas não possui a chave de notas nem uma função de impersonação. As ações são registradas em `staff_log`. A configuração de teste por `STAFF_DEV_EMAIL` só se aplica a hosts locais e não deve existir em produção.

O Stripe recebe o pagamento em páginas próprias. O plugin Better Auth autentica o webhook e atualiza assinaturas; `billing.ts` deduplica registros por ID do evento. Cancelar ou mudar plano altera os direitos que as APIs consultam; a existência de uma página de checkout aberta não concede plano pago. Segredos ficam no ambiente de execução e não em `src/` ou assets. Veja [configuração operacional](deploy.md).

## Cotas e retenção

A [tabela dos planos](spec/blup-cloud-sync.md#planos-executados) é derivada de `plans.ts`: Grátis 50 MiB, Pro 1 GiB, Max 10 GiB e Ultra 50 GiB; notas e sites compartilham esse total. Arquivo máximo: 10, 25, 200 e 500 MiB, respectivamente. No sync, o servidor recebe blocos cifrados e não tem o tamanho em claro por arquivo para verificar sozinho essa restrição lógica do cliente. O limite de blocos e manifestos é independente.

A API de sites verifica hash e tamanho armazenado, quantidade de sites, cota e manifesto de até 20.000 caminhos. No sync, IDs são escopados por conta, a criação de versão usa pai condicional e o histórico consome cota até expirar/ser limpo. Não trate limites de Workers, D1 ou Stripe como se fossem os limites comerciais da tabela.

A cota de IA é mensal, com janela móvel de sete dias limitada à metade. Modelos mais caros reduzem a cota mais rapidamente. A voz usa preço validado do catálogo e contagem de caracteres; a tradução pode registrar o custo reportado pelo provedor. Isso não garante atomicidade entre chamada de IA e gravação de uso ou entrega do resultado ao computador. Erro/repetição pode ocorrer depois de trabalho remoto já iniciado.

A auditoria mantém regressões como `it.fails` em [security.test.ts](../cloud/worker/security.test.ts): criações simultâneas podem ultrapassar a quantidade de sites, uploads podem ultrapassar armazenamento e chamadas de IA podem ultrapassar as janelas semanal/mensal. São limites consultados antes da operação sem reserva atômica compartilhada. Uma tradução incompleta também pode ter custo do provedor sem registro de uso. Há reprodução de limpeza de deploy concorrente removendo arquivos necessários à publicação ativa. Uma suíte verde com esses casos como falha esperada não comprova os limites ou a integridade da entrega sob concorrência; a decisão de correção cabe ao responsável pelo código e produto.

Lembretes têm limite de 300 itens por lista, cron de até 200 vencidos por rodada, tolerância de 12 horas e cota mensal de 50 no Max/100 no Ultra. IDs usam HMAC com chave local `reminder-key`, sem título ou caminho em claro. A fila é por conta e as instalações substituem a mesma lista; não existe identidade global de evento entre máquinas. Reivindicar um item evita duas rodadas processarem o mesmo pendente, mas o envio marcado antes do provedor não é garantia de entrega nem de repetição após falha. [Calendário](spec/calendario-e-lembretes.md) detalha o contrato.

Limpeza do sync rareia versões e remove blocos sem referência antigos. Limpeza de lembretes retira itens com mais de 36 horas e contadores antigos. A exclusão individual de site tem remoção do prefixo R2 próprio. O fluxo de exclusão de usuário cancela assinatura e remove a conta; não se deve presumir purga completa de todos os objetos e metadados de produto sem uma rotina explícita que percorra esses recursos. A política de retenção e a verificação de exclusão precisam ser decididas para o lançamento.

[serve-site-security.test.ts](../cloud/worker/serve-site-security.test.ts) também mantém falhas esperadas de revogação: o Worker público não verifica plano nem existência do usuário em cada entrega. Reduzir para Grátis ou apagar apenas o usuário pode deixar o site no ar enquanto seu registro e objetos permanecerem. O fluxo Apagar do Studio só relaciona destinos publicados GitHub/Cloudflare; retirar a pasta local não retira automaticamente o site Blup Cloud. O painel da conta possui exclusão própria pela API de sites.

## Limites e decisões de produção

1. Provisionar e verificar Access, Turnstile, remetente de e-mail, secrets e webhook no ambiente correto. Placeholders no repositório não provam operação segura.
2. Tratar a inclusão e propagação de `getblup.site` na PSL como processo operacional ainda a comprovar, seguindo a [PSL](https://publicsuffix.org/submit/). Não anunciar certificado de `www.nome.getblup.site` sem cobertura TLS adicional.
3. Definir política de retenção, purga de conta, backups e resposta a falhas de envio. A documentação não inventa garantias de apagar cópias externas ou recuperar cifra sem chave.
4. Manter separada a promessa comercial de domínio próprio no Blup Cloud: há flag no plano, mas a integração atual serve somente `getblup.site`.
5. Assinar e identificar corretamente artefatos desktop. Um checksum não equivale sozinho à verificação de identidade do editor em cada plataforma.
6. Resolver ou aceitar explicitamente as falhas conhecidas de admissão concorrente, contabilização de IA, limpeza de publicações e revogação. Manter sua reprodução marcada como falha esperada não corrige o comportamento.

Há também uma diferença de portabilidade: `sharedSite`, em `internal/app/cloud_sites.go`, não inclui `translations`. As notas traduzidas podem ser recuperadas pelo sync, mas a lista de idiomas publicada não é restaurada por esse TOML numa instalação nova. O contrato está em [notas e sync](spec/notas-e-sync.md).

Os testes automatizados exercitam criptografia, caminhos, entradas de API, HTML e quotas conforme seus cenários. Eles não substituem a revisão do ambiente real nem autorizam testes de ataque contra produção. As verificações desta revisão ficam limitadas a código local e simulações; os comandos estão em [testes e build](spec/testes-e-build.md).
