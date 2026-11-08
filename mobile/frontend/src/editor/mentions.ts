import { autocompletion, type Completion, type CompletionContext } from "@codemirror/autocomplete";
import { EditorSelection } from "@codemirror/state";
import type { EditorView } from "@codemirror/view";
import { t } from "../desk/i18n";

// The Studio's @ menu: notes, which become a link that reads "@Name", and
// dates. It opens at the start of a line or after a space, so that an e-mail
// doesn't open it, and never inside code.
export type MentionNote = { name: string; path: string };

const startsWord = (ctx: CompletionContext, at: number) => at === ctx.state.doc.lineAt(at).from || /\s/.test(ctx.state.sliceDoc(at - 1, at));
// The link goes by the path relative to the note, as in the Studio.
function noteHref(self: string, target: string) {
  const from = self.split("/").slice(0, -1);
  const to = target.split("/");
  let i = 0;
  while (i < from.length && i < to.length - 1 && from[i] === to[i]) i++;
  const parts = [...from.slice(i).map(() => ".."), ...to.slice(i)];
  return parts.map((p) => p.replace(/%/g, "%25").replace(/ /g, "%20").replace(/\(/g, "%28").replace(/\)/g, "%29")).join("/");
}
const mdLink = (text: string, self: string, path: string) => `[${text.replace(/[[\]]/g, "\\$&")}](${noteHref(self, path)})`;

function dates(): Completion[] {
  const day = (n: number) => { const d = new Date(); d.setDate(d.getDate() + n); return d; };
  // The words and dates go by the app's language, read on each opening.
  const e = t.editor, locale = t.locale;
  const list: [string, Date][] = [[e.today, day(0)], [e.tomorrow, day(1)], [e.yesterday, day(-1)]];
  for (let n = 2; n <= 7; n++) list.push([day(n).toLocaleDateString(locale, { weekday: "long" }), day(n)]);
  return list.map(([label, d], i) => {
    const text = d.toLocaleDateString(locale, { day: "2-digit", month: "2-digit", year: "numeric" });
    return {
      label, detail: d.toLocaleDateString(locale, { day: "numeric", month: "long" }), type: "mention", section: { name: e.secDates, rank: 1 }, boost: -i,
      apply: (v: EditorView, _c: Completion, from: number, to: number) => v.dispatch({ changes: { from: from - 1, to, insert: text }, selection: EditorSelection.cursor(from - 1 + text.length) }),
    };
  });
}

export function mentions(notes: () => MentionNote[], self: () => string) {
  const source = (ctx: CompletionContext) => {
    const m = ctx.matchBefore(/@[^\s@\[\]()]*/u);
    if (!m || !startsWord(ctx, m.from)) return null;
    const options: Completion[] = notes().filter((n) => n.name !== "_index" && n.path !== self()).map((n) => ({
      label: n.name, detail: n.path.includes("/") ? n.path.slice(0, n.path.lastIndexOf("/")) : "", type: "mention", section: { name: t.editor.secNotes, rank: 0 },
      apply: (v: EditorView, _c: Completion, from: number, to: number) => {
        // The space leaves the cursor off the link, which then reads "@Name".
        const md = mdLink("@" + n.name, self(), n.path) + " ";
        v.dispatch({ changes: { from: from - 1, to, insert: md }, selection: EditorSelection.cursor(from - 1 + md.length) });
      },
    }));
    return { from: m.from + 1, options: [...options, ...dates()], validFor: /^[^\s@\[\]()]*$/u };
  };
  return autocompletion({ override: [source], icons: false, activateOnTypingDelay: 100 });
}
