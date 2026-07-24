const NOMBRES_OPERACION = Object.freeze(new Set([
  "carga", "detalle", "guardado", "postguardado", "cas",
]));

function validarNombre(nombre) {
  if (!NOMBRES_OPERACION.has(nombre)) throw new TypeError("operación de borradores no válida");
}

/**
 * Arbitra cancelación y generaciones sin conocer datos ni decisiones de UI.
 * Una revocación invalida incluso respuestas de dobles que ignoren la señal.
 */
export function crearCoordinadorOperacionesBorradores({
  crearControlador = () => new AbortController(),
} = {}) {
  if (typeof crearControlador !== "function") {
    throw new TypeError("fábrica de AbortController no válida");
  }
  const activas = new Map();
  let generacion = 0;

  function iniciar(nombre) {
    validarNombre(nombre);
    const anterior = activas.get(nombre);
    anterior?.abort();
    const controlador = crearControlador();
    if (!controlador || typeof controlador.abort !== "function"
      || typeof controlador.signal?.aborted !== "boolean") {
      throw new TypeError("AbortController de operación no válido");
    }
    activas.set(nombre, controlador);
    const generacionInicial = generacion;
    return Object.freeze({
      signal: controlador.signal,
      vigente: () => generacionInicial === generacion
        && activas.get(nombre) === controlador && !controlador.signal.aborted,
      finalizar: () => {
        if (activas.get(nombre) === controlador) activas.delete(nombre);
      },
    });
  }

  function cancelar(nombre) {
    validarNombre(nombre);
    const controlador = activas.get(nombre);
    if (!controlador) return false;
    activas.delete(nombre);
    controlador.abort();
    return true;
  }

  function invalidar() {
    generacion += 1;
    for (const controlador of activas.values()) controlador.abort();
    activas.clear();
  }

  return Object.freeze({ cancelar, iniciar, invalidar });
}
