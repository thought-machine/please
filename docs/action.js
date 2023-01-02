(() => {
  const setupMenu = () => {
    const menuButtonNode = document.querySelector("#menu-button");
    const menuListNode = document.querySelector("#menu-list");

    if (menuButtonNode) {
      menuButtonNode.addEventListener("click", () => {
        if (menuListNode.classList.contains("dn")) {
          menuListNode.classList.remove("dn");
        } else {
          menuListNode.classList.add("dn");
        }
      });
    }
  };

  const fadeNavGraphicOnScroll = () => {
    const img = document.querySelector("#nav-graphic");

    if (img) {
      const fade = () => {
        img.style.opacity = Math.max(400 - window.scrollY, 100.0) / 400.0;
      };

      // Initial trigger
      fade();

      window.addEventListener("scroll", fade, { passive: true });
    }
  };

  const angle = (el) => {
    const t = window.getComputedStyle(el).getPropertyValue("transform");
    if (t === "none") {
      return 0;
    }

    const tValues = t.substring(t.indexOf("(") + 1, t.indexOf(")"));
    const [a, b] = tValues.split(",");

    return Math.round(Math.atan2(b, a) * (180 / Math.PI));
  };

  const rotateOnScroll = (selector) => {
    const els = document.querySelectorAll(selector);
    const rots = Array.from(els, (el) => angle(el));

    window.addEventListener(
      "scroll",
      () => {
        const rot = window.scrollY / 100;

        els.forEach((el, index) => {
          el.style.transform = `rotate(${rot + rots[index]}deg)`;
        });
      },
      { passive: true }
    );
  };

  const setupSpyScroll = () => {
    const targets = document.querySelectorAll(
      "h1[id], h2[id], h3[id], h4[id], h5[id], h6[id], a[id]"
    );

    const idToTargetMapping = {};
    targets.forEach((target) => {
      idToTargetMapping[target.id] = target;
    });

    const sameCurrentPageMenuLinks = document.querySelectorAll(
      `#menu-list a[href^="${window.location.pathname}"]`
    );
    const menuLinksWithAvailableTargets = {};
    sameCurrentPageMenuLinks.forEach((menuLink) => {
      if (menuLink.href.includes("#")) {
        const hash = menuLink.href.substring(menuLink.href.indexOf("#") + 1);
        if (hash in idToTargetMapping) {
          menuLinksWithAvailableTargets[hash] = menuLink;
        }
      }
    });

    if (Object.keys(menuLinksWithAvailableTargets).length > 0) {
      window.addEventListener(
        "scroll",
        () => {
          let maxY = 0;
          let winner;

          for (const hash in menuLinksWithAvailableTargets) {
            // Clear previously set value
            menuLinksWithAvailableTargets[hash].style.color = "";

            const y = idToTargetMapping[hash].offsetTop;
            if (y < window.scrollY + window.innerHeight / 2 && y > maxY) {
              maxY = y;
              winner = hash;
            }
          }
          if (winner) {
            menuLinksWithAvailableTargets[winner].style.color = "white";
          }
        },
        { passive: true }
      );
    }
  };

  document.addEventListener("DOMContentLoaded", () => {
    fadeNavGraphicOnScroll();
    setupMenu();
    rotateOnScroll("#nav-graphic, #side-images img");
    setupSpyScroll();
  });
})();
