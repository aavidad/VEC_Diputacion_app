/** Textos del encuadre de consulta y revisión de Dietas. */
export const MENSAJES_REVISION_DIETAS_ES = Object.freeze({
  revision_mis_comisiones: "Mis comisiones",
  revision_subtitulo: "Consulte sus borradores registrados y prepare una nueva comisión.",
  revision_titulo: "Revisión de comisión",
  revision_instruccion: "Seleccione un borrador para comprobar sus datos y el recibo conservado.",
  revision_ayuda_propia: "La consulta muestra únicamente sus borradores autorizados. El cálculo es orientativo; enviar a revisión y adjuntar justificantes requieren sus servicios propios.",
  revision_ayuda_circuito: "Las vistas de jefatura y gestión describen el circuito pendiente. No muestran solicitudes personales ni ejecutan decisiones hasta disponer de consulta y autorización propias.",
  revision_jefatura_sin_servicio: "La bandeja y las decisiones de jefatura aún no están conectadas al servicio autorizado.",
  revision_gestion_sin_servicio: "La liquidación, fiscalización y consulta de pago aún no están conectadas al circuito económico autorizado.",
  revision_accion_sin_servicio: "Acción pendiente del servicio autorizado",
});

export function crearTraductorRevisionDietas(traducirBase) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_REVISION_DIETAS_ES, clave)) return traducirBase(clave, variables);
    return MENSAJES_REVISION_DIETAS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
