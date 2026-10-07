
/** Textos del recorrido sin fuente de permisos. No habilita operaciones. */
export let MENSAJES_CRONOS_PERMISOS;
export function instalarMENSAJES_CRONOS_PERMISOS(mensajes) { MENSAJES_CRONOS_PERMISOS = mensajes; }

/** Consulta de pendientes; este catálogo no habilita el registro documental. */
export let MENSAJES_JUSTIFICACION_CRONOS;
export function instalarMENSAJES_JUSTIFICACION_CRONOS(mensajes) { MENSAJES_JUSTIFICACION_CRONOS = mensajes; }

export function crearTraductorJustificacionCronos(mensajes = MENSAJES_JUSTIFICACION_CRONOS) {
  if (!MENSAJES_JUSTIFICACION_CRONOS) throw new TypeError("catálogo de justificación Cronos sin preparar");
  const claves = Object.keys(MENSAJES_JUSTIFICACION_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de justificación Cronos incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de justificación Cronos desconocida");
    return mensajes[clave];
  };
}
