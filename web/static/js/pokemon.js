/* ===========================================================================
   pokemon.js — interações da página de detalhe:
   - registra a visita no histórico (localStorage)
   - botão adicionar/remover do time (localStorage)
   - alternar sprite shiny
   - "Inspecionar" a linha evolutiva
   =========================================================================== */
(function () {
  "use strict";
  const S = window.PokeStore;

  const btn = document.querySelector(".team-toggle");
  const slug = btn ? btn.dataset.slug : null;

  /* registra no histórico assim que a página abre */
  if (slug) S.recordView(slug);

  /* --- adicionar/remover do time --- */
  if (btn) {
    const sync = () => {
      const on = S.inTeam(slug);
      btn.classList.toggle("is-in", on);
      btn.setAttribute("aria-pressed", String(on));
    };
    sync();
    btn.addEventListener("click", () => {
      const res = S.inTeam(slug) ? S.removeTeam(slug) : S.addTeam(slug);
      if (!res.ok) { alert(res.error || "Não foi possível atualizar o time."); return; }
      sync();
    });
  }

  /* --- alternar sprite shiny --- */
  const shinyBtn = document.getElementById("shinyToggle");
  const portraitImg = document.querySelector(".poke__portrait .sprite");
  if (shinyBtn && portraitImg) {
    shinyBtn.addEventListener("click", () => {
      const on = shinyBtn.classList.toggle("is-on");
      const m = portraitImg.getAttribute("src").match(/sprites\/(?:shiny\/)?(\d+)\.png/);
      if (!m) return;
      portraitImg.src = on ? `sprites/shiny/${m[1]}.png` : `sprites/${m[1]}.png`;
      shinyBtn.title = on ? "Ver normal" : "Ver shiny";
    });
  }

  /* --- inspecionar linha evolutiva --- */
  const insp = document.getElementById("evoInspect");
  const chain = document.getElementById("evoChain");
  if (insp && chain) {
    insp.addEventListener("click", () => {
      const on = chain.classList.toggle("is-inspecting");
      insp.textContent = on ? "Ocultar" : "Inspecionar";
      insp.classList.toggle("is-active", on);
    });
  }
})();
