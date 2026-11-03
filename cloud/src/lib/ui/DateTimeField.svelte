<script lang="ts">
	import Calendar from "@lucide/svelte/icons/calendar";
	import ChevronLeft from "@lucide/svelte/icons/chevron-left";
	import ChevronRight from "@lucide/svelte/icons/chevron-right";
	import ChevronUp from "@lucide/svelte/icons/chevron-up";
	import ChevronDown from "@lucide/svelte/icons/chevron-down";
	import Globe from "@lucide/svelte/icons/globe";
	import { t, intlLocale } from "#lib/i18n.svelte.ts";

	// A date and time, in the panel's look: the same tall box as FloatField,
	// with its label on the border, and under it a month to pick the day and
	// the hour and minute. The value is an instant (ms); it is shown and
	// picked in the browser's time zone, which is named under the time.
	// `min` and `max` bound it.
	let { label, value = $bindable(), min, max, surface = "bg-neutral-50 dark:bg-neutral-950", disabled = false }: {
		label: string;
		value: number;
		min?: number;
		max?: number;
		surface?: string;
		disabled?: boolean;
	} = $props();
	const id = $props.id();
	let open = $state(false);

	const locale = $derived(intlLocale());
	const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
	const shown = $derived(new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short" }).format(value));
	const offset = $derived(new Intl.DateTimeFormat(locale, { timeZoneName: "longOffset" }).formatToParts(value).find((p) => p.type === "timeZoneName")?.value ?? "");
	const hour12 = $derived(new Intl.DateTimeFormat(locale, { hour: "numeric" }).resolvedOptions().hourCycle?.startsWith("h1") ?? false);
	// The first day of the week where the person is (Monday unless the
	// locale says otherwise).
	const firstDay = $derived.by(() => {
		const info = new Intl.Locale(locale) as Intl.Locale & { getWeekInfo?: () => { firstDay: number }; weekInfo?: { firstDay: number } };
		return (info.getWeekInfo?.() ?? info.weekInfo)?.firstDay ?? 1;
	});

	// The month shown: the value's, until another is opened with the arrows.
	let month = $derived.by(() => { const d = new Date(value); return new Date(d.getFullYear(), d.getMonth(), 1).getTime(); });
	const monthName = $derived(new Intl.DateTimeFormat(locale, { month: "long", year: "numeric" }).format(month));
	const weekdays = $derived.by(() => {
		const format = new Intl.DateTimeFormat(locale, { weekday: "short" });
		// 2024-01-01 was a Monday (ISO day 1).
		return Array.from({ length: 7 }, (_, i) => format.format(new Date(2024, 0, ((firstDay - 1 + i) % 7) + 1)));
	});
	const days = $derived.by(() => {
		const start = new Date(month), lead = (start.getDay() - (firstDay % 7) + 7) % 7;
		return Array.from({ length: 42 }, (_, i) => new Date(start.getFullYear(), start.getMonth(), 1 - lead + i));
	});
	const dayLabel = $derived(new Intl.DateTimeFormat(locale, { dateStyle: "full" }));
	const sameDay = (a: Date, b: Date) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
	const dayEnd = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate() + 1).getTime() - 1;
	const allowed = (d: Date) => (min === undefined || dayEnd(d) >= min) && (max === undefined || d.getTime() <= max);
	const clamp = (ms: number) => Math.min(max ?? ms, Math.max(min ?? ms, ms));

	const current = $derived(new Date(value));
	const hour = $derived(hour12 ? current.getHours() % 12 || 12 : current.getHours());
	const pm = $derived(current.getHours() >= 12);
	const two = (n: number) => String(n).padStart(2, "0");
	const period = $derived(new Intl.DateTimeFormat(locale, { hour: "numeric", hour12: true }).formatToParts(value).find((p) => p.type === "dayPeriod")?.value ?? (pm ? "PM" : "AM"));

	// A new instant from the parts shown (local time).
	const at = (d: Date, hours: number, minutes: number) => clamp(new Date(d.getFullYear(), d.getMonth(), d.getDate(), hours, minutes).getTime());
	const pick = (d: Date) => (value = at(d, current.getHours(), current.getMinutes()));
	function step(part: "hour" | "minute" | "period", by: number) {
		const hours = current.getHours() + (part === "hour" ? by : part === "period" ? (pm ? -12 : 12) : 0);
		value = at(current, hours, current.getMinutes() + (part === "minute" ? by : 0));
	}
	function moveMonth(by: number) { const d = new Date(month); month = new Date(d.getFullYear(), d.getMonth() + by, 1).getTime(); }
	const canMove = (by: number) => { const d = new Date(month), first = new Date(d.getFullYear(), d.getMonth() + by, 1), last = new Date(d.getFullYear(), d.getMonth() + by + 1, 0); return allowed(by < 0 ? last : first); };
	const stepper = "grid size-8 place-items-center rounded-lg text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 disabled:opacity-30 dark:hover:bg-neutral-800 dark:hover:text-white";
</script>

<div class="relative">
	<span class="pointer-events-none absolute -top-2.5 left-2 z-10 px-1 text-[13px] text-neutral-500 {surface}">{label}</span>
	<button
		type="button"
		{disabled}
		popovertarget="{id}-panel"
		aria-haspopup="dialog"
		aria-expanded={open}
		aria-label="{label}: {shown} {offset}"
		style:anchor-name="--date-{id}"
		class="flex min-h-14 w-full items-center gap-3 rounded-md border px-3 text-left text-base transition-colors disabled:opacity-50 {surface} {open ? 'border-blup' : 'border-neutral-300 hover:border-neutral-400 dark:border-neutral-700'}"
	>
		<Calendar class="size-[18px] shrink-0 text-neutral-500" />
		<span class="truncate">{shown}</span><span class="shrink-0 text-sm text-neutral-500">{offset}</span>
	</button>
	<div
		id="{id}-panel"
		popover
		role="dialog"
		aria-label={label}
		style:position-anchor="--date-{id}"
		class="date-panel w-[19rem] rounded-xl border border-neutral-200 bg-white p-3 text-sm shadow-lg dark:border-neutral-800 dark:bg-neutral-900 dark:text-white"
		ontoggle={(event) => (open = event.newState === "open")}
	>
		<div class="flex items-center justify-between">
			<button type="button" class={stepper} aria-label={t("datePrevMonth")} disabled={!canMove(-1)} onclick={() => moveMonth(-1)}><ChevronLeft class="size-4" /></button>
			<p class="font-semibold first-letter:uppercase" aria-live="polite">{monthName}</p>
			<button type="button" class={stepper} aria-label={t("dateNextMonth")} disabled={!canMove(1)} onclick={() => moveMonth(1)}><ChevronRight class="size-4" /></button>
		</div>
		<div class="mt-2 grid grid-cols-7 text-center text-xs text-neutral-500" aria-hidden="true">{#each weekdays as day, i (i)}<span class="py-1.5">{day}</span>{/each}</div>
		<div class="grid grid-cols-7 gap-0.5">
			{#each days as day (day.getTime())}
				{@const inMonth = day.getMonth() === new Date(month).getMonth()}
				{@const selected = sameDay(day, current)}
				<button
					type="button"
					aria-label={dayLabel.format(day)}
					aria-pressed={selected}
					disabled={!allowed(day)}
					class="grid h-9 place-items-center rounded-lg tabular-nums disabled:opacity-30 {selected ? 'bg-blup font-semibold text-white' : inMonth ? 'hover:bg-neutral-100 dark:hover:bg-neutral-800' : 'text-neutral-400 hover:bg-neutral-100 dark:text-neutral-600 dark:hover:bg-neutral-800'} {!selected && sameDay(day, new Date()) ? 'ring-1 ring-neutral-300 dark:ring-neutral-700' : ''}"
					onclick={() => pick(day)}
				>{day.getDate()}</button>
			{/each}
		</div>
		<div class="mt-3 flex items-center justify-center gap-2 border-t border-neutral-200 pt-3 dark:border-neutral-800">
			<div class="grid justify-items-center"><button type="button" class={stepper} aria-label={t("dateHourUp")} onclick={() => step("hour", 1)}><ChevronUp class="size-4" /></button><span class="w-8 text-center text-base font-semibold tabular-nums">{two(hour)}</span><button type="button" class={stepper} aria-label={t("dateHourDown")} onclick={() => step("hour", -1)}><ChevronDown class="size-4" /></button></div>
			<span class="pb-0.5 text-base font-semibold" aria-hidden="true">:</span>
			<div class="grid justify-items-center"><button type="button" class={stepper} aria-label={t("dateMinuteUp")} onclick={() => step("minute", 1)}><ChevronUp class="size-4" /></button><span class="w-8 text-center text-base font-semibold tabular-nums">{two(current.getMinutes())}</span><button type="button" class={stepper} aria-label={t("dateMinuteDown")} onclick={() => step("minute", -1)}><ChevronDown class="size-4" /></button></div>
			{#if hour12}<div class="grid justify-items-center"><button type="button" class={stepper} aria-label={t("datePeriod")} onclick={() => step("period", 1)}><ChevronUp class="size-4" /></button><span class="w-10 text-center text-base font-semibold">{period}</span><button type="button" class={stepper} aria-label={t("datePeriod")} onclick={() => step("period", 1)}><ChevronDown class="size-4" /></button></div>{/if}
		</div>
		<p class="mt-3 flex items-center gap-2 rounded-lg bg-neutral-50 px-3 py-2 text-xs text-neutral-500 dark:bg-neutral-800/60"><Globe class="size-3.5 shrink-0" /><span class="font-semibold text-neutral-700 dark:text-neutral-200">{offset}</span><span class="truncate">{zone.replaceAll("_", " ")}</span></p>
	</div>
</div>

<style>
	.date-panel {
		position: fixed;
		inset: auto;
		top: anchor(bottom);
		left: anchor(left);
		margin-top: 0.375rem;
		position-try-fallbacks: flip-block, flip-inline;
	}
</style>
