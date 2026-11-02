import {it,expect} from "vitest";
import {render} from "svelte/server";
import {setLanguage} from "#lib/i18n.svelte.ts";
import Projects from "../routes/+page.svelte";
import Monitors from "../routes/monitores/+page.svelte";
import Alerts from "../routes/avisos/+page.svelte";
import Pages from "../routes/paginas/+page.svelte";
it.each([['pt','Projetos','Monitores','Página de status'],['en','Projects','Monitors','Status pages'],['es','Proyectos','Monitores','Páginas de estado']] as const)("renders Pulse screens in %s without starting network requests",(language,projects,monitors,pages)=>{
 setLanguage(language);
 expect(render(Alerts).body).toContain(language==="en"?"Alerts":"Avisos");
 expect(render(Projects).body).toContain(projects);expect(render(Monitors).body).toContain(monitors);expect(render(Pages).body).toContain(pages);
});
