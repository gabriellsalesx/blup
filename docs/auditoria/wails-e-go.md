# Auditoria de Wails e Go do Blup

Data: 30/10/2026. Auditora: Lupa, a pedido de Gabriell e do Maestro Claude Code.

Referência inicial: commit `975554bcef7b007de97ba4758967f6f50512c8e5`. O checkout é compartilhado com outros agentes; as linhas abaixo correspondem aos arquivos lidos nesta auditoria. Nenhum código, manifesto, binding ou arquivo de build foi alterado. Não houve instalação de ferramentas nem commit. A gravação desta auditora no repositório se limita a este relatório.

## Resultado

O projeto está alinhado à versão estável atual do Wails v3. As verificações executadas passaram. Os principais problemas estão na contenção do acesso a arquivos, no desfazer de snapshots e na adaptação de estado originalmente usado pelo CLI para chamadas concorrentes do Studio.

| Classificação | Alta | Média | Baixa |
| --- | ---: | ---: | ---: |
| Achados confirmados por inspeção do código | 3 | 15 | 3 |
| Sugestões e riscos condicionais | 1 | 2 | 2 |

“Confirmado” significa que a implementação e o caminho causal foram identificados no código. Os cenários de exploração, perda de estado, falta de memória e falhas de disco descritos abaixo não foram provocados no aplicativo. A execução se limitou às verificações existentes e à compilação das variantes indicadas. Em particular, nenhuma falha de concorrência foi detectada pela suíte existente com `-race`; os acessos sem sincronização apontados por leitura não são cobertos por essa conclusão.

Alta: acesso fora da fronteira de arquivos ou remoção indevida de conteúdo. Média: inconsistência de estado, indisponibilidade, exposição local de segredo ou falha de entrega. Baixa: manutenção, diagnóstico ou recursos com impacto limitado. Para sugestões, a gravidade se aplica se a condição descrita ocorrer.

## Escopo e execução

Foram inventariados os dois módulos Go, 133 arquivos `.go`, com 25.821 linhas incluindo testes, e todos os pacotes: `cmd/blup`, `internal/app`, `config`, `deploy/cfpages`, `deploy/github`, `hugo`, `i18n`, `kokoro`, `lock`, `narrate`, `notesync`, `obsidian`, `pagefind`, `pwa`, `scaffold`, `ui`, `update`, `wikilinks` e `studio`. A revisão estática priorizou fronteiras de arquivos, persistência, erros, processos, contexto e concorrência. Também foram lidos Taskfiles, manifests, Dockerfiles, configuração de empacotamento, workflow de release e pontos de integração do frontend com Wails.

O MCP `wails` foi usado nas operações de leitura `wails_project_inspect`, `wails_project_task_list` e `wails_project_doctor`. Ele reconheceu `studio/` como projeto Wails e informou `ready: true`. O CLI do diagnóstico foi compilado com `gtk3`; isso descreve o próprio CLI. O projeto foi compilado com a stack padrão GTK4/WebKitGTK 6.0 instalada.

Ambiente: Linux/amd64, Go `go1.27.3`, `CGO_ENABLED=1`. Os dois `go.mod` declaram `go 1.27.3`. Os comandos de teste usaram `-mod=readonly`, explicitamente ou via `GOFLAGS`. Na primeira rodada, downloads de módulos foram desabilitados com `GOPROXY=off` e `GOSUMDB=off`, usando dependências já presentes. Nenhum `go get`, `go install`, gerador ou tarefa de instalação foi executado.

| Verificação | Local/variante | Resultado |
| --- | --- | --- |
| `wails3 version` | CLI instalado | `v3.0.0` |
| `go vet ./...` | raiz | Passou |
| `go vet ./...` | `studio/` | Passou |
| `go test -count=1 -timeout=120s -coverprofile=... ./...` | raiz | Passou após liberar sockets locais de `httptest` |
| Mesmo comando | `studio/` | Passou |
| `go test -mod=readonly -race -count=1 -timeout=120s ./...` | raiz | Passou |
| Mesmo comando | `studio/` | Passou |
| `CGO_ENABLED=0 go test -run '^$' ./...` | raiz | Compilou; testes não executados |
| `go test -tags production -run '^$' ./...` | `studio/` | Compilou; testes não executados |
| `CGO_ENABLED=0 go test -tags server,production -run '^$' ./...` | `studio/` | Compilou; testes não executados |
| Comparação dos nomes dos métodos Go com as funções geradas TS | `Studio` | 120 de cada lado, sem nome ausente ou excedente |
| `gofmt -l` | fontes Go | Apenas `internal/app/cloud_sites_test.go` listado |

Os avisos C de depreciação de `gdk_x11_display_get_xdisplay` e `gdk_x11_surface_get_xid` vieram da dependência Wails, não dos fontes do Blup. Não impediram as verificações. A primeira rodada de testes da raiz falhou porque o sandbox bloqueou os servidores locais de `httptest`; a repetição autorizada passou. Essa falha inicial não é um defeito do projeto.

Cobertura de statements: raiz 52,4%; `internal/app` 33,9%; `internal/kokoro` 41,2%; `studio` 1,5%. Os relatórios temporários ficaram em `/tmp/blup-audit-root.cover` e `/tmp/blup-audit-studio.cover`. O teste de inferência real `internal/kokoro/kokoro_test.go:157` depende de `BLUP_KOKORO_DIR` e não valida uma instalação real na execução padrão.

Não foram executados o aplicativo gráfico, inferência real, instalação de voz, publicação, sincronização com contas reais, empacotamento, assinatura, Docker ou tarefas que regeneram arquivos. Não houve execução em Windows/macOS nem varredura de vulnerabilidades com ferramenta externa.

## Versão e práticas do Wails v3

`studio/go.mod:9`, o CLI `wails3 version` e `studio/frontend/package.json:24` estão alinhados em `v3.0.0`/`3.0.0`. A API pública de releases do GitHub confirmou a estável v3.0.0, publicada em 26/10/2026, e a prévia mais recente listada, `v3.1.0-nightly.20261028`, publicada em 28/10/2026. Portanto, não há atualização estável pendente identificada. A nightly pertence a outro canal. Fontes: [release estável](https://github.com/wailsapp/wails/releases/tag/v3.0.0), [nightly](https://github.com/wailsapp/wails/releases/tag/v3.1.0-nightly.20261028) e [changelog oficial](https://wails.io/changelog/).

| Área | Avaliação |
| --- | --- |
| `main.go` | `studio/main.go:34` usa `application.New`, `application.NewService`, AssetOptions, criação explícita da janela e `app.Run`, coerentes com v3. |
| Services/bindings | Os bindings TypeScript são gerados e usados pelo frontend. A conferência por nomes passou; não substitui regeneração e comparação completa de assinaturas/IDs. Contexto e estado exigem correções em C06 a C09. |
| Eventos | `term`, `notesChanged`, `progress`, `narrate` e `voiceInstall` são registrados com tipos. `previewChanged` é a exceção, tratada em S05. `studio/frontend/src/Preview.svelte:63` remove seu listener. |
| Janelas | `studio/main.go:55` define tamanho mínimo, URL local e estado inicial. `studio/desktop.go:86` mostra, restaura e focaliza a janela. Há problemas no processo de preview, não nessa configuração da janela. |
| Single instance | `studio/main.go:42` usa UniqueID estável e trata apenas `--toggle` ou mostrar a janela. Não executa os argumentos recebidos. A ausência de EncryptionKey não constitui falha demonstrada nesse uso limitado. [Referência oficial](https://wails.io/guides/single-instance/). |
| Tray | `studio/desktop.go:44` cria menu e ícone, cancela fechamento quando a preferência de tray está ativa e permite saída explícita por `quitting`. Não foi validada a disponibilidade real do tray em cada desktop Linux. |
| Menus | Existe menu customizado do tray; não existe menu nativo customizado da aplicação. Na dependência Wails v3.0.0 instalada, o arquivo `pkg/application/application_darwin.go`, linha 305, fornece `DefaultApplicationMenu()` quando nenhum foi definido. Não se confirmou ausência do menu básico do macOS. |
| Assets | `studio/main.go:20` incorpora `all:frontend/dist` e usa `AssetFileServerFS`. É o padrão esperado para assets embarcados. Não há necessidade de recomendar `fs.Sub` sem evidência de falha de resolução do framework. |
| Build tags | As receitas usam `production` e variantes de plataforma. O cgo de Kokoro tem tags complementares para Linux/macOS e fallback sem cgo/Windows. As variantes locais de compilação acima passaram. |
| Empacotamento | Há receitas Linux AppImage/deb/rpm/Arch, macOS app/dmg/universal e Windows NSIS/MSIX em `studio/build`. A stack Linux declarada no nFPM corresponde ao build padrão. Ver C17, C21, S02 e S03. |

A organização do Taskfile segue a divisão atual do Wails entre tarefas comuns e específicas por plataforma. A auditoria não executou `wails3 build` porque suas dependências incluem `go mod tidy`, instalação frontend, geração de bindings e ícones. [Referência do build](https://wails.io/guides/build/customization/).

## Achados confirmados

### C01. Alta: ImageData aceita travessia de diretórios

- Local: `studio/settings.go:71`, `studio/settings.go:75`.
- Problema: a validação compara o prefixo do argumento original com a raiz, sem limpar ou resolver o caminho. Para uma raiz `/tmp/notas`, `/tmp/notas/../fora/segredo.txt` passa na validação e é aberto por `os.ReadFile`. Não existe restrição efetiva a imagens.
- Por que importa: uma chamada ao binding pode obter em base64 arquivos legíveis pelo processo fora da pasta de notas, inclusive configurações e credenciais. Isso confirma uma quebra da fronteira pretendida; não demonstra por si só um vetor remoto para chamar o binding.
- Correção sugerida: receber caminho relativo e ler por um `os.Root` aberto na pasta autorizada; validar tipo e tamanho antes de devolver a imagem. A recomendação de contenção também consta no [guia de segurança do Wails](https://wails.io/guides/security/).

### C02. Alta: validações lexicais não contêm links simbólicos

- Local: `studio/studio.go:146`, `studio/studio.go:165`, `studio/studio.go:178`, `internal/app/files_api.go:24`, `internal/notesync/scan.go:113`, `internal/notesync/notesync.go:734`, `internal/notesync/state.go:82`.
- Problema: `notePath` e `notesPath` só verificam caminhos lexicalmente. Um arquivo `.md` ou diretório intermediário que seja symlink pode apontar para fora da raiz. Na sincronização, ignorar symlinks no scan não protege os pais de destinos recebidos: `atalho/nova.md` pode passar em `unchanged` quando `atalho` aponta para um diretório externo e o arquivo ainda não existe.
- Por que importa: leituras, gravações e recebimento de arquivos podem atingir dados fora da pasta selecionada. O rename de um temporário resolve os pais do caminho e também pode atravessar esse symlink.
- Correção sugerida: usar operações relativas a `os.Root` para a fronteira inteira de leitura e escrita. Validar só com `EvalSymlinks` antes de abrir deixa uma janela entre conferência e uso. Adicionar cenários com symlink no arquivo e nos diretórios pais. [Referência oficial](https://wails.io/guides/security/).

### C03. Alta: desfazer snapshot pode remover arquivos que já existiam

- Local: `internal/app/snapshot.go:58`, `internal/app/snapshot.go:119`, `internal/app/snapshot.go:160`.
- Problema: `TakeSnapshot` omite arquivos quando o total excederia 300 MiB, mas não grava um inventário dos arquivos omitidos. `SnapshotChanges` considera qualquer arquivo atual de até 20 MiB ausente no snapshot como `added`; `UndoSnapshot` o envia à lixeira.
- Por que importa: com 15 arquivos de 20 MiB ordenados antes de `z.md`, o snapshot já atinge 300 MiB e omite `z.md`. Desfazer pode retirar essa nota mesmo sem qualquer alteração feita pelo agente. A remoção é para a lixeira, mas ainda altera indevidamente conteúdo preexistente.
- Correção sugerida: persistir um manifesto que diferencie arquivos copiados, omitidos e realmente inexistentes na origem; desfazer apenas os itens cuja classificação seja comprovada. Alternativamente, recusar snapshot incompleto de forma explícita.

### C04. Média: senha própria do site é persistida em arquivo 0644

- Local: `internal/config/config.go:57`, `internal/config/config.go:251`, `internal/app/settings.go:327`.
- Problema: `Site.Password` contém a senha em texto e `Site.Save` grava `blup.toml` com permissão 0644. A política 0600 de `Credentials.Save` não cobre essa senha.
- Por que importa: em diretórios atravessáveis por outros usuários, eles podem ler a senha dos posts protegidos. O arquivo do site costuma ficar fora do diretório privado de credenciais.
- Correção sugerida: mover o segredo para armazenamento privado associado ao site. Como medida intermediária, usar 0600 quando houver senha e proteger backups; em Windows, verificar ACLs, já que o modo Unix não estabelece sozinho a mesma proteção.

### C05. Média: restauração confia em .origin sem validar o destino

- Local: `internal/app/files_api.go:171`, `internal/app/settings.go:399`, `internal/app/settings.go:403`.
- Problema: `inTrash` valida a localização da origem na lixeira, mas `RestoreSection` usa o conteúdo de `.origin` diretamente em `filepath.Join`. Um `.origin` com `../../fora/nova.md` produz destino fora das notas.
- Por que importa: metadados locais alterados ou corrompidos podem fazer a restauração criar ou mover conteúdo para fora da fronteira esperada. É necessário controlar esse arquivo de metadados para disparar o cenário.
- Correção sugerida: validar `.origin` como caminho relativo local e resolver o destino por `os.Root`; rejeitar saída da raiz, symlinks externos e destinos inválidos antes de mover ou limpar a origem.

### C06. Média: o mutex do Studio protege o ponteiro, mas não o Site

- Local: `studio/studio.go:92`, `studio/settings.go:120`, `studio/publish.go:86`, `internal/app/settings.go:161`, `internal/config/config.go:246`.
- Problema: `current` devolve `s.site` depois de liberar o mutex. Os métodos seguintes leem e alteram a mesma struct sem sincronização. Até `Site.Save` altera `Version`. O mutex `busy` cobre somente Publish e Sync, não os demais bindings.
- Por que importa: chamadas sobrepostas de salvar configurações, domínio e publicação podem disputar campos, serializar estado intermediário ou usar destinos diferentes dentro da mesma operação. O bloqueio aplicado apenas pelo frontend não protege a API Go.
- Correção sugerida: definir posse e serialização das mutações por site; fornecer cópias imutáveis para leituras e operações longas e uma etapa controlada de persistência. Evitar manter um mutex genérico bloqueado durante uma chamada de rede inteira.

### C07. Média: Report e idioma são globais mutáveis sem sincronização

- Local: `studio/publish.go:34`, `internal/ui/ui.go:45`, `internal/ui/ui.go:61`, `internal/i18n/i18n.go:18`, `internal/i18n/i18n.go:34`, `studio/about.go:28`, `studio/studio.go:352`.
- Problema: `running` troca `ui.Report`, enquanto outros fluxos podem ler a função. Por exemplo, CreateSite chega a `writeTemplates`, que chama `ui.OK` fora de `busy`. `i18n.Set` e `s.lang` também podem ser escritos enquanto publicação, leitura de idioma ou outras chamadas os consultam.
- Por que importa: existem acessos concorrentes sem relação de sincronização. No caso de Report, o teste `!= nil` e a chamada são leituras separadas, podendo observar a remoção do callback entre elas. O idioma também pode mudar no meio da geração de mensagens e nomes derivados.
- Correção sugerida: passar reporter e idioma como dependências da operação. Se houver estado global, ler uma cópia do callback sob sincronização e usar atomic/mutex para trocar valores, sem chamar callbacks sob o mesmo lock.

### C08. Média: credenciais podem perder alterações concorrentes

- Local: `internal/app/deploy_api.go:61`, `internal/app/deploy_api.go:80`, `internal/app/deploy_api.go:90`, `internal/app/narration_api.go:294`, `internal/app/settings.go:317`, `internal/config/config.go:207`.
- Problema: cada operação carrega a struct inteira de credenciais, altera um campo e salva tudo. Não há transação protegendo o ciclo completo.
- Por que importa: conectar GitHub e OpenRouter simultaneamente pode salvar duas versões baseadas no mesmo estado antigo; a última substitui a chave recém-salva pela outra. O mesmo vale para desconectar ou mudar a senha geral. A gravação atômica impede arquivos parciais, mas não evita perda de atualização, inclusive entre CLI e Studio.
- Correção sugerida: centralizar atualização transacional das credenciais e usar bloqueio entre processos quando houver dois escritores. Preservar 0600 e a substituição por temporário.

### C09. Média: operações longas descartam o contexto da chamada e do app

- Local: `studio/publish.go:105`, `studio/publish.go:128`, `studio/narration.go:41`, `studio/narration.go:95`, `studio/cloud.go:12`, `studio/studio.go:197`, `internal/kokoro/ort_cgo.go:68`.
- Problema: os bindings criam `context.Background()` para operações longas e não recebem `context.Context`. Encerramento/troca de site não cancela publicação, sync, download ou narração em andamento. A inferência ONNX usa `Run` com RunOptions nulo e verifica contexto apenas entre partes no Go.
- Por que importa: cancelar a promessa frontend não interrompe o trabalho subjacente; uma operação iniciada para o site anterior pode continuar fazendo I/O ou gastando recursos. Dentro de uma inferência, nem um contexto cancelado chega ao runtime nativo.
- Correção sugerida: receber o contexto da chamada nos métodos Wails; usar o contexto de ServiceStartup para recursos da aplicação e cancelar operações por site quando necessário. Implementar shutdown com espera limitada das goroutines. Para inferência, considerar RunOptions e terminação suportada pelo ONNX. [Ciclo de vida dos services](https://wails.io/features/bindings/services/).

### C10. Média: início e término do preview deixam estado inconsistente

- Local: `studio/studio.go:190`, `studio/studio.go:219`, `studio/studio.go:249`, `studio/studio.go:259`.
- Problema: StartPreview libera o lock antes de iniciar o processo e só registra `s.preview` após receber a URL. Duas chamadas podem iniciar processos; StopPreview, Open ou shutdown durante esse intervalo não encontram o processo em criação. Quando Hugo encerra sozinho, `cmd.Wait` não limpa `s.preview`/`s.url`.
- Por que importa: pode sobrar preview do site antigo ou retornar uma URL morta em tentativas futuras. A atribuição final pode ocorrer depois de uma troca de site.
- Correção sugerida: modelar estados de início, execução e parada; registrar cancelamento e identidade da execução antes de iniciar I/O. Limpar o estado no término somente se ainda pertencer à mesma execução e aguardar o encerramento controladamente.

### C11. Média: linha longa na saída do Hugo pode bloquear processos e goroutines

- Local: `studio/studio.go:214`, `studio/studio.go:219`, `studio/studio.go:224`, `internal/app/preview.go:43`, `internal/app/preview.go:53`, `internal/app/preview.go:69`.
- Problema: `bufio.Scanner` mantém o limite padrão de token e `sc.Err()` não é tratado. O reader de `io.Pipe` não é fechado quando a leitura termina. Com uma linha acima do limite, o consumidor para e o encaminhamento de stdout/stderr pode ficar preso no pipe; `cmd.Wait` espera esse encaminhamento e o writer só seria fechado após Wait.
- Por que importa: o CLI pode ficar esperando indefinidamente; no Studio, matar o processo no timeout não resolve necessariamente a goroutine presa na cópia para o pipe.
- Correção sugerida: estabelecer uma política explícita para tamanho de linha, verificar `sc.Err()`, fechar o reader para desbloquear escritores e cancelar o processo ao abandonar a leitura. Garantir Wait e fechamento em todos os caminhos, inclusive falha em Start.

### C12. Média: notas e preferências são truncadas no próprio destino

- Local: `studio/studio.go:178`, `studio/notes.go:62`, `studio/layout.go:91`, `studio/layout.go:138`, `studio/layout.go:155`, `internal/obsidian/props.go:142`, `internal/obsidian/props.go:176`.
- Problema: os salvamentos usam `os.WriteFile` no arquivo final. SaveNote ainda salva corpo e propriedades em duas etapas. Preferências fazem leitura, alteração e regravação sem serialização.
- Por que importa: falha durante escrita pode deixar nota ou JSON parcial; leitura concorrente de preferências pode receber JSON truncado e assumir defaults. Duas alterações podem sobrescrever uma à outra. Uma falha nas propriedades pode ocorrer depois de o corpo já ter sido modificado.
- Correção sugerida: montar o documento final em memória e gravar uma vez por temporário único, com substituição e tratamento de erro; preservar permissões e considerar Sync para dados que exijam durabilidade. Serializar atualizações de preferências e usar política de conflito para edição externa de notas.

### C13. Média: downloads e áudio podem consumir memória sem limite efetivo

- Local: `internal/hugo/hugo.go:185`, `internal/pagefind/pagefind.go:131`, `internal/kokoro/install.go:253`, `internal/narrate/openrouter.go:173`, `internal/app/narration_api.go:370`.
- Problema: respostas são acumuladas em memória sem teto de bytes. Kokoro conhece os tamanhos esperados, mas só valida hash depois de ler tudo. O limite de áudio é verificado após geração e acumulação. Hugo, Pagefind e Kokoro usam `http.DefaultClient` sem timeout próprio; Background no Studio também não oferece deadline.
- Por que importa: resposta excessiva, conexão parada ou texto muito grande pode consumir memória ou manter uma operação presa. O SHA-256 protege integridade, mas só depois da alocação.
- Correção sugerida: impor limites com detecção de excesso, deadlines por etapa e tamanho máximo descompactado. Baixar e calcular hash por streaming para temporário; limitar narração antes e durante a geração. Para tamanho variável, definir um teto plausível e verificar `limit+1`.

### C14. Média: caches e instalação compartilham temporários fixos sem exclusão

- Local: `internal/hugo/hugo.go:106`, `internal/pagefind/pagefind.go:67`, `internal/kokoro/install.go:166`, `internal/kokoro/install.go:219`, `internal/kokoro/install.go:226`, `studio/narration.go:40`, `studio/narration.go:47`.
- Problema: downloads usam `.part` determinístico. Install remove esse staging no início e Remove pode apagar a instalação durante outro uso. No final, a instalação antiga é removida antes do rename da nova, sem rollback se esse rename falhar.
- Por que importa: chamadas simultâneas, ou CLI e Studio juntos, podem apagar o staging alheio, substituir binário em uso ou perder uma instalação válida numa falha final. Os métodos de voz não usam a exclusão de `running`.
- Correção sugerida: temporários únicos, exclusão por recurso entre processos e troca com backup/rollback. Impedir remoção enquanto houver sessões ativas ou definir posse explícita do recurso instalado.

### C15. Média: renomear anexo reporta sucesso mesmo sem corrigir os links

- Local: `internal/app/attachments_api.go:106`, `internal/app/attachments_api.go:142`, `internal/app/attachments_api.go:153`.
- Problema: `eachNoteText` descarta falhas de leitura e travessia; o callback de RenameAttachment ignora erros de `os.WriteFile`. O arquivo já foi renomeado quando isso acontece.
- Por que importa: falta de permissão, disco cheio ou outra falha pode deixar notas apontando para o nome antigo, enquanto a API retorna sucesso.
- Correção sugerida: propagar erros da travessia e da escrita; devolver resultado parcial com os arquivos afetados ou aplicar rollback controlado. Gravar cada nota de modo atômico e evitar confundir sucesso do rename com sucesso da operação completa.

### C16. Média: alocação C da saída ONNX não é conferida antes de memcpy

- Local: `internal/kokoro/ort_cgo.go:71`, `internal/kokoro/ort_cgo.go:77`, `internal/kokoro/ort_cgo.go:78`, `internal/kokoro/ort_cgo.go:142`.
- Problema: o wrapper C calcula `count * sizeof(float)`, chama `malloc` e passa o resultado para `memcpy` sem verificar overflow, tamanho máximo ou retorno nulo. Depois, converte count para `int64_t` e `int`.
- Por que importa: sob falta de memória ou saída excessiva, o wrapper pode encerrar todo o Studio por falha nativa. O detector de corrida Go e os testes atuais não exercitam essa condição.
- Correção sugerida: validar tipo/tamanho da saída, multiplicação e conversões; conferir `malloc` e devolver erro pelo mesmo contrato dos status ONNX. Manter liberação de OrtValue e memória em todos os caminhos. Não há evidência de uso indevido de ponteiro Go retido pelo C: os inputs já são copiados para memória C.

### C17. Média: comandos de build Linux não protegem caminhos com espaços

- Local: `studio/build/linux/Taskfile.yml:60`, `studio/build/linux/Taskfile.yml:121`, `studio/build/linux/Taskfile.yml:156`, `studio/build/linux/Taskfile.yml:172`.
- Problema: há expansões como `-o {{.OUTPUT}}`, `-out {{.ROOT_DIR}}/bin`, `-builddir {{.ROOT_DIR}}/...` e `mkdir -p {{.ROOT_DIR}}/...` sem aspas. A raiz deste checkout contém espaços.
- Por que importa: a expansão do template gera vários argumentos de shell; empacotamento pode usar destinos incorretos ou falhar. Em `mkdir`, pode criar múltiplos diretórios. As receitas não foram executadas por serem mutantes.
- Correção sugerida: proteger cada argumento de caminho nas tarefas, inclusive variáveis derivadas e diretórios de saída, e verificar a receita numa cópia temporária cujo caminho tenha espaços.

### C18. Média: automação de testes omite o módulo desktop

- Local: `.github/workflows/release.yml:3`, `.github/workflows/release.yml:25`, `studio/go.mod:1`, `studio/hyprland_test.go:8`.
- Problema: o workflow encontrado roda apenas em tags e executa `go test ./...` na raiz. O módulo aninhado `studio/` fica fora desse padrão. Não há etapa explícita de vet/race ou validação do desktop. O único arquivo de teste Go do Studio cobre helpers de Hyprland e sua cobertura observada é 1,5%.
- Por que importa: uma release do CLI pode passar com regressões de compilação ou de comportamento do Studio. A suíte também não exercita os ciclos de vida e fronteiras apontados nesta revisão.
- Correção sugerida: criar verificação de PR/push dos dois módulos, vet separado e cenários dirigidos a caminhos, snapshot incompleto, cancelamento e chamadas simultâneas. Validar variantes de cgo e plataformas de distribuição conforme a matriz suportada. Evitar usar apenas um percentual global como gate de qualidade.

### C19. Baixa: processos abertos com Start não são coletados com Wait

- Local: `studio/files.go:96`, `internal/ui/ui.go:263`.
- Problema: os helpers iniciam `open`, `xdg-open` ou `rundll32` e não chamam Wait nem outra forma explícita de coleta.
- Por que importa: no aplicativo de longa duração, aberturas repetidas podem acumular processos terminados não coletados e recursos associados. Falhas posteriores ao Start também ficam invisíveis.
- Correção sugerida: preferir a integração de browser do Wails quando aplicável ou iniciar uma goroutine que faça Wait após Start; usar Run se esperar pelo helper for apropriado e limitado.

### C20. Baixa: timer do watcher anterior sobrevive à troca de site

- Local: `studio/terminal.go:230`, `studio/terminal.go:255`, `studio/terminal.go:277`, `studio/studio.go:268`.
- Problema: fechar o watcher encerra seu loop, mas não cancela o `time.AfterFunc` pendente. O callback pode emitir `notesChanged` relativo à raiz anterior depois que outro site abriu. Shutdown também não fecha explicitamente o watcher.
- Por que importa: o frontend recebe invalidação sem identidade do site, provocando recargas e diagnósticos incorretos. Não se demonstrou perda de conteúdo decorrente desse evento.
- Correção sugerida: dar ao watcher contexto, identidade e método de fechamento que pare o timer e espere callbacks; ignorar eventos de gerações anteriores e fechar o recurso no shutdown.

### C21. Baixa: versão exibida no Studio permanece dev nas receitas padrão

- Local: `internal/app/app.go:26`, `studio/about.go:12`, `studio/build/linux/Taskfile.yml:62`, `studio/build/darwin/Taskfile.yml:47`, `studio/build/windows/Taskfile.yml:60`, `studio/build/config.yml:14`.
- Problema: o frontend recebe `app.Version`, cujo valor padrão é `dev`, mas as receitas desktop não passam o `-X ...internal/app.Version=...` usado pelo GoReleaser do CLI. Os manifests declaram 0.1.0 separadamente.
- Por que importa: o binário produzido pelo caminho padrão pode exibir `dev` e dificultar identificação de versão em suporte e diagnóstico, apesar de o pacote anunciar 0.1.0.
- Correção sugerida: definir uma fonte de versão e propagá-la aos ldflags e manifests das três plataformas; verificar o valor como parte da entrega.

## Sugestões e riscos condicionais

### S01. Alta, se exposto: servidor reutiliza a API privilegiada do desktop

- Local: `studio/main.go:37`, `studio/terminal.go:95`, `studio/publish.go:55`, `studio/Taskfile.yml:52`, `studio/build/docker/Dockerfile.server:64`, `studio/build/Taskfile.yml:324`.
- Observação: o target server compila o mesmo Studio. Não há autenticação configurada pelo app. A receita Docker usa `WAILS_SERVER_HOST=0.0.0.0` e publica a porta em todas as interfaces do host. O Wails instalado protege origem/Host, mas isso não autentica alguém que acesse diretamente o próprio servidor.
- Por que importa: se esse target for distribuído/exposto após resolver a receita Docker, clientes podem alcançar métodos de arquivos, configuração e terminal com as permissões do processo. Não foi encontrado ou testado um servidor assim em execução; o desktop normal não equivale a essa exposição.
- Correção sugerida: remover/desabilitar targets server que não façam parte do produto. Se forem necessários, usar um serviço específico, autenticação e autorização antes das rotas privilegiadas e bind restrito por padrão. Revisar também o acesso a eventos/streams. [Guia oficial de segurança](https://wails.io/guides/security/).

### S02. Média: receita Docker precisa ser adaptada ao monorepo

- Local: `studio/build/docker/Dockerfile.server:24`, `studio/build/docker/Dockerfile.server:35`, `studio/build/docker/Dockerfile.server:38`, `studio/build/Taskfile.yml:301`, `studio/go.mod:8`, `studio/go.mod:59`.
- Observação: o contexto Docker é `studio/`; o módulo pai fica fora dele. O Dockerfile remove o `replace` local e passa a depender da resolução remota de `github.com/gabriellsalesx/blup v0.0.0`.
- Por que importa: a imagem não compila necessariamente o código pai deste checkout e depende de esse placeholder existir remotamente. Não se consultou a disponibilidade dessa versão nem se construiu a imagem; o defeito de entrega fica condicionado ao uso desse target.
- Correção sugerida: se o target for mantido, usar o repositório completo como contexto e preservar a relação entre módulos, ou depender de uma versão publicada real e intencionalmente fixada. Confirmar o build em ambiente limpo, sem remover o replace de forma genérica.

### S03. Média: validar assinatura e empacotamento nas plataformas de destino

- Local: `studio/build/darwin/Taskfile.yml:174`, `studio/build/darwin/Taskfile.yml:208`, `studio/build/windows/Taskfile.yml:167`, `studio/build/linux/Taskfile.yml:185`.
- Observação: existem tarefas de assinatura/notarização, mas o diagnóstico local não encontrou identidades/certificados de assinatura. O pacote macOS padrão usa assinatura ad hoc. Não há pipeline desktop no workflow lido.
- Por que importa: um pacote compilado não comprova instalação, confiança do sistema, funcionamento de tray/menu ou compatibilidade com os runtimes das máquinas finais. Isso não prova que releases existentes sejam distribuídas sem assinatura.
- Correção sugerida: documentar e verificar a política de distribuição de cada plataforma; executar testes de instalação e abertura e os passos de assinatura exigidos pela entrega. Tratar ausência de chave nesta máquina como limitação do diagnóstico, sem armazenar segredos no repositório.

### S04. Baixa: reduzir o acoplamento entre CLI, serviço e infraestrutura desktop

- Local: `internal/app/app.go:1`, `internal/app/deploy_api.go:59`, `studio/studio.go:30`, `studio/publish.go:34`, `studio/terminal.go:76`, `studio/desktop.go:38`.
- Observação: `cmd/blup` pequeno e pacotes por domínio em `internal/` são adequados. Contudo, `internal/app` reúne fluxos interativos e API do Studio, e um único service expõe 120 métodos enquanto várias dependências permanecem globais. Algumas fachadas também transformam erros tipados em texto com `errors.New(describe(err))`.
- Por que importa: isso dificulta testar dois contextos independentes e definir posse de estado, contribuindo para C06 a C09. O tamanho ou a quantidade de arquivos, isoladamente, não é um defeito.
- Correção sugerida: extrair dependências de configuração, reporte, processos e filesystem e organizar services por responsabilidade quando isso ajudar os limites de estado. Preservar reutilização das regras de domínio e a causa dos erros, traduzindo a mensagem na fronteira de apresentação. [Services no Wails](https://wails.io/features/bindings/services/).

### S05. Baixa: registrar e tipar também previewChanged

- Local: `studio/studio.go:236`, `studio/frontend/bindings/events/index.ts:11`, `studio/frontend/src/Preview.svelte:62`.
- Observação: o evento é emitido e consumido por string, mas não está entre os tipos registrados. No Wails v3.0.0 instalado, eventos não registrados são permitidos; não se confirmou falha funcional por esse motivo.
- Por que importa: perde-se a checagem de contrato e a constante gerada disponível para os outros eventos.
- Correção sugerida: registrar o evento com o payload pretendido e usar as definições geradas nas chamadas frontend. Regenerar os bindings somente numa etapa autorizada de implementação. [Eventos customizados](https://wails.io/features/events/custom/).

## Controles positivos e limites das conclusões

- `internal/config/config.go:207` e `internal/config/config.go:274` mantêm credenciais em 0600 e criam o diretório privado com 0700; a troca usa temporário único. A exceção da senha própria do site está em C04.
- GitHub e Cloudflare têm clientes com timeout, contexto em requests e erros tipados com Unwrap. GitHub limita tamanho de resposta e implementa espera cancelável e controle de escrita.
- `notesync` usa workers limitados, WaitGroup, conflitos preservados e escrita por temporário com Sync. O problema de pais symlink em C02 permanece mesmo com essas proteções.
- Hugo, Pagefind, updater e Kokoro verificam SHA-256 antes de usar o payload baixado. Kokoro fixa hashes no código e mantém os inputs ONNX em memória C. Em sucesso, manter a biblioteca carregada durante a vida do processo pode ser uma decisão de ciclo de vida válida; não foi classificado como vazamento por si só.
- `internal/lock/lock.go:142` usa salt e nonce aleatórios, derivação de chave e AES-GCM. Os testes incluem compatibilidade com o browser. O build tenta remover o output ao falhar a proteção de posts; não se realizou uma auditoria criptográfica formal.
- Execuções de Hugo, Pagefind, eSpeak e LAME usam argumentos separados e contexto. O texto de narração vai por stdin. A montagem PowerShell da lixeira em `internal/app/delete_site.go:168` escapa aspas simples; não foi confirmada injeção de shell nesses caminhos.
- A busca por padrões conhecidos de chaves privadas e tokens nos arquivos do projeto não encontrou correspondências. Isso não cobre histórico Git, todos os formatos de segredo ou dependências externas e não equivale a certificação de ausência de credenciais.
- A correção de formatação apontada por `gofmt -l` é apenas cosmética e ficou fora da contagem de defeitos. Nenhum fonte foi formatado nesta auditoria.

Prioridade sugerida: corrigir C01 a C03 primeiro, depois proteção de segredos e fronteiras de estado/cancelamento, e acrescentar testes dirigidos aos cenários corrigidos nos dois módulos. Avaliar S01 antes de habilitar qualquer entrega em modo servidor.
