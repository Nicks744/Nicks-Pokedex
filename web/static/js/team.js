/* ===========================================================================
   team.js — renderiza o time e a matriz de cobertura a partir do localStorage.
   =========================================================================== */
(async function () {
  "use strict";
  const S = window.PokeStore, P = window.Poke;

  const els = {
    roster: document.getElementById("teamRoster"),
    empty: document.getElementById("teamEmpty"),
    count: document.getElementById("teamCount"),
    clear: document.getElementById("teamClear"),
    coverage: document.getElementById("teamCoverage"),
    matrix: document.getElementById("teamMatrix"),
  };

  const { map } = await S.loadIndex();

  function sprite(p) {
    const i = document.createElement("img");
    i.className = "sprite"; i.src = p.sprite; i.alt = p.name; i.loading = "lazy";
    i.width = 96; i.height = 96;
    i.onerror = () => i.classList.add("sprite--missing");
    return i;
  }
  // chip de tipo (sem link — o card já leva ao Pokémon): usa o componente único.
  const chip = (t) => P.typeBadge(t, { link: false });

  function rosterCard(p) {
    const card = document.createElement("div");
    card.className = "roster-card";

    // Botão remover: canto superior direito (posicionado via CSS, primeiro no
    // DOM para receber foco cedo na navegação por teclado).
    const rem = document.createElement("button");
    rem.className = "roster-card__rem"; rem.type = "button"; rem.dataset.slug = p.slug;
    rem.title = "Remover do time";
    rem.setAttribute("aria-label", "Remover " + p.name + " do time");
    rem.textContent = "✕";

    const id = document.createElement("div");
    id.className = "roster-card__id"; id.textContent = "Nº " + String(p.dex).padStart(4, "0");

    const port = document.createElement("div");
    port.className = "roster-card__portrait"; port.appendChild(sprite(p));

    const name = document.createElement("a");
    name.className = "roster-card__name"; name.href = "pokemon/" + p.slug; name.textContent = p.name;

    const types = document.createElement("div");
    types.className = "roster-card__types";
    (p.types || []).forEach((t) => types.appendChild(chip(t)));

    card.append(rem, id, port, name, types);
    return card;
  }

  function renderMatrix(team) {
    const thead = document.createElement("thead");
    const hr = document.createElement("tr");
    hr.innerHTML = '<th class="matrix__corner">Ataque</th>';
    team.forEach((p) => {
      const th = document.createElement("th");
      th.className = "matrix__member";
      th.innerHTML = `<a href="pokemon/${p.slug}">${p.name}</a>`;
      hr.appendChild(th);
    });
    hr.insertAdjacentHTML("beforeend", '<th class="matrix__count">Fracos</th>');
    thead.appendChild(hr);

    const tbody = document.createElement("tbody");
    P.TYPES.forEach((atk) => {
      const tr = document.createElement("tr");
      const td = document.createElement("td");
      td.className = "matrix__type"; td.appendChild(chip(atk));
      tr.appendChild(td);

      let weak = 0;
      team.forEach((p) => {
        const m = S.defMult(atk, p.types);
        if (m > 1) weak++;
        const c = document.createElement("td");
        c.className = "cell " + S.multClass(m);
        c.textContent = S.mult(m) + "×";
        tr.appendChild(c);
      });

      const wc = document.createElement("td");
      wc.className = "wc" + (weak >= 3 ? " wc--hot" : weak >= 2 ? " wc--warn" : "");
      wc.textContent = String(weak);
      tr.appendChild(wc);
      tbody.appendChild(tr);
    });

    els.matrix.replaceChildren(thead, tbody);
  }

  function render() {
    const team = S.getTeam().map((s) => map.get(s)).filter(Boolean);
    els.count.textContent = `(${team.length}/6)`;
    els.empty.hidden = team.length > 0;
    els.clear.hidden = team.length === 0;
    els.coverage.hidden = team.length === 0;
    els.roster.replaceChildren(...team.map(rosterCard));
    if (team.length) renderMatrix(team);
  }

  els.clear.addEventListener("click", () => {
    if (confirm("Esvaziar o time?")) { S.clearTeam(); render(); }
  });
  els.roster.addEventListener("click", (e) => {
    const b = e.target.closest(".roster-card__rem");
    if (b) { S.removeTeam(b.dataset.slug); render(); }
  });

  render();
})();
