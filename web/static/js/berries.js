/* ===========================================================================
   berries.js — Berries & Sucos com foco em EV/IV. Reaproveita api/items.json,
   filtrando as categorias relevantes; reusa itemCard + ItemModal.
   =========================================================================== */
(function () {
  "use strict";
  const P = window.Poke;
  const els = {
    q: document.getElementById("bq"),
    chips: document.getElementById("bChips"),
    count: document.getElementById("bCount"),
    grid: document.getElementById("bGrid"),
    loading: document.getElementById("bLoading"),
    empty: document.getElementById("bEmpty"),
  };

  // Categorias desta tela (subconjunto do catálogo), com um chip de atalho cada.
  const CATS = ["Berry", "Suco", "Vitamina / EV", "Pena / Asa (EV)", "IV (Bottle Cap)"];
  const state = { q: "", cats: new Set() };
  let all = [];

  function matches(it) {
    if (state.cats.size ? !state.cats.has(it.category) : false) return false;
    if (state.q && !it.name.toLowerCase().includes(state.q) && !(it.desc || "").toLowerCase().includes(state.q)) return false;
    return true;
  }

  function apply() {
    const rank = (c) => CATS.indexOf(c.category);
    const list = all.filter(matches).sort((a, b) => (rank(a) - rank(b)) || a.name.localeCompare(b.name));
    els.count.textContent = list.length + " itens";
    const frag = document.createDocumentFragment();
    for (const it of list) frag.appendChild(P.itemCard(it, P.ItemModal.open));
    els.grid.replaceChildren(frag);
    els.empty.hidden = list.length !== 0;
  }

  function debounce(fn, ms) { let id; return (...a) => { clearTimeout(id); id = setTimeout(() => fn(...a), ms); }; }

  async function init() {
    els.q.addEventListener("input", debounce(() => { state.q = els.q.value.trim().toLowerCase(); apply(); }, 90));
    try {
      const items = await (await fetch("api/items.json")).json();
      all = items.filter((it) => CATS.includes(it.category));
    } catch (e) { els.loading.textContent = "Falha ao carregar."; return; }
    if (!all.length) { els.loading.textContent = "Nenhuma berry na base (rode `import-items`)."; return; }
    P.buildChips(els.chips, CATS.map((c) => ({ value: c, label: c })), (c, on) => {
      if (on) state.cats.add(c); else state.cats.delete(c);
      apply();
    });
    els.loading.remove();
    apply();
  }

  init();
})();
