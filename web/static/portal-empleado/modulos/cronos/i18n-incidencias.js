import { cargarTextos } from "../../../comun/textos.js";
import { MENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";

export const MENSAJES_CRONOS_INCIDENCIAS = (await cargarTextos("cronos-incidencias")).seccion("incidencias");
const CATALOGO = Object.freeze({ ...MENSAJES_CRONOS_SOLICITUDES, ...MENSAJES_CRONOS_INCIDENCIAS });
const CLAVES = Object.freeze(Object.keys(CATALOGO));

export function crearTraductorIncidenciasCronos(mensajes = CATALOGO) {
  const catalogo = { ...CATALOGO, ...mensajes };
  if (CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo de incidencias Cronos incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave de incidencias Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}
