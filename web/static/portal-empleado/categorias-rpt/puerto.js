/** La escritura queda sin implementación hasta conectar su circuito autorizado. */
export function crearPuertoCategorias({ listarOpciones, solicitarAlta = null, solicitarDeshabilitacion = null } = {}) {
  if (typeof listarOpciones !== "function"
    || (solicitarAlta !== null && typeof solicitarAlta !== "function")
    || (solicitarDeshabilitacion !== null && typeof solicitarDeshabilitacion !== "function")) {
    throw new TypeError("puerto de categorías no disponible");
  }
  return Object.freeze({ listarOpciones, solicitarAlta, solicitarDeshabilitacion });
}
