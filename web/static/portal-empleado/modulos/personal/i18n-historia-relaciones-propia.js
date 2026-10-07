import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const catalogo = await cargarTextos("personal-historia-relaciones");
export const traducirHistoriaRelaciones = (clave, variables) => catalogo.traducir(clave, variables);
export const formatearFechaHistoriaRelaciones = (valor) => new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T12:00:00Z`));
export const formatearInstanteHistoriaRelaciones = (valor) => new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
export const formatearNumeroHistoriaRelaciones = (valor) => new Intl.NumberFormat(LOCALIZACION_ACTUAL).format(valor);
