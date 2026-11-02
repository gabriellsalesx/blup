package app

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// The guide for AI agents run in Blup Studio's terminal (Claude Code,
// Codex, OpenCode, Gemini, Antigravity): AGENTS.md at the top of the notes
// folder explains how a Blup site works, so they write notes the site
// understands. CLAUDE.md and GEMINI.md point to it. They are never pages
// (scaffold.AgentGuides) and are not rewritten once there.

// WriteAgentGuides creates the guides that are missing.
func WriteAgentGuides(site *config.Site) error {
	guide := agentGuideEN
	switch site.Lang {
	case "pt":
		guide = agentGuidePT
	case "es":
		guide = agentGuideES
	}
	for _, name := range scaffold.AgentGuides {
		body := "@AGENTS.md\n"
		if name == "AGENTS.md" {
			body = guide
		}
		p := filepath.Join(site.ContentDir(), name)
		// Lstat: a link there (to nothing, in a downloaded folder) is left
		// alone, not followed to write the guide wherever it points.
		if _, err := os.Lstat(p); err == nil {
			continue
		}
		if err := writeNew(site.ContentDir(), p, []byte(body)); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
	}
	return nil
}

const agentGuidePT = `# Este é um site do Blup

Esta pasta é um site. O Blup transforma as notas em páginas com o Hugo, e o
Blup Studio mostra a prévia ao vivo. Escreva sempre em português.

## Como as notas viram o site

- Cada nota .md vira uma página; cada pasta vira uma seção.
- A nota _index.md de uma pasta é a página da seção.
- A _index.md da raiz guarda as configurações do site (site_name,
  site_description, logo, favicon, menu, social, button_text, button_link).
  Só mexa nela se pedirem.

## Propriedades (no topo da nota, em YAML)

- title, description: título e resumo.
- date: data no formato 2026-10-29.
- tags: [Uma, Outra]; authors: [Nome].
- image: "[[capa.jpg]]" (capa).
- draft: true (só na prévia); hidden: true (fora das listas e do Google).
- password: sim (senha geral do site) ou uma senha própria.
- weight: ordem nos menus (menor primeiro).

## Seções do tema Bento

A _index.md de cada seção tem a propriedade blup: docs, roadmap, changelog,
help ou blog.
- Roadmap: uma nota por item, com status (planejado, em andamento ou
  concluído) e category.
- Changelog: uma nota por versão, com date e tags.

## Escrevendo

- Links entre notas no formato Markdown padrão, com caminho relativo:
  [texto](../Pasta/Nota.md). Links para títulos: Nota.md#titulo.
- Imagens ficam na pasta de anexos (Anexos); use ![descrição](../Anexos/foto.jpg).
- Vídeo do YouTube: ![título](https://www.youtube.com/watch?v=...).
- Funcionam callouts do Obsidian (> [!tip] Título), fórmulas ($x^2$), notas
  de rodapé e tarefas. %%Comentários%% nunca vão para o site.
- Nada de travessões (— ou –) no texto.

## O que não fazer

- Não apague nem renomeie pastas sem pedirem.
- Não mexa na pasta .obsidian nem nestes arquivos (AGENTS.md, CLAUDE.md,
  GEMINI.md).
- Não rode o Hugo: a prévia do Studio atualiza sozinha. Para publicar, a
  pessoa usa o botão Publicar do Studio.
`

const agentGuideEN = `# This is a Blup site

This folder is a site. Blup turns the notes into pages with Hugo, and Blup
Studio shows a live preview. Always write in English.

## How notes become the site

- Each .md note becomes a page; each folder becomes a section.
- A folder's _index.md note is the section's page.
- The top _index.md holds the site's settings (site_name, site_description,
  logo, favicon, menu, social, button_text, button_link). Only change it
  when asked.

## Properties (at the top of the note, in YAML)

- title, description: title and summary.
- date: a date like 2026-10-29.
- tags: [One, Another]; authors: [Name].
- image: "[[cover.jpg]]" (cover).
- draft: true (preview only); hidden: true (out of lists and Google).
- password: yes (the site's general password) or its own password.
- weight: order in menus (lower first).

## Bento theme sections

Each section's _index.md has the property blup: docs, roadmap, changelog,
help or blog.
- Roadmap: one note per item, with status (planned, in progress or done)
  and category.
- Changelog: one note per version, with date and tags.

## Writing

- Links between notes in standard Markdown, with a relative path:
  [text](../Folder/Note.md). Links to headings: Note.md#heading.
- Images live in the attachments folder (Attachments); use
  ![description](../Attachments/photo.jpg).
- YouTube video: ![title](https://www.youtube.com/watch?v=...).
- Obsidian callouts (> [!tip] Title), math ($x^2$), footnotes and tasks
  work. %%Comments%% never reach the site.
- No em or en dashes (— or –) in the text.

## Don't

- Don't delete or rename folders unless asked.
- Don't touch the .obsidian folder or these files (AGENTS.md, CLAUDE.md,
  GEMINI.md).
- Don't run Hugo: the Studio's preview updates by itself. To publish, the
  person uses the Studio's Publish button.
`

const agentGuideES = `# Este es un sitio de Blup

Esta carpeta es un sitio. Blup convierte las notas en páginas con Hugo, y
Blup Studio muestra una vista previa en vivo. Escribe siempre en español.

## Cómo las notas se vuelven el sitio

- Cada nota .md se vuelve una página; cada carpeta se vuelve una sección.
- La nota _index.md de una carpeta es la página de la sección.
- La _index.md de la raíz guarda la configuración del sitio (site_name,
  site_description, logo, favicon, menu, social, button_text, button_link).
  Solo cámbiala si te lo piden.

## Propiedades (al inicio de la nota, en YAML)

- title, description: título y resumen.
- date: una fecha como 2026-10-29.
- tags: [Una, Otra]; authors: [Nombre].
- image: "[[portada.jpg]]" (portada).
- draft: true (solo en la vista previa); hidden: true (fuera de las listas
  y de Google).
- password: sí (la contraseña general del sitio) o una contraseña propia.
- weight: orden en los menús (menor primero).

## Secciones del tema Bento

La _index.md de cada sección tiene la propiedad blup: docs, roadmap,
changelog, help o blog.
- Roadmap: una nota por elemento, con status (planeado, en curso o listo) y
  category.
- Changelog: una nota por versión, con date y tags.

## Escritura

- Enlaces entre notas en Markdown estándar, con ruta relativa:
  [texto](../Carpeta/Nota.md). Enlaces a títulos: Nota.md#titulo.
- Las imágenes van en la carpeta de adjuntos (Adjuntos); usa
  ![descripción](../Adjuntos/foto.jpg).
- Video de YouTube: ![título](https://www.youtube.com/watch?v=...).
- Funcionan los callouts de Obsidian (> [!tip] Título), las fórmulas
  ($x^2$), las notas al pie y las tareas. Los %%comentarios%% nunca llegan
  al sitio.
- Nada de rayas ni guiones largos (— o –) en el texto.

## No hagas

- No borres ni renombres carpetas a menos que te lo pidan.
- No toques la carpeta .obsidian ni estos archivos (AGENTS.md, CLAUDE.md,
  GEMINI.md).
- No ejecutes Hugo: la vista previa del Studio se actualiza sola. Para
  publicar, la persona usa el botón Publicar del Studio.
`
