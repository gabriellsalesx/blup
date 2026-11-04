# Blup Deals

Especificação de produto e técnica baseada na nota conectada **Blup Deals- decisões**, aprovada pelo dono. Segue a organização da [spec do Pulse](blup-pulse.md). Contratos de API, tabelas e proteções descritos como propostos orientam a implementação; não são uma afirmação de entrega, homologação de lojas ou liberação comercial. Exemplos usam marcadores, sem credenciais reais.

## Estado da entrega por fases

**Especificação inicial, com pesquisa incorporada.** A nota define F0 a F3. Este documento não atribui gates ou funcionalidades entregues a nenhuma fase sem um handoff e conferência do código correspondente. Incorpora a nota conectada **Blup Deals- pesquisa**, lida integralmente: a Lupa conferiu documentação oficial, incluindo o MCP Cloudflare Docs, sem executar integrações de negócio. Tarifas são referências em USD; cadências, consumo estimado e contratos propostos continuam identificados como tal.

As bases já existentes que podem ser aproveitadas são [API Cloud](../../cloud/worker/index.ts), [autenticação](../../cloud/worker/auth-options.ts), [cobrança](../../cloud/worker/billing-api.ts), [fila Pulse](../../cloud/worker/pulse/outbox.ts), [transportes Pulse](../../cloud/worker/pulse/transports.ts), [cliente RPC Pulse](../../cloud/pulse/src/lib/api.ts), [PanelShell](../../cloud/src/lib/ui/PanelShell.svelte), [SignInForm](../../cloud/src/lib/ui/SignInForm.svelte), [SignInFrame](../../cloud/src/lib/ui/SignInFrame.svelte) e [SetupGuide](../../cloud/pulse/src/lib/SetupGuide.svelte). Reutilizar essas peças exige adaptar seus contratos: um alerta privado Pulse e uma oferta publicada em canal têm conteúdo, permissões e limites diferentes.

## 1. Produto e público

O Blup Deals ajuda uma pessoa a manter canais de ofertas no Telegram e no Discord. Ela conecta o destino, informa seus identificadores de afiliado quando os tiver, escolhe assuntos e revisa o primeiro post. Depois pode publicar um link manualmente ou ativar caçadas automáticas. O foco é quem está começando, sem acesso a APIs de lojas e sem histórico de vendas.

Uma **oferta** é um produto, promoção ou brinde com origem e data de verificação. Uma **caçada** é uma regra que escolhe ofertas de uma fonte para certos canais. Um **espelho** é uma caçada cuja fonte é um canal acessível pelos meios técnicos definidos na seção 6, sem exigir autorização do dono da origem por decisão do dono do Blup. Um **canal** é o destino confirmado onde o bot pode publicar. Um catálogo global evita buscar a mesma fonte para cada assinante.

O produto inclui jogos, links colados e espelhamento. O Blup ajuda a preparar e distribuir posts; comissão depende de cadastro, aprovação, atribuição e regras da loja. Não prometer renda, aprovação em programa de afiliados, preço disponível até a compra ou comissão em qualquer link. WhatsApp fica fora da Fase 1. O produto usa bots oficiais, sem conta pessoal automatizada ou userbot.

## 2. Planos e limites

Deals é um adicional **somente do Ultra**. O card do Ultra tem o switch **Adicionar Blup Deals**, independente do switch Pulse. Não haverá Deals avulso ou adicional de Grátis, Pro ou Max nesta proposta.

| Variante Ultra | Pulse | Deals |
| --- | --- | --- |
| `ultra` | Não | Não |
| `ultra_pulse` | Sim | Não |
| `ultra_deals` | Não | Sim |
| `ultra_pulse_deals` | Sim | Sim |

As variantes preservam os direitos do Ultra: 10 GiB de notas, arquivos de até 100 MiB, 10 sites, computadores sem limite de quantidade e os demais direitos da [spec Cloud](blup-cloud-sync.md#planos-executados). Posts, fontes e cliques do Deals não são notas sincronizadas nem consomem sites Hugo. Deals não concede Pulse quando o respectivo switch está desligado.

| Limite da nota, ainda proposto | Ultra com Deals |
| --- | --- |
| Canais de destino ativos | 10 no total, somando Telegram e Discord |
| Caçadas | 20 no total |
| Espelhos | 10, contados também dentro das 20 caçadas |
| Posts por dia | 100 no total da conta (decisão do dono, 04/11/2026) |
| Histórico | 90 dias |

A nota pede validar esses limites com custos. A definição proposta de post é uma publicação em **um destino**: enviar a mesma oferta a três canais consome três unidades. Repetir a tentativa da mesma entrega não cria três novas reservas. Usar dia civil UTC, sem acumular saldo; exibir o horário de renovação no painel. Testes devem cobrir meia-noite, virada de mês/ano e mudança de assinatura sem zerar consumo. Essa semântica precisa ser confirmada antes de vender.

Reservar a cota em D1 antes de enfileirar, sob condição atômica. Uma reserva não entregue deve ser concluída ou liberada uma única vez, com recibo que impeça recriar entregas para contornar a cota. A escolha de cobrar ou liberar uma tentativa cujo aceite pelo provedor ficou incerto é uma pendência explícita. Limites de leitura das fontes e dos bots são adicionais à cota comercial.

## 3. Blup, cor e domínios

| Item aprovado | Valor |
| --- | --- |
| Blup do produto | `turma-8`, coral `#ff7a59` |
| Destaque do painel | `--color-blup: #c2410c`, para texto branco com contraste AA |
| Painel privado | `https://deals.getblup.io` |
| Links curtos, F3 | `https://deals.getblup.site` |
| Vitrine, F3 | Gerada e publicada pelo Blup Studio a partir da conexão Deals; sem vitrine externa no Deals |

Reaproveitar o shell, temas claro/escuro, foco de teclado e componentes do Cloud. Interface e mensagens usam `pt/en/es`; os caminhos ficam em inglês. Nomes e textos de ofertas mantêm o idioma da fonte, salvo edição explícita. Ícones funcionais vêm do Lucide; marcas, quando necessárias, usam os arquivos Simple Icons CC0 permitidos pelo projeto. Não desenhar marcas nem usar emoji como ícone de interface.

O coral da personagem e a cor de destaque têm funções diferentes. Conferir também contraste de bordas, estados desabilitados, links e foco, sem usar só a cor para informar falha ou oferta expirada.

## 4. Fontes de ofertas por loja e jogos

As fontes abaixo constam da decisão de produto. A coluna de condição distingue contratos de API pesquisados, validações ainda pendentes e o risco do espelhamento aberto assumido pelo dono. Fonte pública não significa licença irrestrita para republicar seu conteúdo; essa ressalva não cria uma exigência de autorização do dono da origem no Deals.

| Fonte | Uso pretendido e fase | Condição de entrada |
| --- | --- | --- |
| GamerPower | Jogos grátis, brindes e resgates, sem chave; F1 | Uso comercial permitido com hyperlink ativo de atribuição; até 10 requests/s; não revender o catálogo ou declarar os dados como próprios |
| CheapShark | Promoções de jogos de PC e histórico; F1 | Fonte mantida por decisão do dono, com os links da própria CheapShark, sem substituir pelo afiliado do usuário; a documentação permite assinatura e proíbe afiliados próprios nas lojas acompanhadas |
| Amazon | Link manual com a tag da pessoa; automação F3 | Creators API da própria pessoa, com pelo menos 10 vendas qualificadas nos últimos 30 dias; Telegram/Discord não têm aprovação nominal geral confirmada |
| Mercado Livre | Link manual; caçadas de loja F2 | `matt_word` sozinho não tem contrato oficial de geração de afiliado confirmado na pesquisa inicial; APIs exigem autenticação. Não apresentar um parâmetro montado como comissão validada |
| Shopee | Caçadas com credencial da própria pessoa, F2 | API oficial gera até 50 links por lote; habilitação e cota dependem da conta/região |
| AliExpress | Caçadas com credencial da própria pessoa, F2 | Método oficial gera até 50 links por chamada; exige App Key, assinatura e Tracking ID habilitados |
| Outra loja | Colar link, F1 | Aceitar URL pública válida, com edição manual do post; não inventar conversão de afiliado para uma loja sem adaptador |
| Canal Telegram | Espelhamento aberto, F2 | Só canais públicos pela prévia `t.me/s/<canal>`, sem login/userbot; não exige autorização do dono da origem; risco dos termos registrado e assumido pelo dono do Blup |
| Canal Discord | Espelhamento aberto, F2 | Só servidores onde o bot foi adicionado e canais com permissões técnicas de leitura; não exige autorização adicional do dono da origem; risco dos termos assumido pelo dono do Blup |

O dono decidiu manter CheapShark como fonte, usando os links da própria CheapShark, sem trocar pelo afiliado do usuário. Essa é uma exceção explícita à personalização de afiliados. A regra também vale para uma oferta depois encontrada em outra loja: não usar a integração para descobrir uma promoção e contornar a restrição trocando seu afiliado fora do caminho publicado pela fonte. O risco de coleta automatizada excessiva permanece registrado. [Documentação oficial CheapShark](https://apidocs.cheapshark.com/).

### Jogos: atribuição e coleta

GamerPower permite uso comercial gratuito com atribuição por link ativo. Manter esse crédito no post e na futura importação Studio. O limite publicado de 10 requests/s é teto externo, não frequência desejada do coletor. Cobrar pela ferramenta de automação não deve virar venda de dump dos dados sem permissão. [API e termos GamerPower](https://www.gamerpower.com/api-read).

CheapShark permite assinatura e anúncios, exige User-Agent e os seus links de redirect. Não seguir automaticamente esses redirects para extrair/substituir o destino. Preços usam USD; a atualização costuma ser horária. Não há cota universal numérica publicada: respeitar 429/Retry-After e acertar o volume do catálogo compartilhado antes de ativar coleta sistemática. A documentação alerta para bloqueio por automação excessiva. Essa restrição prevalece sobre uma regra geral de trocar afiliados nos posts. [Usage Notes e Caching](https://apidocs.cheapshark.com/).

### Mercado Livre

Busca e recursos públicos exigem access token; autenticação não garante acesso a todo endpoint. Usar OAuth autorizado, sem pedir senha; a rotação do refresh token precisa ser serializada por conta. A geração de afiliado oficialmente descrita usa Gerador de Links/Barra de Afiliados. Etiqueta de campanha não comprova a identidade do afiliado. Até existir contrato de conversão validado, aceitar e preservar o link que a pessoa já gerou oficialmente. [Autenticação dos recursos](https://developers.mercadolivre.com.br/pt_br/desenvolvimento-seguro), [OAuth](https://developers.mercadolivre.com.br/pt_br/mensagens-post-venda/autenticacao-e-autorizacao), [geração de links](https://www.mercadolivre.com.br/l/afiliados-gere-seus-links) e [etiquetas](https://www.mercadolivre.com.br/l/afiliados-crie-etiquetas).

O checklist brasileiro inclui Telegram público e restringe grupos privados e locais não declarados. Discord não aparece como autorização nominal geral nessa relação. Conferir os destinos declarados pela pessoa, sem generalizar a permissão de um canal para todos. [Checklist](https://www.mercadolivre.com.br/l/checklist), [perguntas frequentes](https://www.mercadolivre.com.br/l/primeiros-passos-perguntas-frequentes-para-afiliados).

### Shopee e AliExpress

O Explorer brasileiro da Shopee expõe GraphQL e assinatura SHA-256 com App ID, timestamp, payload e Secret. `generateShortLink` aceita até cinco `subIds`; `generateBatchShortLink`, até 50 links por chamada. Isso comprova o método para conta habilitada, sem comprovar uma cota universal ou acesso imediato de uma conta iniciante. Sem API, a pessoa pode colar seu link já convertido oficialmente. [Explorer](https://open-api.affiliate.shopee.com.br/explorer/v2), [acesso à API](https://affiliate.shopee.com.br/open_api) e [geração manual](https://help.shopee.com.br/portal/10/article/128461-Como-gerar-seus-links-de-Afiliado-ou-ID-de-produto-para-compartilhar).

AliExpress documenta `aliexpress.affiliate.link.generate`, com `app_key`, timestamp, assinatura por App Secret, `tracking_id` e até 50 URLs em `source_values`. O método consta como `chargeable=false`; `access_token` é opcional **nesse método**, sem dispensar credenciais ou aprovação. Preservar o `promotion_link` recebido. O contrato público consultado não fixa requests/s ou cota mensal universal; verificar o console e o uso das chaves por um serviço terceiro. [Método oficial](https://openservice.aliexpress.com/doc/api.htm#/api?cid=3&path=aliexpress.affiliate.link.generate&methodType=GET%2FPOST), [schema oficial](https://openservice.aliexpress.com/handler/share/apidoc/getApi.json?cid=3&path=aliexpress.affiliate.link.generate&methodType=GET%2FPOST).

### Amazon: acesso e canais

A ajuda brasileira de candidatura lista determinadas redes sociais, sem aprovação nominal geral de Telegram/Discord. A permissão contratual para mensagens solicitadas não autoriza automaticamente qualquer canal. Confirmar o destino com o programa antes de anunciar essa integração. Manter identificação publicitária clara e a declaração de Associado exigida, inclusive ao limpar um espelho. Tag tecnicamente montável não comprova que a publicação é elegível. [Candidatura](https://associados.amazon.com.br/help/node/topic/G8TW5AE9XL2VX9VM), [políticas](https://associados.amazon.com.br/help/operating/policies) e [divulgação](https://associados.amazon.com.br/help/node/topic/GPXFHVYZMTGPUMPE).

Automação usa Creators API, sucessora da PA-API 5, com credenciais próprias e ao menos 10 vendas qualificadas em 30 dias. A documentação inicial indica até 1 request/s e 8.640/dia nos primeiros 30 dias, respeitando ambos os tetos; depois ajusta por receita atribuída, até 10 requests/s. `GetItems` aceita até dez ASINs por chamada. Não apresentar uma conta sem vendas como pronta para catálogo automático. [Introdução](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/introduction), [limites](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/concepts/api-rates) e [substituição da PA-API](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/paapiv5-deprecation).

Autenticação usa o fluxo OAuth da Creators API, sem reaproveitar chaves AWS da API antiga. Boas práticas indicam cache de Offers/BrowseNodeInfo por uma hora e outros campos por um dia. Preservar parâmetros dos links devolvidos. Histórico de 90 dias, imagens e encurtamento Blup dependem da licença específica, sem aplicar a retenção geral automaticamente. [Autenticação](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/get-started/using-curl), [cache e links](https://affiliate-program.amazon.com/creatorsapi/docs/en-us/concepts/best-programming-practices).

O catálogo guarda a origem de cada observação, loja, moeda, região, preço, instante de coleta, validade quando informada e link permitido. Um preço riscado só aparece se a fonte o sustentar; menor preço observado significa o menor **na janela coletada**, não um recorde de toda a loja. Ausência de preço é `null`, não zero. Brinde gratuito deve indicar plataforma, requisito e prazo quando conhecidos, sem confundir com teste temporário ou assinatura necessária.

Não combinar preços de moedas, edições, plataformas ou regiões diferentes para calcular desconto. Cupons e frete devem aparecer como condições, sem reduzir silenciosamente o preço. Uma fonte indisponível mostra atraso e suspende suas caçadas; não fabricar ofertas nem apresentar o último preço como recém-verificado.

## 5. Afiliados e colar link

O formulário Afiliados apresenta a situação por loja: sem identificador, identificador salvo, API não configurada, aguardando aprovação, conexão válida ou falha de conexão. ID público de afiliado e segredo de API são campos distintos. Salvar um ID não comprova que a conta pode usar uma API ou receber comissão.

Fluxo proposto de **Colar link**:

1. Validar a URL, identificar a loja e remover apenas parâmetros que o adaptador conhece como rastreio anterior.
2. Expandir link curto somente por adaptador permitido, com limite de redirects e validação de cada destino público. Não fazer fetch irrestrito de uma URL colada.
3. Consultar metadados apenas pelo método permitido da loja. Sem acesso, pedir título, texto, preço opcional e imagem opcional à pessoa.
4. Gerar o link afiliado pelo contrato da loja. Sem suporte ou credencial, preservar um link de produto limpo e avisar **Sem comissão configurada**; a pessoa pode fornecer seu link já emitido pelo programa, quando permitido.
5. Mostrar a prévia com destino, título, imagem, preço/moeda, condições e indicação de link afiliado. Só publicar após confirmação, salvo uma caçada ativada explicitamente.

A instrução original da nota de montar Amazon/Mercado Livre apenas com `tag`/`matt_word` fica como intenção de simplificar o cadastro. A implementação precisa seguir a pesquisa oficial por loja. Não aplicar uma troca genérica de query string a links assinados, cupons, encurtadores ou caminhos de atribuição de outro programa.

URLs só podem usar HTTP(S), sem usuário/senha, caracteres de controle ou esquema executável. Fetchs do Worker exigem destino público, timeout, teto de bytes e política de redirects por adaptador. Não buscar imagens arbitrárias em rede privada. A URL da loja vai direto ao `sendPhoto` ou embed quando seu uso for autorizado; a F1 não baixa nem guarda imagens em R2.

## 6. Espelhamento: funcionamento, limites e cuidados

Espelhamento faz parte do produto aprovado e fica **aberto, sem exigir autorização do dono do canal de origem**, por decisão do dono do Blup. É um tipo de fonte de Caçadas, com origem identificada, destino, filtros e intervalo. O dono assume o risco dos termos do Telegram e do Discord; essa decisão não equivale a uma licença concedida pelas plataformas. Não criar uma sexta aba apenas para espelhos.

### Fluxo proposto

1. A pessoa cadastra a origem: canal público Telegram por `t.me/s` ou canal Discord acessível ao bot instalado no servidor. Não há etapa de autorização do dono da origem.
2. O servidor normaliza a identidade da fonte e cria ou reaproveita uma assinatura global. Dez pessoas usando o mesmo canal não causam dez coletas da mesma página por rodada.
3. O coletor lê só mensagens novas, com cursor persistente, tamanho limitado e sem voltar indefinidamente no histórico.
4. O adaptador extrai campos de oferta e produtos reconhecidos. HTML, embeds e mensagens são dados não confiáveis, nunca componentes executáveis do painel.
5. A caçada filtra as ofertas, remove o texto promocional específico do afiliado anterior e gera o vínculo da própria pessoa **quando o programa permitir**. Mantém a procedência e os avisos obrigatórios; não atribui autoria ao Blup ou ao novo afiliado.
6. A prévia mostra a transformação. Depois de ativada, a caçada usa o mesmo renderizador e fila dos posts manuais, com dedupe e limites por conta/destino.

### Telegram

A origem aprovada é a pré-visualização pública `https://t.me/s/<canal>`, sem sessão, número de telefone, userbot ou acesso a canais privados. A Bot API não oferece uma consulta geral ao histórico de qualquer canal público. O Deals não exige autorização do dono da origem. Ler HTML público não concede licença geral para republicar textos/imagens de terceiros ou remover atribuição: o risco contratual fica registrado e assumido pelo dono do Blup. O coletor preserva os limites técnicos, sem contornar bloqueio ou limite. [Licenciamento de conteúdo](https://telegram.org/tos/content-licensing), [termos de bots](https://telegram.org/tos/bot-developers).

A pesquisa não encontrou cota, SLA ou licença geral de espelhamento comercial por `t.me/s`. A prévia é um recorte, não contrato de histórico completo. Por decisão do dono, a habilitação não fica limitada a canais próprios/autorizados nem depende de confirmação escrita do Telegram. Isso não elimina o risco dos termos ou direitos de terceiros. Proteção de conteúdo continua sendo respeitada.

Mudança de HTML, resposta 403/429, canal removido ou bloqueio técnico da origem pausa a coleta e mostra motivo curto. Não resolver por proxies rotativos, login pessoal ou coleta oculta. Conteúdo sem produto reconhecível fica para revisão, sem copiar automaticamente todo o post.

### Discord

O bot só acessa canais permitidos de um servidor em que foi instalado. Verificar `VIEW_CHANNEL`, `READ_MESSAGE_HISTORY` e `MESSAGE_CONTENT` para texto/embeds/anexos comuns. A restrição de conteúdo alcança REST também: polling não contorna o intent. Adicionar o bot ao servidor não concede todas as permissões. O projeto precisa decidir entre leitura HTTP incremental e Gateway na F2. Não considerar que um webhook de saída recebe mensagens de outros canais. [Messages](https://docs.discord.com/developers/resources/message).

Não exigir autorização adicional do dono do canal de origem para espelhar. Essa abertura foi decidida pelo dono do Blup, que assume o risco dos termos e do conteúdo de terceiros. Instalação do bot, acesso ao servidor e permissões técnicas continuam obrigatórios.

Se houver Gateway, revisar intents, heartbeat, reconexão, retomada e infraestrutura persistente, sem prometer que um cron substitui a conexão. A F1 permanece sem DO. A política anunciada em 11/06/2026 separa dois processos: revisão de intents privilegiados ao alcançar 10.000 usuários únicos com acesso ao app, com revisão anual depois da aprovação; verificação do aplicativo para passar de 100 servidores. A revisão de intent deve ser solicitada no prazo de 90 dias após a notificação indicada pelo Discord. Conferir o estado real no Developer Portal, sem usar a regra antiga de 100 servidores como único limiar de intent. [Gateway](https://docs.discord.com/developers/events/gateway), [mudança dos intents](https://support-dev.discord.com/hc/en-us/articles/40281523410967-Changes-to-Privileged-Intent-Access-for-Discord-Apps) e [verificação do app](https://support-dev.discord.com/hc/en-us/articles/23926564536471-How-Do-I-Get-My-App-Verified).

### Dedupe e repetição

Guardar a identidade da mensagem de origem e uma chave normalizada do produto/oferta. A chave da entrega inclui conta, destino e ocorrência publicável; repetir a coleta não reenfileira a mesma ocorrência. Oferta igual em dois espelhos não produz dois posts no mesmo canal. Mudança real de preço ou novo período de gratuidade pode gerar nova ocorrência conforme a regra da caçada. Não usar só um hash do texto, pois mudar pontuação contornaria o dedupe.

Os dez espelhos são limite por conta, não dez downloads obrigatórios por rodada. O cenário de custos usa cinco minutos, como premissa de dimensionamento, sem autorização de polling nesse intervalo para toda fonte. Intervalos efetivos, limite de mensagens por fonte, janela de repetição e TTL dependem da fonte e da homologação; não publicar uma promessa de tempo real nesta revisão.

## 7. Arquitetura Cloudflare e fronteiras

```mermaid
flowchart LR
  Pessoa[Pessoa autenticada] --> Painel[deals.getblup.io]
  Painel --> API[Worker Cloud: Hono e Better Auth]
  API --> D1[D1: configuracao, catalogo e fila]
  Cron[Cron por minuto] --> Coleta[Coleta compartilhada por fonte]
  Coleta --> Fontes[APIs e canais permitidos]
  Coleta --> D1
  Cron --> Cacadas[Filtros por conta]
  Cacadas --> D1
  Cron --> Fila[Despachante persistente]
  D1 --> Fila
  Fila --> Bots[Telegram e Discord]
  Visitante[Visitante, F3] --> Publico[deals.getblup.site]
  Publico --> D1
  API --> Studio[Conexao autenticada com o Studio, F3]
  Studio --> Site[Site da vitrine gerado pelo Studio]
```

API proposta em `cloud/worker/deals`, montada em `/api/deals/*` no Worker Cloud. Painel SvelteKit estático em `cloud/deals`, com configuração proposta `cloud/wrangler.deals.jsonc`. Esses caminhos de organização não afirmam que os arquivos já foram criados. Dados em D1 com Drizzle, sem DO ou R2 na F1. A fila persistente inicial é uma tabela D1, não um novo serviço Cloudflare Queues implícito.

Cada fonte global tem uma lease, revisão, cursor e `nextPollAt`. O cron existente `* * * * *` seleciona apenas fontes vencidas, em lotes curtos; coleta uma vez, valida e salva as observações. Não executar uma varredura completa do catálogo a cada minuto. Credenciais privadas de loja pertencem à conta: só compartilhar a coleta entre usuários quando o contrato do provedor permitir; uma chave pessoal não vira chave global do Blup.

O mesmo cron filtra caçadas, cria entregas, despacha lotes e limpa expirados. Cada etapa tem orçamento de trabalho, cursor de continuação e lease independente, sem manter transação D1 aberta durante a rede. Cron atrasado não implica compensar todas as ofertas antigas em uma rajada. São 43.200 rodadas em 30 dias, compartilhadas, sem trigger por cliente/caçada. [Cron Triggers Cloudflare](https://developers.cloudflare.com/workers/configuration/cron-triggers/).

No Workers Paid, cron com intervalo menor que uma hora tem 30 segundos de CPU por invocação e até 15 minutos de duração total; espera de rede não é CPU. O runtime limita conexões de saída simultâneas a seis, memória a 128 MB e subrequests a 10.000 por invocação. Trabalhar abaixo desses tetos, dividindo o orçamento também com Cloud/Pulse, sem tentar consumir tudo em cada rodada. D1 permite até 1.000 consultas individuais por invocação e 100 parâmetros por instrução. Esses são tetos de plataforma, não tamanho recomendado de lote. [Workers limits](https://developers.cloudflare.com/workers/platform/limits/), [D1 limits](https://developers.cloudflare.com/d1/platform/limits/).

Leases e mudanças de cursor/estado usam operações condicionais, versão e recibos. Uma resposta atrasada não sobrescreve a configuração nova da pessoa ou reativa caçada sem plano. Coleta compartilhada não mistura o afiliado de uma conta com o de outra: o catálogo guarda a oferta base e a personalização acontece na entrega.

### Login e CORS

Reutilizar a sessão Cloud, com cookie restrito a `cloud.getblup.io`. O navegador do painel Deals chama a API Cloud com `credentials: include`. A nova variável **proposta** `DEALS_APP_URL` autoriza a origem exata `https://deals.getblup.io` em CORS, checagem de mutações e `trustedOrigins`, como a integração Pulse. Não usar `*` com credenciais nem habilitar cookie no domínio pai apenas para compartilhar login. A integração oficial confirma CORS explícito antes da rota de autenticação e credenciais também no cliente RPC. [Better Auth com Hono](https://better-auth.com/docs/integrations/hono#cors).

`/sign-in` usa SignInFrame/SignInForm com a identidade coral. Passkeys preservam o RP ID comum configurado no Cloud (`getblup.io`); a origem exata Deals precisa integrar a lista permitida do plugin, além do CORS. Não criar outra identidade de passkey por produto. Retorno ao Deals tem destino fixo permitido, sem redirect arbitrário por query. O host público `deals.getblup.site` não recebe credenciais ou CORS administrativo. Fontes: [segurança Better Auth](https://better-auth.com/docs/reference/security), [opções atuais de autenticação](../../cloud/worker/auth-options.ts) e [cliente existente](../../cloud/pulse/src/lib/api.ts).

O painel pode ser estático e carregar dados privados pela API. Rotas dinâmicas como `/hunts/[id]` precisam de fallback/roteamento compatível com `adapter-static`; HTML gerado no build não inclui dados de conta. [SvelteKit adapter-static](https://svelte.dev/docs/kit/adapter-static).

### Vitrine pela conexão Deals e Studio, F3

A atualização da decisão exclui uma vitrine hospedada pelo Deals. O Studio recebe ofertas selecionadas por uma conexão autenticada e monta o site da vitrine com seu pipeline de geração/publicação. `deals.getblup.site` fica para os links curtos. Não criar uma rota pública de catálogo ou vitrine nesse host.

Contrato proposto da conexão: leitura paginada de ofertas permitidas do próprio dono, com título, link final permitido, imagem autorizada, preço/moeda, origem, data de verificação, validade e aviso de afiliado. Nunca exportar chaves de lojas, destinos dos bots, mensagens brutas do espelho ou dados de outras contas. Seleção do que publicar, atualização, tratamento de ofertas vencidas e destino do site precisam de controles no Studio. Uma cópia estática não se atualiza sozinha; não prometer remover ofertas expiradas sem uma nova geração/publicação. Veja o [pipeline atual](scaffold-e-build.md) e o [Studio](studio.md).

Não conceder sites adicionais por ligar Deals. A vitrine segue o destino de publicação escolhido no Studio e as regras desse destino. Template, formato das notas importadas, frequência de atualização e eventual sincronização automática serão definidos na F3.

## 8. Modelo de dados proposto

Os nomes abaixo são proposta de schema, não migração já gerada. Todas as relações privadas são verificadas pelo dono da sessão. Identidade Cloud e assinatura continuam nas tabelas existentes.

| Tabela proposta | Campos e finalidade |
| --- | --- |
| `deals_account` | `user_id` único, revisão, pausa geral, preferências e eventual prazo de retirada do produto |
| `deals_channel` | ID, dono, tipo Telegram/Discord, rótulo, destino cifrado, estado/permissões, última falha genérica e datas |
| `deals_channel_link` | Dono, hash de código de vínculo, expiração, candidato e confirmação; sem guardar o código em claro |
| `deals_affiliate` | Dono + loja únicos, ID público, região, nível declarado/verificado, estado e datas de verificação |
| `deals_credential` | Dono + provedor, credencial cifrada/versionada, escopo, expiração e estado; sem recuperar o segredo na API de leitura |
| `deals_source` | Tipo, identidade canônica, região, estado, lease, revisão, cursor, `next_poll_at`, última coleta e erro genérico |
| `deals_offer` | Origem + ID externo únicos, produto/edição/plataforma, loja, URL canônica, título, imagem autorizada, moeda, preço em unidade menor ou `null`, validade e timestamps |
| `deals_price` | Oferta/produto, fonte, região/moeda, valor e instante observado; índices por produto e tempo; retenção limitada pelo contrato da fonte |
| `deals_hunt` | Dono, nome, tipo de fonte, filtros tipados, intervalo, janela de horário, região/moeda, estado e revisão |
| `deals_hunt_source` | Vínculo caçada/fonte; índice reverso para uma coleta alimentar várias caçadas |
| `deals_hunt_channel` | Vínculo caçada/destino, sem canal de outro dono; chave única do par |
| `deals_delivery` | Dono, canal, oferta/ocorrência, chave de dedupe única, estado, horário, lease, tentativas, ID externo e erro genérico |
| `deals_day` | Dono + dia UTC únicos; totais/reservas de posts e controle de cota atômico |
| `deals_short_link`, F3 | Código público aleatório único, dono, oferta, destino já validado, estado e expiração; sem aceitar URL livre no redirect |
| `deals_click_day`, F3 | Link + dia UTC únicos e contagem agregada; sem tabela permanente de cliques individuais |
| `deals_studio_export`, F3, se necessário | Dono, seleção de ofertas/caçadas e cursor/revisão para a conexão Studio; sem hospedar uma vitrine no Deals |

Adicionar índices para `(state,next_poll_at)`, fila por `(state,not_before,id)`, expiração e vínculos por dono. Paginação usa timestamp + ID para preservar empates. Medir o custo dos índices: D1 também conta escritas de manutenção desses índices. Migrações são geradas pelo Drizzle, revisadas e aplicadas antes do Worker que consulta o novo schema. Não reservar neste documento um número de migração que outra área possa consumir.

### Retenção e retirada

Os 90 dias da nota são proposta para histórico de atividade e estatísticas. Não autorizam guardar conteúdo de uma API por 90 dias quando sua licença exige TTL menor. Separar conteúdo de oferta, observação permitida de preço, recibo técnico de dedupe/cota e agregado diário de clique. Consultas devem filtrar expirados antes da limpeza física, feita em lotes.

Perda do direito Deals deve interromper novas coletas privadas, reservas e envios, inclusive os já enfileirados. Revalidar o direito antes de cada envio. Prazo de leitura, purga após retirar Deals e comportamento da vitrine ainda não foram aprovados. Exclusão da conta Cloud segue bloqueio imediato e a [retenção de 30 dias](../seguranca.md), integrando os dados Deals na purga e revogando vínculos externos quando aplicável. Não presumir que a carência Pulse de 14 dias foi aprovada também para Deals.

## 9. API Hono tipada

**Contrato proposto.** Todas as rotas privadas ficam sob `/api/deals`, com sessão Cloud e dono. Criação, ativação, prévias que consultam provedores e envio exigem direito gravável. `/usage` pode explicar `enabled:false`; revogar/remover um vínculo continua permitido ao dono sem Deals. Demais leituras após retirada dependem do prazo ainda a definir. A F0 implementa apenas a base necessária para telas vazias e uso; demais rotas seguem sua fase. IDs são strings opacas no padrão do Cloud, validadas em params. Não aceitar `userId`, preço Stripe, plano ou limite como autoridade do cliente.

Usar TypeScript estrito, Zod para body/query/params e respostas de provedores, com JSON de entrada estrito e tamanho limitado. Frontend importa o AppType/RPC e infere DTOs, sem duplicá-los manualmente ou converter `unknown` por cast. Middleware global de erro não aparece automaticamente na inferência RPC: tratar o envelope comum também no cliente. [RPC Hono](https://hono.dev/docs/guides/rpc), [validação Hono](https://hono.dev/docs/guides/validation) e [schemas Zod](https://zod.dev/api).

| Método e caminho relativo | Contrato proposto | Fase |
| --- | --- | --- |
| `GET /usage` | `{enabled,limits,day,posts,reserved,channels,hunts,mirrors}`; valores inteiros não negativos, `limits:null` quando não aplicável | F0 |
| `GET /offers` | Filtros `source,store,q,cursor,limit`; lista fechada `{offers,nextCursor}` | F1 |
| `POST /offers/preview` | `{url,title?,text?,price?,currency?,imageUrl?}`; retorna prévia sanitizada, situação do afiliado e ticket curto, sem publicar | F1 |
| `POST /offers/publish` | `{ticket,channelIds}` distintos; reconfere dono, configuração, validade e cota; `202 {deliveries}` | F1 |
| `GET /deliveries` | Filtros de estado/canal/cursor; `{deliveries,nextCursor}` com falha genérica, sem segredo de destino | F1 |
| `DELETE /deliveries/:delivery` | Cancela uma entrega ainda não enviada; conflito se o envio já foi reivindicado/concluído | F1 |
| `GET /channels` | `{channels}` com ID, tipo, rótulo e estado; nunca devolve token/webhook completo | F1 |
| `POST /channels/telegram/link` | `{}`; código temporário por link oficial do bot, com expiração | F1 |
| `GET /channels/telegram/link/:link` | Estado/candidato permitido, sem devolver segredo do vínculo | F1 |
| `POST /channels/telegram/link/:link/confirm` | Confirma destino depois de provar controle e permissão do bot | F1 |
| `POST /channels/discord/link` | Inicia instalação/confirmação por OAuth2 oficial com state temporário | F1 |
| `GET /channels/discord/callback` | Valida code/state e conclui verificação do canal permitido, sem redirect arbitrário | F1 |
| `DELETE /channels/:channel` | Desativa, cancela pendências e remove o vínculo; `{ok:true}` | F1 |
| `POST /channels/:channel/test` | `{}`; teste explícito e limitado; `202 {queued:true}`, não confirmação de entrega | F1 |
| `GET /affiliates` | `{affiliates}` com loja, ID público e estado da credencial; sem segredo | F1 |
| `PUT /affiliates/:store` | União discriminada por loja: identificador/região e campos permitidos; credenciais são só de escrita | F1/F2 |
| `DELETE /affiliates/:store` | Remove configuração e revoga/pausa caçadas dependentes | F1/F2 |
| `POST /affiliates/:store/test` | `{}`; verificação explícita por adaptador, sem consumir uma compra ou simular venda | F2 |
| `GET/POST /hunts` | Listagem com cursor ou criação com fonte discriminada, filtros, destinos e intervalo | F1/F2 |
| `GET/PATCH/DELETE /hunts/:hunt` | Detalhe, edição parcial estrita com revisão, pausa/exclusão e cancelamento das pendências | F1/F2 |
| `POST /hunts/:hunt/preview` | `{}`; amostra sanitizada sem publicar, informando atraso/limite da fonte | F1/F2 |
| `GET /sources` | Opções permitidas/estado global, sem credenciais ou dados privados de outros donos | F1/F2 |
| `POST /sources/check` | Verifica a fonte candidata de espelho sob limite, sem ativar coleta contínua | F2 |
| `GET /stats` | Query validada de período/canal/caçada, limitada aos 90 dias aprovados para estatísticas | F3 |
| `GET /studio/offers` | Exportação autenticada e paginada da seleção do dono para o Studio, com DTO publicável e sem credenciais; mecanismo de conexão ainda a definir | F3 |

Os exemplos de shape são propostas, não bindings já publicados. Limite sugerido de página: padrão 20, máximo 50; datas ISO UTC, valores ausentes `null`, listas vazias `[]` e preços em inteiros na unidade menor da moeda. Não usar número em ponto flutuante para armazenar dinheiro. Lista de lojas, fontes, tipos de canal e estados é enum/união discriminada, com refinamentos por adaptador.

Erros seguem o formato Cloud `{code,message?}`, sem ecoar corpo, URL secreta ou stack. Usar 400 para entrada inválida, 401 sem sessão, 403 sem direito/permissão, 404 para recurso alheio ou ausente, 409 para cota/conflito e 429 para excesso de chamadas com espera informada. Cota de posts e rate limit de tráfego têm códigos diferentes. Falha transitória de provedor não deve aparecer como identificador de afiliado inválido.

Webhooks de bots são exceções à sessão do painel: validar assinatura/secret e esquema do provedor antes de aceitar. Proposta `POST /api/deals/telegram/webhook` para updates Telegram; callback/interações Discord exigem os mecanismos oficiais próprios. Repetição do mesmo update/interação é idempotente. Credencial do bot não autoriza administrar a conta Cloud.

### Rotas públicas da F3

Proposta `GET/HEAD https://deals.getblup.site/r/:code` para redirect temporário ao destino salvo e validado. O caminho final ainda precisa ser fechado pelo dono da implementação. Não aceitar `?url=` como destino. Rate limit, cache e indisponibilidade do agregado não devem abrir redirect livre ou impedir a navegação normal quando o link continua válido. A vitrine é gerada pelo Studio, sem rota pública de vitrine no Deals.

Contagem significa acesso ao link, não pessoa única, compra ou comissão. Pré-busca, robôs e prévias de Telegram/Discord podem contar; HEAD deve seguir a política documentada de contagem. Agregação em memória de um isolate é aproximada e pode perder dados; se for usada, o painel precisa assumir essa limitação, sem chamar a métrica de auditável.

## 10. Telas e rotas do painel

| Rota | Tela e conteúdo |
| --- | --- |
| `/sign-in` | Login comum com lado coral; retorno fixo permitido |
| `/` | **Ofertas**: catálogo, busca, filtros, idade do preço, origem, prévia e colar link |
| `/hunts` | **Caçadas**: fontes, status, próxima rodada, últimos resultados e pausa |
| `/channels` | **Canais**: destinos confirmados, permissão do bot, teste e remoção |
| `/affiliates` | **Afiliados**: IDs por loja, chaves privadas, nível e instruções de cadastro |
| `/stats` | **Cliques**: estado vazio até F3; depois totais agregados, período e limites da medição |
| `/start` | Assistente inicial e progresso retomável |
| `/offers/new` | Colar link, completar campos, escolher destinos e confirmar prévia |
| `/hunts/new` | Criar caçada de jogos, loja ou espelho, conforme a fase entregue |
| `/hunts/[id]` | Fonte, filtros, janela, destinos, prévia, histórico e pausa/exclusão |
| `/channels/new` | Escolher Telegram/Discord, instalar bot e provar controle |

São cinco abas. Formulários auxiliares não criam uma navegação paralela de cobrança ou uma aba exclusiva para espelhos. No início, mostrar estados vazios úteis, sem números de exemplo que pareçam dados reais. Recursos de fases futuras ficam identificados como ainda indisponíveis, sem aceitar configuração que nunca será executada.

O SetupGuide mostra etapas reais, sem concluir uma etapa apenas por abrir a página. Sem direito Deals, o painel explica o adicional e leva aos planos Cloud. Idioma usa a preferência comum `pt/en/es`, com troca sem recarregar. Textos de configuração, mensagens de erro e prévias seguem o idioma do painel; textos originais de ofertas não são traduzidos automaticamente.

### Assistente `/start`

1. **Assunto do canal**: escolher jogos ou assunto livre, sem ativar uma fonte indisponível.
2. **Destino**: conectar Telegram/Discord, confirmar o canal e verificar que o bot pode publicar.
3. **IDs de afiliado**: informar por loja ou **Pular**. Pular deixa claro que o post não ganha comissão automaticamente.
4. **Prévia**: escolher uma oferta/link, revisar procedência, preço/condições, destino e identificação de publicidade.
5. **Ligar**: ação explícita que ativa a caçada ou confirma a publicação. Antes desse passo nenhum post de teste ou oferta é enviado sem confirmação própria.

O progresso deve sobreviver à volta da tela. Permitir revisar e desligar depois. Falha na conexão de um bot mantém o assistente no destino; não marcar publicação pronta com base apenas no retorno do OAuth ou no clique no link do Telegram.

## 11. Coleta, caçadas e entrega

Uma caçada associa fonte, filtros e destinos: assunto/palavras, loja/plataforma, região/moeda, preço máximo opcional, desconto mínimo quando calculável, horário e intervalo. Não permitir filtros que a fonte não oferece sem indicar que serão aplicados depois da coleta. Intervalos efetivos respeitam a cadência global e os limites externos; criar vinte caçadas não multiplica a frequência permitida da API.

Estados propostos da entrega: `queued`, `leased`, `sent`, `retry`, `cancelled`, `failed`, `uncertain`. Antes de enviar, conferir plano, canal, caçada/revisão, oferta ainda válida e reserva. Lease vencida precisa de recibo e tratamento de resultado incerto, sem afirmar entrega exatamente uma vez em Telegram/Discord. Guardar ID da mensagem retornada e horários, não o corpo integral da resposta externa.

Reaproveitar a estrutura de outbox e orçamentos dos transportes Pulse, criando payload específico de oferta. Telegram usa `sendMessage` ou `sendPhoto`, com legenda limitada e fallback para texto sem inventar sucesso. Discord usa mensagem/embed do bot autorizado, sem menções em massa. Imagem inválida não deve causar repetição infinita ou envio repetido do texto.

Respeitar 429, `Retry-After`, `retry_after` e os buckets aplicáveis de cada plataforma. O limite de 100 posts/dia não substitui limites por chat, bot, rota ou servidor. Tentativas usam espera persistida e limite de idade, sem `sleep` longo segurando o Worker. Escalonar dez destinos para não criar rajada a cada cron.

| Plataforma | Limites e permissões oficiais a respeitar |
| --- | --- |
| Telegram | Evitar mais de uma mensagem/s no mesmo chat; grupos, até 20/min; broadcast gratuito em torno de 30 mensagens/s por bot, compartilhadas entre contas |
| Telegram em canal | Bot adicionado com `can_post_messages`; texto até 4.096 caracteres ou legenda de foto até 1.024, depois do processamento das entidades |
| Discord HTTP do bot | Limite global normal de 50 requests/s, além dos buckets por rota/recurso; observar Bucket, Remaining, Reset-After e Retry-After/retry_after |

Não fixar uma regra universal de cinco envios a cada dois segundos para webhooks Discord. Webhook de entrada pode simplificar só a postagem F1, como alternativa técnica a fechar, com URL secreta e `wait=true`; não concede leitura para espelhos. O produto continua sem self-bot/token de usuário. [Telegram FAQ](https://core.telegram.org/bots/faq), [Bot API](https://core.telegram.org/bots/api), [rate limits Discord](https://docs.discord.com/developers/topics/rate-limits) e [webhooks Discord](https://docs.discord.com/developers/resources/webhook).

Manter paid broadcasts Telegram desligado: trata-se de recurso cobrado, não capacidade gratuita implícita do plano. A 10.000 contas consumindo 300 envios/dia cada, a média contínua já seria cerca de 34,7 envios/s; capacidade do bot e distribuição de horários precisam ser revistas antes dessa escala. Limites publicados não garantem entrega imediata.

## 12. Privacidade, permissões e dados sensíveis

Deals não tem cifra de ponta a ponta: o servidor precisa ler filtros, identificadores e ofertas para personalizar e publicar. Telegram, Discord e lojas recebem os campos exigidos por suas integrações. As notas do Studio não são fonte automática de ofertas, e o aceite de IA/Pulse não autoriza usar conteúdo das notas no Deals.

Credenciais de lojas, destinos de webhook e tokens temporários ficam cifrados no servidor com Web Crypto AES-GCM, nonce aleatório por gravação e AAD contendo dono, recurso e finalidade. Chave do serviço fica em secret, com versão para rotação; não usar o ID público de afiliado como chave. Nome proposto: `DEALS_CHANNEL_KEY`, sem valor neste documento. Decidir separação de chaves por finalidade antes da implementação. APIs de leitura retornam estado/rótulo, nunca o segredo gravado.

Bots usam credenciais do serviço, guardadas como secrets, e permissões mínimas. O cadastro precisa provar que a pessoa administra o destino; colar um ID de canal não basta. Essa prova vale para publicar no destino, sem exigir autorização do dono da origem espelhada. Para Telegram, confirmar permissões administrativas de publicação no canal. Para Discord, validar o servidor, canal e permissão do usuário e do bot, protegendo OAuth por state com prazo e uso único.

Tratar HTML, títulos, imagens e mensagens como conteúdo externo: escape de saída, limites, proibição de scripts e menções, nenhum HTML arbitrário no painel/vitrine. Logs usam códigos de erro e IDs internos, sem URLs com credenciais, cabeçalhos de autorização, conteúdo privado ou tokens de OAuth. Não coletar listas de membros nem mensagens privadas para montar uma oferta.

Cliques da F3 usam contagem diária agregada, sem perfil individual, fingerprint de navegador ou cookie de afiliado criado pelo Blup. IP pode ser observado pela infraestrutura para segurança; não o guardar no relatório comercial como identificador de visitante. Identificar claramente links de afiliados e a procedência, conforme as regras de cada programa.

## 13. Switch e cobrança Stripe

**Somente sandbox nesta rodada.** O switch Deals acrescenta `ultra_deals` ou `ultra_pulse_deals` ao catálogo existente. Proposta: uma assinatura e um item recorrente que representam a combinação, como o contrato atual do Pulse. Não abrir uma segunda assinatura para o mesmo adicional.

| Nova variante proposta | Lookup key mensal proposta | Lookup key anual proposta |
| --- | --- | --- |
| Ultra + Deals | `blup_ultra_deals_mensal` | `blup_ultra_deals_anual` |
| Ultra + Pulse + Deals | `blup_ultra_pulse_deals_mensal` | `blup_ultra_pulse_deals_anual` |

Com as cinco variantes pagas existentes, são sete variantes pagas e quatorze lookup keys no total, se essa organização for mantida. Valores/moedas vêm do Price Stripe validado; período sem preço válido permanece indisponível para compra.

### Preços propostos, ainda sem aprovação do dono

| Variante | Mensal proposto | Anual proposto, cobrado uma vez |
| --- | --- | --- |
| Ultra + Deals | R$ 109,90 | R$ 1.099,00 |
| Ultra + Pulse + Deals | R$ 169,90 | R$ 1.699,00 |

As bases informadas pelo Maestro são Ultra a R$ 59,90/mês e Ultra + Pulse a R$ 129,90/mês. O acréscimo proposto do Deals é R$ 50,00/mês sobre Ultra ou R$ 40,00/mês sobre Ultra + Pulse. Anual equivale a dez mensalidades: bases R$ 599,00 e R$ 1.299,00, com acréscimos de R$ 500,00 e R$ 400,00. Esses valores são proposta para avaliação e ensaio sandbox, sem autorização para cobrar em produção.

Proposta de `/me.plan`: preservar `name:"ultra"`, manter `billingName` da variante completa e acrescentar `deals:{enabled,limits}` sem inferir direito só pelo prefixo do nome. `/plans` precisa representar Pulse e Deals de forma independente. Confirmar o shape com a Lupa antes de criar consumidores ou afirmar contrato final.

Estender as rotas próprias `/api/billing/checkout`, `/preview`, `/confirm` e `/state`, sem criar cobrança mutável paralela no plugin Better Auth. Adicionar Deals usa prévia e confirmação com proporcionalidade; retirar Deals ou reduzir o plano deve seguir o fim pago, com efeito e retenção explícitos. O webhook assinado e a reconciliação do estado atual continuam concedendo direitos; página de sucesso não concede acesso. A [spec Pulse](blup-pulse.md#11-switch-e-cobrança-stripe) descreve o mecanismo existente, que ainda precisa ser estendido e testado para as novas combinações.

Não modificar Prices de clientes existentes em massa. Criar Prices sandbox próprios e metadata da variante completa; testar troca entre as quatro combinações Ultra, pagamento pendente, retirada de só um adicional, cancelamento, retorno e webhook repetido/fora de ordem. Produção depende de preço aprovado, catálogo real validado e liberação do dono.

## 14. Custos e viabilidade

Estimativa da pesquisa da Lupa, com aritmética conferida para 30 dias. É um orçamento do **adicional Deals**, sem o consumo do Ultra/Cloud/Pulse. Nenhum preço de venda do Deals foi aprovado na nota; portanto não há margem comercial fechada. Tarifas em USD, sem impostos ou câmbio, sujeitas a nova conferência antes de vender.

### Tarifas oficiais

Workers Paid/Standard tem base de US$ 5/mês por conta, com 10 milhões de requests e 30 milhões de CPU-ms incluídos. Excedentes: US$ 0,30/milhão de requests e US$ 0,02/milhão de CPU-ms. Fetch de saída não é request de entrada cobrado por unidade, mas usa CPU/limites de execução. [Workers pricing](https://developers.cloudflare.com/workers/platform/pricing/).

D1 Paid inclui 25 bilhões de linhas lidas/mês, 50 milhões escritas/mês e 5 GB totais. Excedentes: US$ 0,001/milhão lido, US$ 1/milhão escrito e US$ 0,75/GB-mês. Índices, atualizações e exclusões também geram escrita; leitura cobra o que a consulta varre, não só a quantidade devolvida. Medir `meta.rows_read` e `meta.rows_written`. Franquias aplicam-se **uma vez à conta**, compartilhadas com os outros produtos. [D1 pricing](https://developers.cloudflare.com/d1/platform/pricing/).

### Cenário por cliente

O limite de posts passou a 100 por dia por decisão do dono. O cenário abaixo foi calculado com 300 envios por dia e fica mantido como teto conservador, três vezes o limite atual: o custo real por cliente tende a ser menor, e os saldos da proposta de preço, maiores.

| Premissa de cálculo, não cota contratada | Quantidade em 30 dias |
| --- | --- |
| Destinos/caçadas/espelhos | 10 / 20 / 10 |
| Publicações somadas entre destinos | 300/dia, ou 9.000/mês |
| Registros de envio na retenção de 90 dias | Cerca de 27.000 |
| Avaliação de 20 caçadas a cada cinco minutos | 172.800 avaliações |
| Dez espelhos distintos consultados a cada cinco minutos | Até 86.400 leituras de fonte |
| Cliques de exemplo, sem limite aprovado | 3.000/dia, ou 90.000/mês |
| Envelope de requests de entrada, com folga operacional | 150.000 |
| CPU estimada | 3 milhões de ms |
| Linhas D1 lidas/escritas, incluindo índices e manutenção | 10 milhões / 1 milhão |
| Armazenamento D1 em regime estável | 0,25 GB |

Cinco minutos é hipótese de cálculo; não estabelece SLA nem licença de coleta. Uma fonte compartilhada é lida uma vez por rodada, não por seguidor. GamerPower/CheapShark são custos globais conforme uso permitido. Credenciais de loja privadas não podem ser compartilhadas entre contas apenas para reduzir custo. Imagens não passam por R2 na F1.

| Parcela marginal, depois de esgotar as franquias | Conta ilustrativa |
| --- | --- |
| Requests | 0,15 milhão × US$ 0,30 = US$ 0,045 |
| CPU | 3 milhões de ms × US$ 0,02/milhão = US$ 0,060 |
| D1 leituras | 10 milhões × US$ 0,001/milhão = US$ 0,010 |
| D1 escritas | 1 milhão × US$ 1/milhão = US$ 1,000 |
| D1 armazenamento | 0,25 GB × US$ 0,75 = US$ 0,1875 |
| **Total marginal do cenário** | **US$ 1,3025 por cliente/mês** |

Esse total não é benchmark ou custo máximo. Reserva de orçamento inicial: cerca de US$ 1,30 por cliente, mais rateio da base da conta e os custos excluídos. Amplificação de escrita, catálogo retido, cliques e tentativas precisam de medição no protótipo.

### Franquias e capacidade são coisas diferentes

| Clientes, apenas esse workload hipotético | Total estimado da conta com base/franquias uma vez | Média |
| --- | --- | --- |
| 1 | US$ 5,00/mês | US$ 5,00 |
| 100 | US$ 76,90/mês | US$ 0,769 |
| 1.000 | US$ 1.240,15/mês | US$ 1,24015 |

Essas simulações supõem as franquias ainda livres para esse workload; Cloud/Pulse podem já tê-las consumido. Também supõem capacidade distribuída onde necessário. A 0,25 GB por cliente, 100 clientes representam 25 GB e 1.000 representam 250 GB. **Não cabem em um D1 de 10 GB.** O custo baixo não remove o teto físico por banco. No início, o D1 compartilhado precisa de alertas antes do limite e poda/compactação; crescimento exige distribuição ou outra organização de dados, a validar antes dessa escala. [Limites D1](https://developers.cloudflare.com/d1/platform/limits/).

### Sensibilidade, cliques e importação Studio

Se 300 significasse ofertas replicadas em dez canais, seriam 90.000 envios/mês, dez vezes o cenário. Os 81.000 extras, com dez linhas escritas por envio como hipótese, acrescentariam US$ 0,81 de escrita, além de CPU, leitura e armazenamento. A viabilidade proposta pressupõe **300 envios totais**.

Cada 1.000 cliques/dia adicionais gera 30.000 requests/mês: US$ 0,009 marginal em requests e US$ 0,03 em escrita se houver um UPDATE cobrado por clique, fora CPU/índices. Agregar por dia reduz registros, mas não elimina escritas se cada clique atualiza o contador. Não anunciar poucas escritas sem um mecanismo persistente de agrupamento medido; memória de isolate pode perder incrementos.

A conexão Studio é separada do redirect. Exemplo: uma importação diária com dez chamadas soma 300 requests/mês, equivalentes a US$ 0,00009 de requests excedentes; 300 ofertas lidas por dia somam 9.000 linhas/mês, ou US$ 0,000009 de leitura, antes de índices, CPU e consultas auxiliares. São exemplos de escala, não contrato da futura exportação. Build, domínio e hospedagem da vitrine pertencem ao destino de publicação escolhido no Studio.

### Outbox e custos excluídos

Manter outbox D1 na implementação inicial. Queues é opção futura se atraso, concorrência ou contrapressão justificarem: Paid inclui 1 milhão de operações e cobra US$ 0,40/milhão excedente. Uma mensagem pequena normalmente usa gravar/ler/apagar; 9.000 entregas sem repetição representam 27.000 operações, ou US$ 0,0108 marginal. Não incluir Queues no custo inicial como se já estivesse escolhido. Sua entrega é pelo menos uma vez, e D1 + Queue não se tornam uma gravação atômica; recibos e dedupe continuam necessários. [Preço Queues](https://developers.cloudflare.com/queues/platform/pricing/), [garantias](https://developers.cloudflare.com/queues/reference/delivery-guarantees/).

Ficam fora da estimativa Cloudflare: direitos já usados do Ultra, APIs comerciais, Stripe, impostos, suporte, IA, logs excessivos e proxy de mídia. Deals não acrescenta armazenamento R2 na F1, mas o Ultra continua tendo custo próprio de notas/sites/operações. O cálculo incremental abaixo não trata US$ 1,3025 como custo total de Ultra + Deals.

### Saldo incremental dos preços propostos

Usar a referência PTAX informada pelo Maestro, **R$ 5,4212 por US$ 1**, sem apresentá-la como cotação do dia: `US$ 1,3025 × 5,4212 = R$ 7,061113`, arredondado a **R$ 7,06/mês**. Com custo Cloudflare em dobro, são R$ 14,12/mês. O cálculo mantém a precisão completa e arredonda apenas os resultados exibidos.

A simulação aplica **4,69% sobre o acréscimo** da assinatura: 3,99% de Payments para cartão nacional mais 0,7% de Billing. A tarifa fixa de R$ 0,39 por transação já existe na cobrança do plano-base e não é somada outra vez ao acréscimo, considerando uma única assinatura/cobrança recorrente. Faturas extras de proporcionalidade podem ter custo próprio. [Tarifas Stripe Brasil](https://stripe.com/br/pricing).

| Acréscimo mensal Deals | Receita adicional | Stripe incremental, 4,69% | Cloudflare | Saldo | Saldo com Cloudflare em dobro |
| --- | --- | --- | --- | --- | --- |
| Sobre Ultra | R$ 50,00 | R$ 2,35 | R$ 7,06 | **R$ 40,59** | **R$ 33,53** |
| Sobre Ultra + Pulse | R$ 40,00 | R$ 1,88 | R$ 7,06 | **R$ 31,06** | **R$ 24,00** |

O anual cobra dez mensalidades, mas presta serviço por doze meses. Por isso o custo anual usa `12 × R$ 7,061113 = R$ 84,733356`; em dobro, R$ 169,47 após arredondamento.

| Acréscimo anual Deals | Receita adicional | Stripe incremental, 4,69% | Cloudflare por 12 meses | Saldo | Saldo com Cloudflare em dobro |
| --- | --- | --- | --- | --- | --- |
| Sobre Ultra | R$ 500,00 | R$ 23,45 | R$ 84,73 | **R$ 391,82** | **R$ 307,08** |
| Sobre Ultra + Pulse | R$ 400,00 | R$ 18,76 | R$ 84,73 | **R$ 296,51** | **R$ 211,77** |

O saldo incremental é positivo nos quatro cenários, inclusive com o dobro do custo Cloudflare. É uma simulação da contribuição adicional do Deals antes dos demais custos excluídos, sem afirmar lucro total do plano ou aprovar os preços.

## 15. Riscos, fases e critérios de aceite

| Fase aprovada | Escopo | Critério de conclusão proposto |
| --- | --- | --- |
| **F0** | Switch Ultra + Deals, variantes sandbox, app com login, shell, cinco abas/estados vazios, sessão/direito e schema inicial D1 | Login/logout e origem exata; sem acesso de outro dono/plano; nenhum post/coletor real; migração local do zero; combinações de plano tipadas e compra sandbox sem produção |
| **F1** | Destinos Telegram/Discord, IDs de afiliados, colar link e jogos automáticos | Prova de controle do destino; prévia fiel; regras de API por fonte e links CheapShark preservados; dedupe/cota atômicos; 429/retry/resultado incerto; dados externos validados; sem promessa de comissão não comprovada |
| **F2** | Caçadas Shopee/AliExpress/Mercado Livre e espelhamento aberto | Credencial individual e regra por loja; coleta compartilhada; sem autorização do dono da origem; limites técnicos, intervalo e risco dos termos registrados; leitura incremental; revisão/pausa/concorrência; arquitetura Discord definida |
| **F3** | Cliques/relatórios, conexão Deals com o Studio para gerar a vitrine e Amazon automática | Link curto sem open redirect; exportação autenticada só do dono e sem segredos; publicação da vitrine pelo Studio; agregação explicada; elegibilidade/API Amazon e regras comerciais verificadas |

Aceitar uma fase por testes normais, sem `it.fails`. Cobrir schema estrito, respostas externas ausentes/nulas, dono errado, código de vínculo repetido/expirado, assinatura removida entre fila e envio, mesma oferta em duas fontes, lease vencida durante fetch/envio, virada UTC e limite conjunto da conta. Injetar sentinelas de segredos para verificar ausência em logs, DTOs públicos e mensagens.

Gates da área Cloud quando houver implementação: TypeScript do Worker, Vitest, check e build de todos os painéis afetados, migrações D1 locais vazias e upgrade de banco anterior. Consultar MCP Svelte e rodar autofixer em cada `.svelte` alterado. Esta tarefa altera apenas Markdown e valida links, travessões, padrões de segredo e whitespace; não atribui esses gates de código ao Deals.

Riscos prioritários: disponibilidade/licença das fontes; atribuição de afiliado não comprovada; rajadas ou duplicação de posts; coleta de URL maliciosa; vazamento de credenciais; custo crescente de polling por usuário; ausência de dados de preço suficientes; bot sem permissão; e queda do provedor entre aceite e recibo. Cada fonte pode ficar desligada sem impedir colar link ou usar outra fonte disponível.

## 16. Decisões aprovadas e pendências

### Aprovado na nota

- Produto para afiliados iniciantes, com Telegram e Discord oficiais, sem WhatsApp na F1.
- Adicional exclusivo Ultra; combinações Ultra, Ultra + Pulse, Ultra + Deals e Ultra + Pulse + Deals.
- Espelhamento aberto como fonte dentro de Caçadas, sem exigir autorização do dono da origem; risco dos termos Telegram/Discord registrado e assumido pelo dono do Blup, preservando os limites técnicos.
- CheapShark mantida como fonte, com os links da própria CheapShark e sem troca pelo afiliado do usuário; risco de coleta excessiva registrado.
- Blup `turma-8`, coral `#ff7a59`, destaque do painel `#c2410c`.
- Painel `deals.getblup.io`; links curtos F3 em `deals.getblup.site`; vitrine gerada no Studio pela conexão Deals, sem vitrine externa no Deals.
- Worker/API Cloud, painel estático próprio, sessão comum, D1, coleta compartilhada e outbox; sem DO/R2 na F1.
- Cinco abas, rotas em inglês, assistente assunto, destino, IDs opcionais, prévia e ligar.
- Entregas em F0 a F3; Stripe apenas sandbox nesta rodada.

### Falta fechar

- Confirmar os limites à luz da pesquisa: 10 canais, 20 caçadas, 10 espelhos, **100 envios somados por dia** e histórico de 90 dias; medir CPU, escrita e armazenamento do protótipo. A estimativa marginal de US$ 1,3025 não fecha a capacidade de um banco D1.
- Aprovar ou revisar a proposta de preços: Ultra + Deals R$ 109,90/mês ou R$ 1.099/ano; Ultra + Pulse + Deals R$ 169,90/mês ou R$ 1.699/ano. Regras de cortesia e futura liberação comercial também continuam pendentes.
- Contrato oficial de conversão de afiliado Mercado Livre; elegibilidade e permissão de cada loja para publicar em Telegram/Discord. O simples `matt_word` permanece não confirmado.
- Volume/cadência de coleta CheapShark e demais fontes; a permanência da CheapShark com seus próprios links já foi decidida.
- Método de leitura Discord e eventual infraestrutura da F2; planejar revisão de intents no alcance de 10.000 usuários únicos e verificação para mais de 100 servidores, sem confundir os critérios. Não incluir Gateway persistente silenciosamente no custo F1.
- Prazo e purga ao retirar Deals; efeito sobre vitrine/links e recibos de cota, sem importar automaticamente a carência Pulse.
- Semântica final de reserva/contagem de post em resultado incerto, teto/TTL do catálogo e intervalo mínimo das caçadas.
- Caminho de links curtos, política de contagem HEAD/robôs e limite de cliques; contrato da conexão Deals e Studio, template, atualização e destino da vitrine.
- Nomes finais dos bots, credenciais de serviço, escopos mínimos, rotação de segredos e ensaios externos autorizados.

Essas pendências técnicas e comerciais não reabrem a decisão de espelhamento aberto ou a permanência da CheapShark com seus próprios links. O risco assumido pelo dono continua registrado sem ser apresentado como autorização das plataformas.
