<script lang="ts">
  import { onDestroy, onMount, untrack, type Snippet } from "svelte";
  import { app } from "../lib/app.svelte";
  import Sheet from "../ui/Sheet.svelte";

  // The app's sheet for what the Studio shows in a dialog or a floating menu:
  // the phone's back button and a slide down close it, as any other sheet.
  // The next step of a flow takes its place on the screen, one sheet at a
  // time, and going back brings this one again: each is a step of the history.
  let { title, onclose, children, top, tall = false }: { title: string; onclose: () => void; children: Snippet; top?: Snippet; tall?: boolean } = $props();
  let level = $state(0);

  onMount(() => {
    level = untrack(() => (app.place.over ?? 0)) + 1;
    app.go({ ...untrack(() => app.place), over: level });
  });
  $effect(() => { if (level && (app.place.over ?? 0) < level) onclose(); });
  // Closed by what was picked in it: the step in the history goes too.
  onDestroy(() => { if ((app.place.over ?? 0) >= level && level) app.back(); });
</script>

<Sheet {title} {tall} {top} behind={!!level && (app.place.over ?? 0) > level} onclose={() => app.back()}>{@render children()}</Sheet>
