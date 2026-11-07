# Modelo de segurança

Escopo: código local, tema e Blup Cloud observados em 02/11/2026. Este guia descreve proteções e limites implementados; não é certificação nem afirma que a configuração remota já foi auditada. [Arquitetura](arquitetura.md), [produção](deploy.md) e [spec](spec/README.md) têm os contratos complementares.

## Ativos e fronteiras

| Ativo | Fronteira e armazenamento |
| --- | --- |
| Texto e anexos das notas | Legíveis no disco da pessoa; cifrados no transporte lógico do sync Blup Cloud |
| Chave mestra de sync | Aberta no cliente; cópia local base64 em `credentials.toml`; embrulhos no servidor |
| Senha das notas e recuperação | Abrem a chave no computador; não são o login Better Auth |
| Sessão Blup, chaves GitHub/Cloudflare/OpenRouter | Arquivo de credenciais local `0600`; secrets do Worker são outra configuração |
| Site publicado e anexos referenciados | Saída pública; anexos exclusivos do corpo protegido podem ser incorporados na cifra do post |
| Senha de post | Protege o corpo pós-processado; não restringe todo o diretório publicado |
| Conta e cobrança | D1, Better Auth e Stripe; números de cartão ficam no fluxo hospedado pelo Stripe |
| Agenda | Notas locais; por e-mail saem apenas IDs opacos e horários |
| Pulse, contratos locais das Fases 1 a 4 | Eventos e checagens no DO; configuração/índice no D1; destinos de avisos protegidos com chave de serviço; DTO público separado |

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

Não existe fluxo de troca da senha das notas na CLI ou no Studio. A API de embrulhos existentes exige preservar `by_recovery`; a migração `0012` remove `previous_key`, que não tinha leitor. Na criação inicial, uma disputa perdida retorna `409 KEY_EXISTS`, sem fingir persistência da chave candidata. A recuperação depende da senha, da recuperação ou da chave local; não existe um botão da staff que decifre as notas. Alterar um embrulho não revogaria uma chave já copiada nem recifraria os objetos.

`credentials.toml` não usa keyring do sistema: `notes_key` é base64, não uma cifra do arquivo local. Notas, anexos e backups ZIP locais continuam legíveis no disco. O sync GitHub usa repositório privado, sem a camada E2E do Blup Cloud. Uma senha literal escrita numa nota viaja como conteúdo dessa nota em qualquer sync; no destino GitHub o provedor consegue lê-la.

## Dados visíveis fora do sync

| Operação | Dados que o serviço pode ler |
| --- | --- |
| Login e conta | E-mail quando vinculado, idioma salvo em `user.language`, passkeys públicas, sessões, User-Agent e dados de uso |
| Sync cifrado | Metadados e bytes cifrados descritos acima |
| Publicação | HTML, CSS, JS, imagens, MP3 e manifesto públicos; nome/slug/título do site |
| Narração cloud | Texto falável e voz/modelo, lidos pelo Worker e OpenRouter |
| Tradução cloud | Texto das partes e instruções, lidos pelo Worker e OpenRouter |
| IA com chave própria | Texto vai diretamente ao OpenRouter a partir do cliente |
| Kokoro e tradução local | Texto processado no computador; instalação inicial baixa ferramentas/modelos |
| Lembretes por e-mail | Conta, ID opaco, instante, contador e destinatário, sem título ou corpo do evento |
| Comentários e Feedback | Conteúdo público e identidade GitHub nas issues; não são notas privadas cifradas |
| Podcast | Feed, endereço de contato configurado, narração e capas públicos |
| Pulse, contratos locais das Fases 1 a 4 | Mensagens/frames tratados, monitores e resultados HTTP; textos públicos escolhidos pela pessoa; resumo de aviso enviado ao canal configurado |

O Go exige aceite local separado para cada destino de IA, `openrouter` e `blup`, antes do primeiro envio de texto. A escolha pode ser revogada nas Configurações e fica em `ai-consent.json`, não sincronizado. Arquivo inválido não concede aceite. Notas com senha são recusadas pela tradução remota e pela narração; a tradução local pode processá-las. Escolher voz local não provoca fallback automático para a nuvem. IA remota não é processamento cifrado de ponta a ponta: o texto precisa ser fornecido ao serviço para ser narrado ou traduzido. O fato de a mesma nota estar cifrada no bucket de sync não muda isso. A seleção da voz e o custo estão em [narração](spec/narracao-e-voz.md); o campo de contato público do feed está em [podcast](spec/podcast.md).

## Pulse: limites da entrega local

As Fases 1 a 3 do Pulse têm gates locais fechados; os contratos da Fase 4 foram conferidos, com homologação de infraestrutura pendente. Não há confirmação de publicação ou contratação liberada. Eventos, monitores e incidentes não usam a cifra de ponta a ponta das notas: o serviço processa os dados aceitos. O parser projeta e trata os eventos antes de persistir, descartando contextos como pedido HTTP, usuário, breadcrumbs, tags e anexos. Texto livre ainda pode conter dados pessoais. O aceite e o filtro específico do Studio continuam pendentes antes de enviar seus erros automaticamente.

Monitores validam URL, faixas de IP e DNS, revalidam até três redirecionamentos, compartilham um timeout de dez segundos e cancelam o corpo sem armazená-lo. O caminho usa `fetch()` público com `global_fetch_strictly_public`. A restrição de rede efetiva é uma premissa da plataforma, a ensaiar em homologação com destinos controlados; DNS consultado antes do pedido não fixa a resolução usada depois. A documentação da flag não é prova universal contra DNS rebinding. Contas comuns não podem usar os hosts Blup/Workers; a exceção operacional do dono admite páginas públicas Blup e mantém bloqueadas ingestão e rotas API do Blup. Ver [premissa, fontes oficiais e validações](spec/blup-pulse.md#7-monitores-http-e-páginas-de-status).

A página pública usa uma projeção fechada, sem URL monitorada, DSN, dono ou erro privado. O gateway revalida conta, plano e publicação antes do cache interno de até 15 segundos. HTML escapa textos e não executa JavaScript; HTML e `status.json` usam ETag/304. Revogação/despublicação não depende de expirar esse cache. Textos de componentes e incidentes são públicos por escolha explícita; o cache não transforma texto privado em conteúdo seguro para publicar.

As consultas respeitam retenção de 30/90 dias e detalhe de checagem de sete dias. Ao perder o direito Pulse, a leitura privada dura 14 dias, sem renovar a retenção; depois fica elegível à purga em lotes. Exclusão da conta Cloud bloqueia acesso imediatamente e integra a purga Pulse ao prazo de 30 dias. Alarmes e cron retomam limpeza; não há promessa de remoção física no segundo exato. A Fase 4 acrescenta cobrança própria e domínio Ultra + Pulse, com homologação de infraestrutura pendente. A leitura após retirar Pulse se ancora no fim pago salvo e não se estende por visita tardia; recibos técnicos de cota sem corpo de erro e totais mensais permanecem na janela de 95 dias arredondada ao mês UTC para não zerar uso ao recontratar, e entram na purga definitiva da conta; redução com excesso exige selecionar recursos mantidos e dá 14 dias privados aos demais. Provedores externos continuam sujeitos à homologação. Os gates finais locais foram informados pela Lupa; o ensaio remoto de infraestrutura continua pendente. Os [contratos locais](spec/blup-pulse.md#estado-da-entrega-por-fases) distinguem essa implementação da oferta-alvo.

Destinos Discord/Telegram usam AES-GCM com AAD ligado a dono/canal/tipo e chave de serviço `PULSE_CHANNEL_KEY`; o servidor consegue abri-los para enviar. Não são dados de sync cifrado de ponta a ponta. Rótulos, vínculos e o chat candidato temporário ficam legíveis no D1. A API não devolve a URL do webhook. O vínculo Telegram exige código temporário, secret do webhook, remetente real em chat privado, deduplicação e confirmação com a sessão do dono; o DSN não autoriza configuração. A fila guarda referências e revalida acesso antes de enviar, com até cinco tentativas em 24 horas, sem garantir entrega exatamente uma vez. Ver [avisos e proteção dos destinos](spec/blup-pulse.md#8-avisos-discord-e-telegram).

No domínio de página Pulse, TXT precede a chamada Cloudflare e a reivindicação atômica compartilha o namespace dos sites Hugo. O gateway exige prova/SSL/estado ativo, publicação, página retida e direito Ultra + Pulse. Perda do direito bloqueia o host sem carência extra; a remoção externa preserva o ID até concluir. Cobrança por ticket/confirmar não escreve o plano da assinatura: só o webhook assinado com consulta atual reconcilia os direitos no D1. Portal e rotas mutáveis concorrentes do plugin ficam restritos para não contornar a prévia; ver [contratos de cobrança](spec/blup-pulse.md#11-switch-e-cobrança-stripe).

IDs de deduplicação e estados antigos de problemas deixam de valer no corte de retenção mesmo antes da purga física. O alarme preserva um prazo anterior e repete em um minuto enquanto houver lotes vencidos de eventos ou checagens; o cron recupera limpeza interrompida. A janela técnica de cota acima não mantém corpos de erros. Integridade/FKs do SQLite local, testes e dry-runs foram informados como aprovados pela Lupa; esses resultados não confirmam bindings remotos ou staging.

## Publicação, senha e navegador

[internal/lock](../internal/lock/lock.go) protege regiões de corpo dos posts depois do Hugo, com AES-256-GCM e PBKDF2-SHA256 de 600.000 iterações. `lock.js` abre o corpo no navegador e guarda senhas bem-sucedidas somente na sessão da aba. Isso é diferente de cifrar notas no sync: título, autores, capa, URL e outros metadados permanecem públicos, e a proteção de anexos tem o escopo descrito abaixo.

`hidden` retira de listagens que aplicam o filtro, mas mantém URL acessível. `noindex` é instrução a indexadores, não autorização. `draft: true`, `publish: false` e `private: true` entram na prévia e ficam fora do build normal. Os mounts também excluem traduções relacionadas a notas não publicadas; links para esses destinos viram texto sem criar uma página pública. A prévia mostra texto protegido e deve permanecer local, não exposta como site público. Narrar uma nota com senha é recusado; aplicar senha depois não apaga automaticamente um MP3 já existente. O feed de podcast exclui conteúdo protegido, escondido ou rascunho.

[attachments.go](../internal/lock/attachments.go) incorpora referências locais de anexos do corpo protegido como URLs `data:` antes da cifra, com teto de 10 MiB por arquivo. Imagens responsivas escolhem uma versão e removem as referências alternativas. `lock.Prune` remove mídias geradas sem referência na saída pública, inclusive as usadas só em conteúdo protegido ou comentários. Um anexo também usado por uma página pública continua público. Capas/metadados explícitos, arquivos em overrides e arquivos acima do teto não recebem garantia de sigilo por esse mecanismo; um anexo grande exclusivo pode deixar de funcionar após a poda. O MP3 indicado fora do corpo também exige cuidado. A rotina não equivale a cifrar todo o diretório de saída.

Marcadores de proteção são verificados mesmo com manifesto vazio; erro apaga a saída e impede publicar texto provisório. Comentário `%%` sem fechamento esconde o restante do conteúdo renderizado, preservando código conforme o filtro. Essas regras não apagam conteúdo de releases ou caches anteriores.

O tema usa escape e Goldmark sem HTML arbitrário habilitado. O feedback recebido passa por `markdown` e `safeURL` em [feedback.js](../theme/assets/js/feedback.js), que copiam elementos permitidos e filtram endereços. Por escolha de produto, a autorização GitHub do visitante fica em `localStorage` para persistir entre visitas; sair remove a chave. Estado temporário do retorno OAuth e caches usam `sessionStorage`. Essa persistência exige evitar scripts ativos vindos de comentários e não cifra conversas públicas.

No Studio, [guard.go](../studio/guard.go) exige POST e `x-wails-client-id` para `/wails/runtime` e recusa metadados de origem cruzada nas rotas Wails. A prévia passa por [PreviewGuard](../internal/app/preview.go), proxy em `127.0.0.1` com porta aleatória: Host deve ser `127.0.0.1:P` ou `localhost:P`, e Origin, quando presente, deve ser da prévia ou de uma origem autorizada do app. Hugo fica em outra porta interna aleatória. Host/Origin limitam uso cross-site e DNS rebinding. A prévia fica em iframe de outra origem local com sandbox; o receptor confere origem e identifica o frame. `OpenURL` permite somente HTTP, HTTPS, e-mail e telefone. As APIs de arquivos usam `app.Within` para verificar caminhos após links simbólicos. Isso reduz caminhos de ataque de notas e HTML da prévia; não é proteção contra código local com acesso à conta do sistema.

O Studio declara CSP em `studio/frontend/index.html`, com scripts da própria origem, permissão necessária ao WASM e iframe de prévia em loopback. O frontend de desenvolvimento tem ajustes próprios para Vite. O painel da conta tem CSP gerado pelo SvelteKit e headers em `static/_headers`; o Worker aplica headers também às próprias respostas. Os sites de usuários têm outro Worker e outros headers. A memória/cache do servidor, o service worker do site e o cache do navegador têm contratos distintos; excluir ou substituir uma publicação não é promessa de apagar cópias já baixadas por visitantes.

## Staff, autenticação e cobrança

Better Auth valida a origem configurada e usa OTP armazenado por hash, passkeys e sessões. O cadastro com passkey sem sessão também passa por Turnstile e limite por IP. O device flow aceita somente o identificador do Studio e requer aprovação no navegador. A configuração concreta é [auth-options.ts](../cloud/worker/auth-options.ts); a referência de proteção de origem é [Better Auth security](https://better-auth.com/docs/reference/security).

Cloudflare Access protege o host da equipe. A API valida novamente o JWT, issuer e audience com JWKS, e só então consulta `support`/`admin` ou o proprietário de bootstrap. A staff pode ver metadados de conta, planos, uso e eventos financeiros, revogar acesso e executar operações autorizadas, mas não possui a chave de notas nem uma função de impersonação. As ações são registradas em `staff_log`. A configuração de teste por `STAFF_DEV_EMAIL` só se aplica a hosts locais e não deve existir em produção.

O Stripe recebe pagamentos em páginas próprias. O handler próprio [billing.ts](../cloud/worker/billing.ts) verifica a assinatura do corpo original, limita seu tamanho e confere o ambiente test/live. Zod valida os objetos externos usados na reconciliação. Um evento fornece IDs; o estado mutável vem de consultas atuais ao Stripe. Lease por assinatura antecede o retrieve; lease por conta serializa a adoção de referência pendente compartilhada por assinaturas distintas. Estado, registro financeiro e recibo por `event.id` são gravados no mesmo batch, condicionado às leases. Falha ou disputa retorna 503 para repetição; assinatura inválida retorna 400. Os handlers webhook/success do plugin foram desativados para impedir sobrescrita por snapshots. O success próprio só valida e redireciona para URL local, sem mudar plano. Cancelar ou mudar plano altera direitos conferidos nas APIs; checkout aberto não concede plano pago. Recibos ficam elegíveis para limpeza após 90 dias, até 500 por cron diário, podendo permanecer mais com acúmulo; também entram na purga de conta. Migração `0013` é obrigatória para esse contrato.

Segredos ficam no ambiente, sem entrar em `src/` ou assets. O catálogo `/plans` não fixa valores: período sem preço válido é `null` e seu checkout é bloqueado no servidor. Esses controles não substituem operação do Stripe, resposta a disputas ou política de privacidade pública. O template `internal/app/privacy.go` pertence aos sites das pessoas; a Store depende da publicação de `/privacidade`, `/termos` e `/feedback` no institucional Hugo privado. São URLs a publicar em `getblup.io`; o canal de relato de IA será `/feedback`, com contato `help@getblup.io`, sem botão de denúncia no Studio. Veja [políticas de distribuição](auditoria/politicas-distribuicao.md) e [configuração operacional](deploy.md).

## Cotas, concorrência e revogação

A [tabela dos planos](spec/blup-cloud-sync.md#planos-executados) vem de `plans.ts`: Grátis 50 MiB, Pro 1 GiB, Max 5 GiB e Ultra 10 GiB; notas e sites dividem o total. Arquivo máximo, na mesma ordem: 10, 25, 50 e 100 MiB. Sites hospedados: 0, 1, 3 e 10. O cliente valida tamanhos, blocos e caminhos do manifesto decifrado e limita a remontagem aos bytes declarados, até 100 MiB. O servidor de sync vê blocos cifrados, sem tamanho em claro por arquivo para aplicar sozinho esse limite lógico. Blocos e manifestos têm tetos próprios.

[quota.ts](../cloud/worker/quota.ts) usa reservas condicionais no D1 antes de criar sites/cofres/aparelhos, consumir armazenamento ou chamar IA. Uso confirmado e reservas em andamento entram na admissão; a liberação é idempotente e o cron expira reservas. No upload de site, tamanho conhecido permite streaming incremental com staging R2 e hash, até o limite do plano; sem tamanho conhecido, o buffer fica limitado a 4 MiB. Leases por site serializam publicação e limpeza; manifestos substituídos protegem seus arquivos por pelo menos dez minutos. Isso fecha as reproduções anteriores de ultrapassar cotas ou remover arquivos ativos por concorrência.

A IA reserva uso mensal e semanal antes da chamada. A voz prefere custo reportado nos headers de uso, senão a estimativa de catálogo. Tradução contabiliza `usage.cost` quando disponível, inclusive em resposta cobrada porém incompleta ou inválida. A confirmação de custo ocorre antes de devolver o áudio ou texto. Não há transação distribuída com o provedor: uma falha de rede ou após o trabalho remoto não garante resultado entregue nem ausência de cobrança externa. Repetir uma chamada pode produzir mais trabalho pago.

Fusíveis de tráfego separam controle (120/minuto por conta e IP), transferências (500/dez segundos por conta) e OTP (cinco/minuto por IP). Não substituem as cotas comerciais ou a autorização. As regressões resolvidas estão como testes normais, sem `it.fails`, em `security.test.ts` e `serve-site-security.test.ts`.

O gateway público tem cache desativado e verifica conta, bloqueio, exclusão, plano, quantidade retida, referência ativa, hostname e prazo antes de chamar o entrypoint `CachedSite`. A Cache API anterior foi removida. Props confiadas de site/deploy separam conteúdo mesmo com host/query normalizados; cookies e Authorization não chegam ao cache público. Purga persistente por tags executa por `SITE_CACHE` no contexto dono do cache. Uma fila de purga atrasada não pula a autorização do gateway. Exclusão e bloqueio impedem novas entregas do servidor; não removem cópias no navegador ou offline. O Studio agora inclui `BlupSite` na seleção de exclusão remota, separada da remoção local e do cofre sincronizado.

## Retenção e direitos após mudança de plano

[lifecycle.ts](../cloud/worker/lifecycle.ts) implementa exclusão lógica imediata: revoga sessões e aprovações, bloqueia APIs/sites, cancela lembretes pendentes e agenda purga para depois de 30 dias. O cancelamento de assinatura é tentado em segundo plano e repetido. A staff pode restaurar antes de começar a purga e antes de vencer o prazo; isso não refaz uma assinatura já cancelada nem fornece a chave das notas.

A purga diária opera em lotes de até 25 contas e aguarda o cancelamento financeiro. Remove prefixos R2 e registros D1 de produto e autenticação. O Custom Hostname é apagado na Cloudflare antes de descartar sua referência local, tanto ao excluir site quanto na purga. Falha conserva os identificadores para repetir. Trinta dias é o prazo mínimo para elegibilidade, sem garantia de remoção no instante exato. Backups/cópias locais, dados já baixados e retenções do Stripe têm outro ciclo.

Redução com excesso de sites/storage abre 14 dias, sem ampliar prazo já aberto. Durante a carência de storage vale o teto anterior, com uso total incluindo histórico; depois volta ao novo teto, sem apagar dados excedentes automaticamente. Depois do prazo de sites, Pro mantém o escolhido ou o mais antigo por data/ID, e Grátis mantém zero. A seleção só aceita site do dono até o prazo. Arquivos dos demais ficam guardados. Domínio próprio tem prazo separado de 14 dias ao sair de Max/Ultra, exposto em `disableAt`; voltar antes do desligamento pode cancelá-lo. Após desativação, reconfigurar exige plano elegível e nova prova.

Max/Ultra permitem criar um hostname por site, com prova TXT aleatória em `_blup.<hostname>`, obrigatória antes de provisionar/servir. A API verifica unicidade dos vínculos comprovados e Cloudflare/SSL; só apontar CNAME não basta. O mapeamento de hostname tem cache de até 30 segundos, mas o gateway relê vínculo exato/status/prazo no D1, impedindo usar um mapeamento antigo para liberar domínio removido. No desligamento, só remove externamente depois de ganhar `UPDATE ... RETURNING`; um upgrade que limpou o prazo impede o DELETE. A integração SaaS e a baseURL/republicação estão em [publicação](spec/publicacao.md#domínio-próprio-no-blup-cloud).

O preparo de sync reserva saldo líquido sob lease exclusiva da conta. Uploads normais usam leases por bloco compatíveis com concorrência do cliente; não entram durante preparo/purga. Um PUT preparado usa chave física única e adoção por CAS, após reler estado dentro da lease. O commit referencia somente o recibo adotado, preservando bytes diante de PUT atrasado ou confirmação D1 perdida. Head, referências, uso e remoção lógica/fila de descarte são confirmados juntos no D1. A exclusão R2 vem depois; só então desconta bytes, de forma idempotente. Essas proteções não criam transação distribuída D1/R2 nem idempotência sem prazo.

Acima da cota, a operação sem crescimento líquido pode continuar. A exceção para versão sem blocos novos limita o manifesto cifrado ao tamanho real do manifesto do head no R2 mais 4096 bytes; acima disso retorna `413 QUOTA`, e head sem metadados disponíveis retorna 503. A folga permite renomear ou remover sem autorizar inflação arbitrária do manifesto. Os bytes reais continuam contabilizados, e crescimento normal reserva armazenamento mesmo sem blocos novos. Não se apagam dados só para caber no novo plano. A política normal de expiração/rareamento de histórico continua. A purga manual `DELETE /api/sync/vaults/:id/history` só limpa o cofre selecionado, preserva head e blocos usados por outros e não desconta em falha R2. O tamanho de histórico exibido é da conta; `null` significa indisponível, nunca zero presumido.

Lembretes têm até 300 itens por lista, 100 vencidos por rodada e até três tentativas de envio, com espera de cinco minutos e leases recuperáveis. Só envios aceitos pelo provedor contam na cota mensal Max 50/Ultra 100; isso não comprova entrega na caixa de entrada. Uma queda depois da aceitação e antes de persistir pode repetir uma mensagem. Cancelar preferências ou sair tenta esvaziar a lista, com repetição se houver falha de rede. A fila não consegue desfazer um envio já em andamento. [Calendário](spec/calendario-e-lembretes.md) descreve IDs e retenção curta da fila.

## Tipos, arquivos e atualização

A [spec de contratos](spec/tipagem-e-contratos.md) descreve Zod nas entradas Hono, tipos RPC inferidos, eventos Wails registrados e normalização de arquivos externos. Tipos TypeScript, generics Drizzle e `json.Unmarshal` não validam sozinhos a forma em execução. Frontend do Studio usa narrowing para dados dinâmicos em vez de `any`; bindings gerados podem representar `any` Go e devem continuar gerados.

Os wrappers genéricos de GitHub e Cloudflare Workers continuam como limite residual: decodificam DTOs Go sem schema semântico estrito; JSON `null` ou campos ausentes podem virar zero value. Validações de permissão e verificações posteriores continuam valendo. Esse ponto foi aceito pela equipe para outra rodada, sem afirmar validação uniforme de todo provedor externo.

Autoatualização da CLI verifica Ed25519 sobre os bytes exatos de `checksums.txt`, com chaves públicas confiadas em `internal/update/keys.go`, e só depois SHA-256 do arquivo. Recusa assinatura ausente/inválida, hash divergente e versão igual ou anterior. A assinatura vem de `scripts/sign-release`; a chave privada fica fora do repositório e só seu nome de secret aparece na configuração CI. A instalação inicial por `install.sh` ou `install.ps1` ainda usa somente SHA-256, com confiança no download da release; não recebe a verificação Ed25519 do atualizador. Assinatura nativa e notarização do Studio são etapas separadas de distribuição.

## Limites e decisões de produção

1. Provisionar e verificar Access, Turnstile, remetente, secrets, webhook e Cloudflare for SaaS no ambiente correto. Placeholders não comprovam operação segura.
2. Comprovar inclusão e propagação de `getblup.site` na [PSL](https://publicsuffix.org/submit/) e cobertura TLS das variantes anunciadas. O repositório não comprova essas etapas externas.
3. Preparar recuperação operacional de D1/R2, monitoramento de purga/cron e tratamento de falhas externas. Restauração dentro dos 30 dias não é backup independente do serviço.
4. Medir cache/CPU/R2/D1 e validar purga/103 em staging, preservando o gateway sem cache e o entrypoint interno sem rota pública. Budget alert avisa, sem bloquear custos.
5. Testar os formatos desktop publicados e informar a distribuição Windows/macOS sem assinatura de desenvolvedor definida para este lançamento. Os passos de exceção por aplicativo estão no [guia de instalação](instalacao.md); não exigem desativar a proteção global. Manter a chave Ed25519 da CLI privada e planejar rotação compatível. Instalação inicial só por SHA-256 e DTOs residuais dos provedores continuam limites conhecidos.

Os testes automatizados exercitam criptografia, caminhos, entradas de API, contratos e cotas em simulações locais. Não substituem revisão do ambiente real nem autorizam ataques contra produção. Os comandos estão em [testes e build](spec/testes-e-build.md).
