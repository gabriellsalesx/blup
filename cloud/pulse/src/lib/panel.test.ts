import {it,expect} from "vitest";
import {render} from "svelte/server";
import {setLanguage} from "#lib/i18n.svelte.ts";
import Projects from "../routes/+page.svelte";
import Monitors from "../routes/monitores/+page.svelte";
import Alerts from "../routes/avisos/+page.svelte";
import Pages from "../routes/paginas/+page.svelte";
it.each([['pt','Projetos','Monitores','Página de status'],['en','Projects','Monitors','Status pages'],['es','Proyectos','Monitores','Páginas de estado']] as const)("renders Pulse screens in %s without starting network requests",(language,projects,monitors,pages)=>{
 setLanguage(language);
 const base={signedIn:false};
 expect(render(Alerts,{props:{data:{...base,projects:[],channels:null}}}).body).toContain(language==="en"?"Alerts":"Avisos");
 expect(render(Projects,{props:{data:{...base,projects:[]}}}).body).toContain(projects);expect(render(Monitors,{props:{data:{...base,monitors:[],histories:{}}}}).body).toContain(monitors);expect(render(Pages,{props:{data:{...base,pages:[],monitors:[]}}}).body).toContain(pages);
});
