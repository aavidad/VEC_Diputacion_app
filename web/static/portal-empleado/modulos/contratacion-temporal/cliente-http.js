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
import { validarConfiguracionAnalisis, validarReciboAnalisis,
  validarSolicitudRectificacionAnalisis, validarSolicitudRegistroAnalisis } from "./contrato-analisis.js";
import { crearAsignacionClienteHTTP, RUTA_ASIGNACION_CONTRATACION_TEMPORAL } from "./cliente-http-asignacion.js";
import { crearConsultasRRHHClienteHTTP, RUTAS_CONSULTA_RRHH } from "./cliente-http-consultas-rrhh.js";
import { crearInformeJuridicoClienteHTTP, RUTA_PREPARACION_INFORME_JURIDICO } from "./cliente-http-informe-juridico.js";
import { crearFiscalizacionClienteHTTP, RUTA_RESULTADOS_FISCALIZACION } from "./cliente-http-fiscalizacion.js";
import { crearLlamamientoClienteHTTP, RUTAS_LLAMAMIENTO, prefijoErrorLlamamiento, conflictoLlamamientoValido } from "./cliente-http-llamamiento.js";
import { crearResolucionFormalizacionClienteHTTP, RUTA_RESOLUCION_FORMALIZACION } from "./cliente-http-resolucion-formalizacion.js";
import { crearIncorporacionEjercicioClienteHTTP, RUTA_INCORPORACION_EJERCICIO } from "./cliente-http-incorporacion-ejercicio.js";
import { crearFichaGINPIXClienteHTTP, RUTA_FICHA_GINPIX, NOMBRE_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";
import { crearClienteSeguimientoIncorporacion, RUTA_SEGUIMIENTO_INCORPORACION } from "./cliente-http-seguimiento-incorporacion.js";
import { crearClienteAnotacionAdministrativaHTTP, RUTA_ANOTACION_ADMINISTRATIVA, RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA } from "./cliente-http-anotacion-administrativa.js";
import { crearClienteCierreAdministrativoHTTP, RUTA_CIERRE_ADMINISTRATIVO } from "./cliente-http-cierre-administrativo.js";
import { crearClienteSubsanacionReparosHTTP, RUTA_SUBSANACION_REPAROS } from "./cliente-http-subsanacion-reparos.js";
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
const MAXIMO_SOLICITUD_COBERTURA_BYTES = 64 * 1024;
const MAXIMO_SOLICITUD_ANALISIS_BYTES = 64 * 1024;
const MAXIMO_RESPUESTA_ANALISIS_BYTES = 16 * 1024;
const MAXIMO_RESPUESTA_CONFIGURACION_ANALISIS_BYTES = 64 * 1024;
const MAXIMO_RESPUESTA_COBERTURA_BYTES = 256 * 1024;
const MAXIMO_ERROR_BYTES = 16 * 1024;
const MAXIMO_FRAGMENTOS = 4096;
const MAXIMO_FRAGMENTOS_ERROR = 256;
const PATRON_CORRELACION = /^corr_(?:[0-9a-f]{32}|no_disponible)$/u;
const PATRON_REFERENCIA_OPACA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE_CATALOGO = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_INSTANTE_UTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
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
const CODIGOS_RESULTADO_COBERTURA_POR_ESTADO = new Map([
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
  [503, new Set(["servicio_no_disponible"])],
]);
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
  const recibidos = Object.keys(valor);
  return recibidos.length === campos.length
    && recibidos.every((campo) => campos.includes(campo))
    && campos.every((campo) => Object.hasOwn(valor, campo));
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
  return Object.freeze({ estado_actual: dto.estado_actual, preparada_en: dto.preparada_en, preparacion });
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
  conservarBytes = false,
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
      return conservarBytes ? { valor, bytes } : valor;
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
  const rutaBase = ruta.split("?")[0];
  const prefijo = [
    RUTAS_HTTP_CONTRATACION_TEMPORAL.preparacionCierreSinCese,
    RUTA_CIERRE_ADMINISTRATIVO,
  ].includes(rutaBase)
    ? "api.contratacion_temporal.cierre_administrativo.error."
    : rutaBase === RUTA_SUBSANACION_REPAROS
    ? "api.contratacion_temporal.subsanacion_reparos.error."
    : rutaBase === RUTA_ANOTACION_ADMINISTRATIVA
    || rutaBase === RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA
    ? "api.contratacion_temporal.anotacion_administrativa.error."
    : rutaBase === RUTA_FICHA_GINPIX
    ? "api.contratacion_temporal.ficha_ginpix.error."
    : [RUTA_INCORPORACION_EJERCICIO, RUTA_SEGUIMIENTO_INCORPORACION].includes(ruta.split("?")[0])
    ? "api.contratacion_temporal.incorporacion_ejercicio.error."
    : ruta.split("?")[0] === RUTA_RESOLUCION_FORMALIZACION
    ? "api.contratacion_temporal.resolucion_formalizacion.error."
    : prefijoErrorLlamamiento(ruta) ?? (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.alta
    ? "api.contratacion_temporal.alta.error."
    : ruta === RUTA_ASIGNACION_CONTRATACION_TEMPORAL
      ? "api.contratacion_temporal.asignacion.error."
    : ruta === RUTA_PREPARACION_INFORME_JURIDICO
      ? "api.contratacion_temporal.informe_juridico.error."
    : ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.cuadroRRHH
      || ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.detalleRRHH
      ? "api.contratacion_temporal.consulta_rrhh.error."
      : "api.contratacion_temporal.cobertura.error.");
  return clave === `${prefijo}${codigo}`;
}

function codigoValidoParaRuta(ruta, estado, codigo) {
  if (ruta.split("?")[0] === RUTA_CIERRE_ADMINISTRATIVO) {
    if ((estado === 401 && codigo === "autenticacion_requerida")
      || (estado === 403 && codigo === "acceso_denegado")
      || (estado === 409 && [
        "version_en_conflicto",
        "clave_idempotencia_reutilizada",
      ].includes(codigo))) return true;
    return estado !== 409 && CODIGOS_POR_ESTADO.get(estado)?.has(codigo) === true;
  }
  if (ruta.split("?")[0] === RUTA_FICHA_GINPIX && estado === 409) {
    return codigo === "recibo_no_confirmado";
  }
  if (ruta === RUTA_RESOLUCION_FORMALIZACION && estado === 409) {
    return ["conflicto", "version_en_conflicto", "clave_idempotencia_reutilizada"].includes(codigo);
  }
  if (prefijoErrorLlamamiento(ruta) && estado === 409) {
    return conflictoLlamamientoValido(ruta, codigo);
  }
  if (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.resultadoCobertura) {
    return CODIGOS_RESULTADO_COBERTURA_POR_ESTADO
      .get(estado)?.has(codigo) === true;
  }
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

function construirCabeceras(
  HeadersImpl,
  tipoContenido = "application/json; charset=utf-8",
  conCuerpo = true,
) {
  const cabeceras = new HeadersImpl();
  cabeceras.set("Accept", "application/json");
  if (conCuerpo) cabeceras.set("Content-Type", tipoContenido);
  const nombresEsperados = conCuerpo ? "accept,content-type" : "accept";
  if (typeof cabeceras.keys !== "function"
    || [...cabeceras.keys()].map((nombre) => nombre.toLowerCase()).sort()
      .join(",") !== nombresEsperados
    || cabeceras.get("accept") !== "application/json"
    || (conCuerpo && cabeceras.get("content-type") !== tipoContenido)
    || (!conCuerpo && cabeceras.has("content-type"))) {
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
  const consultasResultadoEnCurso = new Map();

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
      ? serializarAcotado(entrada, maximoSolicitud)
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
      );
    } catch (error) {
      const fallo = error instanceof ErrorClienteHTTPContratacionTemporal
        ? error
        : signal?.aborted
          ? errorAborto(signal)
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
          throw await construirErrorRespuesta(respuesta, signal, ruta);
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
          respuesta, signal, maximoRespuesta, MAXIMO_FRAGMENTOS, fichero,
        );
        validada = validarRespuesta(
          fichero ? leida.valor : extraerDatos(leida),
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

  async function ejecutarAnalisis({
    solicitud,
    opciones,
    ruta,
    operacion,
    validarSolicitud,
  }) {
    const { signal } = validarOpciones(opciones);
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
    const { signal } = validarOpciones(opciones);
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
      ({ signal } = validarOpciones(opciones));
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
    const { signal } = validarOpciones(opciones);
    const ruta = `${RUTAS_HTTP_CONTRATACION_TEMPORAL.preparacionCierreSinCese}?expediente_ref=${encodeURIComponent(entrada.expediente_ref)}&seguimiento_ref=${encodeURIComponent(entrada.seguimiento_ref)}`;
    return ejecutar({
      metodo: "GET", ruta, signal, estadoEsperado: 200, maximoRespuesta: 16 * 1024,
      efecto: false,
      validarRespuesta: (respuesta) => adaptarPreparacionCierreSinCese(respuesta, entrada),
    });
  }

  return Object.freeze({
    modo: "http",
    ...crearAltaClienteHTTP({ ejecutar, validarOpciones }),
    ...crearConsultasRRHHClienteHTTP({ ejecutar, validarOpciones }),
    ...crearAsignacionClienteHTTP({ ejecutar, validarOpciones, serializarAcotado }),
    ...crearInformeJuridicoClienteHTTP({ ejecutar, validarOpciones, serializarAcotado }),
    ...crearFiscalizacionClienteHTTP({ ejecutar, validarOpciones, serializarAcotado }),
    ...crearClienteSubsanacionReparosHTTP({ ejecutar, validarOpciones, serializarAcotado }),
    ...crearLlamamientoClienteHTTP({ ejecutar, validarOpciones }),
    ...crearResolucionFormalizacionClienteHTTP({ ejecutar, validarOpciones }),
    ...crearIncorporacionEjercicioClienteHTTP({ ejecutar, validarOpciones }),
    anotacionAdministrativa: crearClienteAnotacionAdministrativaHTTP({ ejecutar, validarOpciones }),
    ...crearClienteCierreAdministrativoHTTP({ ejecutar, validarOpciones }),
    seguimientoIncorporacion: crearClienteSeguimientoIncorporacion({
      validarOpciones,
      consultar: (expedienteRef, { signal }) => ejecutar({
        metodo: "GET",
        ruta: `${RUTA_SEGUIMIENTO_INCORPORACION}?expediente_ref=${encodeURIComponent(expedienteRef)}`,
        signal, estadoEsperado: 200, maximoRespuesta: 512 * 1024, efecto: false,
        validarRespuesta: (datos) => datos,
      }),
    }),
    ...crearFichaGINPIXClienteHTTP({ validarOpciones, descargar: ({ ruta, signal, maximoRespuesta }) => ejecutar({
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
