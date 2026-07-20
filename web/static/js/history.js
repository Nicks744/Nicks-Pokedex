/* ===========================================================================
   history.js — renderiza o histórico de consultas a partir do localStorage.
   =========================================================================== */
(async function () {
  "use strict";
  const S = window.PokeStore, P = window.Poke;

  const els = {
    wrap: document.getElementById("history"),
    empty: document.getElementById("histEmpty"),
    count: document.getElementById("histCount"),
    clear: document.getElementById("histClear"),
  };

  const { map } = await S.loadIndex();

  function chip(t) {
    const s = document.createElement("span");
    s.className = "type"; s.style.setProperty("--type-color", P.typeColor(t)); s.textContent = P.title(t);
    return s;
  }

  function card({ p, at }) {
    const a = document.createElement("a");
    a.className = "hist-card"; a.href = "pokemon/" + p.slug;

    const port = document.createElement("div");
    port.className = "hist-card__portrait";
    const img = document.createElement("img");
    img.className = "sprite"; img.src = p.sprite; img.alt = p.name; img.loading = "lazy";
    img.width = 96; img.height = 96;
    img.onerror = () => img.classList.add("sprite--missing");
    port.appendChild(img);

    const body = document.createElement("div");
    body.className = "hist-card__body";
    const dex = document.createElement("span");
    dex.className = "hist-card__dex"; dex.textContent = "Nº " + String(p.dex).padStart(4, "0");
    const name = document.createElement("span");
    name.className = "hist-card__name"; name.textContent = p.name;
    const types = document.createElement("div");
    types.className = "hist-card__types";
    (p.types || []).forEach((t) => types.appendChild(chip(t)));
    body.append(dex, name, types);

    const when = document.createElement("span");
    when.className = "hist-card__when"; when.textContent = S.relTime(at);

    a.append(port, body, when);
    return a;
  }

  function render() {
    const hist = S.getHistory().map((e) => ({ p: map.get(e.slug), at: e.at })).filter((x) => x.p);
    els.count.textContent = `(${hist.length})`;
    els.empty.hidden = hist.length > 0;
    els.clear.hidden = hist.length === 0;
    els.wrap.replaceChildren(...hist.map(card));
  }

  els.clear.addEventListener("click", () => {
    if (confirm("Limpar o histórico?")) { S.clearHistory(); render(); }
  });

  render();
})();
