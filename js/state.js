/* state.js — état du joueur : stats dérivées, progression (XP/niveau),
   inventaire, équipement, jutsus, missions, et sauvegarde localStorage.
   Exposé sous SH.state. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});
  const data = SH.data;

  // Crée un héros neuf pour un village donné (avec apparence personnalisée).
  function createHero(name, villageId, appearance) {
    const v = data.VILLAGE_BY_ID[villageId];
    const base = {
      tai: 2 + (v.start.tai || 0),
      nin: 2 + (v.start.nin || 0),
      gen: 2 + (v.start.gen || 0),
      vit: 5,
    };
    const p = {
      name: (name || "Ninja").trim().slice(0, 14) || "Ninja",
      village: villageId,
      appearance: data.normAppearance(appearance),
      level: 1, xp: 0, statPoints: 0,
      base,
      pvMaxBase: 60, ckMaxBase: 40,
      hp: 60, ck: 40,
      ryo: 80,
      equip: { arme: null, armure: null, bandeau: null },
      inv: { potion_soin: 2 },           // consommables : id -> quantité
      owned: [],                          // équipements possédés (ids)
      jutsus: [v.teaches[0]],             // jutsu de départ du village
      mission: null,                      // mission active
      missionProgress: 0,
      missionsDone: 0,
      pos: { x: 0, y: 0 },                // fixé après génération de la carte
      seed: (Math.random() * 2 ** 32) >>> 0,
      discovered: {},                     // id village secret -> true
      playtime: 0,
    };
    return p;
  }

  // Somme des bonus d'équipement pour une clé donnée.
  function equipBonus(p, key) {
    let sum = 0;
    for (const slot of ["arme", "armure", "bandeau"]) {
      const id = p.equip[slot];
      if (id && data.ITEMS[id] && data.ITEMS[id].bonus && data.ITEMS[id].bonus[key]) {
        sum += data.ITEMS[id].bonus[key];
      }
    }
    return sum;
  }

  // Statistiques effectives = base + équipement + échelle de niveau.
  function derive(p) {
    return {
      tai: p.base.tai + equipBonus(p, "tai"),
      nin: p.base.nin + equipBonus(p, "nin"),
      gen: p.base.gen + equipBonus(p, "gen"),
      vit: p.base.vit + equipBonus(p, "spd"),
      def: Math.floor(p.level * 0.5) + equipBonus(p, "def"),
      pvMax: p.pvMaxBase + equipBonus(p, "pvMax"),
      chakraMax: p.ckMaxBase + equipBonus(p, "chakraMax"),
    };
  }

  function clampVitals(p) {
    const d = derive(p);
    if (p.hp > d.pvMax) p.hp = d.pvMax;
    if (p.ck > d.chakraMax) p.ck = d.chakraMax;
    if (p.hp < 0) p.hp = 0;
    if (p.ck < 0) p.ck = 0;
  }

  function healFull(p) {
    const d = derive(p);
    p.hp = d.pvMax; p.ck = d.chakraMax;
  }

  // Ajoute de l'XP ; gère les montées de niveau. Renvoie le nombre de niveaux gagnés.
  function gainXp(p, amount) {
    p.xp += amount;
    let levels = 0;
    while (p.xp >= data.xpForLevel(p.level)) {
      p.xp -= data.xpForLevel(p.level);
      p.level++;
      p.statPoints += 3;
      p.pvMaxBase += 12;
      p.ckMaxBase += 6;
      levels++;
    }
    if (levels > 0) healFull(p); // remise en forme à chaque niveau
    return levels;
  }

  // Dépense un point de caractéristique dans une stat de base.
  function trainStat(p, key) {
    if (p.statPoints <= 0) return false;
    if (!["tai", "nin", "gen", "vit"].includes(key)) return false;
    p.base[key]++;
    p.statPoints--;
    return true;
  }

  // ---- Inventaire / boutique ----
  function addConsumable(p, id, n) { p.inv[id] = (p.inv[id] || 0) + (n || 1); }

  function buy(p, item) {
    if (p.ryo < item.price) return "Pas assez de ryō.";
    if (p.level < item.levelReq) return "Niveau requis : " + item.levelReq + ".";
    if (item.slot === "conso") {
      p.ryo -= item.price; addConsumable(p, item.id, 1); return null;
    }
    if (p.owned.includes(item.id)) return "Déjà possédé.";
    p.ryo -= item.price; p.owned.push(item.id); return null;
  }

  // Équipe un objet possédé dans son emplacement.
  function equipItem(p, id) {
    const it = data.ITEMS[id];
    if (!it || it.slot === "conso" || !p.owned.includes(id)) return false;
    p.equip[it.slot] = id;
    clampVitals(p);
    return true;
  }

  // Consomme une potion. Renvoie un texte de résultat, ou null si impossible.
  function useConsumable(p, id) {
    if (!(p.inv[id] > 0)) return null;
    const it = data.ITEMS[id];
    const d = derive(p);
    let msg = [];
    if (it.use.hp) { const b = Math.min(it.use.hp, d.pvMax - p.hp); p.hp += b; if (b > 0) msg.push("+" + b + " PV"); }
    if (it.use.ck) { const b = Math.min(it.use.ck, d.chakraMax - p.ck); p.ck += b; if (b > 0) msg.push("+" + b + " chakra"); }
    if (msg.length === 0) return "Aucun effet (déjà au max).";
    p.inv[id]--;
    if (p.inv[id] <= 0) delete p.inv[id];
    return it.name + " : " + msg.join(", ") + ".";
  }

  function learnJutsu(p, jutsuId, price) {
    if (p.jutsus.includes(jutsuId)) return "Jutsu déjà connu.";
    if (p.ryo < price) return "Pas assez de ryō.";
    p.ryo -= price; p.jutsus.push(jutsuId); return null;
  }

  // ---- Comptes locaux (connexion / inscription) ----
  // Stockage 100 % local (localStorage). Ce n'est pas une sécurité réseau : le
  // mot de passe est juste haché pour ne pas être stocké en clair.
  const ACCOUNTS_KEY = "shinobi_yuukan_accounts_v1";
  const session = { user: null };

  function hash(s) {                       // djb2 -> hex (léger, non cryptographique)
    let h = 5381;
    for (let i = 0; i < s.length; i++) h = ((h << 5) + h + s.charCodeAt(i)) >>> 0;
    return h.toString(16);
  }
  function accountsAll() {
    try { return JSON.parse(localStorage.getItem(ACCOUNTS_KEY) || "{}"); } catch (e) { return {}; }
  }
  function accountsPersist(obj) {
    try { localStorage.setItem(ACCOUNTS_KEY, JSON.stringify(obj)); return true; } catch (e) { return false; }
  }
  function normUser(u) { return (u || "").trim().toLowerCase(); }

  function register(user, pass) {
    const u = normUser(user);
    if (u.length < 3) return { ok: false, error: "Identifiant : au moins 3 caractères." };
    if ((pass || "").length < 4) return { ok: false, error: "Mot de passe : au moins 4 caractères." };
    const all = accountsAll();
    if (all[u]) return { ok: false, error: "Cet identifiant existe déjà." };
    all[u] = { pass: hash(pass), display: user.trim(), hero: null, createdAt: Date.now() };
    accountsPersist(all);
    session.user = u;
    return { ok: true };
  }
  function login(user, pass) {
    const u = normUser(user);
    const all = accountsAll();
    const acc = all[u];
    if (!acc || acc.pass !== hash(pass)) return { ok: false, error: "Identifiant ou mot de passe incorrect." };
    session.user = u;
    return { ok: true, hero: acc.hero };
  }
  function logout() { session.user = null; }
  function currentHero() {
    if (!session.user) return null;
    const acc = accountsAll()[session.user];
    return acc ? acc.hero : null;
  }

  // ---- Sauvegarde (dans le compte connecté) ----
  function save(p) {
    if (!session.user) return false;
    try {
      const all = accountsAll();
      if (!all[session.user]) return false;
      p.savedAt = Date.now();
      all[session.user].hero = p;
      return accountsPersist(all);
    } catch (e) { return false; }
  }
  function load() { return currentHero(); }
  function hasSave() { return !!currentHero(); }
  function wipe() {                          // efface le héros du compte (repartir de zéro)
    if (!session.user) return;
    const all = accountsAll();
    if (all[session.user]) { all[session.user].hero = null; accountsPersist(all); }
  }

  SH.session = session;
  SH.state = {
    createHero, derive, clampVitals, healFull, gainXp, trainStat,
    addConsumable, buy, equipItem, useConsumable, learnJutsu,
    save, load, hasSave, wipe,
    register, login, logout, currentHero,
  };
})(window);
