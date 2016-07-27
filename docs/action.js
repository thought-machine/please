document.addEventListener('DOMContentLoaded', function() {
    var elems = document.querySelectorAll('a[name]');
    var elemMap = {}, menuItems = {};
    for (var i = 0; i < elems.length; ++i) {
        elemMap[elems[i].name] = elems[i];
    }
    var menuElems = document.querySelectorAll('.menu .selected a');
    var found = false;
    for (var i = 0; i < menuElems.length; ++i) {
        var href = menuElems[i].href;
        var hash = href.indexOf('#');
        if (hash != -1) {
            var name = href.substr(hash + 1);
            if (name in elemMap) {
                menuItems[name] = menuElems[i];
                found = true;
            }
        }
    }
    if (found) {
        window.addEventListener('scroll', function(e) {
            var maxY = 0;
            var winner = "";
            for (var key in menuItems) {
                var y = elemMap[key].offsetTop;
                if (y < window.scrollY + window.innerHeight/2 && y > maxY) {
                    maxY = y;
                    winner = key;
                }
                menuItems[key].className = "";
            }
            if (winner) {
                menuItems[winner].className = "selected";
            }
        });
    }
}, false);
