# Podcast dos blogs

O Podcast é uma seção dos blogs Mochi e Nori que só o Blup Studio oferece (em Seções). Ele junta as narrações dos posts como episódios numa página com player próprio e publica um feed para Spotify, Apple Podcasts e outros apps. A CLI não oferece a seção; um site que já a tem continua gerando a página e o feed também pela CLI.

Fontes: [configuração do programa](../../internal/app/podcast.go), [serviço Studio](../../studio/podcast.go), [painel](../../studio/frontend/src/PodcastPanel.svelte), [feed](../../theme/layouts/section.podcast.xml) e [player](../../theme/assets/js/podcast.js). Conferido em 02/11/2026.

## Seção e nota

- `internal/app/starter.go` define `blogKinds = ["podcast"]` e `sectionKinds(model)`: Bento usa `docsKinds` mais `studioKinds`; Mochi e Nori, só `blogKinds`. `writeSamples` nunca copia a pasta `Podcast` num blog novo; ela vem só por `AddSection`.
- A pasta modelo existe em `internal/app/starter/{pt,en,es}/{mochi,nori}/Podcast/_index.md`, com `blup: podcast`, `outputs: [html, podcast]` e as propriedades do programa.
- `AddSection` aceita o tipo só quando ele está em `sectionKinds` do modelo. Num blog com menu próprio (propriedade `menu` da nota inicial), acrescenta a pasta ao menu; sem menu próprio o tema já lista todas as pastas.
- `internal/obsidian` reconhece a seção pela marca `blup: podcast` no front matter.

Propriedades da nota (todas opcionais):

| Propriedade | Uso | Sem valor |
| --- | --- | --- |
| `podcast_title` | Nome do programa no feed | Nome do site |
| `description` | Resumo do Google, do compartilhamento do link e do feed (não aparece na página) | Começo do texto da nota, depois a descrição do site |
| `podcast_author` | `itunes:author` e dono | Autor do site |
| `podcast_email` | `itunes:owner`, pelo qual Spotify e Apple confirmam a posse; fica público no feed | Sem `itunes:owner` |
| `podcast_category` | `itunes:category`, nome da lista da Apple em inglês | `Technology` |
| `podcast_explicit` | `itunes:explicit` | `false` |
| `image` | Capa do programa | Capa do episódio mais novo, depois o logo do site |

## Episódios

`theme/layouts/_partials/podcast/episodes.html` lista os posts do idioma da página com `audio`, do mais novo para o mais antigo. Ficam de fora notas com `hidden`, rascunhos, conteúdo não publicado (`publish: false` ou `private: true`) e posts protegidos (`password`), porque o áudio entregaria o texto. A capa de cada episódio é a capa do post (`mochi/cover.html`: a propriedade `image` ou a primeira imagem da nota).

## Página

- `mochi/section.html` e `nori/section.html` desviam para o podcast quando `podcast/is.html` é verdadeiro. Cada modelo mantém o próprio cabeçalho: Mochi com breadcrumb e `.h1`, Nori com `.nori-head` na coluna estreita. Embaixo do título vai o texto da nota (`content.html`, sem os comentários `%%`), como nas outras seções; a propriedade `description` fica para o Google, o compartilhamento e o feed.
- `podcast/page.html` desenha a lista, o botão de copiar o feed, o "Carregar mais episódios" (dez por vez) e o player fixo. As capas usam `mochi/picture.html` com proporção `1:1`, WebP e `sizes` fixo de 112 px.
- `assets/css/podcast.css` é carregado só nessa página (`head/css.html`), com os tokens do tema (`--main-*`, e `--panel`/`--ring` do Nori quando existem). O `<head>` também ganha o `<link rel="alternate">` do feed.
- Na prévia do Studio sem suporte a MP3 (`HUGO_BLUP_NO_MP3`), a página mostra os episódios sem player, como `listen.html`.

## Player (`assets/js/podcast.js`)

- Um único `<audio>` toca todos os episódios. Ao terminar um, o player solicita o próximo no mesmo elemento para favorecer continuidade em celulares. Reprodução com tela apagada e avanço automático dependem das políticas do navegador e sistema; o código não garante esse comportamento em todo aparelho.
- A Media Session publica título, nome do site e capa (JPEG quadrado de 512 px, `podcast/art.html`) e trata play, pausa, anterior, próximo, avançar, voltar e posição.
- Controles: tocar e pausar, 15 s para trás e para frente, barra de posição, tempo, velocidade (1, 1,25, 1,5, 2 e 0,75), sem som e a lista de reprodução.
- As notas do episódio buscam a página do post e copiam o texto (`.prose`) para dentro do episódio, sem sair da página; o áudio continua.
- O episódio e o ponto onde o leitor parou ficam no `localStorage` do aparelho (`blup-podcast:<caminho>`); ao voltar, o player aparece pausado ali.

## Feed

`internal/scaffold` declara o formato de saída `podcast` (`application/rss+xml`, `feed.xml`). A nota da seção pede esse formato em `outputs`, e `theme/layouts/section.podcast.xml` gera um feed por idioma do site (`/podcast/feed.xml`, `/en/podcast/feed.xml`).

- Programa: título, link, descrição, idioma, `atom:link`, `itunes:author`, `itunes:owner`, `itunes:image`, `itunes:category`, `itunes:explicit` e `itunes:type episodic`.
- Episódio: título, link do post, `guid` por `translationKey`, senão `RelPermalink`, data, descrição, `enclosure` com o MP3 (tamanho real em bytes e `?v=` com o `audio_hash`), `itunes:duration` (`audio_seconds`) e `itunes:image`. Sem chave explícita estável, renomear a página pode mudar o GUID.
- As capas do feed são JPEG quadrados de 1400 px, recortados com `Fill ... smart`; o player usa uma variante de 512 px. O aviso de imagem do Studio usa a faixa de 1400 a 3000 px.
- Os endereços saem completos com o endereço do build: o domínio próprio (`blup.toml` `domain`) ou o endereço onde o site é publicado (Blup Cloud, Cloudflare Pages, GitHub Pages), que a publicação passa como `baseURL`. Só a prévia, sem endereço, gera caminhos relativos.

## Tela Podcast no Studio

`PodcastPanel.svelte` aparece no menu quando o blog tem a seção. Usa `Podcast()` e `SavePodcast()` (`studio/podcast.go`, `internal/app/podcast.go`):

- O bloco Idiomas do Studio também aparece nas notas de seção (antes só nas outras), e a tradução da nota do Podcast fica em `Podcast/<idioma>/_index.md`, com as mesmas propriedades.
- Nome, descrição (a do Google, do compartilhamento e do feed), autor, e-mail de contato, categoria (lista da Apple traduzida, valor em inglês), conteúdo explícito e capa (`PickCover`, com aviso quando a imagem não é um quadrado de 1400 a 3000 px).
- Quantos episódios há: posts do idioma principal com narração que o podcast lista.
- O endereço do feed de cada idioma, com Copiar e os atalhos para Spotify for Creators e Apple Podcasts Connect: pelo domínio próprio, senão pelo endereço onde o site foi publicado (`LoadTarget`). Antes da primeira publicação, a tela explica que é preciso publicar.
- `SavePodcast` só grava as propriedades da lista `PodcastProps` e recusa e-mail sem `@`.

Os atalhos abrem serviços externos; o Blup não cadastra automaticamente o programa nem garante aprovação em diretórios. `podcast_email` fica público no feed. Cada idioma usa os posts com áudio daquele idioma: traduzir uma nota não cria uma narração, e narrar a tradução consulta sua voz em `Prefs.Voices[lang]`. O [fluxo de narração](narracao-e-voz.md) descreve como gerar e atualizar esses arquivos.


## URL, proteção e publicação

Feed, canonical e URLs absolutas dos episódios usam a baseURL do build. No Blup Cloud, domínio ativo com SSL e direito ou carência válida pode ser a baseURL; o Studio republica em segundo plano quando essa base muda. Ambos os hosts servem o mesmo deploy sem redirecionamento entre eles, e URLs antigas só mudam no novo build concluído, conforme [publicação](publicacao.md#domínio-próprio-no-blup-cloud). O filtro de episódios evita divulgar o corpo protegido em áudio, mas configurar senha depois de uma narração não apaga automaticamente MP3 já criado ou baixado. A poda de anexos e seus limites estão em [temas](temas-hugo.md).
