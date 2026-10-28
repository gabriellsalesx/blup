<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/logo-dark.svg">
    <img src=".github/assets/logo.svg" alt="blup" width="220">
  </picture>
</p>

<p align="center"><b>Suas notas do Obsidian viram um site.</b><br>Grátis, rápido e funciona offline. Um comando para publicar.</p>

<p align="center"><a href="README.md">Read in English</a></p>

<p align="center"><img src=".github/assets/blup.jpg" alt="O Blup, uma bolha azul, ao lado de um caderno aberto cujas páginas voam até um site" width="720"></p>

O Blup é um programa de terminal que transforma uma pasta do seu cofre do Obsidian num site: um blog pessoal ou um site de produto, com documentação, roadmap, changelog e central de ajuda. Você continua escrevendo no Obsidian. O Blup gera as páginas, a busca, as imagens e o SEO, e publica de graça no Cloudflare Pages ou no GitHub Pages.

Não precisa saber programar. O Blup faz uma pergunta de cada vez, cada uma com a resposta sugerida, em português ou inglês.

## O que vem pronto

- **Dois temas**, cada um com um site de exemplo completo que ensina a usar o Blup:
  - **Mochi**: blog pessoal com post em destaque, tags, autores, posts relacionados e barra de leitura.
  - **Bento**: site de produto com documentação, roadmap, changelog, central de ajuda e blog. Cada parte é opcional.
- **Do jeito do Obsidian**: propriedades, imagens, links entre notas, callouts, fórmulas, notas de rodapé, tarefas e `%%comentários%%` que não vão para o site.
- **Embutido**: busca no site, tema claro e escuro, imagens em WebP, imagem de compartilhamento para cada página, RSS e mapa do site.
- **Posts com senha**, criptografados quando o site é gerado. Sem servidor.
- **Comentários e reações** no Bento: votos no roadmap, reações no changelog e comentários nos posts, guardados em issues de um repositório público `<site>-feedback` que o Blup cria. Quem visita entra com o GitHub e participa sem sair do site (pelo [blup-api](https://github.com/gabriellsalesx/blup-api)). Liga e desliga em `blup personalizar`.
- **Autores**: uma nota por pessoa, com foto, cargo, redes e bio.
- **Offline primeiro**: as notas ficam no seu computador. Sincronize com um repositório privado no GitHub quando tiver internet e continue em outro computador. Sem git.
- **Hospedagem grátis** no Cloudflare Pages (`seusite.pages.dev`) ou no GitHub Pages, e domínio próprio se você quiser.
- **Tudo se configura pelo Obsidian ou pela CLI**: nome, menu, logo, favicon, redes e domínio são propriedades de uma nota ou respostas a perguntas.

| Mochi | Bento |
| --- | --- |
| <img src=".github/assets/mochi.webp" alt="Tema Mochi: página inicial do blog"> | <img src=".github/assets/bento.webp" alt="Tema Bento: documentação com menu lateral"> |

## Instalar

macOS e Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Os scripts baixam a última versão publicada, conferem o SHA-256 e instalam sem pedir senha de administrador.

Com o Go 1.27 ou mais novo, também dá para usar:

```sh
go install github.com/gabriellsalesx/blup/cmd/blup@latest
```

## Seu primeiro site em cinco minutos

1. Rode `blup`. Na primeira vez ele pergunta o idioma e abre o assistente de site novo.
2. Escolha o cofre do Obsidian e a pasta que vira site (ou o cofre inteiro).
3. Escolha o tema, dê um nome ao site e aperte Enter nas sugestões.
4. Deixe o Blup ajustar o Obsidian para os links funcionarem na web (ele guarda uma cópia das suas configurações).
5. Abra a prévia: o site aparece no navegador e atualiza a cada nota salva.
6. Quando estiver bom, rode `blup publicar` e escolha Cloudflare Pages ou GitHub Pages.

Se a pasta estiver vazia, o Blup a preenche com o site de exemplo do tema escolhido: um tutorial completo escrito como notas comuns. Leia, abra as notas no Obsidian para ver como cada página foi feita, depois edite e troque pelo seu conteúdo.

Errou alguma resposta? Aperte **Esc** para voltar uma pergunta. **Ctrl+C** sai.

## Comandos

Rodar só `blup` abre um menu com tudo. Cada ação também tem um comando, em português ou inglês:

| Português | Inglês | O que faz |
| --- | --- | --- |
| `blup novo` | `blup new` | Cria um site novo |
| `blup ver` | `blup preview` | Prévia no navegador, que atualiza sozinha |
| `blup publicar` | `blup publish` | Coloca o site no ar |
| `blup personalizar` | `blup customize` | Menu, seções, redes, logo, favicon, autor, domínio próprio, senha, comentários |
| `blup sincronizar` | `blup sync` | Guarda as notas num repositório privado no GitHub |
| `blup converter` | `blup convert` | Troca `[[wikilinks]]` por links padrão |
| `blup gerar` | `blup build` | Gera o site em `public/`, sem perguntas; `--url https://site.pages.dev/bento/ --saida ../landing/bento` gera para uma subpasta e copia para outra pasta |
| `blup idioma` | `blup language` | Troca o idioma do Blup |
| `blup atualizar` | `blup update` | Atualiza o Blup (`--rollback` volta uma versão) |

## Escrevendo

Cada nota da pasta do site vira uma página; cada subpasta vira uma seção. Propriedades opcionais no topo da nota controlam o resto:

| Propriedade | O que faz |
| --- | --- |
| `title`, `description` | Título e o resumo que aparece nas listas, no Google e nas redes |
| `date`, `tags`, `image` | Data do post, assuntos, capa (`"[[minha-capa.jpg]]"`) |
| `authors` | Quem escreveu, como `[Ana, Bruno]` |
| `draft: true` | Só na prévia, nunca publicado |
| `hidden: true` | Publicado, mas fora das listas e do Google |
| `password` | Tranca o post; `password: sim` usa a senha geral |
| `weight` | Ordem nos menus e na documentação |

As configurações do site ficam nas propriedades da nota `_index` da pasta do site: `site_name`, `site_description`, `logo`, `favicon`, `menu`, `social`, `button_text` e `button_link`. O site de exemplo explica cada uma.

## Suas notas e seu site

Ficam separados de propósito:

```
você escreve no Obsidian
        │
        ├── blup sincronizar ──► repositório PRIVADO (só as notas)
        │                          └─ outro computador baixa e continua escrevendo
        │
        └── blup publicar ─────► site PÚBLICO (só as páginas geradas)
```

Rascunhos, notas particulares e o texto aberto dos posts com senha ficam nas notas. O site público recebe só o que deve mostrar.

## Atualizações

Uma vez por dia, ao abrir o menu, o Blup confere se há versão nova e sempre pergunta antes de atualizar. Os temas vêm dentro do Blup, então os sites recebem o tema novo na próxima prévia ou publicação. `blup atualizar --rollback` volta para a versão anterior.

---

## Desenvolver

```sh
go build -o bin/blup ./cmd/blup
./bin/blup        # abre o menu
go test ./...
```

O Blup baixa o Hugo e o Pagefind sozinho na primeira vez que precisa (versões fixas, conferidas por SHA-256).

| Pasta | O quê |
| --- | --- |
| `cmd/blup` | ponto de entrada |
| `internal/app` | menu, assistente, prévia, publicação, personalizar, sincronização; `starter/` tem os sites de exemplo |
| `internal/i18n` | todos os textos da CLI (EN/PT) |
| `internal/obsidian` | cofres, configurações do Obsidian, propriedades das notas |
| `internal/scaffold` | gera o `hugo.toml` e copia o tema |
| `internal/deploy/github`, `internal/deploy/cfpages` | GitHub Pages e Cloudflare Pages pelas APIs, sem git nem wrangler |
| `internal/notesync` | sincronização das notas com um repositório privado |
| `internal/lock` | criptografia dos posts com senha |
| `theme/` | o tema Hugo, embutido no executável |

### Publicar uma versão

1. Escreva as notas da versão (PT e EN).
2. `git tag v1.2.3 && git push origin v1.2.3`
3. O GitHub Actions roda os testes e o GoReleaser publica os executáveis e o `checksums.txt`.

## Licença

[MIT](LICENSE). Os ícones são da [Lucide](https://lucide.dev) (ISC, veja `theme/licenses`).
