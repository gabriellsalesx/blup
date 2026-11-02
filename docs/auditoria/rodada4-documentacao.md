# Rodada 4: tipagem, distribuição e checklist

Conferência de 02/11/2026. A fase 1 foi somente de leitura: revisão independente de Go, bindings Wails, Svelte, cliente Blup Cloud, Hono/RPC, entradas Zod, contratos externos e código sem consumidor. Achados foram enviados por Maestri aos donos com arquivo/linha e sugestão de regressão. A fase 2 atualiza documentação após os handoffs, sem alteração de código por Pergaminho.

## Achados encaminhados

Foram oito grupos: dois médios e seis baixos. Não encontrei achado novo crítico ou alto nesta revisão; isso não equivale a certificação de segurança. As linhas abaixo são as localizações no momento do envio, antes das correções.

| Gravidade | Dono | Fonte auditada | Fechamento conferido |
| --- | --- | --- | --- |
| Médio | Prumo | `internal/blupcloud/sites.go:61-84` | Sites, criação e consulta de slug agora conferem presença/invariantes, IDs, URLs, enums e identidade do slug; resposta inválida vira `ErrBadAnswer` |
| Médio | Prumo | `studio/frontend/src/Preview.svelte:44-59` | `MessageEvent<unknown>`, parser de forma e origem/frame/marca conferidos antes de alterar estado; reinício em outra porta recarrega |
| Baixo | Prumo | `studio/frontend/src/FieldInput.svelte:29,104` | `typedValue` salva número finito e evita gravação sem mudança; vazio limpa; teste da função pura |
| Baixo | Lupa | `cloud/src/routes/(acesso)/entrar/+page.svelte:24` | Configuração pública usa cliente RPC e trata erro HTTP; tipo manual removido |
| Baixo | Lupa | `cloud/worker/index.ts:24`, `reminders.ts:4,7`, `sites-cleanup.ts:1`, `sync.ts:36` e testes | Dez símbolos sem uso removidos; `noUnusedLocals` e `noUnusedParameters` agora fazem parte do tsconfig Worker |
| Baixo | Lupa | `cloud/worker/ai.ts:100`, `index.ts:35` | Envelope de erro de IA mantém `code` e `error`, inclusive rejeições comuns por origem, headers, tráfego e corpo |
| Baixo | Prumo | `internal/blupcloud/client.go:426-437,535-548`, `reminders.go:26-38` | Purga/lembretes exigem campos e faixas; `missing` exige lista única pertencente ao lote; cofres/embrulhos incompletos são recusados |
| Baixo | Prumo | `studio/frontend/src/localtranslate.ts:19-24` | Resposta do worker é `unknown`, validada por parser, com ID seguro e quantidade de textos igual à solicitação |

Houve ainda observação baixa de empacotamento: `Maintainer` do `.deb` precisa de nome/e-mail válido, sem URL ocupando o lugar do contato. O nfpm recebe `DEB_MAINTAINER`, e o checklist orienta o contato corporativo escolhido pelo dono.

As regressões estão em [sites_answer_test.go](../../internal/blupcloud/sites_answer_test.go), [previewurl.test.ts](../../studio/frontend/tests/previewurl.test.ts), [fieldvalue.test.ts](../../studio/frontend/tests/fieldvalue.test.ts), [translatereply.test.ts](../../studio/frontend/tests/translatereply.test.ts), [api.test.ts](../../cloud/src/lib/api.test.ts), [security.test.ts](../../cloud/worker/security.test.ts) e testes associados. Os testes frontend citados cobrem parsers/funções puras, sem montar os componentes nem executar o worker de tradução completo em Node.

## Contratos documentados

- `/plans` usa preços Stripe por período, anuláveis, com cache de dez minutos por isolate, atualização agrupada, último bom em falha e backoff de trinta segundos. Falta de preço bloqueia checkout com 503. O script de setup só confere catálogo. Unidades monetárias e preços legados estão descritos sem fixar valores comerciais.
- Webhook próprio assinado consulta o estado atual no Stripe sob lease, grava estado/recibo no mesmo batch e deduplica `event.id`. Webhook/success do plugin ficam desativados. Lease por conta também protege referência pendente compartilhada por assinaturas distintas. Success só redireciona para URL local; recibos ficam elegíveis após 90 dias, até 500 por cron diário, podendo permanecer mais com acúmulo.
- Migrações finais `0000..0013`. A atualização de instalação existente não deve remover as colunas da `0012` com o Worker anterior ainda gravando. Checklist exige plano compatível ou janela de manutenção e alerta sobre rollback de código/schema.
- Sync acima da cota: exceção sem novos blocos limitada ao manifesto cifrado real do head no R2 mais 4096 bytes; crescimento normal também reserva uso. Criação concorrente de chave perdida retorna `409 KEY_EXISTS`. Não existe troca da senha das notas na interface.
- `/me.id` validado e copiável; `historyBytes` continua nullable; `GET /reminders.emailMissing` informa ausência de e-mail real sem consumir envio/cota. O aviso aparece no Studio, com notificações locais preservadas.
- Apagar definitivamente uma pasta do topo fora do Studio reinicia a prévia. Campos numéricos, mensagens de prévia e respostas de tradução têm validação de forma. Código morto, bindings sem tela e imagens removidas deixaram de ser apresentados como recursos ativos.
- Studio e Cloud são apresentados como produtos próprios. A marca de outro aplicativo fica apenas na compatibilidade da CLI; nomes internos de pacote e caminhos técnicos continuam documentados.

## Entregas e distribuição

O [checklist](../deploy-checklist.md) reúne em ordem contas, alertas de gasto, D1/R2, migrações, Email Service, Turnstile, Access, catálogo/portal/webhook Stripe, DNS/SaaS/PSL, secrets sem valores, dois Workers, crons, primeiro teste e assinatura/publicação desktop. [Instalação](../instalacao.md) explica SmartScreen e Gatekeeper por aplicativo, sem desativar proteções globais.

Linux: AppImage, AUR, bundle `.flatpak` no GitHub e `.deb`. Flathub está fora deste lançamento. Os arquivos do Prumo foram lidos; os jobs configurados publicam os pacotes Linux com SHA-256 e Ed25519. MSIX x64/arm64 fica como artefato sem assinatura, condicionado à identidade Partner Center. Builds diretos Windows/macOS seguem sem assinatura de desenvolvedor, por decisão do dono.

O handoff do Prumo registra validações locais de AppImage, MetaInfo, desktop entry, AUR e bundle Flatpak, e inspeção do `.deb`. Não foram executados os jobs GitHub nem MSIX/MakeAppx no Windows. Lintian e instalações Debian/Ubuntu estão configurados no CI; abrir a GUI nos sistemas alvo continua etapa operacional.

As [políticas de distribuição](politicas-distribuicao.md) têm links oficiais e separam regra publicada de decisão do projeto. A Microsoft documenta desenvolvimento/publicação assistidos por IA; a Store exige política de privacidade para desktop e declaração/canal de relato para IA dinâmica. Nas AUR guidelines examinadas não encontrei veto específico a PKGBUILD com IA, sem garantir aceitação. Distribuir `.deb` na release não exige submissão Debian/Ubuntu.

Por decisão do dono, `/privacidade`, `/termos` e `/feedback` ficam no novo institucional Hugo privado. São URLs **a publicar no site institucional** em `getblup.io`, com `help@getblup.io`. O relato de IA usa o Feedback institucional, sem botão no Studio. A Store permanece documentalmente bloqueada até essas páginas estarem no ar e conferidas.

Arquivos desta entrega: `README.md`, `README.pt-BR.md`; guias de arquitetura, deploy, checklist, instalação e segurança; specs de índice, Cloud, calendário, CLI, narração, notas/sync, publicação, scaffold, Studio, tema, build e tipagem; registro de políticas e este relatório. A conferência histórica ganhou apenas ajuste de apresentação, preservando seu parecer. `cloud/README.md` foi lido, sem edição por esta entrega.

```text
README.md
README.pt-BR.md
docs/arquitetura.md
docs/deploy.md
docs/deploy-checklist.md
docs/instalacao.md
docs/seguranca.md
docs/spec/README.md
docs/spec/blup-cloud-sync.md
docs/spec/calendario-e-lembretes.md
docs/spec/cli.md
docs/spec/narracao-e-voz.md
docs/spec/notas-e-sync.md
docs/spec/publicacao.md
docs/spec/scaffold-e-build.md
docs/spec/studio.md
docs/spec/temas-hugo.md
docs/spec/testes-e-build.md
docs/spec/tipagem-e-contratos.md
docs/auditoria/conferencia-spec.md
docs/auditoria/politicas-distribuicao.md
docs/auditoria/rodada4-documentacao.md
```

## Checks executados pelo Pergaminho

| Check | Resultado |
| --- | --- |
| Raiz `go vet ./...` e `go test ./...` | Passaram, incluindo os pacotes de scripts e clientes |
| Studio `go vet .` e `go test ./...` | Passaram |
| Studio/frontend `npm run check` | Zero erros e zero avisos Svelte |
| Studio/frontend `npm test` e `npm run build` | Passaram |
| Cloud `npx tsc -p worker/tsconfig.json --noEmit` | Passou, com checagem de símbolos sem uso |
| Cloud `npx vitest run` | 235 testes em 21 arquivos passaram |
| Cloud `npm run check` | Zero erros e zero avisos Svelte; inclui tipos de Worker e scripts |
| Cloud `npm run build` | Passou |
| Markdown, links e diff | 28 arquivos, 418 links/âncoras locais e sete diagramas; sem destino ausente, âncora inválida, cerca aberta, travessão ou padrão de credencial privada; `git diff --check` limpo |

O fechamento da Lupa confirmou os mesmos 235 testes, check/build e ausência de mudança de API após o handoff. Ela aplicou `0000..0013` do zero no D1 local por Wrangler, conferiu `integrity_check` e EXPLAIN com índices de assinaturas, recibos, lembretes e purga. Esses checks do banco foram informados pela área Cloud, sem aplicação remota ou repetição por documentação.

Os avisos restantes observados são de ferramentas: configuração npm `minimum-release-age`, API experimental no ambiente Node e tamanho de chunk do build Studio. Não houve erro de check/build. As validações locais usam serviços simulados; não comprovam provisionamento nem comportamento remoto de produção.

## Pendências e limites

1. Publicar e conferir o institucional e seu canal de atendimento; inserir URL da política e declarar IA no Partner Center. Configurar identidade MSIX e testar no Windows antes da Store.
2. Confirmar app ID antes do primeiro release; manter o atual até eventual decisão coordenada. Preparar conta/chave SSH AUR e atualizar receita/checksum após existir a release real.
3. Executar CI e testar abertura dos pacotes nos sistemas alvo. Não confundir template/compilação/ldd com aplicativo publicado e homologado.
4. Provisionar e conferir produção, incluindo webhook antes de cobrar, SaaS/DNS/TLS/PSL, e-mail, Access, Turnstile, cron, recuperação e alertas. Testes pagos completos ficam no sandbox.
5. Permanecem limites aceitos: wrappers genéricos GitHub/Cloudflare Pages sem schema semântico estrito; instalação inicial da CLI só SHA-256, distinta da autoatualização Ed25519; testes frontend não equivalem a uma suíte de componentes/navegador.

## Fontes e disciplina

Consultei os MCPs Svelte e Wails e o Context7 para Cloudflare e Better Auth atuais, além das referências oficiais de Go, Hono, Drizzle, Zod, TypeScript, Stripe, Apple, Microsoft, Flatpak, Arch e Debian/Ubuntu. Links estão nos guias, em [contratos](../spec/tipagem-e-contratos.md#referências-oficiais) e no registro de políticas. Código e lockfiles prevaleceram sobre mensagens de commit; não houve atualização de dependência por documentação.

Sem edição de código, commit, push, deploy, cobrança, e-mail ou chamada paga. Não iniciei nem reiniciei o Studio do dono e não toquei nas portas reservadas. Temporários próprios de reproduções foram removidos; arquivos temporários das outras áreas foram preservados.
