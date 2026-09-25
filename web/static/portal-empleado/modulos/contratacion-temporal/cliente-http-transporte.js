/** Funciones auxiliares de red, lectura incremental y validación HTTP para contratación temporal. */

import { RUTA_ASIGNACION_CONTRATACION_TEMPORAL } from "./cliente-http-asignacion.js";
import { RUTA_PREPARACION_INFORME_JURIDICO } from "./cliente-http-informe-juridico.js";
import { conflictoLlamamientoValido, prefijoErrorLlamamiento } from "./cliente-http-llamamiento.js";
import { RUTA_RESOLUCION_FORMALIZACION } from "./cliente-http-resolucion-formalizacion.js";
import { RUTA_INCORPORACION_EJERCICIO } from "./cliente-http-incorporacion-ejercicio.js";
import { RUTA_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";
import { RUTA_SEGUIMIENTO_INCORPORACION } from "./cliente-http-seguimiento-incorporacion.js";
import { RUTA_ANOTACION_ADMINISTRATIVA, RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA } from "./cliente-http-anotacion-administrativa.js";
import { RUTA_CIERRE_ADMINISTRATIVO } from "./cliente-http-cierre-administrativo.js";
import { RUTA_SUBSANACION_REPAROS } from "./cliente-http-subsanacion-reparos.js";
import { CONFLICTOS_SEGUIMIENTO_CESE, RUTAS_SEGUIMIENTO_CESE } from "./cliente-http-seguimiento-cese.js";

export const MAXIMO_ERROR_BYTES = 16 * 1024;
export const MAXIMO_FRAGMENTOS = 4096;
export const MAXIMO_FRAGMENTOS_ERROR = 256;
export const PATRON_CORRELACION = /^corr_(?:[0-9a-f]{32}|no_disponible)$/u;

export const CODIGOS_POR_ESTADO = new Map([
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
    "preparacion_pendiente",
  ])],
  [413, new Set(["peticion_demasiado_grande"])],
  [415, new Set(["tipo_contenido_no_admitido"])],
  [422, new Set(["contenido_no_valido"])],
  [500, new Set(["error_interno"])],
  [502, new Set(["resultado_no_confiable"])],
  [503, new Set(["servicio_no_disponible", "operacion_pendiente"])],
  [504, new Set(["plazo_agotado"])],
]);

export const CODIGOS_RESULTADO_COBERTURA_POR_ESTADO = new Map([
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

export function validarSignal(signal, errorCliente) {
  if (signal === undefined) return undefined;
  if (!signal || typeof signal !== "object"
    || typeof signal.aborted !== "boolean"
    || typeof signal.addEventListener !== "function"
    || typeof signal.removeEventListener !== "function") {
    throw errorCliente("signal_no_valida");
  }
  if (signal.aborted) throw errorCliente("operacion_abortada");
  return signal;
}

export function validarOpciones(opciones, esRegistro, errorCliente) {
  if (opciones === undefined) return Object.freeze({ signal: undefined });
  if (!esRegistro(opciones)
    || Object.keys(opciones).some((campo) => campo !== "signal")) {
    throw errorCliente("opciones_no_validas");
  }
  return Object.freeze({ signal: validarSignal(opciones.signal, errorCliente) });
}

export function errorAborto(signal, errorCliente) {
  void signal;
  return errorCliente("operacion_abortada");
}

export async function ejecutarAbortable(iniciar, signal, alAbortar, alResolverTardio, errorCliente) {
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
      reject(errorAborto(signal, errorCliente));
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

export async function cancelarRespuesta(respuesta, lector = null) {
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

export function longitudDeclarada(respuesta, maximo, errorCliente) {
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

export function validarTipoJSON(respuesta, errorCliente) {
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

export async function leerJSONAcotado(
  respuesta,
  signal,
  maximoBytes,
  maximoFragmentos,
  conservarBytes = false,
  errorCliente,
) {
  validarTipoJSON(respuesta, errorCliente);
  const declarada = longitudDeclarada(respuesta, maximoBytes, errorCliente);
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
        null,
        errorCliente,
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

export function serializarAcotado(valor, maximoBytes, errorCliente) {
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

export function extraerDatos(envoltorio, errorCliente, exigirCamposExactos, esRegistro) {
  if (!exigirCamposExactos(envoltorio, ["data"])
    || !esRegistro(envoltorio.data)) {
    throw errorCliente("respuesta_incompatible");
  }
  return envoltorio.data;
}

export function claveI18nValida(ruta, codigo, clave, rutas) {
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
    rutas.preparacionCierreSinCese,
    RUTA_CIERRE_ADMINISTRATIVO,
  ].includes(rutaBase)
    ? "api.contratacion_temporal.cierre_administrativo.error."
    : rutaBase === RUTA_SUBSANACION_REPAROS
    ? "api.contratacion_temporal.subsanacion_reparos.error."
    : RUTAS_SEGUIMIENTO_CESE.includes(rutaBase)
    ? "api.contratacion_temporal.seguimiento.error."
    : rutaBase === RUTA_ANOTACION_ADMINISTRATIVA
    || rutaBase === RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA
    ? "api.contratacion_temporal.anotacion_administrativa.error."
    : rutaBase === RUTA_FICHA_GINPIX
    ? "api.contratacion_temporal.ficha_ginpix.error."
    : [RUTA_INCORPORACION_EJERCICIO, RUTA_SEGUIMIENTO_INCORPORACION].includes(ruta.split("?")[0])
    ? "api.contratacion_temporal.incorporacion_ejercicio.error."
    : ruta.split("?")[0] === RUTA_RESOLUCION_FORMALIZACION
    ? "api.contratacion_temporal.resolucion_formalizacion.error."
    : prefijoErrorLlamamiento(ruta) ?? (ruta === rutas.alta
    ? "api.contratacion_temporal.alta.error."
    : ruta === RUTA_ASIGNACION_CONTRATACION_TEMPORAL
      ? "api.contratacion_temporal.asignacion.error."
    : ruta === RUTA_PREPARACION_INFORME_JURIDICO
      ? "api.contratacion_temporal.informe_juridico.error."
    : ruta === rutas.cuadroRRHH
      || ruta === rutas.detalleRRHH
      ? "api.contratacion_temporal.consulta_rrhh.error."
      : "api.contratacion_temporal.cobertura.error.");
  return clave === `${prefijo}${codigo}`;
}

export function codigoValidoParaRuta(ruta, estado, codigo, rutas) {
  if (RUTAS_SEGUIMIENTO_CESE.includes(ruta.split("?")[0]) && estado === 409) {
    return CONFLICTOS_SEGUIMIENTO_CESE.includes(codigo);
  }
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
  if (ruta === rutas.resultadoCobertura) {
    return CODIGOS_RESULTADO_COBERTURA_POR_ESTADO
      .get(estado)?.has(codigo) === true;
  }
  if (!CODIGOS_POR_ESTADO.get(estado)?.has(codigo)) return false;
  if (codigo === "operacion_pendiente"
    && ruta === rutas.propuestaCobertura) {
    return false;
  }
  if (estado !== 409) return true;
  if (ruta === rutas.alta) return codigo === "clave_idempotencia_reutilizada";
  if (ruta.split("?")[0] === rutas.incorporacionEjercicio) {
    return codigo === "conflicto" || codigo === "preparacion_pendiente";
  }
  return codigo === "conflicto";
}

export async function construirErrorRespuesta(respuesta, signal, ruta, errorCliente, ErrorClienteClass, exigirCamposExactos, rutas) {
  let envoltorio;
  try {
    envoltorio = await leerJSONAcotado(
      respuesta,
      signal,
      MAXIMO_ERROR_BYTES,
      MAXIMO_FRAGMENTOS_ERROR,
      false,
      errorCliente,
    );
  } catch (error) {
    if (error instanceof ErrorClienteClass
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
    || !codigoValidoParaRuta(ruta, respuesta.status, detalle.codigo, rutas)
    || !claveI18nValida(ruta, detalle.codigo, detalle.clave_i18n, rutas)
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

export function construirCabeceras(
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
