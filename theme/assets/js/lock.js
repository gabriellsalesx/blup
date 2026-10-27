// Unlocks a password-protected post. The content was encrypted at build time
// (AES-256-GCM, key from PBKDF2-SHA256; see internal/lock); nothing readable
// is in the page until the right password is typed. Passwords that worked
// are remembered for this tab only (sessionStorage), so the site-wide
// password unlocks the other protected posts without asking again.
const KEY = "blup:passwords";
const enc = new TextEncoder();
const bytes = (b64) => Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));

async function decrypt(password, d) {
  const base = await crypto.subtle.importKey("raw", enc.encode(password), "PBKDF2", false, ["deriveKey"]);
  const key = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt: bytes(d.salt), iterations: Number(d.iter), hash: "SHA-256" },
    base, { name: "AES-GCM", length: 256 }, false, ["decrypt"]);
  const plain = await crypto.subtle.decrypt({ name: "AES-GCM", iv: bytes(d.iv) }, key, bytes(d.ct));
  return new TextDecoder().decode(plain);
}

function remembered() {
  try { return JSON.parse(sessionStorage.getItem(KEY)) || []; } catch { return []; }
}
function remember(pw) {
  try {
    const list = [pw, ...remembered().filter((p) => p !== pw)].slice(0, 5);
    sessionStorage.setItem(KEY, JSON.stringify(list));
  } catch {}
}

async function setup(box) {
  const data = box.querySelector("[data-blup-cipher]")?.dataset;
  const form = box.querySelector("form");
  const target = box.querySelector("[data-blup-content]");
  const error = box.querySelector("[data-blup-error]");
  if (!data || !form || !target) return;

  const open = (html) => {
    target.innerHTML = html;
    target.hidden = false;
    form.hidden = true;
    box.classList.add("is-unlocked");
  };

  for (const pw of remembered()) {
    try { open(await decrypt(pw, data)); return; } catch {}
  }

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const input = form.querySelector("input");
    const button = form.querySelector("button");
    const label = button.textContent;
    error.hidden = true;
    button.disabled = true;
    button.textContent = button.dataset.busy || label;
    try {
      open(await decrypt(input.value, data));
      remember(input.value);
    } catch {
      error.hidden = false;
      input.select();
    } finally {
      button.disabled = false;
      button.textContent = label;
    }
  });
}

document.querySelectorAll("[data-blup-lock]").forEach(setup);
