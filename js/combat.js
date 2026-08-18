/* combat.js — combat au tour par tour, façon shinobi.fr.
   Un seul combattant agit par tour ; l'ordre de départ dépend de la vitesse.
   Le joueur choisit : Taijutsu (gratuit) · Jutsu (coûte du chakra) · Objet ·
   Fuir. Le combat mute directement les PV/chakra du joueur (persistants), et
   renvoie un résultat que la couche jeu applique (XP, ryō, mission).
   Exposé sous SH.combat. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;
  const rng = SH.rng;

  function rand(a, b) { return a + rng.f() * (b - a); }

  // Dégâts = (puissance - défense atténuée) × variance, minimum 1.
  function damage(power, def) {
    return Math.max(1, Math.round((power - def * 0.9) * rand(0.85, 1.15)));
  }

  // Construit un ennemi à partir d'un modèle de monstre, mis à l'échelle du niveau.
  function makeFoe(foeId, level) {
    const m = data.MONSTERS[foeId];
    const k = level - 1;
    const hpMax = Math.round(m.hp * (1 + k * 0.35));
    const ckMax = 30 + level * 4;
    return {
      id: m.id, name: m.name, palette: m.palette,
      level,
      hpMax, hp: hpMax, ckMax, ck: ckMax,
      atk: Math.round(m.atk + k * 1.3),
      def: Math.round(m.def + k * 0.7),
      spd: Math.round(m.spd + k * 0.5),
      jutsu: m.jutsu,
      xp: Math.round(m.xp * (1 + k * 0.5)),
      ryo: Math.round(m.ryo * (1 + k * 0.4)),
    };
  }

  function start(p, foeId, level) {
    const foe = makeFoe(foeId, level);
    const c = {
      p, foe,
      log: [],
      turn: null,          // "hero" | "foe"
      over: false,
      result: null,        // { outcome, ... }
      heroStun: false,
      foeStun: false,
    };

    function push(text, cls) { c.log.push({ text, cls: cls || "" }); }
    function d() { return SH.state.derive(p); }

    push(foe.name + " (niv. " + level + ") surgit !", "info");

    // Ordre de départ selon la vitesse.
    c.turn = d().vit >= foe.spd ? "hero" : "foe";
    if (c.turn === "foe") { push(foe.name + " est plus rapide et attaque en premier.", "info"); foeTurn(); }

    // ---- Tour de l'ennemi ----
    function foeTurn() {
      if (c.over) return;
      if (c.foeStun) { c.foeStun = false; push(foe.name + " est paralysé et ne peut agir !", "info"); c.turn = "hero"; return; }

      const ds = d();
      let power, label;
      if (foe.jutsu && foe.ck >= (data.JUTSUS[foe.jutsu] ? data.JUTSUS[foe.jutsu].cost : 999) && rng.chance(0.4)) {
        const j = data.JUTSUS[foe.jutsu];
        foe.ck -= j.cost;
        power = j.power + foe.atk * 1.2;
        label = j.name;
      } else {
        power = foe.atk * 1.6 + 4;
        label = "attaque";
      }
      const dmg = damage(power, ds.def + p.base.tai * 0.15);
      p.hp -= dmg;
      push(foe.name + " utilise " + label + " : −" + dmg + " PV.", "hit");
      if (p.hp <= 0) { p.hp = 0; return lose(); }
      c.turn = "hero";
    }

    // Enchaîne le tour ennemi puis rend la main au joueur (après une action du héros).
    function afterHero() {
      if (c.over) return;
      c.turn = "foe";
      foeTurn();
    }

    // ---- Actions du joueur ----
    function heroAttack() {
      if (!guard()) return;
      const ds = d();
      const dmg = damage(ds.tai * 1.5 + 5, foe.def);
      foe.hp -= dmg;
      push("Vous frappez : −" + dmg + " PV.", "hit");
      if (foe.hp <= 0) return win();
      afterHero();
    }

    function heroJutsu(jutsuId) {
      if (!guard()) return;
      const j = data.JUTSUS[jutsuId];
      if (!j || !p.jutsus.includes(jutsuId)) return "Jutsu inconnu.";
      if (p.ck < j.cost) return "Chakra insuffisant.";
      p.ck -= j.cost;
      const ds = d();
      const scale = ds[j.scale] || 0;

      if (j.effect === "heal") {
        const heal = Math.round(scale * 0.8 + j.power);
        const before = p.hp;
        p.hp = Math.min(ds.pvMax, p.hp + heal);
        push(j.name + " : +" + (p.hp - before) + " PV.", "heal");
        return afterHero(), null;
      }

      const dmg = damage(scale * 1.4 + j.power, foe.def);
      foe.hp -= dmg;
      let extra = "";
      if (j.effect === "stun" && rng.chance(0.5)) { c.foeStun = true; extra = " L'ennemi est paralysé !"; }
      if (j.effect === "debuff_def") { foe.def = Math.max(0, Math.round(foe.def * 0.7)); extra = " Défense ennemie réduite !"; }
      if (j.effect === "buff_spd") { p.base.__spdBuff = true; extra = " Votre vitesse s'envole !"; }
      push(j.name + " : −" + dmg + " PV." + extra, "hit");
      if (foe.hp <= 0) return win();
      afterHero();
      return null;
    }

    function heroItem(itemId) {
      if (!guard()) return;
      const res = SH.state.useConsumable(p, itemId);
      if (!res) return "Objet indisponible.";
      push(res, "heal");
      afterHero();
      return null;
    }

    function heroFlee() {
      if (!guard()) return;
      const ds = d();
      const chance = Math.min(0.9, 0.35 + (ds.vit - foe.spd) * 0.04);
      if (rng.chance(chance)) {
        push("Vous prenez la fuite !", "info");
        c.over = true;
        c.result = { outcome: "flee" };
        return;
      }
      push("Fuite ratée !", "info");
      afterHero();
    }

    function guard() {
      if (c.over || c.turn !== "hero") return false;
      return true;
    }

    function win() {
      c.over = true;
      SH.state.clampVitals(p);
      c.result = { outcome: "win", foeId: foe.id, foeName: foe.name, xp: foe.xp, ryo: foe.ryo, level: foe.level };
      push("★ " + foe.name + " est vaincu !", "heal");
    }
    function lose() {
      c.over = true;
      c.result = { outcome: "lose" };
      push("✖ Vous êtes vaincu…", "hit");
    }

    Object.assign(c, { heroAttack, heroJutsu, heroItem, heroFlee });
    return c;
  }

  SH.combat = { start, makeFoe };
})(window);
