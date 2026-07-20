/* ===========================================================================
   pokemon.js — interações da página de detalhe:
   - botão adicionar/remover do time (fetch, sem recarregar)
   - "Inspecionar" a linha evolutiva (revela as condições nas setas)
   =========================================================================== */
(function () {
  "use strict";

  /* --- adicionar/remover do time --- */
  const btn = document.querySelector(".team-toggle");
  if (btn) {
    btn.addEventListener("click", async () => {
      const slug = btn.dataset.slug;
      const inTeam = btn.classList.contains("is-in");
      const url = inTeam ? "/team/remove" : "/team/add";
      btn.disabled = true;
      try {
        const res = await fetch(url, {
          method: "POST",
          headers: { "X-Requested-With": "fetch", "Content-Type": "application/x-www-form-urlencoded" },
          body: "slug=" + encodeURIComponent(slug),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) { alert(data.error || "Não foi possível atualizar o time."); return; }
        btn.classList.toggle("is-in", !inTeam);
        btn.setAttribute("aria-pressed", String(!inTeam));
      } finally {
        btn.disabled = false;
      }
    });
  }

  /* --- alternar sprite shiny --- */
  const shinyBtn = document.getElementById("shinyToggle");
  const portraitImg = document.querySelector(".poke__portrait .sprite");
  if (shinyBtn && portraitImg) {
    shinyBtn.addEventListener("click", () => {
      const on = shinyBtn.classList.toggle("is-on");
      const m = portraitImg.getAttribute("src").match(/\/sprites\/(?:shiny\/)?(\d+)\.png/);
      if (!m) return;
      portraitImg.src = on ? `/sprites/shiny/${m[1]}.png` : `/sprites/${m[1]}.png`;
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
