import type { SiteInfo } from "../bindings/github.com/gabriellsalesx/blup/studio/models";

// The Blup that stands for a site or vault: one per kind, as on the home
// screen and in the switcher.
export const blupOf = (s?: Pick<SiteInfo, "model" | "kind">) =>
  s?.kind === "notes" ? "turma-7" : ({ bento: "turma-4", nori: "turma-2" } as Record<string, string>)[s?.model ?? ""] ?? "turma-3";

// The home screen's section to show whole when it opens ("See all" in the
// switcher).
export const homeView = $state({ expand: "" as "" | "sites" | "vaults" });
