# Blup

O Blup é um editor de notas e publicador de sites. O Studio permite escrever no computador, organizar cofres e criar um blog ou site de produto; a CLI transforma notas Markdown em páginas Hugo. As notas continuam sendo arquivos, compatíveis com o Obsidian. O Blup Cloud acrescenta conta, sincronização cifrada, histórico, IA e hospedagem conforme o plano.

<p align="center"><img src=".github/assets/blup.jpg" alt="Blup ao lado de um caderno cujas páginas viram um site" width="720"></p>

## Documentação

| Guia | Conteúdo |
| --- | --- |
| [Especificação técnica](docs/spec/README.md) | Contratos atuais da CLI, Studio, temas, configuração e serviços |
| [Arquitetura](docs/arquitetura.md) | Componentes, persistência e fluxos com diagramas |
| [Deploy de produção](docs/deploy.md) | Cloudflare, banco, storage, e-mail, Stripe, DNS e distribuição desktop |
| [Segurança](docs/seguranca.md) | Cifra, dados visíveis, ameaças, limites e cotas |
| [Blup Cloud](cloud/README.md) | Desenvolvimento do painel e dos Workers |
| [Testes e build](docs/spec/testes-e-build.md) | Dois módulos Go, frontend, integração e pacotes |

A documentação foi revisada em 02/11/2026 contra o código. Recursos disponíveis no código e configuração de uma conta de produção são coisas distintas; o guia de deploy identifica os passos operacionais.

## O que o projeto oferece

- Studio desktop em Wails v3 e Svelte 5, com editor Markdown, propriedades, busca, anexos, autores, terminal de agentes, lixeira e prévia.
- Cofres sem publicação, com nota do dia, calendário, lembretes no computador, mapa de notas, canvas, desenhos e bancos de dados baseados em arquivos.
- Interface em português, inglês e espanhol, com troca de idioma sem recarregar o Studio.
- Sites estáticos com busca Pagefind, imagens WebP, SEO, RSS, modo claro/escuro, app instalável e leitura offline.
- Posts com corpo protegido por senha no build. Título, URL e capa continuam públicos; anexos exclusivos do corpo podem ser incorporados na cifra, com limites descritos na spec; `hidden` não é controle de acesso.
- Comentários, reações e votos em issues do GitHub, com login pelo serviço [blup-api](https://github.com/gabriellsalesx/blup-api). O Bento tem quadro de pedidos Feedback acrescentado pelo Studio.
- Narração local com Kokoro ou remota com OpenRouter/Blup Cloud, com voz por idioma. Blogs podem agrupar as narrações num Podcast com página e feed.
- Tradução local com Firefox Translations ou remota com IA, gravada em notas por idioma. Sites traduzidos escolhem uma versão existente conforme o visitante e lembram a escolha manual.
- Publicação em Cloudflare Pages, GitHub Pages ou `nome.getblup.site` pelo Blup Cloud. Domínio próprio também no Blup Cloud Max/Ultra, com prova TXT/SSL; URLs absolutas do build continuam em `getblup.site`.
- Sync por repositório privado GitHub, ou pelo Blup Cloud com cifra de ponta a ponta. Conta e senha das notas são mecanismos separados.
- Lembretes genéricos por e-mail no Max, até 50 por mês, e Ultra, até 100, separados das notificações gratuitas do computador.

## Modelos de site

| Modelo | Uso |
| --- | --- |
| Mochi | Blog com destaque, cartões, tags, autores e leitura |
| Nori | Blog em coluna, listas compactas e compartilhamento ao fim dos posts |
| Bento | Documentação, roadmap, changelog com calendário, ajuda, blog e Feedback |

O Studio oferece os três modelos. O assistente de criação da CLI oferece Mochi e Bento. Feedback e Podcast são seções acrescentadas pelo Studio; a CLI continua gerando sites que já tenham essas notas. Os exemplos existem em português, inglês e espanhol.

## Instalar e começar

CLI em macOS/Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

CLI em PowerShell:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Os scripts de instalação inicial buscam a release e conferem somente SHA-256. A autoatualização CLI exige também assinatura Ed25519 dos checksums. O fluxo de distribuição do Studio é separado da release CLI; consulte [builds e assinatura](docs/deploy.md#9-lançamento-do-studio-e-da-cli).

Para a CLI, execute `blup`, escolha o cofre/pasta, tema e dados do site, abra a prévia e depois publique. Uma pasta vazia recebe o tutorial do tema como notas editáveis. O Studio também aceita pastas comuns e oferece criação de cofre sem site.

| Inglês | Português | Operação |
| --- | --- | --- |
| `blup new` | `blup novo` | Criar site |
| `blup preview` | `blup ver` | Abrir prévia |
| `blup build` | `blup gerar` | Gerar `public/` |
| `blup publish` | `blup publicar` | Publicar |
| `blup sync` | `blup sincronizar` | Fluxo interativo de sync GitHub |
| `blup customize` | `blup personalizar` | Personalizar site |
| `blup convert` | `blup converter` | Converter wikilinks com backup |
| `blup language` | `blup idioma` | Trocar idioma da CLI |
| `blup update` | `blup atualizar` | Atualizar CLI; há opção de rollback |

Contas, sync Blup Cloud, calendário, voz e tradução têm suas telas no Studio. A [spec da CLI](docs/spec/cli.md) distingue os comandos dos serviços expostos pelo desktop.

## Dados e privacidade

Notas e configurações técnicas ficam em diretórios separados. Publicar envia a saída estática, não o diretório do cofre inteiro. Sincronizar não publica o conteúdo. No Blup Cloud, notas, nomes de cofres e manifestos são cifrados antes de sair do computador; o servidor ainda vê metadados de uso. IA remota recebe o texto em claro para processar. Não existe recuperação da cifra sem senha, recuperação ou chave local. Consulte [segurança](docs/seguranca.md).

## Desenvolvimento

O módulo Go principal é `github.com/gabriellsalesx/blup`. `studio/` tem seu próprio `go.mod` com `replace` para a raiz. `cloud/` contém Hono, Better Auth, Drizzle/D1, R2, Stripe e painel SvelteKit estático. O tema Hugo fica embutido no executável; não é o painel do Blup Cloud.

```sh
go build -o bin/blup ./cmd/blup
go vet ./...
go test ./...
(cd studio && go vet . && go test ./...)
(cd studio/frontend && npx svelte-check)
(cd cloud && npx tsc -p worker/tsconfig.json --noEmit && npx vitest run)
```

O build desktop requer frontend compilado e dependências nativas. Os testes de Hugo precisam de binário compatível para executar as integrações em vez de pulá-las. Os [comandos de build e empacotamento](docs/spec/testes-e-build.md) detalham isso. Não use configuração de produção ou serviços pagos reais para testes automatizados.

## Licença e contribuição

[GNU AGPL-3.0](LICENSE). Dependências mantêm suas licenças em [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). Ícones de interface usam Lucide. Contribuições seguem [CONTRIBUTING.md](CONTRIBUTING.md) e o [CLA](CLA.md).
