# Configuração e persistência

Fontes principais: [tipos e leitura TOML](../../internal/config/config.go), [configuração portátil](../../internal/app/cloud_sites.go), [preferências do Studio](../../studio/layout.go), `internal/app/settings.go` e `internal/app/site_notes.go`.

## Locais e responsabilidades

O diretório global é `filepath.Join(os.UserConfigDir(), "blup")`. Em Linux normalmente é `~/.config/blup`, respeitando a resolução feita por Go. Os caminhos de cache usam `os.UserCacheDir()` e não esse diretório.

| Arquivo | Conteúdo | Escrita |
| --- | --- | --- |
| `<site>/blup.toml` | Configuração técnica e destino deste site | TOML, modo `0644`, substituição por arquivo temporário e rename |
| `<config>/blup/config.toml` | Idioma e sites conhecidos | TOML, `0644`, escrita atômica |
| `<config>/blup/credentials.toml` | Tokens e senha geral entre sites | TOML, `0600`, escrita atômica |
| `<config>/blup/studio-prefs.json` | Preferências globais e navegação do Studio | JSON indentado, `0644` |
| `<config>/blup/studio.json` | Ordem e agrupamentos de sites na tela inicial | JSON indentado, `0644` |
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
| `lang` | texto | Idioma do site, normalmente `pt` ou `en`, independente da interface |
| `model` | texto | Tema `mochi`, `nori` ou `bento` |
| `vault` | texto | Caminho da raiz das notas/cofre |
| `folder` | texto | Subpasta publicada; vazio significa a raiz inteira |
| `domain` | texto opcional | Host de domínio próprio; a URL de build usa `https://<domain>/` |
| `password` | texto opcional | Senha geral deste site para notas marcadas com senha geral |

`dir` é estado privado da estrutura Go, derivado do diretório passado a `LoadSite` ou `NewSite`. Não é serializado. `ContentDir()` retorna `filepath.Join(vault, folder)`.

| Tabela e campo | Tipo | Uso |
| --- | --- | --- |
| `[deploy].provider` | texto | `github` ou `cloudflare` |
| `[deploy.cloudflare].account` | texto | ID da conta |
| `[deploy.cloudflare].project` | texto | Nome do projeto Pages |
| `[deploy.cloudflare].subdomain` | texto | Host real retornado pelo Pages, inclusive eventual sufixo |
| `[deploy.github].owner` | texto | Dono do repositório de publicação |
| `[deploy.github].repo` | texto | Repositório da saída estática |
| `[sync].owner`, `[sync].repo` | texto | Repositório de notas, separado do deploy |
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

## Estado global e credenciais

`config.toml` contém `lang` e `sites`, lista de caminhos dos diretórios técnicos. `AddSite` retira duplicata exata e põe o caminho no início. `ExistingSites` filtra pela existência de `blup.toml`; não tenta validar cada arquivo. Ausência de arquivo global ou de credenciais produz estrutura vazia, sem erro. TOML inválido pode impedir o carregamento.

```toml
lang = "pt"
sites = ["/home/maria/Sites/meu-blog"]
```

`credentials.toml` tem os campos opcionais `cloudflare_token`, `github_token`, `openrouter_token` e `post_password`. Os clientes usam as chaves salvas aqui. Não há leitura geral de `GITHUB_TOKEN` ou de tokens de provedor por variável de ambiente nos fluxos da aplicação.

A senha geral é resolvida por `app.PostPassword`: primeiro `Site.Password`, depois `Credentials.PostPassword`. Uma senha literal em uma nota substitui essa escolha para aquele conteúdo. O arquivo portátil não recebe nenhuma dessas senhas gerais, nem tokens.

## Preferências do Studio

`Prefs()` começa com valores padrão e depois decodifica o JSON sobre a estrutura. Falhas de leitura ou decode são ignoradas. `SavePrefs` mantém a lista `recent` do estado em disco para não substituí-la com uma cópia antiga da tela. As preferências são globais à máquina.

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
| `voices` | mapa | Idioma para voz de narração escolhida |
| `welcomed` | booleano | Telas iniciais já foram vistas |

Voz local usa `kokoro:<id>`. Voz OpenRouter usa `<modelo>|<voz>`, inclusive voz vazia para o padrão do modelo. Nomes antigos sem separador são tratados como voz do modelo remoto recomendado.

`studio.json` é uma lista de `HomeItem`: item simples `{ "site": "/caminho" }` ou grupo `{ "folder": "Pessoais", "sites": ["/caminho"] }`. `Layout()` remove referências indisponíveis e duplicadas e acrescenta os sites conhecidos ainda não posicionados. Agrupamentos não são pastas físicas.

## Propriedades das notas

`internal/obsidian/props.go` lê YAML delimitado por `---`, fornece `String`/`List` e altera propriedades preservando as demais, sua ordem e o corpo. Valor `nil` remove a chave. `studio/notes.go` mantém corpo e edições de propriedade separados; o formulário converte valores vazios, `false` e zero para remoção da propriedade.

| Nota ou propriedade | Uso |
| --- | --- |
| `_index.md` na raiz publicada | Nota inicial e identidade do site |
| `site_name`, `site_description` | Precedem `title`/`description` de `blup.toml` na identidade do tema |
| `logo`, `favicon` | Referência a imagem do cofre, incluindo `[[arquivo.png]]` |
| `menu` | Lista de `Destino` ou `Nome \| Destino`; resolve pastas, notas, home, `/caminho`, HTTP/HTTPS ou `mailto:` |
| `button_text`, `button_link` | Botão do cabeçalho |
| `social` | Lista de URLs ou emails usada em redes sociais |
| `title`, `description` | Título e descrição da página |
| `date`, `publishDate`, `created` | Datas consideradas pelo scaffold; fallback varia por modelo |
| `lastmod`, `updated`, `modified` | Atualização, com fallback ao mtime |
| `tags`, `authors` | Taxonomias; autor padrão vem da configuração |
| `image`, `cover` | Capa, resolvida por `mochi/cover.html` |
| `featured` | Seleção de destaque em Mochi/Nori |
| `weight` | Ordenação de seções e páginas Bento |
| `draft` | Incluído na prévia; omitido no build normal de publicação |
| `hidden` | Retira de listas filtradas, RSS e marcação de busca; mantém URL da página |
| `noindex` | Emite meta robots noindex; não é um controle de acesso |
| `password` | Literal, ou `true`, `sim`, `yes` para senha geral |
| `audio`, `audio_hash`, `audio_seconds` | Referência, revisão do texto e duração da narração |
| `blup` em `_index.md` de pasta direta | `docs`, `roadmap`, `changelog`, `help`, `blog` ou `authors` |
| `blup: privacy` | Identifica a política de privacidade |
| `status`, `category` | Estado e categoria do roadmap |
| `sort_by_votes: false` | Mantém a ordem do roadmap sem ordenar por votos |

Perfis ficam na pasta com `blup: authors`, normalmente `Autores` ou `Writers`. Cada arquivo leva o nome do autor e usa `photo`, `role`, `social`; o corpo é a bio. A pasta é montada como recurso e excluída das páginas, enquanto `/authors/` é a taxonomia de posts.

O contrato portátil de `.blup/site.toml`, o estado do sync e os caches estão detalhados em [notas e sync](notas-e-sync.md) e [scaffold e build](scaffold-e-build.md).

## Variáveis de ambiente e flags de processo

| Variável | Leitura ou efeito |
| --- | --- |
| `BLUP_NO_SEARCH` | Qualquer valor não vazio pula Pagefind no build de publicação |
| `BLUP_NO_UPDATE_CHECK` | Qualquer valor não vazio pula a oferta automática de atualização da CLI |
| `BLUP_VOICE_TOOLS` | Lida ao iniciar o processo; troca a origem pelo mesmo pacote de voz local, mantendo hash e tamanho fixos da tabela |
| `HUGO_BLUP_MODE` | Definida pelo Blup como `preview` ou `publish`; o tema decide exposição do corpo, manifestos e PWA |
| `HUGO_BLUP_NO_MP3` | Definida pelo Studio para omitir player na prévia sem suporte detectado |
| `LANG` | Fallback do idioma inicial do Studio |
| `SHELL` | Shell do terminal do Studio em sistemas não Windows |
| `HYPRLAND_INSTANCE_SIGNATURE` | Detecta sessão Hyprland em Linux; o Studio lê os arquivos `~/.config/hypr/hyprland.lua` ou `.conf` para reconhecer Omarchy e, nesse caso, escreve o atalho nos bindings |
| `GST_PLUGIN_PATH` | Diretórios adicionais de detecção dos parsers de áudio |
| `XDG_CONFIG_HOME`, `XDG_DATA_HOME` | Afetam resolução Linux de configuração/autostart e lixeira |
| `NO_COLOR` | Pula a animação de piscar do splash da CLI conforme `internal/ui/splash.go` |
| `WAILS_VITE_PORT` | Porta do frontend de desenvolvimento |
| `BLUP_INSTALL_DIR`, `BLUP_VERSION` | Destino e versão dos instaladores CLI shell/PowerShell |

As variáveis `BLUP_KEEP`, `BLUP_STARTER_OUT` e `BLUP_KOKORO_DIR` são auxiliares de testes, descritas em [testes e build](testes-e-build.md). `--hidden` e `--toggle` são argumentos do executável Studio, tratados pela integração desktop; não são chaves TOML.
