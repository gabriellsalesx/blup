<script lang="ts" module>
  import { LucideIcons } from "../../bindings/github.com/gabriellsalesx/blup/studio/studio";

  // The theme's Lucide icons (name -> SVG paths), loaded once.
  let icons: Promise<Record<string, string>> | null = null;
  const load = () => (icons ??= LucideIcons().then((s) => JSON.parse(s) as Record<string, string>));

  // Words people type in Portuguese or Spanish, to the English of the
  // icons' names (Lucide names them in English).
  const WORDS: Record<string, string[]> = {
    ajuda: ["help"], ayuda: ["help"], duvida: ["help", "question"], pergunta: ["help", "question"], pregunta: ["help", "question"],
    foguete: ["rocket"], cohete: ["rocket"], comeco: ["rocket", "play", "flag"], inicio: ["house", "rocket"], casa: ["house", "home"],
    livro: ["book"], libro: ["book"], guia: ["book", "map", "compass"], documento: ["file"], arquivo: ["file", "archive"], archivo: ["file", "archive"],
    pasta: ["folder"], carpeta: ["folder"], usuario: ["user"], pessoa: ["user"], persona: ["user"], equipe: ["users"], equipo: ["users"],
    configuracao: ["settings", "sliders", "cog"], config: ["settings", "sliders", "cog"], ajuste: ["settings", "sliders"], ferramenta: ["wrench", "hammer"], herramienta: ["wrench", "hammer"],
    seguranca: ["shield", "lock"], seguridad: ["shield", "lock"], cadeado: ["lock"], candado: ["lock"], chave: ["key"], llave: ["key"], senha: ["key", "lock"],
    dinheiro: ["banknote", "coins", "wallet", "dollar"], dinero: ["banknote", "coins", "wallet", "dollar"], pagamento: ["credit-card", "wallet"], pago: ["credit-card", "wallet"], cartao: ["credit-card"], tarjeta: ["credit-card"],
    loja: ["store", "shopping"], tienda: ["store", "shopping"], carrinho: ["shopping-cart"], carrito: ["shopping-cart"], compra: ["shopping"], presente: ["gift"], regalo: ["gift"],
    email: ["mail"], correo: ["mail"], mensagem: ["message", "mail"], mensaje: ["message", "mail"], conversa: ["message"], chat: ["message"], telefone: ["phone"], telefono: ["phone"],
    calendario: ["calendar"], data: ["calendar"], fecha: ["calendar"], relogio: ["clock", "watch"], reloj: ["clock", "watch"], tempo: ["clock", "timer"], tiempo: ["clock", "timer"],
    imagem: ["image"], imagen: ["image"], foto: ["image", "camera"], camera: ["camera"], camara: ["camera"], video: ["video", "film"], musica: ["music"], audio: ["audio", "headphones"],
    grafico: ["chart"], dados: ["database", "chart"], datos: ["database", "chart"], codigo: ["code"], nuvem: ["cloud"], nube: ["cloud"], mundo: ["globe", "earth"], idioma: ["languages", "globe"],
    coracao: ["heart"], corazon: ["heart"], estrela: ["star"], estrella: ["star"], bandeira: ["flag"], bandera: ["flag"], mapa: ["map"], lugar: ["map-pin"], busca: ["search"], buscar: ["search"],
    lista: ["list"], tarefa: ["list-todo", "check"], tarea: ["list-todo", "check"], sino: ["bell"], aviso: ["bell", "alert", "triangle-alert"], alerta: ["alert", "triangle-alert"], raio: ["zap"], rayo: ["zap"],
    compartilhar: ["share"], compartir: ["share"], conta: ["user", "circle-user"], cuenta: ["user", "circle-user"], empresa: ["building", "briefcase"], trabalho: ["briefcase"], trabajo: ["briefcase"],
    escola: ["school", "graduation"], escuela: ["school", "graduation"], curso: ["graduation", "book"], aula: ["presentation", "graduation"], ideia: ["lightbulb"], idea: ["lightbulb"],
  };
  const fold = (s: string) => s.toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "").trim();
</script>

<script lang="ts">
  import { t } from "../i18n";
  import X from "@lucide/svelte/icons/x";

  // A help collection's icon: any of the theme's Lucide icons, picked from
  // a grid with a search (English names, or common words in Portuguese and
  // Spanish). "" leaves it to the theme, which takes turns.
  let { value = "", onPick }: { value?: string; onPick: (name: string) => void } = $props();
  const ip = t.iconPicker;
  const MAX = 240; // icons drawn at once; the search narrows the rest

  let all = $state<Record<string, string>>({});
  let open = $state(false);
  let query = $state("");
  load().then((m) => (all = m)).catch(() => {});

  const found = $derived.by(() => {
    const names = Object.keys(all);
    const q = fold(query);
    if (!q) return names;
    const terms = [q.replace(/\s+/g, "-"), ...Object.entries(WORDS).filter(([w]) => w.startsWith(q) || q.startsWith(w)).flatMap(([, en]) => en)];
    return names.filter((n) => terms.some((term) => n.includes(term)));
  });

  function pick(name: string) {
    onPick(name);
    open = false;
    query = "";
  }
  const svg = (name: string) => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${all[name] ?? ""}</svg>`;
  const focus = (el: HTMLInputElement) => el.focus();
</script>

<svelte:window onkeydown={(e) => open && e.key === "Escape" && (open = false)} />

<div class="ip">
  <button type="button" class="ip-current" aria-expanded={open} onclick={() => (open = !open)}>
    {#if value && all[value]}<span class="ip-ico">{@html svg(value)}</span><span>{value}</span>
    {:else}<span class="muted">{ip.auto}</span>{/if}
  </button>
  {#if open}
    <!-- A click outside closes it (Esc too, on the window). -->
    <div class="ip-scrim" role="presentation" onclick={() => (open = false)}></div>
    <div class="ip-pop" role="dialog" aria-label={ip.title}>
      <div class="ip-head">
        <input type="text" bind:value={query} placeholder={ip.search} aria-label={ip.search} {@attach focus} />
        <button type="button" class="icon-btn" title={ip.close} onclick={() => (open = false)}><X size={16} /></button>
      </div>
      <button type="button" class="ip-auto" class:on={!value} onclick={() => pick("")}>{ip.auto}</button>
      <div class="ip-grid">
        {#each found.slice(0, MAX) as name (name)}
          <button type="button" class="ip-cell" class:on={name === value} title={name} aria-label={name} onclick={() => pick(name)}>{@html svg(name)}</button>
        {/each}
      </div>
      {#if !found.length}<p class="muted ip-note">{ip.none}</p>
      {:else if found.length > MAX}<p class="muted ip-note">{ip.more.replace("{n}", String(found.length - MAX))}</p>{/if}
    </div>
  {/if}
</div>

<style>
  .ip { position: relative; }
  .ip-current { display: flex; align-items: center; gap: .5rem; width: 100%; padding: .5rem .7rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--bg); text-align: start; font-size: .9rem; }
  .ip-current:hover { border-color: var(--muted); }
  .ip-ico { display: grid; width: 1.15rem; height: 1.15rem; }
  .ip-ico :global(svg), .ip-cell :global(svg) { width: 100%; height: 100%; }
  .ip-scrim { position: fixed; inset: 0; z-index: 20; }
  .ip-pop { position: absolute; z-index: 21; inset-inline: 0; inset-block-start: calc(100% + .35rem); display: grid; gap: .5rem; padding: .6rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); box-shadow: 0 12px 32px rgb(0 0 0 / .18); }
  .ip-head { display: flex; gap: .4rem; align-items: center; }
  .ip-head input { flex: 1; min-width: 0; }
  .ip-auto { justify-self: start; padding: .25rem .7rem; border: 1px solid var(--line); border-radius: 999px; font-size: .8rem; font-weight: 600; }
  .ip-auto.on, .ip-cell.on { border-color: var(--blue); color: var(--blue); }
  .ip-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(2.4rem, 1fr)); gap: .3rem; max-height: 16rem; overflow: auto; }
  .ip-cell { display: grid; place-items: center; aspect-ratio: 1; padding: .5rem; border: 1px solid transparent; border-radius: .45rem; color: var(--text); }
  .ip-cell:hover { background: var(--soft); }
  .ip-note { font-size: .8rem; }
</style>
