# Quadro de Feedback do Bento

Fontes: [seções disponíveis](../../internal/app/starter.go), [configuração das seções](../../internal/app/settings.go), [quadro Hugo](../../theme/layouts/_partials/bento/feedback-board.html), [cliente do site](../../theme/assets/js/feedback.js), [API do dono](../../internal/app/feedback_board.go) e [painel do Studio](../../studio/frontend/src/FeedbackPanel.svelte). O serviço de login fica no repositório irmão [blup-api](../../../blup-api/README.md); a rota de leitura foi conferida em [src/index.ts](../../../blup-api/src/index.ts).

## Ativação e origem dos dados

Feedback é uma seção Bento com `blup: feedback` no `_index.md` de uma pasta direta das notas. `obsidian.Sections`, em `internal/obsidian/obsidian.go`, reconhece a marca independentemente do nome da pasta; `bento/kind.html` escolhe o layout pela mesma propriedade. A home passa a ter o cartão dessa seção.

`internal/app/starter.go` mantém `feedback` em `studioKinds`, separado de `docsKinds`. `LoadSettings` junta as duas listas para o painel Seções do Studio. O assistente de criação, inclusive `CreateSite`, continua aceitando somente as cinco seções de `docsKinds`; Feedback é acrescentado depois, em Seções. A CLI não o oferece nos seus seletores. Essa restrição é da interface, pois o tema interpreta uma pasta marcada já existente.

Há uma pasta modelo `Feedback` em cada idioma:

- [Português](../../internal/app/starter/pt/bento/Feedback/_index.md).
- [Inglês](../../internal/app/starter/en/bento/Feedback/_index.md).
- [Espanhol](../../internal/app/starter/es/bento/Feedback/_index.md).

`AddSection` copia os arquivos ausentes desse modelo, sem sobrescrever notas. O `_index.md` traz título, `linkTitle`, `blup`, peso, descrição e exemplos. Remover a seção leva sua pasta à lixeira do Blup. Adicionar, remover e restaurar reiniciam uma prévia em execução, como descrito em [Studio](studio.md#prévia).

O quadro tem duas fontes de conteúdo: exemplos guardados na própria nota e pedidos reais nas issues do repositório de comentários configurado em `[feedback]`. Acrescentar a seção não habilita comentários por si só. O dono prepara o repositório e o App no painel Comentários; os requisitos de `Feedback.On()` continuam valendo.

## Quadro no site

`bento/feedback-board.html` gera a estrutura; `feedback.js`, na função `board`, faz a leitura e as interações. `bento.css` define a apresentação. O quadro oferece:

| Controle | Comportamento |
| --- | --- |
| Categorias na lateral | Todas, pedidos de recurso (`feature`), relatos de bug (`bug`), melhorias (`improvement`) e outros (`other`), com contagens do conjunto carregado, sem aplicar os filtros de busca e status |
| Filtro de status | Menu do tema com listbox, seleção, setas, Enter, espaço e Escape; oferece todos, ativos e cada status. Ativos reúne `new`, `planned` e `progress` |
| Mais votados | Ordem por reações `+1`; pedidos reais empatados usam a data mais recente |
| Recentes | Pedidos reais por data de criação decrescente; exemplos pelo menor `days` |
| Busca | Texto em minúsculas, sem remoção de acentos; nos pedidos reais procura em título e corpo HTML convertido para texto, nos exemplos no texto do cartão |
| Novo pedido | Formulário com tipo, título obrigatório de até 140 caracteres e descrição; envio pelo GitHub após login |
| Carregar mais pedidos | Começa com dez e acrescenta dez do conjunto já carregado; mudar categoria, status, ordem ou busca volta ao limite de dez |

Cada cartão mostra voto, status quando diferente de `new`, categoria, título, trecho, quantidade de comentários, avatar e tempo desde a criação. Nos pedidos reais, o avatar identifica o autor e o detalhe mostra também seu nome. Clicar no cartão abre o mesmo destino do título; o botão de voto mantém uma ação separada e cursor de clique. O voto real usa a reação `+1` da issue.

`#p-N` mostra o detalhe de uma issue carregada, no mesmo documento, com texto, autoria, data, votos, reações e comentários. `bento/feedback-block.html` é compartilhado com o feedback das páginas normais. O quadro esconde listagem, barra e botão de carregar mais durante o detalhe. Voltar e o histórico do navegador alternam essas vistas. Esses fragmentos não geram páginas Hugo ou arquivos separados por pedido.

## Exemplos da nota

`examples` é uma lista YAML editável na nota da seção. O modelo vem com cinco pedidos, cobrindo as quatro categorias e os cinco status. Estrutura de um item, com dados fictícios:

```yaml
examples:
  - title: "Melhorar a busca"
    kind: improvement
    status: planned
    votes: 12
    days: 4
    body: "Permitir filtrar o resultado por assunto."
    replies:
      - author: visitante
        days: 2
        text: "Isso ajudaria a encontrar as notas."
```

| Campo | Uso |
| --- | --- |
| `title`, `body` | Título e texto do pedido; o corpo do exemplo entra como texto em um parágrafo, sem renderização Markdown |
| `kind` | Categoria fixa; ausência ou valor desconhecido usa `other` |
| `status` | Estado fixo; ausência ou valor desconhecido usa `new`, que não exibe selo no cartão |
| `votes` | Contagem inicial de votos; ausência usa zero |
| `days` | Número de dias exibido como hoje, há um dia ou há N dias; também ordena os exemplos recentes. É um valor fixo, sem avançar com o calendário |
| `replies` | Lista de comentários de exemplo; cada item tem `author`, `days` e `text`; a contagem vem do tamanho dessa lista |

`feedback-examples.html` desenha os cartões, e `feedback-example-pages.html` desenha os detalhes `#ex-0`, `#ex-1` e seguintes, pelo índice na lista. Eles existem tanto na prévia quanto no site publicado. Só a prévia mostra o aviso de que são exemplos; o primeiro pedido real encontrado na leitura substitui todos os exemplos no quadro. Se nenhum pedido real passa pelos filtros atuais, os exemplos não voltam: a decisão usa o conjunto carregado antes dos filtros.

O voto de exemplo apenas alterna sua contagem no DOM daquela página, sem login, persistência ou envio ao GitHub. A caixa de comentar e a ação de reação do detalhe apenas informam que se trata de um exemplo; não criam comentários reais. Apagar `examples` da nota remove esses exemplos num próximo build. Se a leitura remota falhar e houver exemplos, o cliente continua com eles.

Sem comentários ligados, o script do quadro ainda é incluído por `mochi/scripts.html`. Não lê GitHub, permite filtrar e abrir exemplos e mostra um aviso de configuração ausente. Novo pedido abre o formulário; ao enviar um título preenchido, informa que é preciso ativar Comentários no Studio. Não cria uma issue nesse caso.

## Issues, marcas e status

Pedidos reais são criados diretamente por `POST /repos/{owner}/{repo}/issues` na API do GitHub, com título e corpo. O cliente acrescenta ao corpo uma marca invisível, por exemplo `<!-- blup:feedback kind=bug lang=es -->`. `kind` identifica o tipo e `lang` o idioma da página de origem; `lang` é opcional no leitor. O quadro e o Studio procuram essa marca, descartam pull requests e não confundem esses pedidos com as issues de páginas gerenciadas pelo build e sua etiqueta `blup`. Um tipo desconhecido na marca cai em `other`.

No cliente do site e no backend do Studio, o status vem da primeira etiqueta cujo nome começa com `status:` e cujo valor, depois de retirar espaços, pertence aos cinco status conhecidos. Etiquetas com outros valores são ignoradas. Sem etiqueta aplicável, issue aberta significa `new` e fechada significa `done`. Categoria e status são listas fixas no tema, no cliente e no painel do Studio; o Studio permite escolher o estado de um pedido, sem criar, renomear ou remover essas opções.

| Status | Apresentação em português | Efeito de `SetFeedbackStatus` |
| --- | --- | --- |
| `new` | Novo | Retira as etiquetas dos status gerenciados pelo quadro e reabre a issue |
| `planned` | Planejado | Aplica `status: planned` e reabre |
| `progress` | Em andamento | Aplica `status: progress` e reabre |
| `done` | Feito | Aplica `status: done` e fecha com motivo `completed` |
| `declined` | Recusado | Aplica `status: declined` e fecha com motivo `not_planned` |

`internal/app/feedback_board.go` exige número positivo e status conhecido, lê a issue individual por `BoardIssue` e recusa pull requests ou issues sem a marca de pedido. Portanto a alteração pode alcançar uma issue mais antiga que o recorte da listagem. Garante e acrescenta a etiqueta escolhida, remove somente as etiquetas de status reconhecidos que conflitam e então altera o estado aberto/fechado. Preserva as demais etiquetas, inclusive valores `status:` fora da lista gerenciada, sem substituir a lista inteira. São chamadas separadas, sem transação entre etiquetas e fechamento; a autorização é a credencial GitHub do dono.

Com serviço de login configurado, o site exige login GitHub para abrir o formulário real e cria a issue com a autorização do visitante. O envio valida a categoria contra a lista fixa, remove marcas de feedback escritas no corpo pelo visitante e acrescenta a marca gerada, com idioma somente entre `pt`, `en` e `es`. Sem esse serviço, Novo pedido encaminha ao formulário de issue do GitHub. Esse link externo não acrescenta automaticamente a marca; uma issue sem a marca não entra no quadro.

As credenciais do visitante ficam em `localStorage`, no navegador dele, para não pedir login a cada visita (decisão do dono); sair apaga a chave. A defesa contra roubo dela é não executar HTML de visitantes (abaixo). `feedback.js` copia o HTML de pedidos, comentários e prévias Markdown por uma lista de elementos permitidos, sem handlers, estilos, embeds ativos ou SVG. Endereços inseridos pelo cliente passam por `safeURL`. Os exemplos da nota usam o escape dos templates Hugo. Esse tratamento conserva o conteúdo de texto sem executar HTML fornecido por visitantes.

## Leitura por `/posts` e limites

`feedback.js` usa `read` para escolher a origem. Visitantes sem login usam `GET /posts?repo=owner%2Frepo` do serviço, com fallback para GitHub se falhar. Autenticados ou visitantes que acabaram de alterar feedback usam diretamente `GET /repos/{owner}/{repo}/issues?state=all&sort=created&direction=desc&per_page=100`.

No [blup-api](../../../blup-api/src/index.ts), `readPosts` valida `repo`, usa a autorização da instalação do App e consulta até três páginas de 100 issues abertas e fechadas, das mais recentes para as mais antigas. Descarta pull requests e corpos sem o prefixo literal `<!-- blup:feedback`; o cliente aplica depois o regex completo. A resposta contém número, título, URL, estado, data, comentários, reações, usuário, `body`, `body_html` e somente as etiquetas `status:`. O cache público dura 60 segundos; a ausência de instalação produz 404. Essa rota lê, sem criar pedidos nem mudar status.

O navegador não percorre outras páginas de issues: pela chamada direta vê no máximo as primeiras 100, antes da filtragem. O serviço e o Studio consultam no máximo 300 issues do repositório, também antes de filtrar pedidos. Carregar mais só revela outros dez itens recebidos, sem ampliar esses limites. A leitura de comentários retorna até 100 por issue, sem paginação adicional.

O cache em sessão do cliente dura um minuto para leituras do serviço e cinco minutos para GitHub. Mudanças feitas pelo próprio visitante limpam o cache e favorecem leituras diretas por três minutos. Mudar status no Studio não envia evento para visitantes, não faz polling e não exige rebuild do site. Uma aba já aberta precisa de nova leitura para receber a alteração; o TTL de um minuto não garante atualização automática nessa aba.

## Idiomas e tradução

Os controles do site usam `theme/i18n/{pt,en,es}.toml`, conforme o idioma da página, inclusive categorias, status, formulário, avisos, datas relativas e carregar mais. Em espanhol, os pedidos são chamados de `solicitud`, com textos como `Nueva solicitud` e `Cargar más solicitudes`; em inglês, o filtro geral é `All statuses`. O painel do Studio usa seu próprio `studio/frontend/src/i18n.ts`, no idioma da interface.

Título e descrição da seção, `linkTitle` e textos dos exemplos dependem da tradução da nota. Em [internal/app/translate.go](../../internal/app/translate.go), `exampleTexts` e `walkExamples` percorrem, em ordem, `title`, `body` e `replies[].text` não vazios. A tradução remota usa `PropTexts`; a local recebe os mesmos textos por `TranslationTexts` e grava por `SaveTranslation`. `translatedProps` só substitui a lista quando a quantidade de textos traduzidos coincide. Votos, categorias, status, autores, `days` e demais campos conservam seus valores.

`WriteTranslation` valida uma lista de textos de exemplos fornecida antes de mudar a original ou a tradução: exige somente strings e a mesma quantidade esperada. Dados inválidos retornam erro, sem marcar exemplos não traduzidos como atuais. `walkExamples` conserva itens YAML de outros tipos sem tentar tratá-los como pedidos.

`sourceHash` acrescenta esses textos à impressão de título, descrição e corpo daquela nota. Notas sem `examples` não mudam de impressão por essa ampliação. Alterar textos de um exemplo deixa sua tradução desatualizada; mudar apenas votos, status ou `days` não altera esse hash. A operação não consulta issues para traduzi-las.

O cabeçalho do quadro chama `site/ai-badge.html`: com `ai_translated` e uma página original ligada, mostra o selo de tradução com IA e o acesso à original. Marcar a tradução como revisada remove a marca. O selo se refere à nota traduzida, sem afirmar que as issues foram traduzidas.

Pedidos e comentários reais mantêm o texto de quem os escreveu, compartilhado por todos os idiomas do site. O pedido usa `lang` na marca de feedback; novos comentários pelo site recebem `<!-- blup:lang=es -->`, por exemplo. Em outra versão, o cliente mostra o idioma de origem. Comentários sem essa marca não recebem inferência automática de idioma. A [documentação dos temas](temas-hugo.md#comentários-reações-e-votos) descreve também como páginas traduzidas compartilham uma issue.
