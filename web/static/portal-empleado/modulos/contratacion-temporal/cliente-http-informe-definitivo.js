/** Descarga binaria de la consulta RRHH; no registra actuaciones ni genera documentos. */
import { ErrorClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { RUTAS_CONSULTA_RRHH } from "./cliente-http-consultas-rrhh.js";

// Perfiles de representación, no de identidad ni autorización.
export const PERFILES_BORRADOR_RRHH = Object.freeze({
  informe_definitivo: Object.freeze({
    accept: "application/pdf; documento=informe-definitivo-desarrollo",
    nombre: "informe-definitivo-borrador.pdf",
  }),
  resolucion: Object.freeze({
    accept: "application/pdf; documento=resolucion-desarrollo",
    nombre: "resolucion-borrador.pdf",
  }),
});
const MAXIMO_PDF = 2 * 1024 * 1024;
const PREFIJO_ERROR = "api.contratacion_temporal.consulta_rrhh.error.";
const CODIGOS = Object.freeze({
  400: ["peticion_no_valida", "peticion_no_permitida"],
  401: ["autenticacion_requerida"], 403: ["acceso_denegado"],
  404: ["recurso_no_encontrado"], 405: ["metodo_no_permitido"],
  406: ["representacion_no_aceptable"], 408: ["peticion_cancelada"],
  409: ["documento_no_disponible"], 413: ["peticion_demasiado_grande"],
  415: ["tipo_contenido_no_admitido"], 422: ["contenido_no_valido"],
  500: ["error_interno"], 502: ["resultado_no_confiable"],
  503: ["servicio_no_disponible"], 504: ["plazo_agotado"],
});
const errorCliente = (codigo, opciones) => new ErrorClienteHTTPContratacionTemporal(codigo, opciones);

function camposExactos(valor, campos) {
  if (!valor || Object.getPrototypeOf(valor) !== Object.prototype
    || Reflect.ownKeys(valor).length !== campos.length) return false;
  return campos.every((campo) => {
    const descriptor = Object.getOwnPropertyDescriptor(valor, campo);
    return descriptor?.enumerable === true && Object.hasOwn(descriptor, "value");
  });
}

function esperar(promesa, signal) {
  return new Promise((resolve, reject) => {
    const cancelar = () => reject(errorCliente("operacion_abortada"));
    signal.addEventListener("abort", cancelar, { once: true });
    Promise.resolve(promesa).then(resolve, reject).finally(() => {
      signal.removeEventListener("abort", cancelar);
    });
    if (signal.aborted) cancelar();
  });
}

async function leerAcotado(respuesta, signal, maximo) {
  const longitud = respuesta.headers.get("Content-Length");
  let lector;
  try {
    if (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud)
      || Number(longitud) > maximo)) throw errorCliente("resultado_no_confiable");
    lector = respuesta.body?.getReader();
    if (!lector) throw errorCliente("resultado_no_confiable");
    const fragmentos = [];
    let total = 0;
    for (let cantidad = 0; ; cantidad += 1) {
      if (cantidad > 4096) throw errorCliente("resultado_no_confiable");
      const { value, done } = await esperar(lector.read(), signal);
      if (done) break;
      if (!(value instanceof Uint8Array)) throw errorCliente("resultado_no_confiable");
      total += value.byteLength;
      if (total > maximo) throw errorCliente("resultado_no_confiable");
      fragmentos.push(value);
    }
    if (longitud !== null && Number(longitud) !== total) throw errorCliente("resultado_no_confiable");
    const bytes = new Uint8Array(total);
    let posicion = 0;
    for (const fragmento of fragmentos) {
      bytes.set(fragmento, posicion);
      posicion += fragmento.byteLength;
    }
    return bytes;
  } finally {
    if (lector) {
      void lector.cancel().catch(() => {});
      lector.releaseLock();
    } else {
      void respuesta.body?.cancel().catch(() => {});
    }
  }
}

async function errorConsulta(respuesta, signal) {
  const bytes = await leerAcotado(respuesta, signal, 16 * 1024);
  let envoltorio;
  try {
    if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
      throw new TypeError();
    }
    envoltorio = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  } catch { return errorCliente("respuesta_error_no_valida"); }
  const detalle = envoltorio?.error;
  if (!camposExactos(envoltorio, ["error"])
    || !camposExactos(detalle, ["codigo", "clave_i18n", "correlacion_ref"])
    || !CODIGOS[respuesta.status]?.includes(detalle.codigo)
    || detalle.clave_i18n !== PREFIJO_ERROR + detalle.codigo
    || typeof detalle.correlacion_ref !== "string"
    || !/^corr_(?:[0-9a-f]{32}|no_disponible)$/u.test(detalle.correlacion_ref)) {
    return errorCliente("respuesta_error_no_valida");
  }
  return errorCliente(detalle.codigo, {
    estado: respuesta.status, claveI18n: detalle.clave_i18n,
    correlacionRef: detalle.correlacion_ref, envelopeValido: true,
  });
}

export function crearClienteHTTPBorradorRRHH({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async descargarBorrador(solicitud, { tipo = "informe_definitivo", signal } = {}) {
      if (typeof tipo !== "string" || !Object.hasOwn(PERFILES_BORRADOR_RRHH, tipo)) {
        throw errorCliente("solicitud_no_valida");
      }
      const perfil = PERFILES_BORRADOR_RRHH[tipo];
      if (!camposExactos(solicitud, ["expediente_ref", "version_observada"])
        || typeof solicitud.expediente_ref !== "string"
        || !/^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u.test(solicitud.expediente_ref)
        || solicitud.version_observada !== 7) throw errorCliente("solicitud_no_valida");
      if (typeof fetchImpl !== "function") throw errorCliente("servicio_no_disponible");
      const controlador = new AbortController();
      const cancelar = () => controlador.abort();
      signal?.addEventListener("abort", cancelar, { once: true });
      const temporizador = setTimeout(cancelar, 15000);
      let respuesta;
      try {
        if (signal?.aborted) cancelar();
        if (controlador.signal.aborted) throw errorCliente("operacion_abortada");
        const pendiente = Promise.resolve(fetchImpl(RUTAS_CONSULTA_RRHH.detalleRRHH, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Accept: perfil.accept,
          },
          body: JSON.stringify({ expediente_ref: solicitud.expediente_ref, version_observada: 7 }),
          signal: controlador.signal, mode: "same-origin", credentials: "same-origin",
          cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        }));
        void pendiente.then((tardia) => {
          if (controlador.signal.aborted) void tardia.body?.cancel().catch(() => {});
        }, () => {});
        respuesta = await esperar(pendiente, controlador.signal);
        if (respuesta.redirected) throw errorCliente("resultado_no_confiable");
        if (respuesta.status !== 200) throw await errorConsulta(respuesta, controlador.signal);
        if (respuesta.headers.get("Content-Type") !== "application/pdf"
          || respuesta.headers.get("Content-Disposition") !== `attachment; filename="${perfil.nombre}"`) {
          throw errorCliente("resultado_no_confiable");
        }
        const bytes = await leerAcotado(respuesta, controlador.signal, MAXIMO_PDF);
        if (controlador.signal.aborted) throw errorCliente("operacion_abortada");
        if (new TextDecoder().decode(bytes.subarray(0, 5)) !== "%PDF-") {
          throw errorCliente("resultado_no_confiable");
        }
        return new Blob([bytes], { type: "application/pdf" });
      } catch (error) {
        if (controlador.signal.aborted) throw errorCliente("operacion_abortada");
        if (error instanceof ErrorClienteHTTPContratacionTemporal) throw error;
        throw errorCliente("servicio_no_disponible");
      } finally {
        clearTimeout(temporizador);
        signal?.removeEventListener("abort", cancelar);
        if (respuesta && !respuesta.bodyUsed) void respuesta.body?.cancel().catch(() => {});
      }
    },
  });
}
