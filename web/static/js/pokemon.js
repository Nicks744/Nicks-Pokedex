/* ===========================================================================
   pokemon.js — interações da página de detalhe:
   - registra a visita no histórico (localStorage)
   - botão adicionar/remover do time (localStorage)
   - alternar sprite shiny (normal/shiny via data-attrs, funciona com formas)
   - carrossel da linha evolutiva (setas ◀ ▶) + "Ver condições"
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

  /* --- alternar sprite shiny (usa data-normal/data-shiny) --- */
  const shinyBtn = document.getElementById("shinyToggle");
  const hero = document.getElementById("heroSprite");
  if (shinyBtn && hero) {
    shinyBtn.addEventListener("click", () => {
      const on = shinyBtn.classList.toggle("is-on");
      const next = on ? hero.dataset.shiny : hero.dataset.normal;
      if (next) hero.src = next;
      shinyBtn.title = on ? "Ver normal" : "Ver shiny";
    });
  }

  /* --- carrossel da linha evolutiva --- */
  const strip = document.getElementById("evoStrip");
  if (strip) {
    const cells = [...strip.querySelectorAll(".evo-cell")];
    const cur = cells.findIndex((c) => c.classList.contains("is-current"));

    // deixa a etapa atual visível no strip
    const current = cells[cur < 0 ? 0 : cur];
    if (current) current.scrollIntoView({ block: "nearest", inline: "center" });

    const go = (i) => { if (cells[i]) window.location.href = cells[i].getAttribute("href"); };
    const prev = document.querySelector(".focus-nav--prev");
    const next = document.querySelector(".focus-nav--next");
    if (prev) { prev.disabled = cur <= 0; prev.addEventListener("click", () => go(cur - 1)); }
    if (next) { next.disabled = cur < 0 || cur >= cells.length - 1; next.addEventListener("click", () => go(cur + 1)); }

    // "Ver condições": revela a condição de evolução em cada célula
    const insp = document.getElementById("evoInspect");
    if (insp) {
      insp.addEventListener("click", () => {
        const conds = strip.querySelectorAll(".evo-cell__cond");
        const on = insp.classList.toggle("is-active");
        conds.forEach((c) => { c.hidden = !on; });
        strip.classList.toggle("is-inspecting", on);
        insp.textContent = on ? "Ocultar" : "Ver condições";
      });
    }
  }
})();
