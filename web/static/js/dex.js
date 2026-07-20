/* ===========================================================================
   dex.js — Pokédex com busca instantânea + filtros (tipo, geração, ordenação)
   Carrega um índice leve uma vez (/api/pokemon) e filtra no cliente.
   =========================================================================== */
(function () {
  "use strict";

  const { TYPES, typeColor, renderCard } = window.Poke;

  const els = {
    q: document.getElementById("q"),
    typeChips: document.getElementById("typeChips"),
    gen: document.getElementById("gen"),
    sort: document.getElementById("sort"),
    clear: document.getElementById("clearFilters"),
    count: document.getElementById("count"),
    grid: document.getElementById("grid"),
    loading: document.getElementById("loading"),
    empty: document.getElementById("empty"),
  };

  const { PokeStore: S } = window;
  const state = { q: "", types: new Set(), gen: "", sort: "dex" };
  let all = [];

  /* --- filtros de tipo (chips) --- */
  function buildChips() {
    TYPES.forEach((t) => {
      const chip = document.createElement("button");
      chip.type = "button";
      chip.className = "chip";
      chip.textContent = t.toUpperCase();
      chip.style.setProperty("--c", typeColor(t));
      chip.addEventListener("click", () => {
        if (state.types.has(t)) { state.types.delete(t); chip.classList.remove("is-on"); }
        else { state.types.add(t); chip.classList.add("is-on"); }
        apply();
      });
      els.typeChips.appendChild(chip);
    });
  }

  /* --- filtragem + ordenação --- */
  function matches(p) {
    const q = state.q;
    if (q) {
      const byNum = String(p.dex) === q || String(p.dex).padStart(4, "0").includes(q);
      if (!p.name.toLowerCase().includes(q) && !p.slug.includes(q) && !byNum) return false;
    }
    if (state.gen && p.gen !== state.gen) return false;
    if (state.types.size) {
      for (const t of state.types) if (!p.types.includes(t)) return false;
    }
    return true;
  }

  const sorters = {
    dex: (a, b) => a.dex - b.dex,
    name: (a, b) => a.name.localeCompare(b.name),
    bst: (a, b) => b.bst - a.bst,
  };

  function apply() {
    const list = all.filter(matches).sort(sorters[state.sort] || sorters.dex);
    els.count.textContent = list.length + " Pokémon";

    const frag = document.createDocumentFragment();
    for (const p of list) frag.appendChild(renderCard(p, S.inTeam(p.slug)));
    els.grid.replaceChildren(frag);
    els.empty.hidden = list.length !== 0;
  }

  /* --- botão "+" no card: adiciona/remove do time (localStorage) --- */
  function toggleTeam(slug, btn) {
    const res = S.inTeam(slug) ? S.removeTeam(slug) : S.addTeam(slug);
    if (!res.ok) { showToast(res.error || "Não foi possível atualizar o time."); return; }
    const on = S.inTeam(slug);
    btn.classList.toggle("is-in", on);
    btn.textContent = on ? "✓" : "＋";
    btn.title = on ? "No time — clique para remover" : "Adicionar ao time";
    btn.classList.remove("pop"); void btn.offsetWidth; btn.classList.add("pop");
  }

  function showToast(msg) {
    let t = document.getElementById("toast");
    if (!t) { t = document.createElement("div"); t.id = "toast"; t.className = "toast"; document.body.appendChild(t); }
    t.textContent = msg;
    t.classList.add("is-on");
    clearTimeout(showToast._id);
    showToast._id = setTimeout(() => t.classList.remove("is-on"), 2200);
  }

  /* --- debounce para a busca --- */
  function debounce(fn, ms) {
    let id;
    return (...a) => { clearTimeout(id); id = setTimeout(() => fn(...a), ms); };
  }

  function wire() {
    els.q.addEventListener("input", debounce(() => {
      state.q = els.q.value.trim().toLowerCase();
      apply();
    }, 90));
    els.gen.addEventListener("change", () => { state.gen = els.gen.value; apply(); });
    els.sort.addEventListener("change", () => { state.sort = els.sort.value; apply(); });
    els.clear.addEventListener("click", () => {
      state.q = ""; state.gen = ""; state.sort = "dex"; state.types.clear();
      els.q.value = ""; els.gen.value = ""; els.sort.value = "dex";
      els.typeChips.querySelectorAll(".chip.is-on").forEach((c) => c.classList.remove("is-on"));
      apply();
    });

    // clique no "+" do card (delegação): não navega, só mexe no time.
    els.grid.addEventListener("click", (e) => {
      const btn = e.target.closest(".pcard__add");
      if (!btn) return;
      e.preventDefault();
      e.stopPropagation();
      toggleTeam(btn.dataset.slug, btn);
    });
  }

  /* --- deep-link: ?type=grass e ?q=... vindos de badges/nav --- */
  function applyURLParams() {
    const p = new URLSearchParams(location.search);
    const type = p.get("type");
    if (type && TYPES.includes(type)) {
      state.types.add(type);
      const chip = [...els.typeChips.children].find((c) => c.textContent.toLowerCase() === type);
      if (chip) chip.classList.add("is-on");
    }
    const q = p.get("q") || els.q.value;
    if (q) { state.q = q.trim().toLowerCase(); els.q.value = q; }
  }

  async function init() {
    buildChips();
    wire();
    applyURLParams();
    try {
      all = await (await fetch("api/pokemon.json")).json();
    } catch (e) {
      els.loading.textContent = "Falha ao carregar a base.";
      return;
    }
    els.loading.remove();
    apply();
    els.q.focus();
  }

  init();
})();
