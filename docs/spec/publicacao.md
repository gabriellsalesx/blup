# Publicação e domínio próprio

Fontes: [fluxo GitHub](../../internal/app/publish.go), [fluxo Cloudflare](../../internal/app/cloudflare.go), [Blup Cloud](../../internal/app/blup_publish.go), [APIs do Studio](../../internal/app/deploy_api.go), `internal/blupcloud/sites.go`, `internal/deploy/github/`, `internal/deploy/cfworkers/` e `internal/app/domain_blup.go`.

Domínio próprio existe só no Blup Cloud (Max e Ultra). Sites no Cloudflare Workers ou no GitHub Pages ficam no endereço gratuito do provedor; o arquivo do site não tem campo `domain`, e o Studio e a CLI não têm o item Personalizar > Domínio próprio.

## Destino e contas

A publicação sempre gera o site localmente e envia a saída estática. O destino é salvo em `Site.Deploy`; a sincronização de notas usa outro campo e outro repositório. Cloudflare recebe uploads por HTTP e GitHub recebe blobs e árvores pela API, sem `wrangler`, Node ou `git` no caminho de publicação.

Na CLI, `publish` pergunta o provedor se ainda estiver vazio, obtém credenciais, prepara o destino, gera, envia e salva o site. Depois oferece abrir a URL. No Studio, o formulário conecta contas e salva o destino antes de `PublishNow`; esse método exige destino e credenciais já definidos, mas usa o mesmo build.

`LoadConnections` verifica o usuário GitHub com a chave salva, devolvendo `githubError` quando falha; `Connections` é a estrutura de resposta. Para Cloudflare, a consulta de conexões informa a existência de token; `ConnectCloudflare` faz a verificação e lista as contas antes de salvar. `Disconnect` remove a chave local de GitHub, Cloudflare ou OpenRouter, sem revogar a chave no serviço.

`PublishNow` obtém clientes por `savedGitHub` ou `savedCloudflare`, que devolvem erro quando falta chave, sem oferecer formulários de conexão. Já a exclusão ainda chama os clientes interativos da CLI, conforme descrito adiante. O build compartilhado também mantém suas próprias condições de senha e feedback, descritas em [CLI](cli.md).

## Blup Cloud

Destino `blup` (planos pagos; o Grátis tem `sites: 0`). `Site.Deploy.Blup` guarda `id`, `slug`, `url` e `base` da última publicação: o endereço `<slug>.getblup.site` é escolhido antes da primeira publicação (sugestão pelo título, `blupSlug`) e fica fixo depois. `deployBlup` cria o site na nuvem na primeira vez (`ErrSlugTaken`, `ErrSitesLimit`, `ErrPlan`) e salva imediatamente a identidade, inclusive se o upload posterior falhar. Escolhe a base por `blupBase` e gera com o mesmo `build` (Hugo com `--minify --gc`) para o domínio próprio elegível ou o endereço `getblup.site`, empacota com `blupcloud.PackSite` (arquivos de texto em gzip no nível máximo quando ficam menores; o hash SHA-256 é dos bytes guardados) e envia com `DeploySite`: pergunta quais faltam, manda só esses (seis por vez) e publica a lista de caminhos. Se a nuvem disser que faltam arquivos, manda de novo uma vez. Max e Ultra também permitem [domínio próprio](#domínio-próprio-no-blup-cloud), com vínculo independente do arquivo de configuração local. O serviço está em [Blup Cloud](../../cloud/README.md); o provisionamento está em [deploy](../deploy.md).

Na tela Publicar do Studio, Blup Cloud é a primeira opção: sem conta, entra (`CloudAccount` com `notes={false}`, sem a senha das notas); no Grátis, explica que vem nos planos pagos e abre os planos; senão, pede o endereço e mostra quantos sites do plano estão em uso. O campo aceita só minúsculas, números e hífens (acentos saem, o resto vira hífen) e, 400 ms depois de digitar, pergunta à nuvem (`CheckSiteSlug`): disponível, já é deste site, em uso (com até três nomes parecidos livres para tocar), reservado ou inválido. Publicar só fica liberado com o endereço livre. A tela Publicar só trata do site; Sincronizar abre a mesma tela no modo das notas (`autostart` `sync`). O domínio próprio do Blup Cloud tem painel `BlupDomain.svelte`, depois da primeira publicação; Cloudflare e GitHub não têm painel de domínio, e os passos para conectar essas contas só aparecem ao tocar em Conectar a conta.

### API e armazenamento dos sites

[sites.ts](../../cloud/worker/sites.ts) autentica e verifica o dono por site. `POST /api/sites` cria; `GET /api/sites/check` consulta o nome; `POST /api/sites/:id/missing` aceita até 1.000 hashes; `PUT /api/sites/:id/files/:hash` confere SHA-256 e cota; `POST /api/sites/:id/deploys` ativa o manifesto; `DELETE /api/sites/:id` retira o registro e agenda apagar o prefixo R2. O slug tem de 3 a 40 caracteres, sem hífen nas pontas ou `--`, com nomes reservados em `site-rules.ts`. O cliente sugere nomes, mas a decisão é do servidor.

No bucket `SITES`, `s/<id>/f/<hash>` guarda o arquivo e `s/<id>/d/<deploy>` o manifesto `caminho -> [hash, Content-Type, gzip]`. Um deploy aceita de 1 a 20.000 caminhos únicos; recusa segmentos `.`/`..`, barras invertidas, controles e arquivos especiais de Pages. Notas, histórico e sites dividem armazenamento. Arquivo máximo por plano: 10/25/50/100 MiB para Grátis/Pro/Max/Ultra. O Grátis não tem publicação de sites; Pro permite 1 site, Max 3 e Ultra 10.

[upload.ts](../../cloud/worker/upload.ts) verifica o corpo recebido, SHA-256 e reserva de armazenamento antes de confirmar o arquivo. Com tamanho conhecido, usa staging R2 e `crypto.DigestStream`, evitando guardar todo arquivo grande em memória; confere o tamanho real e tem prazo de 120 segundos. Sem `Content-Length`, limita o buffer a 4 MiB ou ao teto menor do plano. Objetos temporários não entram no manifesto ativo.

Uma lease D1 por site serializa publicação e limpeza. Ativar um manifesto exige ainda possuir a lease; perder a disputa retorna 409 `DEPLOY_BUSY`. O deploy anterior vai para `retired_deploy` e fica protegido por pelo menos dez minutos. [sites-cleanup.ts](../../cloud/worker/sites-cleanup.ts) preserva referências do manifesto ativo e dos anteriores ainda protegidos, e só remove blobs antigos sem referência. A limpeza trabalha em lotes, valida manifestos e adia remoção quando não pode reunir o conjunto seguro. Isso corrige a limpeza concorrente que podia apagar arquivos da publicação ativa. Os manifestos anteriores são tolerância operacional, sem interface de histórico ou rollback garantido.

### Domínio próprio no Blup Cloud

Fontes: [domínios no servidor](../../cloud/worker/domains.ts), [provedor](../../cloud/worker/domain-provider.ts), [cliente Go](../../internal/blupcloud/domain.go), [coordenação](../../internal/app/domain_blup.go) e [tela](../../studio/frontend/src/BlupDomain.svelte).

Max e Ultra permitem um hostname por site já criado. A API autentica o dono, normaliza o hostname e recusa esquema, caminho, porta, IP e nomes reservados. A normalização aceita IDNA. `POST /api/sites/:id/domain` aceita estritamente `{hostname}`; trocar para outro hostname exige remover o anterior. Vínculos verificados têm unicidade no D1.

| Operação | Resultado |
| --- | --- |
| `GET /api/sites/:id/domain` | `DomainInfo` ou JSON `null` quando não há vínculo |
| `POST /api/sites/:id/domain` | Cria ou verifica novamente o mesmo vínculo; exige Max/Ultra |
| `DELETE /api/sites/:id/domain` | Remove no provedor e no D1; retorna `{ok: true}`; permitido após downgrade |

`DomainInfo` tem `hostname`, `status` (`pending_ownership`, `pending`, `active`, `error`, `disabled`), `disableAt` (instante ISO UTC ou `null`), `ssl` (`pending`, `active`, `error`) e `records`, até dez registros `{type: TXT|CNAME, name, value}`, com valor de até 1.024 caracteres. O servidor gera prova TXT aleatória por vínculo em `_blup.<hostname>`. Só após resolver essa prova exata por DNS over HTTPS provisiona o Custom Hostname Cloudflare. O CNAME aponta ao destino definido pelo operador; registros adicionais de validação SSL também aparecem na resposta. O domínio só é servido quando propriedade, hostname e SSL estão ativos, e há direito atual ou carência válida. O gateway confere host exato e prazo no D1 mesmo que a resolução de domínio esteja em cache.

O Studio usa `BlupDomain`, `SetBlupDomain` e `RemoveBlupDomain`, mostrando registros, estados, orientação para raiz com CNAME flattening/ALIAS e botão de verificar. Enquanto o painel está aberto e o domínio não está ativo, consulta a cada minuto; o servidor limita verificações externas a intervalos de 30 segundos. A pessoa altera o DNS no seu provedor. A infraestrutura SaaS exigida está em [deploy](../deploy.md#7-dns-tls-e-separação-dos-sites).

O vínculo fica no servidor. `SetBlupDomain`/`RemoveBlupDomain` não gravam o hostname no arquivo do site, mas o endereço oficial do build agora acompanha seu estado. `DomainInfo.Official` exige `status = active`, `ssl = active` e plano com domínio ou `disableAt` ainda futuro. `officialBase` usa `https://<hostname>/`; sem elegibilidade usa `Deploy.Blup.URL`. `Deploy.Blup.Base` registra a base do último publish. Em falha temporária de consulta cloud, `blupBase` conserva essa base em vez de alternar por indisponibilidade.

No Studio, abrir um site publicado no Blup Cloud ou consultar/adicionar/remover seu domínio pode detectar `Republish`. A republicação ocorre em segundo plano pelo mesmo mutex de publicação/sync, no máximo uma vez a cada dez minutos por site. Operação já ocupada permite nova tentativa na próxima consulta; falha mostra aviso e tenta de novo numa consulta posterior, respeitando o intervalo. O evento tipado `republish` informa `start`, `done` com URL ou `error`. Não existe serviço de republicação com o Studio fechado.

O novo build atualiza canonical, `og:url`, RSS, sitemap e referência do sitemap em robots. Os dois hosts continuam servindo o mesmo deploy, sem redirecionamento entre `getblup.site` e domínio próprio; os links absolutos refletem a base da última publicação concluída. Antes de republicar, uma saída antiga pode conservar seus links anteriores. Normalização de HTTPS/`www` e redirecionamento de pasta para barra final são comportamentos separados.

Ao passar de Max/Ultra para Pro ou Grátis, o domínio existente ganha 14 dias, com `disableAt` exposto no GET. Voltar ao direito de domínio antes do desligamento limpa o prazo. Ao vencer, GET informa `disabled`, SSL `pending`, registros vazios e preserva hostname/data; o gateway já recusa entrega, e o cron remove o Custom Hostname. Após desativação, voltar ao plano elegível exige novo POST de reconfiguração, que reinicia a prova. O endereço `getblup.site` só continua nos sites que cabem no plano após a carência.

Ao remover vínculo, excluir site ou purgar conta, a API remove o Custom Hostname antes de apagar os identificadores D1. Falha externa conserva o necessário para repetir. No desligamento por prazo, `UPDATE ... RETURNING` ganha o claim antes do DELETE externo; se um upgrade já limpou o prazo, não remove o hostname. GET/DELETE continuam disponíveis depois do downgrade.

### Worker público e cache

[sites-entry.ts](../../cloud/worker/sites-entry.ts) atende `<slug>.getblup.site` e domínios próprios verificados/ativos, separados da conta. [serve-site.ts](../../cloud/worker/serve-site.ts) permite GET/HEAD, resolve `index.html`, redireciona pastas para a barra final preservando query e usa `/404.html` com status 404 quando disponível. O código normaliza HTTP e `www.<slug>` para HTTPS sem `www` quando a requisição chega ao Worker; a cobertura TLS desse segundo nível depende da infraestrutura, conforme [deploy](../deploy.md#7-dns-tls-e-separação-dos-sites).

A entrada default do Worker não tem cache. Em cada pedido, o gateway consulta D1 para conferir usuário existente, bloqueio/exclusão, deploy ativo, direito de quantidade de sites, hostname exato e carência. Após 14 dias acima da quantidade do novo plano, aplica `retainedSiteIds`: Pro mantém o escolhido ou o mais antigo, Grátis mantém zero. `GET /api/sites` expõe prazo e escolha; `PUT /api/sites/retained` salva somente site do dono durante o prazo. Arquivos dos demais sites são preservados. O [contrato de mudança de plano](blup-cloud-sync.md#exclusão-retenção-e-mudança-de-plano) é separado da purga de conta após 30 dias.

### Workers Cache, chaves e purga

`wrangler.sites.jsonc` desliga cache da entrada default e liga só o entrypoint `CachedSite`. Depois da autorização, `ctx.exports.CachedSite({props: {siteId, deploy}}).fetch(...)` atende o conteúdo público imutável. A Cache API anterior foi removida. `ctx.props` separa site/deploy; o gateway retira query, cookies, Authorization, Range e headers arbitrários, e normaliza gzip/identity com `Vary: Accept-Encoding`. A chave gerenciada não depende de hostname, por isso o isolamento por props é obrigatório. O cache de manifestos em memória também inclui site e deploy e tem até 32 entradas. Referência: [chaves Workers Cache](https://developers.cloudflare.com/workers/cache/cache-keys/).

A resposta interna usa 600 segundos e tags `site:<id>`/`deploy:<id>:<deploy>`. O gateway restaura a política de navegador: HTML, XML/feed, JSON/manifest e `sw.js` revalidam, fingerprint fica imutável por um ano e outros assets por uma hora. Preserva ETag/304, HEAD sem corpo e a query dos redirecionamentos de barra final. Resposta inexistente ou bloqueada usa `no-store`. O mapa de resolução de domínio ainda tem até 30 segundos/1.024 entradas, mas a autorização confere no D1 se o hostname exato continua vinculado e ativo; esse mapa não libera domínio removido.

Bloqueio/desbloqueio, exclusão/restauração, remoção de site, vencimento de carência e desligamento de domínio enfileiram purga persistente por site. O cron chama o service binding `SITE_CACHE`, que executa `CachedSite.purgePending` no contexto dono do cache. Agrupa até cem tags e limita a uma tentativa por minuto; o cron atual roda a cada cinco minutos. Falhas preservam a fila, e uma geração nova impede uma purga antiga de apagar um evento concorrente. O gateway bloqueia entrega revogada enquanto a purga aguarda. Escopo e limites seguem a [purga Workers Cache](https://developers.cloudflare.com/workers/cache/purge/).

HTML pode incluir Link preload do CSS reconhecido e Inter presentes naquele manifesto. O provisionamento pede `ssl.settings.early_hints = "on"`, e a zona precisa de Early Hints habilitado. A presença do header não comprova emissão de 103; depende da camada Cloudflare e do navegador, conforme [Early Hints](https://developers.cloudflare.com/workers/examples/103-early-hints/).

Hit do cache evita executar o serving interno e reduz leituras R2 esperadas, mas o gateway/D1 continuam por acesso. Requests, inclusive o loopback, continuam sujeitos à cobrança; medir hits, CPU, D1 e R2 antes de afirmar economia total. Os [limites do cache](https://developers.cloudflare.com/workers/cache/limitations/) e a configuração de [produção](../deploy.md) fazem parte da operação. Nenhuma purga ou política do servidor apaga cópias já baixadas pelo navegador/PWA.

## Domínio das páginas Blup Pulse

A Fase 4 local acrescenta `GET/POST/DELETE /api/pulse/pages/:page/domain`, com homologação de infraestrutura pendente e sem confirmação de deploy/venda. A página Pulse não usa build Hugo nem consome um site desse plano. POST aceita só `{hostname}` e exige Ultra + Pulse; GET retorna `DomainInfo` ou `null` e DELETE `{ok:true}`. GET segue disponível após downgrade enquanto houver leitura privada e DELETE permanece disponível ao dono.

O fluxo exige TXT aleatório em `_blup.<hostname>` antes de chamar Cloudflare. `hostname_claim` reivindica o mesmo namespace atômico dos sites Hugo, evitando adotar um hostname para os dois serviços. A entrega exige prova, status/SSL ativos, publicação e página/direito elegíveis. O domínio serve `/` e `/status.json`; o endereço padrão é `pulse.getblup.site/<slug>/`. Na perda do direito Pulse, o domínio para de servir sem a carência adicional dos domínios Hugo. A exclusão externa preserva `provider_id` até concluir e é repetida em falha. Reativar exige novo POST/prova. Os [contratos Pulse](blup-pulse.md#7-monitores-http-e-páginas-de-status) distinguem essa regra dos sites. Fontes: [domínios Pulse](../../cloud/worker/pulse/domains.ts), [retirada](../../cloud/worker/pulse/domain-lifecycle.ts), [namespace](../../cloud/worker/hostname-claim.ts) e [Worker público](../../cloud/worker/sites-entry.ts).

## Autenticação GitHub

`githubClient` usa `credentials.toml`, verifica `GET /user` e orienta a criação de token pessoal quando necessário. O link abre o formulário fine-grained com Contents, Pages e Issues em escrita, com Metadata em leitura, sem Administration, e duração sugerida de 366 dias. A interface orienta o escopo de repositórios. O cliente usa a API versionada em `2026-03-10` e cabeçalhos de autenticação implementados em `internal/deploy/github/client.go`.

A conexão GitHub da CLI e do Studio usa token pessoal informado pela pessoa. O Device Flow sem consumidor e o arquivo `internal/deploy/github/auth.go` foram removidos na rodada 4. Isso não remove a autorização de aparelho do Blup Cloud, que é outro fluxo, em `internal/blupcloud` e Better Auth.

O cliente HTTP classifica erros de token, permissão, conflitos, rate limit, arquivo grande e Pages. Possui repetição, controle de intervalos entre escritas e leitura do atraso de rate limit. Os testes substituem endpoints HTTP por servidores falsos.

## GitHub Pages

`deployGitHub` confirma o repositório de saída, atribuindo o proprietário ao usuário autenticado. `EnsureRepoWith` consulta o repositório e ainda tenta `POST /user/repos` quando ausente. Se a credencial permite, conserva a criação automática; se essa tentativa recebe erro de permissão ou ausência, devolve `ErrRepoMissing` com link de `github.com/new`. Com o token fine-grained restrito orientado pela interface, a pessoa cria o repositório e o inclui na seleção de acesso da chave. Um existente pode ser usado se a chave tiver permissão. Não se presume que mudar o campo owner publique automaticamente em uma organização.

O endereço esperado é `https://<owner>.github.io/<repo>/`, exceto quando o repositório se chama `<owner>.github.io`, que usa a raiz. O build sempre usa esse endereço; o Blup não configura domínio próprio no GitHub Pages.

`github.Publish` em `internal/deploy/github/publish.go`:

1. Percorre arquivos regulares da saída e calcula SHA-1 de blob Git. Ignora `.git` e arquivos não regulares; rejeita arquivo acima de 100 MiB.
2. Acrescenta `.nojekyll` virtual.
3. Compara com a árvore remota da branch `gh-pages`. Evita reenviar conteúdo existente, reutiliza blobs e envia o que mudou.
4. Monta a árvore espelhando a saída: arquivos ausentes são retirados da branch de publicação.
5. Cria a branch órfã se necessário e atualiza a referência conforme a estratégia de publicação, que permite atualização forçada. Isso difere do fast-forward exigido pelo sync das notas.
6. Quando a árvore não muda, retorna `Unchanged` sem criar commit.

A aplicação usa mensagem `blup AAAA-MM-DD HH:MM`. Depois de uma mudança, `EnsurePages` configura publicação clássica da raiz da branch `gh-pages`, inclusive substituindo fonte de workflow se necessário. `WaitBuild` consulta o último build a cada três segundos, aguardando o commit enviado por até dez minutos. Falha de espera gera aviso no fluxo da aplicação depois do upload. Se o resultado for `Unchanged`, a aplicação retorna antes de habilitar ou esperar Pages.

Se `EnsurePages` falhou em uma tentativa anterior depois do upload, uma repetição cuja árvore seja idêntica retorna por `Unchanged` e não tenta habilitá-lo novamente. Nesse caso, um novo upload sem mudanças não repara sozinho a configuração Pages; isso depende de a saída realmente permanecer idêntica.

O repositório de publicação contém o site gerado. Notas privadas e estado de sync não são copiados por esse fluxo, a menos que o usuário os coloque manualmente na saída estática.

## Cloudflare Workers

O site vai para um Worker só de arquivos estáticos (Workers com static assets), sem código de servidor. `cloudflareClient` abre formulário de token com `Account > Workers Scripts > Edit` já marcado (`TokenTemplateURL`). `VerifyToken` tenta token de usuário e, quando necessário, verificação por conta. Essa verificação de token não prova sozinha a permissão de Workers; a primeira chamada à conta devolve `ErrMissingPermission` quando falta.

O fluxo escolhe conta entre as retornadas e lê o subdomínio workers.dev da conta. Uma conta nova ainda não tem um: a CLI pergunta o nome (sugestão pelo nome da conta) e o Studio pede na tela Publicar (`CloudflareSubdomain`, `SetCloudflareSubdomain`). Ele é escolhido uma vez por conta e vale para todos os Workers dela; `ErrSubdomainTaken` indica que outra conta já o usa. O Blup só registra o subdomínio quando a conta não tem nenhum.

O nome do site é o nome do Worker: letras minúsculas, números e hífen, sem hífen inicial/final, com máximo de 63 caracteres, porque também é o primeiro rótulo do endereço. A sugestão vem do título (`NameFromTitle`). Nomes de Worker são por conta e publicar substitui o Worker com aquele nome; por isso, antes da primeira publicação (`url` ainda vazio), `Exists` confere o nome e o Blup recusa com `ErrNameTaken` quando ele já pertence a outro Worker da conta.

O endereço é `https://<worker>.<subdomínio>.workers.dev/` (por exemplo `meu-blog.ana.workers.dev`). A aplicação lê o subdomínio atual antes do build, gera o site com esse endereço e grava em `[deploy.cloudflare]` as chaves `account`, `worker`, `subdomain` e `url`. `url` só é preenchido depois da primeira publicação concluída; a partir daí o nome é considerado deste site. Trocar conta ou nome no Studio limpa esse estado.

`internal/deploy/cfworkers/deploy.go` implementa o Direct Upload de assets:

1. Envia o manifesto (`/caminho -> {hash, size}`) para `/accounts/{a}/workers/scripts/{nome}/assets-upload-session`, que devolve um JWT e os buckets de arquivos que faltam.
2. Envia cada bucket para `/workers/assets/upload?base64=true` em multipart, com o conteúdo em base64 e o Content-Type de cada arquivo. Quando o JWT pede envio arquivo a arquivo, usa esse modo. O último envio devolve o JWT de conclusão; sem nada a enviar, vale o JWT da sessão.
3. Publica o Worker com `PUT /workers/scripts/{nome}` só com metadados, sem código: JWT de conclusão, `compatibility_date` e `assets.config` com `html_handling: auto-trailing-slash`, `not_found_handling: 404-page` e o conteúdo de `_headers` e `_redirects`, quando existem.
4. Liga o endereço workers.dev com `POST /workers/scripts/{nome}/subdomain` (`enabled: true`, `previews_enabled: false`).

O hash é `blake3(base64(conteúdo) + extensão_sem_ponto)`, codificado em hexadecimal e truncado a 32 caracteres, como no wrangler. A extensão segue semântica do `path.extname` do Node e preserva caixa.

Limites implementados: 25 MiB por arquivo; até 100.000 arquivos no cliente (no plano gratuito a Cloudflare aplica o limite de 20.000); três buckets simultâneos, ou mais no modo arquivo a arquivo; tentativas e backoff para upload, gateway e publicação.

O scanner ignora `.DS_Store`, `node_modules` e `.git`; na raiz exclui `_worker.js`, `_redirects`, `_headers` e `.assetsignore` do conjunto de arquivos. Só envia arquivos regulares, sem seguir links simbólicos, evitando copiar por um link conteúdo de fora da saída. `_headers` e `_redirects` vão na configuração dos assets, não como arquivos servidos.

O resultado inclui o ID da versão e as quantidades enviadas e reutilizadas. `deployCloudflare` apresenta e retorna `live`, calculada antes do build a partir do nome do Worker e do subdomínio da conta.

## Feedback e exclusão de destinos

O repositório de feedback é público e separado do repositório de saída e do sync. Pode ser utilizado com qualquer provedor de hospedagem. O build atualiza issues antes de enviar arquivos; detalhes em [temas](temas-hugo.md).

O Studio permite selecionar em `DeleteChoice`, de `internal/app/delete_site.go`, o Worker Cloudflare (só quando o Blup já publicou nele, com `url` salvo) ou repositório Pages (`Published`), site Blup Cloud (`BlupSite`), repositório de notas GitHub (`SyncRepo`), repositório de feedback (`Feedback`), pasta local de notas (`Notes`) e diretório técnico (`Site`). As partes locais selecionadas vão à lixeira do sistema; a exclusão remota usa APIs e é permanente. `NotesOwned` é uma inferência do caminho para distinguir a pasta padrão de um cofre anterior, não prova de propriedade nem bloqueio geral da API.

`PartsOf` relaciona o destino `Deploy.Blup` mesmo quando o provedor selecionado depois mudou. `DeleteSite` pode retirar esse site pela sessão Blup Cloud, trata destino já ausente de forma idempotente e limpa sua identidade local se o diretório técnico for mantido. O cofre de sync Blup Cloud continua sendo outro recurso: apagar a pasta local ou retirar o site da lista não purga automaticamente esse cofre remoto. A exclusão de conta segue a retenção de 30 dias.

Falhas remotas retornam `DeleteProblem` com parte, erro e link para ação manual; as partes restantes podem continuar. As exclusões remotas também usam `githubClient` ou `cloudflareClient`, com o mesmo possível caminho de formulários de terminal. Ao chegar ao fim, `DeleteSite` chama `ForgetSite`, mesmo havendo problemas remotos; erro ao carregar o site ou ao mover uma parte local à lixeira retorna antes desse ponto e pode impedir a retirada da lista. Nada disso é executado apenas por gerar ou sincronizar o site.
