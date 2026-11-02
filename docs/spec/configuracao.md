# Configuração e persistência

Fontes principais: [tipos e leitura TOML](../../internal/config/config.go), [configuração portátil](../../internal/app/cloud_sites.go), [preferências do Studio](../../studio/layout.go), `internal/app/settings.go` e `internal/app/site_notes.go`.

## Locais e responsabilidades

O diretório global é `filepath.Join(os.UserConfigDir(), "blup")`. Em Linux normalmente é `~/.config/blup`, respeitando a resolução feita por Go. Os caminhos de cache usam `os.UserCacheDir()` e não esse diretório.

| Arquivo | Conteúdo | Escrita |
| --- | --- | --- |
| `<site>/blup.toml` | Configuração técnica e destino deste site | TOML, modo `0644`, substituição por arquivo temporário e rename |
| `<config>/blup/config.toml` | Idioma e sites conhecidos | TOML, `0644`, escrita atômica |
| `<config>/blup/credentials.toml` | Tokens e senha geral entre sites | TOML, `0600`, escrita atômica |
| `<config>/blup/ai-consent.json` | Aceite explícito para envio de texto a `openrouter` e `blup` | JSON, `0600`, escrita atômica; arquivo inválido não autoriza |
| `<config>/blup/signout-pending` | Sessão de trabalho para cancelar lembretes após sair sem rede | `0600`; removido quando a pendência termina |
| `<config>/blup/studio-prefs.json` | Preferências globais e navegação do Studio | JSON indentado, `0644` |
| `<config>/blup/studio.json` | Ordem e agrupamentos de sites na tela inicial | JSON indentado, `0644` |
| `<config>/blup/reminders-sent.json` | Registro local dos avisos já exibidos | JSON, `0600` |
| `<config>/blup/reminder-key` | Chave aleatória local de 32 bytes para IDs HMAC de lembretes | Binário, `0600`; não é sincronizado |
| `<notas>/_index.md` | Identidade, menu e aparência do site | Propriedades YAML e corpo Markdown |
| `<notas>/.blup/site.toml` | Parte portátil do site para sync | TOML, `0644`; só reescrito quando o conteúdo muda |
| `<site>/.blup/sync-state.json` | Base da última sincronização | JSON, escrita atômica |
| `<site>/data/blup_feedback.json` | Relação entre notas e issues | JSON gerenciado pelo build de feedback |
| `<site>/hugo.toml`, `<site>/hugo.preview.toml` | Configuração gerada de publicação e prévia | TOML, `0644`, sobrescrito pelo scaffold |
| `<vault>/.obsidian/app.json` | Preferência de links Markdown relativos | JSON; ajuste opcional, com cópia anterior em `app.json.blup-backup` quando o original existe |
| `<vault>/.obsidian/templates.json`, `<vault>/.obsidian/core-plugins.json` | Pasta de templates e ativação do plugin | JSON; `core-plugins.json` ausente não é criado por essa rotina |
| `<cache>/blup/update.json` | Data, versão, notas e decisão da consulta de atualização | JSON, `0644` |
| `~/.config/hypr/hyprland.lua` ou `.conf` | Detecção de Omarchy | Leitura; caminho baseado em `UserHomeDir`, sem usar `XDG_CONFIG_HOME` |
| `~/.config/hypr/bindings.lua` ou `.conf` | Atalho global no Omarchy | Escrita da linha marcada do Studio e reload do Hyprland; ignora `XDG_CONFIG_HOME` |
| `$XDG_CONFIG_HOME/autostart/blup-studio.desktop` | Autostart Linux | Gerenciado via Wails; a consulta e a correção usam `~/.config` como fallback |

O pacote config cria seu diretório global com `0700`. O Studio também pode criá-lo por seus próprios métodos com `0755`. O modo de `blup.toml` não se torna privado por conter `password`; a proteção `0600` pertence especificamente ao arquivo de credenciais.

## `blup.toml`

`SiteVersion` é 1. `Save` define a versão como 1. `LoadSite` rejeita versão maior e modelo fora de `mochi`, `nori`, `bento`. O comentário do tipo prevê migrações futuras, mas a leitura atual não implementa rotina de migração ou backup de versões antigas. Também não valida todos os campos, como existência do cofre ou idioma.

| Campo | Tipo | Uso atual |
| --- | --- | --- |
| `version` | inteiro | Versão do formato |
| `title` | texto | Nome e fallback de identidade do site |
| `description` | texto | Descrição padrão |
| `author` | texto | Autor padrão, usado no cascade e em fallback do tema |
| `lang` | texto | Idioma do site: `pt`, `en` ou `es`, independente da interface |
| `translations` | lista de texto | Outros idiomas em que o site é publicado (`en`, `es`, `pt`); a tradução de cada nota fica numa pasta com o código do idioma ao lado dela. A tradução com IA acrescenta o idioma aqui |
| `model` | texto | Tema `mochi`, `nori` ou `bento` |
| `kind` | texto opcional | Vazio para site; `notes` para cofre de anotações, com editor e sync, sem prévia nem publicação |
| `vault` | texto | Caminho da raiz das notas/cofre |
| `folder` | texto | Subpasta publicada; vazio significa a raiz inteira |
| `domain` | texto opcional | Host de domínio próprio; a URL de build usa `https://<domain>/` |
| `home` | texto opcional | Bento: nome da pasta da seção exibida na home; vazio usa padrão (cartões, ou a única seção) |
| `logo_link` | texto opcional | URL HTTP/HTTPS ou caminho do site iniciado por uma `/`; muda o destino do logo/nome no cabeçalho dos três modelos |
| `password` | texto opcional | Senha geral deste site para notas marcadas com senha geral |

`dir` é estado privado da estrutura Go, derivado do diretório passado a `LoadSite` ou `NewSite`. Não é serializado. `ContentDir()` retorna `filepath.Join(vault, folder)`.

| Tabela e campo | Tipo | Uso |
| --- | --- | --- |
| `[deploy].provider` | texto | `blup`, `github` ou `cloudflare` |
| `[deploy.blup].id`, `.slug`, `.url` | texto | Identidade e endereço nativo do site em `getblup.site` |
| `[deploy.blup].base` | texto | BaseURL da última publicação; domínio próprio elegível ou endereço nativo; usada para detectar republicação |
| `[deploy.cloudflare].account` | texto | ID da conta |
| `[deploy.cloudflare].project` | texto | Nome do projeto Pages |
| `[deploy.cloudflare].subdomain` | texto | Host real retornado pelo Pages, inclusive eventual sufixo |
| `[deploy.github].owner` | texto | Dono do repositório de publicação |
| `[deploy.github].repo` | texto | Repositório da saída estática |
| `[sync].owner`, `[sync].repo` | texto | Repositório de notas, separado do deploy |
| `[sync].remote` | texto | `blup` seleciona sync cifrado pelo Blup Cloud no Studio; vazio mantém GitHub |
| `[sync].vault` | texto | ID do cofre remoto no Blup Cloud |
| `[feedback].enabled` | booleano | Habilita apresentação de feedback |
| `[feedback].owner`, `[feedback].repo` | texto | Repositório público de issues |
| `[feedback].api` | texto opcional | URL alternativa do serviço de login |
| `[feedback].app` | texto opcional | Slug alternativo do GitHub App |

`Feedback.On()` exige `enabled`, `owner` e `repo`. A API padrão é `https://blup-api.grupovought.workers.dev`; o App padrão é `blup-comments`. A presença desses campos não prova que o App está instalado: o build consulta isso antes de colocar a API nos dados do tema.

| Campo de `[cookies]` | Tipo | Uso |
| --- | --- | --- |
| `banner` | booleano | Pede consentimento antes de ferramentas e vídeos |
| `privacy` | texto | URL explícita apenas do link da política no banner; só é passada ao tema com `banner = true` |
| `google_analytics` | texto | ID de medição Analytics |
| `google_tag_manager` | texto | ID de contêiner GTM |
| `meta_pixel` | texto | ID do Pixel |
| `umami` | texto | ID do website Umami |
| `umami_script` | texto | Script próprio; vazio usa `https://cloud.umami.is/script.js` |
| `plausible` | texto | Domínio do site para Plausible |
| `plausible_script` | texto | Script próprio; vazio usa `https://plausible.io/js/script.js` |

Cada ferramenta é ativada por ID não vazio. `Tools()` não depende do banner. `SaveCookies` valida IDs com os padrões de `internal/app/cookies.go`, exige HTTP/HTTPS para URLs e pode criar uma nota de privacidade. Sem banner, o JavaScript carrega as ferramentas imediatamente. No banner sem URL explícita, o tema procura a nota `blup: privacy`; nos rodapés usa sempre essa nota, sem consultar `cookies.privacy`. A descoberta em `site/privacy.html` exige o valor `privacy` com essa caixa exata.

Exemplo mínimo de site já criado:

```toml
version = 1
title = "Meu blog"
description = "Notas públicas"
author = "Maria"
lang = "pt"
model = "nori"
vault = "/home/maria/Notas"
folder = "Blog"

[deploy]
provider = "github"

[deploy.github]
owner = "maria"
repo = "meu-blog"
```

`SaveHome` aceita apenas Bento e uma pasta direta de seção reconhecida (`docs`, `roadmap`, `changelog`, `help`, `blog`, `feedback`), ou vazio para voltar ao padrão. Guarda o nome da pasta, não o tipo ou ID: renomear/remover a pasta invalida a escolha e o tema usa o padrão. Com uma só seção, a home a mostra automaticamente. A escolha segue em `.blup/site.toml`.

`SaveLogoLink` apara espaços e aceita somente HTTP/HTTPS com host e sem credenciais na URL, ou caminho começando por uma única `/`. Recusa espaços/controles, barras invertidas, caracteres perigosos e mais de 2.000 caracteres. O frontend valida antes de salvar, e o Go valida novamente. Vazio volta à home do idioma; caminho segue a base do site, incluindo `/repo/` do GitHub Pages. O clique usa a mesma aba. O tema e a leitura portátil ignoram valor inválido em vez de transformá-lo em link ativo.

## Estado global e credenciais

`config.toml` contém `lang` e `sites`, lista de caminhos dos diretórios técnicos. `AddSite` retira duplicata exata e põe o caminho no início. `ExistingSites` filtra pela existência de `blup.toml`; não tenta validar cada arquivo. Ausência de arquivo global ou de credenciais produz estrutura vazia, sem erro. TOML inválido pode impedir o carregamento.

```toml
lang = "pt"
sites = ["/home/maria/Sites/meu-blog"]
```

`credentials.toml` tem os campos opcionais `cloudflare_token`, `github_token`, `openrouter_token`, `post_password`, `blup_token` e `notes_key`. Os dois últimos guardam a sessão Bearer do Studio no Blup Cloud e a chave mestra de notas em base64, já aberta neste computador. Base64 não cifra o arquivo local. Os clientes usam as chaves salvas aqui; não há leitura geral de `GITHUB_TOKEN` ou de tokens de provedor por variável de ambiente nos fluxos da aplicação. O [modelo de segurança](../seguranca.md) distingue essa persistência local da cifra usada no transporte das notas.

A senha geral é resolvida por `app.PostPassword`: primeiro `Site.Password`, depois `Credentials.PostPassword`. Uma senha literal em uma nota substitui essa escolha para aquele conteúdo. O arquivo portátil não recebe nenhuma dessas senhas gerais, nem tokens.

## Preferências do Studio

`Prefs()` começa com padrões e decodifica numa cópia. Falha de leitura ou JSON danificado devolve os padrões completos, sem aproveitar um campo parcialmente decodificado. A leitura e `SavePrefs` normalizam enums e faixas: fonte do editor 13 a 22, terminal 11 a 20, largura lateral 0 a 4.000 e prévia 0 a 8.000; valores fora de faixa voltam ao padrão. `SavePrefs` mantém a lista `recent` do estado em disco para não substituí-la com uma cópia antiga da tela. As preferências são globais à máquina.

| Chave JSON | Tipo | Padrão inicial e significado |
| --- | --- | --- |
| `theme` | texto | `system`; também `light` ou `dark` |
| `editorFont` | inteiro | 16 px |
| `editorWidth` | texto | `narrow`; alternativa `wide` |
| `spellcheck` | booleano | `true` |
| `previewOff` | booleano | `false`; abre sites com a prévia ligada |
| `termFont` | inteiro | 13 px |
| `spellLang` | texto | Vazio acompanha a interface; dicionários `pt`, `en`, `es` |
| `words` | lista de texto | Palavras adicionadas ao dicionário |
| `sideWidth` | inteiro | 0 usa largura padrão do painel de arquivos |
| `previewWidth` | inteiro | 0 divide o espaço disponível |
| `tray` | booleano | `false`; ativa permanência na bandeja |
| `toggleKey` | texto | `Alt+Space`; vazio remove o atalho global |
| `keys` | mapa | Alterações nos atalhos de ações do Studio; valor vazio desativa uma ação |
| `shortcutTip` | booleano | Indica que a dica do atalho global foi respondida |
| `lastSite` | texto | Último diretório técnico aberto |
| `lastNotes` | mapa | Diretório técnico para última nota aberta |
| `recent` | lista de texto | Até 12 sites recentes, inclusive os retirados da lista |
| `voices` | mapa | Voz de narração de cada idioma (`pt`, `en`, `es`), para todos os sites; a chave é o idioma da nota resolvido por `NoteOrigin`, incluindo traduções |
| `welcomed` | booleano | Telas iniciais já foram vistas |
| `cloudTip` | booleano | O convite ao Blup Cloud depois do primeiro site foi respondido |
| `backupDir` | texto | Pasta da cópia diária das notas; vazio desliga |
| `translateAI` | texto | Quem traduz: `local`, `openrouter` ou `blup`, escolhidos explicitamente; vazio acompanha a opção local, sem escolher provedor remoto por disponibilidade |
| `railNames` | booleano | `false`; mostra nomes junto dos ícones da faixa lateral quando ativado |
| `notifications` | booleano | `true`; avisos locais e lista de lembretes por e-mail dos planos Max/Ultra |

Voz local usa `kokoro:<id>`, voz pelo plano Blup Cloud usa `blup:<modelo>|<voz>` e voz OpenRouter usa `<modelo>|<voz>`, inclusive voz vazia para o padrão do modelo. A forma antiga `blup:<voz>` usa o modelo recomendado. Sem escolha remota explícita, a resolução fica local; faltar instalação não autoriza enviar texto à nuvem. Nomes antigos sem separador são tratados como voz do modelo remoto recomendado. A resolução por `voiceFor` e os fallbacks de idioma estão em [narração e voz](narracao-e-voz.md#idioma-da-nota-e-escolha-de-voz).

`studio.json` é uma lista de `HomeItem`: item simples `{ "site": "/caminho" }` ou grupo `{ "folder": "Pessoais", "sites": ["/caminho"] }`. `Layout()` remove referências indisponíveis e duplicadas e acrescenta os sites conhecidos ainda não posicionados. Agrupamentos não são pastas físicas.

## Propriedades das notas

`internal/obsidian/props.go` lê YAML delimitado por `---`, fornece `String`/`List` e altera propriedades preservando as demais, sua ordem e o corpo. Valor `nil` remove a chave. `studio/notes.go` mantém corpo e edições de propriedade separados; o formulário converte valores vazios, `false` e zero para remoção da propriedade.

| Nota ou propriedade | Uso |
| --- | --- |
| `_index.md` na raiz publicada | Nota inicial e identidade do site |
| `site_name`, `site_description` | Precedem `title`/`description` de `blup.toml` na identidade do tema |
| `logo`, `favicon` | Referência a imagem do cofre, incluindo `[[arquivo.png]]` |
| `menu` | Lista de `Destino` ou `Nome \| Destino`; resolve pastas, notas, home, `/caminho`, HTTP/HTTPS ou `mailto:`. Submenu: `+ ` na frente põe a linha no submenu da linha de cima, que então só abre o submenu, sem link (o Studio grava essa linha como `Nome \|`). Um nível só |
| `button_text`, `button_link` | Botão do cabeçalho |
| `social` | Lista de URLs ou emails usada em redes sociais |
| `title`, `description` | Título e descrição da página |
| `linkTitle` | Nome curto da seção em cartões e menus; incluído na tradução da nota |
| `date`, `publishDate`, `created` | Datas consideradas pelo scaffold; fallback varia por modelo |
| `lastmod`, `updated`, `modified` | Atualização, com fallback ao mtime |
| `tags`, `authors` | Taxonomias; autor padrão vem da configuração |
| `image`, `cover` | Capa, resolvida por `mochi/cover.html` |
| `featured` | Seleção de destaque em Mochi/Nori |
| `weight` | Ordenação de seções e páginas Bento |
| `draft` | Incluído na prévia; omitido no build normal de publicação |
| `publish: false`, `private: true` | Tratados como não publicados no build normal, incluindo traduções relacionadas; aparecem na prévia |
| `hidden` | Retira de listas filtradas, RSS e marcação de busca; mantém URL da página |
| `noindex` | Emite meta robots noindex; não é um controle de acesso |
| `password` | Literal, ou `true`, `sim`, `sí`, `si`, `yes` para senha geral |
| `audio`, `audio_hash`, `audio_seconds` | Referência, revisão do texto e duração da narração |
| `blup` em `_index.md` de pasta direta | `docs`, `roadmap`, `changelog`, `help`, `blog`, `feedback`, `podcast` ou `authors`; Feedback e Podcast são oferecidas em Seções no Studio, nos modelos compatíveis |
| `blup: privacy` | Identifica a política de privacidade |
| `status`, `category` | Estado e categoria do roadmap |
| `sort_by_votes: false` | Mantém a ordem do roadmap sem ordenar por votos |
| `examples` em `blup: feedback` | Lista de pedidos de exemplo: `title`, `kind`, `status`, `votes`, `days`, `body` e `replies` com `author`, `days`, `text`; não cria issues |
| `translationKey` | Identidade compartilhada entre original e traduções para ligar páginas no Hugo |
| `translation_hash` | Impressão da original usada para indicar tradução desatualizada |
| `ai_translated` | Marca de tradução com IA, removida ao marcar como revisada; permite o selo do tema |

Perfis ficam na pasta com `blup: authors`, normalmente `Autores` ou `Writers`. Cada arquivo leva o nome do autor e usa `photo`, `role`, `social`; o corpo é a bio. A tradução mantém o nome em `<autores>/<idioma>/<Nome>.md` e traduz cargo e bio. A pasta, incluindo essas traduções, é montada como recurso e excluída das páginas, enquanto `/authors/` é a taxonomia de posts.

No Feedback, `status` dentro de `examples` é apenas o estado do exemplo. Pedidos reais têm status nas etiquetas GitHub `status: planned`, `status: progress`, `status: done` ou `status: declined`; não há nota por pedido. Categorias e opções de status são fixas no código, sem configuração pelo Studio. O [contrato do quadro](feedback.md) detalha esses campos e o que é traduzido.

O vínculo Blup Cloud fica em `site_domain`, com `disableAt` no contrato HTTP. Não é gravado em `Site.Domain` ou `Deploy.Blup.URL`; o publish registra a base usada em `Deploy.Blup.Base` e o Studio pode republicar ao mudar o endereço oficial. `Deploy` e `Feedback` ficam fora do TOML portátil; `translations`, `home` e `logo_link` viajam. Link inválido vindo do TOML portátil é ignorado. O contrato portátil de `.blup/site.toml`, o estado do sync e os caches estão detalhados em [notas e sync](notas-e-sync.md) e [scaffold e build](scaffold-e-build.md).

Os campos `podcast_*`, `outputs` e `audio*` do programa e seus episódios estão em [Podcast](podcast.md). Eventos de cofre usam `date`, `time`, `event`, `repeat` e `remind`, conforme [Calendário e lembretes](calendario-e-lembretes.md). A preferência `blup-lang` do visitante fica no `localStorage` do site publicado, sem alterar `config.toml` ou as preferências do Studio.

As configurações de servidor, bindings D1/R2, planos e nomes de secrets do Blup Cloud estão em [Blup Cloud](blup-cloud-sync.md) e no [guia de deploy](../deploy.md). Elas pertencem ao operador do serviço e não a `blup.toml`.

## Variáveis de ambiente e flags de processo

| Variável | Leitura ou efeito |
| --- | --- |
| `BLUP_NO_SEARCH` | Qualquer valor não vazio pula Pagefind no build de publicação |
| `BLUP_NO_UPDATE_CHECK` | Qualquer valor não vazio pula a oferta automática de atualização da CLI |
| `BLUP_VOICE_TOOLS` | Lida ao iniciar o processo; troca a origem pelo mesmo pacote de voz local, mantendo hash e tamanho fixos da tabela |
| `BLUP_CLOUD_URL` | Substitui `https://cloud.getblup.io` no cliente Go, para desenvolvimento/testes; produção deve usar a origem correta |
| `HUGO_BLUP_MODE` | Definida pelo Blup como `preview` ou `publish`; o tema decide exposição do corpo, manifestos e PWA |
| `HUGO_BLUP_NO_MP3` | Definida pelo Studio para omitir player na prévia sem suporte detectado |
| `LANG` | Fallback do idioma inicial do Studio |
| `SHELL` | Shell do terminal do Studio em sistemas não Windows |
| `HYPRLAND_INSTANCE_SIGNATURE` | Detecta sessão Hyprland em Linux; o Studio lê os arquivos `~/.config/hypr/hyprland.lua` ou `.conf` para reconhecer Omarchy e, nesse caso, escreve o atalho nos bindings |
| `GST_PLUGIN_PATH` | Diretórios adicionais de detecção dos parsers de áudio |
| `XDG_CONFIG_HOME`, `XDG_DATA_HOME` | Afetam resolução Linux de configuração/autostart e lixeira |
| `NO_COLOR` | Pula a animação de piscar do splash da CLI conforme `internal/ui/splash.go` |
| `WAILS_VITE_PORT` | Porta do frontend de desenvolvimento |
| `BLUP_RELEASE_KEY` | Somente CI/ferramenta `scripts/sign-release`: chave privada de assinatura fora do repositório; não é configuração de site ou Studio |
| `BLUP_INSTALL_DIR`, `BLUP_VERSION` | Destino e versão dos instaladores CLI shell/PowerShell |

As variáveis `BLUP_KEEP`, `BLUP_STARTER_OUT` e `BLUP_KOKORO_DIR` são auxiliares de testes, descritas em [testes e build](testes-e-build.md). `--hidden` e `--toggle` são argumentos do executável Studio, tratados pela integração desktop; não são chaves TOML.
