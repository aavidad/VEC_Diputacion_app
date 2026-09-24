import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js?v=20260924-dietas-d1d2d4";

/** Extensión D1 del catálogo común de Dietas. */
export const MENSAJES_D1_DIETAS_ES = Object.freeze(Object.fromEntries([
  "d1_titulo",
  "d1_subtitulo",
  "d1_limite",
  "d1_ayuda_etiqueta",
  "d1_ayuda",
  "d1_estado_disponible",
  "d1_estado_no_configurado",
  "d1_empleado",
  "d1_empleado_tarea",
  "d1_administrativo",
  "d1_administrativo_tarea",
  "d1_responsable",
  "d1_responsable_tarea",
  "d1_rrhh",
  "d1_rrhh_tarea",
  "d1_intervencion",
  "d1_intervencion_tarea",
].map((clave) => [clave, MENSAJES_DIETAS_ES[clave]])));

export function crearTraductorD1Dietas(traducirBase = crearTraductorDietas(MENSAJES_DIETAS_ES)) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_D1_DIETAS_ES, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* El catálogo común puede preceder a esta extensión. */ }
    return MENSAJES_D1_DIETAS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_texto, variable) => String(variables[variable] ?? ""));
  };
}
