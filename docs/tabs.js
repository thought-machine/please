function switchTab(event) {
    const tab = event.target;
    const lang = tab.dataset.tab;
    const examples = tab.closest(".tab-container");

    examples.querySelector('.tabs .selected').classList.remove("selected");
    tab.classList.add("selected");

    examples.querySelector('.tab-contents > .content.selected').classList.remove("selected");
    examples.querySelector(`.tab-contents > .content[data-tab="${lang}"]`).classList.add("selected");
}

document.addEventListener("DOMContentLoaded", function(event) {
    document.querySelectorAll(".tabs > .tab").forEach(tab => {
        tab.onclick = switchTab;
    });

    document.querySelectorAll(".tabs > .tab:first-child").forEach(tab => {
        tab.classList.add("selected");
    });

    document.querySelectorAll(".tab-contents > .content:first-child").forEach(content => {
        content.classList.add("selected");
    });
});
