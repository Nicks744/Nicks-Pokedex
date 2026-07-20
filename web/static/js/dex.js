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

  const state = { q: "", types: new Set(), gen: "", sort: "dex" };
  let all = [];
  const team = new Set(); // slugs no time (pro botão "+")

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
    for (const p of list) frag.appendChild(renderCard(p, team.has(p.slug)));
    els.grid.replaceChildren(frag);
    els.empty.hidden = list.length !== 0;
  }

  /* --- botão "+" no card: adiciona/remove do time sem sair da página --- */
  async function toggleTeam(slug, btn) {
    const inTeam = team.has(slug);
    const url = inTeam ? "/team/remove" : "/team/add";
    btn.disabled = true;
    try {
      const res = await fetch(url, {
        method: "POST",
        headers: { "X-Requested-With": "fetch", "Content-Type": "application/x-www-form-urlencoded" },
        body: "slug=" + encodeURIComponent(slug),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) { showToast(data.error || "Não foi possível atualizar o time."); return; }
      if (inTeam) team.delete(slug); else team.add(slug);
      const on = team.has(slug);
      btn.classList.toggle("is-in", on);
      btn.textContent = on ? "✓" : "＋";
      btn.title = on ? "No time — clique para remover" : "Adicionar ao time";
      btn.classList.remove("pop"); void btn.offsetWidth; btn.classList.add("pop");
    } finally {
      btn.disabled = false;
    }
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
      const [pokeRes, teamRes] = await Promise.all([fetch("/api/pokemon"), fetch("/api/team")]);
      all = await pokeRes.json();
      const t = await teamRes.json().catch(() => ({ team: [] }));
      (t.team || []).forEach((s) => team.add(s));
    } catch (e) {
      els.loading.textContent = "Falha ao carregar a base. Rodou `go run . import`?";
      return;
    }
    els.loading.remove();
    apply();
    els.q.focus();
  }

  init();
})();
