// The callout types the Blup theme draws (theme/layouts/_partials/mochi/
// icons.html, i18n and mochi.css): name, color and icon, for the editor's
// menu and colors. Keep in step with the theme.
export type CalloutType = { type: string; pt: string; en: string; color: string; icon: string };

export const CALLOUTS: CalloutType[] = [
  {
    "type": "note",
    "pt": "Nota",
    "en": "Note",
    "color": "8 109 221",
    "icon": "<path d=\"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z\"/><path d=\"m15 5 4 4\"/>"
  },
  {
    "type": "abstract",
    "pt": "Resumo",
    "en": "Summary",
    "color": "0 191 188",
    "icon": "<rect width=\"8\" height=\"4\" x=\"8\" y=\"2\" rx=\"1\" ry=\"1\"/><path d=\"M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2\"/><path d=\"M12 11h4\"/><path d=\"M12 16h4\"/><path d=\"M8 11h.01\"/><path d=\"M8 16h.01\"/>"
  },
  {
    "type": "info",
    "pt": "Informação",
    "en": "Info",
    "color": "8 109 221",
    "icon": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"M12 16v-4\"/><path d=\"M12 8h.01\"/>"
  },
  {
    "type": "todo",
    "pt": "A fazer",
    "en": "To do",
    "color": "8 109 221",
    "icon": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"m16 9-5.5 5.5L8 12\"/>"
  },
  {
    "type": "tip",
    "pt": "Dica",
    "en": "Tip",
    "color": "0 191 188",
    "icon": "<path d=\"M12 3q1 4 4 6.5t3 5.5a1 1 0 0 1-14 0 5 5 0 0 1 1-3 1 1 0 0 0 5 0c0-2-1.5-3-1.5-5q0-2 2.5-4\"/>"
  },
  {
    "type": "success",
    "pt": "Concluído",
    "en": "Done",
    "color": "8 185 78",
    "icon": "<path d=\"M20 6 9 17l-5-5\"/>"
  },
  {
    "type": "question",
    "pt": "Pergunta",
    "en": "Question",
    "color": "236 117 0",
    "icon": "<circle cx=\"12\" cy=\"12\" r=\"10\"/><path d=\"M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3\"/><path d=\"M12 17h.01\"/>"
  },
  {
    "type": "warning",
    "pt": "Atenção",
    "en": "Warning",
    "color": "236 117 0",
    "icon": "<path d=\"m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3\"/><path d=\"M12 9v4\"/><path d=\"M12 17h.01\"/>"
  },
  {
    "type": "failure",
    "pt": "Falha",
    "en": "Failure",
    "color": "233 49 71",
    "icon": "<path d=\"M18 6 6 18\"/><path d=\"m6 6 12 12\"/>"
  },
  {
    "type": "danger",
    "pt": "Perigo",
    "en": "Danger",
    "color": "233 49 71",
    "icon": "<path d=\"M15.914 4a1.5 1.5 0 00-2.474-1.561l-9 9A1.5 1.5 0 005.5 14h4.002a.5.5 0 01.471.666L8.086 20a1.5 1.5 0 002.475 1.56l9-9A1.5 1.5 0 0018.5 10h-3.997a.5.5 0 01-.472-.667z\"/>"
  },
  {
    "type": "bug",
    "pt": "Bug",
    "en": "Bug",
    "color": "233 49 71",
    "icon": "<path d=\"M12 20v-9\"/><path d=\"M14 7a4 4 0 0 1 4 4v3a6 6 0 0 1-12 0v-3a4 4 0 0 1 4-4z\"/><path d=\"M14.12 3.88 16 2\"/><path d=\"M21 21a4 4 0 0 0-3.81-4\"/><path d=\"M21 5a4 4 0 0 1-3.55 3.97\"/><path d=\"M22 13h-4\"/><path d=\"M3 21a4 4 0 0 1 3.81-4\"/><path d=\"M3 5a4 4 0 0 0 3.55 3.97\"/><path d=\"M6 13H2\"/><path d=\"m8 2 1.88 1.88\"/><path d=\"M9 7.13V6a3 3 0 1 1 6 0v1.13\"/>"
  },
  {
    "type": "example",
    "pt": "Exemplo",
    "en": "Example",
    "color": "120 82 238",
    "icon": "<path d=\"M3 5h.01\"/><path d=\"M3 12h.01\"/><path d=\"M3 19h.01\"/><path d=\"M8 5h13\"/><path d=\"M8 12h13\"/><path d=\"M8 19h13\"/>"
  },
  {
    "type": "quote",
    "pt": "Citação",
    "en": "Quote",
    "color": "158 158 158",
    "icon": "<path d=\"M16 3a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2 1 1 0 0 1 1 1v1a2 2 0 0 1-2 2 1 1 0 0 0-1 1v2a1 1 0 0 0 1 1 6 6 0 0 0 6-6V5a2 2 0 0 0-2-2z\"/><path d=\"M5 3a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2 1 1 0 0 1 1 1v1a2 2 0 0 1-2 2 1 1 0 0 0-1 1v2a1 1 0 0 0 1 1 6 6 0 0 0 6-6V5a2 2 0 0 0-2-2z\"/>"
  }
];

// Obsidian's other names for the same types.
export const CALLOUT_ALIASES: Record<string, string> = {
  summary: "abstract", tldr: "abstract", hint: "tip", important: "tip", check: "success", done: "success",
  help: "question", faq: "question", caution: "warning", attention: "warning", fail: "failure", missing: "failure",
  error: "danger", cite: "quote",
};

export const calloutIcon = (c: CalloutType, size = 16) =>
  `<svg viewBox="0 0 24 24" width="${size}" height="${size}" fill="none" stroke="rgb(${c.color})" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${c.icon}</svg>`;
