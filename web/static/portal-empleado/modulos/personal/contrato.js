/** Capacidad declarada por el manifiesto real de Personal. */
export const CAPACIDAD_CONSULTAR_EMPLEADO = "personal.empleado.read";
export const ESTADOS_CONSULTA_PERSONAL = Object.freeze(["cargando", "denegado", "error", "no_habilitada"]);
export function validarEstadoConsultaPersonal(estado) { if (!ESTADOS_CONSULTA_PERSONAL.includes(estado)) throw new TypeError("estado de consulta de Personal no válido"); return estado; }
