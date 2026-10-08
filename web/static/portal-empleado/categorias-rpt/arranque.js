// La consulta vive dentro del portal; la dirección antigua lleva a ella
// conservando el idioma elegido.
import { IDIOMA_ACTUAL } from "../../comun/idioma.js";

window.location.replace(`/portal-empleado/?lang=${encodeURIComponent(IDIOMA_ACTUAL)}#contratacion-temporal/categorias-rpt`);
