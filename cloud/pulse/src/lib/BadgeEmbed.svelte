<script lang="ts">
 import {t} from "#lib/i18n.svelte.ts";
 import FormSection from "#lib/ui/FormSection.svelte";
 import Segmented from "#lib/ui/Segmented.svelte";
 import CodeBlock from "#lib/ui/CodeBlock.svelte";
 import {publicPageURL} from "./status-pages";
 // The badge another site shows in an iframe: the page's state with the
 // same mark as its public page, linking to it. The preview is the badge
 // itself, on the background of the chosen theme.
 let {slug,published}:{slug:string;published:boolean}=$props();
 type Theme="light"|"dark"|"auto";
 let theme=$state<Theme>("auto");
 const themes=$derived([{value:"auto",label:t("system")},{value:"light",label:t("light")},{value:"dark",label:t("dark")}] as const);
 const src=$derived(`${publicPageURL(slug)}badge?theme=${theme}`);
 const code=$derived(`<iframe src="${src}" width="250" height="30" title="Status" loading="lazy" style="border:0;color-scheme:normal"></iframe>`);
 const surface={auto:"bg-white dark:bg-neutral-950",light:"bg-white",dark:"bg-neutral-950"} as const satisfies Record<Theme,string>;
</script>

<FormSection title={t("pulseBadge")} help={t("pulseBadgeHelp")}>
 <div class="grid grid-cols-[minmax(0,1fr)] gap-4">
  <Segmented label={t("pulseBadgeTheme")} options={themes} bind:value={theme}/>
  {#if published}
   <div class="rounded-lg border border-neutral-200 px-4 py-3 dark:border-neutral-800 {surface[theme]}"><iframe {src} width="250" height="30" title={t("pulseBadge")} class="block border-0 [color-scheme:normal]"></iframe></div>
  {:else}<p class="text-sm text-amber-700 dark:text-amber-300">{t("pulseBadgeUnpublished")}</p>{/if}
  <CodeBlock {code}/>
 </div>
</FormSection>
