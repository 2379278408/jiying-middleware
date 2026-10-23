(function () {
  "use strict";
  try {
    var t = localStorage.getItem("relay-theme");
    if (t !== "dark" && t !== "light") t = "dark";
    document.documentElement.setAttribute("data-bs-theme", t);
  } catch (e) {}
})();
