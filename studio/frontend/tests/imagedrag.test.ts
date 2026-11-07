// Dragging an image in the editor moves its Markdown between the blocks of
// the note, and an image that comes as a data: address never lands as text.
import { test } from "node:test";
import assert from "node:assert/strict";
import { EditorState } from "@codemirror/state";
import { ensureSyntaxTree } from "@codemirror/language";
import { markdown, markdownLanguage } from "@codemirror/lang-markdown";
import { dataImage, dropPoint, moveImage, removeImage } from "../src/imagedrag.ts";

const IMG = "![](foto.png)";

function state(doc: string) {
  const s = EditorState.create({ doc, extensions: [markdown({ base: markdownLanguage })] });
  ensureSyntaxTree(s, s.doc.length, 5000);
  return s;
}
// The note after dropping the image on the text `on`, above or below it.
function drop(doc: string, on: string, below: boolean, image = IMG) {
  const s = state(doc);
  const from = doc.indexOf(image);
  const at = dropPoint(s, doc.indexOf(on) + 1, below);
  const changes = moveImage(s, { from, to: from + image.length }, at);
  return changes ? s.update({ changes }).state.doc.toString() : null;
}

test("moves up and down between paragraphs, without piling blank lines", () => {
  const doc = `Um.\n\nDois.\n\n${IMG}\n\nTrês.\n`;
  assert.equal(drop(doc, "Um.", false), `${IMG}\n\nUm.\n\nDois.\n\nTrês.\n`);
  assert.equal(drop(doc, "Um.", true), `Um.\n\n${IMG}\n\nDois.\n\nTrês.\n`);
  assert.equal(drop(doc, "Três.", true), `Um.\n\nDois.\n\nTrês.\n\n${IMG}\n`);
});

test("never lands inside a list, a table or a code block", () => {
  assert.equal(drop(`${IMG}\n\n- um\n- dois\n- três\n\nFim.\n`, "dois", true), `- um\n- dois\n- três\n\n${IMG}\n\nFim.\n`);
  assert.equal(drop(`${IMG}\n\n\`\`\`\ncódigo\nmais\n\`\`\`\n\nFim.\n`, "mais", true), `\`\`\`\ncódigo\nmais\n\`\`\`\n\n${IMG}\n\nFim.\n`);
  assert.equal(drop(`Fim.\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n${IMG}\n`, "| 1", false), `Fim.\n\n${IMG}\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n`);
});

test("stays after the front matter", () => {
  const front = "---\ntitle: Nota\n---\n";
  assert.equal(drop(`${front}\nUm.\n\n${IMG}\n`, "title", false), `${front}\n${IMG}\n\nUm.\n\n`);
});

test("dropped where it is, nothing changes", () => {
  assert.equal(drop(`Um.\n\n${IMG}\n\nDois.\n`, "foto", false), null);
  assert.equal(drop(`Um.\n\n${IMG}\n\nDois.\n`, "foto", true), null);
});

test("an aligned image takes its alignment with it", () => {
  assert.equal(drop(`Um.\n\n${IMG}\n{.text-center}\n\nDois.\n`, "Dois.", true), `Um.\n\nDois.\n\n${IMG}\n{.text-center}\n`);
});

test("an image glued to the text leaves only itself", () => {
  assert.equal(drop(`Um.\n\n${IMG}\nTexto colado.\n`, "Um.", false), `${IMG}\n\nUm.\n\nTexto colado.\n`);
  assert.equal(drop(`Antes ${IMG} depois.\n\nDois.\n`, "Dois.", true), `Antes  depois.\n\nDois.\n\n${IMG}\n`);
});

test("works with ![[image.png]]", () => {
  const wiki = "![[foto.png]]";
  assert.equal(drop(`Um.\n\n${wiki}\n\nDois.\n`, "Um.", false, wiki), `${wiki}\n\nUm.\n\nDois.\n`);
});

test("an image as a data: address becomes a file; other text does not", () => {
  const file = dataImage("data:image/png;base64,iVBORw0KGgo=");
  assert.equal(file?.type, "image/png");
  assert.equal(file?.size, 8);
  assert.equal(dataImage("data:text/plain;base64,aGk="), null);
  assert.equal(dataImage("veja data:image/png;base64,iVBORw0KGgo="), null);
  assert.equal(dataImage("data:image/png;base64,@@@"), null);
  assert.equal(dataImage(undefined), null);
});

test("removing an image takes its line and leaves one blank line", () => {
  const out = (doc: string) => {
    const s = state(doc), from = doc.indexOf(IMG);
    return s.update({ changes: removeImage(s, { from, to: from + IMG.length }) }).state.doc.toString();
  };
  assert.equal(out(`Um.\n\n${IMG}\n\nDois.\n`), "Um.\n\nDois.\n");
  assert.equal(out(`${IMG}\n\nDois.\n`), "Dois.\n");
  assert.equal(out(`Antes ${IMG} depois.\n`), "Antes  depois.\n");
});
