import { cargarTextos } from "../../comun/textos.js";
import { montarSelectorIdioma } from "../../comun/idioma.js";
import { montarModulos } from "./vista.js?v=20261002-admin-modulos-v1";

const textos = await cargarTextos("administracion-modulos");
const t = textos.traducir;
document.documentElement.lang = textos.idioma;
document.title = t("general.titulo_documento");
document.querySelectorAll("[data-i18n]").forEach((nodo) => { nodo.textContent = t(nodo.dataset.i18n); });
document.querySelectorAll("[data-i18n-label]").forEach((nodo) => { nodo.setAttribute("aria-label", t(nodo.dataset.i18nLabel)); });
montarSelectorIdioma(document.getElementById("idioma"));
// Entrada autónoma cerrada. La composición autorizada podrá inyectar sus puertos.
const vista = montarModulos(document.getElementById("espacio-trabajo"), { t });
window.addEventListener("pagehide", () => vista.desmontar(), { once: true });
