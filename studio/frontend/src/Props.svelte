<script lang="ts">
  import { AuthorNames, PickCover, LinkedImage, AIConnected, LocalVoice, Narrate, NarrationHash, NarrationEstimate } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import { Events } from "@wailsio/runtime";
  import { toast } from "./toast.svelte";
  import AudioLines from "@lucide/svelte/icons/audio-lines";
  import Switch from "./Switch.svelte";
  import X from "@lucide/svelte/icons/x";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import { t } from "./i18n";
  import NoteLangs from "./NoteLangs.svelte";

  // The note's properties as fields. onEdit(key, value) reports a change;
  // note is the note's path; onSettings opens the Studio's settings (tab).
  let { props, kind, isIndex, note, onEdit, onSettings, onOpenNote }: { props: Record<string, any>; kind: string; isIndex: boolean; note: string; onEdit: (key: string, value: any) => void; onSettings: (tab: string) => void; onOpenNote?: (path: string) => void } = $props();

  const p = t.props;
  let open = $state(false);
  let more = $state(false);
  const list = (v: any): string[] => (Array.isArray(v) ? v.map(String) : v ? [String(v)] : []);

  let title = $state(String(props.title ?? ""));
  let description = $state(String(props.description ?? ""));
  let date = $state(String(props.date ?? "").slice(0, 10));
  let tags = $state(list(props.tags));
  let authors = $state(list(props.authors));
  let image = $state(String(props.image ?? ""));
  let draft = $state(props.draft === true);
  let hidden = $state(props.hidden === true);
  const pw0 = props.password === undefined || props.password === null || props.password === false ? "" : String(props.password);
  let pwMode = $state(pw0 === "" ? "none" : ["sim", "sí", "si", "yes", "true"].includes(pw0.toLowerCase()) ? "general" : "own");
  let pwOwn = $state(pwMode === "own" ? pw0 : "");
  let weight = $state(props.weight ? Number(props.weight) : 0);
  let status = $state(String(props.status ?? ""));
  let category = $state(String(props.category ?? ""));

  let cover = $state("");
  const loadCover = async () => (cover = image ? await LinkedImage(image).catch(() => "") : "");
  loadCover();

  async function pickCover() {
    const link = await PickCover();
    if (!link) return;
    image = link;
    onEdit("image", link);
    loadCover();
  }
  // Narration: a voice (OpenRouter) reads the post; the MP3 goes to the
  // attachments and the "audio" property links it. audio_hash is the
  // fingerprint of the text read, to tell when the post changed since.
  const n = t.narration;
  const canNarrate = !isIndex && (kind === "" || kind === "blog") && !props.blup;
  let audio = $state(String(props.audio ?? ""));
  let audioHash = $state(String(props.audio_hash ?? ""));
  let nowHash = $state("");
  let seconds = $state(Number(props.audio_seconds ?? 0));
  const audioFile = $derived(audio.replace(/^\[\[|\]\]$/g, ""));
  // The narration is heard in the live preview, as on the site.
  const clock = (s: number) => `${Math.floor(s / 60)}:${String(Math.round(s % 60)).padStart(2, "0")}`;
  let aiOn = $state<boolean | null>(null);
  let cost = $state<{ percent: number; left: number; enough: boolean } | null>(null);
  let narrating = $state<{ done: number; total: number } | null>(null);
  const checkNarration = async () => {
    if (!canNarrate) return;
    // Ready with an OpenRouter key or the local voice installed.
    // Through Blup Cloud: what this text would use of the month's AI.
    cost = await NarrationEstimate(note).catch(() => null);
    aiOn = !!cost || (await AIConnected().catch(() => false)) || (await LocalVoice().catch(() => null))?.installed === true;
    nowHash = await NarrationHash(note).catch(() => "");
  };
  checkNarration();
  // The settings changed what can narrate (local voice, OpenRouter key).
  $effect(() => {
    window.addEventListener("blup:ai", checkNarration);
    return () => window.removeEventListener("blup:ai", checkNarration);
  });
  const stale = $derived(!!audio && !!nowHash && !!audioHash && nowHash !== audioHash);
  // Where the narration is: preparing the text (no parts yet), reading part
  // by part, then saving. The bar fills by part; with a single part (or none
  // yet) it is a band going back and forth.
  const stage = $derived(!narrating ? "" : narrating.total === 0 ? n.stagePrep : narrating.done >= narrating.total ? n.stageSave
    : narrating.total > 1 ? n.progress.replace("{done}", String(narrating.done + 1)).replace("{total}", String(narrating.total)) : n.stageVoice);
  const fill = $derived(narrating && narrating.total > 1 ? Math.max(4, (narrating.done / narrating.total) * 100) : 0);
  const offNarrate = Events.On("narrate", (ev: any) => { if (narrating && ev.data?.note === note) narrating = { done: ev.data.done, total: ev.data.total }; });
  $effect(() => () => offNarrate());
  async function narrate() {
    narrating = { done: 0, total: 0 };
    try {
      const r = await Narrate(note);
      audio = r.link; audioHash = r.hash; nowHash = r.hash; seconds = Math.round(r.seconds);
      cost = await NarrationEstimate(note).catch(() => null);
      onEdit("audio", r.link);
      onEdit("audio_hash", r.hash);
      onEdit("audio_seconds", Math.round(r.seconds));
      toast(n.done.replace("{min}", String(Math.max(1, Math.round(r.seconds / 60)))).replace("{mb}", (r.bytes / 1048576).toFixed(1)));
      if (r.used) toast(n.usedOther.replace("{voice}", r.used), { kind: "info", ms: 9000 });
    } catch (e) {
      toast(String(e).includes("no OpenRouter key") ? n.needKey : String(e).replace(/^(Runtime)?Error: /, ""), { kind: "error" });
      aiOn = (await AIConnected().catch(() => false)) || (await LocalVoice().catch(() => null))?.installed === true;
    }
    narrating = null;
  }
  function removeNarration() {
    audio = ""; audioHash = ""; seconds = 0;
    onEdit("audio", null);
    onEdit("audio_hash", null);
    onEdit("audio_seconds", null);
  }

  function password() {
    onEdit("password", pwMode === "none" ? null : pwMode === "general" ? ({ pt: "sim", es: "sí" }[t.lang] ?? "yes") : pwOwn);
  }
  // The switch turns the password on (the site's general one first).
  let pwOn = $state(pwMode !== "none");
  function togglePassword(on: boolean) {
    pwMode = on ? (pwMode === "none" ? "general" : pwMode) : "none";
    password();
  }

  // A chips field: Enter or comma adds, × removes.
  let tagInput = $state("");
  let authorInput = $state("");
  function addChip(e: KeyboardEvent, which: "tags" | "authors") {
    if (e.key !== "Enter" && e.key !== ",") return;
    e.preventDefault();
    const input = which === "tags" ? tagInput : authorInput;
    const v = input.trim();
    if (!v) return;
    if (which === "tags") { if (!tags.includes(v)) tags.push(v); tagInput = ""; onEdit("tags", [...tags]); }
    else { if (!authors.includes(v)) authors.push(v); authorInput = ""; onEdit("authors", [...authors]); }
  }
  // The authors the site knows (a note in the authors folder, or the site's
  // author): suggested while typing, so the name matches and the photo shows.
  let known = $state<string[]>([]);
  AuthorNames().then((n) => (known = n ?? []));
  let authorFocus = $state(false);
  let authorPick = $state(-1);
  const authorOpts = $derived(known.filter((n) => !authors.includes(n) && n.toLowerCase().includes(authorInput.trim().toLowerCase())).slice(0, 8));
  const unknownAuthors = $derived(authors.filter((a) => !known.some((k) => k.toLowerCase() === a.toLowerCase())));
  function addAuthor(name: string) {
    if (!authors.includes(name)) authors.push(name);
    authorInput = "";
    authorPick = -1;
    onEdit("authors", [...authors]);
  }
  function authorKey(e: KeyboardEvent) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      const n = authorOpts.length;
      if (n) authorPick = e.key === "ArrowDown" ? (authorPick + 1) % n : authorPick <= 0 ? n - 1 : authorPick - 1;
      return;
    }
    if (e.key === "Enter" && authorPick >= 0 && authorOpts[authorPick]) { e.preventDefault(); addAuthor(authorOpts[authorPick]); return; }
    if (e.key === "Escape") { authorFocus = false; return; }
    addChip(e, "authors");
  }
  // Order only matters where pages are not listed by date: a folder's
  // _index (the menu) and Bento's docs, help center and roadmap.
  const ordered = $derived(isIndex || ["docs", "help", "roadmap"].includes(kind));

  const statuses = { pt: ["planejado", "em andamento", "concluído"], es: ["planeado", "en curso", "listo"] }[t.lang] ?? ["planned", "in progress", "done"];
</script>

<section class="props">
  <button class="head" onclick={() => (open = !open)} aria-expanded={open}>
    <span class="chev" class:closed={!open}><ChevronDown size={15} /></span>{p.title}
  </button>
  {#if open}
    <div class="grid">
      <label class="field wide"><span>{p.titleField}</span><input type="text" bind:value={title} oninput={() => onEdit("title", title)} /></label>
      <label class="field wide"><span>{p.description}</span><textarea rows="2" bind:value={description} oninput={() => onEdit("description", description)}></textarea><small>{p.descriptionHelp}</small></label>

      {#if !isIndex}
        <label class="field"><span>{p.date}</span><input type="date" bind:value={date} onchange={() => onEdit("date", date)} /></label>
        {#if kind === "roadmap"}
          <label class="field"><span>{p.status}</span>
            <select bind:value={status} onchange={() => onEdit("status", status)}>
              {#each statuses.includes(status) || !status ? statuses : [status, ...statuses] as s (s)}<option value={s}>{s}</option>{/each}
            </select>
          </label>
          <label class="field"><span>{p.category}</span><input type="text" bind:value={category} oninput={() => onEdit("category", category)} /></label>
        {/if}

        <div class="field wide"><span>{p.tags}</span>
          <div class="chips">
            {#each tags as tag, i (tag)}<span class="chip">{tag}<button title={t.remove} onclick={() => { tags.splice(i, 1); onEdit("tags", [...tags]); }}><X size={12} /></button></span>{/each}
            <input type="text" bind:value={tagInput} onkeydown={(e) => addChip(e, "tags")} placeholder={p.addTag} />
          </div>
        </div>

        <div class="field wide"><span>{p.cover}</span>
          <div class="cover">
            <div class="thumb">{#if cover}<img src={cover} alt="" />{:else}<span class="muted">{t.look.none}</span>{/if}</div>
            <button class="btn ghost" onclick={pickCover}>{image ? t.change : t.choose}</button>
            {#if image}<button class="btn ghost" onclick={() => { image = ""; cover = ""; onEdit("image", null); }}>{t.remove}</button>{/if}
          </div>
        </div>

        <!-- On/off choices, one per row, like the settings. -->
        <div class="wide switches">
          <Switch label={p.draft} help={p.draftHelp} bind:checked={draft} onchange={(v) => onEdit("draft", v)} />
          <Switch label={p.hidden} help={p.hiddenHelp} bind:checked={hidden} onchange={(v) => onEdit("hidden", v)} />
          <Switch label={p.password} help={p.pwHelp} bind:checked={pwOn} onchange={togglePassword} />
          {#if pwOn}
            <div class="pw">
              <div class="seg" role="radiogroup" aria-label={p.password}>
                <button role="radio" aria-checked={pwMode === "general"} class:on={pwMode === "general"} onclick={() => { pwMode = "general"; password(); }}>{p.pwGeneral}</button>
                <button role="radio" aria-checked={pwMode === "own"} class:on={pwMode === "own"} onclick={() => { pwMode = "own"; password(); }}>{p.pwOwn}</button>
              </div>
              {#if pwMode === "own"}
                <input type="text" bind:value={pwOwn} oninput={password} placeholder={p.pwOwnField} aria-label={p.pwOwnField} />
              {:else}<small class="muted">{p.pwGeneralHelp}</small>{/if}
            </div>
          {/if}
        </div>
      {/if}

      {#if canNarrate}
        <div class="field wide"><span>{n.title}</span>
          <div class="narration">
            {#if aiOn === false}
              <p class="muted">{n.needKey}</p>
              <button class="btn ghost" onclick={() => onSettings("ai")}>{n.openSettings}</button>
            {:else if aiOn}
              {#if audio}
                <p><b>{audioFile}</b>{seconds ? ` · ${clock(seconds)}` : ""}</p>
                <small class:warn={stale} class:muted={!stale}>{stale ? n.stale : n.fresh} {n.inPreview}</small>
              {/if}
              <div class="cover">
                <button class="btn ghost" disabled={!!narrating || (cost !== null && !cost.enough)} onclick={narrate}><AudioLines size={15} />
                  {#if narrating}{n.working}{:else}{audio ? n.again : n.make}{/if}
                </button>
                {#if audio && !narrating}
                  <button class="btn ghost" onclick={removeNarration}>{t.remove}</button>
                {/if}
              </div>
              {#if narrating}
                <div class="nbar" class:indet={fill === 0} role="progressbar" aria-label={stage} aria-valuemin="0" aria-valuemax="100" aria-valuenow={fill || undefined}>
                  <span style:width={fill ? `${fill}%` : undefined}></span>
                </div>
                <small class="muted stage">{stage}</small>
              {:else if !audio}<small class="muted">{n.help}</small>{/if}
              {#if cost && !narrating}
                <small class:muted={cost.enough} class:warn={!cost.enough}>{(cost.enough ? n.cost : n.costOver).replace("{p}", String(Math.max(1, cost.percent))).replace("{left}", String(cost.left))}</small>
              {/if}
            {/if}
          </div>
        </div>
      {/if}

      {#if onOpenNote && !props.blup}<NoteLangs {note} onOpen={onOpenNote} {onSettings} />{/if}

      <button class="more wide" onclick={() => (more = !more)}>{more ? "−" : "+"} {p.more}</button>
      {#if more}
        {#if !isIndex}
          <div class="field wide"><span>{p.authors}</span>
            <div class="chips suggest">
              {#each authors as a, i (a)}<span class="chip">{a}<button title={t.remove} onclick={() => { authors.splice(i, 1); onEdit("authors", [...authors]); }}><X size={12} /></button></span>{/each}
              <input type="text" bind:value={authorInput} onkeydown={authorKey} oninput={() => (authorPick = -1)} onfocus={() => (authorFocus = true)} onblur={() => setTimeout(() => (authorFocus = false), 150)} placeholder={known.length ? p.pickAuthor : p.addAuthor} />
              {#if authorFocus && authorOpts.length}
                <ul class="opts" role="listbox">
                  {#each authorOpts as n, i (n)}<li><button role="option" aria-selected={i === authorPick} class:on={i === authorPick} onmousedown={(e) => e.preventDefault()} onclick={() => addAuthor(n)}>{n}</button></li>{/each}
                </ul>
              {/if}
            </div>
            {#if unknownAuthors.length}<small>{p.unknownAuthor.replace("{names}", unknownAuthors.join(", "))}</small>{/if}
          </div>
        {/if}
        {#if ordered}
          <label class="field"><span>{p.weight}</span><input type="number" min="0" bind:value={weight} oninput={() => onEdit("weight", Number(weight) || 0)} /><small>{isIndex ? p.weightHelp : p.weightHelpPage}</small></label>
        {/if}
      {/if}
    </div>
  {/if}
</section>

<style>
  .props { border-block-end: 1px solid var(--line); background: var(--bg); max-height: 75%; overflow: auto; flex: none; scrollbar-gutter: stable; }
  .head { display: flex; align-items: center; gap: .4rem; width: 100%; padding: .6rem 1.25rem; font-size: .75rem; font-weight: 700; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); }
  .chev { display: inline-grid; transition: transform .15s; }
  .chev.closed { transform: rotate(-90deg); }
  .grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .9rem 1rem; padding: .25rem 1.25rem 1.25rem; max-width: 52rem; }
  .wide { grid-column: 1 / -1; }
  .chips { display: flex; flex-wrap: wrap; gap: .35rem; align-items: center; padding: .35rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--bg); }
  .chips input { flex: 1; min-width: 8rem; border: 0; padding: .25rem; background: transparent; }
  .chips input:focus-visible { outline: none; }
  .chip { display: inline-flex; align-items: center; gap: .25rem; padding: .15rem .3rem .15rem .55rem; border-radius: 999px; background: var(--soft); font-size: .8rem; font-weight: 500; }
  .chip button { display: inline-grid; color: var(--muted); }
  .cover { display: flex; align-items: center; gap: .75rem; }
  .thumb { width: 6rem; height: 3.5rem; display: grid; place-items: center; border: 1px dashed var(--line); border-radius: .4rem; overflow: hidden; font-size: .7rem; }
  .thumb img { width: 100%; height: 100%; object-fit: cover; }
  .switches { display: grid; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
  .switches > :global(.switch), .switches > .pw { padding: .75rem 1rem; }
  .switches > :global(.switch + .switch) { border-block-start: 1px solid var(--line); }
  .switches > .pw { display: grid; gap: .5rem; padding-block-start: 0; }
  .pw input { max-width: 22rem; }
  .pw small { font-size: .8rem; }
  .seg { justify-self: start; display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--line); border-radius: .55rem; }
  .seg button { padding: .3rem .7rem; border-radius: .4rem; font-size: .82rem; font-weight: 600; color: var(--muted); }
  .seg button.on { background: var(--soft); color: var(--text); }
  /* In the flow (the panel scrolls and would cut a floating list). */
  .opts { flex-basis: 100%; list-style: none; margin: 0; padding: .3rem; border: 1px solid var(--line); border-radius: .5rem; background: var(--panel); box-shadow: 0 10px 30px rgb(0 0 0 / .25); }
  .opts button { width: 100%; padding: .4rem .6rem; border-radius: .35rem; text-align: start; font-size: .875rem; }
  .opts button:hover, .opts button.on { background: var(--soft); }
  .narration { display: grid; gap: .5rem; padding: .75rem 1rem; border: 1px solid var(--line); border-radius: .6rem; background: var(--panel); }
  .narration p { margin: 0; font-size: .85rem; }
  .narration small { font-size: .8rem; }
  /* The narration's progress: a filling bar with a moving sheen, or a band
     going back and forth while the length is unknown. */
  .nbar { position: relative; height: 6px; border-radius: 999px; background: var(--soft); overflow: hidden; }
  .nbar span { position: absolute; inset: 0 auto 0 0; border-radius: inherit; background: var(--blue); transition: width .5s ease; overflow: hidden; }
  .nbar span::after { content: ""; position: absolute; inset: 0; background: linear-gradient(90deg, transparent, rgb(255 255 255 / .45), transparent); transform: translateX(-100%); animation: nsheen 1.4s ease-in-out infinite; }
  .nbar.indet span { width: 35%; animation: nslide 1.3s ease-in-out infinite; }
  .nbar.indet span::after { display: none; }
  @keyframes nslide { 0% { left: -35%; } 100% { left: 100%; } }
  @keyframes nsheen { to { transform: translateX(100%); } }
  @media (prefers-reduced-motion: reduce) { .nbar span, .nbar span::after { animation-duration: 3s; } }
  .stage { font-variant-numeric: tabular-nums; }
  .narration .warn { color: var(--warn, #d97706); }
  .narration .btn { display: inline-flex; align-items: center; gap: .4rem; }
  .more { justify-self: start; font-size: .8rem; font-weight: 600; color: var(--blue); }
</style>
