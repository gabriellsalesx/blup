# Temas Hugo e comportamento no navegador

Fontes principais: [estrutura base](../../theme/layouts/baseof.html), `theme/layouts/_partials/`, `theme/layouts/_markup/`, `theme/assets/css/`, `theme/assets/js/`, `internal/lock/lock.go` e `internal/pwa/pwa.go`.

## Organização do tema

Há um tema embutido chamado `blup`, com três modelos selecionados por `site.Params.model`. `theme/hugo.toml` declara Hugo mínimo `0.169.0` e modelo padrão Mochi. O scaffold passa o modelo escolhido no `blup.toml`.

`baseof.html` produz o documento HTML, idioma, head de metadados, SEO, CSS, link de pular para conteúdo, ícones, cabeçalho e rodapé do modelo. Os layouts `home.html`, `page.html`, `section.html`, `term.html` e `taxonomy.html` delegam ao partial do modelo correspondente. Há também `404.html`, `robots.txt`, RSS e formatos JSON auxiliares.

| Conjunto | Função |
| --- | --- |
| `theme/layouts/_partials/mochi/` | Blog em cards, componentes compartilhados, imagens, autores, busca, proteção e scripts |
| `theme/layouts/_partials/nori/` | Blog de coluna central, listas compactas, destaque e páginas próprias |
| `theme/layouts/_partials/bento/` | Documentação, roadmap, changelog, ajuda, blog, quadro de Feedback e navegação |
| `theme/layouts/_partials/site/` | Configuração, menu, redes, privacidade e perfis |
| `theme/layouts/_partials/head/` | Metadados, SEO, JSON-LD e CSS |
| `theme/layouts/_markup/` | Hooks de imagem, links, headings, callouts e fórmulas |
| `theme/assets/css/{mochi,bento,nori}.css` | Folha escolhida por modelo |
| `theme/assets/js/` | Tema de cores, navegação, busca, player, proteção, feedback, consentimento e PWA |
| `theme/assets/favicon.svg` | Favicon padrão e fallback de origem dos ícones do app |
| `theme/assets/defaults/` | `apple-touch-icon.png`, `avatar.png` e `avatar.svg` padrão |
| `theme/assets/og/` | Base e fonte da geração de imagem social |
| `theme/i18n/{pt,en,es}.toml` | Textos dos temas, independentes das mensagens da CLI e da interface do Studio |
| `theme/static/fonts/` | Inter local em WOFF2 e licença |
| `theme/data/lucide.json`, `theme/licenses/` | Dados de ícones e licença correspondente |

O CSS passa por `css.Build`; tanto a minificação quanto o fingerprint da folha do modelo são condicionados a `hugo.IsProduction`. JavaScript passa por `js.Build` com minificação e fingerprint em ambos os modos. A fonte Inter é pré-carregada localmente. O código de busca é importado sob demanda. Não há etapa Node obrigatória para compilar o tema do site.

### Ícones do quadro

[mochi/icons.html](../../theme/layouts/_partials/mochi/icons.html) contém um sprite SVG inline, uma cópia por página, com símbolos `i-<nome>` referenciados por `<use>`. Os símbolos acrescentados para o quadro (`plus`, `folder`, `list-filter`, `user` e `message-square`) foram extraídos dos SVGs do Lucide baixado em `UNA-v.15.0.0-RC3/plugins_public/lucide/icons`; suas formas conferem com esses arquivos, sem desenho manual. `feedback.js` também usa `<use>` ao criar cartões e detalhes no navegador. O build Hugo inclui o sprite já convertido, sem gerá-lo de novo a partir dessa pasta externa.

O partial [icon.html](../../theme/layouts/_partials/icon.html) é outro mecanismo: inclui um ícone pelo nome a partir de `theme/data/lucide.json`, com aviso e fallback para `circle-help` se desconhecido. A licença está em `theme/licenses/lucide-ISC.txt`. O comentário do sprite identifica separadamente símbolos de marcas existentes, vindos de `design/icones/contorno/`; a procedência Lucide dos símbolos novos do quadro não se estende a esses símbolos de marcas.

## Mochi

`mochi/home.html` lista posts com um destaque: primeiro `featured: true`, ou o primeiro post disponível. O destaque sai da lista restante, que pagina de dez em dez conforme scaffold. A home apresenta título e o texto introdutório da nota, caindo para a descrição do site somente quando esse texto está vazio. Cards mostram capa ou fallback, tags, título, resumo e autoria.

`mochi/posts.html` define os posts visíveis usados na home. Se estiver no blog do Bento, limita àquela seção. Páginas de tags e autores usam outro caminho: `mochi/term.html` filtra `.Pages` com `visible.html` e `blog-only.html`, que no Bento mantém somente a seção de blog. As pills do Mochi contam tags pela taxonomia global, incluindo posts escondidos; na home de blog do Bento recebem explicitamente o conjunto filtrado. Portanto a contagem de tags não representa sempre o mesmo conjunto visível da listagem.

`mochi/page.html` mostra hero com título, descrição explícita, autoria, data, capa, player, corpo, tags e compartilhamento. Possui barra de leitura, sumário H2/H3 para conteúdo não protegido, perfis e até quatro relacionados, obtidos por `site.RegularPages.Related` e completados com posts recentes. O scaffold não configura critérios de relação, portanto a spec não atribui prioridade própria a tags. No Bento, os relacionados são filtrados para a mesma seção. `blup: privacy` usa página simples, sem o tratamento comum de post.

Os três modelos têm o mesmo cabeçalho do Nori: logo e nome à esquerda (o nome com reticências em telas estreitas), e à direita busca, tema e o botão do menu, juntos em `.header-actions`. Bento usa `mochi/header.html` e o CSS do Mochi. `site/network.html` reconhece a rede pelo endereço; o ícone do Discord vem do Tabler Icons (MIT, licença em `theme/licenses/`), os demais seguem o desenho do tema.

`mochi/more.html` oferece link de próxima página. Com JavaScript, `blup.js` busca a página e acrescenta os cards em `.post-grid` ou entradas em `.changelog`; sem JavaScript, o link continua navegável. Depois de importar os scripts dos itens, dispara `blup:added`. Hoje somente `lock.js` escuta esse evento: os itens acrescentados não recebem inicialização e contagens de feedback. O carregar mais comum não trata `.nori-list`, então no Nori o link navega para a página seguinte.

## Nori

Nori usa uma coluna central e componentes em `nori/`, com cabeçalho, rodapé, home, listas, datas, avatares, destaque e páginas de taxonomia próprios. Compartilha resolução de capas, perfis, conteúdo, proteção, player, busca e feedback com as peças comuns.

Na home também escolhe um destaque e pagina os demais. Com feedback, o markup inclui abas de recentes, mais reagidos e mais comentados, inicialmente ocultas. `feedback.js` só as mostra quando há pelo menos uma issue carregada com reações e pelo menos uma com comentários; podem ser issues diferentes. As abas de mais reagidos/comentados têm uma busca própria de páginas seguintes para ordenar a lista, separada do carregar mais de `blup.js`. O post apresenta capa, título, descrição, autores, data, tags, áudio, corpo e comentários. Política de privacidade usa layout simples.

Nori está implementado nos templates, em CSS, exemplos e telas do Studio. Sua ausência no seletor da CLI é uma diferença de interface, não ausência de tema.

## Bento

O tipo da página vem de `bento/kind.html`, pela propriedade `blup` da primeira seção. Não é inferido do nome literal da pasta.

| Tipo | Implementação e apresentação |
| --- | --- |
| `docs` | `doc.html`, `tree.html`, `tree-item.html`, `order.html`, `toc.html`, `pager.html`: navegação em árvore, grupos, sumário e sequência de leitura |
| `roadmap` | `roadmap.html`, `roadmap-item.html`, `status.html`: quadro por estado e página de cada item |
| `changelog` | `changelog.html`, `release.html`, `changelog-entry.html`: lista de versões e detalhes |
| `help` | `help-home.html`, `help-collection.html`, `help-row.html`, `help-article.html`: coleções e artigos de ajuda |
| `blog` | Usa peças do Mochi para posts e listagem de blog |
| `feedback` | `feedback-board.html`, `feedback-examples.html`, `feedback-example-pages.html` e `feedback-block.html`: quadro de pedidos com votos e comentários, oferecido em Seções no Studio |
| Sem tipo conhecido | `plain.html` e layouts de fallback |

A home do Bento apresenta cartões para as seções reconhecidas, em ordem de peso. A documentação usa o índice da seção, suas páginas por peso e subpastas agrupadas; o título remove prefixos numéricos de exibição, mas isso não significa remover esses prefixos das URLs.

O roadmap normaliza vários valores de `status` em `planned`, `progress` e `done`. Valor ausente ou não reconhecido cai em planejado. `category` fornece o rótulo; sem ela, tenta tag e depois o estado. As colunas abertas podem ser reordenadas por votos no cliente, com peso como desempate. `sort_by_votes: false` no `_index.md` impede a ordenação por votos.

Remover ou adicionar uma seção é uma mudança nos arquivos de notas. O Studio oferece ações que movem a pasta à lixeira ou escrevem exemplos; o template apenas interpreta o que existe.

### Seção Feedback

Feedback é reconhecida por `blup: feedback`, inclusive quando a pasta tem outro nome. Fica em `studioKinds`, separada das cinco seções `docsKinds` do assistente de criação e da CLI; o Studio a acrescenta pelo painel Seções após criar o site. Há modelos em `internal/app/starter/{pt,en,es}/bento/Feedback/_index.md`.

O quadro tem categorias fixas com contagens, menu próprio de status (todos, ativos e os cinco estados), Mais votados, Recentes, busca, Novo pedido e dez itens por vez com Carregar mais pedidos. O cartão inteiro abre o detalhe e o botão de voto tem ação própria. Categorias e opções de status não são editáveis pelo Studio. Pedidos reais são issues marcadas, abertas em `#p-N`; a leitura usa `/posts` do blup-api ou GitHub diretamente, e o dono altera etiquetas `status:` no painel Feedback do Studio.

`examples` no front matter fornece títulos, corpos, votos, dias e comentários `replies`. Os exemplos aparecem também na publicação até haver um pedido real carregado; somente a prévia mostra o aviso de exemplo. Os detalhes `#ex-N` pertencem à mesma página. Sem comentários habilitados, filtros e exemplos funcionam, o voto é apenas local e Novo pedido explica a configuração necessária ao tentar enviar.

Os controles são traduzidos por `theme/i18n/{pt,en,es}.toml`; título e exemplos vêm da tradução da nota, que pode exibir selo de IA. Pedidos e comentários reais conservam seu texto original em todos os idiomas. Em espanhol usa-se `solicitud`. O [contrato completo do quadro](feedback.md) detalha schema, status, login, limites, cache e tradução.

## Conteúdo, links e imagens

`title.html` usa título disponível, primeiro H1 renderizado ou nome do arquivo, retirando prefixo numérico da apresentação. `content.html` remove o H1 inicial para o layout não o repetir e passa pelo filtro de comentários Obsidian. `uncomment.html` remove `%%...%%` do conteúdo renderizado, preservando os trechos de código tratados pelo filtro.

`description.html` prefere descrição explícita; na home usa a configuração; depois extrai texto, normaliza espaços e limita a 160 caracteres. Para post protegido retorna descrição genérica traduzida. Algumas áreas visuais de página usam `.Description` diretamente, portanto descrição explícita e demais metadados não devem ser considerados criptografados.

`render-link.html` resolve caminhos Markdown relativos com `Page.GetPage`, preserva fragmentos normalizados e publica anexos encontrados. Link externo recebe `rel="noopener"`; não é forçado a abrir nova aba. Destino interno ausente gera aviso Hugo.

`vault-resource.html` aceita links relativos, caminhos do cofre e referências de propriedade como `[[foto.png]]`. Tenta diretório da nota, raiz publicada, raiz do cofre e busca por nome. O fallback por nome significa que arquivos homônimos podem depender da primeira correspondência.

`render-image.html`:

- Resolve recurso local e separa tamanho opcional do alt `texto|300` ou `texto|300x200`; usa a largura pedida e proporção da imagem.
- Para imagem processável, gera WebP q80 nas larguras 480, 800, 1200 e 1600 sem ampliar acima do original. Imagem menor recebe conversão apropriada. Fornece `srcset`, dimensões e carregamento preguiçoso.
- SVG, GIF e recursos não processáveis usam o arquivo direto. URL externa não recebe as transformações locais.
- Áudio e vídeo locais embutidos viram players HTML; PDF e outros anexos viram links.
- URL de YouTube reconhecida no formato de imagem embed gera iframe em `youtube-nocookie.com`, ou placeholder dependente de consentimento.

O hook de heading inclui âncoras a partir de H2 e suporta classe. `render-blockquote.html` interpreta callouts do Obsidian, aliases de tipos e título padrão traduzido. `+`/`-` após o tipo geram `<details>` aberto/fechado. Tipo desconhecido recebe aparência de nota. Fórmulas são transformadas em MathML por `transform.ToMath` no build, com aviso e fallback quando falham, sem biblioteca matemática no navegador.

## Identidade, autoria, SEO e feeds

`site/settings.html` lê `_index.md`: `site_name`, `site_description`, logo, favicon, redes e `header_name` (`false` deixa só o logo no cabeçalho; sem a propriedade, logo e nome); usa `blup.toml` como fallback do nome e descrição. `site/menu.html` resolve os itens na ordem salva, indica item atual e avisa para destinos não encontrados.

`site/author-profile.html` lê a nota do autor como asset na pasta marcada `authors`, extraindo `photo`, `role`, `social` e bio Markdown. Em uma versão traduzida do site procura primeiro `<autores>/<idioma>/<Nome>.md`, depois o perfil original; mantém o nome do arquivo, que identifica o autor. A foto pode cair para imagem no corpo. `mochi/authors.html` usa a taxonomia `authors` e depois o autor padrão. Perfis e suas traduções são recursos, excluídos das páginas de conteúdo; `/authors/` é a taxonomia. A tradução de cargo e bio está descrita em [Studio](studio.md#idiomas-e-tradução).

O head inclui título composto, canonical, descrição, Open Graph, Twitter card, datas, tags e JSON-LD. `head/jsonld.html` emite `WebSite` para a home, `TechArticle` para toda página regular do Bento, inclusive blog e roadmap, e `BlogPosting` para páginas regulares dos demais modelos; também inclui breadcrumbs fora da home. `og-image.html` usa capa quando apropriado ou gera imagem social com os assets em `theme/assets/og/`. `hidden`, `noindex` e 404 recebem meta robots noindex.

`rss.xml` limita a 20 páginas recentes, retira escondidas e usa descrição genérica para protegidas. Na home e taxonomias do Bento filtra para notícias de blog e changelog. `robots.txt` e sitemap dependem também da geração Hugo; `hidden` é uma convenção de apresentação do tema, não uma remoção universal de todo arquivo ou referência.

## Busca Pagefind

O corpo público recebe `data-pagefind-body` e metadado do título. Corpo protegido e nota escondida não recebem essa marcação nos templates de conteúdo. A indexação acontece depois da criptografia no build Go.

`mochi/search.html` fornece o diálogo e textos. `blup.js` controla sua abertura e importa `search.js`; os atalhos são Ctrl/Cmd+K e `/`, este último quando o foco não está em input, textarea ou select. O módulo carrega `pagefind.js` dinamicamente, ajusta a URL base para suportar subpastas, usa debounce de 150 ms e mostra até oito resultados com excerto. Enter abre o primeiro resultado. Falha de carregamento mostra indisponibilidade.

A CLI prepara índice no início da prévia; o Studio não o gera em `StartPreview`. Uma pasta `.blup/preview/pagefind` previamente criada pode estar disponível pelo mount, mas não se deve deduzir que o Studio sempre mantenha a busca atualizada.

## Posts protegidos

`locked.html` considera `password` presente. `true`, `sim` e `yes`, sem distinção de caixa, viram `__global__`; outros valores não vazios são senha literal. `mochi/lock.html` tem três modos:

| Modo | Corpo |
| --- | --- |
| `HUGO_BLUP_MODE=preview` | Visível com aviso |
| `HUGO_BLUP_MODE=publish` | Entre marcadores provisórios para pós-processamento |
| Sem modo Blup | Omitido; formulário permanece sem corpo legível |

O manifesto `blup-locks.json` relaciona `File.UniqueID` a senha. `internal/lock.Process` percorre HTML, substitui regiões por um elemento `data-blup-cipher` e remove o manifesto. Usa AES-256-GCM, PBKDF2-SHA256 com 600.000 iterações, salt aleatório de 16 bytes e IV de 12 bytes. Verifica marcadores residuais nos HTML processados. A camada de build apaga a saída quando a proteção falha.

`lock.js` usa WebCrypto para derivar a mesma chave e inserir o HTML decifrado no DOM. Lembra até cinco senhas bem-sucedidas em `sessionStorage`, na chave `blup:passwords`, somente para aquela sessão de aba. Reutiliza-as em outros posts protegidos. Não há servidor de autenticação para desbloquear posts.

A proteção envolve o corpo delimitado. Título, capa, autores, URL e outros dados renderizados fora dele continuam públicos. Anexos referenciados são arquivos estáticos separados. O código de narração recusa gerar áudio de nota protegida, mas o template de player é independente da criptografia do corpo. `hidden` e `noindex` não oferecem a mesma proteção.

## Comentários, reações e votos

`feedback-kind.html` habilita feedback para roadmap, changelog e blog do Bento; em Mochi/Nori, para páginas de post que não sejam política de privacidade. O manifesto de publicação exclui páginas com `hidden`.

`internal/app/feedback.go` cria ou atualiza uma issue por chave de nota em minúsculas com `/`. `bento/feedback-key.html` usa o caminho da original no idioma principal quando uma tradução ligada está disponível; assim todas as versões de uma página compartilham comentários, votos e reações. Sem original ligada, usa o caminho da própria página. A issue recebe um marcador `<!-- blup:<chave> -->` e label gerenciada. O mapeamento é salvo em `<site>/data/blup_feedback.json`, com repositório, URL do site, API de login habilitada e mapa de issues. O build pode repetir Hugo para incorporar a nova relação. Renomear a original altera sua chave; não há identidade de feedback independente do caminho.

Configurar feedback cria ou usa um repositório público `<slug>-feedback`, ativa issues e verifica permissões. O site de hospedagem pode estar no GitHub ou Cloudflare. `appInstalled` consulta `/installed?repo=...` no serviço de login; somente instalação confirmada acrescenta `data-fb-api` aos scripts gerados. Sem ela, os controles encaminham ao GitHub.

`feedback.js` lê contagens e comentários, faz login GitHub pelo serviço, recebe e valida estado do retorno e mantém a autenticação em `localStorage`, no navegador do visitante (decisão do dono: não pedir login a cada visita); sair apaga a chave. Renova a autorização quando necessário. Visitante anônimo pode ler pelo serviço com cache; autenticado usa diretamente GitHub com sua chave. Existem caches em sessão e invalidação após mudanças. HTML recebido para corpos e prévias passa pela cópia de elementos permitidos em `markdown`, sem handlers, estilos ou embeds ativos; `safeURL` filtra endereços.

O cliente não pagina a lista de issues nem a de comentários. Nas chamadas diretas ao GitHub pede `per_page=100` para cada uma e só usa a primeira resposta, podendo omitir issues ou comentários além desse conjunto. As leituras via serviço também não têm loop de paginação no navegador. No repositório irmão, `blup-api/src/index.ts` percorre até cinco páginas para `/issues`, até três para `/posts` e uma para `/comments`; o [contrato de `/posts`](feedback.md#leitura-por-posts-e-limites) distingue isso da paginação visual de dez itens. A lista de nós de feedback das páginas é capturada uma vez ao carregar o script e não é ampliada por `blup:added`; o quadro liga seus detalhes dinamicamente por `attach`.

Os pedidos do [quadro Feedback](feedback.md) são issues independentes das notas, criadas por visitantes com `<!-- blup:feedback kind=X lang=Y -->`. O build não cria uma issue por exemplo da nota. `status: planned`, `status: progress`, `status: done` e `status: declined` são etiquetas geridas pelo Studio; Novo retira essas etiquetas, e Feito e Recusado também fecham a issue.

Os controles permitem votar, reagir, escrever com prévia Markdown, responder por citação e remover comentários conforme permissões. Respostas são comentários na mesma issue, com citação e menção, sem árvore própria no banco do Blup. A API de login pertence a `github.com/gabriellsalesx/blup-api`; seu servidor não está implementado neste repositório.

Ao postar pelo site, `feedback.js` acrescenta `<!-- blup:lang=pt -->` ao comentário, usando o idioma da página. Lido em outro idioma, o comentário mostra um rótulo como "comentado em English". A resposta por citação retira essa marca antes de montar o trecho citado. O texto real não é traduzido e comentários sem marca não têm idioma inferido. No quadro, a marca do pedido também permite informar em qual idioma ele foi escrito.

## Cookies e vídeos

`consent.html` injeta configuração de ferramentas e, quando `banner` está ativo, banner e diálogo. Categorias são necessárias, estatísticas, marketing e mídia. Necessárias ficam selecionadas; mídia controla vídeos. O link de cookies do rodapé reabre as escolhas.

`consent.js` carrega Analytics, Umami e Plausible para estatísticas e Pixel para marketing. GTM pode carregar após autorização de uma dessas categorias, recebendo o Consent Mode e suas atualizações. Sem banner, tudo configurado carrega diretamente. A escolha fica em `localStorage` por um ano. Retirar um consentimento anteriormente dado grava a nova escolha e recarrega a página para que as ferramentas recusadas não carreguem novamente.

`internal/app/privacy.go` cria uma nota de política com base nas ferramentas e contato, sem sobrescrever uma já existente. O banner pode usar `cookies.privacy` explícito, passado pelo scaffold somente com `banner = true`; sem ele procura a nota de política. Os rodapés sempre usam `site/privacy.html`, que procura `Params.blup` igual a `privacy` e diferencia caixa. Já a escolha de layout simples da página converte `blup` para minúsculas, portanto reconhecer o layout não garante descobrir o link no rodapé.

## Prévia integrada ao Studio

Com `HUGO_BLUP_MODE=preview`, `mochi/scripts.html` injeta um script que só atua dentro de iframe. Envia `postMessage` de `loaded`, `scroll` e `ready`, incluindo posição, caminho e query da página, e obedece à mensagem `scrollTo`. `Preview.svelte` usa essas mensagens para trocar o iframe reconstruído na mesma posição. Esse script não entra no site em modo publish; não sincroniza a página por linha do Markdown.

## Player e instalação como app

`listen.html` resolve `audio` e inclui `audio_hash` como versão na URL. `blup.js` fornece play/pause, avanços de 15 segundos, seek, tempo e velocidades 1, 1,25, 1,5, 2 e 0,75. Pausa outro player da página ao iniciar. Sem JavaScript, ficam os controles nativos do áudio.

O botão de guardar narração só aparece com service worker disponível. Faz download com progresso e salva em Cache Storage `blup-audio`; novo toque remove a cópia. Uma versão nova do mesmo caminho substitui a antiga. A duração pode vir de `audio_seconds` e depois dos metadados reais.

O scaffold habilita `manifest.json` em qualquer modo. O link para ele no head, a publicação da cópia intermediária `blup-icon.<ext>` e a inclusão de `pwa.js` dependem de `HUGO_BLUP_MODE=publish`. A origem da cópia é o favicon configurado ou, como fallback, `theme/assets/favicon.svg`. `internal/pwa.Write` lê esses arquivos, gera PNGs de 192/512, ícone maskable 512, apple touch 180 e `favicon.ico` com 16/32/48, remove a cópia intermediária e escreve `sw.js`.

O worker tem versão calculada por hash. A lista inicial inclui todos os `index.html` fora de `vault/` e `pagefind/`, convertidos para suas URLs de página; inclui também arquivos dessas áreas percorridas com extensões `.css`, `.js`, `.woff2`, `.svg`, `.ico` e `.json`, exceto o próprio `sw.js`, além de todos os arquivos de `icons/`. Imagens raster fora de `icons/` não entram nessa lista inicial. O download de todas essas páginas ocorre ao instalar o worker e pode pesar em sites grandes.

Navegação tenta a rede antes; na falha, tenta a página em cache ignorando query e depois a home em cache. Outros recursos da mesma origem/escopo tentam cache primeiro e podem ir para `blup-seen`. O worker não limpa nem revalida periodicamente esse cache, que pode manter conteúdo antigo de arquivos sem fingerprint; `blup-seen` e `blup-audio` sobrevivem à troca da versão principal. APIs de outra origem não são interceptadas.

Áudio não é guardado automaticamente por streaming: o worker usa a cópia solicitada no player, inclusive respondendo a ranges com HTTP 206. A chave considera a URL com versão. `pwa.js` registra o worker e apresenta botão de instalar quando o navegador oferece `beforeinstallprompt`; a experiência de instalação depende do navegador. A prévia não registra esse worker pelo tema.

## Mapa do changelog (Bento)

A página do changelog abre com um mapa dos últimos 12 meses, como o de contribuições do GitHub ([changelog-map.html](../../theme/layouts/_partials/bento/changelog-map.html)): uma coluna por semana, um quadrado por dia, a partir de 364 dias antes da data do build, recuando até o domingo daquela semana. Usa as páginas visíveis recursivas da seção, independentemente da página atual da listagem, e desenha somente dias até hoje.

[change-kind.html](../../theme/layouts/_partials/bento/change-kind.html) prefere `type` reconhecido (`new`, `improvement`, `fix`, `security`, `breaking`); senão procura termos nas tags em português, inglês ou espanhol. A prioridade é `breaking`, `security`, `fix`, `new`, `improvement` e `other`. Correção é vermelha, novidade verde, melhoria azul, segurança roxa e mudança importante laranja. Um dia com várias versões usa a categoria de maior prioridade e a classe `many`, com cor mais forte.

Passar o mouse ou focar mostra a data e as versões daquele dia; clicar abre a primeira versão do grupo no permalink gerado. Há legenda e total de versões. O calendário é produzido no build, sem consulta remota; `blup.js` posiciona a caixa e põe o scroll nos meses mais novos. Meses, dias e rótulos acompanham o idioma da versão do site por Hugo e `theme/i18n`.

