/* rng.js — petit générateur pseudo-aléatoire déterministe (mulberry32).
   Utilisé pour générer une carte reproductible à partir d'une graine, et
   pour des tirages ponctuels (rencontres, butin). Tout est exposé sous SH. */
(function (global) {
  "use strict";
  const SH = (global.SH = global.SH || {});

  // Crée une fonction rng() -> [0,1) à partir d'une graine entière.
  function mulberry32(seed) {
    let a = seed >>> 0;
    return function () {
      a |= 0;
      a = (a + 0x6d2b79f5) | 0;
      let t = Math.imul(a ^ (a >>> 15), 1 | a);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  // Générateur global pour les tirages non-déterministes du jeu.
  let _global = mulberry32((Math.random() * 2 ** 32) >>> 0);

  SH.rng = {
    make: mulberry32,
    // reseed le générateur global (ex. depuis une sauvegarde) — optionnel.
    reseed(seed) { _global = mulberry32(seed >>> 0); },
    // float [0,1)
    f() { return _global(); },
    // entier [min, max] inclus
    int(min, max) { return min + Math.floor(_global() * (max - min + 1)); },
    // vrai avec la probabilité p
    chance(p) { return _global() < p; },
    // élément au hasard d'un tableau
    pick(arr) { return arr[Math.floor(_global() * arr.length)]; },
  };
})(window);
