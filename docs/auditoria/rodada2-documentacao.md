# Rodada 2: tipagem e documentação

Registro histórico da rodada 2 em 02/11/2026. As mudanças posteriores de carências, baseURL, sync e cache estão na [rodada 3](rodada3-documentacao.md); os contratos atuais estão no [índice da spec](../spec/README.md). A fase de auditoria foi somente leitura; os achados foram encaminhados via Maestri à Lupa (Cloud) e ao Prumo (Studio/Go), com arquivo, linha e correção sugerida. As alterações desta entrega são apenas documentação. A equipe responsável implementou as correções de código e seus testes; esta entrega conferiu o resultado e executou os checks de integração.

## Achados e fechamento

| Gravidade | Grupo verificado | Estado atual |
| --- | --- | --- |
| Alto | Admissão concorrente de armazenamento/sites/IA e contabilização de resposta paga inválida | Reservas D1, confirmação de custo e regressões normais; sem `it.fails` |
| Alto | Entrega de site após exclusão/bloqueio ou perda do plano | Conta/plano verificados antes do cache; retenção e tolerância explícitas |
| Alto | Forma de manifesto decifrado e remontagem de arquivo fora do teto | Validação antes do estado, máximo 100 MiB, limite de bytes recebidos e SHA final |
| Alto | Proteção de conteúdo incompleta e envio remoto sem escolha/aceite válido | Verificação de marcadores, limpeza da saída em falha e consentimento por destino no Go |
| Médio | Hono com tipos manuais, rotas fora de AppType, params/query/body sem contrato e erros textuais | RPC gerado, Zod/limites de corpo, normalização de erro e testes de JSON/multipart malformados |
| Médio | Prefs/agenda e formatos JSON sincronizados aceitos parcialmente ou sem normalização | Cópia de Prefs, mapa de lembretes não nulo e validação de canvas/desenho/banco/mapa da prévia |
| Médio | Eventos/frontend com payloads escapando da tipagem e respostas HTTP sem verificação semântica | Eventos Wails registrados, frontend sem `any` de escape e verificações nos contratos Blup Cloud |
| Médio | Catálogos/custos externos e cursores/versões inválidos | Ranges, inteiros seguros, limites de resposta e ordenação de versões |
| Baixo | Diferenças entre contratos novos e documentação anterior | Specs, arquitetura, produção e segurança atualizadas sem prometer provisionamento remoto |

Foram enviados nove grupos de Cloud e dez grupos iniciais de Studio/Go, seguidos dos complementos identificados durante a conferência. Essa contagem agrupa achados relacionados; não é contagem de vulnerabilidades independentes. O achado 7 de erros HTTP está encerrado: a resposta textual de `HTTPException` é normalizada, e os testes conferem status, Content-Type JSON e `code`, inclusive a tradução com JSON malformado.

## Documentação entregue

- [Arquitetura](../arquitetura.md), com seis diagramas Mermaid e fluxos de domínio, retenção, cobrança, sync e IA.
- [Deploy](../deploy.md), com migração `0010`, reservas/rate limits, SaaS/TXT/SSL, cron, purga, secrets só por nome, versão de produção e assinatura Ed25519.
- [Segurança](../seguranca.md), com cifragem, metadados visíveis, consentimento, anexos, revogação, cotas e limites residuais.
- [Índice da spec](../spec/README.md) e [README principal](../../README.md), alinhados ao estado atual.
- Specs de Studio, temas, scaffold, publicação, sync, Blup Cloud, calendário, configuração, CLI, narração, Podcast, Feedback e testes/build.
- Nova [spec de tipagem e contratos](../spec/tipagem-e-contratos.md), cobrindo Hono RPC/Zod, bindings/eventos Wails, null/opcionais/números e validação de arquivos/dados externos.

Domínio próprio é Max/Ultra, um hostname por site, com prova TXT antes de provisionar/servir. Canonical, feed e sitemap continuam em `getblup.site`, pois vincular não muda baseURL nem republica, limite confirmado pelo Prumo. Exclusão lógica é imediata, purga após 30 dias em lotes; prazo de 14 dias só em pago para Grátis. Max/Ultra para Pro mantém sites e retira domínio próprio. Atualização de planos posterior a esta rodada: arquivo máximo atual de 10/25/50/100 MiB para Grátis/Pro/Max/Ultra; o texto comercial usa MB. Os demais comportamentos históricos deste parágrafo têm os contratos atuais na [spec](../spec/blup-cloud-sync.md).

## Checks executados nesta entrega

| Comando e local | Resultado |
| --- | --- |
| `go vet ./...`, raiz | Passou |
| `go test ./...`, raiz | Passou, incluindo `scripts/sign-release` e integração scaffold/Hugo |
| `go vet .`, Studio | Passou |
| `go test ./...`, Studio | Passou |
| `npx tsc -p worker/tsconfig.json --noEmit`, Cloud | Passou |
| `npx vitest run`, Cloud | 145 testes passaram em 12 arquivos, sem `it.fails` |
| `npm run check`, Cloud | Zero erros e zero avisos; contrato RPC gerado e Worker conferido |
| `npm run build`, Cloud | Passou |
| `npx svelte-check`, Studio/frontend | Zero erros, 53 avisos em nove arquivos |
| `npm test`, Studio/frontend | Sete arquivos de testes passaram na execução Node |
| `npm run build`, Studio/frontend | Passou; avisos Svelte e de chunk grande mantidos |
| Validação Markdown e `git diff --check` | Links/âncoras locais existentes, cercas pareadas, sem travessões ou padrões de segredo; diff sem espaços inválidos |

Os avisos do Studio incluem estado capturado na inicialização, acessibilidade de drag/drop e CSS sem uso. npm também informa configuração `minimum-release-age`; os builds emitem avisos de ambiente de cor/chunk. Foram reportados sem alterar código fora da área autorizada. A primeira execução Go foi bloqueada pelo sandbox em cache/sockets; a reexecução com permissão adequada passou. Nenhum servidor próprio foi iniciado e os temporários da sessão foram removidos ao concluir.

## Limites e pendências

1. Os wrappers genéricos `internal/deploy/github/client.go` e `internal/deploy/cfpages/client.go` mantêm DTOs sem schema semântico estrito: `null`/campos ausentes podem virar zero value. O Prumo confirmou essa fronteira para próxima rodada, preservando checagens de permissão e posteriores. Não foi reaberta integração de código nesta entrega.
2. Instalação inicial por `install.sh`/`install.ps1` continua só com SHA-256; a autoatualização CLI exige Ed25519. Uma eventual extensão da assinatura aos instaladores é outra mudança, fora da área desta documentação.
3. Configuração real de D1/R2, SaaS, DNS/TLS, PSL, e-mail, Access, Turnstile, Stripe e certificados/secret CI exige o procedimento do operador. Esta revisão não provisionou, cobrou, enviou e-mail ou publicou nada. As limitações de cache de hostname e URLs absolutas estão documentadas.
4. Os 53 avisos do Svelte e avisos de build não são erros de tipagem e ficam para revisão da área Studio; nenhuma correção de código foi feita por documentação.

## Fontes oficiais consultadas

Documentação atual por MCP Svelte (TypeScript, tipos SvelteKit e estado), MCP Wails (inspeção do projeto, sem executar o app) e MCP Context7 (Cloudflare e Better Auth). Também foram conferidas referências oficiais Go/JSON/Ed25519, Hono/RPC/validação, Drizzle, Zod, TypeScript, bindings/eventos Wails, Cloudflare for SaaS e Better Auth. Os links estão em [arquitetura](../arquitetura.md#referências-oficiais-consultadas), [deploy](../deploy.md) e [contratos](../spec/tipagem-e-contratos.md#referências-oficiais).

Sem commit, push ou deploy. Alterações ficam no working tree para revisão do Maestro.
