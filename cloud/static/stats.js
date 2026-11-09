// The visit statistics (Rybbit), as Zaraz used to load them: a file, not
// inline code, so the Content-Security-Policy only allows app.rybbit.io
// (shared/sentry-csp.ts). Never in the staff area.
(function () {
	var h = location.hostname;
	if (h === "staff.getblup.io" || h === "localhost" || h.endsWith(".localhost") || window.__blupRybbit) return;
	window.__blupRybbit = 1;
	var s = document.createElement("script");
	s.src = "https://app.rybbit.io/api/script.js?siteId=6e781295a801";
	s.defer = true;
	document.head.appendChild(s);
})();
