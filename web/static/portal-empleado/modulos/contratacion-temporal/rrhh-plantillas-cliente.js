/** Transporte de la configuración RRHH. La identidad y V3 se resuelven en el servidor. */
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-circular-v3";
import { MENSAJES_RRHH_PLANTILLAS_ES, MENSAJES_RRHH_PLANTILLAS_EN } from "./rrhh-plantillas-i18n.js";
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";

export const RUTA_RRHH_PLANTILLAS = "/api/vec/contratacion-temporal/plantillas";
export const RUTA_RRHH_PLANTILLAS_ENTRADAS = `${RUTA_RRHH_PLANTILLAS}/entradas`;
export const RUTA_RRHH_PLANTILLAS_PUBLICAR = `${RUTA_RRHH_PLANTILLAS}/publicar`;

const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const RECIBO = /^recibo:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const MAXIMO_RESPUESTA = 17_000_000;
const MAXIMO_PETICION = 256 * 1024;
const t = crearTraductorContratacionTemporal(IDIOMA_ACTUAL === IDIOMA_POR_DEFECTO
  ? MENSAJES_RRHH_PLANTILLAS_ES : MENSAJES_RRHH_PLANTILLAS_EN);

export class ErrorPlantillasRRHH extends Error {
  constructor(codigo, estado = 0, resultadoIndeterminado = false) {
    super(`plantillas RRHH: ${codigo}`);
    this.name = "ErrorPlantillasRRHH";
    this.codigo = codigo;
    this.estado = estado;
    this.resultadoIndeterminado = resultadoIndeterminado;
  }
}

const objeto = (valor) => valor !== null && typeof valor === "object" && !Array.isArray(valor)
  && Object.getPrototypeOf(valor) === Object.prototype;
const entero = (valor, minimo) => Number.isSafeInteger(valor) && valor >= minimo;
const texto = (valor, maximo, vacio = false) => typeof valor === "string" && valor.length <= maximo
  && (vacio || valor.length > 0) && valor.trim() === valor && !/[\u0000-\u0008\u000b-\u001f\u007f]/u.test(valor);
const instante = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/u.test(valor)
  && Number.isFinite(Date.parse(valor));

export function validarCatalogoPlantillas(valor) {
  if (!objeto(valor) || !texto(valor.id, 160) || !texto(valor.fuente_ref, 160)
    || !entero(valor.version, 1) || !entero(valor.revision, 1)
    || !["borrador", "publicado"].includes(valor.estado) || typeof valor.huella_sha256 !== "string"
    || !HUELLA.test(valor.huella_sha256) || !Array.isArray(valor.entradas) || valor.entradas.length > 33) {
    throw new ErrorPlantillasRRHH("respuesta_incompatible");
  }
  const claves = new Set();
  for (const entrada of valor.entradas) {
    if (!objeto(entrada) || !CLAVE.test(entrada.clave) || claves.has(entrada.clave)
      || !texto(entrada.etiqueta, 256) || !texto(entrada.descripcion ?? "", 4000, true)
      || !entero(entrada.orden, 0) || !instante(entrada.vigente_desde)
      || (entrada.vigente_hasta && !instante(entrada.vigente_hasta))
      || !objeto(entrada.atributos) || Object.keys(entrada.atributos).length > 96
      || Object.entries(entrada.atributos).some(([clave, dato]) => !CLAVE.test(clave) || !texto(dato, 65536))) {
      throw new ErrorPlantillasRRHH("respuesta_incompatible");
    }
    claves.add(entrada.clave);
  }
  return valor;
}

export function validarConsultaPlantillas(valor) {
  if (!objeto(valor) || !Object.hasOwn(valor, "borrador") || !Object.hasOwn(valor, "publicado")
    || typeof valor.puede_editar !== "boolean" || typeof valor.puede_publicar !== "boolean") {
    throw new ErrorPlantillasRRHH("respuesta_incompatible");
  }
  return {
    borrador: valor.borrador === null ? null : validarCatalogoPlantillas(valor.borrador),
    publicado: valor.publicado === null ? null : validarCatalogoPlantillas(valor.publicado),
    puede_editar: valor.puede_editar,
    puede_publicar: valor.puede_publicar,
  };
}

export function validarRespuestaGuardadoPlantillas(valor, solicitud, estadoEsperado, estadoHTTP) {
  const operacion = estadoEsperado === "publicado" ? "publicar" : "editar";
  const versionResultado = estadoEsperado === "publicado" || solicitud?.revision_esperada > 0
    ? solicitud?.version_esperada : solicitud?.version_esperada + 1;
  const revisionResultado = estadoEsperado === "publicado" ? solicitud?.revision_esperada
    : solicitud?.revision_esperada === 0 ? 1 : solicitud?.revision_esperada + 1;
  if (!objeto(solicitud) || !UUID.test(solicitud.clave_idempotencia)
    || !entero(versionResultado, 1) || !entero(revisionResultado, 1)
    || !objeto(valor) || !objeto(valor.recibo) || typeof valor.recibo.recibo_ref !== "string"
    || valor.recibo.recibo_ref.length !== 43 || !RECIBO.test(valor.recibo.recibo_ref)
    || valor.recibo.clave_idempotencia !== solicitud.clave_idempotencia
    || valor.recibo.operacion !== operacion
    || valor.recibo.version !== versionResultado || valor.recibo.revision !== revisionResultado
    || !HUELLA.test(valor.recibo.catalogo_huella_sha256)
    || !instante(valor.recibo.registrado_en)
    || !(estadoHTTP === 201 && valor.recibo.estado_replay === "registrado"
      || estadoHTTP === 200 && valor.recibo.estado_replay === "replay")) {
    throw new ErrorPlantillasRRHH("respuesta_incompatible", 0, true);
  }
  let catalogo;
  try { catalogo = validarCatalogoPlantillas(valor.catalogo); }
  catch { throw new ErrorPlantillasRRHH("respuesta_incompatible", 0, true); }
  if (catalogo.estado !== estadoEsperado || catalogo.version !== versionResultado
    || catalogo.revision !== revisionResultado
    || catalogo.huella_sha256 !== valor.recibo.catalogo_huella_sha256) {
    throw new ErrorPlantillasRRHH("respuesta_incompatible", 0, true);
  }
  return { catalogo, recibo: valor.recibo };
}

async function leerAcotado(respuesta, signal) {
  const tipo = respuesta.headers?.get?.("content-type") ?? "";
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(tipo)) throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status);
  const longitud = Number(respuesta.headers?.get?.("content-length"));
  if (Number.isFinite(longitud) && longitud > MAXIMO_RESPUESTA) throw new ErrorPlantillasRRHH("respuesta_excesiva", respuesta.status);
  if (!respuesta.body?.getReader) throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status);
  const lector = respuesta.body.getReader();
  let bytes = new Uint8Array(64 * 1024);
  let tamano = 0;
  let vacios = 0;
  try {
    while (true) {
      if (signal?.aborted) throw new DOMException(t("plantillas_rrhh_lectura_cancelada"), "AbortError");
      const { done, value } = await lector.read();
      if (done) break;
      if (!(value instanceof Uint8Array)) throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status);
      if (value.byteLength === 0) {
        if (++vacios > 1024) throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status);
        continue;
      }
      vacios = 0;
      const siguiente = tamano + value.byteLength;
      if (siguiente > MAXIMO_RESPUESTA) throw new ErrorPlantillasRRHH("respuesta_excesiva", respuesta.status);
      if (siguiente > bytes.byteLength) {
        const ampliados = new Uint8Array(Math.min(MAXIMO_RESPUESTA, Math.max(siguiente, bytes.byteLength * 2)));
        ampliados.set(bytes.subarray(0, tamano));
        bytes = ampliados;
      }
      bytes.set(value, tamano);
      tamano = siguiente;
    }
  } catch (error) {
    await lector.cancel().catch(() => {});
    throw error;
  } finally {
    lector.releaseLock();
  }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes.subarray(0, tamano))); }
  catch { throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status); }
}

function validarSolicitud(valor) {
  if (!objeto(valor) || !UUID.test(valor.clave_idempotencia) || !entero(valor.version_esperada, 0)
    || !entero(valor.revision_esperada, 0) || !texto(valor.motivo, 4000)
    || !texto(valor.fuente_ref, 160)
    || !objeto(valor.entrada) || !CLAVE.test(valor.entrada.clave)) {
    throw new TypeError("petición de plantilla no válida");
  }
  const cuerpo = JSON.stringify(valor);
  if (new TextEncoder().encode(cuerpo).byteLength > MAXIMO_PETICION) throw new TypeError("petición de plantilla excesiva");
  return cuerpo;
}

function validarSolicitudPublicacion(valor) {
  if (!objeto(valor) || !UUID.test(valor.clave_idempotencia) || !entero(valor.version_esperada, 1)
    || !entero(valor.revision_esperada, 1) || !texto(valor.motivo, 4000)
    || !texto(valor.aprobacion_ref, 160)) throw new TypeError("petición de publicación no válida");
  const cuerpo = JSON.stringify(valor);
  if (new TextEncoder().encode(cuerpo).byteLength > MAXIMO_PETICION) throw new TypeError("petición de publicación excesiva");
  return cuerpo;
}

export function crearClientePlantillasRRHH({ fetchImpl = globalThis.fetch, HeadersImpl = globalThis.Headers } = {}) {
  if (typeof fetchImpl !== "function" || typeof HeadersImpl !== "function") throw new TypeError("transporte no disponible");
  async function solicitar(ruta, metodo, cuerpo, signal) {
    if (signal?.aborted) throw new DOMException(t("plantillas_rrhh_peticion_cancelada"), "AbortError");
    const headers = new HeadersImpl({ Accept: "application/json" });
    if (metodo === "POST") headers.set("Content-Type", "application/json");
    let respuesta;
    try {
      respuesta = await fetchImpl(ruta, {
        method: metodo, headers, ...(cuerpo ? { body: cuerpo } : {}), signal,
        credentials: "same-origin", mode: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer",
      });
    } catch (error) {
      if (signal?.aborted) throw error;
      throw new ErrorPlantillasRRHH("resultado_indeterminado", 0, metodo === "POST");
    }
    if (respuesta.redirected || !Number.isInteger(respuesta.status)) {
      throw new ErrorPlantillasRRHH("respuesta_incompatible", 0, metodo === "POST");
    }
    if (respuesta.status >= 400) {
      let codigo = "error_http";
      try { const detalle = await leerAcotado(respuesta, signal); codigo = detalle?.error?.codigo ?? detalle?.codigo ?? codigo; }
      catch { /* El estado sigue siendo fiable; el contenido se descarta. */ }
      if (signal?.aborted) throw new DOMException(t("plantillas_rrhh_lectura_cancelada"), "AbortError");
      throw new ErrorPlantillasRRHH(codigo, respuesta.status, metodo === "POST" && respuesta.status >= 500);
    }
    if (metodo === "GET" ? respuesta.status !== 200 : ![200, 201].includes(respuesta.status)) {
      throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status, metodo === "POST");
    }
    try { return { datos: await leerAcotado(respuesta, signal), estado: respuesta.status }; }
    catch (error) {
      if (signal?.aborted) throw error;
      throw new ErrorPlantillasRRHH("respuesta_incompatible", respuesta.status, metodo === "POST");
    }
  }
  return Object.freeze({
    async consultar({ signal } = {}) {
      const { datos } = await solicitar(RUTA_RRHH_PLANTILLAS, "GET", undefined, signal);
      return validarConsultaPlantillas(datos);
    },
    async guardar(solicitud, { signal } = {}) {
      const respuesta = await solicitar(RUTA_RRHH_PLANTILLAS_ENTRADAS, "POST", validarSolicitud(solicitud), signal);
      return validarRespuestaGuardadoPlantillas(respuesta.datos, solicitud, "borrador", respuesta.estado);
    },
    async publicar(solicitud, { signal } = {}) {
      const respuesta = await solicitar(RUTA_RRHH_PLANTILLAS_PUBLICAR, "POST", validarSolicitudPublicacion(solicitud), signal);
      return validarRespuestaGuardadoPlantillas(respuesta.datos, solicitud, "publicado", respuesta.estado);
    },
  });
}
