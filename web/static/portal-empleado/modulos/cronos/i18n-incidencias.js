import { MENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";

export let MENSAJES_CRONOS_INCIDENCIAS;
export function instalarMENSAJES_CRONOS_INCIDENCIAS(mensajes) { MENSAJES_CRONOS_INCIDENCIAS = mensajes; }
export function crearTraductorIncidenciasCronos(mensajes) {
  if (!MENSAJES_CRONOS_SOLICITUDES || !MENSAJES_CRONOS_INCIDENCIAS) throw new Error("catálogo de incidencias Cronos sin preparar");
  const CATALOGO = Object.freeze({ ...MENSAJES_CRONOS_SOLICITUDES, ...MENSAJES_CRONOS_INCIDENCIAS });
  const CLAVES = Object.keys(CATALOGO);
  const catalogo = { ...CATALOGO, ...mensajes };
  if (CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo de incidencias Cronos incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave de incidencias Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}
