import {it,expect} from "vitest";
import {render} from "svelte/server";
import {setLanguage} from "#lib/i18n.svelte.ts";
import Projects from "../routes/+page.svelte";
import Monitors from "../routes/monitors/+page.svelte";
import Alerts from "../routes/alerts/+page.svelte";
import Pages from "../routes/status-pages/+page.svelte";
it.each([['pt','Projetos','Monitores','Página de status'],['en','Projects','Monitors','Status pages'],['es','Proyectos','Monitores','Páginas de estado']] as const)("renders Pulse screens in %s without starting network requests",(language,projects,monitors,pages)=>{
 setLanguage(language);
 const base={signedIn:false};
 expect(render(Alerts,{props:{params:{},data:{...base,projects:[],channels:null}}}).body).toContain(language==="en"?"Alerts":"Avisos");
 expect(render(Projects,{props:{params:{},data:{...base,projects:[]}}}).body).toContain(projects);expect(render(Monitors,{props:{params:{},data:{...base,monitors:[],histories:{}}}}).body).toContain(monitors);expect(render(Pages,{props:{params:{},data:{...base,pages:[]}}}).body).toContain(pages);
});
