/* game.js — contrôleur principal : écran-titre, boucle de jeu, la dynamique de
   déplacement (grille + coût en chakra + régénération), les rencontres, l'entrée
   dans les villages et la sauvegarde. Point d'entrée du jeu. Exposé sous SH.game. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;
  const $ = (id) => document.getElementById(id);

  const TILE = SH.render.TILE;
  const WALK_MS = 170;   // délai entre deux pas (marche)
  const RUN_MS = 105;    // délai entre deux pas (course)

  let player = null, world = null, canvas = null, ctx = null;
  let paused = false, inCombat = false;
  let rx = 0, ry = 0, dir = 0, walkPhase = 0;
  let moveTimer = 0, encImmunity = 0;
  let lastTime = 0, rafId = 0;
  const keys = [];       // pile des directions maintenues (dernière = active)

  const NAMES = ["Kaito", "Ren", "Aiko", "Haru", "Sora", "Yuki", "Rin", "Taro", "Mika", "Jin"];

  // -------------------------------------------------------- Écran-titre
  let selectedVillage = "chikara";
  function buildTitle() {
    const wrap = $("village-choice");
    wrap.innerHTML = data.VILLAGES.map((v) =>
      '<div class="village-card' + (v.id === selectedVillage ? " selected" : "") + '" data-v="' + v.id +
      '" style="--vc:' + v.color + '">' +
      '<div class="vc-badge" style="background:' + v.color + '">' + v.kanji + "</div>" +
      "<h3>" + v.name + "</h3>" +
      '<div class="vc-spec">' + v.specName + "<br>" + v.blurb + "</div></div>"
    ).join("");
    wrap.querySelectorAll(".village-card").forEach((c) => {
      c.onclick = () => { selectedVillage = c.dataset.v; buildTitle(); };
    });

    $("hero-name").placeholder = "ex. " + NAMES[Math.floor(Math.random() * NAMES.length)];
    $("btn-continue").disabled = !SH.state.hasSave();
    $("title-hint").textContent = SH.state.hasSave()
      ? "Une sauvegarde existe — « Continuer » la reprend."
      : "Choisis un village puis lance-toi. Ta progression est sauvegardée automatiquement.";
  }

  function startNewGame() {
    const name = ($("hero-name").value || "").trim() || NAMES[Math.floor(Math.random() * NAMES.length)];
    if (SH.state.hasSave()) {
      if (!confirm("Une sauvegarde existe déjà. Démarrer une nouvelle partie l'effacera. Continuer ?")) return;
      SH.state.wipe();
    }
    player = SH.state.createHero(name, selectedVillage);
    world = SH.world.generate(player.seed);
    placeAtHome();
    SH.state.save(player);
    enterGame();
  }

  function continueGame() {
    const p = SH.state.load();
    if (!p) { SH.ui.toast("Aucune sauvegarde."); return; }
    player = p;
    world = SH.world.generate(player.seed);
    // Ré-appliquer les découvertes (villages secrets) sur la carte régénérée.
    for (const v of world.villages) {
      if (v.hidden && player.discovered[v.id]) v.discovered = true;
    }
    // Valider la position.
    if (!player.pos || !SH.world.inBounds(player.pos.x, player.pos.y)) placeAtHome();
    enterGame();
  }

  function placeAtHome() {
    const home = world.villages.find((v) => v.id === player.village) || world.villages[0];
    player.pos = { x: home.x, y: home.y };
  }

  function enterGame() {
    $("title-screen").classList.add("hidden");
    $("game-screen").classList.remove("hidden");
    canvas = $("view");
    ctx = canvas.getContext("2d");
    SH.render.setHeroVillage(player.village);
    rx = player.pos.x; ry = player.pos.y;
    SH.state.clampVitals(player);
    SH.ui.refreshHUD(player, world);
    lastTime = performance.now();
    cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(loop);
    SH.ui.toast("Bienvenue à " + data.VILLAGE_BY_ID[player.village].name + ". Explore le pays de Yuukan.", 4000);
  }

  // -------------------------------------------------------- Déplacement
  function tileAt(x, y) { return world.tiles[SH.world.idx(x, y)]; }

  function tryStep(dt) {
    moveTimer -= dt * 1000;
    const active = keys[keys.length - 1];
    if (active == null) return;
    dir = active;
    if (moveTimer > 0) return;

    const running = keyState.shift;
    const delta = [[0, 1], [-1, 0], [1, 0], [0, -1]][active]; // 0 bas,1 gauche,2 droite,3 haut
    const nx = player.pos.x + delta[0], ny = player.pos.y + delta[1];
    moveTimer = running ? RUN_MS : WALK_MS;

    if (!SH.world.inBounds(nx, ny)) return;
    const t = data.TERR_BY_ID[tileAt(nx, ny)];
    if (!t.walk) { return; } // eau / obstacle

    // Coût en chakra du pas (la course coûte plus cher).
    let cost = t.chakra;
    if (running) cost = Math.ceil(cost * 1.6);
    if (cost > 0 && player.ck < cost) {
      SH.ui.toast("Chakra épuisé — repose-toi (retourne au village ou sur une route).");
      moveTimer = 260;
      return;
    }
    player.ck -= cost;
    player.pos.x = nx; player.pos.y = ny;

    // Découverte du village secret.
    for (const v of world.villages) {
      if (v.hidden && !v.discovered && Math.abs(v.x - nx) + Math.abs(v.y - ny) <= 3) {
        v.discovered = true; player.discovered[v.id] = true;
        SH.ui.toast("✦ Tu as découvert le village caché de " + v.name + " !", 4200);
        SH.state.save(player);
      }
    }

    // Rencontre sauvage.
    if (encImmunity > 0) encImmunity--;
    else maybeEncounter(t, nx, ny);
  }

  function maybeEncounter(terr, x, y) {
    if (!terr.pool.length || terr.encounter <= 0) return;
    if (SH.world.villageAt(world, x, y)) return;
    if (!SH.rng.chance(terr.encounter)) return;

    const foeId = SH.rng.pick(terr.pool);
    let lvl = player.level + SH.rng.int(-1, 1);
    if (terr.id === data.TERR.MONTAGNE.id || terr.id === data.TERR.DESERT.id) lvl += 1;
    lvl = Math.max(1, lvl);
    startCombat(foeId, lvl);
  }

  // -------------------------------------------------------- Combat
  function startCombat(foeId, lvl) {
    inCombat = true;
    keys.length = 0;
    const c = SH.combat.start(player, foeId, lvl);
    SH.ui.openCombat(player, world, c, onCombatEnd);
  }

  function awardXp(amount) {
    const lv = SH.state.gainXp(player, amount);
    if (lv > 0) SH.ui.toast("↑ Niveau " + player.level + " ! Rang : " + data.rankForLevel(player.level) +
      ". +" + (lv * 3) + " points de caractéristique.", 4000);
  }

  function onCombatEnd(result) {
    inCombat = false;
    encImmunity = 4; // petite immunité après un combat
    if (result.outcome === "win") {
      player.ryo += result.ryo;
      let msg = "Victoire ! +" + result.xp + " XP, +" + result.ryo + " ₽.";
      // Butin éventuel.
      if (SH.rng.chance(0.22)) { SH.state.addConsumable(player, "potion_soin", 1); msg += " Butin : Potion de soin."; }
      // Progression de mission.
      if (player.mission && player.mission.type === "hunt" && player.mission.target === result.foeId) {
        player.missionProgress++;
        if (player.missionProgress >= player.mission.count) {
          const m = player.mission;
          player.ryo += m.ryo;
          awardXp(m.xp);
          player.missionsDone++;
          player.mission = null; player.missionProgress = 0;
          msg += " ★ Mission « " + m.title + " » accomplie ! +" + m.xp + " XP, +" + m.ryo + " ₽.";
        }
      }
      awardXp(result.xp);
      SH.ui.toast(msg, 4200);
    } else if (result.outcome === "lose") {
      const lost = Math.floor(player.ryo * 0.2);
      player.ryo -= lost;
      SH.state.healFull(player);
      placeAtHome();
      rx = player.pos.x; ry = player.pos.y;
      SH.ui.toast("Vous vous réveillez à " + data.VILLAGE_BY_ID[player.village].name +
        ". Vous perdez " + lost + " ₽.", 4200);
    } else {
      SH.ui.toast("Vous avez fui le combat.");
    }
    SH.state.clampVitals(player);
    SH.state.save(player);
    SH.ui.refreshHUD(player, world);
  }

  // -------------------------------------------------------- Interactions
  function interact() {
    if (paused || inCombat) return;
    const v = SH.world.villageAt(world, player.pos.x, player.pos.y);
    if (v) SH.ui.openVillage(player, world, v);
    else SH.ui.toast("Aucun village ici. Suis les routes pour en rejoindre un.");
  }

  // -------------------------------------------------------- Boucle
  function loop(now) {
    const dt = Math.min(0.05, (now - lastTime) / 1000);
    lastTime = now;

    if (!paused && !inCombat) {
      // Déplacement.
      const before = player.pos.x + "," + player.pos.y;
      tryStep(dt);
      const moving = keys.length > 0;

      // Régénération : rapide à l'arrêt, lente en mouvement.
      const d = SH.state.derive(player);
      const ckRate = moving ? 2.5 : 8;
      player.ck = Math.min(d.chakraMax, player.ck + ckRate * dt);
      if (!moving && player.hp < d.pvMax) player.hp = Math.min(d.pvMax, player.hp + 1.5 * dt);

      if (moving) walkPhase += dt * 10;
      SH.ui.refreshHUD(player, world);
    }

    // Interpolation douce vers la tuile courante.
    const ease = Math.min(1, dt * 12);
    rx += (player.pos.x - rx) * ease;
    ry += (player.pos.y - ry) * ease;
    if (Math.abs(player.pos.x - rx) < 0.01) rx = player.pos.x;
    if (Math.abs(player.pos.y - ry) < 0.01) ry = player.pos.y;

    if (ctx) SH.render.drawWorld(ctx, world, rx, ry, dir, keys.length ? walkPhase : 0);
    rafId = requestAnimationFrame(loop);
  }

  function setPaused(v) { paused = v; if (v) keys.length = 0; }

  // -------------------------------------------------------- Entrées clavier
  const keyState = { shift: false };
  const DIRKEYS = {
    ArrowDown: 0, s: 0, S: 0,
    ArrowLeft: 1, q: 1, Q: 1, a: 1, A: 1,
    ArrowRight: 2, d: 2, D: 2,
    ArrowUp: 3, z: 3, Z: 3, w: 3, W: 3,
  };

  function onKeyDown(e) {
    if (e.key === "Shift") { keyState.shift = true; return; }
    if (e.repeat) return;
    if ($("game-screen").classList.contains("hidden")) return;

    if (e.key === "Escape") { if (!inCombat) SH.ui.closeOverlay(); return; }
    if (paused || inCombat) return;

    if (e.key in DIRKEYS) {
      const dv = DIRKEYS[e.key];
      const i = keys.indexOf(dv);
      if (i !== -1) keys.splice(i, 1);
      keys.push(dv);
      e.preventDefault();
      return;
    }
    if (e.key === "e" || e.key === "E" || e.key === " ") { interact(); e.preventDefault(); }
    else if (e.key === "c" || e.key === "C") { SH.ui.openCharacter(player, world); }
  }

  function onKeyUp(e) {
    if (e.key === "Shift") { keyState.shift = false; return; }
    if (e.key in DIRKEYS) {
      const dv = DIRKEYS[e.key];
      const i = keys.indexOf(dv);
      if (i !== -1) keys.splice(i, 1);
    }
  }

  // -------------------------------------------------------- Init
  function init() {
    buildTitle();
    $("btn-start").onclick = startNewGame;
    $("btn-continue").onclick = continueGame;
    $("hero-name").addEventListener("keydown", (e) => { if (e.key === "Enter") startNewGame(); });

    $("btn-menu").onclick = interact;
    $("btn-char").onclick = () => SH.ui.openCharacter(player, world);
    $("btn-save").onclick = () => { if (player) { SH.state.save(player); SH.ui.toast("Partie sauvegardée."); } };

    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("keyup", onKeyUp);
    // Relâcher les touches si la fenêtre perd le focus (évite un déplacement fantôme).
    window.addEventListener("blur", () => { keys.length = 0; keyState.shift = false; });
  }

  SH.game = { init, setPaused };
  document.addEventListener("DOMContentLoaded", init);
})(window);
