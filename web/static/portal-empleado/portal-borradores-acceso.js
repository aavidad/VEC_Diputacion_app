import { ErrorAPIBorradores } from "./portal-borradores-api.js";
import { traducirPortal } from "./portal-i18n.js?v=20260721-acceso-real-v2";

const FASE_INICIAL = "inicial";
const FASE_COMPROBANDO = "comprobando";
const FASE_DISPONIBLE = "disponible";
const FASE_DENEGADA = "denegada";
const FASE_ERROR = "error";

function accesoVisible(fase, traducir) {
  switch (fase) {
    case FASE_DISPONIBLE:
      return Object.freeze({
        disponible: true,
        vista: "elaboracion",
        estado: "disponible",
        etiqueta: traducir("acceso_borradores_disponible"),
      });
    case FASE_DENEGADA:
      return Object.freeze({
        disponible: false,
        vista: "",
        estado: "denegado",
        etiqueta: traducir("acceso_borradores_denegado"),
      });
    case FASE_ERROR:
      return Object.freeze({
        disponible: false,
        vista: "",
        estado: "error",
        etiqueta: traducir("acceso_borradores_error"),
        reintentar: true,
      });
    case FASE_INICIAL:
    case FASE_COMPROBANDO:
    default:
      return Object.freeze({
        disponible: false,
        vista: "",
        estado: "cargando",
        etiqueta: traducir("acceso_borradores_cargando"),
      });
  }
}

function opcionesComprobacionValidas(opciones) {
  return opciones !== null && typeof opciones === "object" && !Array.isArray(opciones)
    && opciones.capacidades !== null && typeof opciones.capacidades === "object"
    && typeof opciones.capacidades.consultar === "boolean";
}

/**
 * Comprueba únicamente la capacidad global de consulta. No lista borradores,
 * no conserva credenciales y no convierte un fallo de red en autorización.
 */
export function crearControlAccesoBorradores({
  consultarOpciones, alCambiar, traducir = traducirPortal,
} = {}) {
  if (typeof consultarOpciones !== "function" || typeof alCambiar !== "function"
    || typeof traducir !== "function") {
    throw new TypeError("dependencias del control de acceso a borradores no válidas");
  }

  let fase = FASE_INICIAL;
  let opciones = null;
  let error = null;
  let promesa = null;
  let controlador = null;
  let revision = 0;

  function cambiarFase(nuevaFase, nuevasOpciones = null, nuevoError = null) {
    fase = nuevaFase;
    opciones = nuevasOpciones;
    error = nuevoError;
    alCambiar(accesoVisible(fase, traducir));
  }

  async function ejecutarComprobacion(revisionActual, signal) {
    try {
      const recibidas = await consultarOpciones({ signal });
      if (revisionActual !== revision || signal.aborted) return false;
      if (!opcionesComprobacionValidas(recibidas)) {
        throw new ErrorAPIBorradores(
          traducir("error_capacidad_consultar_invalida"),
          0,
          undefined,
          { codigo: "capacidad_consultar_invalida" },
        );
      }
      if (recibidas.capacidades.consultar !== true) {
        cambiarFase(FASE_DENEGADA, null, new ErrorAPIBorradores(
          traducir("error_capacidad_consultar_denegada"),
          403,
          undefined,
          { codigo: "capacidad_consultar_no_concedida" },
        ));
        return false;
      }
      cambiarFase(FASE_DISPONIBLE, recibidas, null);
      return true;
    } catch (causa) {
      if (revisionActual !== revision || signal.aborted) return false;
      const denegada = causa instanceof ErrorAPIBorradores
        && (causa.estado === 401 || causa.estado === 403);
      cambiarFase(denegada ? FASE_DENEGADA : FASE_ERROR, null, causa);
      return false;
    }
  }

  function comprobar({ forzar = false } = {}) {
    if (typeof forzar !== "boolean") {
      return Promise.reject(new TypeError("opción de comprobación no válida"));
    }
    if (!forzar && promesa !== null) return promesa;
    if (!forzar && fase === FASE_DISPONIBLE) return Promise.resolve(true);
    if (!forzar && fase === FASE_DENEGADA) return Promise.resolve(false);

    controlador?.abort();
    controlador = new AbortController();
    revision += 1;
    cambiarFase(FASE_COMPROBANDO);
    const revisionActual = revision;
    const actual = ejecutarComprobacion(revisionActual, controlador.signal);
    promesa = actual.finally(() => {
      if (revisionActual === revision) promesa = null;
    });
    return promesa;
  }

  function invalidar(error) {
    const denegado = error instanceof ErrorAPIBorradores
      && (error.estado === 401 || error.estado === 403);
    if (!denegado) return false;
    controlador?.abort();
    controlador = null;
    revision += 1;
    promesa = null;
    cambiarFase(FASE_DENEGADA, null, error);
    return true;
  }

  return Object.freeze({
    comprobar,
    invalidar,
    obtenerAcceso: () => accesoVisible(fase, traducir),
    obtenerError: () => error,
    obtenerOpciones: () => opciones,
  });
}
