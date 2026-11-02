# Rodada 3: conferência e documentação

Revisão de 02/11/2026 sobre o código estabilizado da Lupa e do Prumo, incluindo commits `e3b4898` e `fa7f0c6`, contratos Cloud/migração `0011` e o ajuste nullable de histórico. As mudanças desta entrega são só documentação em `docs/` e no README principal. Correções de código e regressões foram implementadas pelas áreas responsáveis.

## Achados encaminhados e fechados

| Gravidade | Fonte | Reprodução e fechamento |
| --- | --- | --- |
| Médio, integridade | `cloud/worker/sync-transfers.ts`, `putPreparedBlock` | Dois PUTs liam `ready` antes da lease; um atrasado podia gravar depois do commit e só então falhar no guard. Agora relê estado após lease, usa chave R2 única e CAS de adoção; commit referencia `object_key` do recibo. Confirmação D1 perdida não apaga chave possivelmente adotada |
| Médio, disponibilidade | `cloud/worker/domain-downgrade.ts`, `disableExpiredDomain` | Upgrade podia limpar `disable_at`, UPDATE afetar zero linhas e ainda ocorrer DELETE externo. Agora `UPDATE ... RETURNING` ganha o claim; sem linha não remove hostname |

Ambos foram reproduzidos em memória com funções reais extraídas/transpiladas, sem editar arquivos de código ou usar rede. Foram enviados à Lupa por Maestri com arquivo/linha e teste sugerido. A equipe implementou testes normais com barreiras em `round3-security.test.ts`, cobrindo PUT retomado depois do commit e upgrade antes do claim. A conferência final e a suíte passaram; esses dois achados não ficam pendentes.

## Contratos documentados

- Carência de 14 dias em redução com excesso de sites/storage, teto anterior de storage durante o prazo, uso total incluindo histórico e preservação dos excedentes. Não confundir com a retenção de conta excluída por 30 dias.
- Pro escolhe site retido por `PUT /api/sites/retained` até o prazo; GET sites devolve `downgrade`. Sem escolha usa data/ID; Grátis mantém zero sites depois da carência. Arquivos permanecem guardados.
- Domínio Max/Ultra tem carência própria de 14 dias, `disableAt` ISO/null e estado `disabled`; upgrade antes do desligamento pode cancelá-lo, e reconfiguração posterior exige plano elegível/prova.
- URL oficial exige domínio/SSL ativos e direito ou carência válida. O Studio republica em segundo plano, com mutex e teto de uma vez/dez minutos por site; falha de consulta conserva a base anterior. Dois hosts servem o mesmo deploy sem redirecionar entre si.
- Preparo por ticket após 413, blocos novos retidos em memória no cliente, reserva líquida, PUT com ticket, commit idempotente no prazo e cancelamento/expiração. Upload normal mantém paralelismo do cliente, com exclusividade de preparo/purga. Versão sem blocos novos tem exceção explícita mesmo se o manifesto aumentar o uso.
- Histórico: `historyBytes` é total da conta e pode ser `null`, preservado por Go `*int64` e TS; a limpeza manual é só do cofre do site aberto. Mantém head e blocos necessários, e só desconta após confirmação R2, sem duplicar desconto.
- Workers Cache substitui Cache API: gateway sem cache consulta D1 por pedido, props separam site/deploy, fila de purga persistente usa `SITE_CACHE`/`CachedSite` no cron. TTL interno 600 segundos e política de navegador separada. Early Hints/103 e economia real exigem validação externa.
- Home Bento como seção, título/descrição do site em `/`, paginação `/page/2/`, canonical/sitemap sem entrada duplicada, fallback ao renomear pasta; nenhum redirect desse recurso.
- `logo_link` nos três temas, mesma aba, caminho seguindo base/subpasta de Pages, validação Go/TS e valor inválido do sync ignorado.
- `INSTRUCTIONS.md` por tema/cofre em `ContentDir()`, três arquivos de entrada e migração por hash de texto intacto, preservando edições da pessoa. Canvas guarda tipos desconhecidos e arestas compatíveis sem renderizar; remove a aresta guardada quando sua ponta conhecida foi apagada.
- Calendário com um único botão de criar no painel do dia; pasta segue o idioma do Studio (`Agenda` pt/es, `Calendar` en).

## Arquivos desta entrega

Guias [arquitetura](../arquitetura.md), [deploy](../deploy.md) e [segurança](../seguranca.md); [README principal](../../README.md); índice e specs de Blup Cloud, publicação, Studio, configuração, notas/sync, tema Hugo, scaffold/build, CLI, Podcast, Feedback, calendário, testes/build e tipagem. Este relatório é novo. O relatório da rodada 2 ganhou aviso de registro histórico para não confundir contratos substituídos.

O guia de deploy inclui migração `0011`, binding `SITE_CACHE` e ordem de publicar primeiro o Worker dos sites, além de configurações de cache/Early Hints, validações de staging e budget alert sem valores. `cloud/README.md` é o contrato final da área Cloud e não foi editado por esta entrega.

## Validação executada pelo Pergaminho

| Check | Resultado |
| --- | --- |
| Raiz `go vet ./...` e `go test ./...` | Passaram, incluindo integrações Hugo |
| Studio `go vet .` e `go test ./...` | Passaram |
| Cloud `npx tsc -p worker/tsconfig.json --noEmit` | Passou |
| Cloud `npx vitest run` | 177 testes em 14 arquivos passaram; sem `it.fails` |
| Cloud `npm run check` | Zero erros e zero avisos |
| Cloud `npm run build` | Passou |
| Studio/frontend `npx svelte-check` | Zero erros; 53 avisos existentes em nove arquivos |
| Studio/frontend `npm test` | 46 testes passaram |
| Studio/frontend `npm run build` | Passou, com avisos Svelte/bundle e ferramentas |
| Markdown e diff | 22 arquivos conferidos, 338 links/âncoras locais, sete diagramas e 70 tabelas; cercas pareadas, sem travessões/padrões de segredo; `git diff --check` limpo |

A aplicação de migrações `0000..0011` do zero no D1 local por Wrangler e o EXPLAIN com índices novos foram informados pela Lupa. Não foram repetidos por documentação nem aplicados remotamente. A suíte local usa infraestrutura simulada, sem provar hit gerenciado, propagação global de purga ou 103 real.

## Limites que permanecem

Os wrappers genéricos GitHub/Cloudflare Pages continuam com DTOs Go sem schema semântico estrito, limite já registrado na rodada 2. A instalação inicial CLI continua só SHA-256, separada da autoatualização Ed25519. Os 53 avisos Svelte existentes e avisos de ferramentas/build não foram alterados nesta área.

Produção ainda exige conferir recursos, SaaS, DNS/TLS/PSL, secrets/assinaturas, e-mail, Access, Turnstile, Stripe e alertas reais. Ganho de custo do cache precisa de medição: gateway/D1 e invocações continuam. Nenhuma dessas etapas foi declarada concluída pelo código ou pelos testes desta entrega. Não há pendência nova de contrato entre Lupa e Prumo para estes documentos.

## Fontes e disciplina de trabalho

Leitura do código prevaleceu sobre títulos de commit e handoffs. Consultei Cloudflare Workers pelo MCP Context7 e referências oficiais de Workers Cache/chaves/purga/limites, R2, Early Hints e budget alerts, citadas nos guias. As referências oficiais e MCPs de Svelte, Wails, Better Auth, Go, Hono, Drizzle, Zod e TypeScript consultadas na rodada anterior continuam listadas em arquitetura e contratos; não houve atualização de dependência nesta entrega.

Sem alterações de código, commit, push ou deploy; sem iniciar Studio/servidores, tocar portas reservadas ou chamar serviços pagos. Temporários próprios foram removidos ao terminar. As mudanças de documentação foram deixadas para revisão do Maestro.
