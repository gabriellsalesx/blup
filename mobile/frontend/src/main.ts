import { mount } from "svelte";
import App from "./App.svelte";
import "./app.css";
import { SettingsService } from "../bindings/github.com/gabriellsalesx/blup/mobile";
import { pickLanguage, setLanguage } from "./desk/i18n";

// The language first: the one picked in Account, else the phone's. The Go
// side is told too, for its messages and the reminders.
SettingsService.Language()
  .catch(() => "")
  .then((saved) => {
    const lang = pickLanguage(saved);
    setLanguage(lang);
    // Awaited: the first vault's welcome note is written in this language.
    return SettingsService.UseLanguage(lang).catch(() => {});
  })
  .finally(() => mount(App, { target: document.getElementById("app")! }));
