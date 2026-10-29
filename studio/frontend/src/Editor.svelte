<script lang="ts">
  import { onMount } from "svelte";
  import { EditorView, basicSetup } from "codemirror";
  import { keymap } from "@codemirror/view";
  import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
  import { languages } from "@codemirror/language-data";

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
</style>
