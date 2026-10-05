import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const catalogo = await cargarTextos("personal-historia-servicios");
export const traducirHistoriaServicios = (clave, variables) => catalogo.traducir(clave, variables);
export const formatearFechaHistoriaServicios = (valor) => new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T12:00:00Z`));
export const formatearInstanteHistoriaServicios = (valor) => new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
export const formatearNumeroHistoriaServicios = (valor) => new Intl.NumberFormat(LOCALIZACION_ACTUAL).format(valor);
