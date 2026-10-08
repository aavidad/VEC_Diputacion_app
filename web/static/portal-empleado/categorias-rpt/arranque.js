// La consulta vive dentro del portal; la dirección antigua lleva a ella
// conservando el idioma elegido.
import { prepararIdiomas, resolverIdiomaNavegacion } from "../../comun/idioma.js";

try { await prepararIdiomas(); } catch { /* Se conserva el idioma de respaldo común. */ }
const idioma = resolverIdiomaNavegacion({ ubicacion: window.location });
window.location.replace(`/portal-empleado/?lang=${encodeURIComponent(idioma)}#contratacion-temporal/categorias-rpt`);
