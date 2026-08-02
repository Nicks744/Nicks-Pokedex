/* ===========================================================================
   components.js — biblioteca compartilhada (fonte única de verdade do front)
   - TYPES / cores de tipo / ícones (gems)
   - Componentes de tipo: typeGem, typeBadge, typesRow, buildTypeChips
   - Web Component <type-badge type="grass">
   - Cards: renderCard (Pokémon), moveCard + moveModal (golpes)
   Tudo relativo a <base href> para funcionar no GitHub Pages.
   =========================================================================== */
(function (global) {
  "use strict";

  const TYPE_COLORS = {
    normal: "#A8A77A", fire: "#EE8130", water: "#6390F0", electric: "#F7D02C",
    grass: "#7AC74C", ice: "#96D9D6", fighting: "#C22E28", poison: "#A33EA1",
    ground: "#E2BF65", flying: "#A98FF3", psychic: "#F95587", bug: "#A6B91A",
    rock: "#B6A136", ghost: "#735797", dragon: "#6F35FC", dark: "#705746",
    steel: "#B7B7CE", fairy: "#D685AD",
  };

  const TYPES = [
    "normal", "fire", "water", "electric", "grass", "ice",
    "fighting", "poison", "ground", "flying", "psychic", "bug",
    "rock", "ghost", "dragon", "dark", "steel", "fairy",
  ];

  const CATEGORIES = ["Physical", "Special", "Status"];
  const CAT_PT = { Physical: "Físico", Special: "Especial", Status: "Status" };

  const typeColor = (t) => TYPE_COLORS[t] || "#777";
  const title = (s) => (s ? s[0].toUpperCase() + s.slice(1) : s);
  /* Caminho relativo do gem do tipo (resolvido pelo <base href>). */
  const typeGem = (t) => "types/" + String(t).toLowerCase() + ".png";

  /* --- <TypeIcon /> : só o ícone (gem) do tipo --- */
  function typeIcon(t, size = 14) {
    const img = document.createElement("img");
    img.className = "type__gem";
    img.src = typeGem(t);
    img.alt = "";
    img.width = size; img.height = size;
    img.loading = "lazy";
    return img;
  }

  /* --- <TypeBadge /> : gem + rótulo. `link` (default true) filtra a Pokédex.
     Use link:false quando o pai já for um <a> (evita <a> dentro de <a>). --- */
  function typeBadge(t, opts = {}) {
    const link = opts.link !== false;
    const el = document.createElement(link ? "a" : "span");
    el.className = "type";
    if (link) el.href = "?type=" + t;
    el.style.setProperty("--type-color", typeColor(t));
    const label = document.createElement("span");
    label.className = "type__label";
    label.textContent = title(t);
    el.append(typeIcon(t), label);
    return el;
  }

  /* --- <PokemonTypes /> : linha com os badges dos tipos --- */
  function typesRow(types, opts = {}) {
    const row = document.createElement("div");
    row.className = opts.className || "types-row";
    (types || []).forEach((t) => row.appendChild(typeBadge(t, opts)));
    return row;
  }

  /* --- <FilterChip /> compartilhado + builder dos chips de tipo ---
     Cada chip mostra o gem + a sigla e alterna um Set de tipos ativos. */
  function buildTypeChips(container, onToggle) {
    TYPES.forEach((t) => {
      const chip = document.createElement("button");
      chip.type = "button";
      chip.className = "chip";
      chip.dataset.type = t;
      chip.style.setProperty("--c", typeColor(t));
      chip.append(typeIcon(t, 15));
      const label = document.createElement("span");
      label.textContent = title(t);
      chip.appendChild(label);
      chip.addEventListener("click", () => {
        const on = chip.classList.toggle("is-on");
        onToggle(t, on, chip);
      });
      container.appendChild(chip);
    });
  }

  /* Web Component: <type-badge type="grass"> (light DOM, usa o CSS global).
     Mantido para compatibilidade; delega no factory typeBadge(). */
  class TypeBadge extends HTMLElement {
    connectedCallback() { this.render(); }
    static get observedAttributes() { return ["type", "nolink"]; }
    attributeChangedCallback() { if (this.isConnected) this.render(); }
    render() {
      const t = this.getAttribute("type") || "";
      const link = !this.hasAttribute("nolink");
      this.replaceChildren(typeBadge(t, { link }));
    }
  }
  if (!customElements.get("type-badge")) customElements.define("type-badge", TypeBadge);

  /* --- <PokemonCard /> : card do grid da Pokédex.
     `inTeam` define o estado do botão "+".
     Estrutura: <div.pcard>[<a.pcard__link stretched>…</a>][<button.pcard__add>] */
  function renderCard(p, inTeam) {
    const a = document.createElement("a");
    a.className = "pcard__link";
    a.href = "pokemon/" + p.slug;

    const portrait = document.createElement("div");
    portrait.className = "pcard__portrait";
    if (p.sprite) {
      const img = document.createElement("img");
      img.className = "sprite";
      img.src = p.sprite;
      img.alt = p.name;
      img.loading = "lazy";
      img.width = 96; img.height = 96;
      img.onerror = () => img.classList.add("sprite--missing");
      portrait.appendChild(img);
    }

    // Selo de forma no card (esconde o "Base" genérico; mostra Kanto/Alola/Mega…).
    if (p.form && p.form !== "Base") {
      const tag = document.createElement("span");
      tag.className = "pcard__form" + (p.isForm ? " is-variant" : "");
      tag.textContent = p.form;
      portrait.appendChild(tag);
    }

    const dex = document.createElement("span");
    dex.className = "pcard__dex";
    dex.textContent = "Nº " + String(p.dex).padStart(4, "0");

    const name = document.createElement("span");
    name.className = "pcard__name";
    name.textContent = p.form ? p.name.replace(/\s*\(.*\)\s*$/, "") : p.name;

    // Tipos como spans (o card inteiro já é um link).
    const types = typesRow(p.types, { className: "pcard__types", link: false });

    const foot = document.createElement("div");
    foot.className = "pcard__foot";
    foot.innerHTML = `<span>${p.gen || ""}</span><span>BST ${p.bst}</span>`;

    a.append(portrait, dex, name, types, foot);

    const card = document.createElement("div");
    card.className = "pcard";

    const add = document.createElement("button");
    add.type = "button";
    add.className = "pcard__add" + (inTeam ? " is-in" : "");
    add.dataset.slug = p.slug;
    add.title = inTeam ? "No time — clique para remover" : "Adicionar ao time";
    add.setAttribute("aria-label", add.title);
    add.textContent = inTeam ? "✓" : "＋";

    card.append(a, add);
    return card;
  }

  /* --- helpers de exibição de golpe --- */
  const catClass = (c) => "cat cat--" + String(c || "").toLowerCase();
  const catLabel = (c) => CAT_PT[c] || c || "—";
  const numOrDash = (n) => (n > 0 ? String(n) : "—");
  const accOrInf = (n) => (n > 0 ? String(n) : "∞");

  /* --- <MoveCard /> : card de golpe no browser de ataques. Recebe um move do
     api/moves.json e dispara onOpen(move) ao clicar (abre o modal). --- */
  function moveCard(mv, onOpen) {
    const card = document.createElement("button");
    card.type = "button";
    card.className = "mcard";
    card.dataset.slug = mv.slug;

    const head = document.createElement("div");
    head.className = "mcard__head";
    const name = document.createElement("span");
    name.className = "mcard__name";
    name.textContent = mv.name;
    head.append(name);
    if (mv.type) head.append(typeBadge(mv.type, { link: false }));

    const meta = document.createElement("div");
    meta.className = "mcard__meta";
    const cat = document.createElement("span");
    cat.className = catClass(mv.category);
    cat.textContent = catLabel(mv.category);
    meta.append(cat);
    meta.insertAdjacentHTML("beforeend",
      `<span class="mcard__stat"><b>${numOrDash(mv.power)}</b> Pow</span>` +
      `<span class="mcard__stat"><b>${accOrInf(mv.accuracy)}</b> Prec</span>` +
      `<span class="mcard__stat"><b>${numOrDash(mv.pp)}</b> PP</span>`);

    card.append(head, meta);
    if (mv.desc) {
      const d = document.createElement("p");
      d.className = "mcard__desc";
      d.textContent = mv.desc;
      card.append(d);
    }
    card.addEventListener("click", () => onOpen(mv));
    return card;
  }

  /* --- <MoveDetailsModal /> : diálogo com os detalhes completos do golpe.
     Singleton reutilizável; open(move) preenche e mostra. --- */
  const MoveModal = (function () {
    let root, box, closeBtn, lastFocus;

    function ensure() {
      if (root) return;
      root = document.createElement("div");
      root.className = "modal";
      root.hidden = true;
      root.innerHTML =
        '<div class="modal__backdrop" data-close></div>' +
        '<div class="modal__box" role="dialog" aria-modal="true" aria-label="Detalhes do golpe">' +
        '<button class="modal__close" type="button" aria-label="Fechar" data-close>✕</button>' +
        '<div class="modal__body"></div></div>';
      document.body.appendChild(root);
      box = root.querySelector(".modal__body");
      closeBtn = root.querySelector(".modal__close");
      root.addEventListener("click", (e) => { if (e.target.dataset.close != null) close(); });
      document.addEventListener("keydown", (e) => { if (e.key === "Escape" && !root.hidden) close(); });
    }

    function open(mv) {
      ensure();
      lastFocus = document.activeElement;
      const typeBit = mv.type ? typeBadge(mv.type, { link: false }).outerHTML : "";
      box.innerHTML =
        `<div class="modal__head">` +
        `<h2 class="modal__title">${mv.name}</h2>` +
        `<div class="modal__tags">${typeBit}` +
        `<span class="${catClass(mv.category)}">${catLabel(mv.category)}</span></div></div>` +
        `<div class="stat-tiles stat-tiles--compact">` +
        `<div class="tile"><span class="tile__label">Poder</span><span class="tile__val">${numOrDash(mv.power)}</span></div>` +
        `<div class="tile"><span class="tile__label">Precisão</span><span class="tile__val">${accOrInf(mv.accuracy)}</span></div>` +
        `<div class="tile"><span class="tile__label">PP</span><span class="tile__val">${numOrDash(mv.pp)}</span></div></div>` +
        (mv.desc ? `<p class="modal__desc">${mv.desc}</p>` : "") +
        `<a class="btn btn--sm modal__link" href="move/${mv.slug}">Ver quem aprende →</a>`;
      root.hidden = false;
      document.body.classList.add("modal-open");
      closeBtn.focus();
    }
    function close() {
      if (!root) return;
      root.hidden = true;
      document.body.classList.remove("modal-open");
      if (lastFocus && lastFocus.focus) lastFocus.focus();
    }
    return { open, close };
  })();

  global.Poke = {
    TYPES, TYPE_COLORS, CATEGORIES, CAT_PT,
    typeColor, title, typeGem, typeIcon, typeBadge, typesRow, buildTypeChips,
    renderCard, moveCard, MoveModal, catClass, catLabel,
  };
})(window);
