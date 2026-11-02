# Publicação e domínio próprio

Fontes: [fluxo GitHub](../../internal/app/publish.go), [fluxo Cloudflare](../../internal/app/cloudflare.go), [Blup Cloud](../../internal/app/blup_publish.go), [APIs do Studio](../../internal/app/deploy_api.go), `internal/blupcloud/sites.go`, `internal/deploy/github/`, `internal/deploy/cfpages/` e `internal/app/domain_api.go`.

## Destino e contas

A publicação sempre gera o site localmente e envia a saída estática. O destino é salvo em `Site.Deploy`; a sincronização de notas usa outro campo e outro repositório. Cloudflare recebe uploads por HTTP e GitHub recebe blobs e árvores pela API, sem `wrangler`, Node ou `git` no caminho de publicação.

Na CLI, `publish` pergunta o provedor se ainda estiver vazio, obtém credenciais, prepara o destino, gera, envia e salva o site. Depois tenta conectar o domínio e oferece abrir a URL. No Studio, o formulário conecta contas e salva o destino antes de `PublishNow`; esse método exige destino e credenciais já definidos, mas usa o mesmo build. A conexão ou consulta de domínio é uma ação separada nas APIs do Studio.

`LoadConnections` verifica o usuário GitHub com a chave salva, devolvendo `githubError` quando falha; `Connections` é a estrutura de resposta. Para Cloudflare, a consulta de conexões informa a existência de token; `ConnectCloudflare` faz a verificação e lista as contas antes de salvar. `Disconnect` remove a chave local de GitHub, Cloudflare ou OpenRouter, sem revogar a chave no serviço.

`PublishNow` obtém clientes por `savedGitHub` ou `savedCloudflare`, que devolvem erro quando falta chave, sem oferecer formulários de conexão. Já as APIs de domínio e de exclusão ainda chamam os clientes interativos da CLI, conforme descrito adiante. O build compartilhado também mantém suas próprias condições de senha e feedback, descritas em [CLI](cli.md).

## Blup Cloud

Destino `blup` (planos pagos; o Grátis tem `sites: 0`). `Site.Deploy.Blup` guarda `id`, `slug` e `url`: o endereço `<slug>.getblup.site` é escolhido antes da primeira publicação (sugestão pelo título, `blupSlug`) e fica fixo depois. `deployBlup` cria o site na nuvem na primeira vez (`ErrSlugTaken`, `ErrSitesLimit`, `ErrPlan`) e salva imediatamente a identidade, inclusive se o upload posterior falhar. Gera com o mesmo `build` (Hugo com `--minify --gc`) para o endereço `getblup.site`, empacota com `blupcloud.PackSite` (arquivos de texto em gzip no nível máximo quando ficam menores; o hash SHA-256 é dos bytes guardados) e envia com `DeploySite`: pergunta quais faltam, manda só esses (seis por vez) e publica a lista de caminhos. Se a nuvem disser que faltam arquivos, manda de novo uma vez. Domínio próprio no Blup Cloud ainda não existe, mesmo nos planos com `customDomain: true`. O serviço está em [Blup Cloud](../../cloud/README.md); o provisionamento está em [deploy](../deploy.md).

Na tela Publicar do Studio, Blup Cloud é a primeira opção: sem conta, entra (`CloudAccount` com `notes={false}`, sem a senha das notas); no Grátis, explica que vem nos planos pagos e abre os planos; senão, pede o endereço e mostra quantos sites do plano estão em uso. O campo aceita só minúsculas, números e hífens (acentos saem, o resto vira hífen) e, 400 ms depois de digitar, pergunta à nuvem (`CheckSiteSlug`): disponível, já é deste site, em uso (com até três nomes parecidos livres para tocar), reservado ou inválido. Publicar só fica liberado com o endereço livre. A tela Publicar só trata do site; Sincronizar abre a mesma tela no modo das notas (`autostart` `sync`). O domínio próprio aparece só depois que o site está no ar pelo Cloudflare ou pelo GitHub, e os passos para conectar essas contas só aparecem ao tocar em Conectar a conta.

### API e armazenamento dos sites

[sites.ts](../../cloud/worker/sites.ts) autentica e verifica o dono por site. `POST /api/sites` cria; `GET /api/sites/check` consulta o nome; `POST /api/sites/:id/missing` aceita até 1.000 hashes; `PUT /api/sites/:id/files/:hash` confere SHA-256 e cota; `POST /api/sites/:id/deploys` ativa o manifesto; `DELETE /api/sites/:id` retira o registro e agenda apagar o prefixo R2. O slug tem de 3 a 40 caracteres, sem hífen nas pontas ou `--`, com nomes reservados em `site-rules.ts`. O cliente sugere nomes, mas a decisão é do servidor.

No bucket `SITES`, `s/<id>/f/<hash>` guarda o arquivo e `s/<id>/d/<deploy>` o manifesto `caminho -> [hash, Content-Type, gzip]`. Um deploy aceita de 1 a 20.000 caminhos únicos; recusa segmentos `.`/`..`, barras invertidas, controles e arquivos especiais de Pages. Arquivos obedecem ao tamanho do plano; notas, histórico e sites dividem armazenamento. Só o deploy ativo é mantido: a limpeza posterior remove o manifesto anterior e arquivos que saíram da lista. Esse mecanismo não oferece histórico de publicação nem rollback automático. Trocar a referência D1 e limpar R2 são etapas distintas; publicação concorrente não é uma transação distribuída entre elas.

### Worker público e cache

[sites-entry.ts](../../cloud/worker/sites-entry.ts) atende o host canônico `<slug>.getblup.site`, separado da conta. [serve-site.ts](../../cloud/worker/serve-site.ts) permite GET/HEAD, resolve `index.html`, redireciona pastas para a barra final preservando query e usa `/404.html` com status 404 quando disponível. O código normaliza HTTP e `www.<slug>` para HTTPS sem `www` quando a requisição chega ao Worker; a cobertura TLS desse segundo nível depende da infraestrutura, conforme [deploy](../deploy.md#7-dns-tls-e-separação-dos-sites).

O Worker público consulta a referência do site, sem verificar o plano em cada leitura. Os endpoints de criação/upload/deploy verificam direitos, mas uma redução de plano não é, por esse código de entrega, uma retirada automática das publicações existentes. Excluir o site é uma operação própria.

A referência do deploy tem cache de 30 segundos por isolate, limitado a 1.024 entradas; os manifestos têm até 32 entradas. A Cache API usa slug, deploy e caminho para evitar misturar versões. HTML, XML/feed, JSON/manifest e `sw.js` revalidam; assets com fingerprint são imutáveis por um ano; os demais ficam uma hora no navegador. ETag permite 304. Gzip é entregue quando aceito, respeitando `q=0`; clientes que recusam recebem descompressão. HEAD retorna metadados sem corpo e não grava sua resposta vazia no cache de GET. A remoção ou nova publicação pode ainda encontrar a referência antiga durante a janela de cache; não é invalidação global instantânea.

## Autenticação GitHub

`githubClient` usa `credentials.toml`, verifica `GET /user` e orienta a criação de token pessoal quando necessário. O link abre o formulário fine-grained com Contents, Pages, Administration e Issues em escrita e duração sugerida de 366 dias. A interface orienta o escopo de repositórios. O cliente usa a API versionada em `2026-03-10` e cabeçalhos de autenticação implementados em `internal/deploy/github/client.go`.

O pacote tem implementação de OAuth Device Flow em `internal/deploy/github/auth.go`, com início, polling, expiração, `authorization_pending` e `slow_down`. Os fluxos atuais da CLI e do Studio conectam por token colado e não acionam esse fluxo como login padrão.

O cliente HTTP classifica erros de token, permissão, conflitos, rate limit, arquivo grande e Pages. Possui repetição, controle de intervalos entre escritas e leitura do atraso de rate limit. Os testes substituem endpoints HTTP por servidores falsos.

## GitHub Pages

`deployGitHub` confirma ou cria o repositório de saída, atribuindo o proprietário ao usuário autenticado. `EnsureRepo` cria repositório público quando ausente e recusa criar sob conta diferente. Um repositório já existente pode ser usado se a chave tiver permissão. Não se presume que mudar o campo owner publique automaticamente em uma organização.

O endereço esperado é `https://<owner>.github.io/<repo>/`, exceto quando o repositório se chama `<owner>.github.io`, que usa a raiz. Domínio próprio substitui esse endereço no build.

`github.Publish` em `internal/deploy/github/publish.go`:

1. Percorre arquivos regulares da saída e calcula SHA-1 de blob Git. Ignora `.git` e arquivos não regulares; rejeita arquivo acima de 100 MiB.
2. Acrescenta `.nojekyll` virtual e, quando configurado, `CNAME` com domínio e quebra de linha.
3. Compara com a árvore remota da branch `gh-pages`. Evita reenviar conteúdo existente, reutiliza blobs e envia o que mudou.
4. Monta a árvore espelhando a saída: arquivos ausentes são retirados da branch de publicação.
5. Cria a branch órfã se necessário e atualiza a referência conforme a estratégia de publicação, que permite atualização forçada. Isso difere do fast-forward exigido pelo sync das notas.
6. Quando a árvore não muda, retorna `Unchanged` sem criar commit.

A aplicação usa mensagem `blup AAAA-MM-DD HH:MM`. Depois de uma mudança, `EnsurePages` configura publicação clássica da raiz da branch `gh-pages`, inclusive substituindo fonte de workflow se necessário. `WaitBuild` consulta o último build a cada três segundos, aguardando o commit enviado por até dez minutos. Falha de espera gera aviso no fluxo da aplicação depois do upload. Se o resultado for `Unchanged`, a aplicação retorna antes de habilitar ou esperar Pages.

Se `EnsurePages` falhou em uma tentativa anterior depois do upload, uma repetição cuja árvore seja idêntica retorna por `Unchanged` e não tenta habilitá-lo novamente. Nesse caso, um novo upload sem mudanças não repara sozinho a configuração Pages; isso depende de a saída realmente permanecer idêntica.

O repositório de publicação contém o site gerado. Notas privadas e estado de sync não são copiados por esse fluxo, a menos que o usuário os coloque manualmente na saída estática.

## Cloudflare Pages

`cloudflareClient` abre formulário de token com `Account > Cloudflare Pages > Edit`. `VerifyToken` tenta token de usuário e, quando necessário, verificação por conta. Essa verificação de token não prova sozinha a permissão de criação ou envio ao projeto.

O fluxo escolhe conta entre as retornadas, valida nome de projeto e chama `EnsureProject`. O nome usa letras minúsculas, números e hífen, sem hífen inicial/final, com máximo de 58 caracteres. O projeto usa Direct Upload e branch de produção `main`.

A URL é baseada no `Subdomain` retornado pela API, que pode incluir sufixo para evitar colisão global. A aplicação salva esse host e gera o site com ele antes do envio. Não deduz sempre o host a partir do nome do projeto.

`internal/deploy/cfpages/client.go` descreve e implementa o protocolo:

1. Obtém JWT temporário em `/accounts/{a}/pages/projects/{p}/upload-token`.
2. Consulta `/pages/assets/check-missing` com os hashes.
3. Envia os assets ausentes para `/pages/assets/upload` em base64.
4. Tenta registrar todos os hashes em `/pages/assets/upsert-hashes`; o erro dessa etapa é descartado, sem interromper a criação do deployment.
5. Cria deployment por multipart com manifesto, branch e arquivos de regras suportados.

O hash é `blake3(base64(conteúdo) + extensão_sem_ponto)`, codificado em hexadecimal e truncado a 32 caracteres. A extensão segue semântica do `path.extname` do Node e preserva caixa. O pacote determina Content-Type sem acrescentar charset.

Limites implementados: 25 MiB por arquivo; padrão de 20.000 arquivos, podendo variar pela claim `max_file_count_allowed` do JWT; lotes de até 40 MiB de conteúdo bruto e 2.000 arquivos; três uploads simultâneos; tentativas e backoff para upload, gateway, check-missing e criação do deployment. JWT expirado ou erro de autorização específico do Pages pode provocar renovação.

O scanner ignora `.DS_Store`, `node_modules` e `.git`; na raiz exclui `_worker.js`, `_redirects`, `_headers`, `_routes.json` e `.wrangler` do conjunto comum de assets. Só envia arquivos regulares, sem seguir links simbólicos, evitando copiar por um link conteúdo de fora da saída. `_headers` e `_redirects` são tratados como partes próprias do deployment. Uma pasta chamada `functions` continua sendo conteúdo estático, sem compilação de Pages Functions.

O resultado inclui URL de produção, URL exclusiva, ID de deployment e quantidades enviadas e reutilizadas. `deployCloudflare` apresenta e retorna `live`, calculada antes do build por `siteURL(site, project.URL())` a partir de `EnsureProject` ou do domínio próprio. Não usa `res.URL` para escolher a URL mostrada.

## Domínio próprio

`normalizeDomain`, em `internal/app/domain.go`, primeiro apara espaços e converte para minúsculas. Remove apenas os prefixos `https://` e `http://`, corta a partir da primeira `/` e retira ponto final. Tenta converter para IDNA ASCII; se `ToASCII` falhar, mantém o texto anterior e continua a validação. Valida formato de host e sufixo público ICANN e recusa subdomínios dos serviços `pages.dev` e `github.io`. `publicsuffix` distingue domínio raiz de subdomínio, incluindo sufixos como `com.br`.

`SetDomain` salva o domínio e tenta conectar. Ao trocar de domínio, tenta desconectar o antigo antes, descartando erro dessa desconexão anterior. `RemoveDomain` desconecta o host salvo e só então limpa o campo: se a desconexão falhar, retorna erro e mantém `site.Domain`. Essas operações não reescrevem automaticamente todos os links de um build anterior; o painel orienta republicar.

`CheckDomain`, `SetDomain` e `RemoveDomain` chegam a `githubClient` ou `cloudflareClient`, inclusive por `disconnectDomainQuiet`. Esses clientes podem abrir navegador, pedir chave e confirmar nova tentativa por formulários de terminal quando a chave falta ou é inválida. O nome `Quiet` não remove esse caminho interativo. As chamadas do Studio não substituem esses formulários por diálogos Wails; o efeito exato de executar esse caminho sem terminal não foi validado, portanto não se afirma que funcione, bloqueie ou falhe de uma forma específica.

| Estado de `DomainStatus` | Significado |
| --- | --- |
| `none` | Sem domínio |
| `unpublished` | Destino ainda não tem dados suficientes |
| `active` | Provedor considera o domínio ativo |
| `dns` | Mostra registros a criar |
| `waiting` | Cloudflare está preparando um domínio em sua zona |
| `apex-cloudflare` | Domínio raiz fora do fluxo de zona Cloudflare, com orientação e alternativa `www` |

No Cloudflare, adiciona o domínio ao projeto, tenta revalidar quando não ativo e apresenta CNAME para o host real `pages.dev`. O Blup não cria registros em um registrador externo. Para domínio raiz, a interface orienta adicionar a zona à conta Cloudflare ou usar `www`; não promete que um CNAME convencional na raiz será criado pelo aplicativo.

No GitHub, configura o `cname` via Pages API, consulta saúde DNS até cinco vezes quando pendente, com intervalos de dois segundos, e tenta exigir HTTPS quando elegível. Para domínio raiz, apresenta os quatro endereços A codificados no programa: `185.199.108.153`, `185.199.109.153`, `185.199.110.153` e `185.199.111.153`, além do CNAME `www` para `<owner>.github.io`. Subdomínios recebem CNAME para esse host. Esses valores descrevem o código atual.

O campo `https` do status do GitHub indica a exigência HTTPS conhecida ou ativada. O código não controla o tempo de propagação DNS ou de emissão do certificado.

## Feedback e exclusão de destinos

O repositório de feedback é público e separado do repositório de saída e do sync. Pode ser utilizado com qualquer provedor de hospedagem. O build atualiza issues antes de enviar arquivos; detalhes em [temas](temas-hugo.md).

O Studio permite selecionar em `DeleteChoice`, de `internal/app/delete_site.go`, o projeto Cloudflare ou repositório Pages (`Published`), repositório de notas (`SyncRepo`), repositório de feedback (`Feedback`), pasta local de notas (`Notes`) e diretório técnico (`Site`). As partes locais selecionadas vão à lixeira do sistema; a exclusão remota usa APIs e é permanente. `NotesOwned` é uma inferência do caminho para distinguir a pasta padrão de um cofre anterior, não prova de propriedade nem bloqueio geral da API.

`PartsOf` e esse fluxo `DeleteSite` não incluem o destino `blup` nem a remoção do cofre de sync Blup Cloud. O painel web da conta tem exclusão de site por `DELETE /api/sites/:id`, e o cliente Go também possui esse método, mas não se deve prometer que Apagar o site/cofre na home do Studio execute a mesma remoção remota. Retirar da lista é outra ação local.

Falhas remotas retornam `DeleteProblem` com parte, erro e link para ação manual; as partes restantes podem continuar. As exclusões remotas também usam `githubClient` ou `cloudflareClient`, com o mesmo possível caminho de formulários de terminal. Ao chegar ao fim, `DeleteSite` chama `ForgetSite`, mesmo havendo problemas remotos; erro ao carregar o site ou ao mover uma parte local à lixeira retorna antes desse ponto e pode impedir a retirada da lista. Nada disso é executado apenas por gerar ou sincronizar o site.
