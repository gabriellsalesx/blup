// A picture from the phone, ready to go in the note. A camera photo has
// many megabytes, which the note's vault and the Cloud would carry forever: a
// big one is made smaller, as a JPEG. An animated GIF or an SVG stays as it is.

const LONGEST = 2048; // pixels of the longest side
const SMALL = 1_500_000; // bytes: up to this, a picture stays as it came
const KEPT = ["image/png", "image/jpeg", "image/webp", "image/avif"];

const toBase64 = (file: Blob) => new Promise<string>((ok, fail) => {
  const r = new FileReader();
  r.onload = () => ok(String(r.result).split(",")[1] ?? "");
  r.onerror = () => fail(r.error);
  r.readAsDataURL(file);
});

async function smaller(file: File) {
  const bitmap = await createImageBitmap(file); // upright, whatever the photo says
  const share = Math.min(1, LONGEST / Math.max(bitmap.width, bitmap.height));
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(bitmap.width * share);
  canvas.height = Math.round(bitmap.height * share);
  const pen = canvas.getContext("2d")!;
  pen.fillStyle = "#fff"; // a JPEG has no transparency
  pen.fillRect(0, 0, canvas.width, canvas.height);
  pen.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  bitmap.close();
  const blob = await new Promise<Blob | null>((ok) => canvas.toBlob(ok, "image/jpeg", 0.85));
  if (!blob) throw new Error("image");
  return blob;
}

// The file's name and bytes (base64), as NotesService.AddImage takes them.
export async function prepareImage(file: File) {
  // The phone's picker names a photo by a number: "image" gets it a dated name.
  const own = file.name.replace(/\.[^.]*$/, "");
  const stem = /^\d*$/.test(own) ? "image" : own;
  if (file.type === "image/gif" || file.type === "image/svg+xml" || (KEPT.includes(file.type) && file.size <= SMALL)) {
    const ext = file.type.split("/")[1].replace("jpeg", "jpg").replace("svg+xml", "svg");
    return { name: `${stem}.${ext}`, data: await toBase64(file) };
  }
  return { name: `${stem}.jpg`, data: await toBase64(await smaller(file)) };
}
