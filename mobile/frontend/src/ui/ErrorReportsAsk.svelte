<script lang="ts">
  import Bug from "@lucide/svelte/icons/bug";
  import Sheet from "./Sheet.svelte";
  import { t } from "../desk/i18n";

  // A pergunta, uma vez, numa gaveta: o app pode mandar relatórios de erro
  // para o Blup? O que vai, o que nunca vai, e que muda nos Ajustes.
  // Fechar a gaveta é "agora não": pergunta de novo na próxima abertura.
  let { onAnswer }: { onAnswer: (yes: boolean | null) => void } = $props();
</script>

<Sheet title={t.reportsAsk.title} onclose={() => onAnswer(null)}>
  <div class="body">
    <p class="lead"><span class="ico"><Bug size={22} /></span>{t.reportsAsk.lead}</p>
    <ul>
      <li>{t.reportsAsk.only}</li>
      <li>{t.reportsAsk.never}</li>
      <li>{t.reportsAsk.change}</li>
    </ul>
    <button class="yes" onclick={() => onAnswer(true)}>{t.reportsAsk.send}</button>
    <button class="no" onclick={() => onAnswer(false)}>{t.reportsAsk.dontSend}</button>
  </div>
</Sheet>

<style>
  .body { display: grid; gap: .9rem; padding-block-end: .5rem; }
  .lead { display: flex; gap: .75rem; align-items: flex-start; margin: 0; font-size: 1rem; line-height: 1.45; }
  .ico { color: var(--accent); flex-shrink: 0; }
  ul { margin: 0; padding-inline-start: 1.2rem; display: grid; gap: .4rem; font-size: .92rem; color: var(--muted); }
  button { min-height: 3.1rem; border-radius: .8rem; font-size: 1rem; font-weight: 600; }
  .yes { background: var(--accent); color: var(--on-accent); }
  .no { background: transparent; color: var(--text); border: 1px solid var(--line); }
</style>
