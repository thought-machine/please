function switchTab(event) {
    let tab = event.target

    let lang = tab.dataset.lang

    let examples = tab.closest(".language-examples")

    examples.querySelector('.tabs .selected').classList.remove("selected")
    tab.classList.add("selected")

    examples.querySelector('.examples .selected').classList.remove("selected")
    examples.querySelector(`.examples > [data-lang="${lang}"]`).classList.add("selected")
}

document.addEventListener("DOMContentLoaded", function(event) {
    document.querySelectorAll(".tabs > li").forEach(tab => {
        tab.onclick = switchTab
    })
});
