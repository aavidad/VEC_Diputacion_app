/**
 * Catálogo cerrado de la interfaz común de estado de entrega.
 * Los textos variables pertenecen a quien monta la vista y se escapan al
 * renderizar; este archivo solo contiene la redacción fija en castellano.
 */
export const MENSAJES_ESTADO_ENTREGA_ES = Object.freeze({
  estado_conectado_etiqueta: "Conectado",
  estado_conectado_encabezado: "Disponible con la conexión actual",
  estado_pendiente_etiqueta: "Pendiente de conexión",
  estado_pendiente_encabezado: "Superficie preparada; conexión pendiente",
  estado_bloqueado_etiqueta: "Bloqueado por una dependencia",
  estado_bloqueado_encabezado: "No disponible hasta resolver una dependencia",
  aria_estado: "Estado de entrega: {estado}",
  pendientes_vacios: "No hay elementos pendientes declarados.",
  pendientes_titulo: "Qué falta para terminarlo",
  fuente_etiqueta: "Fuente:",
  conexion_etiqueta: "Conexión:",
});

export const CLAVES_ESTADO_ENTREGA = Object.freeze(Object.keys(MENSAJES_ESTADO_ENTREGA_ES));

function esCatalogoCerrado(catalogo) {
  if (catalogo === null || typeof catalogo !== "object" || Array.isArray(catalogo)
    || Object.getPrototypeOf(catalogo) !== Object.prototype) return false;
  const claves = Object.keys(catalogo);
  return claves.length === CLAVES_ESTADO_ENTREGA.length
    && CLAVES_ESTADO_ENTREGA.every((clave) => typeof catalogo[clave] === "string" && catalogo[clave].trim() !== "");
}

/** Crea un traductor estricto: no admite catálogos ni claves incompletas. */
export function crearTraductorEstadoEntrega(catalogo = MENSAJES_ESTADO_ENTREGA_ES) {
  if (!esCatalogoCerrado(catalogo)) throw new TypeError("catálogo de estado de entrega incompleto o no permitido");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_ESTADO_ENTREGA_ES, clave)) {
      throw new RangeError(`clave de estado de entrega desconocida: ${String(clave)}`);
    }
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
}
