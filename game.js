"use strict";
/* ============================================================================
   SHADOW OF THE WARRIOR — v3  "Les Cinq Portails"
   Moteur : combat rendu en WebGL (PixiJS) pour une ambiance à la Darkest
   Dungeon (pénombre, halos de torche, ombres portées, brouillard, sang).
   Carte / boutiques / menus rendus sur un canvas 2D superposé.
   ========================================================================== */

/* ------------------------------- CANVAS 2D -------------------------------- */
const cv = document.getElementById("map2d");
const ctx = cv.getContext("2d");
const W = cv.width, H = cv.height;
const pixiCanvas = document.getElementById("pixi");

/* -------------------------------- ENTRÉES --------------------------------- */
const keys = {}, justPressed = {};
addEventListener("keydown", (e) => {
  const k = e.key.toLowerCase();
  if (!keys[k]) justPressed[k] = true;
  keys[k] = true;
  if (["arrowup","arrowdown","arrowleft","arrowright"," "].includes(k)) e.preventDefault();
});
addEventListener("keyup", (e) => { keys[e.key.toLowerCase()] = false; });
function pressed(){ for (let i=0;i<arguments.length;i++) if (justPressed[arguments[i]]) return true; return false; }
function held(){ for (let i=0;i<arguments.length;i++) if (keys[arguments[i]]) return true; return false; }
function clearJust(){ for (const k in justPressed) delete justPressed[k]; }

/* --------------------------- HELPERS DE DESSIN 2D ------------------------- */
function lg(c,x0,y0,x1,y1,stops){ const g=c.createLinearGradient(x0,y0,x1,y1); for(const s of stops)g.addColorStop(s[0],s[1]); return g; }
function rg(c,x0,y0,r0,x1,y1,r1,stops){ const g=c.createRadialGradient(x0,y0,r0,x1,y1,r1); for(const s of stops)g.addColorStop(s[0],s[1]); return g; }
function rr(c,x,y,w,h,r){ if(typeof r==="number")r=[r,r,r,r]; c.beginPath();
  c.moveTo(x+r[0],y); c.arcTo(x+w,y,x+w,y+h,r[1]); c.arcTo(x+w,y+h,x,y+h,r[2]);
  c.arcTo(x,y+h,x,y,r[3]); c.arcTo(x,y,x+w,y,r[0]); c.closePath(); }
function rrf(x,y,w,h,r,fill){ rr(ctx,x,y,w,h,r); ctx.fillStyle=fill; ctx.fill(); }
function rrs(x,y,w,h,r,st,lw){ rr(ctx,x,y,w,h,r); ctx.strokeStyle=st; ctx.lineWidth=lw||1; ctx.stroke(); }
function txt(t,x,y,size,color,align,weight){
  ctx.textAlign=align||"left"; ctx.font=(weight?weight+" ":"")+size+"px 'Trebuchet MS',Georgia,serif";
  ctx.fillStyle="rgba(0,0,0,0.8)"; ctx.fillText(t,x+1.2,y+1.4);
  ctx.fillStyle=color; ctx.fillText(t,x,y);
}
function wrap(t,x,y,maxW,lh,size,color,align,weight){
  ctx.font=(weight?weight+" ":"")+size+"px 'Trebuchet MS',Georgia,serif";
  const words=t.split(" "); let line="",cy=y;
  for(const w of words){ const test=line?line+" "+w:w;
    if(ctx.measureText(test).width>maxW&&line){ txt(line,x,cy,size,color,align,weight); line=w; cy+=lh; }
    else line=test; }
  if(line) txt(line,x,cy,size,color,align,weight); return cy;
}
function bar2d(x,y,w,h,ratio,c1,c2){
  ratio=Math.max(0,Math.min(1,ratio));
  rrf(x,y,w,h,h/2,"rgba(0,0,0,0.72)");
  if(ratio>0.012){ rr(ctx,x+1.5,y+1.5,Math.max(2,(w-3)*ratio),h-3,(h-3)/2);
    ctx.fillStyle=lg(ctx,x,y,x,y+h,[[0,c1],[1,c2]]); ctx.fill();
    rr(ctx,x+3,y+2.4,Math.max(1,(w-6)*ratio),Math.max(1,h*0.22),1); ctx.fillStyle="rgba(255,255,255,0.26)"; ctx.fill(); }
  rrs(x,y,w,h,h/2,"rgba(0,0,0,0.85)",1.2);
}
function panel2d(x,y,w,h,glow){
  ctx.save(); ctx.shadowColor="rgba(0,0,0,0.7)"; ctx.shadowBlur=22; ctx.shadowOffsetY=7;
  rr(ctx,x,y,w,h,10); ctx.fillStyle=lg(ctx,x,y,x,y+h,[[0,"rgba(30,24,16,0.97)"],[1,"rgba(12,9,6,0.98)"]]); ctx.fill();
  ctx.restore();
  rrs(x+1.5,y+1.5,w-3,h-3,9,glow||"rgba(200,162,74,0.55)",1.6);
  rrs(x+5,y+5,w-10,h-10,6,"rgba(200,162,74,0.16)",1);
}
function shade(hex,f){
  const r=Math.min(255,Math.round(parseInt(hex.slice(1,3),16)*f));
  const g=Math.min(255,Math.round(parseInt(hex.slice(3,5),16)*f));
  const b=Math.min(255,Math.round(parseInt(hex.slice(5,7),16)*f));
  return "rgb("+r+","+g+","+b+")";
}
function hx(hex){ return parseInt(hex.slice(1),16); }

/* ------------------------------- ÉTAT GLOBAL ------------------------------ */
const S = { LOAD:0, SELECT:1, MAP:2, PORTAL:3, COMBAT:4, SHOP:5, MSG:6, EVOLVE:7, GAMEOVER:8, REST:9 };
let state = S.LOAD, time = 0, selCursor = 0, evoCursor = 0, portalCursor = 0;
let shake = 0;

/* ============================ CLASSES & PASSIFS ============================ */
/* Chaque classe : stats de base, capacité active, PASSIF unique, apparence.  */
const CLASSES = [
  { id:"guerrier", nom:"Guerrier", titre:"Le Rempart", couleur:"#c14a33",
    stats:{ pv:70, force:11, vitesse:3, esprit:2 },
    arme:{ nom:"Épée ébréchée", degats:4 },
    passif:{ nom:"Peau de fer", desc:"Réduit de 20% tous les dégâts subis." },
    capacite:{ id:"titan", nom:"Colère du Titan", cout:30, desc:"Frappe imparable à 230% des dégâts." },
    desc:"Un colosse bardé d'acier. Il encaisse là où les autres tombent." },
  { id:"ombre", nom:"Lame des Ombres", titre:"L'Insaisissable", couleur:"#6a5cd8",
    stats:{ pv:48, force:6, vitesse:11, esprit:3 },
    arme:{ nom:"Wakizashi terni", degats:4 },
    passif:{ nom:"Célérité", desc:"25% de chance de rejouer immédiatement après une attaque." },
    capacite:{ id:"danse", nom:"Danse des Ombres", cout:28, desc:"3 frappes fulgurantes à 85% chacune." },
    desc:"Il frappe entre deux battements de cœur et disparaît dans le noir." },
  { id:"mage", nom:"Mage de Guerre", titre:"L'Éveillé", couleur:"#b24ad8",
    stats:{ pv:44, force:3, vitesse:5, esprit:12 },
    arme:{ nom:"Bâton fêlé", degats:3 },
    passif:{ nom:"Flux mystique", desc:"Régénère +8 énergie de plus à chaque tour." },
    capacite:{ id:"feu", nom:"Trait de Feu", cout:24, desc:"Dégâts magiques imparables, croît avec l'Esprit." },
    desc:"Sa colère brûle l'air lui-même. L'ennemi n'a nulle part où fuir." },
];
const EVOLUTIONS = {
  guerrier:[
    { id:"berserker", nom:"Berserker", couleur:"#ff4a24", bonus:{pv:20,force:6,vitesse:3,esprit:0},
      passif:{ nom:"Sang bouillant", desc:"Sous 40% de PV, inflige +35% de dégâts." },
      capacite:{ id:"rage", nom:"Rage Sanglante", cout:38, desc:"340% imparable, mais coûte 8% de vos PV." },
      desc:"La douleur n'est qu'un carburant de plus." },
    { id:"paladin", nom:"Paladin", couleur:"#e8c24a", bonus:{pv:36,force:4,vitesse:0,esprit:4},
      passif:{ nom:"Lumière rédemptrice", desc:"Regagne 4 PV au début de chaque tour." },
      capacite:{ id:"jugement", nom:"Jugement Sacré", cout:40, desc:"190% et vous soigne de 28% des PV max." },
      desc:"Un bastion béni ; chaque coup porté est une prière." },
    { id:"seigneur", nom:"Seigneur de Guerre", couleur:"#c8823a", bonus:{pv:26,force:4,vitesse:3,esprit:2},
      passif:{ nom:"Aura de commandement", desc:"+2 armure permanente supplémentaire." },
      capacite:{ id:"etendard", nom:"Étendard de Guerre", cout:34, desc:"+6 Force, +8 armure, +12% esquive (4 tours)." },
      desc:"Là où flotte son étendard, la mort recule." },
  ],
  ombre:[
    { id:"maitre", nom:"Maître des Ombres", couleur:"#9a7cff", bonus:{pv:14,force:2,vitesse:6,esprit:3},
      passif:{ nom:"Fondu", desc:"+8% d'esquive permanente." },
      capacite:{ id:"voile", nom:"Voile d'Ombre", cout:34, desc:"Esquive garantie 2 tours ; frappe suivante ×2.6." },
      desc:"On ne tue pas ce qu'on ne voit pas." },
    { id:"duelliste", nom:"Duelliste", couleur:"#3ab8d8", bonus:{pv:20,force:4,vitesse:4,esprit:1},
      passif:{ nom:"Riposte", desc:"25% de riposte automatique quand vous esquivez." },
      capacite:{ id:"estocade", nom:"Estocade Fulgurante", cout:40, desc:"4 frappes précises à 75% chacune." },
      desc:"L'élégance est une arme tranchante." },
    { id:"traqueur", nom:"Traqueur", couleur:"#4ad86a", bonus:{pv:16,force:3,vitesse:5,esprit:2},
      passif:{ nom:"Toxines", desc:"Vos attaques normales ont 30% d'empoisonner." },
      capacite:{ id:"venin", nom:"Lame Empoisonnée", cout:34, desc:"140% + poison violent 3 tours (Esprit)." },
      desc:"Sa proie court encore, mais elle est déjà morte." },
  ],
  mage:[
    { id:"archimage", nom:"Archimage", couleur:"#d84af0", bonus:{pv:12,force:0,vitesse:2,esprit:7},
      passif:{ nom:"Surcharge arcanique", desc:"Vos sorts infligent +15% de dégâts." },
      capacite:{ id:"meteore", nom:"Météore", cout:44, desc:"Dégâts magiques colossaux et imparables." },
      desc:"Il fait tomber le ciel sur ceux qui le défient." },
    { id:"necro", nom:"Nécromancien", couleur:"#4ae89a", bonus:{pv:18,force:1,vitesse:1,esprit:5},
      passif:{ nom:"Moisson d'âmes", desc:"Récupère 5 PV à chaque ennemi vaincu." },
      capacite:{ id:"drain", nom:"Drain de Vie", cout:38, desc:"Dégâts magiques qui vous soignent d'autant." },
      desc:"La mort, pour lui, n'est qu'une matière première." },
    { id:"sage", nom:"Sage", couleur:"#4ab8e8", bonus:{pv:28,force:1,vitesse:1,esprit:5},
      passif:{ nom:"Sérénité", desc:"Commence chaque combat avec l'énergie pleine +20." },
      capacite:{ id:"benediction", nom:"Bénédiction", cout:34, desc:"Gros soin (Esprit) + 8 armure (3 tours)." },
      desc:"Une paix de fer que rien ne saurait entamer." },
  ],
};

/* ================================ PORTAILS ================================ */
/* 5 portails, chacun 10 étages ; étage 10 = boss. Difficulté croissante.     */
const PORTALS = [
  { id:0, nom:"Les Catacombes", couleur:"#8a9a6a", niveauReq:1,
    ambiance:{ mur1:"#2a2c22", mur2:"#14150f", sol1:"#312c20", sol2:"#161207", brume:"#3a4030", torche:"#ff9a3a" },
    pool:["gobelin","goule","chauve","bandit"],
    boss:{ nom:"Le Croque-Os", type:"boss_osseux", pv:340, dgt:26, esq:8, or:220, xp:180, crit:12, taille:1.6, tint:"#c9c2a8", eye:"#8affd2",
      special:"Charnier : frappe deux fois si vous êtes sous 50% de PV." } },
  { id:1, nom:"La Forêt Maudite", couleur:"#4a9a5a", niveauReq:6,
    ambiance:{ mur1:"#1c2c1e", mur2:"#0c140c", sol1:"#26301c", sol2:"#0e1408", brume:"#2c4030", torche:"#9aff6a" },
    pool:["loup","sylphe","goule","araignee"],
    boss:{ nom:"La Sylve Affamée", type:"boss_arbre", pv:520, dgt:32, esq:6, or:340, xp:300, crit:10, taille:1.75, tint:"#5a7a3a", eye:"#ffd23a",
      special:"Racines : soigne 30 PV quand vous ratez une attaque." } },
  { id:2, nom:"Le Sanctuaire Brisé", couleur:"#c8a24a", niveauReq:11,
    ambiance:{ mur1:"#2c2618", mur2:"#141008", sol1:"#33291a", sol2:"#161006", brume:"#4a3a24", torche:"#ffcf5a" },
    pool:["cultiste","gargouille","spectre","bandit"],
    boss:{ nom:"L'Inquisiteur Déchu", type:"boss_pretre", pv:720, dgt:40, esq:14, or:520, xp:460, crit:16, taille:1.55, tint:"#b0483a", eye:"#ff5a3a",
      special:"Anathème : dissipe vos buffs et frappe fort tous les 3 tours." } },
  { id:3, nom:"Les Abysses Gelés", couleur:"#5ab8e8", niveauReq:16,
    ambiance:{ mur1:"#1c2830", mur2:"#0a1016", sol1:"#20303a", sol2:"#0a1218", brume:"#2a4a5a", torche:"#7ad2ff" },
    pool:["revenant","gargouille","spectre","araignee"],
    boss:{ nom:"Le Roi Gelé", type:"boss_givre", pv:980, dgt:50, esq:12, or:760, xp:680, crit:18, taille:1.7, tint:"#7aa8d8", eye:"#d2f4ff",
      special:"Gel : a 25% de vous faire perdre votre tour." } },
  { id:4, nom:"Le Trône de Cendres", couleur:"#e8543a", niveauReq:22,
    ambiance:{ mur1:"#2c1614", mur2:"#140806", sol1:"#331612", sol2:"#160604", brume:"#5a2418", torche:"#ff6a2a" },
    pool:["demon","revenant","cultiste","gargouille"],
    boss:{ nom:"Malphas, l'Embrasé", type:"boss_demon", pv:1500, dgt:64, esq:14, or:1200, xp:1100, crit:22, taille:1.9, tint:"#c93a2a", eye:"#ffd23a",
      special:"Immolation : s'enflamme sous 40% et inflige +50% de dégâts." } },
];

/* ================================ BESTIAIRE =============================== */
const MOBS = {
  gobelin:  { nom:"Gobelin charognard", pv:34, dgt:9,  esq:10, or:[10,18],  xp:10, tint:"#5f8a3c", eye:"#ffd23a" },
  goule:    { nom:"Goule affamée",      pv:46, dgt:11, esq:8,  or:[12,22],  xp:13, tint:"#8a9a6a", eye:"#b6ff6a" },
  chauve:   { nom:"Nuée ailée",         pv:30, dgt:8,  esq:24, or:[10,20],  xp:12, tint:"#4a3a5a", eye:"#ff6a9a", flotte:true },
  bandit:   { nom:"Bandit masqué",      pv:50, dgt:13, esq:16, or:[16,30],  xp:16, tint:"#8a6a42", eye:"#ff4a3a" },
  loup:     { nom:"Loup sylvestre",     pv:44, dgt:14, esq:20, or:[14,26],  xp:15, tint:"#6a6258", eye:"#ffd23a" },
  sylphe:   { nom:"Sylphe rancunier",   pv:40, dgt:12, esq:28, or:[16,28],  xp:16, tint:"#4a8a5a", eye:"#d2ff9a", flotte:true },
  araignee: { nom:"Tisseuse noire",     pv:56, dgt:15, esq:14, or:[18,32],  xp:18, tint:"#3a2c3a", eye:"#ff3a5a" },
  cultiste: { nom:"Cultiste voilé",     pv:60, dgt:17, esq:14, or:[22,38],  xp:22, tint:"#7a2c3a", eye:"#ff6a3a" },
  gargouille:{nom:"Gargouille de pierre",pv:82,dgt:19, esq:6,  or:[26,44],  xp:26, tint:"#6a6e78", eye:"#ffb03a", taille:1.2 },
  spectre:  { nom:"Spectre hurlant",    pv:58, dgt:18, esq:30, or:[24,40],  xp:24, tint:"#5c7ec2", eye:"#c9f4ff", flotte:true },
  revenant: { nom:"Revenant gelé",      pv:96, dgt:22, esq:12, or:[30,52],  xp:30, tint:"#5a7a9a", eye:"#d2f4ff" },
  demon:    { nom:"Démon de cendre",    pv:120,dgt:27, esq:14, or:[40,68],  xp:38, tint:"#a83a2a", eye:"#ffd23a", taille:1.25, crit:12 },
};
function makeMob(portal, floor){
  const p = PORTALS[portal];
  const key = p.pool[Math.floor(Math.random()*p.pool.length)];
  const t = MOBS[key];
  // difficulté : portail + étage + niveau du héros
  const boost = (1 + portal*0.55) * (1 + (floor-1)*0.10) * (1 + (player.niveau-1)*0.06);
  return {
    nom:t.nom, type:key, tint:t.tint, eye:t.eye, flotte:!!t.flotte, taille:t.taille||1,
    crit:t.crit||0, esquive:t.esq,
    pvMax:Math.round(t.pv*boost), pv:Math.round(t.pv*boost),
    degats:Math.round(t.dgt*boost),
    or:Math.round((t.or[0]+Math.random()*(t.or[1]-t.or[0]))*(1+portal*0.4)),
    xp:Math.round(t.xp*boost), boss:false, hitFlash:0,
  };
}
function makeBoss(portal){
  const b = PORTALS[portal].boss;
  const boost = 1 + (player.niveau-1)*0.05;
  return { nom:b.nom, type:b.type, tint:b.tint, eye:b.eye, flotte:false, taille:b.taille,
    crit:b.crit, esquive:b.esq, pvMax:Math.round(b.pv*boost), pv:Math.round(b.pv*boost),
    degats:Math.round(b.dgt*boost), or:b.or, xp:b.xp, boss:true, special:b.special,
    specialId:b.type, hitFlash:0, enrage:false };
}

/* ================================ BOUTIQUES =============================== */
/* Les articles se débloquent selon le niveau du héros (niv requis).           */
const ARMES = [
  { nom:"Katana d'acier",      degats:9,  prix:60,   niv:1 },
  { nom:"Fauchon de guerre",   degats:15, prix:150,  niv:4 },
  { nom:"Lame d'obsidienne",   degats:22, prix:320,  niv:7 },
  { nom:"Espadon du dragon",   degats:30, prix:620,  niv:11 },
  { nom:"Faux des damnés",     degats:40, prix:1150, niv:15 },
  { nom:"Lame de l'Aube",      degats:54, prix:2200, niv:20 },
  { nom:"Trancheur d'âmes",    degats:72, prix:4200, niv:26 },
];
const ARMURES = [
  { nom:"Cotte de mailles",    defense:5,  prix:55,   niv:1 },
  { nom:"Harnais clouté",      defense:9,  prix:140,  niv:4 },
  { nom:"Armure d'ombre",      defense:14, prix:300,  niv:7 },
  { nom:"Plates du titan",     defense:20, prix:600,  niv:11 },
  { nom:"Égide runique",       defense:28, prix:1120, niv:15 },
  { nom:"Carapace du golem",   defense:38, prix:2150, niv:20 },
  { nom:"Manteau du néant",    defense:50, prix:4100, niv:26 },
];
function smithItems(){
  const list = [];
  for (const a of ARMES) if (a.niv <= player.niveau + 1) list.push({ ...a, type:"arme", desc:"Arme · +"+a.degats+" dégâts", locked:a.niv>player.niveau });
  for (const a of ARMURES) if (a.niv <= player.niveau + 1) list.push({ ...a, type:"armure", desc:"Armure · +"+a.defense+" défense", locked:a.niv>player.niveau });
  return list;
}
function apothItems(){
  const pot = Math.min(6, 1+Math.floor(player.niveau/3));
  return [
    { nom:"Potion de soin", type:"potion", desc:"Rend "+(30+player.niveau*2)+" PV en combat", prix:12+player.niveau*2, soin:30+player.niveau*2 },
    { nom:"Grande potion", type:"potion", desc:"Rend "+(70+player.niveau*4)+" PV en combat", prix:30+player.niveau*5, soin:70+player.niveau*4, niv:5 },
    { nom:"Élixir de Force", type:"force", desc:"+1 Force permanent", prix:35+player.niveau*6 },
    { nom:"Huile de Célérité", type:"vitesse", desc:"+1 Vitesse permanent", prix:35+player.niveau*6 },
    { nom:"Cristal d'Esprit", type:"esprit", desc:"+1 Esprit permanent", prix:35+player.niveau*6 },
    { nom:"Tonique vital", type:"pvmax", desc:"+12 PV Max permanent", prix:30+player.niveau*5 },
  ].filter(it => !it.niv || player.niveau >= it.niv);
}

/* ============================================================================
   PAINTERS — personnages "peints" sur canvas offscreen (couches, ombrage,
   contour-lumière, texture). Ces canvas deviennent des textures WebGL Pixi.
   ============================================================================ */
function mkCanvas(w,h){ const c=document.createElement("canvas"); c.width=w; c.height=h; return c; }
function noise(g,x,y,w,h,alpha,dark){
  g.save(); g.beginPath(); g.rect(x,y,w,h); g.clip();
  for(let i=0;i<w*h/26;i++){
    const px=x+Math.random()*w, py=y+Math.random()*h, s=0.6+Math.random()*1.4;
    g.fillStyle=(Math.random()<0.5?"rgba(0,0,0,":"rgba(255,255,255,")+(Math.random()*alpha*(dark?1.4:1))+")";
    g.fillRect(px,py,s,s);
  }
  g.restore();
}
function ellipse(g,x,y,rx,ry,fill){ g.beginPath(); g.ellipse(x,y,rx,ry,0,0,7); g.fillStyle=fill; g.fill(); }

/* --- Héros : figure haute, 3/4 face tournée vers la droite (l'ennemi) --- */
function paintHero(clsId, evoCol){
  const w=260,h=360,c=mkCanvas(w,h),g=c.getContext("2d");
  const cx=w*0.46, base=h-40;
  const P = ({
    guerrier:{ body:"#7a2f24", bodyD:"#3e1712", cloak:"#5a1f18", metal:"#c2cad6", metalD:"#5c6472", trim:"#d8b45a", skin:"#e0b088" },
    ombre:   { body:"#2a2f52", bodyD:"#12142a", cloak:"#1a1d38", metal:"#d8dee8", metalD:"#7a8290", trim:"#8f7ae8", skin:"#dcae86" },
    mage:    { body:"#4a2c66", bodyD:"#241436", cloak:"#33204a", metal:"#d2dcee", metalD:"#8a94ac", trim:"#e8c05a", skin:"#e0b088" },
  })[clsId];
  const rim = evoCol || P.trim;
  const warm="rgba(255,180,90,", cool="rgba(90,130,200,";

  // CAPE / MANTEAU (derrière)
  g.save();
  g.beginPath();
  g.moveTo(cx-6,base-232);
  g.bezierCurveTo(cx-64,base-180, cx-78,base-70, cx-52,base-6);
  g.lineTo(cx+30,base-6);
  g.bezierCurveTo(cx+18,base-90, cx+22,base-180, cx+8,base-232);
  g.closePath();
  g.fillStyle=lg(g,cx-70,base-232,cx+20,base,[[0,shade(P.cloak,1.15)],[0.5,P.cloak],[1,shade(P.cloak,0.4)]]);
  g.fill();
  noise(g,cx-80,base-232,110,232,0.12,true);
  // plis de la cape
  g.strokeStyle="rgba(0,0,0,0.35)"; g.lineWidth=3;
  for(let i=0;i<4;i++){ g.beginPath(); g.moveTo(cx-40+i*22,base-190);
    g.quadraticCurveTo(cx-50+i*24,base-90,cx-40+i*20,base-14); g.stroke(); }
  g.restore();

  // JAMBES
  const legTop=base-118;
  for(const s of [-1,1]){
    g.beginPath();
    g.moveTo(cx+s*4-8, legTop);
    g.lineTo(cx+s*4+8, legTop);
    g.lineTo(cx+s*10+6, base-6);
    g.lineTo(cx+s*10-8, base-6);
    g.closePath();
    g.fillStyle=lg(g,cx-14,legTop,cx+14,legTop,[[0,P.bodyD],[0.5,shade(P.body,0.7)],[1,P.bodyD]]);
    g.fill();
  }
  // bottes
  for(const s of [-1,1]){ ellipse(g,cx+s*10-1,base-6,13,7, "#1a1712");
    g.fillStyle=lg(g,cx,base-16,cx,base,[[0,"#3a3128"],[1,"#14110c"]]);
    rr(g,cx+s*10-12,base-20,24,15,[6,6,4,4]); g.fill(); }

  // TORSE / ARMURE — silhouette
  g.beginPath();
  g.moveTo(cx-40,base-210);
  g.bezierCurveTo(cx-48,base-170, cx-40,base-135, cx-30,base-120);
  g.lineTo(cx+30,base-120);
  g.bezierCurveTo(cx+42,base-140, cx+46,base-176, cx+36,base-210);
  g.bezierCurveTo(cx+20,base-232, cx-22,base-232, cx-40,base-210);
  g.closePath();
  g.fillStyle=lg(g,cx-40,base-232,cx+46,base-120,[[0,shade(P.body,1.2)],[0.5,P.body],[1,P.bodyD]]);
  g.fill();
  noise(g,cx-48,base-232,94,112,0.1,true);

  // plaque pectorale (métal) — guerrier surtout
  if(clsId!=="mage"){
    g.beginPath();
    g.moveTo(cx-26,base-206); g.quadraticCurveTo(cx,base-214,cx+26,base-206);
    g.lineTo(cx+22,base-150); g.quadraticCurveTo(cx,base-136,cx-22,base-150); g.closePath();
    g.fillStyle=lg(g,cx-26,base-206,cx+26,base-150,[[0,shade(P.metal,1.1)],[0.45,P.metal],[0.55,P.metalD],[1,shade(P.metalD,0.7)]]);
    g.fill();
    // reflet spéculaire
    g.fillStyle="rgba(255,255,255,0.4)";
    g.beginPath(); g.moveTo(cx-16,base-200); g.quadraticCurveTo(cx-8,base-198,cx-6,base-156);
    g.lineTo(cx-14,base-158); g.closePath(); g.fill();
    // emblème
    g.fillStyle=rim; g.beginPath();
    g.moveTo(cx,base-192); g.lineTo(cx+7,base-180); g.lineTo(cx,base-166); g.lineTo(cx-7,base-180); g.closePath(); g.fill();
  } else {
    // pendentif arcanique du mage
    g.fillStyle=rim; ellipse(g,cx,base-176,7,9,rim);
    g.save(); g.shadowColor=rim; g.shadowBlur=16; ellipse(g,cx,base-176,4,5,"#fff2cc"); g.restore();
    // liseré runique
    g.strokeStyle=rim; g.globalAlpha=0.7; g.lineWidth=2;
    g.beginPath(); g.moveTo(cx-30,base-150); g.quadraticCurveTo(cx,base-140,cx+30,base-150); g.stroke(); g.globalAlpha=1;
  }
  // ceinture
  g.fillStyle="#231a10"; rr(g,cx-34,base-132,68,12,3); g.fill();
  g.fillStyle=P.trim; rr(g,cx-8,base-131,16,10,2); g.fill();

  // BRAS ARRIÈRE (gauche)
  g.save();
  g.fillStyle=lg(g,cx-52,base-206,cx-30,base-150,[[0,shade(P.body,0.9)],[1,P.bodyD]]);
  rr(g,cx-54,base-206,22,64,[11,11,10,10]); g.fill();
  ellipse(g,cx-44,base-150,13,13, P.metalD); // gantelet
  g.restore();

  // PAULDRONS (épaulières)
  if(clsId!=="mage"){
    for(const s of [-1,1]){
      const px=cx+s*38, py=base-208;
      ellipse(g,px,py,20,17, lg(g,px-20,py-16,px+20,py+16,[[0,shade(P.metal,1.1)],[1,P.metalD]]));
      g.fillStyle="rgba(0,0,0,0.3)"; ellipse(g,px+s*3,py+4,14,10,"rgba(0,0,0,0.28)");
      g.fillStyle=rim; ellipse(g,px,py-3,5,4,rim);
    }
  } else {
    // capuche large
    for(const s of [-1,1]){ g.fillStyle=P.cloak;
      g.beginPath(); g.moveTo(cx+s*30,base-214); g.quadraticCurveTo(cx+s*54,base-206,cx+s*40,base-176);
      g.lineTo(cx+s*24,base-196); g.closePath(); g.fill(); }
  }

  // TÊTE
  const hy=base-232;
  ellipse(g,cx-2,hy-2,17,20, lg(g,cx-16,hy-18,cx+16,hy+16,[[0,shade(P.skin,1.05)],[1,shade(P.skin,0.6)]]));
  // ombre du casque/capuche sur le visage
  g.fillStyle="rgba(0,0,0,0.4)"; ellipse(g,cx-2,hy-8,16,12,"rgba(0,0,0,0.4)");
  // yeux luisants
  g.save(); g.shadowColor=rim; g.shadowBlur=8; g.fillStyle=rim;
  g.fillRect(cx-9,hy-2,5,3); g.fillRect(cx+3,hy-2,5,3); g.restore();

  // COIFFE selon classe
  if(clsId==="guerrier"){
    g.fillStyle=lg(g,cx-20,hy-22,cx+20,hy+6,[[0,shade(P.metal,1.1)],[0.5,P.metal],[1,P.metalD]]);
    g.beginPath(); g.moveTo(cx-19,hy+2); g.quadraticCurveTo(cx-2,hy-30,cx+19,hy+2);
    g.lineTo(cx+15,hy+4); g.quadraticCurveTo(cx-2,hy-22,cx-15,hy+4); g.closePath(); g.fill();
    g.fillStyle="#1a1712"; g.fillRect(cx-3,hy-8,6,14); // fente
    g.fillStyle=rim; // panache
    g.beginPath(); g.moveTo(cx-2,hy-26); g.quadraticCurveTo(cx-16,hy-46,cx-30,hy-38);
    g.quadraticCurveTo(cx-14,hy-34,cx-6,hy-20); g.closePath(); g.fill();
  } else if(clsId==="ombre"){
    g.fillStyle=lg(g,cx-22,hy-24,cx+18,hy+8,[[0,shade(P.cloak,1.2)],[1,shade(P.cloak,0.5)]]);
    g.beginPath(); g.moveTo(cx-20,hy+8); g.quadraticCurveTo(cx-24,hy-30,cx+2,hy-30);
    g.quadraticCurveTo(cx+24,hy-28,cx+20,hy+2); g.lineTo(cx+12,hy-2);
    g.quadraticCurveTo(cx+6,hy-22,cx-6,hy-20); g.quadraticCurveTo(cx-14,hy-16,cx-12,hy+8); g.closePath(); g.fill();
    g.fillStyle=rim; g.fillRect(cx-12,hy-6,26,3); // bandeau
    // écharpe flottante
    g.fillStyle=P.cloak; g.beginPath(); g.moveTo(cx-14,hy+4);
    g.quadraticCurveTo(cx-40,hy+18,cx-52,hy+40); g.quadraticCurveTo(cx-30,hy+22,cx-10,hy+16); g.closePath(); g.fill();
  } else {
    g.fillStyle=lg(g,cx-24,hy-30,cx+22,hy+10,[[0,shade(P.cloak,1.2)],[1,shade(P.cloak,0.45)]]);
    g.beginPath(); g.moveTo(cx-22,hy+10); g.quadraticCurveTo(cx-30,hy-36,cx+4,hy-34);
    g.quadraticCurveTo(cx+30,hy-30,cx+22,hy+6); g.lineTo(cx+12,hy);
    g.quadraticCurveTo(cx+8,hy-24,cx-6,hy-22); g.quadraticCurveTo(cx-16,hy-18,cx-12,hy+10); g.closePath(); g.fill();
  }

  // BRAS AVANT + ARME
  g.fillStyle=lg(g,cx+30,base-206,cx+52,base-150,[[0,shade(P.body,1.05)],[1,P.bodyD]]);
  rr(g,cx+32,base-204,22,60,[11,11,10,10]); g.fill();
  const hgx=cx+44, hgy=base-150;
  ellipse(g,hgx,hgy,12,12, P.metalD);
  if(clsId==="guerrier"){
    // grande épée levée
    g.save(); g.translate(hgx,hgy); g.rotate(-0.5);
    g.fillStyle="#2a1f12"; rr(g,-4,-2,8,26,2); g.fill(); // poignée
    g.fillStyle=P.trim; rr(g,-16,-6,32,8,3); g.fill(); // garde
    g.fillStyle=lg(g,-8,-150,8,-6,[[0,"#f4f8ff"],[0.5,"#c8d0dc"],[0.5,"#9aa2b0"],[1,"#5c6472"]]);
    g.beginPath(); g.moveTo(-9,-6); g.lineTo(-9,-140); g.lineTo(0,-158); g.lineTo(9,-140); g.lineTo(9,-6); g.closePath(); g.fill();
    g.fillStyle="rgba(255,255,255,0.55)"; g.fillRect(-6,-138,3,128); // arête lumineuse
    g.restore();
  } else if(clsId==="ombre"){
    g.save(); g.translate(hgx,hgy); g.rotate(-0.2);
    g.fillStyle="#1a1a22"; rr(g,-3,-2,6,20,2); g.fill();
    g.fillStyle=P.trim; rr(g,-11,-4,22,5,2); g.fill();
    g.fillStyle=lg(g,-6,-120,6,-4,[[0,"#f4f8ff"],[0.5,"#cfd6e2"],[1,"#7a8290"]]);
    g.beginPath(); g.moveTo(-5,-4); g.lineTo(-5,-118); g.lineTo(5,-130); g.lineTo(5,-4); g.closePath(); g.fill();
    g.fillStyle="rgba(255,255,255,0.6)"; g.fillRect(-3,-116,2,110);
    g.restore();
  } else {
    // bâton avec orbe
    g.save(); g.translate(hgx,hgy);
    g.fillStyle=lg(g,-3,-140,3,20,[[0,"#6a4e2f"],[1,"#3a2a18"]]); rr(g,-3,-140,6,160,3); g.fill();
    g.fillStyle="#4a3520"; g.fillRect(-4,-40,8,3);
    const og=g.createRadialGradient(0,-150,1,0,-150,26);
    og.addColorStop(0,"#fff2cc"); og.addColorStop(0.3,rim); og.addColorStop(1,"rgba(0,0,0,0)");
    g.fillStyle=og; g.beginPath(); g.arc(0,-150,26,0,7); g.fill();
    g.save(); g.shadowColor=rim; g.shadowBlur=20; ellipse(g,0,-150,8,10,"#fff2cc"); g.restore();
    g.restore();
  }

  // CONTOUR-LUMIÈRE global (rim light warm à droite, cool à gauche)
  g.globalCompositeOperation="source-atop";
  g.fillStyle=lg(g,cx-70,0,cx+70,0,[[0,cool+"0.16)"],[0.4,"rgba(0,0,0,0)"],[0.6,"rgba(0,0,0,0)"],[1,warm+"0.22)"]]);
  g.fillRect(0,0,w,h);
  // ombrage bas (occlusion)
  g.fillStyle=lg(g,0,base-120,0,base,[[0,"rgba(0,0,0,0)"],[1,"rgba(0,0,0,0.4)"]]);
  g.fillRect(0,base-120,w,120);
  g.globalCompositeOperation="source-over";

  return { canvas:c, w, h, anchorX:cx/w, anchorY:base/h };
}

/* --- Ennemis : peints tournés vers la GAUCHE (vers le héros). --- */
function EFEAT(type){
  // caractéristiques par type : archétype + options
  const m = {
    gobelin:{a:"hum",horns:1,ears:1,crouch:1}, goule:{a:"hum",gaunt:1,claws:1},
    bandit:{a:"hum",hood:1,blade:1}, cultiste:{a:"hum",hood:1,robe:1,staff:1},
    revenant:{a:"hum",skeletal:1,blade:1,frost:1}, demon:{a:"hum",horns:2,claws:1,fire:1,big:1},
    chauve:{a:"wraith",wings:1}, sylphe:{a:"wraith",leaf:1}, spectre:{a:"wraith",skull:1},
    loup:{a:"beast"}, araignee:{a:"spider"}, gargouille:{a:"gargoyle"},
    boss_osseux:{a:"hum",skeletal:1,crown:1,big:1,claws:1}, boss_arbre:{a:"tree"},
    boss_pretre:{a:"hum",hood:1,robe:1,crown:1,staff:1,big:1}, boss_givre:{a:"hum",crown:1,frost:1,big:1,blade:1},
    boss_demon:{a:"hum",horns:2,wings:1,fire:1,claws:1,big:1},
  };
  return m[type]||{a:"hum"};
}
function paintEnemy(e){
  const F=EFEAT(e.type), sc=e.taille*(e.boss?1.15:1);
  const w=Math.round(300*sc), h=Math.round(340*sc), c=mkCanvas(w,h), g=c.getContext("2d");
  const cx=w*0.54, base=h-30, tint=e.tint, dk=shade(tint,0.45), lt=shade(tint,1.25), eye=e.eye;
  g.save(); g.translate(cx,base); g.scale(sc,sc); g.translate(-cx,-base);

  if(F.a==="wraith"){
    // spectre : traîne déchirée + torse fantomatique
    g.globalAlpha=0.9;
    g.beginPath(); g.moveTo(cx,base-210);
    g.bezierCurveTo(cx-60,base-150,cx-70,base-40,cx-40,base+4);
    for(let i=-3;i<=3;i++){ g.lineTo(cx+i*14, base-4+((i%2)?14:0)); }
    g.bezierCurveTo(cx+66,base-60,cx+56,base-150,cx,base-210); g.closePath();
    g.fillStyle=lg(g,cx,base-210,cx,base,[[0,lt],[0.5,tint],[1,"rgba(10,12,20,0)"]]); g.fill();
    noise(g,cx-70,base-210,140,210,0.08,true);
    if(F.wings){ g.fillStyle=dk;
      for(const s of [-1,1]){ g.beginPath(); g.moveTo(cx,base-170);
        g.quadraticCurveTo(cx+s*80,base-210,cx+s*96,base-150);
        g.quadraticCurveTo(cx+s*70,base-160,cx+s*40,base-140); g.closePath(); g.fill(); } }
    ellipse(g,cx,base-206,26,30, F.skull?"#d8d2c0":dk);
    if(F.skull){ g.fillStyle="#0a0c10"; ellipse(g,cx-9,base-208,6,7,"#0a0c10"); ellipse(g,cx+9,base-208,6,7,"#0a0c10");
      g.fillRect(cx-3,base-198,6,8); }
    g.save(); g.shadowColor=eye; g.shadowBlur=16; g.fillStyle=eye;
    ellipse(g,cx-9,base-208,5,6,eye); ellipse(g,cx+9,base-208,5,6,eye); g.restore();
  }
  else if(F.a==="beast"){
    // loup / quadrupède
    for(const s of [[-40,4],[ -18,10],[24,6],[46,10]]){
      g.fillStyle=dk; rr(g,cx+s[0],base-60,10,54,4); g.fill();
      ellipse(g,cx+s[0]+5,base-6,9,5,"#14100c"); }
    g.beginPath(); g.moveTo(cx-52,base-110); g.quadraticCurveTo(cx,base-150,cx+66,base-104);
    g.quadraticCurveTo(cx+60,base-60,cx+30,base-56); g.lineTo(cx-40,base-58);
    g.quadraticCurveTo(cx-60,base-80,cx-52,base-110); g.closePath();
    g.fillStyle=lg(g,cx,base-150,cx,base-56,[[0,lt],[1,dk]]); g.fill();
    noise(g,cx-60,base-150,130,100,0.1,true);
    // tête basse à gauche
    g.beginPath(); g.moveTo(cx-44,base-118); g.quadraticCurveTo(cx-84,base-112,cx-92,base-84);
    g.quadraticCurveTo(cx-76,base-70,cx-52,base-78); g.closePath(); g.fillStyle=tint; g.fill();
    g.fillStyle=dk; g.beginPath(); g.moveTo(cx-48,base-128); g.lineTo(cx-40,base-146); g.lineTo(cx-34,base-124); g.closePath(); g.fill();
    g.fillStyle="#e8e2d0"; g.beginPath(); g.moveTo(cx-90,base-80); g.lineTo(cx-96,base-74); g.lineTo(cx-84,base-76); g.closePath(); g.fill();
    g.save(); g.shadowColor=eye; g.shadowBlur=12; g.fillStyle=eye; ellipse(g,cx-66,base-96,4,5,eye); g.restore();
  }
  else if(F.a==="spider"){
    for(let i=0;i<8;i++){ const s=i<4?-1:1, k=i%4;
      g.strokeStyle=dk; g.lineWidth=6; g.lineCap="round"; g.beginPath();
      g.moveTo(cx,base-70);
      g.quadraticCurveTo(cx+s*(50+k*16),base-120-k*8, cx+s*(70+k*22),base-30+k*18); g.stroke(); }
    ellipse(g,cx,base-56,40,34, lg(g,cx,base-90,cx,base-20,[[0,lt],[1,dk]]));
    ellipse(g,cx-30,base-78,20,16, tint);
    noise(g,cx-40,base-90,80,70,0.1,true);
    g.fillStyle="#d8c05a"; ellipse(g,cx,base-56,14,18,"rgba(216,192,90,0.5)");
    g.save(); g.shadowColor=eye; g.shadowBlur=10; g.fillStyle=eye;
    for(const o of [[-38,-84],[-30,-88],[-24,-82],[-40,-76]]) ellipse(g,cx+o[0],base+o[1]+0,3,3,eye); g.restore();
  }
  else if(F.a==="gargoyle"){
    g.fillStyle=dk;
    for(const s of [-1,1]){ g.beginPath(); g.moveTo(cx,base-150);
      g.lineTo(cx+s*96,base-200); g.lineTo(cx+s*80,base-150); g.lineTo(cx+s*104,base-120);
      g.lineTo(cx+s*70,base-130); g.lineTo(cx+s*40,base-100); g.closePath(); g.fill(); }
    ellipse(g,cx,base-96,40,52, lg(g,cx-40,base-150,cx+40,base-40,[[0,lt],[1,dk]]));
    noise(g,cx-40,base-150,80,120,0.14,true);
    for(const s of [-1,1]){ g.fillStyle=tint; g.beginPath(); g.moveTo(cx+s*18,base-150);
      g.lineTo(cx+s*10,base-172); g.lineTo(cx+s*26,base-150); g.closePath(); g.fill(); }
    g.fillStyle=dk; ellipse(g,cx,base-136,26,22,dk);
    g.fillStyle="#e8e2d0"; g.beginPath(); g.moveTo(cx-14,base-126); g.lineTo(cx-18,base-116); g.lineTo(cx-8,base-122); g.closePath(); g.fill();
    g.save(); g.shadowColor=eye; g.shadowBlur=12; g.fillStyle=eye;
    ellipse(g,cx-9,base-140,5,4,eye); ellipse(g,cx+9,base-140,5,4,eye); g.restore();
  }
  else if(F.a==="tree"){
    // boss arbre
    g.fillStyle=lg(g,cx,base-30,cx,base-220,[[0,dk],[1,tint]]);
    g.beginPath(); g.moveTo(cx-46,base); g.quadraticCurveTo(cx-30,base-120,cx-40,base-200);
    g.quadraticCurveTo(cx,base-230,cx+40,base-200); g.quadraticCurveTo(cx+30,base-120,cx+46,base); g.closePath(); g.fill();
    noise(g,cx-46,base-220,92,220,0.12,true);
    for(let i=0;i<5;i++){ g.strokeStyle=dk; g.lineWidth=5; g.beginPath();
      g.moveTo(cx-30+i*14,base-40); g.quadraticCurveTo(cx-34+i*16,base-140,cx-26+i*13,base-200); g.stroke(); }
    // couronne de branches
    g.strokeStyle=tint; g.lineWidth=7; g.lineCap="round";
    for(const a of [-1.2,-0.6,0,0.6,1.2]){ g.beginPath(); g.moveTo(cx,base-200);
      g.lineTo(cx+Math.sin(a)*90, base-200-Math.cos(a)*70); g.stroke(); }
    // gueule béante
    g.fillStyle="#0a0806"; ellipse(g,cx,base-150,20,28,"#0a0806");
    g.save(); g.shadowColor=eye; g.shadowBlur=14; g.fillStyle=eye;
    ellipse(g,cx-16,base-176,6,8,eye); ellipse(g,cx+16,base-176,6,8,eye); g.restore();
  }
  else {
    // HUMANOÏDE (mobs + plusieurs boss)
    const bulk=F.big?1.18:1, gaunt=F.gaunt?0.82:1;
    // jambes
    for(const s of [-1,1]){ g.fillStyle=lg(g,cx-20,base-110,cx+20,base-110,[[0,dk],[0.5,shade(tint,0.7)],[1,dk]]);
      rr(g,cx+s*12*bulk-9,base-110,18,104,[8,8,6,6]); g.fill();
      ellipse(g,cx+s*12*bulk,base-6,13,7,"#14100c"); }
    // torse
    const tw=54*bulk*gaunt;
    g.beginPath();
    g.moveTo(cx-tw/2,base-200*bulk);
    g.bezierCurveTo(cx-tw/2-8,base-150,cx-tw/2+6,base-120,cx-18,base-110);
    g.lineTo(cx+18,base-110);
    g.bezierCurveTo(cx+tw/2-4,base-124,cx+tw/2+8,base-158,cx+tw/2,base-200*bulk);
    g.bezierCurveTo(cx+16,base-224*bulk,cx-16,base-224*bulk,cx-tw/2,base-200*bulk); g.closePath();
    g.fillStyle=lg(g,cx-tw/2,base-224,cx+tw/2,base-110,[[0,lt],[0.5,tint],[1,dk]]); g.fill();
    noise(g,cx-tw/2-4,base-224,tw+8,116,0.1,true);
    if(F.skeletal){ // côtes
      g.strokeStyle="rgba(0,0,0,0.4)"; g.lineWidth=3;
      for(let i=0;i<4;i++){ g.beginPath(); g.moveTo(cx-16,base-196+i*18);
        g.quadraticCurveTo(cx,base-190+i*18,cx+16,base-196+i*18); g.stroke(); } }
    if(F.robe){ g.fillStyle=lg(g,cx,base-140,cx,base-6,[[0,shade(tint,0.8)],[1,dk]]);
      g.beginPath(); g.moveTo(cx-tw/2,base-140); g.quadraticCurveTo(cx-tw*0.7,base-40,cx-tw*0.6,base-6);
      g.lineTo(cx+tw*0.6,base-6); g.quadraticCurveTo(cx+tw*0.7,base-40,cx+tw/2,base-140); g.closePath(); g.fill();
      noise(g,cx-tw*0.7,base-140,tw*1.4,134,0.08,true); }
    // bras
    for(const s of [-1,1]){ g.fillStyle=lg(g,cx-10,base-196,cx+10,base-140,[[0,shade(tint,0.95)],[1,dk]]);
      rr(g,cx+s*(tw/2)*0.96-8,base-198,16,64,[8,8,7,7]); g.fill();
      if(F.claws){ g.fillStyle="#e8e2d0"; for(let k=-1;k<=1;k++){ g.beginPath();
        g.moveTo(cx+s*(tw/2)-4+k*4,base-136); g.lineTo(cx+s*(tw/2)-2+k*4,base-118); g.lineTo(cx+s*(tw/2)+k*4,base-136); g.closePath(); g.fill(); } } }
    // tête
    const hy=base-214*bulk;
    ellipse(g,cx,hy,18*gaunt,21, F.skeletal?"#d8d2c0":lg(g,cx-16,hy-18,cx+16,hy+16,[[0,lt],[1,shade(tint,0.6)]]));
    noise(g,cx-18,hy-20,36,40,0.08,true);
    if(F.skeletal){ g.fillStyle="#0a0c10"; ellipse(g,cx-8,hy-2,6,7,"#0a0c10"); ellipse(g,cx+8,hy-2,6,7,"#0a0c10");
      g.fillRect(cx-2,hy+6,4,8);
      g.strokeStyle="#0a0c10"; g.lineWidth=1.5; for(let i=-1;i<=1;i++){ g.beginPath(); g.moveTo(cx-6+i*6,hy+10); g.lineTo(cx-6+i*6,hy+18); g.stroke(); } }
    if(F.horns){ g.fillStyle=F.horns===2?"#2a1410":shade(tint,0.7);
      for(const s of [-1,1]){ g.beginPath(); g.moveTo(cx+s*10,hy-14); g.quadraticCurveTo(cx+s*30,hy-40,cx+s*18,hy-52);
        g.quadraticCurveTo(cx+s*22,hy-34,cx+s*4,hy-16); g.closePath(); g.fill(); } }
    if(F.ears){ g.fillStyle=tint; for(const s of [-1,1]){ g.beginPath(); g.moveTo(cx+s*16,hy-4);
      g.lineTo(cx+s*36,hy-14); g.lineTo(cx+s*18,hy+6); g.closePath(); g.fill(); } }
    if(F.hood){ g.fillStyle=lg(g,cx-24,hy-24,cx+22,hy+12,[[0,shade(tint,0.9)],[1,dk]]);
      g.beginPath(); g.moveTo(cx-22,hy+14); g.quadraticCurveTo(cx-30,hy-30,cx+2,hy-30);
      g.quadraticCurveTo(cx+28,hy-26,cx+22,hy+8); g.lineTo(cx+12,hy+2);
      g.quadraticCurveTo(cx+8,hy-20,cx-6,hy-18); g.quadraticCurveTo(cx-16,hy-14,cx-12,hy+14); g.closePath(); g.fill();
      g.fillStyle="rgba(0,0,0,0.55)"; ellipse(g,cx,hy-2,12,10,"rgba(0,0,0,0.55)"); }
    if(F.crown){ g.fillStyle="#d8b45a";
      g.beginPath(); g.moveTo(cx-18,hy-16); for(let i=0;i<5;i++){ const x=cx-18+i*9; g.lineTo(x,hy-16); g.lineTo(x+4.5,hy-30); }
      g.lineTo(cx+18,hy-16); g.closePath(); g.fill();
      g.save(); g.shadowColor="#ffe9a0"; g.shadowBlur=10; g.fillStyle="#ffe9a0";
      for(let i=0;i<5;i++) ellipse(g,cx-16+i*9,hy-30,2,2,"#ffe9a0"); g.restore(); }
    if(F.frost){ g.fillStyle="rgba(200,240,255,0.35)"; ellipse(g,cx,base-170,tw*0.6,60,"rgba(200,240,255,0.22)");
      g.strokeStyle="#d2f4ff"; g.lineWidth=2; for(const s of [-1,1]){ g.beginPath(); g.moveTo(cx+s*20,base-200);
        g.lineTo(cx+s*30,base-230); g.stroke(); } }
    if(!F.skeletal){ g.save(); g.shadowColor=eye; g.shadowBlur=10; g.fillStyle=eye;
      ellipse(g,cx-8,hy-1,5,4,eye); ellipse(g,cx+8,hy-1,5,4,eye); g.restore(); }
    if(F.fire){ // braises
      for(let i=0;i<14;i++){ g.fillStyle="rgba(255,"+(120+Math.random()*100|0)+",40,"+(0.3+Math.random()*0.4)+")";
        ellipse(g,cx+(Math.random()*tw-tw/2),base-120-Math.random()*90,1.5+Math.random()*2,1.5+Math.random()*2,""); } }
    if(F.staff){ g.save(); g.translate(cx-tw/2*0.96,base-150); 
      g.fillStyle="#3a2a18"; rr(g,-3,-60,6,120,3); g.fill();
      const og=g.createRadialGradient(0,-62,1,0,-62,20); og.addColorStop(0,"#fff"); og.addColorStop(0.3,eye); og.addColorStop(1,"rgba(0,0,0,0)");
      g.fillStyle=og; g.beginPath(); g.arc(0,-62,20,0,7); g.fill(); g.restore(); }
    if(F.blade){ g.save(); g.translate(cx+tw/2*0.96,base-150); g.rotate(0.3);
      g.fillStyle="#3a2a18"; rr(g,-3,-2,6,20,2); g.fill(); g.fillStyle="#c8a24a"; rr(g,-9,-4,18,4,2); g.fill();
      g.fillStyle=lg(g,-6,-110,6,-4,[[0,"#eef2f8"],[1,"#7a8290"]]);
      g.beginPath(); g.moveTo(-6,-4); g.lineTo(-6,-104); g.lineTo(6,-116); g.lineTo(6,-4); g.closePath(); g.fill(); g.restore(); }
  }

  // rim light + occlusion
  g.globalCompositeOperation="source-atop";
  g.fillStyle=lg(g,cx-90,0,cx+90,0,[[0,"rgba(255,180,90,0.2)"],[0.45,"rgba(0,0,0,0)"],[1,"rgba(90,130,200,0.14)"]]);
  g.fillRect(0,0,w,h);
  g.fillStyle=lg(g,0,base-110,0,base,[[0,"rgba(0,0,0,0)"],[1,"rgba(0,0,0,0.42)"]]); g.fillRect(0,base-110,w,110);
  g.globalCompositeOperation="source-over";
  g.restore();
  return { canvas:c, w, h, anchorX:cx/w, anchorY:base/h };
}

/* ================================= JOUEUR ================================= */
const player = {
  x:450, y:470, w:20, h:26, dir:"down", moving:false, animT:0,
  classe:null, evolution:null,
  niveau:1, xp:0, xpMax:22, or:70,
  pvMax:60, pv:60, force:8, vitesse:6, esprit:5, degatsBase:5, energie:60,
  arme:{ nom:"Poings", degats:0 }, armure:{ nom:"Guenilles", defense:1 },
  potions:1, potionSoin:30,
  portailsFinis:[false,false,false,false,false],
};
function hasPassif(id){
  return (player.classe && player.classe.passif && player.classe.id===id)
      || (player.evolution && player.evolution.id===id);
}
function newGame(cls){
  player.classe=cls; player.evolution=null;
  player.niveau=1; player.xp=0; player.xpMax=22; player.or=70;
  player.pvMax=cls.stats.pv; player.pv=cls.stats.pv;
  player.force=cls.stats.force; player.vitesse=cls.stats.vitesse; player.esprit=cls.stats.esprit;
  player.degatsBase=5; player.arme={nom:cls.arme.nom,degats:cls.arme.degats}; player.armure={nom:"Guenilles",defense:1};
  player.potions=2; player.potionSoin=30; player.energie=energieMax();
  player.portailsFinis=[false,false,false,false,false];
  player.x=450; player.y=470; player.dir="down";
}
let combat=null;
function buffVal(k){ return (combat&&combat.buff)?(combat.buff[k]||0):0; }
function armureTotale(){ let a=player.armure.defense+Math.floor(player.force*0.6)+buffVal("armure");
  if(player.evolution&&player.evolution.id==="seigneur")a+=2; return a; }
function esquivePct(){ let e=Math.min(72,Math.round(player.vitesse*2.2)+buffVal("esquive"));
  if(player.evolution&&player.evolution.id==="maitre")e=Math.min(80,e+8); return e; }
function degatsAttaque(){ let d=player.degatsBase+player.arme.degats+Math.floor((player.force+buffVal("force"))*0.55);
  if(player.evolution&&player.evolution.id==="berserker"&&player.pv<player.pvMax*0.4)d=Math.round(d*1.35); return d; }
function reducDegats(){ return hasPassif("guerrier")?0.20:0; }
function doubleChance(){ return Math.min(28,Math.round(player.vitesse*1.2)); }
function energieMax(){ return 60+player.esprit*5; }
function energieRegen(){ let r=7+player.esprit; if(player.classe&&player.classe.id==="mage")r+=8; return r; }
function capacite(){ return player.evolution?player.evolution.capacite:player.classe.capacite; }
function passifActuel(){ return player.evolution?player.evolution.passif:player.classe.passif; }
function nomClasse(){ return player.evolution?player.evolution.nom:(player.classe?player.classe.nom:"?"); }
function couleurClasse(){ return player.evolution?player.evolution.couleur:(player.classe?player.classe.couleur:"#888"); }
function sortBonus(){ let b=1; if(player.evolution&&player.evolution.id==="archimage")b+=0.15; return b; }

const clog=[]; function log(m){ clog.push(m); if(clog.length>5)clog.shift(); }

/* ============================ RUN / PORTAIL / ÉTAGE ======================= */
let run=null; // { portal, floor }
function entrerPortail(pi){
  run={ portal:pi, floor:1 };
  demarrerCombat();
}
function demarrerCombat(){
  const boss = run.floor>=10;
  const e = boss ? makeBoss(run.portal) : makeMob(run.portal, run.floor);
  combat={ ennemi:e, tour:"joueur", cursor:0, anim:0, fini:false, buff:null, voile:0, critNext:false,
    poison:null, gel:false, playerFlash:0, introT:70, bossTurn:0 };
  player.energie = energieMax();
  if(player.evolution&&player.evolution.id==="sage") player.energie+=20;
  clog.length=0;
  log(boss ? "⚑ "+e.nom+" — Gardien du portail !" : "Étage "+run.floor+" — un "+e.nom+" surgit !");
  buildCombatScene();
  state=S.COMBAT;
}
function finEtage(){
  // victoire de combat non-boss : proposer repos / continuer
  if(run.floor>=10){ portailFini(); return; }
  state=S.REST;
}
function continuerRun(){ run.floor++; demarrerCombat(); }
function portailFini(){
  player.portailsFinis[run.portal]=true;
  const p=PORTALS[run.portal];
  showMsg(["★ PORTAIL PURIFIÉ ★", p.nom+" ne hantera plus ces terres.",
    (run.portal<4?"Un nouveau portail s'ouvre à vous.":"Vous avez vaincu les Cinq Portails. Légende.")],
    ()=>{ run=null; state=S.MAP; }, "map");
}

/* =============================== MESSAGES ================================= */
let msg={lignes:[],suite:null,fond:"map"};
function showMsg(l,s,f){ msg.lignes=Array.isArray(l)?l:[l]; msg.suite=s||(()=>{state=S.MAP;}); msg.fond=f||"map"; state=S.MSG; }

/* ============================ LOGIQUE DE COMBAT =========================== */
function frapper(mult,opt){
  opt=opt||{}; const e=combat.ennemi;
  if(!opt.imparable && Math.random()*100 < e.esquive){
    log("Le "+e.nom+" esquive !"); sceneFloat("enemy","Esquive","#b8b0a0"); return 0;
  }
  let d=Math.round(degatsAttaque()*mult*(0.9+Math.random()*0.25));
  if(combat.critNext){ d=Math.round(d*2.6); combat.critNext=false; sceneFloat("enemy","CRITIQUE !","#ffd34a",20); }
  e.pv=Math.max(0,e.pv-d); e.hitFlash=10;
  sceneLunge("hero"); sceneHit("enemy"); sceneBlood("enemy", Math.min(1,d/40)); sceneSlash("enemy");
  sceneFloat("enemy","-"+d,"#ffffff",18);
  log((opt.label?opt.label+" : ":"Vous infligez ")+d+" dégâts"+(opt.label?" !":"."));
  // Toxines (traqueur passif) sur attaque normale
  if(!opt.noPoison && player.evolution&&player.evolution.id==="traqueur"&&e.pv>0&&Math.random()<0.30){
    combat.poison={dgt:6+player.esprit*2,tours:3}; log("Vos toxines infectent l'ennemi.");
  }
  return d;
}
function magie(base,label,col){
  const e=combat.ennemi; const d=Math.round(base*sortBonus()*(0.9+Math.random()*0.25));
  e.pv=Math.max(0,e.pv-d); e.hitFlash=12;
  sceneHit("enemy"); sceneBlood("enemy",Math.min(1,d/50)); sceneAbilityFlash(col); sceneMagic("enemy",col);
  sceneFloat("enemy","-"+d,col,19); log(label+" : "+d+" dégâts !"); return d;
}
function soigner(n){ n=Math.round(n); player.pv=Math.min(player.pvMax,player.pv+n);
  sceneHeal("hero"); sceneFloat("hero","+"+n,"#6ae87a",17); log("Vous récupérez "+n+" PV."); }

function actionAttaque(){
  frapper(1,{});
  if(combat.ennemi.pv>0 && Math.random()*100<doubleChance()){ sceneFloat("enemy","Double !","#8ae8b0",14); frapper(0.6,{label:"Double frappe",noPoison:true}); }
  // Célérité (ombre base passif) : rejouer
  if(combat.ennemi.pv>0 && player.classe.id==="ombre" && !player.evolution && Math.random()<0.25){
    log("Célérité ! Vous frappez de nouveau."); sceneFloat("hero","Célérité","#8f7ae8",13); frapper(0.7,{label:"Enchaînement",noPoison:true});
  }
  finJoueur();
}
function actionCapacite(){
  const cap=capacite();
  if(player.energie<cap.cout){ log("Énergie insuffisante pour "+cap.nom+"."); return; }
  player.energie-=cap.cout;
  switch(cap.id){
    case "titan": frapper(2.3,{imparable:true,label:cap.nom}); break;
    case "danse": for(let i=0;i<3;i++) if(combat.ennemi.pv>0) frapper(0.85,{label:cap.nom,noPoison:true}); break;
    case "feu": magie(14+player.esprit*3.6,cap.nom,"#ff8a3a"); break;
    case "rage": { const c=Math.max(1,Math.round(player.pvMax*0.08)); player.pv=Math.max(1,player.pv-c);
      sceneFloat("hero","-"+c,"#ff7a6a",13); sceneBlood("hero",0.4); frapper(3.4,{imparable:true,label:cap.nom}); } break;
    case "jugement": frapper(1.9,{imparable:true,label:cap.nom}); soigner(player.pvMax*0.28); break;
    case "etendard": combat.buff={force:6,armure:8,esquive:12,tours:4}; sceneAbilityFlash("#e8c05a");
      log("Étendard levé ! Force et défense accrues (4 tours)."); break;
    case "voile": combat.voile=2; combat.critNext=true; sceneAbilityFlash("#8f7ae8"); sceneMagic("hero","#8f7ae8");
      log("Vous vous fondez dans l'ombre..."); break;
    case "estocade": for(let i=0;i<4;i++) if(combat.ennemi.pv>0) frapper(0.75,{label:cap.nom,noPoison:true}); break;
    case "venin": frapper(1.4,{label:cap.nom,noPoison:true});
      if(combat.ennemi.pv>0){ combat.poison={dgt:8+player.esprit*2,tours:3}; log("Un venin violent ronge l'ennemi !"); } break;
    case "meteore": sceneShake(16); magie(22+player.esprit*4.6,cap.nom,"#ff5a2a"); break;
    case "drain": soigner(magie(11+player.esprit*3,cap.nom,"#4ae89a")); break;
    case "benediction": soigner(14+player.esprit*3.2); combat.buff={force:0,armure:8,esquive:0,tours:3};
      sceneAbilityFlash("#4ab8e8"); log("Une aura sacrée vous protège (3 tours)."); break;
  }
  finJoueur();
}
function actionPotion(){ if(player.potions<=0){ log("Plus aucune potion !"); return; }
  player.potions--; soigner(player.potionSoin); finJoueur(); }
function actionFuite(){ if(combat.ennemi.boss){ log("Impossible de fuir le gardien !"); return; }
  const ch=28+player.vitesse*4;
  if(Math.random()*100<ch){ log("Vous fuyez dans les galeries..."); combat.fini=true;
    setTimeout(()=>{ combat=null; run=null; state=S.MAP; },650); }
  else { log("Fuite ratée !"); finJoueur(); }
}
function finJoueur(){ if(combat.ennemi.pv<=0){ victoire(); return; } combat.tour="ennemi"; combat.anim=48; }

function tourEnnemi(){
  const e=combat.ennemi; sceneLunge("enemy");
  // spéciaux de boss
  let coups=1, mult=1, dissipe=false;
  if(e.boss){
    combat.bossTurn++;
    if(e.specialId==="boss_osseux" && player.pv<player.pvMax*0.5) coups=2;
    if(e.specialId==="boss_pretre" && combat.bossTurn%3===0){ dissipe=true; mult=1.6; }
    if(e.specialId==="boss_demon" && e.pv<e.pvMax*0.4){ if(!e.enrage){ e.enrage=true; log("Malphas s'embrase !"); sceneAbilityFlash("#ff5a2a"); } mult=1.5; }
    if(e.specialId==="boss_givre" && Math.random()<0.25) combat.gel=true;
  }
  if(dissipe && combat.buff){ combat.buff=null; log("Anathème ! Vos bénédictions sont dissipées."); }
  for(let k=0;k<coups;k++){
    if(combat.voile>0){ combat.voile--; log("Insaisissable ! Vous esquivez."); sceneFloat("hero","Esquive !","#8f7ae8",14); continue; }
    if(Math.random()*100<esquivePct()){
      log("Vous esquivez l'attaque du "+e.nom+" !"); sceneFloat("hero","Esquive !","#7ae8b0",14);
      // Riposte (duelliste)
      if(player.evolution&&player.evolution.id==="duelliste"&&Math.random()<0.25&&e.pv>0){
        log("Riposte !"); frapper(0.8,{label:"Riposte",noPoison:true}); if(e.pv<=0){ victoire(); return; } }
      continue;
    }
    let d=Math.round((e.degats*mult)+Math.random()*4-armureTotale());
    d=Math.max(1,Math.round(d*(1-reducDegats())));
    if(e.crit&&Math.random()*100<e.crit){ d=Math.round(d*1.8); log("Coup critique ennemi !"); }
    player.pv=Math.max(0,player.pv-d); combat.playerFlash=12; sceneShake(9); sceneHit("hero"); sceneBlood("hero",Math.min(1,d/40));
    sceneFloat("hero","-"+d,"#ff7a6a",17); log("Le "+e.nom+" vous inflige "+d+" dégâts.");
    if(player.pv<=0){ defaite(); return; }
  }
  // fin de tour ennemi -> buffs
  if(combat.buff){ combat.buff.tours--; if(combat.buff.tours<=0){ combat.buff=null; log("Votre aura se dissipe."); } }
  debutJoueur();
}
function debutJoueur(){
  combat.tour="joueur";
  if(combat.gel){ combat.gel=false; log("Vous êtes figé par le gel — tour perdu !"); sceneFloat("hero","Gelé !","#9adfff",15);
    combat.tour="ennemi"; combat.anim=40; return; }
  player.energie=Math.min(energieMax(),player.energie+energieRegen());
  if(player.evolution&&player.evolution.id==="paladin"){ player.pv=Math.min(player.pvMax,player.pv+4); }
  // poison en début de tour
  if(combat.poison&&combat.poison.tours>0){ const e=combat.ennemi,d=combat.poison.dgt;
    e.pv=Math.max(0,e.pv-d); combat.poison.tours--; e.hitFlash=6; sceneBlood("enemy",0.25);
    sceneFloat("enemy","-"+d+" ☠","#7ae86a",14); log("Le poison ronge l'ennemi ("+d+").");
    if(e.pv<=0){ victoire(); return; } }
  // Sylve boss : soigne si on a raté (géré dans frapper via esquive) -> simplifié : régén boss léger
  if(combat.ennemi.boss&&combat.ennemi.specialId==="boss_arbre"&&combat.lastMiss){ combat.ennemi.pv=Math.min(combat.ennemi.pvMax,combat.ennemi.pv+30);
    log("La Sylve se régénère (+30)."); combat.lastMiss=false; }
}
function levelUp(){
  player.xp-=player.xpMax; player.niveau++; player.xpMax=Math.round(player.xpMax*1.55);
  let dpv=7,df=1,dv=1,de=1; const id=player.classe.id;
  if(id==="guerrier"){ df++; dpv+=5; } else if(id==="ombre"){ dv++; dpv+=2; } else { de++; }
  player.pvMax+=dpv; player.force+=df; player.vitesse+=dv; player.esprit+=de;
  player.pv=player.pvMax; player.energie=energieMax();
  return "PV+"+dpv+" FOR+"+df+" VIT+"+dv+" ESP+"+de;
}
function victoire(){
  combat.fini=true; const e=combat.ennemi;
  sceneDeath("enemy"); sceneBlood("enemy",1.4);
  player.or+=e.or; player.xp+=e.xp;
  let heal=player.esprit; if(player.evolution&&player.evolution.id==="necro")heal+=5;
  player.pv=Math.min(player.pvMax,player.pv+heal);
  const msgs=[e.boss?"⚑ GARDIEN VAINCU !":"Victoire !", "Le "+e.nom+" s'effondre.",
    "Butin : "+e.or+" or · "+e.xp+" XP · +"+heal+" PV"];
  while(player.xp>=player.xpMax) msgs.push("★ NIVEAU "+(player.niveau+1)+" — "+levelUp());
  const evolue = player.niveau>=5 && !player.evolution;
  if(evolue) msgs.push("Votre âme s'éveille... choisissez votre voie !");
  setTimeout(()=>{ showMsg(msgs,()=>{ combat=null;
    if(evolue){ evoCursor=0; state=S.EVOLVE; } else finEtage();
  },"combat"); },700);
}
function defaite(){ combat.fini=true; sceneDeath("hero"); setTimeout(()=>{ combat=null; state=S.GAMEOVER; },600); }
function appliquerEvolution(evo){
  player.evolution=evo; player.pvMax+=evo.bonus.pv; player.force+=evo.bonus.force;
  player.vitesse+=evo.bonus.vitesse; player.esprit+=evo.bonus.esprit;
  player.pv=player.pvMax; player.energie=energieMax();
  showMsg(["Vous devenez "+evo.nom+" !","Passif : "+evo.passif.nom+" — "+evo.passif.desc,
    "Capacité : "+evo.capacite.nom],()=>{ if(run) finEtage(); else state=S.MAP; },"map");
}

/* ============================================================================
   MOTEUR DE COMBAT — PixiJS / WebGL  (ambiance Darkest Dungeon)
   ============================================================================ */
let PX=null; // { app, root, layers, hero, enemy, blood[], fx[], floaters[], torches[] }
const GROUND=506, HERO_X=250, ENEMY_X=648;
const texCache={};

function radialTex(size,inner,outer){
  const c=mkCanvas(size,size),g=c.getContext("2d");
  const gr=g.createRadialGradient(size/2,size/2,1,size/2,size/2,size/2);
  gr.addColorStop(0,inner); gr.addColorStop(1,outer); g.fillStyle=gr; g.fillRect(0,0,size,size);
  return PIXI.Texture.from(c);
}
function bloodTex(){
  if(texCache.blood) return texCache.blood;
  const c=mkCanvas(16,16),g=c.getContext("2d");
  const gr=g.createRadialGradient(8,8,0,8,8,8); gr.addColorStop(0,"#ff2a2a"); gr.addColorStop(0.6,"#b31018"); gr.addColorStop(1,"rgba(120,8,12,0)");
  g.fillStyle=gr; g.beginPath(); g.arc(8,8,8,0,7); g.fill();
  return texCache.blood=PIXI.Texture.from(c);
}
function splatTex(){
  if(texCache.splat) return texCache.splat;
  const c=mkCanvas(48,48),g=c.getContext("2d");
  g.fillStyle="#8a0e14";
  for(let i=0;i<10;i++){ const a=Math.random()*7,r=6+Math.random()*14,x=24+Math.cos(a)*r,y=24+Math.sin(a)*r;
    g.beginPath(); g.arc(x,y,1.5+Math.random()*4,0,7); g.fill(); }
  g.beginPath(); g.arc(24,24,10,0,7); g.fill();
  return texCache.splat=PIXI.Texture.from(c);
}
function slashTex(){
  if(texCache.slash) return texCache.slash;
  const c=mkCanvas(120,120),g=c.getContext("2d");
  g.strokeStyle="#ffffff"; g.lineWidth=8; g.lineCap="round";
  g.beginPath(); g.arc(60,60,44,-0.9,0.7); g.stroke();
  g.strokeStyle="rgba(255,255,255,0.4)"; g.lineWidth=18; g.beginPath(); g.arc(60,60,44,-0.8,0.6); g.stroke();
  return texCache.slash=PIXI.Texture.from(c);
}
function fogTex(){
  if(texCache.fog) return texCache.fog;
  const c=mkCanvas(256,256),g=c.getContext("2d");
  for(let i=0;i<40;i++){ const x=Math.random()*256,y=Math.random()*256,r=30+Math.random()*70;
    const gr=g.createRadialGradient(x,y,0,x,y,r); gr.addColorStop(0,"rgba(255,255,255,0.5)"); gr.addColorStop(1,"rgba(255,255,255,0)");
    g.fillStyle=gr; g.beginPath(); g.arc(x,y,r,0,7); g.fill(); }
  return texCache.fog=PIXI.Texture.from(c);
}

function ensurePixi(){
  if(PX) return;
  const app=new PIXI.Application({ view:pixiCanvas, width:W, height:H, backgroundColor:0x05060a, antialias:true, autoStart:false });
  const root=new PIXI.Container(); app.stage.addChild(root);
  const layers={ bg:new PIXI.Container(), decal:new PIXI.Container(), mid:new PIXI.Container(),
    actors:new PIXI.Container(), blood:new PIXI.Container(), fx:new PIXI.Container(),
    light:new PIXI.Container(), vig:new PIXI.Container(), floats:new PIXI.Container() };
  for(const k of ["bg","decal","mid","actors","blood","fx","light","vig","floats"]) root.addChild(layers[k]);

  // étalonnage sombre / contrasté (Darkest Dungeon)
  const cm=new PIXI.ColorMatrixFilter(); cm.contrast(0.28,false); cm.saturate(-0.12,true); cm.brightness(0.94,true);
  root.filters=[cm];

  // vignette
  const vig=new PIXI.Sprite(radialTex(512,"rgba(0,0,0,0)","rgba(0,0,0,0.86)"));
  vig.width=W*1.15; vig.height=H*1.25; vig.anchor.set(0.5); vig.position.set(W/2,H/2); layers.vig.addChild(vig);

  // brume (2 nappes dérivantes)
  const fogs=[];
  for(let i=0;i<2;i++){ const f=new PIXI.TilingSprite(fogTex(),W+200,240);
    f.alpha=0.10; f.y=200+i*120; f.tint=0x8faab0; layers.mid.addChild(f); fogs.push(f); }

  PX={ app, root, layers, hero:null, enemy:null, blood:[], fx:[], floaters:[], torches:[], fogs, bgSprites:[] };
}

function clearContainer(c){ for(let i=c.children.length-1;i>=0;i--){ const ch=c.children[i]; c.removeChild(ch); if(ch.destroy) ch.destroy(); } }

function buildBackground(portal){
  const amb=PORTALS[portal].ambiance;
  clearContainer(PX.layers.bg); clearContainer(PX.layers.light); PX.torches=[];
  // fond peint
  const c=mkCanvas(W,H),g=c.getContext("2d");
  g.fillStyle=lg(g,0,0,0,H,[[0,amb.mur1],[0.6,amb.mur2],[1,"#050506"]]); g.fillRect(0,0,W,GROUND);
  // arches
  for(const ax of [150,450,750]){ g.fillStyle="rgba(0,0,0,0.5)";
    rr(g,ax-70,90,140,300,[70,70,0,0]); g.fill();
    g.strokeStyle="rgba(255,255,255,0.05)"; g.lineWidth=4; rr(g,ax-70,90,140,300,[70,70,0,0]); g.stroke(); }
  // piliers
  for(const px of [300,600]){ g.fillStyle=lg(g,px-30,0,px+30,0,[[0,shade(amb.mur1,1.3)],[0.5,amb.mur1],[1,shade(amb.mur1,0.5)]]);
    g.fillRect(px-30,70,60,340);
    g.fillStyle="rgba(0,0,0,0.35)"; for(let y=90;y<400;y+=26) g.fillRect(px-30,y,60,3);
    g.fillStyle=shade(amb.mur1,1.4); g.fillRect(px-38,60,76,14); g.fillRect(px-38,398,76,12); }
  noise(g,0,0,W,GROUND,0.05,true);
  // sol
  g.fillStyle=lg(g,0,GROUND,0,H,[[0,amb.sol1],[1,amb.sol2]]); g.fillRect(0,GROUND,W,H-GROUND);
  g.strokeStyle="rgba(0,0,0,0.4)"; g.lineWidth=1;
  for(let i=0;i<5;i++){ const y=GROUND+8+i*i*7; g.beginPath(); g.moveTo(0,y); g.lineTo(W,y); g.stroke(); }
  for(let i=-4;i<=12;i++){ g.beginPath(); g.moveTo(450+(i-4)*54,GROUND); g.lineTo(450+(i-4)*150,H); g.stroke(); }
  noise(g,0,GROUND,W,H-GROUND,0.06,true);
  const bg=new PIXI.Sprite(PIXI.Texture.from(c)); PX.layers.bg.addChild(bg);

  // torches (halo lumineux animé)
  const tcol=hx(amb.torche);
  for(const tx of [150,450,750]){
    const halo=new PIXI.Sprite(radialTex(256, "rgba(255,190,110,0.5)","rgba(255,150,60,0)"));
    halo.anchor.set(0.5); halo.position.set(tx,150); halo.width=halo.height=300; halo.tint=tcol;
    halo.blendMode=PIXI.BLEND_MODES.ADD; PX.layers.light.addChild(halo);
    PX.torches.push({halo, x:tx, y:150, base:1});
  }
  // lueur du sol
  const floorGlow=new PIXI.Sprite(radialTex(256,"rgba(255,170,90,0.25)","rgba(255,170,90,0)"));
  floorGlow.anchor.set(0.5); floorGlow.position.set(W/2,GROUND+10); floorGlow.width=W; floorGlow.height=200;
  floorGlow.blendMode=PIXI.BLEND_MODES.ADD; floorGlow.alpha=0.5; PX.layers.light.addChild(floorGlow);
}

function makeActor(paint, faceLeft){
  const tex=PIXI.Texture.from(paint.canvas);
  const sp=new PIXI.Sprite(tex);
  sp.anchor.set(paint.anchorX, paint.anchorY);
  if(faceLeft) sp.scale.x=-1;
  const ds=new PIXI.filters.DropShadowFilter({ distance:2, blur:4, alpha:0.6, color:0x000000, rotation:90 });
  const gl=new PIXI.filters.GlowFilter({ distance:10, outerStrength:0.8, innerStrength:0, color:0xffcaa0, quality:0.3 });
  sp.filters=[ds,gl];
  sp._glow=gl; sp._base={}; 
  return sp;
}

function buildCombatScene(){
  ensurePixi();
  buildBackground(run.portal);
  clearContainer(PX.layers.actors); clearContainer(PX.layers.blood); clearContainer(PX.layers.fx);
  clearContainer(PX.layers.decal); clearContainer(PX.layers.floats);
  PX.blood=[]; PX.fx=[]; PX.floaters=[];

  const hp=paintHero(player.classe.id, player.evolution?player.evolution.couleur:null);
  const hero=makeActor(hp,false); hero.position.set(HERO_X,GROUND); 
  const hscale=Math.min(1.15, 1.05); hero.scale.set(hscale, hscale);
  PX.layers.actors.addChild(hero); PX.hero=hero; hero._hx=HERO_X; hero._hy=GROUND; hero._t=Math.random()*10;

  const ep=paintEnemy(combat.ennemi);
  const enemy=makeActor(ep,true); enemy.position.set(ENEMY_X,GROUND);
  const es=combat.ennemi.boss?1.1:0.98; enemy.scale.set(-es,es);
  PX.layers.actors.addChild(enemy); PX.enemy=enemy; enemy._hx=ENEMY_X; enemy._hy=GROUND; enemy._t=Math.random()*10;
  enemy._flip=-1;
  pixiCanvas.style.display="block";
}

function actorOf(who){ return who==="hero"?PX.hero:PX.enemy; }
function chestY(sp){ return sp._hy-110*Math.abs(sp.scale.y); }

/* ---- HOOKS appelés par la logique de combat ---- */
function sceneLunge(who){ const s=actorOf(who); if(s) s._lunge=14; }
function sceneHit(who){ const s=actorOf(who); if(s){ s._flash=10; } sceneShake(who==="hero"?7:4); }
function sceneShake(n){ shake=Math.max(shake,n); }
function sceneBlood(who,amt){
  if(!PX) return; const s=actorOf(who); if(!s) return;
  const x=s._hx, y=chestY(s), n=Math.round(10+amt*26);
  for(let i=0;i<n;i++){ const sp=new PIXI.Sprite(bloodTex());
    sp.anchor.set(0.5); sp.position.set(x+(Math.random()*20-10), y+(Math.random()*24-12));
    const sz=4+Math.random()*8; sp.width=sp.height=sz;
    const a=(-0.3+Math.random()*-2.4), spd=2+Math.random()*5*amt+1;
    sp._vx=Math.cos(a)*spd*(who==="hero"?-1:1)*(0.6+Math.random()); sp._vy=Math.sin(a)*spd-2;
    sp._life=30+Math.random()*24; sp._max=sp._life; sp._g=0.28; sp._splat=false;
    PX.layers.blood.addChild(sp); PX.blood.push(sp); }
}
function addSplat(x,y){ const sp=new PIXI.Sprite(splatTex()); sp.anchor.set(0.5);
  sp.position.set(x,y); const sz=16+Math.random()*26; sp.width=sp.height=sz; sp.alpha=0.85; sp.rotation=Math.random()*7;
  sp._decay=0.0016; PX.layers.decal.addChild(sp);
  if(PX.layers.decal.children.length>44) PX.layers.decal.removeChildAt(0);
}
function sceneSlash(who){ const s=actorOf(who); if(!s)return; const sp=new PIXI.Sprite(slashTex());
  sp.anchor.set(0.5); sp.position.set(s._hx,chestY(s)); sp.width=sp.height=150*Math.abs(s.scale.y);
  sp.rotation=Math.random()*1-0.5; sp.blendMode=PIXI.BLEND_MODES.ADD; sp._life=10; sp._max=10;
  PX.layers.fx.addChild(sp); PX.fx.push(sp); }
function sceneMagic(who,col){ const s=actorOf(who); if(!s)return; const x=s._hx,y=chestY(s);
  for(let i=0;i<26;i++){ const sp=new PIXI.Sprite(bloodTex()); sp.tint=hx(col); sp.anchor.set(0.5);
    sp.position.set(x,y); const sz=4+Math.random()*7; sp.width=sp.height=sz; sp.blendMode=PIXI.BLEND_MODES.ADD;
    const a=Math.random()*7,spd=1+Math.random()*5; sp._vx=Math.cos(a)*spd; sp._vy=Math.sin(a)*spd-1;
    sp._life=26+Math.random()*20; sp._max=sp._life; sp._g=0.04; sp._splat=true; // splat true => pas de tache
    PX.layers.blood.addChild(sp); PX.blood.push(sp); } }
function sceneHeal(who){ const s=actorOf(who); if(!s)return; const x=s._hx,y=chestY(s);
  for(let i=0;i<18;i++){ const sp=new PIXI.Sprite(bloodTex()); sp.tint=0x6ae87a; sp.anchor.set(0.5);
    sp.position.set(x+(Math.random()*30-15),y+20); sp.width=sp.height=4+Math.random()*5; sp.blendMode=PIXI.BLEND_MODES.ADD;
    sp._vx=(Math.random()-0.5)*0.6; sp._vy=-1-Math.random()*1.6; sp._life=34; sp._max=34; sp._g=-0.01; sp._splat=true;
    PX.layers.blood.addChild(sp); PX.blood.push(sp); } }
function sceneAbilityFlash(col){ const gfx=new PIXI.Graphics(); gfx.beginFill(hx(col),0.28); gfx.drawRect(0,0,W,H); gfx.endFill();
  gfx.blendMode=PIXI.BLEND_MODES.ADD; gfx._life=13; gfx._max=13; PX.layers.fx.addChild(gfx); PX.fx.push(gfx); }
function sceneDeath(who){ const s=actorOf(who); if(!s)return; s._dying=true; }
function sceneFloat(who,t,col,size){ const s=actorOf(who); if(!s)return;
  const style=new PIXI.TextStyle({ fontFamily:"Georgia", fontSize:(size||16)*1.4, fontWeight:"bold",
    fill:col||"#fff", stroke:"#000000", strokeThickness:4 });
  const tx=new PIXI.Text(t,style); tx.anchor.set(0.5); tx.scale.set(0.7);
  tx.position.set(s._hx+(Math.random()*24-12), chestY(s)-30); tx._life=56; tx._max=56;
  PX.layers.floats.addChild(tx); PX.floaters.push(tx); }

/* --------------------- ANIMATION DE LA SCÈNE (par frame) ------------------ */
function updateCombatScene(dt){
  const t=time*0.001;
  // torches
  for(const tr of PX.torches){ const fl=0.82+0.18*Math.sin(t*6+tr.x)+Math.random()*0.05; tr.halo.alpha=fl*0.9;
    tr.halo.scale.set((300/256)*(0.96+0.06*Math.sin(t*7+tr.x))); }
  // brume
  for(let i=0;i<PX.fogs.length;i++){ PX.fogs[i].tilePosition.x -= (0.25+i*0.15); }
  // acteurs
  for(const who of ["hero","enemy"]){
    const s=actorOf(who); if(!s) continue;
    s._t+=dt*0.001; const dir=who==="hero"?1:-1;
    const baseSX = Math.abs(s.scale.x); const flip = who==="hero"?1:-1;
    if(s._dying){ s._death=(s._death||0)+dt*0.02; const p=Math.min(1,s._death);
      s.rotation = flip* p*0.9; s.y = s._hy + p*40; s.alpha=1-p*0.9; s.tint=0xaa4444;
      continue;
    }
    // bob + respiration
    const bob=Math.sin(s._t*2.2)*3; s.y=s._hy+bob;
    const breel=1+Math.sin(s._t*2.2)*0.012; s.scale.y=baseSX*breel;
    // lunge
    if(s._lunge>0){ s._lunge-=dt*0.06; const l=Math.max(0,s._lunge); s.x=s._hx+dir*Math.sin(l/14*Math.PI)*30; }
    else s.x=s._hx;
    // flash à l'impact
    if(s._flash>0){ s._flash-=dt*0.06; s.tint=0xffffff; }
    else s.tint=0xffffff===s.tint?0xffffff:0xffffff, s.tint=0xffffff;
    // lueur pulsée
    if(s._glow) s._glow.outerStrength=0.6+0.3*Math.sin(t*5+ (who==="hero"?0:2));
  }
  // hit flash override (blanc) — appliqué proprement
  for(const who of ["hero","enemy"]){ const s=actorOf(who); if(s&&!s._dying) s.tint = s._flash>0 ? 0xff8888 : 0xffffff; }
  // sang
  for(let i=PX.blood.length-1;i>=0;i--){ const p=PX.blood[i];
    p.x+=p._vx*dt*0.06; p.y+=p._vy*dt*0.06; p._vy+=p._g*dt*0.06; p._life-=dt*0.06;
    p.alpha=Math.max(0,Math.min(1,p._life/p._max));
    if(!p._splat && p.y>=GROUND-2){ addSplat(p.x,GROUND-2+Math.random()*8); p._life=0; }
    if(p._life<=0){ PX.layers.blood.removeChild(p); p.destroy(); PX.blood.splice(i,1); }
  }
  // taches qui sèchent
  for(let i=PX.layers.decal.children.length-1;i>=0;i--){ const d=PX.layers.decal.children[i];
    if(d._decay){ d.alpha-=d._decay*dt; if(d.alpha<=0){ PX.layers.decal.removeChildAt(i); d.destroy(); } } }
  // fx
  for(let i=PX.fx.length-1;i>=0;i--){ const f=PX.fx[i]; f._life-=dt*0.06; f.alpha=Math.max(0,f._life/f._max);
    if(f._life<=0){ PX.layers.fx.removeChild(f); f.destroy(); PX.fx.splice(i,1); } }
  // floaters
  for(let i=PX.floaters.length-1;i>=0;i--){ const f=PX.floaters[i]; f.y-=dt*0.03; f._life-=dt*0.06;
    f.alpha=Math.min(1,f._life/22); if(f._life<=0){ PX.layers.floats.removeChild(f); f.destroy(); PX.floaters.splice(i,1); } }
  // shake sur la scène
  if(shake>0){ PX.root.position.set((Math.random()-0.5)*shake,(Math.random()-0.5)*shake); }
  else PX.root.position.set(0,0);
  PX.app.renderer.render(PX.app.stage);
}

/* --------------------------- UI DE COMBAT (2D) --------------------------- */
function renderCombatUI(){
  ctx.clearRect(0,0,W,H); // laisse voir la scène Pixi dessous
  const e=combat.ennemi;
  // bandeau ennemi
  const bw=360,bx=W/2-bw/2;
  panel2d(bx,16,bw,e.boss?86:66, e.boss?"rgba(232,84,58,0.7)":null);
  txt(e.nom,W/2,40,16,e.boss?"#ff8a5a":"#f0e6d0","center","bold");
  bar2d(bx+24,50,bw-48,13,e.pv/e.pvMax,"#e0463a","#7a1410");
  txt(e.pv+" / "+e.pvMax,W/2,60,10,"#fff","center","bold");
  if(e.boss) txt("⚑ "+e.special, W/2, 78, 10.5, "#e8b06a","center","italic");
  // étage / portail
  txt(PORTALS[run.portal].nom+"  ·  Étage "+run.floor+"/10", W/2, e.boss?100:88, 11, "#b8a878","center");
  // effets
  let ey=e.boss?120:106;
  if(combat.poison&&combat.poison.tours>0){ txt("☠ Poison "+combat.poison.dgt+" ("+combat.poison.tours+"t)",W/2,ey,11,"#7ae86a","center"); ey+=16; }
  if(combat.buff){ txt("✦ Aura +"+combat.buff.force+" FOR / +"+combat.buff.armure+" arm ("+combat.buff.tours+"t)",W/2,ey,11,"#e8c05a","center"); ey+=16; }
  if(combat.voile>0){ txt("◈ Insaisissable ("+combat.voile+"t)",W/2,ey,11,"#8f7ae8","center"); }

  // panneau bas (log + menu)
  const py=486; panel2d(10,py,W-20,H-py-8);
  for(let i=0;i<clog.length;i++) txt(clog[i],28,py+24+i*16,11.5,i===clog.length-1?"#f0e6d0":"#8a7f68");
  if(combat.tour==="joueur"&&!combat.fini){
    const cap=capacite();
    const opts=[["Attaquer","dégâts "+degatsAttaque()+" · double "+doubleChance()+"%"],
      [cap.nom,cap.cout+" én. · "+(player.energie>=cap.cout?"prête":"insuffisant")],
      ["Potion ("+player.potions+")","+"+player.potionSoin+" PV"],
      [e.boss?"Fuir (impossible)":"Fuir",e.boss?"le gardien vous barre la route":(28+player.vitesse*4)+"%"]];
    const mx=W-330;
    for(let i=0;i<4;i++){ const sel=i===combat.cursor,oy=py+16+i*30;
      if(sel) rrf(mx-10,oy-2,318,27,6,"rgba(200,162,74,0.14)");
      txt((sel?"▶ ":"   ")+(i+1)+". "+opts[i][0],mx,oy+13,13.5,sel?"#f0d888":"#b8bcc8","left",sel?"bold":"");
      txt(opts[i][1],mx+18,oy+25,9.5,(i===1&&player.energie<cap.cout)?"#c86a6a":"#8a7f68"); }
  } else if(!combat.fini){ txt("L'ennemi passe à l'attaque...",W-320,py+40,12.5,"#8a7f68","left","italic"); }
  drawHUD();
}

function updateCombat(dt){
  if(combat.playerFlash>0) combat.playerFlash-=dt*0.06;
  if(combat.introT>0) combat.introT-=dt*0.06;
  if(!combat.fini){
    if(combat.tour==="ennemi"){ if(combat.anim>0) combat.anim-=dt*0.06; else tourEnnemi(); }
    else {
      const n=4;
      if(pressed("arrowup","z","w")) combat.cursor=(combat.cursor+n-1)%n;
      if(pressed("arrowdown","s")) combat.cursor=(combat.cursor+1)%n;
      if(pressed("1"))combat.cursor=0; if(pressed("2"))combat.cursor=1; if(pressed("3"))combat.cursor=2; if(pressed("4"))combat.cursor=3;
      if(pressed("e","enter"," ")){ if(combat.cursor===0)actionAttaque(); else if(combat.cursor===1)actionCapacite();
        else if(combat.cursor===2)actionPotion(); else actionFuite(); }
    }
  }
  updateCombatScene(dt);
  renderCombatUI();
}

/* ============================================================================
   CARTE (HUB)  — 5 portails + forgeron + apothicaire, rendu 2D
   ============================================================================ */
let heroTexCanvas=null, heroTexKey="";
function heroCanvas(){ const key=(player.classe?player.classe.id:"?")+"/"+(player.evolution?player.evolution.id:"");
  if(key!==heroTexKey){ heroTexCanvas=paintHero(player.classe.id, player.evolution?player.evolution.couleur:null); heroTexKey=key; }
  return heroTexCanvas; }

const gates=[]; // portails sur la carte
(function(){ const xs=[130,290,450,610,770];
  for(let i=0;i<5;i++) gates.push({ i, x:xs[i], y:150, w:96, h:132, zone:{x:xs[i]-30,y:210,w:60,h:70} });
})();
const shops=[
  { id:"forgeron", x:120, y:360, w:150, h:104, porte:{x:170,y:436,w:44,h:34} },
  { id:"apoth",    x:640, y:360, w:150, h:104, porte:{x:688,y:436,w:44,h:34} },
];
const mapWalls=[ {x:0,y:0,w:W,h:40},{x:0,y:H-30,w:W,h:30},{x:0,y:0,w:20,h:H},{x:W-20,y:0,w:20,h:H} ];
for(const g of gates) mapWalls.push({x:g.x-40,y:60,w:80,h:150});
for(const s of shops) mapWalls.push({x:s.x,y:s.y,w:s.w,h:s.h-6});

function overlap(a,b){ return a.x<b.x+b.w&&a.x+a.w>b.x&&a.y<b.y+b.h&&a.y+a.h>b.y; }
function pbox(nx,ny){ return {x:(nx==null?player.x:nx)-player.w/2,y:(ny==null?player.y:ny)-player.h/2,w:player.w,h:player.h}; }
function collide(nx,ny){ const b=pbox(nx,ny); for(const w of mapWalls) if(overlap(b,w))return true; return false; }
function nearInteract(){
  const b=pbox(), z={x:b.x-12,y:b.y-12,w:b.w+24,h:b.h+24};
  for(const g of gates) if(overlap(z,g.zone)) return {type:"gate",g};
  for(const s of shops) if(overlap(z,s.porte)) return {type:"shop",s};
  return null;
}
function portalUnlocked(i){ return i===0 || player.portailsFinis[i-1] || player.niveau>=PORTALS[i].niveauReq; }

/* décor de la carte (pré-rendu) */
let mapBg=null;
function buildMapBg(){
  const c=mkCanvas(W,H),g=c.getContext("2d");
  g.fillStyle=lg(g,0,0,0,H,[[0,"#242a1e"],[0.6,"#1a1e14"],[1,"#0e100a"]]); g.fillRect(0,0,W,H);
  for(let i=0;i<1400;i++){ g.fillStyle=["rgba(0,0,0,0.12)","rgba(120,140,90,0.05)","rgba(200,180,120,0.03)"][i%3];
    const x=Math.random()*W,y=Math.random()*H,r=2+Math.random()*10; g.beginPath(); g.ellipse(x,y,r,r*0.5,0,0,7); g.fill(); }
  // dalles vers les portails
  g.strokeStyle="#3a3324"; g.lineWidth=26; g.lineCap="round";
  for(const ga of gates){ g.beginPath(); g.moveTo(450,470); g.lineTo(ga.x,250); g.stroke(); }
  g.strokeStyle="#2a2418"; g.lineWidth=18;
  for(const ga of gates){ g.beginPath(); g.moveTo(450,470); g.lineTo(ga.x,250); g.stroke(); }
  noise(g,0,0,W,H,0.05,true);
  mapBg=PIXI?null:null; return c;
}
let mapBgCanvas=null;

function drawGate(ga){
  const p=PORTALS[ga.i], unlocked=portalUnlocked(ga.i), done=player.portailsFinis[ga.i];
  const col=p.couleur, x=ga.x, y=ga.y;
  // socle
  ctx.fillStyle="rgba(0,0,0,0.4)"; ctx.beginPath(); ctx.ellipse(x,y+72,58,14,0,0,7); ctx.fill();
  // arche de pierre
  ctx.fillStyle=lg(ctx,x-48,0,x+48,0,[[0,"#4a4438"],[0.5,"#33301f"],[1,"#201d14"]]);
  rr(ctx,x-48,y-60,96,132,[48,48,6,6]); ctx.fill();
  rr(ctx,x-34,y-46,68,118,[34,34,2,2]);
  ctx.fillStyle="#0a0906"; ctx.fill();
  // énergie du portail
  if(unlocked){
    const g2=rg(ctx,x,y+6,4,x,y+6,52,[[0,shade(col,1.5)],[0.5,col],[1,"rgba(0,0,0,0.9)"]]);
    ctx.save(); rr(ctx,x-34,y-46,68,118,[34,34,2,2]); ctx.clip();
    ctx.fillStyle=g2; ctx.fillRect(x-40,y-52,80,130);
    // volutes animées
    for(let i=0;i<5;i++){ const a=time*0.001+i*1.3; ctx.globalAlpha=0.3;
      ctx.fillStyle=shade(col,1.6); ctx.beginPath();
      ctx.ellipse(x+Math.sin(a)*18,y+10+Math.cos(a*1.3)*30,10,16,a,0,7); ctx.fill(); }
    ctx.globalAlpha=1; ctx.restore();
    // halo
    ctx.save(); ctx.globalCompositeOperation="lighter";
    ctx.fillStyle=rg(ctx,x,y+6,2,x,y+6,70,[[0,shade(col,1.2)],[1,"rgba(0,0,0,0)"]]); ctx.globalAlpha=0.4;
    ctx.beginPath(); ctx.arc(x,y+6,70,0,7); ctx.fill(); ctx.restore();
  } else {
    ctx.fillStyle="rgba(20,22,28,0.95)"; rr(ctx,x-34,y-46,68,118,[34,34,2,2]); ctx.fill();
    // cadenas
    ctx.fillStyle="#6a6252"; rr(ctx,x-10,y+2,20,16,3); ctx.fill();
    ctx.strokeStyle="#6a6252"; ctx.lineWidth=3; ctx.beginPath(); ctx.arc(x,y+2,7,Math.PI,0); ctx.stroke();
  }
  // clef de voûte
  ctx.fillStyle=done?"#e8c05a":shade(col,1.1); ctx.beginPath();
  ctx.moveTo(x-12,y-58); ctx.lineTo(x+12,y-58); ctx.lineTo(x+8,y-46); ctx.lineTo(x-8,y-46); ctx.closePath(); ctx.fill();
  // plaque nom
  ctx.font="bold 11px 'Trebuchet MS'"; const tw=ctx.measureText(p.nom).width+16;
  rrf(x-tw/2,y-82,tw,17,4,"rgba(8,8,10,0.9)"); rrs(x-tw/2,y-82,tw,17,4,unlocked?col:"#555",1);
  txt(p.nom,x,y-70,11,unlocked?"#e8ddc4":"#7a7466","center","bold");
  txt(done?"✓ purifié":(unlocked?"Niv. "+PORTALS[ga.i].niveauReq:"Verrouillé — niv. "+PORTALS[ga.i].niveauReq), x, y+92, 9.5, done?"#8ae87a":(unlocked?"#b8a878":"#8a5a5a"),"center");
}
function drawShopBuilding(s){
  const x=s.x,y=s.y,w=s.w,h=s.h;
  ctx.fillStyle="rgba(0,0,0,0.35)"; ctx.beginPath(); ctx.ellipse(x+w/2,y+h,w*0.55,10,0,0,7); ctx.fill();
  ctx.fillStyle=lg(ctx,x,0,x+w,0,[[0,"#4a4032"],[1,"#2c261c"]]); rr(ctx,x,y+26,w,h-26,4); ctx.fill();
  noise(ctx,x,y+26,w,h-26,0.05,true);
  // toit
  ctx.beginPath(); ctx.moveTo(x-12,y+30); ctx.lineTo(x+w/2,y-12); ctx.lineTo(x+w+12,y+30); ctx.closePath();
  ctx.fillStyle=s.id==="forgeron"?lg(ctx,x,y-12,x,y+30,[[0,"#5a5060"],[1,"#332c3a"]]):lg(ctx,x,y-12,x,y+30,[[0,"#7a5a30"],[1,"#4a3418"]]);
  ctx.fill();
  ctx.fillStyle="#1a1510"; ctx.fillRect(x-12,y+28,w+24,5);
  // fenêtre lueur
  const gl=0.7+0.3*Math.sin(time/300+x);
  ctx.fillStyle="rgba(255,170,70,"+0.5*gl+")"; ctx.beginPath(); ctx.arc(x+(s.id==="forgeron"?w-40:40),y+58,18,0,7); ctx.fill();
  rrf(x+(s.id==="forgeron"?w-52:28),y+48,22,20,3,shade("#ff9a3a",gl));
  // porte
  const d=s.porte; rrf(d.x-3,d.y-3,d.w+6,d.h+3,4,"#16100a"); rrf(d.x,d.y,d.w,d.h,3,lg(ctx,d.x,d.y,d.x,d.y+d.h,[[0,"#5a4426"],[1,"#33240f"]]));
  ctx.fillStyle="#d8b45a"; ctx.beginPath(); ctx.arc(d.x+d.w-7,d.y+d.h/2,2,0,7); ctx.fill();
  // enseigne
  ctx.font="bold 11px 'Trebuchet MS'"; const label=s.id==="forgeron"?"FORGERON":"APOTHICAIRE";
  const tw=ctx.measureText(label).width+16; rrf(x+w/2-tw/2,y-2,tw,16,3,"rgba(8,8,10,0.9)");
  txt(label,x+w/2,y+10,11,"#e8d29a","center","bold");
  // PNJ
  drawNPC(s.id==="forgeron"?x+w/2:x+w/2, y+h+6, s.id);
}
function drawNPC(x,y,type){
  const bob=Math.sin(time/420+x)*1.4; ctx.save(); ctx.translate(x,y+bob);
  ctx.fillStyle="rgba(0,0,0,0.35)"; ctx.beginPath(); ctx.ellipse(0,4,12,4,0,0,7); ctx.fill();
  const c1=type==="apoth"?"#3a6a46":"#6a4a30", c2=type==="apoth"?"#26472e":"#43301c";
  ctx.beginPath(); ctx.moveTo(-9,-18); ctx.lineTo(-11,4); ctx.lineTo(11,4); ctx.lineTo(9,-18); ctx.closePath();
  ctx.fillStyle=lg(ctx,-11,-18,11,4,[[0,c1],[1,c2]]); ctx.fill();
  if(type==="forgeron"){ ctx.fillStyle="#8a8f99"; ctx.beginPath(); ctx.moveTo(-6,-16); ctx.lineTo(-7,2); ctx.lineTo(7,2); ctx.lineTo(6,-16); ctx.closePath(); ctx.fill(); }
  ctx.fillStyle="#e2b68e"; rr(ctx,-6,-30,12,12,4); ctx.fill();
  ctx.fillStyle="#22242c"; ctx.fillRect(-3,-25,2,2); ctx.fillRect(1,-25,2,2);
  if(type==="apoth"){ ctx.fillStyle="#26472e"; ctx.beginPath(); ctx.ellipse(0,-30,9,3,0,0,7); ctx.fill();
    ctx.beginPath(); ctx.moveTo(-5,-30); ctx.lineTo(0,-42); ctx.lineTo(5,-30); ctx.closePath(); ctx.fill(); }
  else { ctx.fillStyle="#7a5a3a"; ctx.fillRect(-6,-22,12,5); }
  ctx.restore();
}
function drawHeroMap(){
  const hp=heroCanvas(); const th=54, tw=th*hp.w/hp.h;
  ctx.save(); ctx.translate(player.x,player.y);
  const bob=player.moving?Math.abs(Math.sin(player.animT*0.015))*-3:0;
  ctx.fillStyle="rgba(0,0,0,0.4)"; ctx.beginPath(); ctx.ellipse(0,2,15,5,0,0,7); ctx.fill();
  if(player.dir==="left") ctx.scale(-1,1);
  ctx.drawImage(hp.canvas, -tw*hp.anchorX, -th*hp.anchorY+bob, tw, th);
  ctx.restore();
}

function renderMap(){
  pixiCanvas.style.display="none";
  if(!mapBgCanvas) mapBgCanvas=buildMapBg();
  ctx.drawImage(mapBgCanvas,0,0);
  for(const ga of gates) drawGate(ga);
  for(const s of shops) drawShopBuilding(s);
  drawHeroMap();
  // vignette
  ctx.fillStyle=rg(ctx,W/2,H/2,H*0.42,W/2,H/2,H*0.82,[[0,"rgba(0,0,0,0)"],[1,"rgba(0,0,0,0.5)"]]); ctx.fillRect(0,0,W,H);
  // invite
  const it=nearInteract();
  if(it&&state===S.MAP){ let lbl;
    if(it.type==="gate") lbl=portalUnlocked(it.g.i)?("E — Entrer : "+PORTALS[it.g.i].nom):("Verrouillé (niveau "+PORTALS[it.g.i].niveauReq+" requis)");
    else lbl=it.s.id==="forgeron"?"E — Forgeron (armes & armures)":"E — Apothicaire (potions)";
    ctx.font="bold 11px 'Trebuchet MS'"; const tw=ctx.measureText(lbl).width+20;
    const bx=Math.max(tw/2+8,Math.min(W-tw/2-8,player.x)), by=player.y-46;
    rrf(bx-tw/2,by-14,tw,20,6,"rgba(8,8,10,0.92)"); rrs(bx-tw/2,by-14,tw,20,6,"rgba(200,162,74,0.6)",1);
    txt(lbl,bx,by,11,"#e8d29a","center","bold"); }
  drawHUD();
}

/* ================================== HUD =================================== */
function drawHUD(){
  ctx.fillStyle="rgba(6,7,11,0.85)"; ctx.fillRect(0,0,W,40); ctx.fillStyle="rgba(200,162,74,0.4)"; ctx.fillRect(0,39,W,1);
  const cc=couleurClasse();
  ctx.fillStyle=rg(ctx,23,20,2,23,20,14,[[0,cc],[1,shade(cc,0.4)]]); ctx.beginPath(); ctx.arc(23,20,12,0,7); ctx.fill();
  ctx.strokeStyle="rgba(200,162,74,0.7)"; ctx.lineWidth=1.5; ctx.beginPath(); ctx.arc(23,20,12,0,7); ctx.stroke();
  txt(String(player.niveau),23,25,13,"#fff","center","bold");
  txt(nomClasse(),42,16,12,"#e8d29a","left","bold"); txt("Niveau "+player.niveau,42,31,9.5,"#8a7f68");
  txt("PV",150,15,10,"#c8bca0"); bar2d(170,6,120,10,player.pv/player.pvMax,"#e0463a","#7a1410"); txt(player.pv+"/"+player.pvMax,230,15,9,"#fff","center","bold");
  txt("ÉN",150,32,10,"#c8bca0"); bar2d(170,23,120,10,player.energie/energieMax(),"#5a9ae0","#274f78"); txt(Math.round(player.energie)+"/"+energieMax(),230,32,9,"#fff","center","bold");
  txt("XP",300,15,10,"#c8bca0"); bar2d(320,6,84,10,player.xp/player.xpMax,"#c8a24a","#7a5c22");
  txt("Potion ×"+player.potions,300,32,10,"#c07ae0");
  txt("FOR "+player.force,430,15,11,"#e88a8a","left","bold"); txt("arm "+armureTotale()+" · dmg "+degatsAttaque(),490,15,9.5,"#9a8f78");
  txt("VIT "+player.vitesse,430,31,11,"#8ae8b0","left","bold"); txt("esq "+esquivePct()+"% · ×2 "+doubleChance()+"%",490,31,9.5,"#9a8f78");
  txt("ESP "+player.esprit,628,15,11,"#a08ae8","left","bold");
  const pa=passifActuel(); if(pa) txt("✦ "+pa.nom,628,31,9.5,"#c8a24a","left","italic");
  ctx.fillStyle="#e8c05a"; ctx.beginPath(); ctx.arc(W-96,14,6,0,7); ctx.fill();
  txt(String(player.or),W-84,19,13,"#e8c05a","left","bold");
}

/* ================================ BOUTIQUE =============================== */
let shop=null,flash="",flashT=0;
function openShop(id){ shop={id, cursor:0, items: id==="forgeron"?smithItems():apothItems()}; state=S.SHOP; }
function acheter(it){
  if(it.locked){ flash="Niveau "+it.niv+" requis !"; flashT=90; return; }
  if(player.or<it.prix){ flash="Pas assez d'or !"; flashT=90; return; }
  player.or-=it.prix;
  if(it.type==="potion"){ player.potions++; player.potionSoin=Math.max(player.potionSoin,it.soin); }
  else if(it.type==="force")player.force++; else if(it.type==="vitesse")player.vitesse++;
  else if(it.type==="esprit")player.esprit++; else if(it.type==="pvmax"){ player.pvMax+=12; player.pv+=12; }
  else if(it.type==="arme")player.arme={nom:it.nom,degats:it.degats};
  else if(it.type==="armure")player.armure={nom:it.nom,defense:it.defense};
  flash="Acheté : "+it.nom; flashT=90;
}
function renderShop(){
  pixiCanvas.style.display="none"; renderMap(); ctx.fillStyle="rgba(5,6,10,0.78)"; ctx.fillRect(0,0,W,H);
  const items=shop.items, ph=118+items.length*42+20, pw=560, px=(W-pw)/2, py=(H-ph)/2;
  panel2d(px,py,pw,ph);
  drawNPC(px+46,py+92,shop.id==="forgeron"?"forgeron":"apoth");
  txt(shop.id==="forgeron"?"LE FORGERON":"L'APOTHICAIRE",W/2+18,py+38,20,"#e8c05a","center","bold");
  txt("Le stock s'enrichit à mesure de votre gloire.",W/2+18,py+58,11,"#9a8f78","center","italic");
  txt("Or : "+player.or+"   ·   ↑↓ choisir · E acheter · X sortir",W/2,py+80,11,"#c8bca0","center");
  for(let i=0;i<items.length;i++){ const it=items[i],sel=i===shop.cursor,iy=py+94+i*42;
    rrf(px+18,iy,pw-36,37,6,sel?"rgba(200,162,74,0.14)":"rgba(255,255,255,0.03)");
    if(sel) rrs(px+18,iy,pw-36,37,6,"rgba(200,162,74,0.8)",1.4);
    const c=it.locked?"#6a6252":(sel?"#fff":"#d0c9b4");
    txt((sel?"▶ ":"   ")+it.nom+(it.locked?"  🔒":""),px+58,iy+16,13,c,"left",sel?"bold":"");
    txt(it.desc,px+58,iy+31,9.5,it.locked?"#5a5446":"#8a7f68");
    const aff=!it.locked&&player.or>=it.prix; txt(it.prix+" or",px+pw-30,iy+23,13,aff?"#e8c05a":"#8a5a4a","right","bold"); }
  if(flashT>0){ flashT--; txt(flash,W/2,py+ph-10,12,flash.indexOf("Acheté")===0?"#8ae8b0":"#e88a8a","center","bold"); }
}

/* ===================== SÉLECTION DE CLASSE / ÉVOLUTION ==================== */
const cardPaintCache={};
function cardPaint(clsId,evoCol,key){ if(!cardPaintCache[key]) cardPaintCache[key]=paintHero(clsId,evoCol); return cardPaintCache[key]; }
function drawCards(titre,sous,cards,cursor,footer){
  pixiCanvas.style.display="none";
  ctx.fillStyle=lg(ctx,0,0,0,H,[[0,"#161208"],[1,"#070604"]]); ctx.fillRect(0,0,W,H);
  ctx.save(); ctx.shadowColor="#c8a24a"; ctx.shadowBlur=22; txt(titre,W/2,50,28,"#e8c05a","center","bold"); ctx.restore();
  txt(sous,W/2,74,13,"#8a7f68","center","italic");
  for(let i=0;i<cards.length;i++){ const c=cards[i],sel=i===cursor,cw=272,gap=14,x=W/2-(cw*3+gap*2)/2+i*(cw+gap),y=sel?92:100,ch=430;
    if(sel){ ctx.save(); ctx.shadowColor=c.col; ctx.shadowBlur=28; rrf(x,y,cw,ch,12,"rgba(22,18,12,0.98)"); ctx.restore(); rrs(x+1.5,y+1.5,cw-3,ch-3,11,c.col,2.2); }
    else { rrf(x,y,cw,ch,12,"rgba(14,12,8,0.94)"); rrs(x+1.5,y+1.5,cw-3,ch-3,11,"rgba(120,110,90,0.4)",1.2); }
    txt(c.nom,x+cw/2,y+28,17,sel?c.col:"#d0c9b4","center","bold"); txt(c.titre,x+cw/2,y+46,11,"#8a7f68","center","italic");
    // portrait peint
    const hp=c.paint, ph=150, pw=ph*hp.w/hp.h; ctx.save();
    rr(ctx,x+16,y+56,cw-32,150,8); ctx.clip();
    ctx.fillStyle=rg(ctx,x+cw/2,y+150,4,x+cw/2,y+150,120,[[0,shade(c.col,0.5)],[1,"rgba(0,0,0,0)"]]); ctx.fillRect(x+16,y+56,cw-32,150);
    ctx.drawImage(hp.canvas, x+cw/2-pw*hp.anchorX, y+206-ph*hp.anchorY, pw, ph); ctx.restore();
    let sy=y+228;
    for(const st of c.stats){ txt(st[0],x+18,sy,11,st[2],"left","bold");
      if(st[1]!=null) bar2d(x+92,sy-9,150,8,st[1],st[2],shade(st[2],0.5)); else txt(st[3],x+92,sy,11,"#e8d29a","left","bold"); sy+=18; }
    rrf(x+14,sy-2,cw-28,44,6,"rgba(200,162,74,0.08)");
    txt("✦ "+c.passif.nom,x+22,sy+13,11,"#c8a24a","left","bold"); wrap(c.passif.desc,x+22,sy+28,cw-44,12,9.5,"#b0a488");
    sy+=52; rrf(x+14,sy-2,cw-28,44,6,"rgba(140,120,200,0.08)");
    txt("⚔ "+c.cap.nom+" ("+c.cap.cout+" én.)",x+22,sy+13,11,"#a08ae8","left","bold"); wrap(c.cap.desc,x+22,sy+28,cw-44,12,9.5,"#b0a488");
    sy+=52; wrap(c.desc,x+cw/2,sy+8,cw-36,13,10,"#8a7f68","center"); }
  txt(footer,W/2,H-34,12,"#c8bca0","center");
  txt("FORCE → armure & dégâts   ·   VITESSE → esquive, double frappe   ·   ESPRIT → magie & énergie",W/2,H-14,10.5,"#7a7058","center");
}
function renderSelect(){ const cards=CLASSES.map(c=>({nom:c.nom,titre:c.titre,col:c.couleur,passif:c.passif,cap:c.capacite,desc:c.desc,
  paint:cardPaint(c.id,null,"c"+c.id),
  stats:[["PV "+c.stats.pv,c.stats.pv/70,"#e0463a"],["FOR "+c.stats.force,c.stats.force/12,"#e88a8a"],["VIT "+c.stats.vitesse,c.stats.vitesse/12,"#8ae8b0"],["ESP "+c.stats.esprit,c.stats.esprit/12,"#a08ae8"]]}));
  drawCards("SHADOW OF THE WARRIOR","Choisissez votre voie — 3 classes, 9 destinées, 5 portails",cards,selCursor,"← → choisir   ·   E commencer"); }
function renderEvolve(){ const evos=EVOLUTIONS[player.classe.id];
  const cards=evos.map(ev=>({nom:ev.nom,titre:"Éveil de "+player.classe.nom,col:ev.couleur,passif:ev.passif,cap:ev.capacite,desc:ev.desc,
    paint:cardPaint(player.classe.id,ev.couleur,"e"+ev.id),
    stats:[["PV",null,"#e0463a","+"+ev.bonus.pv],["FOR",null,"#e88a8a","+"+ev.bonus.force],["VIT",null,"#8ae8b0","+"+ev.bonus.vitesse],["ESP",null,"#a08ae8","+"+ev.bonus.esprit]]}));
  drawCards("L'ÉVEIL","Niveau 5 — votre âme réclame une forme nouvelle",cards,evoCursor,"← → choisir   ·   E embrasser sa destinée"); }

/* ==================== CONFIRMATION PORTAIL / REPOS ======================= */
let pendingPortal=0;
function renderPortalConfirm(){
  pixiCanvas.style.display="none"; renderMap(); ctx.fillStyle="rgba(5,6,10,0.8)"; ctx.fillRect(0,0,W,H);
  const p=PORTALS[pendingPortal], pw=520,ph=300,px=(W-pw)/2,py=(H-ph)/2; panel2d(px,py,pw,ph,p.couleur);
  ctx.save(); ctx.shadowColor=p.couleur; ctx.shadowBlur=20; txt(p.nom,W/2,py+44,24,shade(p.couleur,1.3),"center","bold"); ctx.restore();
  txt("10 étages · un gardien vous attend au bout.",W/2,py+72,12,"#b0a488","center","italic");
  txt("Niveau recommandé : "+p.niveauReq,W/2,py+100,13,player.niveau>=p.niveauReq?"#8ae87a":"#e8a05a","center","bold");
  txt("Créatures : "+p.pool.map(k=>MOBS[k].nom).join(", "),W/2,py+128,10.5,"#9a8f78","center");
  txt("⚑ Gardien : "+p.boss.nom,W/2,py+158,13,"#ff8a5a","center","bold");
  wrap(p.boss.special,W/2,py+180,pw-80,15,11,"#c8a878","center","italic");
  if(player.niveau<p.niveauReq) txt("⚠ Vous êtes sous le niveau recommandé — la mort rôde.",W/2,py+222,11,"#e88a5a","center","bold");
  txt("E — Franchir le portail      X — Reculer",W/2,py+ph-24,13,"#e8d29a","center","bold");
}
function renderRest(){
  pixiCanvas.style.display="none"; ctx.fillStyle=lg(ctx,0,0,0,H,[[0,"#141810"],[1,"#070805"]]); ctx.fillRect(0,0,W,H);
  const p=PORTALS[run.portal]; ctx.save(); ctx.shadowColor="#ff9a3a"; ctx.shadowBlur=24;
  txt("FEU DE CAMP",W/2,120,26,"#ffb45a","center","bold"); ctx.restore();
  // petit feu
  const fx=W/2,fy=210; for(let i=0;i<3;i++){ const s=1-i*0.28; ctx.fillStyle=["#ff6a2a","#ffa03a","#ffe07a"][i];
    ctx.beginPath(); ctx.ellipse(fx,fy-i*8+Math.sin(time/120+i)*2,16*s,26*s,0,0,7); ctx.fill(); }
  ctx.fillStyle="#3a2a18"; ctx.fillRect(fx-26,fy+20,52,8);
  txt(p.nom+" — étage "+run.floor+" purgé.",W/2,280,15,"#e8ddc4","center","bold");
  txt("Prochain : étage "+(run.floor+1)+(run.floor+1>=10?" — LE GARDIEN":""),W/2,306,13,run.floor+1>=10?"#ff8a5a":"#b0a488","center");
  txt("PV "+player.pv+"/"+player.pvMax+"   ·   Or "+player.or,W/2,334,12,"#c8bca0","center");
  panel2d(W/2-230,380,460,90);
  txt("E — Repos (+15% PV) puis descendre à l'étage suivant",W/2,412,12.5,"#8ae87a","center","bold");
  txt("X — Se replier au village (vous gardez or & niveaux)",W/2,438,12.5,"#e8c05a","center","bold");
}

/* ================================ MESSAGE ================================ */
function renderMsg(){ if(msg.fond==="combat"&&combat){ updateCombatScene(0); renderCombatUI(); } else if(combat===null&&run&&state===S.MSG&&msg.fond==="combat"){ } 
  if(!(msg.fond==="combat"&&combat)){ pixiCanvas.style.display="none"; if(run&&msg.fond==="map") renderMap(); else if(player.classe) renderMap(); else { ctx.fillStyle="#0a0806"; ctx.fillRect(0,0,W,H);} }
  ctx.fillStyle="rgba(4,4,8,0.6)"; ctx.fillRect(0,0,W,H);
  const bw=520,bh=64+msg.lignes.length*24,bx=(W-bw)/2,by=(H-bh)/2; panel2d(bx,by,bw,bh);
  for(let i=0;i<msg.lignes.length;i++){ const l=msg.lignes[i],star=l.indexOf("★")===0||l.indexOf("⚑")===0;
    txt(l,W/2,by+34+i*24,i===0?16:13,i===0?"#e8c05a":star?"#ffd34a":"#cfc6b0","center",(i===0||star)?"bold":""); }
  txt("E / Entrée pour continuer",W/2,by+bh-14,10,"#7a7058","center","italic");
}
function renderGameOver(){
  pixiCanvas.style.display="none"; ctx.fillStyle=lg(ctx,0,0,0,H,[[0,"#1a0a0c"],[1,"#050303"]]); ctx.fillRect(0,0,W,H);
  ctx.save(); ctx.shadowColor="#a01824"; ctx.shadowBlur=30; txt("VOUS ÊTES TOMBÉ",W/2,H/2-40,40,"#b8283a","center","bold"); ctx.restore();
  txt(nomClasse()+" · niveau "+player.niveau,W/2,H/2,14,"#c8bca0","center");
  txt("Vous perdez la moitié de votre or, mais votre âme renaît au village.",W/2,H/2+26,12,"#8a7f68","center","italic");
  txt("E / Entrée pour renaître",W/2,H/2+64,15,"#e8c05a","center","bold");
}
function renaitre(){ player.pv=player.pvMax; player.energie=energieMax(); player.or=Math.floor(player.or/2);
  player.x=450; player.y=470; player.dir="down"; run=null; state=S.MAP; }

/* =============================== UPDATES 2D =============================== */
function updateMap(dt){
  let dx=0,dy=0;
  if(held("arrowleft","q","a")){dx--;player.dir="left";} if(held("arrowright","d")){dx++;player.dir="right";}
  if(held("arrowup","z","w")){dy--;player.dir="up";} if(held("arrowdown","s")){dy++;player.dir="down";}
  player.moving=(dx||dy);
  if(player.moving){ const l=Math.hypot(dx,dy)||1,sp=2.5,mx=dx/l*sp,my=dy/l*sp;
    if(!collide(player.x+mx,player.y))player.x+=mx; if(!collide(player.x,player.y+my))player.y+=my; player.animT+=dt; }
  else player.animT=0;
  const it=nearInteract();
  if(it&&pressed("e","enter"," ")){
    if(it.type==="gate"){ if(portalUnlocked(it.g.i)){ pendingPortal=it.g.i; state=S.PORTAL; } else { showMsg(["Portail scellé","Niveau "+PORTALS[it.g.i].niveauReq+" requis (ou purifiez le portail précédent)."],()=>{state=S.MAP;},"map"); } }
    else openShop(it.s.id);
  }
  renderMap();
}
function updateShop(){ const items=shop.items;
  if(pressed("arrowup","z","w"))shop.cursor=(shop.cursor+items.length-1)%items.length;
  if(pressed("arrowdown","s"))shop.cursor=(shop.cursor+1)%items.length;
  if(pressed("e","enter"," "))acheter(items[shop.cursor]);
  if(pressed("escape","x"))state=S.MAP;
  renderShop();
}
function updateSelect(){ if(pressed("arrowleft","q"))selCursor=(selCursor+2)%3; if(pressed("arrowright","d"))selCursor=(selCursor+1)%3;
  if(pressed("e","enter"," ")){ newGame(CLASSES[selCursor]); showMsg(["Bienvenue, "+player.classe.nom+".",
    "Passif : "+player.classe.passif.nom+" — "+player.classe.passif.desc,
    "Franchissez un portail pour combattre. Forgeron et apothicaire vous équiperont."],()=>{state=S.MAP;},"map"); }
  renderSelect();
}
function updateEvolve(){ const e=EVOLUTIONS[player.classe.id];
  if(pressed("arrowleft","q"))evoCursor=(evoCursor+2)%3; if(pressed("arrowright","d"))evoCursor=(evoCursor+1)%3;
  if(pressed("e","enter"," "))appliquerEvolution(e[evoCursor]); renderEvolve(); }
function updatePortal(){ if(pressed("e","enter"," "))entrerPortail(pendingPortal); if(pressed("escape","x"))state=S.MAP;
  if(state===S.PORTAL) renderPortalConfirm(); }
function updateRest(){ if(pressed("e","enter"," ")){ player.pv=Math.min(player.pvMax,player.pv+Math.round(player.pvMax*0.15)); continuerRun(); }
  if(pressed("escape","x")){ run=null; state=S.MAP; }
  if(state===S.REST) renderRest(); }

/* ============================== BOUCLE PRINCIPALE ========================= */
let last=performance.now();
function loop(now){ const dt=Math.min(50,now-last); last=now; time=now;
  if(shake>0) shake-=dt*0.05; if(shake<0)shake=0;
  switch(state){
    case S.SELECT: updateSelect(); break;
    case S.MAP: updateMap(dt); break;
    case S.PORTAL: updatePortal(); break;
    case S.COMBAT: if(combat) updateCombat(dt); else renderMap(); break;
    case S.SHOP: updateShop(); break;
    case S.EVOLVE: updateEvolve(); break;
    case S.REST: updateRest(); break;
    case S.MSG: renderMsg(); if(pressed("e","enter"," ")){ const s=msg.suite; msg.suite=null; if(s)s(); } break;
    case S.GAMEOVER: renderGameOver(); if(pressed("e","enter"," "))renaitre(); break;
  }
  clearJust(); requestAnimationFrame(loop);
}
function boot(){ const l=document.getElementById("loading"); if(l)l.style.display="none"; state=S.SELECT; requestAnimationFrame(loop); }
if(typeof PIXI==="undefined"){ document.getElementById("loading").textContent="Erreur : moteur PixiJS introuvable."; }
else boot();
