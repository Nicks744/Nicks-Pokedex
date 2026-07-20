/* ===========================================================================
   components.js — biblioteca compartilhada (fonte única de verdade do front)
   - TYPES / cores de tipo
   - Web Component <type-badge type="grass">
   - helpers de render de card
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

  const typeColor = (t) => TYPE_COLORS[t] || "#777";
  const title = (s) => (s ? s[0].toUpperCase() + s.slice(1) : s);

  /* Web Component: <type-badge type="grass"> (light DOM, usa o CSS global) */
  class TypeBadge extends HTMLElement {
    connectedCallback() { this.render(); }
    static get observedAttributes() { return ["type"]; }
    attributeChangedCallback() { if (this.isConnected) this.render(); }
    render() {
      const t = this.getAttribute("type") || "";
      const a = document.createElement("a");
      a.className = "type";
      a.href = "?type=" + t;
      a.style.setProperty("--type-color", typeColor(t));
      a.textContent = title(t);
      this.replaceChildren(a);
    }
  }
  if (!customElements.get("type-badge")) customElements.define("type-badge", TypeBadge);

  /* Card de Pokémon do grid. `inTeam` define o estado do botão "+".
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

    const dex = document.createElement("span");
    dex.className = "pcard__dex";
    dex.textContent = "Nº " + String(p.dex).padStart(4, "0");

    const name = document.createElement("span");
    name.className = "pcard__name";
    name.textContent = p.name;

    const types = document.createElement("div");
    types.className = "pcard__types";
    (p.types || []).forEach((t) => {
      const b = document.createElement("type-badge");
      b.setAttribute("type", t);
      types.appendChild(b);
    });

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

  global.Poke = { TYPES, TYPE_COLORS, typeColor, title, renderCard };
})(window);
