import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { Language } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
// The registered events (eventdata.d.ts types each Events.On callback by
// the event's name) and their data creation code.
import "../bindings/events";
import { setLanguage } from "./i18n";
import { loadTheme } from "./theme.svelte";
import { installUndo } from "./undo";
import { primeSound } from "./audio";

installUndo();

// The language first: the spell check's dictionary follows it. While the
// Studio asks for it, the media engine starts (audio.ts): its pause goes
// into the opening, not into a ▶ later.
const lang = Language();
primeSound();
lang
  .then(setLanguage)
  .catch(() => {})
  .then(loadTheme)
  .catch(() => {})
  .finally(() => mount(App, { target: document.getElementById("app")! }));
