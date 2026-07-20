/* ===========================================================================
   store.js — estado client-side (localStorage) + type chart no navegador.
   Time e Histórico ficam no SEU navegador (o site é estático, sem servidor).
   =========================================================================== */
(function (g) {
  "use strict";

  const TEAM_KEY = "npdx.team", HIST_KEY = "npdx.history";
  const MAX_TEAM = 6, MAX_HIST = 60;

  const read = (k, d) => { try { const v = JSON.parse(localStorage.getItem(k)); return v == null ? d : v; } catch { return d; } };
  const write = (k, v) => localStorage.setItem(k, JSON.stringify(v));

  /* ---- Time ---- */
  const getTeam = () => read(TEAM_KEY, []);
  const inTeam = (s) => getTeam().includes(s);
  function addTeam(s) {
    const t = getTeam();
    if (t.includes(s)) return { ok: true, team: t };
    if (t.length >= MAX_TEAM) return { ok: false, error: "O time já tem 6 pokémon." };
    t.push(s); write(TEAM_KEY, t); return { ok: true, team: t };
  }
  function removeTeam(s) { const t = getTeam().filter((x) => x !== s); write(TEAM_KEY, t); return { ok: true, team: t }; }
  const clearTeam = () => write(TEAM_KEY, []);

  /* ---- Histórico ---- */
  const getHistory = () => read(HIST_KEY, []);
  function recordView(s) {
    let h = getHistory().filter((e) => e.slug !== s);
    h.unshift({ slug: s, at: Date.now() });
    if (h.length > MAX_HIST) h = h.slice(0, MAX_HIST);
    write(HIST_KEY, h);
  }
  const clearHistory = () => write(HIST_KEY, []);
  function relTime(ms) {
    const d = Date.now() - ms, m = 6e4, h = 36e5, day = 864e5;
    if (d < m) return "agora mesmo";
    if (d < h) return "há " + Math.floor(d / m) + " min";
    if (d < day) return "há " + Math.floor(d / h) + " h";
    if (d < 2 * day) return "ontem";
    return "há " + Math.floor(d / day) + " dias";
  }

  /* ---- Índice (api/pokemon.json), cacheado ---- */
  let _idx = null;
  async function loadIndex() {
    if (_idx) return _idx;
    const arr = await (await fetch("api/pokemon.json")).json();
    const map = new Map();
    arr.forEach((p) => map.set(p.slug, p));
    _idx = { list: arr, map };
    return _idx;
  }

  /* ---- Type chart (Gen 6+), igual ao internal/typechart ---- */
  const CHART = {
    normal: { rock: 0.5, ghost: 0, steel: 0.5 },
    fire: { fire: 0.5, water: 0.5, grass: 2, ice: 2, bug: 2, rock: 0.5, dragon: 0.5, steel: 2 },
    water: { fire: 2, water: 0.5, grass: 0.5, ground: 2, rock: 2, dragon: 0.5 },
    electric: { water: 2, electric: 0.5, grass: 0.5, ground: 0, flying: 2, dragon: 0.5 },
    grass: { fire: 0.5, water: 2, grass: 0.5, poison: 0.5, ground: 2, flying: 0.5, bug: 0.5, rock: 2, dragon: 0.5, steel: 0.5 },
    ice: { fire: 0.5, water: 0.5, grass: 2, ice: 0.5, ground: 2, flying: 2, dragon: 2, steel: 0.5 },
    fighting: { normal: 2, ice: 2, poison: 0.5, flying: 0.5, psychic: 0.5, bug: 0.5, rock: 2, ghost: 0, dark: 2, steel: 2, fairy: 0.5 },
    poison: { grass: 2, poison: 0.5, ground: 0.5, rock: 0.5, ghost: 0.5, steel: 0, fairy: 2 },
    ground: { fire: 2, electric: 2, grass: 0.5, poison: 2, flying: 0, bug: 0.5, rock: 2, steel: 2 },
    flying: { electric: 0.5, grass: 2, fighting: 2, bug: 2, rock: 0.5, steel: 0.5 },
    psychic: { fighting: 2, poison: 2, psychic: 0.5, dark: 0, steel: 0.5 },
    bug: { fire: 0.5, grass: 2, fighting: 0.5, poison: 0.5, flying: 0.5, psychic: 2, ghost: 0.5, dark: 2, steel: 0.5, fairy: 0.5 },
    rock: { fire: 2, ice: 2, fighting: 0.5, ground: 0.5, flying: 2, bug: 2, steel: 0.5 },
    ghost: { normal: 0, psychic: 2, ghost: 2, dark: 0.5 },
    dragon: { dragon: 2, steel: 0.5, fairy: 0 },
    dark: { fighting: 0.5, psychic: 2, ghost: 2, dark: 0.5, fairy: 0.5 },
    steel: { fire: 0.5, water: 0.5, electric: 0.5, ice: 2, rock: 2, steel: 0.5, fairy: 2 },
    fairy: { fire: 0.5, fighting: 2, poison: 0.5, dragon: 2, dark: 2, steel: 0.5 },
  };
  const eff = (a, d) => (CHART[a] && CHART[a][d] != null ? CHART[a][d] : 1);
  const defMult = (atk, types) => (types || []).reduce((m, t) => m * eff(atk, t), 1);

  const mult = (m) => (m === 0 ? "0" : m === 0.25 ? "¼" : m === 0.5 ? "½" : String(m));
  function multClass(m) {
    if (m === 0) return "m0";
    if (m < 0.5) return "m025";
    if (m < 1) return "m05";
    if (m === 1) return "m1";
    if (m <= 2) return "m2";
    return "m4";
  }

  g.PokeStore = {
    getTeam, inTeam, addTeam, removeTeam, clearTeam,
    getHistory, recordView, clearHistory, relTime,
    loadIndex, eff, defMult, mult, multClass, CHART,
  };
})(window);
