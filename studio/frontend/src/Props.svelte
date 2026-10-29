<script lang="ts">
  import { PickCover, LinkedImage } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
  import Switch from "./Switch.svelte";
  import X from "@lucide/svelte/icons/x";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import { t } from "./i18n";

  // The note's properties as fields. onEdit(key, value) reports a change.
  let { props, kind, isIndex, onEdit }: { props: Record<string, any>; kind: string; isIndex: boolean; onEdit: (key: string, value: any) => void } = $props();

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
  let pwMode = $state(pw0 === "" ? "none" : ["sim", "yes", "true"].includes(pw0.toLowerCase()) ? "general" : "own");
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
  function password() {
    onEdit("password", pwMode === "none" ? null : pwMode === "general" ? (t.lang === "pt" ? "sim" : "yes") : pwOwn);
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
  const statuses = t.lang === "pt" ? ["planejado", "em andamento", "concluído"] : ["planned", "in progress", "done"];
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

        <div class="wide switches">
          <Switch label={p.draft} help={p.draftHelp} bind:checked={draft} onchange={(v) => onEdit("draft", v)} />
          <Switch label={p.hidden} help={p.hiddenHelp} bind:checked={hidden} onchange={(v) => onEdit("hidden", v)} />
        </div>

        <label class="field"><span>{p.password}</span>
          <select bind:value={pwMode} onchange={password}>
            <option value="none">{p.pwNone}</option>
            <option value="general">{p.pwGeneral}</option>
            <option value="own">{p.pwOwn}</option>
          </select>
        </label>
        {#if pwMode === "own"}
          <label class="field"><span>{p.pwOwnField}</span><input type="text" bind:value={pwOwn} oninput={password} /></label>
        {/if}
      {/if}

      <button class="more wide" onclick={() => (more = !more)}>{more ? "−" : "+"} {p.more}</button>
      {#if more}
        {#if !isIndex}
          <div class="field wide"><span>{p.authors}</span>
            <div class="chips">
              {#each authors as a, i (a)}<span class="chip">{a}<button title={t.remove} onclick={() => { authors.splice(i, 1); onEdit("authors", [...authors]); }}><X size={12} /></button></span>{/each}
              <input type="text" bind:value={authorInput} onkeydown={(e) => addChip(e, "authors")} placeholder={p.addAuthor} />
            </div>
          </div>
        {/if}
        <label class="field"><span>{p.weight}</span><input type="number" min="0" bind:value={weight} oninput={() => onEdit("weight", Number(weight) || 0)} /><small>{p.weightHelp}</small></label>
      {/if}
    </div>
  {/if}
</section>

<style>
  .props { border-block-end: 1px solid var(--line); background: var(--bg); max-height: 55%; overflow: auto; flex: none; }
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
  .switches { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1.5rem; }
  .more { justify-self: start; font-size: .8rem; font-weight: 600; color: var(--blue); }
</style>
