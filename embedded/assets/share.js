/*
 * share.js — optional progressive enhancement. Pages work without it.
 *   - copy button on every <pre>
 *   - syntax highlighting (highlight.js from CDN, loaded only when a
 *     <code class="language-*"> exists; colors come from --syn-* in share.css)
 *   - dark / light theme toggle (persisted in localStorage)
 *   - wrap wide tables for horizontal scroll
 * Include with: <script src="../../assets/share.js" defer></script>
 */
(() => {
  const HLJS = "https://cdn.jsdelivr.net/npm/@highlightjs/cdn-assets@11.11.1/highlight.min.js";
  const THEME_KEY = "share-color-scheme";

  function addCopyButtons() {
    document.querySelectorAll("pre").forEach((pre) => {
      if (pre.querySelector(".copy-btn")) return;
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "copy-btn";
      btn.textContent = "Copy";
      btn.addEventListener("click", async () => {
        const code = pre.querySelector("code") || pre;
        try {
          await navigator.clipboard.writeText(code.innerText.replace(/\n$/, ""));
          btn.textContent = "Copied";
        } catch {
          btn.textContent = "Failed";
        }
        setTimeout(() => (btn.textContent = "Copy"), 1500);
      });
      pre.appendChild(btn);
    });
  }

  function highlight() {
    const blocks = document.querySelectorAll('pre code[class*="language-"]');
    if (!blocks.length) return;
    const s = document.createElement("script");
    s.src = HLJS;
    s.onload = () => blocks.forEach((el) => window.hljs.highlightElement(el));
    document.head.appendChild(s);
  }

  function wrapTables() {
    document.querySelectorAll(".page table").forEach((table) => {
      if (table.parentElement?.classList.contains("table-wrap")) return;
      const wrap = document.createElement("div");
      wrap.className = "table-wrap";
      table.parentNode.insertBefore(wrap, table);
      wrap.appendChild(table);
    });
  }

  function currentMode() {
    const theme = document.documentElement.getAttribute("data-theme");
    if (theme === "dark" || theme === "terminal") return "dark";
    if (theme === "paper") return "light";
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  function applyMode(mode, authoredTheme) {
    const root = document.documentElement;
    if (mode === "dark") {
      root.setAttribute("data-theme", authoredTheme === "terminal" ? "terminal" : "dark");
    } else {
      root.setAttribute("data-theme", "paper");
    }
  }

  function syncToggleLabel(btn) {
    const mode = currentMode();
    const next = mode === "dark" ? "light" : "dark";
    btn.setAttribute("aria-label", `Switch to ${next} theme`);
    btn.title = `Switch to ${next} theme`;
    btn.dataset.mode = mode;
    btn.textContent = mode === "dark" ? "☀" : "☾";
  }

  function initThemeToggle() {
    const root = document.documentElement;
    const authoredTheme = root.getAttribute("data-theme");
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === "light" || saved === "dark") {
      applyMode(saved, authoredTheme);
    }

    if (document.querySelector(".theme-toggle")) return;

    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "theme-toggle";
    syncToggleLabel(btn);
    btn.addEventListener("click", () => {
      const next = currentMode() === "dark" ? "light" : "dark";
      applyMode(next, authoredTheme);
      localStorage.setItem(THEME_KEY, next);
      syncToggleLabel(btn);
    });
    document.body.appendChild(btn);
  }

  function init() {
    addCopyButtons();
    highlight();
    wrapTables();
    initThemeToggle();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
