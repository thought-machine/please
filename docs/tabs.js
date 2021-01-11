(() => {
  const setupTabsWidgets = () => {
    const switchTab = (event) => {
      event.preventDefault();

      const tabElement = event.target;
      const tabValue = tabElement.dataset.tab;
      const widget = tabElement.closest('[data-widget="tabs"]');

      widget
        .querySelector(".tabs__tab--selected")
        .classList.remove("tabs__tab--selected");
      tabElement.classList.add("tabs__tab--selected");

      widget
        .querySelector(".tabs__panel--selected")
        .classList.remove("tabs__panel--selected");
      widget
        .querySelector(`[data-panel="${tabValue}"]`)
        .classList.add("tabs__panel--selected");
    };

    document.querySelectorAll('[data-widget="tabs"]').forEach((tabsWidget) => {
      tabsWidget.querySelectorAll('[role="tab"]').forEach((tabElement) => {
        tabElement.onclick = switchTab;
        tabElement.onkeydown = (event) => {
          if (event.code === "Enter" || event.code === "Space") {
            switchTab(event);
          }
        };
      });

      const firstTabElement = tabsWidget.querySelector('[role="tab"]');
      const firstTabValue = firstTabElement.dataset.tab;

      firstTabElement.classList.add("tabs__tab--selected");

      tabsWidget
        .querySelector(`[data-panel="${firstTabValue}"]`)
        .classList.add("tabs__panel--selected");
    });
  };

  document.addEventListener("DOMContentLoaded", () => {
    setupTabsWidgets();
  });
})();
