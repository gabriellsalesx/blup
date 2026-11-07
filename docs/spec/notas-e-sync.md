# Notas e sincronização

Fontes: [leitura de notas](../../internal/obsidian/obsidian.go), [propriedades](../../internal/obsidian/props.go), [wikilinks](../../internal/wikilinks/index.go), [sync](../../internal/notesync/notesync.go), [configuração portátil](../../internal/app/cloud_sites.go), `internal/app/sync.go` e `internal/app/deploy_api.go`.

## Compatibilidade da CLI com Obsidian

`obsidian.Vaults` lê `obsidian/obsidian.json` sob `os.UserConfigDir()`. Em Linux também procura os registros do Flatpak e Snap, em `~/.var/app/md.obsidian.Obsidian/config/obsidian/obsidian.json` e `~/snap/obsidian/current/.config/obsidian/obsidian.json`. Deduplica caminhos e inclui somente pastas reconhecidas por uma subpasta `.obsidian`. Dentro de cada registro prioriza os timestamps mais recentes.

`Folders` conta arquivos `.md` recursivamente, ignora pastas ocultas, inclui pastas vazias e retorna caminhos até a profundidade pedida. `CountNotes` também ignora pastas ocultas. Esses contadores não equivalem ao conjunto publicado: perfis, templates, rascunhos e notas escondidas têm tratamento posterior.

`EnableHugoFriendlyLinks` preserva as outras opções de `.obsidian/app.json`, guarda a cópia anterior em `app.json.blup-backup` e define `useMarkdownLinks = true` e `newLinkFormat = "relative"`. A configuração é para links futuros; não converte o texto existente. O fluxo informa que é preciso reabrir o Obsidian. `strictLineBreaks` é lido para ajustar as quebras de linha do Hugo.

No fluxo da CLI, os templates do plugin nativo são escritos na pasta configurada em `.obsidian/templates.json`. Se não houver pasta, Blup cria e configura uma. `SetTemplatesFolder` habilita o plugin no formato de lista ou mapa de `.obsidian/core-plugins.json`. Se esse arquivo não existe, não cria a lista de plugins.

## Notas e propriedades

O corpo e o front matter YAML permanecem em arquivos `.md`. `SetProperties` altera os valores das chaves solicitadas usando nós YAML, preserva as demais e sua ordem, mas reserializa todo o front matter com yaml.v3 e indentação de dois espaços; a formatação das outras chaves pode mudar. Valor `nil` remove a chave, e a nota é criada quando ausente. `SetBody` troca o corpo preservando o front matter. Não há banco de dados obrigatório para escrever notas.

`blup: docs`, `roadmap`, `changelog`, `help`, `blog`, `feedback`, `podcast` e `authors` no `_index.md` de uma pasta direta identificam sua função. O nome físico pode ser traduzido ou escolhido pelo usuário. `obsidian.Sections` lê somente `<pasta>/_index.md` das pastas diretas não ocultas e aplica o reconhecimento à área de propriedades delimitada no começo da nota, sem tratar exemplos no corpo como marca da pasta. Não exige nomes fixos como `Docs` ou `Feedback`.

No [quadro de Feedback](feedback.md), a nota da seção guarda a lista `examples`, incluindo comentários `replies`, que viaja com as notas no sync. Os pedidos e comentários reais estão nas issues GitHub, sem um arquivo de nota por pedido; não entram nessa lista nem são copiados pelo sync de notas. As traduções da nota ficam em sua pasta de idioma, como os demais índices.

## Conversão de wikilinks

O tema resolve links Markdown para notas e anexos. Os wikilinks existentes são convertidos por uma operação explícita, com implementação em `internal/wikilinks/{index,convert,anchor,tree}.go` e fluxo em `internal/app/links.go`. A criação pela CLI também oferece essa conversão com confirmação, por `convertLinksIfAny`; a criação pelo Studio não executa esse passo.

`NewIndex` percorre o cofre inteiro, ignorando nomes e pastas ocultos. O índice guarda nomes com e sem extensão `.md`, caminhos e versões sem distinção de maiúsculas. A resolução privilegia correspondência exata de caixa. Entre nomes iguais, escolhe menor distância de pastas, depois menor profundidade, caminho mais curto e ordem lexical. Há suporte a caminhos relativos, absolutos ao cofre e sufixos de caminhos.

| Entrada | Resultado ou limite |
| --- | --- |
| `[[Nota]]` | Link Markdown relativo para o arquivo resolvido |
| `[[Nota\|Texto]]` | Usa o alias como texto |
| `[[Nota#Título]]` | Link para heading da nota |
| `[[#Título]]` | Âncora da própria nota |
| `[[Nota#^bloco]]` | Remove a referência de bloco e liga à nota |
| `[[#^bloco]]` | Mantém o texto e relata problema |
| `![[foto.png]]` | Imagem Markdown |
| `![[foto.png\|300x200]]` | Mantém a largura no alt, como `![foto\|300x200](...)` |
| Embed de nota | Converte para link comum e relata perda de transclusão |
| Embed de anexo não imagem | Converte para link comum e relata aviso |
| Destino ausente | Mantém o wikilink e relata problema |

Os paths resultantes são relativos à nota de origem. Espaços e caracteres que quebrariam Markdown/URL são escapados por segmento; letras Unicode permanecem. Âncoras usam títulos reais quando disponíveis, com IDs personalizados e sufixos de repetição. Se o heading não for encontrado, usa um slug aproximado e registra aviso.

Front matter, blocos cercados de código, código inline da linha e colchetes escapados não são convertidos. Links a notas fora do recorte publicado geram aviso; anexos fora do recorte podem funcionar porque o scaffold monta os anexos do cofre inteiro.

O fluxo CLI simula primeiro, mostra até três exemplos, confirma e faz ZIP somente dos arquivos que mudarão. A aplicação preserva o modo do arquivo e escreve via temporário e rename. O backup fica no diretório técnico, não dentro das notas. Avisos da conversão exibidos pela CLI são limitados aos primeiros cinco, com contagem dos restantes.

## Transportes e escopo

`internal/notesync` faz comparação de arquivos por um contrato `Remote`. No Studio, `sync.remote = "blup"` seleciona o adaptador de [sync cifrado Blup Cloud](blup-cloud-sync.md), com `sync.vault` identificando o cofre. Vazio mantém o transporte GitHub, cujos detalhes estão abaixo. A CLI `sync` ainda prepara explicitamente GitHub; login na nuvem e sync cifrado são fluxos do Studio.

Eventos de calendário, diário, canvas, desenhos, bancos `.table` e registros podem viajar como arquivos elegíveis da pasta. O sync não calcula lembretes; isso cabe ao Studio e à API de e-mails. GitHub privado recebe o conteúdo dos arquivos sem a cifra de ponta a ponta do Blup Cloud. Publicação é outra operação e produz arquivos públicos.

## Repositório GitHub de sync

Nesse transporte, o destino é um repositório GitHub privado, com branch `main`, inicializado quando necessário por `.blup/README.md`. O transporte usa a API REST do GitHub: blobs, árvores, commits e referências usam Git data; a preparação ainda tenta `POST /user/repos`, mas o token fine-grained restrito orientado pela interface exige criar/incluir o repositório pela pessoa em `github.com/new`; inicialização usa `PUT /repos/{owner}/{repo}/contents/...`, e descoberta/configuração portátil usam também `/topics` e `/contents`. Não se cria checkout `.git` e não se chama o executável `git`.

`site.Sync` guarda `owner` e `repo`. Se faltar nome, a aplicação sugere `Slug(title)` com o sufixo de notas traduzido. O estado local fica em `<site>/.blup/sync-state.json`. A lixeira de exclusões recebidas fica por padrão em `<site>/.blup/sync-trash/<AAAA-MM-DD HHMMSS>/`.

O pacote `notesync.EnsureRepo` não torna privado um repositório existente. A CLI pergunta antes de usar um público; o caminho de API usado pelo Studio o recusa. O sync percorre somente `site.ContentDir()`, formado por `Vault + Folder`. Inclui anexos elegíveis dessa pasta mesmo que não sejam usados no site, mas exclui arquivos das outras pastas do cofre.

O scaffold monta anexos do cofre inteiro e pode publicar um anexo referenciado de fora de `ContentDir()`. Portanto os escopos diferem nos dois sentidos: nem tudo sincronizado é publicado e nem todo anexo publicado é sincronizado. Com `folder` preenchido, esses anexos externos à pasta do site não entram no repositório e não são recuperados por `BringSite`; precisam ser obtidos por outro meio para que suas referências voltem a funcionar.

Pastas e arquivos ocultos são ignorados, com a exceção exata de `.blup/site.toml`. Links simbólicos e submódulos remotos são ignorados, assim como caminhos inválidos, colisões de normalização Unicode e arquivos locais acima de 100 MiB. Caminhos são normalizados em NFC e usam `/`; NFC não equivale a converter nomes para minúsculas.

## Comparação de três versões

Cada arquivo é comparado pelo SHA do blob Git entre local, remoto e base da última sincronização. A decisão não é uma fusão textual por linhas.

| Situação | Ação |
| --- | --- |
| Local e remoto iguais | Mantém |
| Só local mudou | Envia |
| Só remoto mudou | Recebe |
| Local excluído, remoto igual à base | Exclui remotamente |
| Remoto excluído, local igual à base | Move o local para lixeira de sync |
| Um lado excluído, outro editado | Preserva a edição e a repõe no lado que excluiu |
| Ambos editados de forma diferente | Mantém local no nome original e remoto em cópia de conflito |
| Sem base e versões diferentes | Preserva ambas como conflito |

A cópia usa `Nome (conflict AAAA-MM-DD HHMM).ext`, com número adicional quando necessário. A versão local e a cópia remota entram na árvore do GitHub. O relatório completo `notesync.Report` relaciona nomes originais em `Conflicts` às cópias em `ConflictCopies`.

A CLI apresenta ambos, além dos itens `Skipped`, seus motivos e a lixeira. A API do Studio devolve `SyncInfo`: `SyncStatus` usa os nomes originais em `conflicts`; `SyncNow` usa os nomes das cópias. Não expõe `Skipped` nem `SkipReasons`; traz contagens de envio/recebimento, repositório, `upToDate` e, após sync, dados da lixeira. `SyncNow` define `upToDate = true` na resposta concluída, mesmo sem transportar os motivos dos arquivos ignorados.

O estado JSON contém `version`, `owner`, `repo`, `branch`, `commit`, `synced_at` e mapa `files` com `sha`, `size` e `mtime`. Estado ausente, corrompido, de versão diferente, com caminhos/SHA/tamanhos inválidos ou de outro destino é tratado como ausência de base, evitando deduzir exclusões de um histórico inválido. Erros de leitura do arquivo que não sejam ausência continuam sendo erros.

O scanner reutiliza hash quando tamanho e mtime correspondem ao cache e o arquivo tem mtime anterior ao início da leitura em mais de dois segundos. Antes de substituir ou remover um arquivo local recebido, o sync confere se o conteúdo ainda é o planejado; uma mudança concorrente nesse destino faz a operação ser ignorada para tentativa posterior. No envio, `uploadOne` relê o arquivo: se mudou desde o scan, envia o conteúdo recém-lido, registra seu SHA e desabilita a reutilização daquele cache de hash.

O commit e a atualização remota ocorrem antes das alterações locais recebidas. A atualização da branch exige fast-forward. Se outro computador mover a branch, refaz o plano até cinco tentativas; esgotamento gera `ErrRemoteMoved`. Downloads ou gravações locais que falham viram itens ignorados no relatório completo do pacote, mantendo base anterior para tentar novamente. A CLI os mostra; `SyncInfo` do Studio não os transmite.

A proteção `ErrLocalFolderEmpty` é específica: quando não há arquivos locais elegíveis, existe base e o plano apagaria mais de um arquivo remoto, o sync recusa. Ela não é um limite geral de quantidade de exclusões para qualquer pasta parcialmente vazia.

Se o remoto reaparece vazio depois de uma base com mais de um arquivo, `snapshot` descarta a base para refazer o envio, sem apagar as notas locais como se cada ausência remota fosse uma exclusão intencional. Para transportes `Ordered`, como Blup Cloud, um head anterior ao conhecido é recusado antes desse tratamento por `ErrRolledBack`. GitHub usa hashes de commits e não implementa essa comparação numérica de versões.

`Status` calcula o plano sem aplicar o sync. Na camada `app`, consultar status pode preparar o repositório, adicionar tópico e salvar o destino local; portanto não é uma operação estritamente sem efeitos externos.

## `.blup/site.toml` e recuperação em outra máquina

`internal/app/cloud_sites.go` define o tópico `blup-site` e o formato `sharedSite`. Antes do sync normal, a aplicação escreve uma cópia portátil ao lado das notas. A adição do tópico é tentada, mas seu erro não bloqueia o sync. Ela exige Administration no GitHub, permissão que a chave fine-grained orientada pelo Blup não pede; por isso a descoberta também aceita os sufixos de nome de notas reconhecidos, mantendo `.blup/site.toml` como confirmação do site.

| Viaja no TOML portátil | Fica fora dele |
| --- | --- |
| `version`, `title`, `description`, `author`, `lang`, `model`, `kind` | Caminhos `vault`, `folder` e diretório técnico |
| `translations`, `home`, `logo_link` | `Site.Password` e senha global |
| `cookies` | `deploy`, `feedback`, tokens, preferências Studio e estado local |
| Identidade e idiomas do site | `sync`, reconstruído pelo destino escolhido |

`sharedSite` não serializa `Deploy` nem `Feedback`, e a leitura de arquivos antigos não importa essas tabelas. A cópia portátil não troca contas ou destinos de publicação. `translations` usa presença explícita: se ausente num arquivo antigo, preserva os idiomas locais; lista presente e vazia limpa a lista, e valores presentes são filtrados para idiomas conhecidos, únicos e diferentes do principal. `home` e `logo_link` continuam portáteis; o domínio próprio não viaja, porque é vínculo do Blup Cloud guardado no servidor. A leitura filtra `logo_link` por `ValidLogoLink`; valor inválido volta à home. `home` guarda o nome da pasta e pode perder validade ao renomeá-la.

Overrides em `layouts/`, `assets/`, `static/`, `i18n/` e `data/` do diretório técnico não são incluídos por esse mecanismo. As notas, inclusive YAML com senhas literais, continuam sendo arquivos enviados ao repositório de notas.

`FindCloudSites` lista repositórios próprios por `OwnRepos`, com `affiliation=owner` e no máximo dez páginas de cem repositórios. Seleciona os privados com o tópico ou com sufixo de nome de notas reconhecido e lê `.blup/site.toml` por `FileContent`, limitado a 1 MiB. Arquivo ilegível, ausente, sem título ou de versão futura faz aquele repositório ser omitido da lista sem aviso individual. Ordena os resultados por atualização remota e associa um diretório local quando aquele `owner/repo` já consta em algum site conhecido. Repositórios públicos usados pela CLI não aparecem nessa descoberta.

`BringSite`, ao encontrar um site local já associado a `owner/repo`, apenas chama `LoadSite` e retorna, sem sincronizar. Sem associação, cria diretórios livres para notas e arquivos técnicos, aplica a configuração portátil, define `Vault` para a nova pasta e `Folder` vazio, salva o destino de sync e recebe o conteúdo. Não escreve sua configuração portátil antes desse primeiro recebimento. Depois lê o arquivo recebido e registra o site. Se o recebimento falhar, os diretórios e `blup.toml` já podem existir, mas o registro na lista global ainda não ocorreu. A janela do Studio abre o workspace a partir do `SiteInfo` retornado.

Ao fim de sync normal, a aplicação aplica as configurações compartilhadas quando `.blup/site.toml` consta como arquivo recebido. A cópia remota de conflito desse arquivo não é aplicada automaticamente. O estado local, os conflitos e as lixeiras não formam uma operação transacional única com todos os arquivos do site.

No Blup Cloud, o mesmo TOML portátil entra no manifesto cifrado; a descoberta e a recuperação usam `BlupSites`/`BringBlupSite`, sem tópico GitHub. O ID remoto fica em `sync.vault`; nome e manifesto são abertos no computador. O sync ao vivo dos planos pagos usa notificações de versão por WebSocket e a mesma comparação de três versões, sem edição colaborativa por caracteres. Veja [histórico e sync ao vivo](blup-cloud-sync.md).


## Guias de agentes na pasta das notas

[agents.go](../../internal/app/agents.go) usa os textos embutidos de [guides](../../internal/app/guides) para Mochi, Nori, Bento ou cofre `notes`. `INSTRUCTIONS.md` contém o guia por modelo/cofre; os três pontos de entrada `AGENTS.md`, `CLAUDE.md` e `GEMINI.md` apontam para ele. O texto dos guias é em inglês e informa o idioma de escrita do site pela substituição `{{LANGUAGE}}`.

Esses arquivos ficam em `site.ContentDir()`, onde o terminal abre, que só corresponde à raiz inteira do cofre quando `folder` está vazio. São criados ao criar site/cofre, abrir um workspace ou terminal e nos caminhos de recuperação que chamam `WriteAgentGuides`. O Studio continua abrindo se não puder atualizar o guia naquele momento.

Uma assinatura SHA-256 de textos conhecidos permite atualizar guias ainda intactos gerados pelo Blup, inclusive o `AGENTS.md` antigo. Arquivo editado pela pessoa não é sobrescrito. Links simbólicos, arquivos não regulares e existentes acima de 1 MiB são preservados. Os quatro nomes na raiz ficam fora das páginas Hugo, árvore de notas, busca e grafo; são arquivos locais de orientação, sem executar agentes ou enviar notas à IA por si sós.

O sync cifrado da rodada 3 também tem preparo por ticket após recusa de cota e purga manual de histórico por cofre. Esses mecanismos não mudam a comparação base/disco/remoto, as cópias de conflito ou o requisito de autenticar e validar o manifesto; o contrato está em [Blup Cloud](blup-cloud-sync.md#armazenamento-e-preparação-transacional).
