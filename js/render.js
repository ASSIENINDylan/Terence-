/* render.js — dessin sur canvas : la carte du monde (caméra suivant le joueur)
   et des sprites procéduraux (ninja / monstres) réutilisés en combat et dans
   l'éditeur d'apparence. Tout est peint au code, aucune image externe.
   Exposé sous SH.render. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;

  const TILE = 48;          // taille d'une case (agrandie)
  const SC = TILE / 32;     // facteur d'échelle par rapport au dessin d'origine

  // Détail décoratif d'une tuile selon son terrain.
  function detailTile(ctx, t, px, py, seed) {
    const r = ((seed * 2654435761) >>> 0) / 4294967296; // pseudo-aléa stable par tuile
    ctx.save();
    if (t === data.TERR.FORET.id) {
      ctx.fillStyle = "#20401f";
      for (let i = 0; i < 3; i++) {
        const ox = (6 + ((r * 999 + i * 90) % 18)) * SC, oy = (6 + ((r * 555 + i * 70) % 18)) * SC;
        ctx.beginPath(); ctx.arc(px + ox, py + oy, 4 * SC, 0, 7); ctx.fill();
      }
    } else if (t === data.TERR.MONTAGNE.id) {
      ctx.fillStyle = "#8b8377";
      ctx.beginPath();
      ctx.moveTo(px + 6 * SC, py + 26 * SC); ctx.lineTo(px + 16 * SC, py + 7 * SC); ctx.lineTo(px + 26 * SC, py + 26 * SC);
      ctx.closePath(); ctx.fill();
      ctx.fillStyle = "#e9e6df";
      ctx.beginPath(); ctx.moveTo(px + 13 * SC, py + 13 * SC); ctx.lineTo(px + 16 * SC, py + 7 * SC); ctx.lineTo(px + 19 * SC, py + 13 * SC); ctx.closePath(); ctx.fill();
    } else if (t === data.TERR.EAU.id) {
      ctx.strokeStyle = "rgba(255,255,255,.15)"; ctx.lineWidth = 1.5 * SC;
      ctx.beginPath();
      ctx.moveTo(px + 5 * SC, py + (12 + r * 6) * SC);
      ctx.quadraticCurveTo(px + 16 * SC, py + (8 + r * 6) * SC, px + 27 * SC, py + (12 + r * 6) * SC);
      ctx.stroke();
    } else if (t === data.TERR.DESERT.id) {
      ctx.fillStyle = "rgba(120,95,40,.4)";
      ctx.fillRect(px + (6 + r * 10) * SC, py + 20 * SC, 3 * SC, 3 * SC);
      ctx.fillRect(px + 18 * SC, py + (10 + r * 8) * SC, 3 * SC, 3 * SC);
    } else if (t === data.TERR.ROUTE.id) {
      ctx.strokeStyle = "rgba(70,60,40,.5)"; ctx.setLineDash([4 * SC, 4 * SC]); ctx.lineWidth = 2 * SC;
      ctx.beginPath(); ctx.moveTo(px + TILE / 2, py); ctx.lineTo(px + TILE / 2, py + TILE); ctx.stroke();
    }
    ctx.restore();
  }

  function drawVillage(ctx, v, px, py) {
    ctx.fillStyle = "#2a303d";
    ctx.fillRect(px + 4 * SC, py + 8 * SC, TILE - 8 * SC, TILE - 12 * SC);
    ctx.fillStyle = v.color;                        // toit (torii) coloré
    ctx.fillRect(px + 2 * SC, py + 4 * SC, TILE - 4 * SC, 8 * SC);
    ctx.fillRect(px + 8 * SC, py, 5 * SC, 7 * SC);
    ctx.fillRect(px + TILE - 13 * SC, py, 5 * SC, 7 * SC);
    ctx.fillStyle = "#fff";                          // blason (kanji de discipline)
    ctx.font = "bold " + Math.round(15 * SC) + "px serif";
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText(v.kanji, px + TILE / 2, py + TILE / 2 + 5 * SC);
  }

  // Ninja vu de dessus (carte). pal = SH.data.appearancePalette(appearance).
  function drawHeroTop(ctx, cx, cy, pal, dir, walkPhase) {
    const R = TILE;
    ctx.save();
    ctx.translate(cx, cy);
    // ombre
    ctx.fillStyle = "rgba(0,0,0,.35)";
    ctx.beginPath(); ctx.ellipse(0, R * 0.36, R * 0.30, R * 0.12, 0, 0, 7); ctx.fill();
    // pieds (léger balancement de marche)
    const s = walkPhase ? Math.sin(walkPhase) * R * 0.06 : 0;
    ctx.fillStyle = pal.outfitDark;
    ctx.fillRect(-R * 0.22, R * 0.16 + s, R * 0.14, R * 0.14);
    ctx.fillRect(R * 0.08, R * 0.16 - s, R * 0.14, R * 0.14);
    // corps (tenue)
    ctx.fillStyle = pal.outfit;
    ctx.beginPath(); ctx.arc(0, 0, R * 0.30, 0, 7); ctx.fill();
    ctx.fillStyle = pal.outfitDark;                  // ombrage à droite
    ctx.beginPath(); ctx.arc(R * 0.10, 0, R * 0.20, 0, 7); ctx.fill();
    // cheveux (couronne derrière la tête)
    ctx.fillStyle = pal.hair;
    ctx.beginPath(); ctx.arc(0, -R * 0.07, R * 0.20, 0, 7); ctx.fill();
    // bandeau
    ctx.fillStyle = pal.band;
    ctx.fillRect(-R * 0.20, -R * 0.13, R * 0.40, R * 0.09);
    // tête
    ctx.fillStyle = pal.skin;
    ctx.beginPath(); ctx.arc(0, -R * 0.04, R * 0.15, 0, 7); ctx.fill();
    // regard selon la direction
    ctx.fillStyle = "#101319";
    const off = [[0, 1], [-1, 0], [1, 0], [0, -1]][dir] || [0, 1];
    const ex = off[0] * R * 0.03, ey = -R * 0.04 + off[1] * R * 0.03;
    ctx.beginPath(); ctx.arc(ex - R * 0.05, ey, R * 0.028, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.arc(ex + R * 0.05, ey, R * 0.028, 0, 7); ctx.fill();
    ctx.restore();
  }

  // Dessine la portion visible de la carte, centrée sur (rx,ry) tuiles (float).
  function drawWorld(ctx, world, rx, ry, dir, walkPhase) {
    const cw = ctx.canvas.width, ch = ctx.canvas.height;
    let camX = rx * TILE + TILE / 2 - cw / 2;
    let camY = ry * TILE + TILE / 2 - ch / 2;
    camX = Math.max(0, Math.min(world.w * TILE - cw, camX));
    camY = Math.max(0, Math.min(world.h * TILE - ch, camY));

    const c0 = Math.floor(camX / TILE), r0 = Math.floor(camY / TILE);
    const c1 = Math.min(world.w - 1, c0 + Math.ceil(cw / TILE) + 1);
    const r1 = Math.min(world.h - 1, r0 + Math.ceil(ch / TILE) + 1);

    for (let ty = r0; ty <= r1; ty++) {
      for (let tx = c0; tx <= c1; tx++) {
        const id = world.tiles[SH.world.idx(tx, ty)];
        const t = data.TERR_BY_ID[id];
        const px = tx * TILE - camX, py = ty * TILE - camY;
        ctx.fillStyle = ((tx + ty) & 1) ? t.color : t.color2;
        ctx.fillRect(px, py, TILE, TILE);
        detailTile(ctx, id, px, py, tx * 73856093 ^ ty * 19349663);
      }
    }

    for (const v of world.villages) {
      if (v.hidden && !v.discovered) continue;
      const px = v.x * TILE - camX, py = v.y * TILE - camY;
      if (px < -TILE || py < -TILE || px > cw || py > ch) continue;
      drawVillage(ctx, v, px, py);
    }

    drawHeroTop(ctx, rx * TILE + TILE / 2 - camX, ry * TILE + TILE / 2 - camY,
      data.appearancePalette(heroAppearance), dir, walkPhase);
  }

  // Apparence courante du héros (fixée par game.js à l'entrée en jeu).
  let heroAppearance = data.defaultAppearance();
  function setHeroAppearance(a) { heroAppearance = data.normAppearance(a); }

  // ------------------------------------------------------------- Sprites de combat
  // Ninja de combat / aperçu, dessiné d'après l'apparence (index).
  function drawNinjaSprite(canvas, appearance, facing) {
    const ctx = canvas.getContext("2d");
    const W = canvas.width, H = canvas.height;
    const pal = data.appearancePalette(appearance);
    ctx.clearRect(0, 0, W, H);
    ctx.save();
    ctx.translate(W / 2, H / 2);
    if (facing === "left") ctx.scale(-1, 1);
    // ombre
    ctx.fillStyle = "rgba(0,0,0,.35)";
    ctx.beginPath(); ctx.ellipse(0, 34, 26, 7, 0, 0, 7); ctx.fill();
    // corps (tenue) + ombrage
    ctx.fillStyle = pal.outfit;
    ctx.beginPath(); ctx.moveTo(-16, 34); ctx.lineTo(-12, -6); ctx.lineTo(12, -6); ctx.lineTo(16, 34); ctx.closePath(); ctx.fill();
    ctx.fillStyle = pal.outfitDark;
    ctx.beginPath(); ctx.moveTo(0, -6); ctx.lineTo(12, -6); ctx.lineTo(16, 34); ctx.lineTo(0, 34); ctx.closePath(); ctx.fill();
    // ceinture (bandeau)
    ctx.fillStyle = pal.band;
    ctx.fillRect(-16, 14, 32, 5);
    // tête (peau)
    ctx.fillStyle = pal.skin;
    ctx.beginPath(); ctx.arc(0, -18, 12, 0, 7); ctx.fill();
    // cheveux (calotte)
    ctx.fillStyle = pal.hair;
    ctx.beginPath(); ctx.arc(0, -20, 12, Math.PI, 2 * Math.PI); ctx.fill();
    ctx.fillRect(-12, -22, 24, 4);
    // bandeau frontal
    ctx.fillStyle = pal.band;
    ctx.fillRect(-12, -24, 24, 6);
    // yeux
    ctx.fillStyle = "#15181f";
    ctx.beginPath(); ctx.arc(-4, -16, 1.8, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.arc(4, -16, 1.8, 0, 7); ctx.fill();
    ctx.restore();
  }

  // Monstre de combat (palette [corps, ombre, accent]).
  function drawMonsterSprite(canvas, palette, facing) {
    const ctx = canvas.getContext("2d");
    const W = canvas.width, H = canvas.height;
    ctx.clearRect(0, 0, W, H);
    ctx.save();
    ctx.translate(W / 2, H / 2);
    if (facing === "left") ctx.scale(-1, 1);
    ctx.fillStyle = "rgba(0,0,0,.35)";
    ctx.beginPath(); ctx.ellipse(0, 34, 26, 7, 0, 0, 7); ctx.fill();
    ctx.fillStyle = palette[0];
    ctx.beginPath(); ctx.moveTo(-16, 34); ctx.lineTo(-12, -6); ctx.lineTo(12, -6); ctx.lineTo(16, 34); ctx.closePath(); ctx.fill();
    ctx.fillStyle = palette[1];
    ctx.beginPath(); ctx.moveTo(0, -6); ctx.lineTo(12, -6); ctx.lineTo(16, 34); ctx.lineTo(0, 34); ctx.closePath(); ctx.fill();
    ctx.fillStyle = "#d9b48a";
    ctx.beginPath(); ctx.arc(0, -18, 12, 0, 7); ctx.fill();
    ctx.fillStyle = palette[2];
    ctx.fillRect(-12, -24, 24, 6);
    ctx.fillStyle = "#15181f";
    ctx.beginPath(); ctx.arc(-4, -16, 1.8, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.arc(4, -16, 1.8, 0, 7); ctx.fill();
    ctx.restore();
  }

  SH.render = { TILE, drawWorld, drawNinjaSprite, drawMonsterSprite, setHeroAppearance };
})(window);
