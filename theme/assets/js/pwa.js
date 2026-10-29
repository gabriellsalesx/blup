// The site as an app: registers the service worker (sw.js, written by the
// blup CLI after the build) that keeps the site working offline, and shows
// the "install app" button when the browser offers to install it (Chrome,
// Edge, Android; on iPhone it is Share > Add to Home Screen).
const script = document.querySelector("script[data-sw]");
if ("serviceWorker" in navigator && script) {
  navigator.serviceWorker.register(script.dataset.sw, { scope: script.dataset.scope }).catch(() => {});
}

let offer = null;
const buttons = [...document.querySelectorAll("[data-install]")];

addEventListener("beforeinstallprompt", (e) => {
  e.preventDefault(); // the button below asks instead
  offer = e;
  buttons.forEach((b) => { b.hidden = false; });
});

for (const b of buttons) {
  b.addEventListener("click", async () => {
    if (!offer) return;
    offer.prompt();
    await offer.userChoice;
    offer = null;
    buttons.forEach((x) => { x.hidden = true; });
  });
}

addEventListener("appinstalled", () => buttons.forEach((b) => { b.hidden = true; }));
