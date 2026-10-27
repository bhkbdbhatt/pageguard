(() => {
  const header = document.querySelector(".site-header");
  const nav = document.getElementById("nav");
  const navToggle = document.querySelector(".nav-toggle");

  const onScroll = () => {
    if (!header) return;
    header.classList.toggle("is-scrolled", window.scrollY > 8);
  };

  onScroll();
  window.addEventListener("scroll", onScroll, { passive: true });

  const closeNav = () => {
    if (!nav || !navToggle) return;
    nav.classList.remove("is-open");
    navToggle.setAttribute("aria-expanded", "false");
  };

  if (nav && navToggle) {
    navToggle.addEventListener("click", () => {
      const open = nav.classList.toggle("is-open");
      navToggle.setAttribute("aria-expanded", String(open));
    });

    nav.addEventListener("click", (event) => {
      if (event.target.closest("a")) closeNav();
    });

    document.addEventListener("click", (event) => {
      if (!nav.classList.contains("is-open")) return;
      if (nav.contains(event.target) || navToggle.contains(event.target)) return;
      closeNav();
    });

    document.addEventListener("keydown", (event) => {
      if (event.key !== "Escape" || !nav.classList.contains("is-open")) return;
      closeNav();
      navToggle.focus();
    });

    window.addEventListener("resize", () => {
      if (window.innerWidth > 920) closeNav();
    });
  }

  const copyText = async (text) => {
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(text);
        return true;
      } catch {}
    }

    const field = document.createElement("textarea");
    field.value = text;
    field.setAttribute("readonly", "");
    field.style.position = "fixed";
    field.style.top = "0";
    field.style.opacity = "0";
    document.body.appendChild(field);
    field.select();

    let copied = false;
    try {
      copied = document.execCommand("copy");
    } catch {
      copied = false;
    }

    field.remove();
    return copied;
  };

  document.querySelectorAll(".copy[data-copy]").forEach((button) => {
    button.addEventListener("click", async () => {
      const selector = button.getAttribute("data-copy");
      const target = selector ? document.querySelector(selector) : null;
      if (!target) return;

      const label = button.querySelector("span");
      const original = label ? label.textContent : "Copy";
      const copied = await copyText(target.innerText);

      if (label) label.textContent = copied ? "Copied" : "Retry";
      button.classList.toggle("is-copied", copied);

      window.setTimeout(() => {
        if (label) label.textContent = original;
        button.classList.remove("is-copied");
      }, 1600);
    });
  });

  document.querySelectorAll("[data-tabs]").forEach((card) => {
    const tabs = Array.from(card.querySelectorAll('[role="tab"]'));
    const panels = Array.from(card.querySelectorAll('[role="tabpanel"]'));
    const copyButton = card.querySelector(".copy");
    if (!tabs.length || !panels.length) return;

    const activate = (selected) => {
      tabs.forEach((tab) => {
        const active = tab === selected;
        tab.setAttribute("aria-selected", String(active));
        tab.tabIndex = active ? 0 : -1;
        tab.classList.toggle("is-active", active);
      });

      panels.forEach((panel) => {
        const active = panel.id === selected.getAttribute("aria-controls");
        panel.classList.toggle("is-active", active);
        panel.hidden = !active;
      });

      if (copyButton) {
        copyButton.setAttribute("data-copy", `#${selected.getAttribute("aria-controls")}`);
      }
    };

    tabs.forEach((tab, index) => {
      tab.addEventListener("click", () => activate(tab));

      tab.addEventListener("keydown", (event) => {
        let next = -1;
        if (event.key === "ArrowRight") next = (index + 1) % tabs.length;
        if (event.key === "ArrowLeft") next = (index - 1 + tabs.length) % tabs.length;
        if (event.key === "Home") next = 0;
        if (event.key === "End") next = tabs.length - 1;
        if (next < 0) return;
        event.preventDefault();
        tabs[next].focus();
        activate(tabs[next]);
      });
    });

    activate(tabs[0]);
  });

  const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const revealItems = Array.from(document.querySelectorAll(".reveal"));

  if (!("IntersectionObserver" in window) || reducedMotion) {
    revealItems.forEach((item) => item.classList.add("is-visible"));
  } else {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (!entry.isIntersecting) return;
          entry.target.classList.add("is-visible");
          observer.unobserve(entry.target);
        });
      },
      { threshold: 0.12, rootMargin: "0px 0px -40px 0px" }
    );

    revealItems.forEach((item) => observer.observe(item));
  }

  const year = document.getElementById("year");
  if (year) year.textContent = String(new Date().getFullYear());
})();
