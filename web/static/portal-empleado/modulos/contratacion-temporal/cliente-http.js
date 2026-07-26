import { validarComandoAlta, validarReciboAlta } from "./contrato.js";
import {
  validarPropuestaCobertura,
  validarReciboCobertura,
  validarSolicitudDecisionCobertura,
  validarSolicitudPropuestaCobertura,
  validarSolicitudRectificacionCobertura,
} from "./contrato-cobertura.js";

export const RUTAS_HTTP_CONTRATACION_TEMPORAL = Object.freeze({
  alta: "/api/vec/contratacion-temporal/solicitudes",
  propuestaCobertura:
    "/api/vec/contratacion-temporal/cobertura/propuesta",
  decisionCobertura:
    "/api/vec/contratacion-temporal/cobertura/decisiones",
  rectificacionCobertura:
    "/api/vec/contratacion-temporal/cobertura/rectificaciones",
});

const MAXIMO_SOLICITUD_ALTA_BYTES = 256 * 1024;
const MAXIMO_SOLICITUD_COBERTURA_BYTES = 64 * 1024;
const MAXIMO_RESPUESTA_ALTA_BYTES = 16 * 1024;
const MAXIMO_RESPUESTA_COBERTURA_BYTES = 256 * 1024;
const MAXIMO_ERROR_BYTES = 16 * 1024;
const MAXIMO_FRAGMENTOS = 4096;
const MAXIMO_FRAGMENTOS_ERROR = 256;
const PATRON_CORRELACION = /^corr_(?:[0-9a-f]{32}|no_disponible)$/u;
const CODIGOS_POR_ESTADO = new Map([
  [400, new Set(["peticion_no_valida", "peticion_no_permitida"])],
  [401, new Set(["autenticacion_requerida"])],
  [403, new Set(["acceso_denegado"])],
  [404, new Set(["recurso_no_encontrado"])],
  [405, new Set(["metodo_no_permitido"])],
  [406, new Set(["representacion_no_aceptable"])],
  [408, new Set(["peticion_cancelada"])],
  [409, new Set([
    "conflicto",
    "clave_idempotencia_reutilizada",
  ])],
  [413, new Set(["peticion_demasiado_grande"])],
  [415, new Set(["tipo_contenido_no_admitido"])],
  [422, new Set(["contenido_no_valido"])],
  [500, new Set(["error_interno"])],
  [502, new Set(["resultado_no_confiable"])],
  [503, new Set(["servicio_no_disponible", "operacion_pendiente"])],
  [504, new Set(["plazo_agotado"])],
]);

export class ErrorClienteHTTPContratacionTemporal extends Error {
  constructor(codigo, {
    estado = 0,
    claveI18n = "api.contratacion_temporal.cliente.error." + codigo,
    correlacionRef = null,
    envelopeValido = false,
    resultadoIndeterminado = codigo === "operacion_pendiente"
      || codigo === "resultado_indeterminado",
  } = {}) {
    super(`cliente HTTP de contratación temporal: ${codigo}`);
    this.name = codigo === "operacion_abortada"
      ? "AbortError"
      : "ErrorClienteHTTPContratacionTemporal";
    this.codigo = codigo;
    this.estado = estado;
    this.claveI18n = claveI18n;
    this.correlacionRef = correlacionRef;
    this.envelopeValido = envelopeValido;
    this.resultadoIndeterminado = resultadoIndeterminado;
    this.requiereRecuperacion = resultadoIndeterminado;
    this.reintentoPermitido = false;
    this.repetible = false;
  }
}

function errorCliente(codigo, opciones) {
  return new ErrorClienteHTTPContratacionTemporal(codigo, opciones);
}

function convertirEnResultadoIndeterminado(error) {
  if (!(error instanceof ErrorClienteHTTPContratacionTemporal)
    || error.resultadoIndeterminado
    || error.envelopeValido) return error;
  return errorCliente(error.codigo, {
    estado: error.estado,
    resultadoIndeterminado: true,
  });
}

function esRegistro(valor) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) {
    return false;
  }
  try {
    if (Object.getPrototypeOf(valor) !== Object.prototype
      || Object.getOwnPropertySymbols(valor).length !== 0) return false;
    return Object.values(Object.getOwnPropertyDescriptors(valor)).every(
      (descriptor) => Object.hasOwn(descriptor, "value")
        && descriptor.enumerable === true,
    );
  } catch {
    return false;
  }
}

function exigirCamposExactos(valor, campos) {
  if (!esRegistro(valor)) return false;
  const recibidos = Object.keys(valor);
  return recibidos.length === campos.length
    && recibidos.every((campo) => campos.includes(campo))
    && campos.every((campo) => Object.hasOwn(valor, campo));
}

function validarSignal(signal) {
  if (signal === undefined) return undefined;
  if (!signal || typeof signal !== "object"
    || typeof signal.aborted !== "boolean"
    || typeof signal.addEventListener !== "function"
    || typeof signal.removeEventListener !== "function") {
    throw errorCliente("signal_no_valida");
  }
  if (signal.aborted) throw errorAborto(signal);
  return signal;
}

function validarOpciones(opciones) {
  if (opciones === undefined) return Object.freeze({ signal: undefined });
  if (!esRegistro(opciones)
    || Object.keys(opciones).some((campo) => campo !== "signal")) {
    throw errorCliente("opciones_no_validas");
  }
  return Object.freeze({ signal: validarSignal(opciones.signal) });
}

function errorAborto(signal) {
  void signal;
  return errorCliente("operacion_abortada");
}

async function ejecutarAbortable(iniciar, signal, alAbortar, alResolverTardio) {
  if (signal === undefined) return iniciar();
  return new Promise((resolve, reject) => {
    let terminada = false;
    const limpiar = () => signal.removeEventListener("abort", abortar);
    const abortar = () => {
      if (terminada) return;
      terminada = true;
      limpiar();
      try {
        Promise.resolve(alAbortar?.()).catch(() => {});
      } catch {
        // La cancelación es de mejor esfuerzo y no sustituye el error.
      }
      reject(errorAborto(signal));
    };
    signal.addEventListener("abort", abortar, { once: true });
    if (signal.aborted) {
      abortar();
      return;
    }
    let operacion;
    try {
      operacion = iniciar();
    } catch (error) {
      terminada = true;
      limpiar();
      reject(error);
      return;
    }
    Promise.resolve(operacion).then(
      (valor) => {
        if (terminada) {
          try {
            Promise.resolve(alResolverTardio?.(valor)).catch(() => {});
          } catch {
            // Una respuesta tardía se descarta sin alterar el resultado.
          }
          return;
        }
        terminada = true;
        limpiar();
        resolve(valor);
      },
      (error) => {
        if (terminada) return;
        terminada = true;
        limpiar();
        reject(error);
      },
    );
  });
}

async function cancelarRespuesta(respuesta, lector = null) {
  const cancelar = lector && typeof lector.cancel === "function"
    ? () => lector.cancel("respuesta descartada")
    : respuesta?.body && typeof respuesta.body.cancel === "function"
      ? () => respuesta.body.cancel("respuesta descartada")
      : null;
  if (cancelar === null) return;
  try {
    await cancelar();
  } catch {
    // La limpieza nunca sustituye el error causal.
  }
}

function longitudDeclarada(respuesta, maximo) {
  const valor = respuesta.headers?.get?.("content-length");
  if (valor === null || valor === undefined || valor === "") return null;
  if (!/^(?:0|[1-9][0-9]*)$/u.test(valor)) {
    throw errorCliente("respuesta_incompatible", {
      estado: respuesta.status,
    });
  }
  const longitud = Number(valor);
  if (!Number.isSafeInteger(longitud) || longitud > maximo) {
    throw errorCliente("respuesta_excesiva", { estado: respuesta.status });
  }
  return longitud;
}

function validarTipoJSON(respuesta) {
  const tipo = respuesta.headers?.get?.("content-type");
  const codificacion = respuesta.headers?.get?.("content-encoding");
  if (typeof tipo !== "string"
    || !/^application\/json;\s*charset=utf-8$/iu.test(tipo)
    || codificacion !== null && codificacion !== undefined
      && codificacion !== "") {
    throw errorCliente("tipo_respuesta_no_valido", {
      estado: respuesta.status,
    });
  }
}

async function leerJSONAcotado(
  respuesta,
  signal,
  maximoBytes,
  maximoFragmentos,
) {
  validarTipoJSON(respuesta);
  const declarada = longitudDeclarada(respuesta, maximoBytes);
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") {
    throw errorCliente("respuesta_no_incremental", {
      estado: respuesta.status,
    });
  }
  const lector = respuesta.body.getReader();
  if (!lector || typeof lector.read !== "function"
    || typeof lector.cancel !== "function") {
    throw errorCliente("respuesta_no_incremental", {
      estado: respuesta.status,
    });
  }
  const fragmentos = [];
  let total = 0;
  try {
    while (true) {
      const fragmento = await ejecutarAbortable(
        () => lector.read(),
        signal,
        () => lector.cancel("operación cancelada"),
      );
      if (!fragmento || typeof fragmento.done !== "boolean") {
        throw errorCliente("respuesta_incompatible", {
          estado: respuesta.status,
        });
      }
      if (fragmento.done) break;
      if (!(fragmento.value instanceof Uint8Array)
        || fragmento.value.byteLength === 0) {
        throw errorCliente("respuesta_incompatible", {
          estado: respuesta.status,
        });
      }
      fragmentos.push(fragmento.value);
      total += fragmento.value.byteLength;
      if (total > maximoBytes || fragmentos.length > maximoFragmentos) {
        throw errorCliente("respuesta_excesiva", {
          estado: respuesta.status,
        });
      }
    }
    if (declarada !== null && total !== declarada) {
      throw errorCliente("respuesta_incompatible", {
        estado: respuesta.status,
      });
    }
    const bytes = new Uint8Array(total);
    let posicion = 0;
    for (const fragmento of fragmentos) {
      bytes.set(fragmento, posicion);
      posicion += fragmento.byteLength;
    }
    let texto;
    try {
      texto = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch (error) {
      throw errorCliente("respuesta_utf8_no_valida", {
        estado: respuesta.status,
        causa: error,
      });
    }
    try {
      const valor = JSON.parse(texto);
      if (JSON.stringify(valor) !== texto) {
        throw new TypeError("JSON no canónico");
      }
      return valor;
    } catch (error) {
      throw errorCliente("respuesta_json_no_valida", {
        estado: respuesta.status,
        causa: error,
      });
    }
  } catch (error) {
    await cancelarRespuesta(respuesta, lector);
    throw error;
  } finally {
    try {
      lector.releaseLock?.();
    } catch {
      // El lector cancelado puede haber liberado ya el bloqueo.
    }
  }
}

function serializarAcotado(valor, maximoBytes) {
  let texto;
  try {
    texto = JSON.stringify(valor);
  } catch (error) {
    throw errorCliente("solicitud_no_serializable", { causa: error });
  }
  if (new TextEncoder().encode(texto).byteLength > maximoBytes) {
    throw errorCliente("solicitud_excesiva");
  }
  return texto;
}

function extraerDatos(envoltorio) {
  if (!exigirCamposExactos(envoltorio, ["data"])
    || !esRegistro(envoltorio.data)) {
    throw errorCliente("respuesta_incompatible");
  }
  return envoltorio.data;
}

function claveI18nValida(ruta, codigo, clave) {
  if (typeof clave !== "string") return false;
  if (clave === `api.vec.ruta_exacta.error.${codigo}`) {
    return [
      "recurso_no_encontrado",
      "autenticacion_requerida",
      "acceso_denegado",
      "servicio_no_disponible",
    ].includes(codigo);
  }
  const prefijo = ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.alta
    ? "api.contratacion_temporal.alta.error."
    : "api.contratacion_temporal.cobertura.error.";
  return clave === `${prefijo}${codigo}`;
}

function codigoValidoParaRuta(ruta, estado, codigo) {
  if (!CODIGOS_POR_ESTADO.get(estado)?.has(codigo)) return false;
  if (codigo === "operacion_pendiente"
    && ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.propuestaCobertura) {
    return false;
  }
  if (estado !== 409) return true;
  return ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.alta
    ? codigo === "clave_idempotencia_reutilizada"
    : codigo === "conflicto";
}

async function construirErrorRespuesta(respuesta, signal, ruta) {
  let envoltorio;
  try {
    envoltorio = await leerJSONAcotado(
      respuesta,
      signal,
      MAXIMO_ERROR_BYTES,
      MAXIMO_FRAGMENTOS_ERROR,
    );
  } catch (error) {
    if (error instanceof ErrorClienteHTTPContratacionTemporal
      && error.codigo === "operacion_abortada") {
      throw error;
    }
    return errorCliente("respuesta_error_no_valida", {
      estado: respuesta.status,
    });
  }
  const detalle = envoltorio?.error;
  if (!exigirCamposExactos(envoltorio, ["error"])
    || !exigirCamposExactos(
      detalle,
      ["codigo", "clave_i18n", "correlacion_ref"],
    )
    || !codigoValidoParaRuta(ruta, respuesta.status, detalle.codigo)
    || !claveI18nValida(ruta, detalle.codigo, detalle.clave_i18n)
    || typeof detalle.correlacion_ref !== "string"
    || !PATRON_CORRELACION.test(detalle.correlacion_ref)) {
    return errorCliente("respuesta_error_no_valida", {
      estado: respuesta.status,
    });
  }
  return errorCliente(detalle.codigo, {
    estado: respuesta.status,
    claveI18n: detalle.clave_i18n,
    correlacionRef: detalle.correlacion_ref,
    envelopeValido: true,
  });
}

function construirCabeceras(HeadersImpl) {
  const cabeceras = new HeadersImpl();
  cabeceras.set("Accept", "application/json");
  cabeceras.set("Content-Type", "application/json; charset=utf-8");
  if (typeof cabeceras.keys !== "function"
    || [...cabeceras.keys()].map((nombre) => nombre.toLowerCase()).sort()
      .join(",") !== "accept,content-type"
    || cabeceras.get("accept") !== "application/json"
    || cabeceras.get("content-type")
      !== "application/json; charset=utf-8") {
    throw new TypeError("cabeceras de contratación temporal no válidas");
  }
  return cabeceras;
}

export function crearClienteHTTPContratacionTemporal(configuracion = {}) {
  if (!esRegistro(configuracion)
    || Object.keys(configuracion).some(
      (campo) => !["fetchImpl", "HeadersImpl"].includes(campo),
    )) {
    throw new TypeError(
      "configuración del cliente HTTP de contratación temporal no válida",
    );
  }
  const {
    fetchImpl = globalThis.fetch,
    HeadersImpl = globalThis.Headers,
  } = configuracion;
  if (typeof fetchImpl !== "function" || typeof HeadersImpl !== "function"
    || typeof TextEncoder !== "function" || typeof TextDecoder !== "function") {
    throw new TypeError(
      "capacidades del cliente HTTP de contratación temporal no disponibles",
    );
  }

  async function ejecutar({
    ruta,
    entrada,
    signal,
    estadoEsperado,
    maximoSolicitud,
    maximoRespuesta,
    validarRespuesta,
    efecto,
  }) {
    const cuerpo = serializarAcotado(entrada, maximoSolicitud);
    let cabeceras;
    try {
      cabeceras = construirCabeceras(HeadersImpl);
    } catch (error) {
      throw errorCliente("cabeceras_no_disponibles", { causa: error });
    }
    let respuesta;
    try {
      respuesta = await ejecutarAbortable(
        () => fetchImpl(ruta, {
          method: "POST",
          headers: cabeceras,
          body: cuerpo,
          signal,
          credentials: "omit",
          mode: "same-origin",
          cache: "no-store",
          redirect: "error",
          referrerPolicy: "no-referrer",
        }),
        signal,
        null,
        (tardia) => cancelarRespuesta(tardia),
      );
    } catch (error) {
      const fallo = error instanceof ErrorClienteHTTPContratacionTemporal
        ? error
        : signal?.aborted
          ? errorAborto(signal)
          : errorCliente("servicio_no_disponible", { causa: error });
      throw efecto ? convertirEnResultadoIndeterminado(fallo) : fallo;
    }
    try {
      if (!respuesta || !Number.isInteger(respuesta.status)
        || respuesta.status < 200 || respuesta.status > 599
        || respuesta.redirected === true) {
        throw errorCliente("respuesta_incompatible");
      }
      if (respuesta.status !== estadoEsperado) {
        if (respuesta.status >= 400) {
          throw await construirErrorRespuesta(respuesta, signal, ruta);
        }
        throw errorCliente("estado_respuesta_no_valido", {
          estado: respuesta.status,
        });
      }
      let validada;
      try {
        validada = validarRespuesta(extraerDatos(await leerJSONAcotado(
          respuesta,
          signal,
          maximoRespuesta,
          MAXIMO_FRAGMENTOS,
        )));
      } catch (error) {
        if (error instanceof ErrorClienteHTTPContratacionTemporal) throw error;
        throw errorCliente("respuesta_incompatible", {
          estado: respuesta.status,
          causa: error,
        });
      }
      return validada;
    } catch (error) {
      throw efecto ? convertirEnResultadoIndeterminado(error) : error;
    } finally {
      await cancelarRespuesta(respuesta);
    }
  }

  async function registrarSolicitud(comando, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarComandoAlta(comando);
    return ejecutar({
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.alta,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_ALTA_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_ALTA_BYTES,
      validarRespuesta: validarReciboAlta,
      efecto: true,
    });
  }

  async function proponerCobertura(solicitud, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarSolicitudPropuestaCobertura(solicitud);
    return ejecutar({
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.propuestaCobertura,
      entrada,
      signal,
      estadoEsperado: 200,
      maximoSolicitud: MAXIMO_SOLICITUD_COBERTURA_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_COBERTURA_BYTES,
      validarRespuesta: validarPropuestaCobertura,
      efecto: false,
    });
  }

  async function decidirCobertura(decision, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarSolicitudDecisionCobertura(decision);
    return ejecutar({
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.decisionCobertura,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_COBERTURA_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_COBERTURA_BYTES,
      validarRespuesta: (respuesta) => {
        const recibo = validarReciboCobertura(respuesta);
        if (recibo.estado === "aplicada"
          && recibo.version_resultante !== entrada.version_esperada + 1) {
          throw new TypeError(
            "el recibo no corresponde a la versión de la decisión",
          );
        }
        return recibo;
      },
      efecto: true,
    });
  }

  async function rectificarCobertura(rectificacion, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarSolicitudRectificacionCobertura(rectificacion);
    return ejecutar({
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.rectificacionCobertura,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_COBERTURA_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_COBERTURA_BYTES,
      validarRespuesta: (respuesta) => {
        const recibo = validarReciboCobertura(respuesta);
        if (recibo.estado === "aplicada"
          && recibo.version_resultante !== entrada.version_esperada + 1) {
          throw new TypeError(
            "el recibo no corresponde a la versión de la rectificación",
          );
        }
        return recibo;
      },
      efecto: true,
    });
  }

  return Object.freeze({
    modo: "http",
    registrarSolicitud,
    proponerCobertura,
    decidirCobertura,
    rectificarCobertura,
  });
}
