/* ui.js — interface HTML : HUD (barres, stats), menus de village
   (repos, entraînement, boutique, jutsus, missions), fiche du ninja, et la
   fenêtre de combat interactive. Exposé sous SH.ui. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;
  const $ = (id) => document.getElementById(id);

  // Prix / niveau requis d'un jutsu, dérivés de sa puissance.
  function jutsuPrice(j) { return 50 + j.power * 10; }
  function jutsuLevelReq(j) { return Math.max(1, Math.floor(j.power / 6)); }

  function esc(s) { return String(s).replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c])); }

  // ---------------------------------------------------------------- HUD
  function refreshHUD(p, world) {
    const d = SH.state.derive(p);
    $("hud-hero").textContent = p.name;
    const v = data.VILLAGE_BY_ID[p.village];
    $("hud-village").textContent = "村 " + v.name;
    $("hud-level").textContent = p.level;
    $("hud-rank").textContent = data.rankForLevel(p.level);

    const pct = (a, b) => Math.max(0, Math.min(100, (a / b) * 100)) + "%";
    $("bar-hp").style.width = pct(p.hp, d.pvMax);
    $("bar-ck").style.width = pct(p.ck, d.chakraMax);
    const xpNeed = data.xpForLevel(p.level);
    $("bar-xp").style.width = pct(p.xp, xpNeed);
    $("txt-hp").textContent = Math.ceil(p.hp) + " / " + d.pvMax;
    $("txt-ck").textContent = Math.ceil(p.ck) + " / " + d.chakraMax;
    $("txt-xp").textContent = p.xp + " / " + xpNeed;

    $("st-tai").textContent = d.tai;
    $("st-nin").textContent = d.nin;
    $("st-gen").textContent = d.gen;
    $("st-spd").textContent = d.vit;
    $("st-ryo").textContent = p.ryo;
    $("st-pts").textContent = p.statPoints;

    if (world) {
      const t = data.TERR_BY_ID[world.tiles[SH.world.idx(p.pos.x, p.pos.y)]];
      const vAt = SH.world.villageAt(world, p.pos.x, p.pos.y);
      let info = "Zone : <b>" + t.name + "</b>";
      if (vAt) info = "Village de <b>" + vAt.name + "</b> — appuie sur <b>E</b> pour entrer.";
      else if (p.mission) info += " · Mission : " + esc(p.mission.title) + " (" + p.missionProgress + "/" + p.mission.count + ")";
      $("loc-info").innerHTML = info;
    }
  }

  // ---------------------------------------------------------------- Toast
  let toastTimer = null;
  function toast(msg, ms) {
    const el = $("toast");
    el.innerHTML = msg;
    el.classList.remove("hidden");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.classList.add("hidden"), ms || 2600);
  }

  // ---------------------------------------------------------------- Overlay
  function overlay() { return $("overlay"); }
  function closeOverlay() {
    const o = overlay();
    o.classList.add("hidden");
    o.innerHTML = "";
    if (SH.game) SH.game.setPaused(false);
  }
  function openModal(html, opts) {
    opts = opts || {};
    const o = overlay();
    o.innerHTML = '<div class="modal">' + html + "</div>";
    o.classList.remove("hidden");
    if (SH.game) SH.game.setPaused(true);
    if (!opts.locked) {
      o.onclick = (e) => { if (e.target === o) closeOverlay(); };
    } else {
      o.onclick = null;
    }
    return o.querySelector(".modal");
  }

  // ---------------------------------------------------------------- Village
  let villageTab = "repos";
  function openVillage(p, world, village) {
    const v = data.VILLAGE_BY_ID[village.id] || village;
    const home = v.id === p.village;
    const tabs = [
      ["repos", "Repos"], ["train", "Entraînement"], ["shop", "Boutique"],
      ["jutsu", "Jutsus"], ["mission", "Missions"],
    ];
    const tabBtns = tabs.map(([k, label]) =>
      '<div class="tab ' + (villageTab === k ? "active" : "") + '" data-tab="' + k + '">' + label + "</div>"
    ).join("");

    const modal = openModal(
      '<button class="btn btn-small modal-close" data-close>Fermer ✕</button>' +
      "<h2>" + v.kanji + " Village de " + v.name + "</h2>" +
      '<p class="modal-sub">' + esc(v.blurb) + (home ? " — <b>votre village</b>." : "") + "</p>" +
      '<div class="tabs">' + tabBtns + "</div>" +
      '<div id="village-body"></div>'
    );

    modal.querySelector("[data-close]").onclick = closeOverlay;
    modal.querySelectorAll(".tab").forEach((t) => {
      t.onclick = () => { villageTab = t.dataset.tab; renderBody(); };
    });

    function renderBody() {
      modal.querySelectorAll(".tab").forEach((t) => t.classList.toggle("active", t.dataset.tab === villageTab));
      const body = modal.querySelector("#village-body");
      if (villageTab === "repos") body.innerHTML = repos(p);
      else if (villageTab === "train") body.innerHTML = train(p);
      else if (villageTab === "shop") body.innerHTML = shop(p);
      else if (villageTab === "jutsu") body.innerHTML = jutsuShop(p, v);
      else if (villageTab === "mission") body.innerHTML = missions(p, v);
      bind(body);
    }

    function bind(body) {
      body.querySelectorAll("[data-act]").forEach((btn) => {
        btn.onclick = () => {
          const act = btn.dataset.act, id = btn.dataset.id;
          if (act === "rest") { SH.state.healFull(p); toast("Vous êtes reposé. PV et chakra au maximum."); }
          else if (act === "train") { if (!SH.state.trainStat(p, id)) toast("Aucun point disponible."); }
          else if (act === "buy") { const err = SH.state.buy(p, data.ITEMS[id]); toast(err || "Acheté : " + data.ITEMS[id].name + "."); }
          else if (act === "equip") { SH.state.equipItem(p, id); toast(data.ITEMS[id].name + " équipé."); }
          else if (act === "learn") { const j = data.JUTSUS[id]; const err = SH.state.learnJutsu(p, id, jutsuPrice(j)); toast(err || "Jutsu appris : " + j.name + " !"); }
          else if (act === "accept") { acceptMission(p, v, id); }
          else if (act === "abandon") { p.mission = null; p.missionProgress = 0; toast("Mission abandonnée."); }
          SH.state.save(p);
          refreshHUD(p, world);
          renderBody();
        };
      });
    }

    renderBody();
  }

  function repos(p) {
    const d = SH.state.derive(p);
    return '<p class="modal-sub">Un moment de calme au village restaure tout votre corps.</p>' +
      '<div class="row"><div class="row-main"><div class="row-title">Se reposer</div>' +
      '<div class="row-desc">PV ' + Math.ceil(p.hp) + "/" + d.pvMax + " · Chakra " + Math.ceil(p.ck) + "/" + d.chakraMax + "</div></div>" +
      '<button class="btn btn-primary" data-act="rest">Se reposer</button></div>';
  }

  function train(p) {
    const rows = [
      ["tai", "Taijutsu", "Dégâts au corps à corps."],
      ["nin", "Ninjutsu", "Puissance des jutsus offensifs et soins."],
      ["gen", "Genjutsu", "Puissance des illusions."],
      ["vit", "Vitesse", "Initiative au combat et fuite."],
    ].map(([k, name, desc]) =>
      '<div class="row"><div class="row-main"><div class="row-title">' + name + " — <b>" + p.base[k] + "</b></div>" +
      '<div class="row-desc">' + desc + "</div></div>" +
      '<button class="btn plus" data-act="train" data-id="' + k + '"' + (p.statPoints > 0 ? "" : " disabled") + ">+</button></div>"
    ).join("");
    return '<p class="modal-sub">Points de caractéristique disponibles : <b>' + p.statPoints +
      "</b> (gagnés en montant de niveau).</p>" + '<div class="list">' + rows + "</div>";
  }

  function shop(p) {
    const rows = data.ITEMS_LIST.map((it) => {
      let right;
      if (it.slot === "conso") {
        right = '<button class="btn" data-act="buy" data-id="' + it.id + '"' +
          (p.ryo >= it.price && p.level >= it.levelReq ? "" : " disabled") + ">Acheter</button>";
      } else if (p.owned.includes(it.id)) {
        const equipped = p.equip[it.slot] === it.id;
        right = equipped ? '<span class="tag">Équipé</span>'
          : '<button class="btn" data-act="equip" data-id="' + it.id + '">Équiper</button>';
      } else {
        right = '<button class="btn" data-act="buy" data-id="' + it.id + '"' +
          (p.ryo >= it.price && p.level >= it.levelReq ? "" : " disabled") + ">Acheter</button>";
      }
      const own = p.inv[it.id] ? " ×" + p.inv[it.id] : "";
      return '<div class="row"><div class="row-main"><div class="row-title">' + esc(it.name) + own +
        (it.levelReq > 1 ? ' <span class="tag">niv. ' + it.levelReq + "</span>" : "") + "</div>" +
        '<div class="row-desc">' + esc(it.desc) + "</div></div>" +
        '<span class="price">' + it.price + " ₽</span>" + right + "</div>";
    }).join("");
    return '<p class="modal-sub">Ryō : <b>' + p.ryo + " ₽</b></p>" + '<div class="shop-list">' + rows + "</div>";
  }

  function jutsuShop(p, v) {
    const rows = v.teaches.map((jid) => {
      const j = data.JUTSUS[jid];
      const known = p.jutsus.includes(jid);
      const price = jutsuPrice(j), req = jutsuLevelReq(j);
      const can = !known && p.ryo >= price && p.level >= req;
      const right = known ? '<span class="tag">Connu</span>'
        : '<button class="btn" data-act="learn" data-id="' + jid + '"' + (can ? "" : " disabled") + ">Apprendre</button>";
      return '<div class="row"><div class="row-main"><div class="row-title">' + esc(j.name) +
        ' <span class="tag ' + j.type + '">' + j.type + "</span>" +
        (req > 1 ? ' <span class="tag">niv. ' + req + "</span>" : "") + "</div>" +
        '<div class="row-desc">' + esc(j.desc) + " · coût " + j.cost + " chakra</div></div>" +
        (known ? "" : '<span class="price">' + price + " ₽</span>") + right + "</div>";
    }).join("");
    return '<p class="modal-sub">Les jutsus enseignés à ' + v.name + " (spécialité : " + v.specName + ").</p>" +
      '<div class="list">' + rows + "</div>";
  }

  function missions(p, v) {
    if (p.mission) {
      return '<p class="modal-sub">Mission en cours pour ' + v.name + ".</p>" +
        '<div class="row"><div class="row-main"><div class="row-title">' + esc(p.mission.title) +
        ' <span class="tag">rang ' + p.mission.rank + "</span></div>" +
        '<div class="row-desc">' + esc(p.mission.desc) + "<br>Progression : <b>" +
        p.missionProgress + " / " + p.mission.count + "</b> · Récompense : " + p.mission.xp + " XP, " + p.mission.ryo + " ₽</div></div>" +
        '<button class="btn" data-act="abandon">Abandonner</button></div>';
    }
    const avail = data.MISSION_TEMPLATES.filter((m) => p.level >= m.minLevel - 1).slice(0, 5);
    const rows = avail.map((m, i) =>
      '<div class="row"><div class="row-main"><div class="row-title">' + esc(m.title) +
      ' <span class="tag">rang ' + m.rank + "</span>" +
      (p.level < m.minLevel ? ' <span class="tag">niv. ' + m.minLevel + "</span>" : "") + "</div>" +
      '<div class="row-desc">' + esc(m.desc) + " · Récompense : " + m.xp + " XP, " + m.ryo + " ₽</div></div>" +
      '<button class="btn" data-act="accept" data-id="' + i + '"' + (p.level >= m.minLevel ? "" : " disabled") + ">Accepter</button></div>"
    ).join("");
    return '<p class="modal-sub">Missions disponibles au bureau de ' + v.name + ".</p>" +
      '<div class="list">' + rows + "</div>";
  }

  function acceptMission(p, v, idx) {
    const avail = data.MISSION_TEMPLATES.filter((m) => p.level >= m.minLevel - 1).slice(0, 5);
    const m = avail[+idx];
    if (!m) return;
    if (p.level < m.minLevel) { toast("Niveau requis : " + m.minLevel + "."); return; }
    p.mission = { title: m.title, desc: m.desc, type: m.type, target: m.target, count: m.count, xp: m.xp, ryo: m.ryo, rank: m.rank };
    p.missionProgress = 0;
    toast("Mission acceptée : " + m.title + ".");
  }

  // ---------------------------------------------------------------- Fiche ninja
  function openCharacter(p, world) {
    const d = SH.state.derive(p);
    const v = data.VILLAGE_BY_ID[p.village];
    const jutsuRows = p.jutsus.map((jid) => {
      const j = data.JUTSUS[jid];
      return '<div class="row"><div class="row-main"><div class="row-title">' + esc(j.name) +
        ' <span class="tag ' + j.type + '">' + j.type + "</span></div>" +
        '<div class="row-desc">' + esc(j.desc) + " · coût " + j.cost + " chakra</div></div></div>";
    }).join("") || '<p class="empty">Aucun jutsu appris.</p>';

    const invIds = Object.keys(p.inv);
    const invRows = invIds.length ? invIds.map((id) => {
      const it = data.ITEMS[id];
      return '<div class="row"><div class="row-main"><div class="row-title">' + esc(it.name) + " ×" + p.inv[id] + "</div>" +
        '<div class="row-desc">' + esc(it.desc) + "</div></div>" +
        '<button class="btn" data-use="' + id + '">Utiliser</button></div>';
    }).join("") : '<p class="empty">Sac vide.</p>';

    const slots = ["arme", "armure", "bandeau"].map((s) => {
      const id = p.equip[s];
      const label = { arme: "Arme", armure: "Armure", bandeau: "Bandeau" }[s];
      return '<div class="row"><div class="row-main"><div class="row-title">' + label + "</div>" +
        '<div class="row-desc">' + (id ? esc(data.ITEMS[id].name) : "— vide —") + "</div></div></div>";
    }).join("");

    const modal = openModal(
      '<button class="btn btn-small modal-close" data-close>Fermer ✕</button>' +
      "<h2>" + esc(p.name) + "</h2>" +
      '<p class="modal-sub">' + v.kanji + " " + v.name + " · " + data.rankForLevel(p.level) + " · niveau " + p.level + "</p>" +
      '<div class="list" style="margin-bottom:14px">' +
      '<div class="row"><div class="row-main">PV max <b>' + d.pvMax + "</b> · Chakra max <b>" + d.chakraMax + "</b> · Défense <b>" + d.def + "</b></div></div>" +
      '<div class="row"><div class="row-main">Taijutsu <b>' + d.tai + "</b> · Ninjutsu <b>" + d.nin + "</b> · Genjutsu <b>" + d.gen + "</b> · Vitesse <b>" + d.vit + "</b></div></div>" +
      "</div>" +
      "<h3>Équipement</h3><div class=\"list\">" + slots + "</div>" +
      '<h3 style="margin-top:14px">Sac</h3><div class="list">' + invRows + "</div>" +
      '<h3 style="margin-top:14px">Jutsus</h3><div class="list">' + jutsuRows + "</div>"
    );
    modal.querySelector("[data-close]").onclick = closeOverlay;
    modal.querySelectorAll("[data-use]").forEach((b) => {
      b.onclick = () => {
        const res = SH.state.useConsumable(p, b.dataset.use);
        if (res) { toast(res); SH.state.save(p); refreshHUD(p, world); closeOverlay(); openCharacter(p, world); }
      };
    });
  }

  // ---------------------------------------------------------------- Combat
  function heroPalette(p) {
    const v = data.VILLAGE_BY_ID[p.village];
    return ["#2b3040", "#20242e", v.color];
  }

  function openCombat(p, world, combat, onDone) {
    const modal = openModal(
      '<div class="combat">' +
      "<h2>Combat</h2>" +
      '<div class="combat-stage">' +
      '<div class="fighter hero"><canvas width="96" height="96"></canvas><div class="fname">' + esc(p.name) + "</div>" +
      '<div class="f-bar"><div class="f-hp"></div></div><div class="f-bar"><div class="f-ck"></div></div></div>' +
      '<div class="fighter foe"><canvas width="96" height="96"></canvas><div class="fname" id="foe-name"></div>' +
      '<div class="f-bar"><div class="f-hp"></div></div></div>' +
      "</div>" +
      '<div class="combat-log" id="clog"></div>' +
      '<div id="combat-controls"></div>' +
      "</div>",
      { locked: true }
    );

    const heroCanvas = modal.querySelectorAll(".fighter.hero canvas")[0];
    const foeCanvas = modal.querySelectorAll(".fighter.foe canvas")[0];
    SH.render.drawFighter(heroCanvas, heroPalette(p), "right");
    SH.render.drawFighter(foeCanvas, combat.foe.palette, "left");
    modal.querySelector("#foe-name").textContent = combat.foe.name + " (niv. " + combat.foe.level + ")";

    function renderBars() {
      const d = SH.state.derive(p);
      modal.querySelector(".fighter.hero .f-hp").style.width = Math.max(0, (p.hp / d.pvMax) * 100) + "%";
      modal.querySelector(".fighter.hero .f-ck").style.width = Math.max(0, (p.ck / d.chakraMax) * 100) + "%";
      modal.querySelector(".fighter.foe .f-hp").style.width = Math.max(0, (combat.foe.hp / combat.foe.hpMax) * 100) + "%";
    }
    function renderLog() {
      const box = modal.querySelector("#clog");
      box.innerHTML = combat.log.map((l) => '<div class="' + l.cls + '">' + esc(l.text) + "</div>").join("");
      box.scrollTop = box.scrollHeight;
    }

    function renderControls() {
      const ctr = modal.querySelector("#combat-controls");
      if (combat.over) {
        ctr.innerHTML = '<button class="btn btn-primary" id="c-continue" style="width:100%;padding:12px">Continuer</button>';
        ctr.querySelector("#c-continue").onclick = () => { closeOverlay(); onDone(combat.result); };
        return;
      }
      ctr.innerHTML =
        '<div class="combat-actions">' +
        '<button class="btn" data-c="attack">Taijutsu</button>' +
        '<button class="btn" data-c="jutsu">Jutsu</button>' +
        '<button class="btn" data-c="item">Objet</button>' +
        '<button class="btn" data-c="flee">Fuir</button>' +
        "</div><div id=\"sub\"></div>";
      ctr.querySelector('[data-c="attack"]').onclick = () => { combat.heroAttack(); step(); };
      ctr.querySelector('[data-c="jutsu"]').onclick = () => showJutsu();
      ctr.querySelector('[data-c="item"]').onclick = () => showItems();
      ctr.querySelector('[data-c="flee"]').onclick = () => { combat.heroFlee(); step(); };
    }

    function showJutsu() {
      const sub = modal.querySelector("#sub");
      if (!p.jutsus.length) { sub.innerHTML = '<p class="empty">Aucun jutsu.</p>'; return; }
      sub.innerHTML = '<div class="jutsu-picker">' + p.jutsus.map((jid) => {
        const j = data.JUTSUS[jid];
        const can = p.ck >= j.cost;
        return '<button class="btn" data-j="' + jid + '"' + (can ? "" : " disabled") + ">" +
          esc(j.name) + " <span class='tag " + j.type + "'>" + j.cost + " ck</span></button>";
      }).join("") + "</div>";
      sub.querySelectorAll("[data-j]").forEach((b) => {
        b.onclick = () => { const err = combat.heroJutsu(b.dataset.j); if (err) toast(err); else step(); };
      });
    }

    function showItems() {
      const sub = modal.querySelector("#sub");
      const ids = Object.keys(p.inv).filter((id) => data.ITEMS[id].slot === "conso");
      if (!ids.length) { sub.innerHTML = '<p class="empty">Aucun objet.</p>'; return; }
      sub.innerHTML = '<div class="jutsu-picker">' + ids.map((id) =>
        '<button class="btn" data-i="' + id + '">' + esc(data.ITEMS[id].name) + " ×" + p.inv[id] + "</button>"
      ).join("") + "</div>";
      sub.querySelectorAll("[data-i]").forEach((b) => {
        b.onclick = () => { const err = combat.heroItem(b.dataset.i); if (err) toast(err); else step(); };
      });
    }

    function step() {
      SH.state.clampVitals(p);
      renderBars(); renderLog(); renderControls();
      refreshHUD(p, world);
    }

    renderBars(); renderLog(); renderControls();
  }

  SH.ui = { refreshHUD, toast, openVillage, openCharacter, openCombat, closeOverlay, jutsuPrice };
})(window);
