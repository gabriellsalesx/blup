import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { Language } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { setLanguage } from "./i18n";
import { loadTheme } from "./theme.svelte";
import { installUndo } from "./undo";

installUndo();

// The language first: the spell check's dictionary follows it.
Language()
  .then(setLanguage)
  .catch(() => {})
  .then(loadTheme)
  .catch(() => {})
  .finally(() => mount(App, { target: document.getElementById("app")! }));
