import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";
import { Language } from "../bindings/github.com/gabriellsalesx/blup/studio/studio";
import { setLanguage } from "./i18n";

Language()
  .then(setLanguage)
  .catch(() => {})
  .finally(() => mount(App, { target: document.getElementById("app")! }));
