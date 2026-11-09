// The theme before the first paint (src/lib/theme.svelte.ts keeps it after).
// A file of its own, not inline, so the Content-Security-Policy needs no
// exception for it.
try {
	var t = localStorage.getItem("blup-theme");
	document.documentElement.dataset.theme = t === "light" || t === "dark" ? t : matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
} catch (e) {}
// The texts of the language in use (/i18n/<lang>.json, made at build by shared/i18n-split.ts) start
// downloading now, together with the panel's JavaScript, instead of after
// it. The same choice as src/lib/i18n.svelte.ts (saved, cookie, browser);
// the staff area is always in Portuguese. The panel checks which language
// came and asks again when it is not the one it wants.
(function () {
	var ok = { pt: 1, en: 1, es: 1 }, lang = null, saved = null, cookie = null;
	try { saved = localStorage.getItem("blup-lang"); } catch (e) {}
	try { cookie = (document.cookie.split(";").map(function (v) { return v.trim(); }).filter(function (v) { return v.indexOf("blup-lang=") === 0; })[0] || "").slice(10); } catch (e) {}
	if (location.hostname.indexOf("staff.") === 0 || /^\/staff(\/|$)/.test(location.pathname)) lang = "pt";
	else if (ok[saved]) lang = saved;
	else if (ok[cookie]) lang = cookie;
	else {
		var list = navigator.languages || [navigator.language || ""];
		for (var i = 0; i < list.length && !lang; i++) { var l = String(list[i]).trim().toLowerCase().split(/[-_]/)[0]; if (ok[l]) lang = l; }
		lang = lang || "en";
	}
	try {
		window.__blupTexts = { lang: lang, table: fetch("/i18n/" + lang + ".json").then(function (r) { return r.ok ? r.json() : null; }).catch(function () { return null; }) };
	} catch (e) {}
})();
