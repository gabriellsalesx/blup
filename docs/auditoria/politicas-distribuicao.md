# Políticas de distribuição conferidas

Consulta somente de leitura em 02/11/2026. Este registro separa regra publicada, observação do código e pendência operacional. Não houve submissão, criação de conta, envio de pacote ou declaração de aprovação.

## Microsoft Store e Partner Center

A Microsoft publica um [quickstart para criar e publicar com assistência de IA](https://learn.microsoft.com/en-us/windows/apps/develop/ai-assisted/quickstart), atualizado em 01/10/2026. Não foi encontrado veto geral a código ou envio assistido por IA na política consultada; isso não dispensa certificação.

A [Store Policies 7.20](https://learn.microsoft.com/en-us/windows/apps/publish/store-policies), vigente desde 22/10/2026, estabelece:

- **10.5.1:** Win32/Desktop Bridge precisa de política de privacidade e URL no Partner Center. Para o Blup, é bloqueante.
- **11.16:** IA dinâmica precisa ser declarada na ficha e submissão, respeitar as políticas e oferecer meio de relato de conteúdo inadequado, com tratamento dos relatos.
- **10.8 e 10.14:** declarar cobrança de terceiros e conferir o tipo de conta exigido para atividade comercial/profissional.

O Studio gera traduções e narrações conforme a entrada da pessoa. O checklist adota a declaração de IA e o canal de relato para esses recursos. Uso de IA no desenvolvimento e recursos de IA oferecidos pelo produto são verificações diferentes.

### Desktop full trust e WebView2

O empacotamento MSIX de um aplicativo desktop em nível `mediumIL` precisa de `runFullTrust`. É capacidade restrita: o mantenedor deve explicar a necessidade em **Submission options** e obter aprovação na certificação. Full trust não significa execução como administrador. Para o Studio, a justificativa deve refletir a edição de arquivos locais, prévia e ferramentas desktop realmente usadas, sem declarar acesso que o produto não precisa. [Capacidades e processo de aprovação](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/app-capability-declarations).

Conferir identidade reservada, Publisher, versão, arquitetura, manifest e o Windows App Certification Kit antes do envio. O WebView2 Runtime precisa estar presente no computador de destino; não basta ter o navegador Edge instalado. [Distribuição WebView2](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution).

A Store pode reassinar MSIX como parte da certificação. O caminho Store por EXE/MSI exige assinatura CA; ele não equivale à distribuição direta sem assinatura definida para este lançamento. [Caminhos e assinatura](https://learn.microsoft.com/windows/apps/package-and-deploy/publish-first-app). A primeira publicação por MSIX passa pelo Partner Center; a automação posterior tem requisitos próprios. [CLI da Store](https://learn.microsoft.com/en-us/windows/apps/publish/msstore-dev-cli/overview).

### Onde ficam privacidade e relato de IA

No código conferido deste repositório, [privacy.go](../../internal/app/privacy.go) gera um modelo para o **site publicado pela pessoa**. Não é a política do Blup Studio ou do Blup Cloud. [Segurança](../seguranca.md) descreve contratos técnicos, sem substituir essa política pública.

Por decisão do dono, política de privacidade e termos serão páginas do novo institucional em Hugo, mantido em repositório privado pelo Confete. URLs **a publicar no site institucional**: `https://getblup.io/privacidade` e `https://getblup.io/termos`. Não são páginas deste painel nem deste repositório. A política precisa descrever conta, arquivos locais, sync cifrado e metadados, IA remota, publicação, terceiros, retenção e controles da pessoa conforme o comportamento real. Cadastrar a URL pública no Partner Center quando estiver no ar.

O canal escolhido para relatos, inclusive de resultados de IA, é `https://getblup.io/feedback`, **a publicar no site institucional**, com contato `help@getblup.io`. Não haverá botão de reportar IA no Studio. O canal institucional precisa receber relatos e ter responsável pelo tratamento; não se confunde com o quadro Feedback dos sites das pessoas. O bloqueio documental para a Store permanece até `/privacidade`, `/termos` e `/feedback` estarem publicados e conferidos. A declaração de IA continua na ficha/submissão.

Texto de loja proposto: Escreva notas em Markdown, organize pastas e links entre notas e crie sites com prévia. O Blup Studio mantém seus arquivos no computador; a conta Blup Cloud oferece serviços opcionais conforme o plano. Tradução e narração por IA são opcionais, com processamento local ou remoto conforme a escolha e o aceite.

## AUR

Foram lidas as [AUR submission guidelines](https://wiki.archlinux.org/title/AUR_submission_guidelines) em inglês por HTTP 200, com revisão indicada em 27/10/2026, e as regras de [.SRCINFO](https://wiki.archlinux.org/title/.SRCINFO). Nas diretrizes examinadas não foi encontrada proibição específica de `PKGBUILD` gerado ou assistido por IA. Isso é resultado de leitura, sem garantir aceitação nem substituir revisão do mantenedor.

Continuam exigidos pacote útil, ausência de duplicação, licença e fontes corretas, nome adequado e responsabilidade sobre o conteúdo enviado. Para binário pré-compilado de software com fontes disponíveis, usar `-bin`. O AUR recebe receita e auxiliares; não recebe o tarball binário. Atualizar `.SRCINFO` com os metadados e testar em ambiente limpo antes de publicar. A página `/tos` redirecionou para login nesta consulta; não se afirma verificação de termos autenticados adicionais.

## Debian e Ubuntu: arquivo .deb fora do repositório

Não há submissão a Debian/Ubuntu neste canal. O mantenedor publica o `.deb` na release do projeto e responde por seu conteúdo, sem apresentá-lo como pacote aprovado pela distribuição.

Conferir nome, versão, arquitetura, descrição e contato operacional de mantenedor no `control`. A lista de dependências precisa corresponder às bibliotecas e ABI do binário testado nas distribuições alvo; empacotar um executável construído em outra distribuição não prova compatibilidade. Usar locais apropriados sob `/usr`, permissões corretas, ícones, desktop entry, MetaInfo e licenças. [Pacotes binários](https://www.debian.org/doc/debian-policy/ch-binary.html), [dependências](https://www.debian.org/doc/debian-policy/ch-relationships.html) e [arquivos](https://www.debian.org/doc/debian-policy/ch-files.html).

Manter scripts de instalação mínimos, repetíveis e sem alterar pastas/configurações pessoais ou buscar código remoto como root. Preferir triggers para menus/ícones quando suficientes. [Scripts de manutenção](https://www.debian.org/doc/debian-policy/ch-maintainerscripts.html).

Antes da publicação: `dpkg-deb --info`, `dpkg-deb --contents`, `lintian` quando disponível e instalação/abertura/atualização/remoção em máquinas limpas de teste. Publicar SHA-256 e assinatura dos checksums e conferir o pacote exato distribuído. A instalação local recomendada usa `sudo apt install ./arquivo.deb`, resolvendo dependências disponíveis nos repositórios configurados. [Debian AptCLI](https://wiki.debian.org/AptCLI), [Ubuntu](https://documentation.ubuntu.com/project/contributors/bug-fix/install-built-packages/).

O Studio instalado como pacote deixa a atualização para esse canal. Sem repositório APT, a pessoa baixa o `.deb` novo. Um repositório APT assinado pode ser considerado depois, para automatizar atualizações; não é requisito para distribuir o arquivo nem faz parte da infraestrutura confirmada.

## Linux nesta rodada

Decisão atual: AppImage, AUR, bundle `.flatpak` no GitHub e `.deb`. **Flathub fora deste lançamento.** Sua [política sobre IA](https://docs.flathub.org/docs/for-app-authors/requirements#generative-ai-policy) proíbe manifests gerados ou assistidos por IA e impõe condições próprias de submissão. Isso não transforma um bundle Flatpak distribuído pelo projeto em pacote do Flathub.
