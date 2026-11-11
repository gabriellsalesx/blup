# Instalar o Blup Studio

Baixe o pacote do seu sistema e arquitetura na [página oficial de releases](https://github.com/gabriellsalesx/blup/releases). Escolha um arquivo identificado como **Blup Studio**. Os arquivos da CLI `blup` são outro programa, usado no terminal. Se uma release não listar o pacote do Studio, ele ainda não foi publicado nela.

Nesta etapa, Windows e macOS são distribuídos sem assinatura de desenvolvedor. As instruções abaixo autorizam a abertura do arquivo específico que você baixou da release oficial.

## Windows

1. Baixe o instalador do Studio para Windows e abra o arquivo.
2. Se aparecer **O Windows protegeu o computador**, clique em **Mais informações**.
3. Confira o nome do arquivo e clique em **Executar assim mesmo**.
4. Siga a instalação e abra o Blup Studio pelo menu Iniciar.

Esse é o caminho do aviso SmartScreen descrito pela [Microsoft](https://learn.microsoft.com/windows/apps/package-and-deploy/publish-first-app). A opção pode estar indisponível em computadores com políticas de administração ou Smart App Control. Nesse caso, procure o administrador; este guia não pede desligar a proteção do Windows.

## macOS

1. Baixe o pacote do Studio para seu Mac. Abra o DMG ou extraia o arquivo, conforme o formato publicado, e mova **Blup Studio** para **Aplicativos**.
2. Tente abrir o app uma vez. Se o macOS bloquear por desenvolvedor não identificado ou ausência de autenticação Apple, feche o aviso.
3. Abra **Ajustes do Sistema > Privacidade e Segurança**.
4. Na parte **Segurança**, encontre o aviso do Blup Studio e escolha **Abrir mesmo assim**. Se a versão mostrar primeiro **Abrir**, clique nele para chegar à confirmação.
5. Confirme **Abrir** quando solicitado e autentique com a senha de início de sessão do Mac, se o sistema pedir.

A Apple informa que essa opção fica disponível por cerca de uma hora após a tentativa. A exceção fica associada ao app, permitindo abri-lo normalmente depois. O procedimento foi conferido no [guia atual do macOS 27](https://support.apple.com/en-asia/guide/mac-help/mh40617/27/mac/27) e no [guia da Apple em português](https://support.apple.com/pt-br/guide/mac-help/-mh40616/mac). Não é necessário executar comandos para desativar o Gatekeeper ou remover a quarentena de todos os arquivos.

## Linux

Os canais definidos são AppImage, AUR, bundle `.flatpak` no GitHub e `.deb`. A publicação no Flathub está [em preparação](flathub.md). A existência dos arquivos no repositório não significa que os pacotes já foram publicados.

### AppImage

Baixe o AppImage da release. Nas propriedades do arquivo, permita executá-lo como programa e abra-o. Pelo terminal, substituindo o nome pelo arquivo baixado:

```sh
chmod +x ./blup-studio.AppImage
./blup-studio.AppImage
```

A compilação Linux usa GTK4, WebKitGTK 6.0 e GStreamer para áudio. A presença de um AppImage não comprova que qualquer distribuição ou arquitetura foi testada. Dependências e limitações do empacotamento estão em [testes e build](spec/testes-e-build.md).

### Bundle Flatpak

Baixe o arquivo `.flatpak` da release e abra-o no instalador de aplicativos da distribuição. Pelo terminal, substitua o nome de exemplo pelo arquivo baixado:

```sh
flatpak --user install ./blup-studio_X.Y.Z_x86_64.flatpak
flatpak --user run io.getblup.Studio
```

O pacote usa `org.gnome.Platform//51`. O Flatpak pode buscar esse runtime no Flathub; o aplicativo vem do arquivo do projeto. Pode ser necessário baixar o runtime mesmo com o bundle em disco. A atualização deste canal é a instalação do bundle novo, sem repositório próprio de atualizações configurado. Veja [bundles Flatpak](https://docs.flatpak.org/en/latest/single-file-bundles.html).

O ID passou a ser `io.getblup.Studio` em 11/11/2026. Quem instalou um bundle anterior com outro ID terá uma instalação separada e precisará reabrir suas pastas de notas. As configurações privadas do Flatpak anterior não são migradas automaticamente; os cuidados estão no [guia](flathub.md#verificação-pelo-domínio).

### AUR

Quando publicado, o pacote será `blup-studio-bin`. Confira a receita no AUR antes de instalá-la pelo fluxo habitual do Arch. Ela baixa o tarball binário da release oficial; não há envio automático ao AUR pelo workflow do GitHub. Os passos do mantenedor estão no [checklist](deploy-checklist.md#12-publicar-os-canais-linux).

### Debian e Ubuntu

O `.deb` desta rodada destina-se a Ubuntu 24.04 ou mais novo e Debian 13 ou mais novo, em amd64. Baixe o arquivo da release. Para instalar com suas dependências:

```sh
sudo apt install ./blup-studio_X.Y.Z_amd64.deb
```

São exigidos GTK 4.10 ou mais novo e WebKitGTK 6.0; GStreamer Good é recomendado para áudio. Sem um repositório APT do projeto, atualize baixando e instalando o `.deb` da próxima release. O pacote é distribuído pelo Blup, sem submissão aos repositórios Debian ou Ubuntu.

## Primeira abertura e atualizações

Você pode criar um site ou abrir uma pasta de notas. A conta Blup Cloud é opcional para o trabalho local. Para usar a nuvem, o Studio abre o navegador para conferir e aprovar o código do aparelho.

Na instalação direta e no AppImage, o Studio consulta novas versões na aba Sobre; não substitui seu executável automaticamente. Quando instalado por Flatpak, pacote do sistema ou Microsoft Store, deixa a atualização para esse canal e não procura releases no GitHub. Instale a nova versão pelo canal usado, preservando suas notas. O mecanismo `blup update` pertence à CLI: exige assinatura Ed25519 dos checksums e SHA-256 do arquivo. Os instaladores iniciais da CLI continuam conferindo somente SHA-256.
