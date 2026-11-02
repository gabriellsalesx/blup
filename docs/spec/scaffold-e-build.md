# Scaffold, ferramentas e build

Fontes: [scaffold](../../internal/scaffold/scaffold.go), [Hugo](../../internal/hugo/hugo.go), [pipeline](../../internal/app/hugo.go), `internal/app/preview.go`, `internal/pagefind/pagefind.go` e `studio/studio.go`.

## Arquivos gerados

`embed.go` inclui `all:theme` no executável. `scaffold.Write` cria `layouts/`, `assets/`, `static/`, `i18n/` e `data/`; apaga e reconstrói `<site>/themes/blup` com os arquivos embutidos; grava `hugo.toml` e `hugo.preview.toml`. Alterações nessas configurações e na cópia do tema são substituídas a cada scaffold.

Os diretórios de overrides são preservados. O tema pode ser substituído parcialmente por layouts, recursos, traduções e dados nesses diretórios. Isso é diferente de editar `theme/` no próprio código do Blup, que exige reconstruir o executável para mudar seu conteúdo embutido.

## Configuração Hugo

`HugoConfig(site, preview)` define:

- URL base `/` ou domínio próprio; título, `locale` `pt-BR`/`en-US`, idioma e `theme = "blup"`.
- Remoção de acentos em caminhos, aliases desativados, robots habilitado, taxonomias `tags` e `authors`, paginação de 10 itens.
- Datas na ordem `date`, `publishDate`, `created`, nome do arquivo e mtime para blogs. Bento usa somente as três propriedades, evitando reordenar documentação por mtime.
- `lastmod` por `lastmod`, `updated`, `modified`, mtime.
- Goldmark com quebra de linha compatível com a configuração `strictLineBreaks` do Obsidian, atributos de bloco e título, `==marca==` e passthrough de `$...$`/`$$...$$`.
- Sumário com níveis H2 a H3. Autor padrão em cascade para páginas, quando configurado.
- Parâmetros do modelo, descrição, autor, pasta publicada, pasta de perfis, feedback e cookies. Senhas e tokens não são parâmetros Hugo do scaffold.

Os mounts usam os seguintes contratos:

| Origem | Destino Hugo | Seleção |
| --- | --- | --- |
| `site.ContentDir()` | `content` | Markdown; exclui pastas ocultas, guias de agentes na raiz, perfis e templates configurados |
| `site.Vault` | `assets/vault` | Anexos conhecidos do cofre inteiro, perfis de autores e suas traduções |
| `<site>/assets` | `assets` | Recursos de override |
| `<site>/layouts` | `layouts` | Templates de override |
| `<site>/static` | `static` | Cópia estática direta |
| `<site>/i18n` | `i18n` | Traduções de override |
| `<site>/data` | `data` | Dados locais, inclusive feedback |
| `<site>/.blup/preview` | `static` | Somente em `hugo.preview.toml` |

Os anexos elegíveis são `png`, `jpg`, `jpeg`, `webp`, `avif`, `gif`, `svg`, `bmp`, `tif`, `tiff`, `pdf`, `mp3`, `m4a`, `ogg`, `wav`, `flac`, `mp4`, `webm`, `mov` e `zip`, com padrões para extensão minúscula e maiúscula. Montá-los como assets permite publicar apenas os recursos efetivamente referenciados. Um arquivo em `static/` tem outro comportamento: ele entra como arquivo estático.

`AGENTS.md`, `CLAUDE.md` e `GEMINI.md` na raiz das notas são guias de terminal excluídos do conteúdo. `obsidian.Sections` descobre a pasta de autores pelas propriedades do `_index.md` de pastas diretas. O diretório de templates vem de `.obsidian/templates.json`, ajustado em relação ao recorte publicado.

Quando há traduções, `translationMounts` monta cada pasta `en`, `es` ou `pt` correspondente a outro idioma no lugar da pasta mãe, somente na versão daquele idioma, por `sites.matrix`. A pasta de autores é excluída desse percurso: seus perfis e traduções entram em `assets/vault` pelos padrões `<autores>/*.md` e `<autores>/*/*.md`, sem virar páginas de conteúdo. `site/author-profile.html` escolhe o perfil traduzido ou o original. `linkTitle` traduzido no índice fornece o nome curto da seção para os cartões e o menu.

No modo normal, os mounts excluem notas com `publish: false` ou `private: true` e suas traduções relacionadas, além do tratamento de rascunhos. A configuração de prévia permite ver esse conteúdo.

O scaffold inclui o tema do [quadro de Feedback](feedback.md), mas não transforma `examples` em páginas ou issues individuais. Hugo gera cartões e detalhes `#ex-N` no HTML da seção. Pedidos reais são lidos pelo navegador; mudar seu status não exige outro build.

## Hugo e Pagefind

`internal/hugo` aceita um Hugo do PATH ou do cache com versão mínima `0.169.0`. A versão fixada para download também é `0.169.0`. Consulta `hugo version`, extrai `vX.Y.Z` e compara os componentes numéricos.

O cache é `<cache>/blup/hugo-0.169.0/hugo`, com `.exe` em Windows. Download usa GitHub Releases do Hugo, verifica SHA-256 contra o arquivo oficial de checksums, extrai somente o executável e troca um arquivo `.part` pelo destino. Plataformas de download: Linux amd64/arm64, Windows amd64/arm64 e FreeBSD amd64. macOS precisa de Hugo instalado por outro meio; encontrar um binário compatível no PATH continua funcionando.

Pagefind é fixado em `1.5.2`, com cache `<cache>/blup/pagefind-1.5.2/pagefind`, usando `pagefind.exe` em Windows. Verifica SHA-256 do pacote e indexa com `--site`, `--output-subdir pagefind`, `--quiet`. A tabela oferece Linux, macOS e Windows em amd64/arm64 e FreeBSD amd64. Não exige Node para executar esse binário.

## Build de publicação

`app.build` é usado por `build`, `publish` e pela publicação do Studio:

1. Garante Hugo compatível e verifica a existência da pasta das notas.
2. Executa `scaffold.Write`.
3. Executa Hugo no diretório técnico com `--gc --minify --cleanDestinationDir --environment production --baseURL <url> --destination <site>/public --logLevel warn`, e `HUGO_BLUP_MODE=publish`.
4. Extrai avisos e erros da saída. Falha de Hugo interrompe com erro de build.
5. Processa `blup-feedback.json`, cria/atualiza issues e dados locais. Se os dados usados pelo tema mudaram, executa Hugo novamente. Falha de feedback é aviso e não impede o restante; sem chave GitHub e sem terminal interativo, pula o sync de feedback com aviso `fb.no_token`, sem pedir chave.
6. Remove o manifesto de feedback e criptografa regiões protegidas usando `internal/lock`. Incorpora anexos locais pequenos no HTML protegido, remove `blup-locks.json` e executa `lock.Prune` para mídias sem referência pública. Verifica marcadores provisórios mesmo com manifesto vazio. Se essa etapa falhar, apaga `public/` e encerra.
7. Executa Pagefind depois da criptografia, exceto com `BLUP_NO_SEARCH` não vazio. Falha de busca gera aviso.
8. Executa `pwa.Write` depois da busca, para o worker considerar a saída final. Falha de PWA também gera aviso.

O build não acrescenta `--buildDrafts`. Sem domínio ou URL de deploy informada, `buildCommand` usa `/`. A cópia opcional de saída é descrita em [CLI](cli.md).

`encryptLocked` consulta o manifesto para saber se precisa de senha geral. Se não houver senha salva e a UI for interativa, pede e salva a senha global. Sem interatividade, devolve erro. A senha específica do site tem precedência sobre a global.

Os modos são determinados por variável de ambiente, não apenas por `--environment production`. Com Hugo puro sem `HUGO_BLUP_MODE`, o tema omite o corpo protegido. Com `HUGO_BLUP_MODE=publish`, Hugo produz regiões provisórias em claro e manifesto com senhas; somente o pipeline Go substitui e remove esses dados.

## Prévia da CLI e do Studio

| Aspecto | CLI | Studio |
| --- | --- | --- |
| Configuração do servidor | `hugo.preview.toml` | `hugo.preview.toml` |
| Corpo de post protegido | Legível, com aviso | Legível, com aviso |
| Rascunhos | `--buildDrafts` | `--buildDrafts` |
| Destino principal | Memória | Memória |
| Atualização | Live reload e `--navigateToChanged` do Hugo | `--disableLiveReload`; evento e troca de iframe |
| Rolagem dentro de iframe | Script de `postMessage` disponível em modo preview | `loaded`, `scroll`, `ready` e `scrollTo` conservam posição ao trocar o iframe; não sincroniza por linha |
| Log | `warn` | `info`, para detectar `Total in` |
| Busca | Build temporário e índice ao iniciar | `StartPreview` não cria índice Pagefind |
| MP3 | Sem verificação do WebView | Pode definir `HUGO_BLUP_NO_MP3=1` |
| PWA | Sem registro do worker pelo tema | Sem registro do worker pelo tema |

A CLI usa um diretório temporário para montar o índice e move `pagefind/` para `<site>/.blup/preview/pagefind`. Seu build temporário usa a configuração padrão, `--buildDrafts`, base local e destino temporário. O índice não acompanha cada salvamento. `BLUP_NO_SEARCH` é verificado no build de publicação; `previewSearch` não consulta essa variável.

A rotina `placeSearch`, em `internal/app/preview.go`, remove o índice anterior e chama `moveDir` para levar `pagefind/` do temporário ao site. `moveDir`, em `internal/app/settings.go`, tenta rename e, se ele falhar, usa cópia e remoção da origem; assim aceita temporário e site em sistemas de arquivos diferentes. Uma falha restante emite aviso e a prévia continua.

`PreviewGuard`, em `internal/app/preview.go`, serve a prévia por proxy `127.0.0.1` em porta aleatória e encaminha para outra porta interna do Hugo. Valida Host exato (`127.0.0.1:P` ou `localhost:P`) e Origin, quando presente, com origens da prévia e do aplicativo. A CLI não acrescenta origens de janela desktop. O encerramento fecha proxy e Hugo.

O Studio retorna uma prévia já em execução se houver processo registrado. Para iniciar, espera a URL até dois minutos. Após a URL, linhas com `Total in` ou `Rebuilt in` geram `previewChanged`, incluindo mudanças de configuração. Em Linux, ausência de plugin de áudio pode ocultar o player apenas nessa prévia.

Ao adicionar, remover ou restaurar uma seção pelo Studio, `sectionsChanged` solicita reinício do servidor de prévia em execução. A cópia de uma pasta inteira pode terminar antes de o watcher Hugo começar a observá-la, deixando notas sem página mesmo após recarregar o iframe. `restartPreview` emite `previewChanged` se a URL permanece igual ou `previewRestarted` se mudou; o workspace volta à página da nota aberta. Para traduções, `previewFollows` compara a configuração gerada com a que iniciou o servidor e reinicia quando idiomas ou mounts mudam. Os detalhes estão em [Studio](studio.md#prévia).

## Saídas auxiliares e mapa

| Saída | Quando e quem consome |
| --- | --- |
| HTML e RSS | Saída padrão da home |
| `blup-locks.json` | IDs e senhas para pós-processamento Go; descartado antes de publicar |
| `manifest.json` | Web app manifest; ícones e worker são completados por Go |
| `blup-feedback.json` | Apenas com feedback configurado; lista em modo publish; removida pelo build |
| `blup-map.json` | Formato habilitado apenas na configuração de prévia |

`theme/layouts/home.blupmap.json` serializa um array de `{ "file": "Pasta/Nota.md", "url": "/pasta/nota/" }` para `site.Pages` com arquivo de origem. `file` é relativo à pasta de conteúdo, com `/` entre segmentos. Inclui páginas de seção que tenham nota; páginas geradas sem arquivo não entram.

`Studio.PreviewPage` busca esse JSON no servidor local com timeout de cinco segundos, status HTTP válido e limite de 16 MiB. A leitura descarta URLs externas ou com controles e arquivos fora das notas, inclusive após links simbólicos, e compara o caminho de nota sem diferenciar maiúsculas. `Studio.NoteAt` faz o caminho inverso, decodificando URL e normalizando `/` final e `index.html`. A interface segue a nota editada e abre no editor a nota de uma página navegada. Taxonomias ou outras páginas sem nota não têm correspondência garantida.
