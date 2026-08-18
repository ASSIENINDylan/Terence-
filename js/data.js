/* data.js — toutes les données de contenu du jeu (villages, jutsus, monstres,
   objets, terrains, missions, progression). Aucune logique : juste des tables
   consultées par world / combat / ui. Exposé sous SH.data. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});

  /* ----------------------------------------------------------------------
   * Terrains de la carte du pays de Yuukan.
   *  - walk      : praticable à pied ?
   *  - chakra    : chakra consommé par pas (le déplacement coûte de l'énergie)
   *  - encounter : probabilité de rencontre sauvage par pas
   *  - pool      : familles de monstres pouvant apparaître
   * -------------------------------------------------------------------- */
  const TERR = {
    PLAINE:   { id: 0, name: "Plaine",      walk: true,  chakra: 2, encounter: 0.06, color: "#5a7d3a", color2: "#4d6d31", pool: ["bandit", "loup"] },
    FORET:    { id: 1, name: "Forêt",       walk: true,  chakra: 3, encounter: 0.14, color: "#2f5a34", color2: "#264a2b", pool: ["loup", "serpent", "deserteur"] },
    MONTAGNE: { id: 2, name: "Montagne",    walk: true,  chakra: 5, encounter: 0.10, color: "#6b6259", color2: "#544d46", pool: ["golem", "deserteur"] },
    DESERT:   { id: 3, name: "Désert",      walk: true,  chakra: 4, encounter: 0.11, color: "#c9b26a", color2: "#b39c58", pool: ["scorpion", "bandit"] },
    EAU:      { id: 4, name: "Eau",         walk: false, chakra: 0, encounter: 0,    color: "#26476e", color2: "#1d3a5c", pool: [] },
    ROUTE:    { id: 5, name: "Route",       walk: true,  chakra: 1, encounter: 0.01, color: "#8a7b5c", color2: "#7a6c50", pool: ["bandit"] },
    VILLAGE:  { id: 6, name: "Village",     walk: true,  chakra: 0, encounter: 0,    color: "#3a4152", color2: "#2f3543", pool: [] },
  };
  const TERR_BY_ID = Object.values(TERR).reduce((m, t) => ((m[t.id] = t), m), {});

  /* ----------------------------------------------------------------------
   * Les trois villages cachés du pays de Yuukan.
   * spec = statistique de prédilection (tai/nin/gen).
   * -------------------------------------------------------------------- */
  const VILLAGES = [
    {
      id: "zambakro", name: "Zambakro", kanji: "力", color: "#e0553b",
      spec: "tai", specName: "Taijutsu",
      blurb: "Le village de la Force. Ses shinobi frappent au corps à corps.",
      start: { tai: 4, nin: 0, gen: 0 },
      teaches: ["lotus_initial", "ouragan_feuille", "danse_fauve", "clone_ombre"],
    },
    {
      id: "abidjan", name: "Abidjan", kanji: "魔", color: "#4aa3ff",
      spec: "nin", specName: "Ninjutsu",
      blurb: "Le village des sceaux. Maîtres des éléments et des ninjutsus.",
      start: { tai: 0, nin: 4, gen: 0 },
      teaches: ["boule_feu", "dragon_eau", "eclair_pourfendeur", "soin_medical"],
    },
    {
      id: "akradjo", name: "Akradjo", kanji: "幻", color: "#c79bff",
      spec: "gen", specName: "Genjutsu",
      blurb: "Le village de l'Illusion. On y brise l'esprit avant le corps.",
      start: { tai: 0, nin: 0, gen: 4 },
      teaches: ["demon_illusoire", "prison_miroir", "tourment_infini", "clone_ombre"],
    },
  ];
  const VILLAGE_BY_ID = VILLAGES.reduce((m, v) => ((m[v.id] = v), m), {});

  /* ----------------------------------------------------------------------
   * Jutsus. scale = stat qui augmente les dégâts/soins.
   * effect : "damage" (défaut) | "heal" | "stun" | "buff_spd" | "debuff_def"
   * -------------------------------------------------------------------- */
  const JUTSUS_LIST = [
    // Ninjutsu (Mahou)
    { id: "boule_feu",        name: "Boule de Feu",         type: "nin", scale: "nin", cost: 10, power: 12, effect: "damage",     desc: "Katon de base : une sphère de flammes." },
    { id: "dragon_eau",       name: "Dragon d'Eau",         type: "nin", scale: "nin", cost: 22, power: 26, effect: "damage",     desc: "Suiton dévastateur en forme de dragon." },
    { id: "eclair_pourfendeur", name: "Éclair Pourfendeur", type: "nin", scale: "nin", cost: 30, power: 36, effect: "damage",     desc: "Raiton perçant, très puissant." },
    { id: "soin_medical",     name: "Ninjutsu Médical",     type: "nin", scale: "nin", cost: 16, power: 26, effect: "heal",       desc: "Referme les blessures en canalisant le chakra." },
    // Genjutsu (Gensou)
    { id: "demon_illusoire",  name: "Démon Illusoire",      type: "gen", scale: "gen", cost: 12, power: 14, effect: "stun",       desc: "Illusion qui peut figer l'ennemi de terreur." },
    { id: "prison_miroir",    name: "Prison de Miroirs",    type: "gen", scale: "gen", cost: 22, power: 24, effect: "debuff_def",  desc: "Enferme l'esprit et brise ses défenses." },
    { id: "tourment_infini",  name: "Tourment Infini",      type: "gen", scale: "gen", cost: 32, power: 38, effect: "damage",     desc: "Fait revivre mille morts en un instant." },
    // Taijutsu (Chikara)
    { id: "lotus_initial",    name: "Lotus Initial",        type: "tai", scale: "tai", cost: 8,  power: 12, effect: "damage",     desc: "Enchaînement de coups rapprochés." },
    { id: "ouragan_feuille",  name: "Ouragan de la Feuille", type: "tai", scale: "tai", cost: 16, power: 22, effect: "damage",     desc: "Coup de pied ascendant fulgurant." },
    { id: "danse_fauve",      name: "Danse du Fauve",       type: "tai", scale: "tai", cost: 26, power: 30, effect: "buff_spd",    desc: "Déchaîne une furie qui accélère le corps." },
    // Commun
    { id: "clone_ombre",      name: "Clone de l'Ombre",     type: "nin", scale: "nin", cost: 14, power: 16, effect: "damage",     desc: "Des doubles frappent l'ennemi de toutes parts." },
  ];
  const JUTSUS = JUTSUS_LIST.reduce((m, j) => ((m[j.id] = j), m), {});

  /* ----------------------------------------------------------------------
   * Familles de monstres. Les stats réelles sont dérivées du niveau dans
   * combat.js à partir de ces coefficients.
   * -------------------------------------------------------------------- */
  const MONSTERS = {
    bandit:    { id: "bandit",    name: "Bandit",           palette: ["#7a5230", "#3a2a1a", "#caa27a"], hp: 22, atk: 5, def: 2, spd: 5, xp: 10, ryo: 12, jutsu: null,          desc: "Un pillard de grand chemin." },
    loup:      { id: "loup",      name: "Loup des bois",    palette: ["#6b6b6b", "#333", "#c9c9c9"],     hp: 18, atk: 6, def: 1, spd: 8, xp: 11, ryo: 8,  jutsu: null,          desc: "Rapide et féroce." },
    serpent:   { id: "serpent",   name: "Serpent géant",    palette: ["#3f7a3a", "#234522", "#9fd08a"],  hp: 30, atk: 6, def: 3, spd: 6, xp: 15, ryo: 14, jutsu: null,          desc: "Ses anneaux broient le chakra." },
    scorpion:  { id: "scorpion",  name: "Scorpion des sables", palette: ["#b08030", "#5c421a", "#e8c878"], hp: 26, atk: 7, def: 4, spd: 6, xp: 16, ryo: 16, jutsu: null,       desc: "Un dard chargé de venin." },
    golem:     { id: "golem",     name: "Golem de pierre",  palette: ["#7d766b", "#4a453e", "#a9a297"],  hp: 55, atk: 8, def: 8, spd: 3, xp: 26, ryo: 28, jutsu: null,          desc: "Lent mais presque indestructible." },
    deserteur: { id: "deserteur", name: "Ninja déserteur",  palette: ["#3a3f52", "#20232e", "#e0553b"],  hp: 34, atk: 8, def: 4, spd: 9, xp: 30, ryo: 34, jutsu: "boule_feu",   desc: "Un shinobi renégat qui connaît des jutsus." },
  };

  /* ----------------------------------------------------------------------
   * Objets : potions (consommables) et équipement (bonus permanents).
   * slot : "conso" | "arme" | "armure" | "bandeau"
   * -------------------------------------------------------------------- */
  const ITEMS_LIST = [
    { id: "potion_soin",     name: "Potion de soin",     slot: "conso", price: 30,  levelReq: 1,  use: { hp: 45 },        desc: "Restaure 45 PV." },
    { id: "potion_soin_sup", name: "Potion supérieure",  slot: "conso", price: 90,  levelReq: 5,  use: { hp: 110 },       desc: "Restaure 110 PV." },
    { id: "pilule_chakra",   name: "Pilule de chakra",   slot: "conso", price: 45,  levelReq: 1,  use: { ck: 50 },        desc: "Restaure 50 points de chakra." },
    { id: "elixir",          name: "Élixir du sennin",   slot: "conso", price: 160, levelReq: 8,  use: { hp: 999, ck: 999 }, desc: "Restaure entièrement PV et chakra." },

    { id: "kunai",           name: "Kunai renforcé",     slot: "arme",   price: 60,   levelReq: 1,  bonus: { tai: 3 },             desc: "+3 Taijutsu." },
    { id: "katana",          name: "Katana d'acier",     slot: "arme",   price: 240,  levelReq: 6,  bonus: { tai: 9 },             desc: "+9 Taijutsu." },
    { id: "lame_chakra",     name: "Lame de chakra",     slot: "arme",   price: 620,  levelReq: 12, bonus: { tai: 16, nin: 6 },    desc: "+16 Taijutsu, +6 Ninjutsu." },

    { id: "gilet_maille",    name: "Gilet de mailles",   slot: "armure", price: 120,  levelReq: 3,  bonus: { pvMax: 30, def: 3 },  desc: "+30 PV max, +3 Défense." },
    { id: "armure_anbu",     name: "Armure de l'Anbu",   slot: "armure", price: 560,  levelReq: 12, bonus: { pvMax: 70, def: 7, spd: 2 }, desc: "+70 PV max, +7 Déf, +2 Vitesse." },

    { id: "bandeau_gravé",   name: "Bandeau gravé",      slot: "bandeau", price: 100, levelReq: 2,  bonus: { chakraMax: 25 },      desc: "+25 chakra max." },
    { id: "bandeau_maitre",  name: "Bandeau du maître",  slot: "bandeau", price: 500, levelReq: 11, bonus: { chakraMax: 60, nin: 5, gen: 5 }, desc: "+60 chakra, +5 Nin, +5 Gen." },
  ];
  const ITEMS = ITEMS_LIST.reduce((m, it) => ((m[it.id] = it), m), {});

  /* ----------------------------------------------------------------------
   * Grades (rang ninja) selon le niveau.
   * -------------------------------------------------------------------- */
  const RANKS = [
    { min: 1,  name: "Genin" },
    { min: 6,  name: "Chunin" },
    { min: 12, name: "Jonin" },
    { min: 20, name: "Anbu" },
    { min: 30, name: "Kage" },
  ];
  function rankForLevel(lvl) {
    let r = RANKS[0].name;
    for (const x of RANKS) if (lvl >= x.min) r = x.name;
    return r;
  }

  // Expérience nécessaire pour passer du niveau `lvl` au suivant.
  function xpForLevel(lvl) { return Math.round(35 * Math.pow(lvl, 1.55)) + 25; }

  /* ----------------------------------------------------------------------
   * Modèles de missions (le contenu concret est instancié par village dans
   * ui.js selon le niveau du joueur).
   * -------------------------------------------------------------------- */
  const MISSION_TEMPLATES = [
    { rank: "D", type: "hunt", target: "bandit",   count: 3, xp: 40,  ryo: 60,  minLevel: 1, title: "Nettoyer la route", desc: "Des bandits rançonnent les voyageurs. Éliminez-en 3." },
    { rank: "D", type: "hunt", target: "loup",     count: 4, xp: 55,  ryo: 70,  minLevel: 2, title: "La meute affamée",   desc: "Une meute de loups menace le village. Abattez-en 4." },
    { rank: "C", type: "hunt", target: "serpent",  count: 3, xp: 90,  ryo: 120, minLevel: 4, title: "Le nid de serpents",  desc: "Des serpents géants infestent la forêt. Tuez-en 3." },
    { rank: "C", type: "hunt", target: "scorpion", count: 4, xp: 110, ryo: 150, minLevel: 5, title: "Danger du désert",    desc: "Les scorpions pullulent. Éliminez-en 4." },
    { rank: "B", type: "hunt", target: "golem",    count: 2, xp: 180, ryo: 260, minLevel: 8, title: "Gardiens de pierre",   desc: "Des golems bloquent un col. Détruisez-en 2." },
    { rank: "B", type: "hunt", target: "deserteur", count: 3, xp: 240, ryo: 340, minLevel: 10, title: "Chasse aux renégats", desc: "Traquez 3 ninjas déserteurs." },
  ];

  /* ----------------------------------------------------------------------
   * Apparence personnalisable du ninja. L'apparence stocke des INDEX dans ces
   * palettes (facile à faire défiler dans l'éditeur) ; render.js les convertit
   * en couleurs concrètes.
   * -------------------------------------------------------------------- */
  const APPEARANCE = {
    skin:    { label: "Peau",    colors: ["#f0c9a4", "#e0b184", "#c98d5f", "#a5673f", "#7a4a2b", "#4e3220"] },
    outfit:  { label: "Tenue",   colors: ["#20242e", "#2b3040", "#39304a", "#123a2e", "#4a1f22", "#1f3a4a", "#3a3320", "#5a5f6b"] },
    hair:    { label: "Cheveux", colors: ["#141414", "#3a2a17", "#6b4a2a", "#9a9a9a", "#c0392b", "#2a4a6b", "#d9c27a"] },
    band:    { label: "Bandeau", colors: ["#e0553b", "#4aa3ff", "#c79bff", "#f0c04a", "#4bd07a", "#e6e6e6", "#e08a2b", "#111111"] },
  };
  const APPEARANCE_KEYS = ["skin", "outfit", "hair", "band"];

  function defaultAppearance() { return { skin: 1, outfit: 0, hair: 0, band: 0 }; }

  // Assainit une apparence (index valides) — utile au chargement d'une sauvegarde.
  function normAppearance(a) {
    const out = defaultAppearance();
    if (a) for (const k of APPEARANCE_KEYS) {
      const n = APPEARANCE[k].colors.length;
      if (Number.isInteger(a[k])) out[k] = ((a[k] % n) + n) % n;
    }
    return out;
  }

  // Assombrit une couleur hex d'un facteur (0..1).
  function shade(hex, f) {
    const n = parseInt(hex.slice(1), 16);
    const r = Math.round(((n >> 16) & 255) * f);
    const g = Math.round(((n >> 8) & 255) * f);
    const b = Math.round((n & 255) * f);
    return "#" + ((1 << 24) | (r << 16) | (g << 8) | b).toString(16).slice(1);
  }

  // Convertit une apparence (index) en couleurs concrètes pour le rendu.
  function appearancePalette(a) {
    a = normAppearance(a);
    const outfit = APPEARANCE.outfit.colors[a.outfit];
    return {
      skin: APPEARANCE.skin.colors[a.skin],
      outfit,
      outfitDark: shade(outfit, 0.7),
      hair: APPEARANCE.hair.colors[a.hair],
      band: APPEARANCE.band.colors[a.band],
    };
  }

  SH.data = {
    TERR, TERR_BY_ID, VILLAGES, VILLAGE_BY_ID,
    JUTSUS, JUTSUS_LIST, MONSTERS, ITEMS, ITEMS_LIST,
    RANKS, rankForLevel, xpForLevel, MISSION_TEMPLATES,
    APPEARANCE, APPEARANCE_KEYS, defaultAppearance, normAppearance, appearancePalette, shade,
  };
})(window);
