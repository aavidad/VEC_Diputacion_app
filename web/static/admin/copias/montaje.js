import { montarVistaCopias } from "./vista.js?v=20261001-cs09-copias-ux-v2";
import { TEXTOS_COPIAS, traducirCopias } from "./i18n.js?v=20261001-cs09-copias-ux-v2";

document.documentElement.lang = TEXTOS_COPIAS.idioma;
document.title = traducirCopias("titulo");
montarVistaCopias({ raiz: document.getElementById("copias-admin") });
