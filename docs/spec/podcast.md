# Podcast dos blogs

O Podcast é uma seção dos blogs Mochi e Nori que só o Blup Studio oferece (em Seções). Ele junta as narrações dos posts como episódios numa página com player próprio e publica um feed para Spotify, Apple Podcasts e outros apps. A CLI não oferece a seção; um site que já a tem continua gerando a página e o feed também pela CLI.

## Seção e nota

- `internal/app/starter.go` define `blogKinds = ["podcast"]` e `sectionKinds(model)`: Bento usa `docsKinds` mais `studioKinds`; Mochi e Nori, só `blogKinds`. `writeSamples` nunca copia a pasta `Podcast` num blog novo; ela vem só por `AddSection`.
- A pasta modelo existe em `internal/app/starter/{pt,en,es}/{mochi,nori}/Podcast/_index.md`, com `blup: podcast`, `outputs: [html, podcast]` e as propriedades do programa.
- `AddSection` aceita o tipo só quando ele está em `sectionKinds` do modelo. Num blog com menu próprio (propriedade `menu` da nota inicial), acrescenta a pasta ao menu; sem menu próprio o tema já lista todas as pastas.
- `internal/obsidian` reconhece a seção pela marca `blup: podcast` no front matter.

Propriedades da nota (todas opcionais):

| Propriedade | Uso | Sem valor |
| --- | --- | --- |
| `podcast_title` | Nome do programa no feed | Nome do site |
| `description` | Texto da página e do feed | Descrição do site |
| `podcast_author` | `itunes:author` e dono | Autor do site |
| `podcast_email` | `itunes:owner`, pelo qual Spotify e Apple confirmam a posse; fica público no feed | Sem `itunes:owner` |
| `podcast_category` | `itunes:category`, nome da lista da Apple em inglês | `Technology` |
| `podcast_explicit` | `itunes:explicit` | `false` |
| `image` | Capa do programa | Capa do episódio mais novo, depois o logo do site |

## Episódios

`theme/layouts/_partials/podcast/episodes.html` lista os posts do idioma da página com `audio`, do mais novo para o mais antigo. Ficam de fora notas com `hidden`, rascunhos e posts protegidos (`password`), porque o áudio entregaria o texto. A capa de cada episódio é a capa do post (`mochi/cover.html`: a propriedade `image` ou a primeira imagem da nota).

## Página

- `mochi/section.html` e `nori/section.html` desviam para o podcast quando `podcast/is.html` é verdadeiro. Cada modelo mantém o próprio cabeçalho: Mochi com breadcrumb e `.h1`, Nori com `.nori-head` na coluna estreita.
- `podcast/page.html` desenha a lista, o botão de copiar o feed, o "Carregar mais episódios" (dez por vez) e o player fixo. As capas usam `mochi/picture.html` com proporção `1:1`, WebP e `sizes` fixo de 112 px.
- `assets/css/podcast.css` é carregado só nessa página (`head/css.html`), com os tokens do tema (`--main-*`, e `--panel`/`--ring` do Nori quando existem). O `<head>` também ganha o `<link rel="alternate">` do feed.
- Na prévia do Studio sem suporte a MP3 (`HUGO_BLUP_NO_MP3`), a página mostra os episódios sem player, como `listen.html`.

## Player (`assets/js/podcast.js`)

- Um único `<audio>` toca todos os episódios. Ao terminar um, o próximo entra no mesmo elemento: celulares continuam tocando com a tela apagada, e o iPhone só passa sozinho para o próximo quando o elemento é o mesmo.
- A Media Session publica título, nome do site e capa (JPEG quadrado de 512 px, `podcast/art.html`) e trata play, pausa, anterior, próximo, avançar, voltar e posição.
- Controles: tocar e pausar, 15 s para trás e para frente, barra de posição, tempo, velocidade (1, 1,25, 1,5, 2 e 0,75), sem som e a lista de reprodução.
- As notas do episódio buscam a página do post e copiam o texto (`.prose`) para dentro do episódio, sem sair da página; o áudio continua.
- O episódio e o ponto onde o leitor parou ficam no `localStorage` do aparelho (`blup-podcast:<caminho>`); ao voltar, o player aparece pausado ali.

## Feed

`internal/scaffold` declara o formato de saída `podcast` (`application/rss+xml`, `feed.xml`). A nota da seção pede esse formato em `outputs`, e `theme/layouts/section.podcast.xml` gera um feed por idioma do site (`/podcast/feed.xml`, `/en/podcast/feed.xml`).

- Programa: título, link, descrição, idioma, `atom:link`, `itunes:author`, `itunes:owner`, `itunes:image`, `itunes:category`, `itunes:explicit` e `itunes:type episodic`.
- Episódio: título, link do post, `guid` estável (`translationKey`, senão o endereço), data, descrição, `enclosure` com o MP3 (tamanho real em bytes e `?v=` com o `audio_hash`), `itunes:duration` (`audio_seconds`) e `itunes:image`.
- As capas do feed são JPEG quadrados de 1400 px, recortados com `Fill ... smart`: a Apple aceita JPEG ou PNG de 1400 a 3000 px, não WebP.
- Os endereços só saem completos quando o site tem domínio (`blup.toml` `domain`, que vira `baseURL`). Sem domínio o feed sai com caminhos relativos e os apps não o aceitam.

## Tela Podcast no Studio

`PodcastPanel.svelte` aparece no menu quando o blog tem a seção. Usa `Podcast()` e `SavePodcast()` (`studio/podcast.go`, `internal/app/podcast.go`):

- Nome, descrição, autor, e-mail de contato, categoria (lista da Apple traduzida, valor em inglês), conteúdo explícito e capa (`PickCover`, com aviso quando a imagem não é um quadrado de 1400 a 3000 px).
- Quantos episódios há: posts do idioma principal com narração que o podcast lista.
- O endereço do feed de cada idioma, com Copiar e os atalhos para Spotify for Creators e Apple Podcasts Connect. Sem domínio, a tela explica que é preciso publicar com um.
- `SavePodcast` só grava as propriedades da lista `PodcastProps` e recusa e-mail sem `@`.
