// The map of notes' layout: notes push each other away, links pull the
// notes they join, and everything drifts to the middle, a little less each
// step until it settles (a force simulation, written here, no library).

export type GNode = { path: string; name: string; folder: string; kind: string; tags: string[]; created: string; x: number; y: number; vx: number; vy: number; degree: number; pinned?: boolean };
export type GEdge = { a: GNode; b: GNode };

export function place(nodes: GNode[]) {
  // A spiral, so the start is calm and even.
  nodes.forEach((n, i) => {
    const r = 18 * Math.sqrt(i + 1), t = i * 2.4;
    n.x = Math.cos(t) * r; n.y = Math.sin(t) * r; n.vx = 0; n.vy = 0;
  });
}

// How strong each force is (the Forces section of the map's panel); 1 is
// the default of each.
export type Forces = { repel: number; link: number; distance: number; center: number };

// One step; alpha (1 to 0) is how much is still moving.
export function step(nodes: GNode[], edges: GEdge[], alpha: number, f: Forces = { repel: 1, link: 1, distance: 70, center: 1 }) {
  const REPEL = 900 * f.repel, SPRING = 0.04 * f.link, LENGTH = f.distance, GRAVITY = 0.012 * f.center, CUT = 420 * 420;
  for (let i = 0; i < nodes.length; i++) {
    const a = nodes[i];
    for (let j = i + 1; j < nodes.length; j++) {
      const b = nodes[j];
      let dx = a.x - b.x, dy = a.y - b.y;
      let d2 = dx * dx + dy * dy;
      if (d2 > CUT) continue;
      if (d2 < 0.01) { dx = Math.random() - 0.5; dy = Math.random() - 0.5; d2 = 0.5; }
      const f = (REPEL * alpha) / d2;
      a.vx += dx * f; a.vy += dy * f;
      b.vx -= dx * f; b.vy -= dy * f;
    }
  }
  for (const { a, b } of edges) {
    const dx = b.x - a.x, dy = b.y - a.y, d = Math.hypot(dx, dy) || 1;
    const f = ((d - LENGTH) * SPRING * alpha) / d;
    a.vx += dx * f; a.vy += dy * f;
    b.vx -= dx * f; b.vy -= dy * f;
  }
  for (const n of nodes) {
    n.vx -= n.x * GRAVITY * alpha; n.vy -= n.y * GRAVITY * alpha;
    if (n.pinned) { n.vx = 0; n.vy = 0; continue; }
    n.vx *= 0.6; n.vy *= 0.6;
    n.x += Math.max(-40, Math.min(40, n.vx));
    n.y += Math.max(-40, Math.min(40, n.vy));
  }
}

// A node's size: bigger when more links reach it.
export const radius = (n: GNode, scale = 1) => (4 + Math.min(14, Math.sqrt(n.degree) * 2.4)) * scale;
