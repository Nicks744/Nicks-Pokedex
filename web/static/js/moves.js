/* ===========================================================================
   moves.js — browser de ataques: busca + filtros (tipo, categoria) + ordenação.
   Carrega api/moves.json uma vez e filtra no cliente. Reutiliza os componentes
   compartilhados (Poke.moveCard, Poke.MoveModal, Poke.buildTypeChips).
   =========================================================================== */
(function () {
  "use strict";

  const P = window.Poke;
  const els = {
    q: document.getElementById("mq"),
    typeChips: document.getElementById("mTypeChips"),
    cat: document.getElementById("mCat"),
    sort: document.getElementById("mSort"),
    clear: document.getElementById("mClear"),
    count: document.getElementById("mCount"),
    grid: document.getElementById("mGrid"),
    loading: document.getElementById("mLoading"),
    empty: document.getElementById("mEmpty"),
  };

  const state = { q: "", types: new Set(), cat: "", sort: "name" };
  let all = [];

  function matches(m) {
    if (state.q && !m.name.toLowerCase().includes(state.q) && !m.slug.includes(state.q)) return false;
    if (state.cat && m.category !== state.cat) return false;
    if (state.types.size && !state.types.has(m.type)) return false;
    return true;
  }

  const sorters = {
    name: (a, b) => a.name.localeCompare(b.name),
    power: (a, b) => (b.power || 0) - (a.power || 0) || a.name.localeCompare(b.name),
    accuracy: (a, b) => (b.accuracy || 0) - (a.accuracy || 0) || a.name.localeCompare(b.name),
    pp: (a, b) => (b.pp || 0) - (a.pp || 0) || a.name.localeCompare(b.name),
  };

  function apply() {
    const list = all.filter(matches).sort(sorters[state.sort] || sorters.name);
    els.count.textContent = list.length + " ataques";
    const frag = document.createDocumentFragment();
    for (const m of list) frag.appendChild(P.moveCard(m, P.MoveModal.open));
    els.grid.replaceChildren(frag);
    els.empty.hidden = list.length !== 0;
  }

  function debounce(fn, ms) {
    let id;
    return (...a) => { clearTimeout(id); id = setTimeout(() => fn(...a), ms); };
  }

  function wire() {
    P.buildTypeChips(els.typeChips, (t, on) => {
      if (on) state.types.add(t); else state.types.delete(t);
      apply();
    });
    els.q.addEventListener("input", debounce(() => {
      state.q = els.q.value.trim().toLowerCase();
      apply();
    }, 90));
    els.cat.addEventListener("change", () => { state.cat = els.cat.value; apply(); });
    els.sort.addEventListener("change", () => { state.sort = els.sort.value; apply(); });
    els.clear.addEventListener("click", () => {
      state.q = ""; state.cat = ""; state.sort = "name"; state.types.clear();
      els.q.value = ""; els.cat.value = ""; els.sort.value = "name";
      els.typeChips.querySelectorAll(".chip.is-on").forEach((c) => c.classList.remove("is-on"));
      apply();
    });
  }

  /* deep-link: ?type=fire e ?q=... vindos de badges/nav */
  function applyURLParams() {
    const p = new URLSearchParams(location.search);
    const type = p.get("type");
    if (type && P.TYPES.includes(type)) {
      state.types.add(type);
      const chip = els.typeChips.querySelector('.chip[data-type="' + type + '"]');
      if (chip) chip.classList.add("is-on");
    }
    const q = p.get("q") || els.q.value;
    if (q) { state.q = q.trim().toLowerCase(); els.q.value = q; }
  }

  async function init() {
    wire();
    applyURLParams();
    try {
      all = await (await fetch("api/moves.json")).json();
    } catch (e) {
      els.loading.textContent = "Falha ao carregar os ataques.";
      return;
    }
    els.loading.remove();
    apply();
    els.q.focus();
  }

  init();
})();
