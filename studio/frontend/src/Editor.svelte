<script lang="ts">
  import { onMount } from "svelte";
  import { EditorView, basicSetup } from "codemirror";
  import { keymap } from "@codemirror/view";
  import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
  import { languages } from "@codemirror/language-data";
  import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
  import { tags } from "@lezer/highlight";

  // Markdown colors from the Studio's tokens, readable in light and dark.
  const markdownStyle = HighlightStyle.define([
    { tag: tags.heading1, fontWeight: "800", fontSize: "1.5em" },
    { tag: tags.heading2, fontWeight: "700", fontSize: "1.3em" },
    { tag: tags.heading3, fontWeight: "700", fontSize: "1.15em" },
    { tag: [tags.heading4, tags.heading5, tags.heading6], fontWeight: "700" },
    { tag: tags.strong, fontWeight: "700" },
    { tag: tags.emphasis, fontStyle: "italic" },
    { tag: tags.strikethrough, textDecoration: "line-through" },
    { tag: [tags.link, tags.url], color: "var(--blue)" },
    { tag: tags.monospace, fontFamily: "ui-monospace, monospace", fontSize: ".9em", backgroundColor: "var(--soft)", borderRadius: "4px" },
    { tag: [tags.processingInstruction, tags.meta, tags.contentSeparator], color: "var(--muted)" },
    { tag: tags.quote, color: "var(--muted)", fontStyle: "italic" },
  ]);

  // A Markdown editor. onChange gets every edit; onSave, Ctrl/Cmd+S.
  let { text, onChange, onSave }: { text: string; onChange: (v: string) => void; onSave: () => void } = $props();
  let host: HTMLDivElement;
  let view: EditorView;

  onMount(() => {
    view = new EditorView({
      parent: host,
      doc: text,
      extensions: [
        basicSetup,
        markdown({ base: markdownLanguage, codeLanguages: languages }),
        syntaxHighlighting(markdownStyle),
        EditorView.lineWrapping,
        keymap.of([{ key: "Mod-s", preventDefault: true, run: () => (onSave(), true) }]),
        EditorView.updateListener.of((u) => { if (u.docChanged) onChange(u.state.doc.toString()); }),
      ],
    });
    view.focus();
    return () => view.destroy();
  });
</script>

<div class="editor" bind:this={host}></div>

<style>
  .editor { height: 100%; overflow: hidden; }
  .editor :global(.cm-editor) { height: 100%; background: var(--panel); font-size: 1rem; }
  .editor :global(.cm-editor.cm-focused) { outline: none; }
  .editor :global(.cm-scroller) { font-family: "Inter", ui-sans-serif, system-ui, sans-serif; line-height: 1.7; padding: 2rem 0; }
  .editor :global(.cm-content) { max-width: 46rem; margin-inline: auto; padding-inline: 2rem; caret-color: var(--text); }
  .editor :global(.cm-gutters) { display: none; }
  .editor :global(.cm-activeLine) { background: transparent; }
  .editor :global(.cm-line) { color: var(--text); }
  /* CodeMirror draws its own cursor and selection: visible in dark too. */
  .editor :global(.cm-cursor), .editor :global(.cm-dropCursor) { border-left: 2px solid var(--text); margin-left: -1px; }
  .editor :global(.cm-selectionBackground), .editor :global(.cm-focused .cm-selectionBackground), .editor :global(.cm-content ::selection) { background: color-mix(in srgb, var(--blue) 30%, transparent) !important; }
  .editor :global(.cm-selectionMatch) { background: color-mix(in srgb, var(--blue) 15%, transparent); }
</style>
