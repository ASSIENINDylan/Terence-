/* world.js — génération de la carte du pays de Yuukan.
   Une grille de tuiles (terrains de data.js), 3 villages cachés reliés par des
   routes, plus un 4e village secret à découvrir. Déterministe via une graine.
   Exposé sous SH.world. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const T = SH.data.TERR;

  const W = 48, H = 34;

  function idx(x, y) { return y * W + x; }
  function inBounds(x, y) { return x >= 0 && y >= 0 && x < W && y < H; }

  // Estampe une « tache » de terrain autour d'un centre (rayon bruité).
  function blob(tiles, rng, cx, cy, radius, terrId, overOnly) {
    const r2 = radius * radius;
    for (let y = cy - radius; y <= cy + radius; y++) {
      for (let x = cx - radius; x <= cx + radius; x++) {
        if (!inBounds(x, y)) continue;
        const d2 = (x - cx) * (x - cx) + (y - cy) * (y - cy);
        if (d2 > r2 * (0.6 + rng() * 0.6)) continue;
        const cur = tiles[idx(x, y)];
        if (overOnly && overOnly.indexOf(cur) === -1) continue;
        tiles[idx(x, y)] = terrId;
      }
    }
  }

  // Estampe un village en forme de croix (centre + 4 voisins) pour la lisibilité.
  function stampVillage(tiles, cx, cy) {
    [[0, 0], [1, 0], [-1, 0], [0, 1], [0, -1]].forEach(([dx, dy]) => {
      if (inBounds(cx + dx, cy + dy)) tiles[idx(cx + dx, cy + dy)] = T.VILLAGE.id;
    });
  }

  // Trace une route en L entre deux points (convertit la terre — et l'eau en pont).
  function carveRoad(tiles, x0, y0, x1, y1) {
    let x = x0, y = y0;
    const step = (a, b) => (a < b ? 1 : a > b ? -1 : 0);
    const put = () => {
      if (inBounds(x, y) && tiles[idx(x, y)] !== T.VILLAGE.id) tiles[idx(x, y)] = T.ROUTE.id;
    };
    while (x !== x1) { put(); x += step(x, x1); }
    while (y !== y1) { put(); y += step(y, y1); }
    put();
  }

  function generate(seed) {
    seed = (seed >>> 0) || 1;
    const rng = SH.rng.make(seed);
    const tiles = new Uint8Array(W * H).fill(T.PLAINE.id);

    // Grand désert dans le coin sud-est.
    blob(tiles, rng, W - 8, H - 6, 9, T.DESERT.id, [T.PLAINE.id]);

    // Lacs et rivières.
    for (let i = 0; i < 5; i++) {
      blob(tiles, rng, 3 + Math.floor(rng() * (W - 6)), 3 + Math.floor(rng() * (H - 6)),
        2 + Math.floor(rng() * 3), T.EAU.id, [T.PLAINE.id, T.DESERT.id]);
    }

    // Chaînes de montagnes.
    for (let i = 0; i < 4; i++) {
      blob(tiles, rng, 4 + Math.floor(rng() * (W - 8)), 4 + Math.floor(rng() * (H - 8)),
        2 + Math.floor(rng() * 3), T.MONTAGNE.id, [T.PLAINE.id]);
    }

    // Forêts (nombreuses).
    for (let i = 0; i < 10; i++) {
      blob(tiles, rng, 3 + Math.floor(rng() * (W - 6)), 3 + Math.floor(rng() * (H - 6)),
        2 + Math.floor(rng() * 4), T.FORET.id, [T.PLAINE.id]);
    }

    // Emplacements des 3 villages (jitter léger), + 1 village secret.
    const anchors = [
      { def: SH.data.VILLAGE_BY_ID.zambakro, fx: 0.20, fy: 0.24 },
      { def: SH.data.VILLAGE_BY_ID.abidjan,  fx: 0.78, fy: 0.22 },
      { def: SH.data.VILLAGE_BY_ID.akradjo,  fx: 0.44, fy: 0.78 },
    ];
    const villages = anchors.map((a) => {
      let cx = Math.round(a.fx * (W - 6)) + 3 + (Math.floor(rng() * 3) - 1);
      let cy = Math.round(a.fy * (H - 6)) + 3 + (Math.floor(rng() * 3) - 1);
      cx = Math.max(2, Math.min(W - 3, cx));
      cy = Math.max(2, Math.min(H - 3, cy));
      // Assainir les alentours (pas de village au milieu d'un lac).
      blob(tiles, rng, cx, cy, 2, T.PLAINE.id, [T.EAU.id, T.MONTAGNE.id]);
      stampVillage(tiles, cx, cy);
      return { id: a.def.id, name: a.def.name, kanji: a.def.kanji, color: a.def.color, x: cx, y: cy, hidden: false };
    });

    // Village secret « Yami » (l'Ombre), caché au fond du désert, à découvrir.
    let hx = W - 5, hy = H - 4;
    blob(tiles, rng, hx, hy, 2, T.PLAINE.id, [T.EAU.id, T.MONTAGNE.id]);
    stampVillage(tiles, hx, hy);
    const hidden = { id: "yami", name: "Yami", kanji: "闇", color: "#9b5de5", x: hx, y: hy, hidden: true, discovered: false };
    villages.push(hidden);

    // Routes reliant les 3 grands villages entre eux.
    carveRoad(tiles, villages[0].x, villages[0].y, villages[1].x, villages[1].y);
    carveRoad(tiles, villages[1].x, villages[1].y, villages[2].x, villages[2].y);
    carveRoad(tiles, villages[2].x, villages[2].y, villages[0].x, villages[0].y);

    return { w: W, h: H, tiles, villages, seed };
  }

  // Renvoie le village dont la « croix » contient (x,y), ou null.
  function villageAt(world, x, y) {
    for (const v of world.villages) {
      if (Math.abs(v.x - x) + Math.abs(v.y - y) <= 1) return v;
    }
    return null;
  }

  // Coût de traversée d'une tuile (pondère le chemin : préfère routes/villages).
  function stepCost(world, x, y) {
    const t = SH.data.TERR_BY_ID[world.tiles[idx(x, y)]];
    return t.walk ? Math.max(0.5, t.chakra) : Infinity;
  }

  // Plus court chemin (Dijkstra 4-directions) de `s` à `g`, pondéré par le coût
  // de terrain. Renvoie la liste des pas (hors départ, `g` inclus) ou null.
  function findPath(world, s, g) {
    if (!inBounds(g.x, g.y) || !SH.data.TERR_BY_ID[world.tiles[idx(g.x, g.y)]].walk) return null;
    if (s.x === g.x && s.y === g.y) return [];
    const N = W * H;
    const dist = new Float64Array(N).fill(Infinity);
    const prev = new Int32Array(N).fill(-1);
    const start = idx(s.x, s.y), goal = idx(g.x, g.y);
    dist[start] = 0;
    // Tas binaire minimal (index, priorité).
    const heap = [{ i: start, d: 0 }];
    const push = (n) => {
      heap.push(n); let c = heap.length - 1;
      while (c > 0) { const p = (c - 1) >> 1; if (heap[p].d <= heap[c].d) break; [heap[p], heap[c]] = [heap[c], heap[p]]; c = p; }
    };
    const pop = () => {
      const top = heap[0], last = heap.pop();
      if (heap.length) { heap[0] = last; let c = 0;
        for (;;) { let l = 2 * c + 1, r = l + 1, m = c;
          if (l < heap.length && heap[l].d < heap[m].d) m = l;
          if (r < heap.length && heap[r].d < heap[m].d) m = r;
          if (m === c) break; [heap[m], heap[c]] = [heap[c], heap[m]]; c = m; } }
      return top;
    };
    while (heap.length) {
      const { i, d } = pop();
      if (i === goal) break;
      if (d > dist[i]) continue;
      const x = i % W, y = (i / W) | 0;
      const nb = [[x + 1, y], [x - 1, y], [x, y + 1], [x, y - 1]];
      for (const [nx, ny] of nb) {
        if (!inBounds(nx, ny)) continue;
        const c = stepCost(world, nx, ny);
        if (c === Infinity) continue;
        const j = idx(nx, ny), nd = d + c;
        if (nd < dist[j]) { dist[j] = nd; prev[j] = i; push({ i: j, d: nd }); }
      }
    }
    if (dist[goal] === Infinity) return null;
    const path = [];
    for (let cur = goal; cur !== start && cur !== -1; cur = prev[cur]) {
      path.push({ x: cur % W, y: (cur / W) | 0 });
    }
    path.reverse();
    return path;
  }

  SH.world = { W, H, generate, idx, inBounds, villageAt, findPath };
})(window);
