/* ===========================================================================
   items.js — catálogo de itens do Pixelmon: busca + filtro por categoria +
   ordenação + modal de detalhes (o que faz + de quais Pokémon dropa).
   Carrega api/items.json uma vez e filtra no cliente.
   =========================================================================== */
(function () {
  "use strict";
  const P = window.Poke;
  const els = {
    q: document.getElementById("iq"),
    chips: document.getElementById("iCatChips"),
    sort: document.getElementById("iSort"),
    onlyDrop: document.getElementById("iOnlyDrop"),
    clear: document.getElementById("iClear"),
    count: document.getElementById("iCount"),
    grid: document.getElementById("iGrid"),
    loading: document.getElementById("iLoading"),
    empty: document.getElementById("iEmpty"),
  };

  const state = { q: "", cats: new Set(), sort: "cat", onlyDrop: false };
  let all = [];
  let catOrder = []; // ordem de categoria como veio do servidor (já ranqueada)

  function matches(it) {
    if (state.q && !it.name.toLowerCase().includes(state.q) && !(it.desc || "").toLowerCase().includes(state.q)) return false;
    if (state.cats.size && !state.cats.has(it.category)) return false;
    if (state.onlyDrop && !(it.drops && it.drops.length)) return false;
    return true;
  }

  const bestChance = (it) => (it.drops && it.drops.length ? it.drops[0].chance : -1);
  const sorters = {
    cat: (a, b) => (catOrder.indexOf(a.category) - catOrder.indexOf(b.category)) || a.name.localeCompare(b.name),
    name: (a, b) => a.name.localeCompare(b.name),
    drop: (a, b) => bestChance(b) - bestChance(a) || a.name.localeCompare(b.name),
  };

  function apply() {
    const list = all.filter(matches).sort(sorters[state.sort] || sorters.cat);
    els.count.textContent = list.length + " itens";
    const frag = document.createDocumentFragment();
    for (const it of list) frag.appendChild(P.itemCard(it, P.ItemModal.open));
    els.grid.replaceChildren(frag);
    els.empty.hidden = list.length !== 0;
  }

  function debounce(fn, ms) { let id; return (...a) => { clearTimeout(id); id = setTimeout(() => fn(...a), ms); }; }

  function buildCatChips() {
    // categorias na ordem de primeira aparição (o servidor já ranqueia).
    const seen = [];
    for (const it of all) if (!seen.includes(it.category)) seen.push(it.category);
    catOrder = seen.slice();
    P.buildChips(els.chips, seen.map((c) => ({ value: c, label: P.itemCat(c) })), (c, on) => {
      if (on) state.cats.add(c); else state.cats.delete(c);
      apply();
    });
  }

  function wire() {
    window.addEventListener("npdx:langchange", () => {
      els.chips.querySelectorAll(".chip").forEach((ch) => { ch.textContent = P.itemCat(ch.dataset.val); });
      apply();
    });
    els.q.addEventListener("input", debounce(() => { state.q = els.q.value.trim().toLowerCase(); apply(); }, 90));
    els.sort.addEventListener("change", () => { state.sort = els.sort.value; apply(); });
    els.onlyDrop.addEventListener("change", () => { state.onlyDrop = els.onlyDrop.checked; apply(); });
    els.clear.addEventListener("click", () => {
      state.q = ""; state.cats.clear(); state.sort = "cat"; state.onlyDrop = false;
      els.q.value = ""; els.sort.value = "cat"; els.onlyDrop.checked = false;
      els.chips.querySelectorAll(".chip.is-on").forEach((c) => c.classList.remove("is-on"));
      apply();
    });
  }

  async function init() {
    wire();
    try {
      all = await (await fetch("api/items.json")).json();
    } catch (e) { els.loading.textContent = "Falha ao carregar os itens."; return; }
    if (!all.length) { els.loading.textContent = "Nenhum item na base (rode `import-items`)."; return; }
    buildCatChips();
    els.loading.remove();
    apply();
    els.q.focus();
  }

  init();
})();
