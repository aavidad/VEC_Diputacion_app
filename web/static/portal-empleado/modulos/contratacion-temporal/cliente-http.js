import { crearAltaClienteHTTP, RUTAS_ALTA_CONTRATACION_TEMPORAL } from "./cliente-http-alta.js";
import {
  validarPropuestaCobertura,
  validarReciboCobertura,
  validarResultadoConsultaCobertura,
  validarSolicitudConsultaResultadoCobertura,
  validarSolicitudDecisionCobertura,
  validarSolicitudPropuestaCobertura,
  validarSolicitudRectificacionCobertura,
} from "./contrato-cobertura.js";
import {
  validarConfiguracionAnalisis,
  validarReciboAnalisis,
  validarSolicitudRectificacionAnalisis,
  validarSolicitudRegistroAnalisis,
} from "./contrato-analisis.js";
import { crearAsignacionClienteHTTP, RUTA_ASIGNACION_CONTRATACION_TEMPORAL } from "./cliente-http-asignacion.js";
import { crearConsultasRRHHClienteHTTP, RUTAS_CONSULTA_RRHH } from "./cliente-http-consultas-rrhh.js";
import { crearInformeJuridicoClienteHTTP, RUTA_PREPARACION_INFORME_JURIDICO } from "./cliente-http-informe-juridico.js";
import { crearFiscalizacionClienteHTTP, RUTA_RESULTADOS_FISCALIZACION } from "./cliente-http-fiscalizacion.js";
import { crearLlamamientoClienteHTTP, RUTAS_LLAMAMIENTO } from "./cliente-http-llamamiento.js";
import { crearResolucionFormalizacionClienteHTTP, RUTA_RESOLUCION_FORMALIZACION } from "./cliente-http-resolucion-formalizacion.js";
import { crearIncorporacionEjercicioClienteHTTP, RUTA_INCORPORACION_EJERCICIO } from "./cliente-http-incorporacion-ejercicio.js";
import { crearFichaGINPIXClienteHTTP, RUTA_FICHA_GINPIX, NOMBRE_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";
import { crearClienteSeguimientoIncorporacion, RUTA_SEGUIMIENTO_INCORPORACION } from "./cliente-http-seguimiento-incorporacion.js";
import { crearClienteAnotacionAdministrativaHTTP, RUTA_ANOTACION_ADMINISTRATIVA, RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA } from "./cliente-http-anotacion-administrativa.js";
import { crearClienteCierreAdministrativoHTTP, RUTA_CIERRE_ADMINISTRATIVO } from "./cliente-http-cierre-administrativo.js";
import { crearClienteSubsanacionReparosHTTP, RUTA_SUBSANACION_REPAROS } from "./cliente-http-subsanacion-reparos.js";
import {
  MAXIMO_ERROR_BYTES,
  MAXIMO_FRAGMENTOS,
  MAXIMO_FRAGMENTOS_ERROR,
  PATRON_CORRELACION,
  validarSignal,
  validarOpciones,
  errorAborto,
  ejecutarAbortable,
  cancelarRespuesta,
  leerJSONAcotado,
  serializarAcotado,
  extraerDatos,
  construirErrorRespuesta,
  construirCabeceras,
} from "./cliente-http-transporte.js";

export const RUTAS_HTTP_CONTRATACION_TEMPORAL = Object.freeze({
  alta: RUTAS_ALTA_CONTRATACION_TEMPORAL.alta,
  propuestaCobertura: "/api/vec/contratacion-temporal/cobertura/propuesta",
  decisionCobertura: "/api/vec/contratacion-temporal/cobertura/decisiones",
  rectificacionCobertura: "/api/vec/contratacion-temporal/cobertura/rectificaciones",
  resultadoCobertura: "/api/vec/contratacion-temporal/cobertura/resultados",
  registroAnalisis: "/api/vec/contratacion-temporal/analisis/registros",
  rectificacionAnalisis: "/api/vec/contratacion-temporal/analisis/rectificaciones",
  configuracionAnalisis: "/api/vec/contratacion-temporal/configuracion-analisis",
  preparacionInformeJuridico: RUTA_PREPARACION_INFORME_JURIDICO,
  resultadosFiscalizacion: RUTA_RESULTADOS_FISCALIZACION,
  ...RUTAS_CONSULTA_RRHH,
  catalogosAlta: RUTAS_ALTA_CONTRATACION_TEMPORAL.catalogosAlta,
  ...RUTAS_LLAMAMIENTO,
  resolucionFormalizacion: RUTA_RESOLUCION_FORMALIZACION,
  incorporacionEjercicio: RUTA_INCORPORACION_EJERCICIO,
  fichaGINPIX: RUTA_FICHA_GINPIX,
  seguimientoIncorporacion: RUTA_SEGUIMIENTO_INCORPORACION,
  anotacionAdministrativa: RUTA_ANOTACION_ADMINISTRATIVA,
  recuperacionAnotacionAdministrativa: RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA,
  preparacionCierreSinCese: "/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese/preparacion",
  cierreAdministrativo: RUTA_CIERRE_ADMINISTRATIVO,
  subsanacionReparos: RUTA_SUBSANACION_REPAROS,
});

export const MAXIMO_SOLICITUD_COBERTURA_BYTES = 64 * 1024;
export const MAXIMO_SOLICITUD_ANALISIS_BYTES = 64 * 1024;
export const MAXIMO_RESPUESTA_ANALISIS_BYTES = 16 * 1024;
export const MAXIMO_RESPUESTA_CONFIGURACION_ANALISIS_BYTES = 64 * 1024;
export const MAXIMO_RESPUESTA_COBERTURA_BYTES = 256 * 1024;
const PATRON_REFERENCIA_OPACA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE_CATALOGO = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_INSTANTE_UTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;

export class ErrorClienteHTTPContratacionTemporal extends Error {
  constructor(codigo, {
    estado = 0,
    claveI18n = "",
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
    Object.freeze(this);
  }
}

function errorCliente(codigo, opciones) {
  return new ErrorClienteHTTPContratacionTemporal(codigo, opciones);
}

const RECHAZOS_ANALISIS_ANTERIORES_AL_EFECTO = new Map([
  [400, new Set(["peticion_no_valida", "peticion_no_permitida"])],
  [401, new Set(["autenticacion_requerida"])],
  [403, new Set(["acceso_denegado"])],
  [404, new Set(["recurso_no_encontrado"])],
  [405, new Set(["metodo_no_permitido"])],
  [406, new Set(["representacion_no_aceptable"])],
  [409, new Set(["conflicto"])],
  [413, new Set(["peticion_demasiado_grande"])],
  [415, new Set(["tipo_contenido_no_admitido"])],
  [422, new Set(["contenido_no_valido"])],
]);

function convertirEnResultadoIndeterminado(error) {
  if (!(error instanceof ErrorClienteHTTPContratacionTemporal)
    || error.resultadoIndeterminado) return error;
  return errorCliente(error.codigo, {
    estado: error.estado,
    claveI18n: error.claveI18n,
    correlacionRef: error.correlacionRef,
    envelopeValido: error.envelopeValido,
    resultadoIndeterminado: true,
  });
}

function rechazoAnalisisAnteriorAlEfecto(error) {
  return error instanceof ErrorClienteHTTPContratacionTemporal
    && error.envelopeValido
    && RECHAZOS_ANALISIS_ANTERIORES_AL_EFECTO
      .get(error.estado)?.has(error.codigo) === true;
}

function clasificarResultadoEfecto(error, rechazoDeterminado) {
  if (!(error instanceof ErrorClienteHTTPContratacionTemporal)
    || error.resultadoIndeterminado) return error;
  const determinado = typeof rechazoDeterminado === "function"
    ? rechazoDeterminado(error)
    : error.envelopeValido;
  return determinado ? error : convertirEnResultadoIndeterminado(error);
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
  const claves = Object.keys(valor);
  if (claves.length !== campos.length) return false;
  const esperados = new Set(campos);
  return claves.every((clave) => esperados.has(clave));
}

function validarConsultaPreparacionCierreSinCese(consulta) {
  if (!exigirCamposExactos(consulta, ["expediente_ref", "seguimiento_ref"])
    || !PATRON_REFERENCIA_OPACA.test(consulta.expediente_ref)
    || !PATRON_REFERENCIA_OPACA.test(consulta.seguimiento_ref)) {
    throw new TypeError("consulta de preparación de cierre sin cese no válida");
  }
  return Object.freeze({ ...consulta });
}

function adaptarPreparacionCierreSinCese(dto, consulta) {
  if (!exigirCamposExactos(dto, [
    "expediente_ref", "seguimiento_ref", "version_actual", "estado_actual", "acciones", "preparada_en",
  ]) || dto.expediente_ref !== consulta.expediente_ref
    || dto.seguimiento_ref !== consulta.seguimiento_ref
    || !PATRON_REFERENCIA_OPACA.test(dto.expediente_ref)
    || !PATRON_REFERENCIA_OPACA.test(dto.seguimiento_ref)
    || !Number.isSafeInteger(dto.version_actual) || dto.version_actual < 1
    || !PATRON_CLAVE_CATALOGO.test(dto.estado_actual)
    || typeof dto.preparada_en !== "string" || !PATRON_INSTANTE_UTC.test(dto.preparada_en)
    || Number.isNaN(Date.parse(dto.preparada_en))
    || !Array.isArray(dto.acciones) || dto.acciones.length > 1) {
    throw new TypeError("preparación de cierre sin cese incompatible");
  }
  let preparacion = null;
  if (dto.acciones.length === 1) {
    const accion = dto.acciones[0];
    if (!exigirCamposExactos(accion, ["transicion_clave", "motivos"])
      || accion.transicion_clave !== "cerrar_administrativamente_sin_cese"
      || !Array.isArray(accion.motivos) || accion.motivos.length < 1 || accion.motivos.length > 256
      || dto.version_actual !== 1 || dto.estado_actual !== "vigente") {
      throw new TypeError("preparación de cierre sin cese incompatible");
    }
    const motivos = accion.motivos.map((motivo) => {
      if (!exigirCamposExactos(motivo, ["motivo_clave"])
        || !PATRON_CLAVE_CATALOGO.test(motivo.motivo_clave)) {
        throw new TypeError("preparación de cierre sin cese incompatible");
      }
      return motivo.motivo_clave;
    });
    if (new Set(motivos).size !== motivos.length) {
      throw new TypeError("preparación de cierre sin cese incompatible");
    }
    preparacion = Object.freeze({
      expediente_ref: dto.expediente_ref,
      seguimiento_ref: dto.seguimiento_ref,
      version_esperada: dto.version_actual,
      motivos: Object.freeze(motivos),
    });
  }
  return Object.freeze({
    expediente_ref: dto.expediente_ref, seguimiento_ref: dto.seguimiento_ref,
    version_actual: dto.version_actual, estado_actual: dto.estado_actual,
    preparada_en: dto.preparada_en, preparacion,
  });
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
  const consultasResultadoEnCurso = new Map();

  function validarOpcionesInterno(opciones) {
    return validarOpciones(opciones, esRegistro, errorCliente);
  }

  function serializarAcotadoInterno(entrada, maximoBytes) {
    return serializarAcotado(entrada, maximoBytes, errorCliente);
  }

  async function ejecutar({
    metodo = "POST",
    ruta,
    entrada,
    signal,
    estadoEsperado,
    maximoSolicitud,
    maximoRespuesta,
    validarRespuesta,
    efecto,
    rechazoDeterminado,
    tipoContenido,
    fichero = false,
  }) {
    if (metodo !== "GET" && metodo !== "POST") {
      throw errorCliente("metodo_no_valido");
    }
    if (fichero && (metodo !== "GET" || ruta.split("?")[0] !== RUTA_FICHA_GINPIX)) {
      throw errorCliente("metodo_no_valido");
    }
    const conCuerpo = metodo === "POST";
    const cuerpo = conCuerpo
      ? serializarAcotadoInterno(entrada, maximoSolicitud)
      : undefined;
    let cabeceras;
    try {
      cabeceras = construirCabeceras(HeadersImpl, tipoContenido, conCuerpo);
    } catch (error) {
      throw errorCliente("cabeceras_no_disponibles", { causa: error });
    }
    let respuesta;
    try {
      const opcionesFetch = {
        method: metodo,
        headers: cabeceras,
        signal,
        credentials: "same-origin",
        mode: "same-origin",
        cache: "no-store",
        redirect: "error",
        referrerPolicy: "no-referrer",
      };
      if (conCuerpo) opcionesFetch.body = cuerpo;
      respuesta = await ejecutarAbortable(
        () => fetchImpl(ruta, opcionesFetch),
        signal,
        null,
        (tardia) => cancelarRespuesta(tardia),
        errorCliente,
      );
    } catch (error) {
      const fallo = error instanceof ErrorClienteHTTPContratacionTemporal
        ? error
        : signal?.aborted
          ? errorAborto(signal, errorCliente)
          : errorCliente("servicio_no_disponible", { causa: error });
      throw efecto
        ? clasificarResultadoEfecto(fallo, rechazoDeterminado)
        : fallo;
    }
    try {
      if (!respuesta || !Number.isInteger(respuesta.status)
        || respuesta.status < 200 || respuesta.status > 599
        || respuesta.redirected === true) {
        throw errorCliente("respuesta_incompatible");
      }
      const estadoValido = Array.isArray(estadoEsperado)
        ? estadoEsperado.includes(respuesta.status)
        : respuesta.status === estadoEsperado;
      if (!estadoValido) {
        if (respuesta.status >= 400) {
          throw await construirErrorRespuesta(
            respuesta,
            signal,
            ruta,
            errorCliente,
            ErrorClienteHTTPContratacionTemporal,
            exigirCamposExactos,
            RUTAS_HTTP_CONTRATACION_TEMPORAL,
          );
        }
        throw errorCliente("estado_respuesta_no_valido", {
          estado: respuesta.status,
        });
      }
      let validada;
      try {
        if (fichero && ![
          `attachment; filename=${NOMBRE_FICHA_GINPIX}`,
          `attachment; filename="${NOMBRE_FICHA_GINPIX}"`,
        ].includes(respuesta.headers?.get?.("content-disposition"))) {
          throw errorCliente("respuesta_incompatible");
        }
        const leida = await leerJSONAcotado(
          respuesta, signal, maximoRespuesta, MAXIMO_FRAGMENTOS, fichero, errorCliente,
        );
        validada = validarRespuesta(
          fichero ? leida.valor : extraerDatos(leida, errorCliente, exigirCamposExactos, esRegistro),
          respuesta.status,
          fichero ? leida.bytes : undefined,
        );
      } catch (error) {
        if (error instanceof ErrorClienteHTTPContratacionTemporal) throw error;
        throw errorCliente("respuesta_incompatible", {
          estado: respuesta.status,
          causa: error,
        });
      }
      return validada;
    } catch (error) {
      throw efecto
        ? clasificarResultadoEfecto(error, rechazoDeterminado)
        : error;
    } finally {
      await cancelarRespuesta(respuesta);
    }
  }

  async function proponerCobertura(solicitud, opciones) {
    const { signal } = validarOpcionesInterno(opciones);
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
    const { signal } = validarOpcionesInterno(opciones);
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
    const { signal } = validarOpcionesInterno(opciones);
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

  async function ejecutarAnalisis({
    solicitud,
    opciones,
    ruta,
    operacion,
    validarSolicitud,
  }) {
    const { signal } = validarOpcionesInterno(opciones);
    const entrada = validarSolicitud(solicitud);
    return ejecutar({
      ruta,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_ANALISIS_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_ANALISIS_BYTES,
      validarRespuesta: (respuesta) => {
        const recibo = validarReciboAnalisis(respuesta);
        if (recibo.operacion !== operacion
          || recibo.expediente_ref !== entrada.expediente_ref
          || recibo.version_resultante !== entrada.version_esperada + 1) {
          throw new TypeError(
            "el recibo no corresponde a la operación de análisis",
          );
        }
        return recibo;
      },
      efecto: true,
      rechazoDeterminado: rechazoAnalisisAnteriorAlEfecto,
    });
  }

  function registrarAnalisis(solicitud, opciones) {
    return ejecutarAnalisis({
      solicitud,
      opciones,
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.registroAnalisis,
      operacion: "registrar",
      validarSolicitud: validarSolicitudRegistroAnalisis,
    });
  }

  function rectificarAnalisis(solicitud, opciones) {
    return ejecutarAnalisis({
      solicitud,
      opciones,
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.rectificacionAnalisis,
      operacion: "rectificar",
      validarSolicitud: validarSolicitudRectificacionAnalisis,
    });
  }

  function obtenerConfiguracionAnalisis(opciones) {
    const { signal } = validarOpcionesInterno(opciones);
    return ejecutar({
      metodo: "GET",
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.configuracionAnalisis,
      signal,
      estadoEsperado: 200,
      maximoRespuesta: MAXIMO_RESPUESTA_CONFIGURACION_ANALISIS_BYTES,
      validarRespuesta: validarConfiguracionAnalisis,
      efecto: false,
    });
  }

  function consultarResultadoCobertura(solicitud, opciones) {
    let signal;
    let entrada;
    try {
      ({ signal } = validarOpcionesInterno(opciones));
      entrada = validarSolicitudConsultaResultadoCobertura(solicitud);
    } catch (error) {
      return Promise.reject(error);
    }
    const claveVuelo = JSON.stringify([
      entrada.expediente_ref,
      entrada.clave_idempotencia,
    ]);
    const existente = consultasResultadoEnCurso.get(claveVuelo);
    if (existente) return existente;

    const operacion = ejecutar({
      ruta: RUTAS_HTTP_CONTRATACION_TEMPORAL.resultadoCobertura,
      entrada,
      signal,
      estadoEsperado: [200, 202],
      maximoSolicitud: MAXIMO_SOLICITUD_COBERTURA_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_COBERTURA_BYTES,
      validarRespuesta: (respuesta, estadoHTTP) => {
        const resultado = validarResultadoConsultaCobertura(respuesta);
        if ((estadoHTTP === 200) !== (resultado.estado === "confirmado")) {
          throw new TypeError(
            "el estado HTTP no corresponde al resultado de cobertura",
          );
        }
        return resultado;
      },
      efecto: false,
    });
    let compartida;
    compartida = operacion.finally(() => {
      if (consultasResultadoEnCurso.get(claveVuelo) === compartida) {
        consultasResultadoEnCurso.delete(claveVuelo);
      }
    });
    consultasResultadoEnCurso.set(claveVuelo, compartida);
    return compartida;
  }

  async function consultarPreparacionCierreSinCese(consulta, opciones) {
    const entrada = validarConsultaPreparacionCierreSinCese(consulta);
    const { signal } = validarOpcionesInterno(opciones);
    const ruta = `${RUTAS_HTTP_CONTRATACION_TEMPORAL.preparacionCierreSinCese}?expediente_ref=${encodeURIComponent(entrada.expediente_ref)}&seguimiento_ref=${encodeURIComponent(entrada.seguimiento_ref)}`;
    return ejecutar({
      metodo: "GET", ruta, signal, estadoEsperado: 200, maximoRespuesta: 16 * 1024,
      efecto: false,
      validarRespuesta: (respuesta) => adaptarPreparacionCierreSinCese(respuesta, entrada),
    });
  }

  return Object.freeze({
    modo: "http",
    ...crearAltaClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    ...crearConsultasRRHHClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    ...crearAsignacionClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno, serializarAcotado: serializarAcotadoInterno }),
    ...crearInformeJuridicoClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno, serializarAcotado: serializarAcotadoInterno }),
    ...crearFiscalizacionClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno, serializarAcotado: serializarAcotadoInterno }),
    ...crearClienteSubsanacionReparosHTTP({ ejecutar, validarOpciones: validarOpcionesInterno, serializarAcotado: serializarAcotadoInterno }),
    ...crearLlamamientoClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    ...crearResolucionFormalizacionClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    ...crearIncorporacionEjercicioClienteHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    anotacionAdministrativa: crearClienteAnotacionAdministrativaHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    ...crearClienteCierreAdministrativoHTTP({ ejecutar, validarOpciones: validarOpcionesInterno }),
    seguimientoIncorporacion: crearClienteSeguimientoIncorporacion({
      validarOpciones: validarOpcionesInterno,
      consultar: (expedienteRef, { signal }) => ejecutar({
        metodo: "GET",
        ruta: `${RUTA_SEGUIMIENTO_INCORPORACION}?expediente_ref=${encodeURIComponent(expedienteRef)}`,
        signal, estadoEsperado: 200, maximoRespuesta: 512 * 1024, efecto: false,
        validarRespuesta: (datos) => datos,
      }),
    }),
    ...crearFichaGINPIXClienteHTTP({ validarOpciones: validarOpcionesInterno, descargar: ({ ruta, signal, maximoRespuesta }) => ejecutar({
      metodo: "GET", ruta, signal, maximoRespuesta, estadoEsperado: 200, efecto: false, fichero: true,
      validarRespuesta: (json, _estado, contenido) => ({
        tipo: "application/json", nombre: NOMBRE_FICHA_GINPIX, json, contenido,
      }),
    }) }),
    proponerCobertura, decidirCobertura, rectificarCobertura,
    consultarResultadoCobertura, obtenerConfiguracionAnalisis, registrarAnalisis, rectificarAnalisis,
    consultarPreparacionCierreSinCese,
  });
}
