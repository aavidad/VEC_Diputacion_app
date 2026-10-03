/** Recupera el PDF original que el servidor custodia una sola vez. */
import { huellaPDFFirmado } from "./firma-vec-api.js?v=20261003-ct-firma-v2-v1";

export const RUTA_ORIGINAL_FIRMABLE = "/api/vec/contratacion-temporal/firmas-documento/original";
const MAXIMO_PDF = 1 << 20;
const MAXIMO_RESPUESTA = Math.ceil(MAXIMO_PDF / 3) * 4 + 16384;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const DOCUMENTO = /^[a-z][a-z0-9_]{1,63}$/u;
const DOCREF = /^ref:[0-9a-f]{64}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const entero = (v) => Number.isSafeInteger(v) && v > 0;
const exacto = (v, claves) => v !== null && typeof v === "object"
  && Object.getPrototypeOf(v) === Object.prototype && Object.keys(v).length === claves.length
  && claves.every((k) => Object.hasOwn(v, k));

export class ErrorOriginalFirmable extends Error {
  constructor(codigo) { super(codigo); this.name = "ErrorOriginalFirmable"; this.codigo = codigo; }
}

async function leerJSON(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")
    || (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA))) {
    throw new ErrorOriginalFirmable("resultado_no_confiable");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new ErrorOriginalFirmable("resultado_no_confiable");
  const partes = []; let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_RESPUESTA) throw new ErrorOriginalFirmable("resultado_no_confiable");
      partes.push(value);
    }
  } finally { void lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total); let inicio = 0;
  for (const parte of partes) { bytes.set(parte, inicio); inicio += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorOriginalFirmable("resultado_no_confiable"); }
}

async function validar(datos, solicitud) {
  const claves = ["esquema", "expediente_ref", "version_observada", "documento", "original_ref",
    "original_version", "original_sha256", "tipo_ref", "mime", "pdf_base64", "documento_custodiado"];
  const c = datos?.documento_custodiado;
  if (!exacto(datos, claves) || datos.esquema !== "vec.contratacion-temporal.original-firmable.v1"
    || datos.expediente_ref !== solicitud.expedienteRef || datos.version_observada !== solicitud.version
    || datos.documento !== solicitud.documento || !DOCREF.test(datos.original_ref)
    || datos.original_version !== solicitud.version || !HUELLA.test(datos.original_sha256)
    || !REFERENCIA.test(datos.tipo_ref) || datos.mime !== "application/pdf"
    || typeof datos.pdf_base64 !== "string" || datos.pdf_base64.length > Math.ceil(MAXIMO_PDF / 3) * 4
    || !exacto(c, ["expediente_ref", "documento_ref", "version", "huella_sha256"])
    || !DOCREF.test(c.expediente_ref) || c.documento_ref !== datos.original_ref
    || c.version !== datos.original_version || c.huella_sha256 !== datos.original_sha256) {
    throw new ErrorOriginalFirmable("resultado_no_confiable");
  }
  let contenido;
  try {
    const binario = atob(datos.pdf_base64);
    if (btoa(binario) !== datos.pdf_base64) throw new Error();
    contenido = Uint8Array.from(binario, (b) => b.charCodeAt(0));
  } catch { throw new ErrorOriginalFirmable("resultado_no_confiable"); }
  if (contenido.length < 10 || contenido.length > MAXIMO_PDF
    || new TextDecoder().decode(contenido.subarray(0, 5)) !== "%PDF-"
    || !new TextDecoder().decode(contenido.subarray(-1024)).includes("%%EOF")
    || await huellaPDFFirmado(contenido) !== datos.original_sha256) throw new ErrorOriginalFirmable("resultado_no_confiable");
  return Object.freeze({ contexto: Object.freeze({ expedienteRef: solicitud.expedienteRef, version: solicitud.version, documento: solicitud.documento }), originalRef: datos.original_ref, originalVersion: datos.original_version,
    originalHuella: datos.original_sha256, tipoRef: datos.tipo_ref,
    documentoCustodiado: Object.freeze({ ...c }), contenido });
}

export function crearClienteOriginalFirmable({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({ async preparar(solicitud, { signal } = {}) {
    const { expedienteRef, version, documento } = solicitud ?? {};
    if (!REFERENCIA.test(expedienteRef ?? "") || !entero(version) || !DOCUMENTO.test(documento ?? "")) {
      throw new ErrorOriginalFirmable("contenido_no_valido");
    }
    if (signal?.aborted) throw new ErrorOriginalFirmable("operacion_abortada");
    if (typeof fetchImpl !== "function") throw new ErrorOriginalFirmable("servicio_no_disponible");
    let respuesta;
    try {
      respuesta = await fetchImpl(RUTA_ORIGINAL_FIRMABLE, {
        method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ expediente_ref: expedienteRef, version_observada: version, documento }),
        signal, mode: "same-origin", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      });
    } catch { throw new ErrorOriginalFirmable(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible"); }
    if (signal?.aborted || respuesta.redirected || respuesta.status !== 200) {
      void respuesta.body?.cancel?.().catch(() => {});
      throw new ErrorOriginalFirmable(signal?.aborted ? "operacion_abortada"
        : [401, 403, 404].includes(respuesta.status) ? "acceso_denegado" : "servicio_no_disponible");
    }
    const envoltorio = await leerJSON(respuesta);
    if (!exacto(envoltorio, ["data"])) throw new ErrorOriginalFirmable("resultado_no_confiable");
    const original = await validar(envoltorio.data, solicitud);
    if (signal?.aborted) throw new ErrorOriginalFirmable("operacion_abortada");
    return original;
  } });
}

/** Composición explícita: prepara bytes, sin conceder ni registrar firmas. */
export function crearDependenciasPreparacionFirma({ clienteOriginal, clientePreflight, crearDocumentos } = {}) {
  if (typeof clienteOriginal?.preparar !== "function" || typeof clientePreflight?.consultar !== "function"
    || typeof crearDocumentos !== "function") throw new ErrorOriginalFirmable("servicio_no_disponible");
  return Object.freeze({
    clientePreflight,
    obtenerVinculoOriginal(solicitud, opciones) { return clienteOriginal.preparar(solicitud, opciones); },
    async obtenerOriginal(solicitud, { signal, vinculoOriginal: vinculo } = {}) {
      if (signal?.aborted) throw new ErrorOriginalFirmable("operacion_abortada");
      if (!vinculo || ["expedienteRef", "version", "documento"].some((k) => vinculo.contexto?.[k] !== solicitud?.[k])
        || vinculo.originalRef !== solicitud.originalRef || vinculo.originalVersion !== solicitud.originalVersion
        || vinculo.originalHuella !== solicitud.originalHuella) throw new ErrorOriginalFirmable("resultado_no_confiable");
      if (solicitud.pasoOrden === 1) {
        if (solicitud.revisionEntradaRef !== vinculo.originalRef || solicitud.revisionEntradaVersion !== vinculo.originalVersion
          || solicitud.revisionEntradaHuella !== vinculo.originalHuella) throw new ErrorOriginalFirmable("resultado_no_confiable");
        if (await huellaPDFFirmado(vinculo.contenido) !== vinculo.originalHuella) throw new ErrorOriginalFirmable("resultado_no_confiable");
        return new Uint8Array(vinculo.contenido);
      }
      if (solicitud.pasoOrden !== 2 || solicitud.revisionEntradaRef === vinculo.originalRef
        || !DOCREF.test(vinculo.documentoCustodiado?.expediente_ref ?? "")) throw new ErrorOriginalFirmable("resultado_no_confiable");
      const fuente = crearDocumentos({ expedienteRef: vinculo.documentoCustodiado.expediente_ref });
      const archivo = await fuente.descargar(solicitud.revisionEntradaRef, { version: solicitud.revisionEntradaVersion,
        huella: solicitud.revisionEntradaHuella, mime: "application/pdf", signal });
      if (signal?.aborted) throw new ErrorOriginalFirmable("operacion_abortada");
      if (!(archivo?.contenido instanceof Uint8Array) || archivo.tipo !== "application/pdf"
        || archivo.contenido.length > MAXIMO_PDF || await huellaPDFFirmado(archivo.contenido) !== solicitud.revisionEntradaHuella) {
        throw new ErrorOriginalFirmable("resultado_no_confiable");
      }
      return new Uint8Array(archivo.contenido);
    },
  });
}
