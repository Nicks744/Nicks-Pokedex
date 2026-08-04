/* ===========================================================================
   i18n.js — seletor de idioma (PT/EN). Guarda a escolha no localStorage e
   aplica em (1) elementos estáticos com data-i18n-en / data-i18n-pt e
   (2) telas dinâmicas, que ouvem o evento "npdx:langchange" e re-renderizam.
   Por enquanto traduz o que os ATAQUES fazem (descrição) e as categorias.
   Biomas ficam sempre em inglês.
   =========================================================================== */
(function (g) {
  "use strict";
  const KEY = "npdx.lang";
  const DEFAULT = "pt"; // usuário é BR; descrições saem em português por padrão

  const get = () => {
    const v = localStorage.getItem(KEY);
    return v === "en" || v === "pt" ? v : DEFAULT;
  };
  const t = (en, pt) => (get() === "en" ? en : pt);

  /* Traduz os elementos estáticos marcados com data-i18n-en/-pt. */
  function applyStatic(root) {
    const lang = get();
    (root || document).querySelectorAll("[data-i18n-en]").forEach((el) => {
      const val = lang === "en" ? el.dataset.i18nEn : el.dataset.i18nPt;
      if (val != null && val !== "") el.textContent = val;
    });
  }

  /* Atualiza os botões do seletor e o atributo lang do <html>. */
  function refreshUI() {
    const lang = get();
    document.documentElement.setAttribute("lang", lang === "en" ? "en" : "pt-br");
    document.querySelectorAll(".lang-toggle__btn").forEach((b) => {
      const on = b.dataset.lang === lang;
      b.classList.toggle("is-active", on);
      b.setAttribute("aria-pressed", on ? "true" : "false");
    });
  }

  function apply() {
    refreshUI();
    applyStatic(document);
  }

  function set(lang) {
    if (lang !== "en" && lang !== "pt") return;
    localStorage.setItem(KEY, lang);
    apply();
    g.dispatchEvent(new CustomEvent("npdx:langchange", { detail: { lang } }));
  }

  function wire() {
    document.querySelectorAll(".lang-toggle__btn").forEach((b) => {
      b.addEventListener("click", () => set(b.dataset.lang));
    });
    apply();
  }

  g.I18N = { get, set, t, apply, applyStatic };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", wire);
  } else {
    wire();
  }
})(window);
