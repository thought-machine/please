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

    var angle = function(el) {
        var tr = window.getComputedStyle(el, null).getPropertyValue("transform");
        if (tr === "none") {
            return 0;
        }
        var values = tr.split('(')[1];
        values = values.split(')')[0];
        values = values.split(',');
        var a = values[0];
        var b = values[1];
        var c = values[2];
        var d = values[3];
        return Math.round(Math.atan2(b, a) * (180/Math.PI));
    }

    var pics = document.querySelectorAll('.menu-graphic img, .sideimg img');
    var rots = new Array(pics.length);
    for (var i = 0; i < pics.length; ++i) {
        rots[i] = angle(pics[i]);
    }

    var fadeImg = function() {
        var img = document.querySelector('.menu-graphic img');
        if (img) {
            img.style.opacity = Math.max(400 - window.scrollY, 100.0) / 400.0;
        }
    }

    window.addEventListener('scroll', function(e) {
        if (found) {
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
        }
        var rot = window.scrollY / 100;
        for (var i = 0; i < pics.length; ++i) {
            pics[i].style.transform = 'rotate(' + (rot + rots[i]) + 'deg)';
        }
        fadeImg();
    });

    var toggleMenu = function() {
        var elem = document.querySelector('nav > ul');
        if (elem.classList.contains('hide')) {
            elem.classList.remove('hide');
        } else {
            elem.classList.add('hide');
        }
    };

    document.querySelector('nav > p a').addEventListener('click', function(e) {
        toggleMenu();
        e.preventDefault();
        return false;
    });

    if (window.innerWidth < 800) {
        toggleMenu();
    }
    fadeImg();
}, false);
