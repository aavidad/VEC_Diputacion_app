import { prepararTextosDietas as prepararSinVersion } from "./i18n.js";
import { prepararTextosDietas as prepararVista } from "./i18n.js?v=20260929-i18n-dietas-v1";

// Las vistas importan la URL versionada; algunas pruebas usan la URL directa.
await Promise.all([prepararSinVersion(), prepararVista()]);
