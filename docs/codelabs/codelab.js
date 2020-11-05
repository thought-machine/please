document.addEventListener("DOMContentLoaded", function(event) {
    // TODO(jpoole): not this
    const arrBack = document.querySelector("#arrow-back")
    arrBack.href = "/codelabs"
    arrBack.innerHTML = "<img src=\"/images/please_build_p.png\">"

    const done = document.querySelector("#done")
    done.href = "/codelabs"
});
