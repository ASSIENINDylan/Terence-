/* render.js — dessin sur canvas : la carte du monde (caméra suivant le joueur)
   et des sprites procéduraux (ninja / monstres) réutilisés en combat.
   Tout est peint au code, aucune image externe. Exposé sous SH.render. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;
  const TILE = 32;

  // Détail décoratif d'une tuile selon son terrain.
  function detailTile(ctx, t, px, py, seed) {
    const r = ((seed * 2654435761) >>> 0) / 4294967296; // pseudo-aléa stable par tuile
    ctx.save();
    if (t === data.TERR.FORET.id) {
      ctx.fillStyle = "#20401f";
      for (let i = 0; i < 3; i++) {
        const ox = 6 + ((r * 999 + i * 90) % 18), oy = 6 + ((r * 555 + i * 70) % 18);
        ctx.beginPath(); ctx.arc(px + ox, py + oy, 4, 0, 7); ctx.fill();
      }
    } else if (t === data.TERR.MONTAGNE.id) {
      ctx.fillStyle = "#8b8377";
      ctx.beginPath();
      ctx.moveTo(px + 6, py + 26); ctx.lineTo(px + 16, py + 7); ctx.lineTo(px + 26, py + 26);
      ctx.closePath(); ctx.fill();
      ctx.fillStyle = "#e9e6df";
      ctx.beginPath(); ctx.moveTo(px + 13, py + 13); ctx.lineTo(px + 16, py + 7); ctx.lineTo(px + 19, py + 13); ctx.closePath(); ctx.fill();
    } else if (t === data.TERR.EAU.id) {
      ctx.strokeStyle = "rgba(255,255,255,.15)"; ctx.lineWidth = 1.5;
      ctx.beginPath();
      ctx.moveTo(px + 5, py + 12 + (r * 6)); ctx.quadraticCurveTo(px + 16, py + 8 + (r * 6), px + 27, py + 12 + (r * 6));
      ctx.stroke();
    } else if (t === data.TERR.DESERT.id) {
      ctx.fillStyle = "rgba(120,95,40,.4)";
      ctx.fillRect(px + 6 + (r * 10), py + 20, 3, 3);
      ctx.fillRect(px + 18, py + 10 + (r * 8), 3, 3);
    } else if (t === data.TERR.ROUTE.id) {
      ctx.strokeStyle = "rgba(70,60,40,.5)"; ctx.setLineDash([4, 4]); ctx.lineWidth = 2;
      ctx.beginPath(); ctx.moveTo(px + 16, py); ctx.lineTo(px + 16, py + TILE); ctx.stroke();
    }
    ctx.restore();
  }

  function drawVillage(ctx, v, px, py) {
    // bâtiment
    ctx.fillStyle = "#2a303d";
    ctx.fillRect(px + 3, py + 8, TILE - 6, TILE - 12);
    // toit (torii) coloré du village
    ctx.fillStyle = v.color;
    ctx.fillRect(px + 1, py + 4, TILE - 2, 7);
    ctx.fillRect(px + 6, py, 4, 6);
    ctx.fillRect(px + TILE - 10, py, 4, 6);
    // kanji
    ctx.fillStyle = "#fff";
    ctx.font = "bold 13px serif";
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText(v.kanji, px + TILE / 2, py + TILE / 2 + 4);
  }

  // Ninja vu de dessus (carte). dir : 0 bas,1 gauche,2 droite,3 haut.
  function drawHeroTop(ctx, cx, cy, color, dir, walkPhase) {
    ctx.save();
    ctx.translate(cx, cy);
    // ombre
    ctx.fillStyle = "rgba(0,0,0,.35)";
    ctx.beginPath(); ctx.ellipse(0, 12, 10, 4, 0, 0, 7); ctx.fill();
    // corps
    ctx.fillStyle = "#20242e";
    ctx.beginPath(); ctx.arc(0, 0, 10, 0, 7); ctx.fill();
    // écharpe / bandeau du village
    ctx.fillStyle = color;
    ctx.fillRect(-10, -3, 20, 4);
    // tête
    ctx.fillStyle = "#d9b48a";
    ctx.beginPath(); ctx.arc(0, -2, 5, 0, 7); ctx.fill();
    // regard selon la direction
    ctx.fillStyle = "#101319";
    const eye = [[0, 4], [-4, 0], [4, 0], [0, -4]][dir] || [0, 4];
    ctx.beginPath(); ctx.arc(eye[0] * 0.6 - 2, -2 + eye[1] * 0.4, 1.4, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.arc(eye[0] * 0.6 + 2, -2 + eye[1] * 0.4, 1.4, 0, 7); ctx.fill();
    // petit balancement de marche
    if (walkPhase) {
      ctx.fillStyle = "#20242e";
      const s = Math.sin(walkPhase) * 3;
      ctx.fillRect(-9, 6 + s, 4, 4);
      ctx.fillRect(5, 6 - s, 4, 4);
    }
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

    // Villages (le secret reste invisible tant qu'il n'est pas découvert).
    for (const v of world.villages) {
      if (v.hidden && !v.discovered) continue;
      const px = v.x * TILE - camX, py = v.y * TILE - camY;
      if (px < -TILE || py < -TILE || px > cw || py > ch) continue;
      drawVillage(ctx, v, px, py);
    }

    // Joueur.
    drawHeroTop(ctx, rx * TILE + TILE / 2 - camX, ry * TILE + TILE / 2 - camY, heroColor, dir, walkPhase);
  }

  // Couleur du bandeau du héros = couleur de son village (fixée par game.js).
  let heroColor = "#e0553b";
  function setHeroVillage(id) {
    const v = data.VILLAGE_BY_ID[id];
    heroColor = v ? v.color : "#e0553b";
  }

  // Sprite de combat (ninja ou monstre) dans un mini-canvas de 96×96.
  function drawFighter(canvas, palette, facing) {
    const ctx = canvas.getContext("2d");
    const W = canvas.width, H = canvas.height;
    ctx.clearRect(0, 0, W, H);
    ctx.save();
    ctx.translate(W / 2, H / 2);
    if (facing === "left") ctx.scale(-1, 1);
    // ombre
    ctx.fillStyle = "rgba(0,0,0,.35)";
    ctx.beginPath(); ctx.ellipse(0, 34, 26, 7, 0, 0, 7); ctx.fill();
    // corps
    ctx.fillStyle = palette[0];
    ctx.beginPath(); ctx.moveTo(-16, 34); ctx.lineTo(-12, -6); ctx.lineTo(12, -6); ctx.lineTo(16, 34); ctx.closePath(); ctx.fill();
    // ombrage
    ctx.fillStyle = palette[1];
    ctx.beginPath(); ctx.moveTo(0, -6); ctx.lineTo(12, -6); ctx.lineTo(16, 34); ctx.lineTo(0, 34); ctx.closePath(); ctx.fill();
    // tête
    ctx.fillStyle = "#d9b48a";
    ctx.beginPath(); ctx.arc(0, -18, 12, 0, 7); ctx.fill();
    // bandeau / accent
    ctx.fillStyle = palette[2];
    ctx.fillRect(-12, -24, 24, 6);
    // yeux
    ctx.fillStyle = "#15181f";
    ctx.beginPath(); ctx.arc(-4, -16, 1.8, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.arc(4, -16, 1.8, 0, 7); ctx.fill();
    ctx.restore();
  }

  SH.render = { TILE, drawWorld, drawFighter, setHeroVillage };
})(window);
