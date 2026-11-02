// The canvas (.canvas) and drawing (.draw) files are read from disk, and a
// file may come from another computer through sync: the parsers trust
// nothing in it. Run: npm test (node:test, no deps).
import { test } from "node:test";
import assert from "node:assert/strict";
import { LIMITS, parseCanvas, writeCanvas } from "../src/canvas.ts";
import { parseDraw, writeDraw } from "../src/draw.ts";

const node = (o: Record<string, unknown> = {}) => ({ id: "a", type: "text", text: "x", x: 0, y: 0, width: 250, height: 60, ...o });
const canvas = (nodes: unknown, edges: unknown = []) => parseCanvas(JSON.stringify({ nodes, edges }));

test("canvas: a root that isn't an object is an empty canvas", () => {
  for (const root of ["null", "[]", "[1,2]", "3", '"x"', "true", ""]) {
    const c = parseCanvas(root);
    assert.deepEqual(c, { nodes: [], edges: [], rest: {}, hidden: { nodes: [], edges: [] } }, root);
  }
  // nodes and edges that aren't lists.
  assert.deepEqual(parseCanvas('{"nodes":{"a":1},"edges":"x"}'), { nodes: [], edges: [], rest: {}, hidden: { nodes: [], edges: [] } });
});

test("canvas: unknown kinds and broken items are left out", () => {
  const c = canvas([
    node({ id: "ok" }),
    node({ id: "k", type: "script" }),
    node({ id: "n", type: undefined }),
    { type: "text" },
    null,
    7,
    [1],
    node({ id: "" }),
    node({ id: "f", type: "file" }), // a file card without its file
    node({ id: 5 }),
  ]);
  assert.deepEqual(c.nodes.map((n) => n.id), ["ok"]);
  // The unknown kind is set aside, not shown; the broken ones are gone.
  assert.deepEqual(c.hidden.nodes.map((h) => h.item.id), ["k"]);
});

test("canvas: a plugin's cards and their arrows are kept untouched, never shown", () => {
  const plugin = { id: "p", type: "mindmap", x: "weird", width: -5, payload: { deep: [1, { a: "<script>" }] } };
  const src = JSON.stringify({
    nodes: [node({ id: "a" }), plugin, node({ id: "b" }), { id: "q", type: "excalidraw", x: 1 }],
    edges: [
      { id: "e1", fromNode: "a", toNode: "b" },
      { id: "e2", fromNode: "a", toNode: "p", style: "x" },
      { id: "e3", fromNode: "p", toNode: "q" },
      { id: "e4", fromNode: "p", toNode: "gone" },
    ],
  });
  const c = parseCanvas(src);
  assert.deepEqual(c.nodes.map((n) => n.id), ["a", "b"]);
  assert.deepEqual(c.edges.map((e) => e.id), ["e1"]);
  assert.deepEqual(c.hidden.nodes.map((h) => h.item), [plugin, { id: "q", type: "excalidraw", x: 1 }]);
  assert.deepEqual(c.hidden.edges.map((h) => h.item.id), ["e2", "e3"]);
  // Saved again: everything in its place, the plugin's cards as they came.
  const back = JSON.parse(writeCanvas(c));
  const orig = JSON.parse(src);
  assert.deepEqual(back.nodes, orig.nodes);
  assert.deepEqual(back.edges, orig.edges.slice(0, 3));
  // A card added and one moved don't lose them either.
  c.nodes.push(node({ id: "c" }));
  c.nodes[0].x = 40;
  const again = JSON.parse(writeCanvas(c));
  assert.deepEqual(again.nodes.map((n: { id: string }) => n.id), ["a", "p", "b", "q", "c"]);
  assert.deepEqual(again.nodes[1], plugin);
});

test("canvas: an arrow kept aside goes when its shown card is deleted", () => {
  const c = canvas([node({ id: "a" }), { id: "p", type: "mindmap" }], [{ id: "e", fromNode: "a", toNode: "p" }]);
  assert.equal(c.hidden.edges.length, 1);
  c.nodes = [];
  const back = JSON.parse(writeCanvas(c));
  assert.deepEqual(back.nodes, [{ id: "p", type: "mindmap" }]);
  assert.deepEqual(back.edges, []);
});

test("canvas: kept cards count toward the limits and share the ids", () => {
  const many = Array.from({ length: LIMITS.nodes + 5 }, (_, i) => ({ id: "h" + i, type: "plugin" }));
  const c = canvas([node({ id: "a" }), ...many, node({ id: "z" })]);
  assert.equal(c.nodes.length + c.hidden.nodes.length, LIMITS.nodes);
  assert.deepEqual(c.nodes.map((n) => n.id), ["a"]);
  // A repeated id keeps the first, shown or not.
  const d = canvas([{ id: "x", type: "plugin" }, node({ id: "x" }), node({ id: "y" }), { id: "y", type: "plugin" }]);
  assert.deepEqual(d.nodes.map((n) => n.id), ["y"]);
  assert.deepEqual(d.hidden.nodes.map((h) => h.item.id), ["x"]);
});

test("canvas: missing, NaN or infinite geometry is mended", () => {
  // JSON has no NaN nor Infinity: they arrive as null, strings or huge numbers.
  const c = parseCanvas('{"nodes":[' +
    '{"id":"a","type":"text","text":"x"},' +
    '{"id":"b","type":"group","x":null,"y":"1","width":-5,"height":0},' +
    '{"id":"c","type":"link","url":"https://e.com","x":1e400,"y":-1e400,"width":1e400,"height":1e999}' +
    "]}");
  for (const n of c.nodes) for (const k of ["x", "y", "width", "height"] as const) assert.ok(Number.isFinite(n[k]), `${n.id}.${k}`);
  const [a, b, cc] = c.nodes;
  assert.deepEqual([a.x, a.y, a.width, a.height], [0, 0, 250, 60]);
  assert.deepEqual([b.x, b.y, b.width, b.height], [0, 0, 250, 60]);
  assert.equal(cc.width, 250);
  assert.ok(Math.abs(cc.x) <= 1e7 && cc.x === 0, "1e400 is Infinity: mended to 0");
  // NaN or Infinity set by code (not JSON) behave the same through a guard.
  const d = canvas([node({ id: "d", x: Number.NaN, width: Number.POSITIVE_INFINITY })]);
  assert.equal(d.nodes[0].x, 0);
  assert.equal(d.nodes[0].width, 250);
  // A far position is kept inside the limits.
  assert.equal(canvas([node({ x: 5e9 })]).nodes[0].x, 1e7);
});

test("canvas: repeated ids keep the first", () => {
  const c = canvas([node({ id: "a", text: "first" }), node({ id: "a", text: "second" })], [
    { id: "e", fromNode: "a", toNode: "a" },
    { id: "e", fromNode: "a", toNode: "a", label: "second" },
  ]);
  assert.equal(c.nodes.length, 1);
  assert.equal(c.nodes[0].text, "first");
  assert.equal(c.edges.length, 1);
  assert.equal(c.edges[0].label, undefined);
});

test("canvas: arrows to a missing card are left out; bad sides and ends are dropped", () => {
  const c = canvas([node({ id: "a" }), node({ id: "b" })], [
    { id: "1", fromNode: "a", toNode: "b", fromSide: "up", toSide: "left", fromEnd: "boom", toEnd: "arrow", color: 3, label: {} },
    { id: "2", fromNode: "a", toNode: "zzz" },
    { id: "3", fromNode: "a" },
    { id: "4", fromNode: ["a"], toNode: "b" },
  ]);
  assert.deepEqual(c.edges.map((e) => e.id), ["1"]);
  const e = c.edges[0];
  assert.equal(e.fromSide, undefined);
  assert.equal(e.toSide, "left");
  assert.equal(e.fromEnd, undefined);
  assert.equal(e.toEnd, "arrow");
  assert.equal("color" in e, false);
  assert.equal("label" in e, false);
});

test("canvas: optional text keys of the wrong type are dropped, the required ones mended", () => {
  const c = canvas([node({ id: "t", text: 5, color: {} }), node({ id: "l", type: "link", url: null, text: undefined })]);
  assert.equal(c.nodes[0].text, "");
  assert.equal("color" in c.nodes[0], false);
  assert.equal(c.nodes[1].url, "");
});

test("canvas: unknown keys survive a round trip (Obsidian, plugins)", () => {
  const src = JSON.stringify({
    nodes: [node({ id: "a", styleAttributes: { shape: "pill" }, "x-plugin": [1, 2] })],
    edges: [{ id: "e", fromNode: "a", toNode: "a", styleAttributes: { path: "dotted" } }],
    metadata: { version: "1.0-1.0", frontmatter: {} },
  });
  const c = parseCanvas(src);
  const back = JSON.parse(writeCanvas(c));
  assert.deepEqual(back.nodes[0].styleAttributes, { shape: "pill" });
  assert.deepEqual(back.nodes[0]["x-plugin"], [1, 2]);
  assert.deepEqual(back.edges[0].styleAttributes, { path: "dotted" });
  assert.deepEqual(back.metadata, { version: "1.0-1.0", frontmatter: {} });
  // And what was valid stays the same.
  assert.deepEqual(parseCanvas(writeCanvas(c)), c);
});

test("canvas: a __proto__ key stays data", () => {
  const c = parseCanvas('{"nodes":[{"id":"a","type":"text","text":"","x":0,"y":0,"width":9,"height":9,"__proto__":{"polluted":1}}],"__proto__":{"polluted":2}}');
  assert.equal(({} as Record<string, unknown>).polluted, undefined);
  assert.equal((c.nodes[0] as Record<string, unknown>).polluted, undefined);
});

const el = (o: Record<string, unknown> = {}) => ({ id: "a", type: "rect", x: 0, y: 0, w: 10, h: 10, stroke: "ink", fill: "none", width: 2, dash: "solid", ...o });
const drawing = (elements: unknown) => parseDraw(JSON.stringify({ version: 1, elements }));

test("draw: a root that isn't an object is an empty drawing", () => {
  for (const root of ["null", "[]", "[{}]", "1", '"x"', ""]) assert.deepEqual(parseDraw(root), { elements: [], rest: {} }, root);
  assert.deepEqual(parseDraw('{"elements":{"0":{}}}'), { elements: [], rest: {} });
});

test("draw: unknown kinds and broken items are left out", () => {
  const d = drawing([el({ id: "ok" }), el({ id: "k", type: "triangle" }), el({ id: "n", type: 3 }), null, "x", el({ id: "" }), el({ id: "i", type: "image" })]);
  assert.deepEqual(d.elements.map((e) => e.id), ["ok"]);
});

test("draw: invalid points are dropped; a line without two points is left out", () => {
  const d = drawing([
    el({ id: "l1", type: "line", points: [[0, 0], [10, "x"], [5], "p", null, [20, 20]] }),
    el({ id: "l2", type: "arrow", points: [[0, 0]] }),
    el({ id: "l3", type: "line", points: "0,0 1,1" }),
    el({ id: "p1", type: "pen", points: [[1, 1]] }),
    el({ id: "p2", type: "pen" }),
    el({ id: "r", type: "rect", points: [[0, 0], [1, 1]] }),
  ]);
  assert.deepEqual(d.elements.map((e) => e.id), ["l1", "p1", "r"]);
  assert.deepEqual(d.elements[0].points, [[0, 0], [20, 20]]);
  assert.equal(d.elements[2].points, undefined, "a rectangle has no points");
});

test("draw: missing or non-finite geometry and style are mended", () => {
  const d = parseDraw('{"elements":[{"id":"a","type":"ellipse","x":null,"y":1e400,"w":"5","stroke":4,"fill":null,"width":-1,"dash":"wavy","fillStyle":"neon","size":"big","text":7}]}');
  const e = d.elements[0];
  assert.deepEqual([e.x, e.y, e.w, e.h], [0, 0, 0, 0]);
  assert.equal(e.stroke, "ink");
  assert.equal(e.fill, "none");
  assert.equal(e.width, 2);
  assert.equal(e.dash, "solid");
  assert.equal("fillStyle" in e, false);
  assert.equal("size" in e, false);
  assert.equal("text" in e, false);
  const t = drawing([el({ id: "t", type: "text", x: Number.NaN })]).elements[0];
  assert.equal(t.x, 0);
  assert.equal(t.text, "");
  // Negative sizes stay (a box drawn up or left; normalize fixes it).
  assert.equal(drawing([el({ w: -30 })]).elements[0].w, -30);
});

test("draw: repeated ids keep the first", () => {
  const d = drawing([el({ id: "a", stroke: "red" }), el({ id: "a", stroke: "blue" })]);
  assert.equal(d.elements.length, 1);
  assert.equal(d.elements[0].stroke, "red");
});

test("draw: unknown keys survive a round trip", () => {
  const src = JSON.stringify({ version: 1, elements: [el({ id: "a", locked: true, meta: { by: "x" } })], background: "#fff" });
  const d = parseDraw(src);
  const back = JSON.parse(writeDraw(d));
  assert.equal(back.elements[0].locked, true);
  assert.deepEqual(back.elements[0].meta, { by: "x" });
  assert.equal(back.background, "#fff");
  assert.deepEqual(parseDraw(writeDraw(d)), d);
});
