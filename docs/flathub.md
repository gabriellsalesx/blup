# Preparar o Blup Studio para o Flathub

O alvo é o aplicativo desktop de `studio/`, em Go/Wails 3 e Svelte 5. Ele compartilha o módulo Go da raiz com a CLI, incorpora o tema Hugo e o frontend compilado e mantém as notas em arquivos locais. Blup Cloud é um serviço opcional; o aplicativo Android de `mobile/` tem outro processo de distribuição.

O Flathub é um novo canal pretendido em 11/11/2026. O bundle direto do GitHub já foi construído, instalado temporariamente e testado por linha de comando e pela interface na sandbox GNOME 51. Ainda faltam o empacotamento a partir das fontes para a loja e a conclusão dos testes de portais antes da submissão.

## Responsabilidade pela submissão

A [política atual de IA](https://docs.flathub.org/docs/for-app-authors/requirements#generative-ai-policy), conferida em 11/11/2026, exige declarar material gerado por IA e sua extensão. Ela proíbe assistência de IA no manifesto e na autoria ou automação do PR de submissão, incluindo mensagens de commit e respostas da revisão. O mantenedor precisa escrever o manifesto e conduzir essa interação pessoalmente. Código da aplicação com assistência de IA é avaliado pelos revisores após declaração; sua presença não estabelece aprovação nem rejeição automática.

Nesta preparação, a configuração de identidade Flatpak, os metadados e o empacotamento do Studio, as adaptações de ferramentas, inicialização automática, atalhos e terminal, os testes e esta documentação receberam assistência de IA. Os avisos novos do terminal e seus bindings também receberam essa assistência. O auxiliar de demonstração e os textos de exemplo das capturas foram escritos com essa assistência; as imagens são capturas nativas da interface em execução, sem edição. Nenhum manifesto, texto de PR ou mensagem de commit para o Flathub foi criado. A declaração do mantenedor também precisa abranger o histórico real do restante do projeto.

## Base existente e pendências

| Item | Situação no projeto | Próxima ação |
| --- | --- | --- |
| Fontes | Repositório `gabriellsalesx/blup`, licença AGPL-3.0-only, tags até `v1.0.8`. As adaptações para o Flatpak estão na `main`, à espera da próxima release. | Escolher uma release estável com todas as correções necessárias e conferir o commit exato publicado. |
| Identidade | `io.getblup.Studio`, adotado em 11/11/2026. Configuração Wails, metadados Linux/macOS/Windows e templates iOS do Studio usam esse identificador. | Preservar o ID no pacote do Flathub e verificar o domínio `getblup.io`. |
| Menu e ícones | `studio/build/linux/blup-studio.desktop` e PNGs de 128, 256 e 512 pixels estão presentes. | Conferir os nomes exportados pelo pacote final e o ícone na janela Wayland. |
| MetaInfo | `studio/build/linux/io.getblup.Studio.metainfo.xml` declara licença, desenvolvedor, descrições e legendas em três idiomas, três capturas nativas fixadas em commit e a lista real de versões e datas (sem placeholders; o fluxo de release recusa uma tag ausente da lista). | Conferir a disponibilidade pública das capturas e o desktop ID instalado no pacote final. |
| Atualizações | `internal/app/channel.go` reconhece `FLATPAK_ID` e `/.flatpak-info`; uma sonda dentro da sandbox confirmou o canal `flatpak`. | Conferir também a apresentação em Sobre no pacote final. |
| Identidade da janela | A janela instalada foi identificada no Hyprland como `io.getblup.Studio`, com o processo pertencendo à instalação temporária. | Conferir o ícone e o comportamento da segunda inicialização no pacote final. |
| Runtime | GNOME Platform 51, GNOME SDK 51 e `org.flatpak.Builder` foram instalados no diretório local de build ignorado pelo Git. A interface e as ferramentas do bundle foram exercitadas. | Preparar o build a partir das fontes e concluir os testes de portais. |

Os [IDs de aplicação](https://docs.flathub.org/docs/for-app-authors/requirements#application-id) vinculam o pacote a um domínio ou repositório. As [diretrizes de MetaInfo](https://docs.flathub.org/docs/for-app-authors/metainfo-guidelines#screenshots) exigem capturas de aplicativos gráficos e URLs fixadas em tag ou commit quando hospedadas em Git. As capturas da interface estão em `.github/assets/screenshots/`; as outras imagens de temas não as substituem.

## Capturas da interface

As três capturas mostram o Studio nativo no Linux, em inglês: notas no claro, notas no escuro e editor de site com a prévia ao lado. Foram revisadas visualmente e contêm somente textos de exemplo. Foram capturadas sem o cursor do mouse e com a barra lateral padrão (só ícones). Os PNGs originais têm 2176 × 1376 pixels, sem metadados de texto e sem edição da imagem. O MetaInfo aponta para o commit `1d51db056ea1453c53ef4986e269e1a0861364fc`, publicado no GitHub em 11/11/2026. A API do GitHub confirmou os três arquivos e suas URLs de download fixadas nesse commit.

O [auxiliar de captura](../studio/tools/storepreview/README.md) usa outro identificador de instância na demonstração nativa e uma instalação temporária com o ID real na demonstração Flatpak. Esta recusa ativar um Blup Flatpak já em execução. Ambas usam diretórios temporários para configuração, notas e cache, sem copiar notas ou credenciais da pessoa nem importar tokens do ambiente. As referências da sessão gráfica, os perfis temporários, os executáveis e os logs não fazem parte dos commits. No Hyprland, a captura identifica a janela pelo processo, confere sua instalação e o foco antes de capturar apenas sua área; em outros ambientes, usar a ferramenta de captura de janela do sistema.

Nesta preparação, as tarefas foram executadas pelo MCP do Wails com acesso à sessão gráfica. Além das capturas nativas publicadas, a janela do Flatpak foi capturada e revisada com o editor e a prévia do site em funcionamento. A evidência deste teste fica em `studio/bin/flatpak-test-proof/`, ignorada pelo Git.

## Verificação pelo domínio

O ID `io.getblup.Studio` associa o aplicativo a `getblup.io`, configurado no Hugo e no Worker do site institucional. A verificação depende do controle desse domínio; o nome do repositório de fontes pode continuar `gabriellsalesx/blup`.

Depois de receber acesso ao repositório do aplicativo no Flathub, o mantenedor abre **Developer Portal > aplicativo > Verification** e obtém o token real. A [documentação de verificação](https://docs.flathub.org/docs/for-app-authors/verification) oferece duas opções:

- **Arquivo HTTPS:** disponibilizar o token em `https://getblup.io/.well-known/org.flathub.VerifiedApps.txt`. No checkout institucional `../site`, o arquivo de origem é `static/.well-known/org.flathub.VerifiedApps.txt`, copiado para `public/.well-known/` pelo Hugo e publicado como asset estático do Worker. Criar o arquivo quando houver o token real e conferir a resposta pública depois do deploy.
- **DNS:** adicionar um registro TXT em `_flathub.getblup.io` com o token fornecido pelo Flathub, preservando valores já existentes. Aguardar a propagação antes de pedir a verificação no portal.

Em 11/11/2026, o token ainda não foi obtido e nenhuma prova de domínio foi publicada nesta preparação. O arquivo HTTPS e o TXT são alternativas; não é necessário configurar ambos. Uma mudança de ID depois da aceitação exige nova submissão e tem [restrições próprias](https://docs.flathub.org/docs/for-app-authors/requirements#renaming-flatpak-id).

Para quem já instalou o bundle anterior `io.github.gabriellsalesx.blupstudio`, o novo ID representa outro aplicativo Flatpak. O estado privado fica em `~/.var/app/io.getblup.Studio/`; configurações do ID anterior não são migradas automaticamente. As pastas de notas continuam onde foram salvas e precisam ser reabertas pelo seletor. Se houver estado local a preservar, avaliar sua cópia com ambos os aplicativos fechados, mantendo o diretório anterior até conferir a nova instalação.

## Compilação e ferramentas

O [bundle.sh](../studio/build/linux/flatpak/bundle.sh) copia o Studio, Hugo e Pagefind já compilados para o runtime. Ele serve à distribuição direta e não comprova o build a partir das fontes exigido pelo Flathub. O [check-bundle.sh](../studio/build/linux/flatpak/check-bundle.sh) instala o bundle temporariamente e verifica bibliotecas, executa as ferramentas e gera uma página e seu índice sem rede. Esse teste não abre a janela nem exercita os portais desktop.

O bundle agora compõe também o catálogo AppStream em `share/app-info/`, além de instalar o MetaInfo original. A composição usa `appstreamcli compose` sem rede; o workflow inclui o pacote `appstream-compose` necessário no Ubuntu. Isso corrigiu o erro `appstream-missing-appinfo-file` encontrado pelo linter do repositório.

Para o bundle direto do GitHub, o workflow prepara as ferramentas com `go run ./scripts/flatpak-tools -out studio/bin`. Esse comando usa os downloads com checksum da aplicação, e o bundle exige os três executáveis antes de iniciar. No Flathub, o mantenedor deve compilar Hugo e Pagefind a partir das fontes no ambiente de empacotamento.

As tarefas locais, executadas a partir de `studio/`, são:

```sh
wails3 task build VERSION=X.Y.Z
wails3 task linux:prepare:flatpak:tools
wails3 task linux:setup:flatpak:runtime
wails3 task linux:validate:metainfo
wails3 task linux:bundle:flatpak VERSION=X.Y.Z
wails3 task linux:check:flatpak RUNTIME_DIR="$PWD/bin/flatpak-runtime"
```

Substituir `X.Y.Z` pela versão que já consta no MetaInfo, com sua data real. A tarefa de runtime usa `studio/bin/flatpak-runtime`, ignorado pelo Git, e não instala nada na instalação normal do usuário. O teste usa outra instalação temporária, lê o runtime local e apaga sua instalação ao terminar. `FLATPAK_OUTPUT` e `FLATPAK_BUNDLE` selecionam os arquivos de saída e de teste. `METAINFO` permite usar uma cópia temporária para testes de pré-lançamento; sem essa opção, o bundle continua exigindo a versão no arquivo do repositório.

`linux:setup:flatpak:builder` prepara o SDK e o Builder oficial. `linux:lint:flatpak:metainfo` valida o MetaInfo com o linter do Flathub. `linux:lint:flatpak:bundle` importa o bundle num repositório temporário e executa a análise do repositório; seu resultado não valida um manifesto de build a partir das fontes.

Para publicar somente as fontes no GitHub, `wails3 task sources:publish` simula o envio. Depois de conferir os commits, `wails3 task sources:publish PUBLISH=true` envia a `main` para a URL explícita do GitHub, sem usar o segundo destino de push do `origin`. A tarefa exige a `main` sem alterações rastreadas pendentes e não cria tags nem releases.

O build para a loja precisa resolver estes pontos:

- Compilar os dois módulos Go juntos, preservando o `replace ... => ../` de `studio/go.mod`. A versão mínima declarada é Go 1.27.3.
- Preparar dependências Go e npm a partir dos lockfiles dos dois módulos e de `studio/frontend/package-lock.json`. A compilação no Flathub ocorre sem acesso à rede; as tarefas atuais executam instalação de dependências e algumas chamam `go mod tidy`. O mantenedor deve usar ferramentas oficiais de preparação de dependências, como os [geradores do Flatpak](https://github.com/flatpak/flatpak-builder-tools), em seu fluxo de empacotamento.
- Compilar o frontend antes do Go, porque `studio/main.go` incorpora `frontend/dist`. Usar as flags de produção e injetar a versão publicada conforme [testes e build](spec/testes-e-build.md).
- Compilar e incluir Hugo, usado pela prévia e publicação. `internal/hugo/hugo.go` exige uma versão compatível no PATH do Flatpak; nesse canal, não usa downloads antigos do cache nem baixa executáveis. Nas instalações nativas, mantém o comportamento anterior.
- Compilar e incluir Pagefind na versão fixada em `internal/pagefind/pagefind.go`. A aplicação procura essa versão no PATH; no Flatpak, não usa downloads antigos do cache nem baixa executáveis. Um teste de publicação ainda deve confirmar a geração real do índice.
- Avaliar separadamente os motores e modelos opcionais de voz e tradução locais. Os downloads atuais e suas dependências nativas não foram demonstrados dentro da sandbox.

## Integração com a sandbox

O seletor de pastas usa os diálogos do Wails com GTK4. A permissão persistente sobre uma pasta escolhida, a reabertura depois de reiniciar e o monitoramento de alterações precisam ser testados com os [portais desktop](https://docs.flatpak.org/en/latest/desktop-integration.html#portals). As permissões amplas usadas no bundle atual não devem ser tomadas como aprovadas para a loja.

As adaptações da aplicação já implementadas precisam ser conferidas no pacote instalado:

- **Terminal e agentes:** no Flatpak, `studio/terminal.go` usa o Bash do runtime sem carregar arquivos de inicialização do host. Os agentes disponíveis são os do PATH da sandbox. Terminal e Configurações explicam essa limitação em português, inglês e espanhol e orientam usar o terminal do computador para agentes instalados fora do Flatpak. A aplicação não solicita acesso à execução no host.
- **Abrir com o computador:** no Flatpak, `studio/background_linux.go` usa `org.freedesktop.portal.Background.RequestBackground`. Somente uma resposta de autorização registra a preferência; recusa e cancelamento não ativam a opção. A aplicação guarda a última autorização em seu próprio diretório de configuração, pois o portal não oferece consulta do estado atual de autostart. Uma revogação externa exige nova confirmação no sistema. As instalações nativas mantêm a implementação anterior.
- **Atalho global:** no Flatpak, a aplicação deixa o Wails usar seu backend de atalhos, que usa `GlobalShortcuts` no Wayland. Não tenta escrever configuração do Hyprland. A disponibilidade do portal e a combinação de teclas concedida dependem do ambiente desktop.

Validar também notificações e seus cliques, bandeja, login pelo navegador, sync, reprodução de áudio, importação/exportação e publicação de um site de teste. Não usar notas pessoais ou serviços pagos reais nos testes automatizados.

## Validação realizada

Em 11/11/2026:

- `go mod verify` em `studio/`: todos os módulos verificados.
- `go test ./...` em `studio/`: passou, com Go 1.27.3 e GTK4/WebKitGTK nativos. Houve avisos de APIs X11 obsoletas no Wails.
- Build Go de produção com tag `production`, versão injetada para a verificação e saída em `/tmp`: passou. Usou o frontend já compilado no checkout; ainda falta recompilá-lo no ambiente de empacotamento sem rede.
- `appstreamcli validate --no-net --strict`: passa no arquivo do repositório como ele está, com as três capturas e a lista real de versões. A verificação pedantic sugere ID inteiramente minúsculo, mas não bloqueia a validação; foi mantida a grafia escolhida. A opção sem rede não comprova disponibilidade pública das imagens.
- `desktop-file-validate`: passou, com sugestão de categoria adicional `Utility`.
- Identificadores e sintaxe de XML, plists e YAML conferidos; script do bundle passou em `bash -n`. Os testes `TestUpdatesBy` e `TestStoreBuildDoesntLookUp` passaram com o novo ID no caso Flatpak.
- Os testes de Hugo/Pagefind cobrem seleção da ferramenta do pacote, rejeição de versões incompatíveis e recusa do cache e do download de executáveis no Flatpak. Os testes desktop cobrem shell, proteção da configuração do host e respostas de aprovação, recusa e cancelamento do portal com uma implementação simulada.
- O frontend passou no `svelte-check`, nos 16 arquivos de testes Node e no build Vite com os novos avisos em três idiomas.
- `go test ./...` em `studio/` passou novamente após adicionar o auxiliar de capturas, incluindo testes do isolamento dos exemplos e da exclusão de credenciais fornecidas no arquivo de sessão. As três capturas nativas foram obtidas e revisadas; o processo de demonstração foi encerrado.
- Frontend de produção recompilado e binário Go com `-tags production`, `-trimpath` e versão `1.1.0-test.1` construído. A versão de teste foi acrescentada somente numa cópia temporária do MetaInfo; o histórico de releases do repositório foi preservado.
- Bundle `1.1.0-test.1` x86_64 construído com GNOME 51, instalado numa instalação temporária e testado com a rede desativada: bibliotecas do Studio resolvidas, Hugo 0.169.0 e Pagefind 1.5.2 executados, uma página HTML gerada e um índice de busca criado. A mesma verificação inicial também passou com o runtime 49. Os pacotes, caches e logs não entram nos commits.
- As fontes foram enviadas à `main` do GitHub, incluindo os commits das capturas e do MetaInfo corrigido. O envio avançou de `667e2fc` a `e8b8afc`, sem forçar o histórico e sem enviar ao segundo destino de push. O commit remoto foi conferido pela API. A validação estrita do MetaInfo com rede habilitada também passou; permaneceu somente a sugestão pedante sobre a grafia do ID.
- A interface do Flatpak abriu com exemplos isolados; editor e prévia do site foram revisados visualmente. `store:exercise:flatpak` passou em edição e salvamento, criação de nota e comando no terminal, conferindo os arquivos resultantes. O terminal confirmou `FLATPAK_ID=io.getblup.Studio` e `SHELL=/bin/bash`.
- A sonda executada dentro da sandbox confirmou detecção Flatpak, canal de atualizações, escolha do Hugo/Pagefind do pacote e recusa de download de executáveis. FileChooser, GlobalShortcuts, OpenURI e Notification anunciaram suas interfaces. Background ficou indisponível nesta sessão Hyprland. A tentativa de fechar a solicitação FileChooser retornou objeto inexistente e não recebeu resposta; o cancelamento e a seleção persistente de pastas permanecem sem validação. A sessão também reportou avisos do registro de acessibilidade AT-SPI.
- O linter oficial do Flathub aprovou o MetaInfo. Após incluir o catálogo AppStream, o linter do repositório do bundle deixou somente `appstream-external-screenshot-url` e `appstream-screenshots-not-mirrored-in-ostree`. O [espelhamento das capturas](https://docs.flathub.org/docs/for-app-authors/linter#appstream-external-screenshot-url) precisa ocorrer no build da loja, com o fluxo oficial do Builder. O bundle direto mantém as URLs públicas reais das capturas; esse resultado não é uma aprovação para inclusão no Flathub.

O ambiente tinha `GOROOT` apontando para Go 1.27.4 enquanto o comando selecionava 1.27.3. A execução dos testes usou explicitamente a toolchain e o GOROOT 1.27.3, com cache em `/tmp`, sem mudar a configuração da máquina.

Ainda faltam manifesto produzido sem IA, build completo a partir das fontes sem rede, linters do manifesto e do repositório resultante, release estável e conclusão dos testes de portais e demais integrações. Outra arquitetura deve ser conferida quando oferecida. A execução pelo shell restrito bloqueia criação de sockets; as capturas e os testes reais foram viabilizados pelas tarefas do MCP do Wails. Os testes do bundle cobrem o núcleo da interface e as ferramentas, mas não demonstram todas as integrações desktop.

## Sequência para publicação

1. Usar `io.getblup.Studio` em toda a distribuição do Studio e confirmar controle de `getblup.io`. O mantenedor confere a compatibilidade da submissão com a política de IA, considerando todo o histórico do aplicativo.
2. Publicar uma release estável com as adaptações da aplicação, capturas e metadados. O histórico local começa em 27/10/2026; os revisores também avaliam maturidade e manutenção conforme os [requisitos de inclusão](https://docs.flathub.org/docs/for-app-authors/requirements#insufficient-development-history).
3. O mantenedor escreve o manifesto e prepara as dependências com ferramentas oficiais. O aplicativo deve compilar a partir de uma fonte pública fixada e executar com as permissões efetivas da sandbox.
4. Construir, instalar, testar e executar os linters de manifesto, repositório e AppStream, seguindo a [documentação de submissão](https://docs.flathub.org/docs/for-app-authors/submission).
5. O mantenedor abre pessoalmente o pedido no GitHub, com base em `flathub/flathub:new-pr`, preenche o formulário e conduz a revisão. A entrada na loja depende da aprovação do Flathub.
