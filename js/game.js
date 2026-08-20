/* game.js — contrôleur principal : connexion/inscription, création de
   personnage (nom + apparence + village), boucle de jeu, la dynamique de
   déplacement (grille + coût en chakra + régénération), les rencontres,
   l'entrée dans les villages et la sauvegarde. Exposé sous SH.game. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;
  const $ = (id) => document.getElementById(id);

  const WALK_MS = 150;   // délai entre deux pas

  const TILE = SH.render.TILE;
  let player = null, world = null, canvas = null, ctx = null;
  let paused = false, inCombat = false;
  let rx = 0, ry = 0, dir = 0, walkPhase = 0;
  let moveTimer = 0, encImmunity = 0;
  let lastTime = 0, rafId = 0;
  let path = [];             // file des pas restants (déplacement à la souris)
  let arrivalVillage = null; // village à ouvrir en arrivant dessus

  const NAMES = ["Kaito", "Ren", "Aiko", "Haru", "Sora", "Yuki", "Rin", "Taro", "Mika", "Jin"];

  // ============================================================ Connexion / inscription
  let authMode = "login";
  function buildAuth() {
    document.querySelectorAll("[data-auth-tab]").forEach((t) => {
      t.classList.toggle("active", t.dataset.authTab === authMode);
      t.onclick = () => { authMode = t.dataset.authTab; buildAuth(); $("auth-hint").textContent = ""; };
    });
    $("auth-pass2-field").classList.toggle("hidden", authMode !== "register");
    $("btn-auth").textContent = authMode === "register" ? "Créer le compte" : "Se connecter";
  }

  function submitAuth() {
    const user = $("auth-user").value, pass = $("auth-pass").value;
    const hint = $("auth-hint");
    if (authMode === "register") {
      if (pass !== $("auth-pass2").value) { hint.textContent = "Les mots de passe ne correspondent pas."; return; }
      const r = SH.state.register(user, pass);
      if (!r.ok) { hint.textContent = r.error; return; }
      goCreate();
    } else {
      const r = SH.state.login(user, pass);
      if (!r.ok) { hint.textContent = r.error; return; }
      if (r.hero) startGameWith(r.hero);
      else goCreate();
    }
  }

  // ============================================================ Création de personnage
  let selectedVillage = "zambakro";
  let createAppearance = data.defaultAppearance();

  function goCreate() {
    $("auth-screen").classList.add("hidden");
    $("create-screen").classList.remove("hidden");
    createAppearance = data.defaultAppearance();
    selectedVillage = "zambakro";
    $("hero-name").value = "";
    $("hero-name").placeholder = "ex. " + NAMES[Math.floor(Math.random() * NAMES.length)];
    $("create-hint").textContent = "";
    buildVillageChoice();
    SH.ui.appearanceEditor($("create-appearance"), createAppearance, null);
    $("hero-name").focus();
  }

  function buildVillageChoice() {
    const wrap = $("village-choice");
    wrap.innerHTML = data.VILLAGES.map((v) =>
      '<div class="village-card' + (v.id === selectedVillage ? " selected" : "") + '" data-v="' + v.id +
      '" style="--vc:' + v.color + '">' +
      '<div class="vc-badge" style="background:' + v.color + '">' + v.kanji + "</div>" +
      "<h3>" + v.name + "</h3>" +
      '<div class="vc-spec">' + v.specName + "<br>" + v.blurb + "</div></div>"
    ).join("");
    wrap.querySelectorAll(".village-card").forEach((c) => {
      c.onclick = () => { selectedVillage = c.dataset.v; buildVillageChoice(); };
    });
  }

  function submitCreate() {
    const name = ($("hero-name").value || "").trim() || NAMES[Math.floor(Math.random() * NAMES.length)];
    player = SH.state.createHero(name, selectedVillage, createAppearance);
    world = SH.world.generate(player.seed);
    placeAtHome();
    SH.state.save(player);
    enterGame(true);
  }

  function backToAuth() {
    SH.state.logout();
    $("create-screen").classList.add("hidden");
    $("auth-screen").classList.remove("hidden");
    $("auth-pass").value = "";
    buildAuth();
  }

  // ============================================================ Entrée en jeu
  function startGameWith(hero) {
    player = hero;
    player.appearance = data.normAppearance(player.appearance);
    world = SH.world.generate(player.seed);
    for (const v of world.villages) if (v.hidden && player.discovered && player.discovered[v.id]) v.discovered = true;
    if (!player.pos || !SH.world.inBounds(player.pos.x, player.pos.y)) placeAtHome();
    enterGame(false);
  }

  function placeAtHome() {
    const home = world.villages.find((v) => v.id === player.village) || world.villages[0];
    player.pos = { x: home.x, y: home.y };
  }

  function enterGame(isNew) {
    $("auth-screen").classList.add("hidden");
    $("create-screen").classList.add("hidden");
    $("game-screen").classList.remove("hidden");
    canvas = $("view");
    ctx = canvas.getContext("2d");
    SH.render.setHeroAppearance(player.appearance);
    rx = player.pos.x; ry = player.pos.y;
    stopMoving();
    SH.state.clampVitals(player);
    SH.ui.refreshHUD(player, world);
    lastTime = performance.now();
    cancelAnimationFrame(rafId);
    rafId = requestAnimationFrame(loop);
    SH.ui.toast((isNew ? "Bienvenue à " : "Bon retour à ") + data.VILLAGE_BY_ID[player.village].name +
      ". Explore le pays de Yuukan.", 4000);
  }

  // ============================================================ Déplacement (souris)
  function tileAt(x, y) { return world.tiles[SH.world.idx(x, y)]; }
  function dirFromDelta(dx, dy) { return dx > 0 ? 2 : dx < 0 ? 1 : dy < 0 ? 3 : 0; }

  // Convertit un clic sur le canvas en case, et lance un déplacement vers elle.
  function canvasClick(e) {
    if (paused || inCombat || !world) return;
    const rect = canvas.getBoundingClientRect();
    const sx = canvas.width / rect.width, sy = canvas.height / rect.height;
    const cam = SH.render.getCam();
    const px = (e.clientX - rect.left) * sx + cam.x;
    const py = (e.clientY - rect.top) * sy + cam.y;
    const tx = Math.floor(px / TILE), ty = Math.floor(py / TILE);
    if (!SH.world.inBounds(tx, ty)) return;

    // Clic sur sa propre case : entrer dans le village s'il y en a un.
    if (tx === player.pos.x && ty === player.pos.y) {
      const v = SH.world.villageAt(world, tx, ty);
      if (v) SH.ui.openVillage(player, world, v);
      return;
    }
    if (!data.TERR_BY_ID[tileAt(tx, ty)].walk) { SH.ui.toast("Zone infranchissable."); return; }

    const p = SH.world.findPath(world, player.pos, { x: tx, y: ty });
    if (!p || !p.length) { SH.ui.toast("Aucun chemin vers cet endroit."); return; }
    path = p;
    arrivalVillage = SH.world.villageAt(world, tx, ty);
    SH.render.setMoveTarget({ x: tx, y: ty });
  }

  // Avance d'un pas le long du chemin courant (appelé par la boucle).
  function followPath(dt) {
    moveTimer -= dt * 1000;
    if (!path.length || moveTimer > 0) return;

    const next = path[0];
    const t = data.TERR_BY_ID[tileAt(next.x, next.y)];
    if (!t.walk) { stopMoving(); return; }               // terrain devenu bloqué
    dir = dirFromDelta(next.x - player.pos.x, next.y - player.pos.y);

    const cost = t.chakra;
    if (cost > 0 && player.ck < cost) {
      SH.ui.toast("Chakra épuisé — repose-toi (retourne au village ou sur une route).");
      stopMoving(); moveTimer = 260; return;
    }
    player.ck -= cost;
    player.pos.x = next.x; player.pos.y = next.y;
    path.shift();
    moveTimer = WALK_MS;

    // Découverte du village secret.
    for (const v of world.villages) {
      if (v.hidden && !v.discovered && Math.abs(v.x - next.x) + Math.abs(v.y - next.y) <= 3) {
        v.discovered = true; player.discovered[v.id] = true;
        SH.ui.toast("✦ Tu as découvert le village caché de " + v.name + " !", 4200);
        SH.state.save(player);
      }
    }

    // Rencontre : interrompt le déplacement.
    if (encImmunity > 0) encImmunity--;
    else if (maybeEncounter(t, next.x, next.y)) { stopMoving(); return; }

    // Arrivée à destination.
    if (!path.length) {
      SH.render.setMoveTarget(null);
      if (arrivalVillage) { const v = arrivalVillage; arrivalVillage = null; SH.ui.openVillage(player, world, v); }
    }
  }

  function stopMoving() { path = []; arrivalVillage = null; SH.render.setMoveTarget(null); }

  // Déclenche un combat le cas échéant ; renvoie true si un combat a démarré.
  function maybeEncounter(terr, x, y) {
    if (!terr.pool.length || terr.encounter <= 0) return false;
    if (SH.world.villageAt(world, x, y)) return false;
    if (!SH.rng.chance(terr.encounter)) return false;

    const foeId = SH.rng.pick(terr.pool);
    let lvl = player.level + SH.rng.int(-1, 1);
    if (terr.id === data.TERR.MONTAGNE.id || terr.id === data.TERR.DESERT.id) lvl += 1;
    lvl = Math.max(1, lvl);
    startCombat(foeId, lvl);
    return true;
  }

  // ============================================================ Combat
  function startCombat(foeId, lvl) {
    inCombat = true;
    stopMoving();
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
    encImmunity = 4;
    if (result.outcome === "win") {
      player.ryo += result.ryo;
      let msg = "Victoire ! +" + result.xp + " XP, +" + result.ryo + " ₽.";
      if (SH.rng.chance(0.22)) { SH.state.addConsumable(player, "potion_soin", 1); msg += " Butin : Potion de soin."; }
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

  // ============================================================ Interactions
  function interact() {
    if (paused || inCombat) return;
    const v = SH.world.villageAt(world, player.pos.x, player.pos.y);
    if (v) SH.ui.openVillage(player, world, v);
    else SH.ui.toast("Aucun village ici. Suis les routes pour en rejoindre un.");
  }

  // ============================================================ Boucle
  function loop(now) {
    const dt = Math.min(0.05, (now - lastTime) / 1000);
    lastTime = now;

    if (!paused && !inCombat) {
      followPath(dt);
      const moving = path.length > 0;
      const d = SH.state.derive(player);
      const ckRate = moving ? 2.5 : 8;
      player.ck = Math.min(d.chakraMax, player.ck + ckRate * dt);
      if (!moving && player.hp < d.pvMax) player.hp = Math.min(d.pvMax, player.hp + 1.5 * dt);
      if (moving) walkPhase += dt * 10;
      SH.ui.refreshHUD(player, world);
    }

    const ease = Math.min(1, dt * 12);
    rx += (player.pos.x - rx) * ease;
    ry += (player.pos.y - ry) * ease;
    if (Math.abs(player.pos.x - rx) < 0.01) rx = player.pos.x;
    if (Math.abs(player.pos.y - ry) < 0.01) ry = player.pos.y;

    if (ctx) SH.render.drawWorld(ctx, world, rx, ry, dir, path.length ? walkPhase : 0);
    rafId = requestAnimationFrame(loop);
  }

  function setPaused(v) { paused = v; if (v) stopMoving(); }

  // ============================================================ Entrées clavier (raccourcis seulement)
  function onKeyDown(e) {
    if ($("game-screen").classList.contains("hidden")) return;
    if (e.key === "Escape") { if (!inCombat) SH.ui.closeOverlay(); return; }
    if (paused || inCombat) return;
    if (e.key === "e" || e.key === "E") interact();          // ouvrir le village (aussi au clic)
    else if (e.key === "c" || e.key === "C") SH.ui.openCharacter(player, world);
  }

  // ============================================================ Init
  function init() {
    buildAuth();
    $("btn-auth").onclick = submitAuth;
    $("auth-pass").addEventListener("keydown", (e) => { if (e.key === "Enter") submitAuth(); });
    $("auth-pass2").addEventListener("keydown", (e) => { if (e.key === "Enter") submitAuth(); });

    $("btn-create").onclick = submitCreate;
    $("btn-back").onclick = backToAuth;
    $("hero-name").addEventListener("keydown", (e) => { if (e.key === "Enter") submitCreate(); });

    $("btn-menu").onclick = interact;
    $("btn-char").onclick = () => SH.ui.openCharacter(player, world);
    $("btn-save").onclick = () => { if (player) { SH.state.save(player); SH.ui.toast("Partie sauvegardée."); } };

    // Déplacement uniquement à la souris : clic sur une case.
    $("view").addEventListener("click", canvasClick);

    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("blur", stopMoving);
  }

  SH.game = { init, setPaused, _pos: () => (player ? { x: player.pos.x, y: player.pos.y } : null) };
  document.addEventListener("DOMContentLoaded", init);
})(window);
