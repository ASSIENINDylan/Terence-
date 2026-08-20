/* render.js — dessin sur canvas : la carte du monde (caméra suivant le joueur),
   un portrait de face procédural (visage, coiffure, yeux, nez, bouche…) et les
   sprites de combat. Tout est peint au code, aucune image externe.
   Exposé sous SH.render. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;

  const TILE = 64;          // taille d'une case (encore agrandie)
  const SC = TILE / 32;     // facteur d'échelle par rapport au dessin d'origine

  // ============================================================ Carte
  function detailTile(ctx, t, px, py, seed) {
    const r = ((seed * 2654435761) >>> 0) / 4294967296;
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
    ctx.fillStyle = v.color;
    ctx.fillRect(px + 2 * SC, py + 4 * SC, TILE - 4 * SC, 8 * SC);
    ctx.fillRect(px + 8 * SC, py, 5 * SC, 7 * SC);
    ctx.fillRect(px + TILE - 13 * SC, py, 5 * SC, 7 * SC);
    ctx.fillStyle = "#fff";
    ctx.font = "bold " + Math.round(15 * SC) + "px serif";
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText(v.kanji, px + TILE / 2, py + TILE / 2 + 5 * SC);
  }

  // Ninja vu de dessus (carte) — couleurs seulement (les traits du visage ne
  // sont pas visibles à cette échelle).
  function drawHeroTop(ctx, cx, cy, pal, dir, walkPhase) {
    const R = TILE;
    ctx.save();
    ctx.translate(cx, cy);
    ctx.fillStyle = "rgba(0,0,0,.35)";
    ctx.beginPath(); ctx.ellipse(0, R * 0.36, R * 0.30, R * 0.12, 0, 0, 7); ctx.fill();
    const s = walkPhase ? Math.sin(walkPhase) * R * 0.06 : 0;
    ctx.fillStyle = pal.outfitDark;
    ctx.fillRect(-R * 0.22, R * 0.16 + s, R * 0.14, R * 0.14);
    ctx.fillRect(R * 0.08, R * 0.16 - s, R * 0.14, R * 0.14);
    ctx.fillStyle = pal.outfit;
    ctx.beginPath(); ctx.arc(0, 0, R * 0.30, 0, 7); ctx.fill();
    ctx.fillStyle = pal.outfitDark;
    ctx.beginPath(); ctx.arc(R * 0.10, 0, R * 0.20, 0, 7); ctx.fill();
    ctx.fillStyle = pal.hair;
    ctx.beginPath(); ctx.arc(0, -R * 0.07, R * 0.20, 0, 7); ctx.fill();
    ctx.fillStyle = pal.band;
    ctx.fillRect(-R * 0.20, -R * 0.13, R * 0.40, R * 0.09);
    ctx.fillStyle = pal.skin;
    ctx.beginPath(); ctx.arc(0, -R * 0.04, R * 0.15, 0, 7); ctx.fill();
    ctx.fillStyle = "#101319";
    const off = [[0, 1], [-1, 0], [1, 0], [0, -1]][dir] || [0, 1];
    const ex = off[0] * R * 0.03, ey = -R * 0.04 + off[1] * R * 0.03;
    ctx.beginPath(); ctx.arc(ex - R * 0.05, ey, R * 0.028, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.arc(ex + R * 0.05, ey, R * 0.028, 0, 7); ctx.fill();
    ctx.restore();
  }

  function drawWorld(ctx, world, rx, ry, dir, walkPhase) {
    const cw = ctx.canvas.width, ch = ctx.canvas.height;
    let camX = rx * TILE + TILE / 2 - cw / 2;
    let camY = ry * TILE + TILE / 2 - ch / 2;
    camX = Math.max(0, Math.min(world.w * TILE - cw, camX));
    camY = Math.max(0, Math.min(world.h * TILE - ch, camY));
    cam.x = camX; cam.y = camY;

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
    // Marqueur de destination (clic).
    if (moveTarget) {
      const mx = moveTarget.x * TILE - camX + TILE / 2, my = moveTarget.y * TILE - camY + TILE / 2;
      const t = (Date.now() % 1000) / 1000;
      ctx.strokeStyle = "rgba(240,192,74,.85)"; ctx.lineWidth = 2.5;
      ctx.beginPath(); ctx.arc(mx, my, TILE * 0.30, 0, 7); ctx.stroke();
      ctx.strokeStyle = "rgba(240,192,74," + (1 - t).toFixed(2) + ")"; ctx.lineWidth = 3;
      ctx.beginPath(); ctx.arc(mx, my, TILE * 0.12 + t * TILE * 0.28, 0, 7); ctx.stroke();
    }

    drawHeroTop(ctx, rx * TILE + TILE / 2 - camX, ry * TILE + TILE / 2 - camY,
      data.appearancePalette(heroAppearance), dir, walkPhase);
  }

  let heroAppearance = data.defaultAppearance();
  function setHeroAppearance(a) { heroAppearance = data.normAppearance(a); }
  const cam = { x: 0, y: 0 };
  function getCam() { return cam; }
  let moveTarget = null;
  function setMoveTarget(t) { moveTarget = t; }

  // ============================================================ Portrait de face
  // Demi-largeur du visage selon la forme.
  function faceHW(face, R) {
    return R * [0.74, 0.86, 0.78, 0.80, 0.62][face || 0];
  }
  function pathFace(ctx, cx, cy, R, face) {
    const hw = faceHW(face, R);
    ctx.beginPath();
    if (face === 2) { // carré (rectangle arrondi)
      const x = cx - hw, y = cy - R, w = 2 * hw, h = 2 * R, rr = R * 0.32;
      ctx.moveTo(x + rr, y);
      ctx.arcTo(x + w, y, x + w, y + h, rr); ctx.arcTo(x + w, y + h, x, y + h, rr);
      ctx.arcTo(x, y + h, x, y, rr); ctx.arcTo(x, y, x + w, y, rr);
    } else if (face === 3) { // anguleux (menton pointu)
      ctx.moveTo(cx - hw, cy - R * 0.55);
      ctx.lineTo(cx - hw * 0.8, cy - R);
      ctx.lineTo(cx + hw * 0.8, cy - R);
      ctx.lineTo(cx + hw, cy - R * 0.55);
      ctx.lineTo(cx + hw * 0.45, cy + R);
      ctx.lineTo(cx - hw * 0.45, cy + R);
    } else { // ovale / rond / fin
      const ry = face === 1 ? R * 0.92 : (face === 4 ? R * 1.02 : R);
      ctx.ellipse(cx, cy, hw, ry, 0, 0, Math.PI * 2);
    }
    ctx.closePath();
  }

  // Dessine un visage complet centré en (cx,cy), demi-hauteur R.
  function drawFace(ctx, cx, cy, R, app) {
    app = data.normAppearance(app);
    const pal = data.appearancePalette(app);
    const hw = faceHW(app.face, R);

    // --- cheveux arrière (long / afro / queue / chignon) ---
    ctx.fillStyle = pal.hair;
    if (app.hair === 3) { // long : rideau derrière jusqu'aux épaules
      ctx.beginPath(); ctx.moveTo(cx - hw * 1.05, cy - R * 0.4);
      ctx.lineTo(cx - hw * 1.15, cy + R * 1.5); ctx.lineTo(cx + hw * 1.15, cy + R * 1.5);
      ctx.lineTo(cx + hw * 1.05, cy - R * 0.4); ctx.closePath(); ctx.fill();
    } else if (app.hair === 5) { // afro : grosse masse arrière
      ctx.beginPath(); ctx.arc(cx, cy - R * 0.25, hw * 1.35, 0, 7); ctx.fill();
    } else if (app.hair === 4) { // queue de cheval
      ctx.beginPath(); ctx.ellipse(cx + hw * 1.1, cy - R * 0.2, R * 0.16, R * 0.5, 0.3, 0, 7); ctx.fill();
    } else if (app.hair === 7) { // chignon
      ctx.beginPath(); ctx.arc(cx, cy - R * 1.05, R * 0.3, 0, 7); ctx.fill();
    }

    // --- cou ---
    ctx.fillStyle = pal.skinDark;
    ctx.fillRect(cx - R * 0.22, cy + R * 0.7, R * 0.44, R * 0.5);

    // --- oreilles ---
    ctx.fillStyle = pal.skin;
    ctx.beginPath(); ctx.ellipse(cx - hw * 0.98, cy + R * 0.08, R * 0.14, R * 0.2, 0, 0, 7); ctx.fill();
    ctx.beginPath(); ctx.ellipse(cx + hw * 0.98, cy + R * 0.08, R * 0.14, R * 0.2, 0, 0, 7); ctx.fill();

    // --- visage ---
    pathFace(ctx, cx, cy, R, app.face); ctx.fillStyle = pal.skin; ctx.fill();
    ctx.save(); pathFace(ctx, cx, cy, R, app.face); ctx.clip(); // ombrage doux à droite
    ctx.fillStyle = pal.skinDark; ctx.globalAlpha = 0.25;
    ctx.fillRect(cx, cy - R, hw + 2, 2 * R); ctx.restore();

    const eyeY = cy - R * 0.02, eyeX = R * 0.36;

    // --- sourcils (couleur cheveux) ---
    ctx.strokeStyle = pal.hair; ctx.lineWidth = R * 0.07; ctx.lineCap = "round";
    const browTilt = app.eyes === 2 ? R * 0.08 : (app.eyes === 3 ? -R * 0.05 : 0); // perçants=froncés, doux=relevés
    [-1, 1].forEach((s) => {
      ctx.beginPath();
      ctx.moveTo(cx + s * (eyeX - R * 0.14), eyeY - R * 0.24 + s * 0 - browTilt * (s < 0 ? 1 : 1));
      ctx.lineTo(cx + s * (eyeX + R * 0.12), eyeY - R * 0.28 + browTilt);
      ctx.stroke();
    });

    // --- yeux ---
    drawEyes(ctx, cx, eyeY, eyeX, R, app, pal);

    // --- nez ---
    ctx.strokeStyle = pal.skinDark; ctx.lineWidth = R * 0.05; ctx.lineCap = "round";
    const noseLen = [R * 0.14, R * 0.22, R * 0.3][app.nose || 0];
    ctx.beginPath();
    ctx.moveTo(cx - R * 0.02, eyeY + R * 0.12);
    ctx.lineTo(cx - R * 0.06, eyeY + R * 0.12 + noseLen);
    ctx.lineTo(cx + R * 0.05, eyeY + R * 0.12 + noseLen);
    ctx.stroke();
    if (app.nose === 2) { // pointu : arête marquée
      ctx.beginPath(); ctx.moveTo(cx, eyeY + R * 0.04); ctx.lineTo(cx, eyeY + R * 0.12 + noseLen); ctx.stroke();
    }

    // --- bouche ---
    drawMouth(ctx, cx, cy + R * 0.52, R, app.mouth || 0);

    // --- cheveux avant + bandeau ---
    drawFrontHair(ctx, cx, cy, R, hw, app, pal);
    drawHeadband(ctx, cx, cy, R, hw, pal);
  }

  function drawEyes(ctx, cx, eyeY, eyeX, R, app, pal) {
    const style = app.eyes || 0;
    [-1, 1].forEach((s) => {
      const x = cx + s * eyeX;
      let rw = R * 0.2, rh = R * 0.17;
      if (style === 1) { rh = R * 0.12; }           // amande
      else if (style === 2) { rh = R * 0.09; }       // perçants
      else if (style === 4) { rw = R * 0.24; rh = R * 0.2; } // grands
      // blanc de l'œil
      ctx.fillStyle = "#f3f3f0";
      ctx.beginPath(); ctx.ellipse(x, eyeY, rw, rh, 0, 0, 7); ctx.fill();
      // iris
      ctx.fillStyle = pal.eye;
      ctx.beginPath(); ctx.arc(x, eyeY, Math.min(rh, R * 0.11), 0, 7); ctx.fill();
      // pupille + reflet
      ctx.fillStyle = "#101319";
      ctx.beginPath(); ctx.arc(x, eyeY, R * 0.05, 0, 7); ctx.fill();
      ctx.fillStyle = "rgba(255,255,255,.85)";
      ctx.beginPath(); ctx.arc(x - R * 0.03, eyeY - R * 0.03, R * 0.02, 0, 7); ctx.fill();
      // paupière supérieure (contour)
      ctx.strokeStyle = pal.skinDark; ctx.lineWidth = R * 0.03;
      ctx.beginPath(); ctx.ellipse(x, eyeY, rw, rh, 0, Math.PI * 1.05, Math.PI * 1.95); ctx.stroke();
    });
  }

  function drawMouth(ctx, cx, y, R, style) {
    ctx.strokeStyle = "#8a3b34"; ctx.fillStyle = "#8a3b34"; ctx.lineWidth = R * 0.06; ctx.lineCap = "round";
    const w = R * 0.32;
    ctx.beginPath();
    if (style === 1) {           // sourire
      ctx.moveTo(cx - w, y - R * 0.02); ctx.quadraticCurveTo(cx, y + R * 0.2, cx + w, y - R * 0.02); ctx.stroke();
    } else if (style === 2) {    // sévère
      ctx.moveTo(cx - w, y + R * 0.06); ctx.quadraticCurveTo(cx, y - R * 0.12, cx + w, y + R * 0.06); ctx.stroke();
    } else if (style === 3) {    // ouverte
      ctx.ellipse(cx, y + R * 0.03, w * 0.8, R * 0.14, 0, 0, 7); ctx.fill();
      ctx.fillStyle = "#f3f3f0"; ctx.fillRect(cx - w * 0.6, y - R * 0.05, w * 1.2, R * 0.05);
    } else if (style === 4) {    // malicieux (asymétrique)
      ctx.moveTo(cx - w, y + R * 0.04); ctx.quadraticCurveTo(cx + w * 0.4, y + R * 0.12, cx + w, y - R * 0.08); ctx.stroke();
    } else {                     // neutre
      ctx.moveTo(cx - w, y); ctx.lineTo(cx + w, y); ctx.stroke();
    }
  }

  // Cheveux de devant (mèche sur le front) selon la coiffure.
  function drawFrontHair(ctx, cx, cy, R, hw, app, pal) {
    if (app.hair === 0) return; // chauve
    ctx.fillStyle = pal.hair;
    const top = cy - R * 1.02;
    const sideOut = app.hair === 6 ? hw * 0.55 : hw * 1.05; // undercut : côtés rasés
    if (app.hair === 2) {
      // hérissé : pics
      ctx.beginPath();
      ctx.moveTo(cx - sideOut, cy - R * 0.35);
      const spikes = 7;
      for (let i = 0; i <= spikes; i++) {
        const t = i / spikes, x = cx - sideOut + t * 2 * sideOut;
        const up = (i % 2 === 0) ? R * 1.25 : R * 0.95;
        ctx.lineTo(x, cy - up);
      }
      ctx.lineTo(cx + sideOut, cy - R * 0.35);
      ctx.quadraticCurveTo(cx, cy - R * 0.62, cx - sideOut, cy - R * 0.35);
      ctx.closePath(); ctx.fill();
    } else {
      // calotte lisse avec frange
      ctx.beginPath();
      ctx.moveTo(cx - sideOut, cy - R * 0.3);
      ctx.quadraticCurveTo(cx - hw * 1.1, top, cx, top);
      ctx.quadraticCurveTo(cx + hw * 1.1, top, cx + sideOut, cy - R * 0.3);
      // frange (bord bas ondulé sur le front)
      ctx.quadraticCurveTo(cx + hw * 0.4, cy - R * 0.42, cx + hw * 0.1, cy - R * 0.5);
      ctx.quadraticCurveTo(cx, cy - R * 0.38, cx - hw * 0.2, cy - R * 0.5);
      ctx.quadraticCurveTo(cx - hw * 0.5, cy - R * 0.42, cx - sideOut, cy - R * 0.3);
      ctx.closePath(); ctx.fill();
    }
  }

  function drawHeadband(ctx, cx, cy, R, hw, pal) {
    const y = cy - R * 0.62, h = R * 0.2;
    ctx.fillStyle = pal.band;
    ctx.fillRect(cx - hw * 1.02, y, hw * 2.04, h);
    // plaque métallique centrale
    ctx.fillStyle = "#c9ccd2";
    const pw = R * 0.5, ph = h * 0.82;
    ctx.fillRect(cx - pw / 2, y + (h - ph) / 2, pw, ph);
    ctx.strokeStyle = "#8a8d94"; ctx.lineWidth = 1.4;
    ctx.strokeRect(cx - pw / 2, y + (h - ph) / 2, pw, ph);
    ctx.beginPath(); ctx.moveTo(cx - pw * 0.3, y + h * 0.5); ctx.lineTo(cx + pw * 0.3, y + h * 0.5); ctx.stroke();
  }

  // Portrait « buste » pour l'éditeur et la fiche.
  function drawPortrait(canvas, app) {
    const ctx = canvas.getContext("2d");
    const W = canvas.width, H = canvas.height;
    const pal = data.appearancePalette(app);
    ctx.clearRect(0, 0, W, H);
    const cx = W / 2, R = H * 0.30, faceCy = H * 0.42;
    // épaules (tenue)
    ctx.fillStyle = pal.outfit;
    ctx.beginPath();
    ctx.moveTo(cx - W * 0.42, H); ctx.quadraticCurveTo(cx - W * 0.42, H * 0.72, cx, H * 0.72);
    ctx.quadraticCurveTo(cx + W * 0.42, H * 0.72, cx + W * 0.42, H); ctx.closePath(); ctx.fill();
    ctx.fillStyle = pal.band; ctx.fillRect(cx - W * 0.06, H * 0.72, W * 0.12, H * 0.28); // col
    drawFace(ctx, cx, faceCy, R, app);
  }

  // ============================================================ Sprites de combat
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
    ctx.beginPath(); ctx.ellipse(0, 38, 26, 7, 0, 0, 7); ctx.fill();
    // corps (tenue) + ombrage
    ctx.fillStyle = pal.outfit;
    ctx.beginPath(); ctx.moveTo(-16, 38); ctx.lineTo(-12, -2); ctx.lineTo(12, -2); ctx.lineTo(16, 38); ctx.closePath(); ctx.fill();
    ctx.fillStyle = pal.outfitDark;
    ctx.beginPath(); ctx.moveTo(0, -2); ctx.lineTo(12, -2); ctx.lineTo(16, 38); ctx.lineTo(0, 38); ctx.closePath(); ctx.fill();
    // ceinture (bandeau)
    ctx.fillStyle = pal.band; ctx.fillRect(-16, 18, 32, 5);
    // tête (visage complet)
    drawFace(ctx, 0, -20, 15, appearance);
    ctx.restore();
  }

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

  SH.render = { TILE, drawWorld, drawPortrait, drawFace, drawNinjaSprite, drawMonsterSprite, setHeroAppearance, getCam, setMoveTarget };
})(window);
