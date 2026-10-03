import { IDIOMA_ACTUAL, IDIOMAS_DISPONIBLES } from "../../../comun/idioma.js";
import { nombreArchivoExportacionServicios } from "./i18n-exportacion-servicios.js?v=20261004-personal-historia-v1";

export const RUTA_EXPORTACION_SERVICIOS = "/api/interna/personal/mi-ficha/servicios/exportaciones";
export const MAXIMO_EXPORTACION_SERVICIOS = 1024 * 1024;
export const MIME_EXPORTACION_SERVICIOS = "text/csv; charset=utf-8";
const RECIBO = /^fichapropia:[0-9a-f-]{36}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u;

export class ErrorExportacionServicios extends Error {
  constructor(codigo, estado = 0) { super(codigo); this.name = "ErrorExportacionServicios"; this.codigo = codigo; this.estado = estado; }
}
const error = (codigo, estado = 0) => new ErrorExportacionServicios(codigo, estado);

export function referenciaExportacionServiciosValida(reciboRef, corte) {
  if (typeof reciboRef !== "string" || !RECIBO.test(reciboRef) || !corte ||
      Object.keys(corte).length !== 2 || typeof corte.vigente_en !== "string" ||
      !/^\d{4}-\d{2}-\d{2}$/u.test(corte.vigente_en) || corte.vigente_en.startsWith("0000") ||
      typeof corte.conocido_en !== "string" || !INSTANTE.test(corte.conocido_en) || !Number.isFinite(Date.parse(corte.conocido_en))) return false;
  const fecha = new Date(`${corte.vigente_en}T12:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === corte.vigente_en;
}

async function leerBytes(respuesta, esperado, maximo = MAXIMO_EXPORTACION_SERVICIOS) {
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw error("respuesta_no_valida", respuesta.status);
  const partes = []; let total = 0; let terminada = false;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) { terminada = true; break; }
      if (!(value instanceof Uint8Array)) throw error("respuesta_no_valida", respuesta.status);
      total += value.byteLength;
      if (total > esperado || total > maximo) throw error("respuesta_no_valida", respuesta.status);
      partes.push(value);
    }
  } finally {
    if (!terminada) { try { await lector.cancel(); } catch {} }
    lector.releaseLock();
  }
  if (total !== esperado) throw error("respuesta_no_valida", respuesta.status);
  const bytes = new Uint8Array(total); let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  return bytes;
}

/** Exporta el recibo elegido. Nunca consulta de nuevo ni envía filas o identidad. */
export function crearClienteExportacionServicios({ fetchImpl = globalThis.fetch, cryptoImpl = globalThis.crypto, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !cryptoImpl?.subtle?.digest || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw error("no_disponible");
  return Object.freeze({
    async exportar({ reciboRef, corte, idioma = IDIOMA_ACTUAL, signal } = {}) {
      if (!referenciaExportacionServiciosValida(reciboRef, corte)) throw error("sin_consulta");
      if (!IDIOMAS_DISPONIBLES.some(({ codigo }) => codigo === idioma)) throw error("sin_consulta");
      if (signal?.aborted) throw error("operacion_abortada");
      const controlador = new AbortController();
      const abortar = () => controlador.abort();
      signal?.addEventListener("abort", abortar, { once: true });
      const temporizador = setTimeout(abortar, plazoMs);
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_EXPORTACION_SERVICIOS, {
          method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
          headers: { "Content-Type": "application/json", Accept: MIME_EXPORTACION_SERVICIOS },
          body: JSON.stringify({ recibo_ref: reciboRef, corte: { vigente_en: corte.vigente_en, conocido_en: corte.conocido_en }, idioma }),
          signal: controlador.signal,
        });
        const estado = respuesta?.status || 0;
        if (estado !== 200 || respuesta?.ok !== true || respuesta.redirected === true) {
          const codigos = { 400: "peticion_invalida", 401: "autenticacion_requerida", 403: "acceso_denegado", 404: "no_encontrada", 503: "no_disponible" };
          const longitudError = respuesta?.headers?.get("Content-Length");
          if (respuesta?.headers?.get("Content-Type") !== "application/json; charset=utf-8" ||
              !/^[1-9]\d{0,3}$/u.test(longitudError || "") || Number(longitudError) > 4096 || !Object.hasOwn(codigos, estado)) throw error("respuesta_no_valida", estado);
          const bruto = await leerBytes(respuesta, Number(longitudError), 4096);
          let sobre;
          try { sobre = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bruto)); } catch { throw error("respuesta_no_valida", estado); }
          if (!sobre || Object.keys(sobre).length !== 1 || sobre.error !== codigos[estado]) throw error("respuesta_no_valida", estado);
          throw error([401, 403, 404].includes(estado) ? "denegado" : estado === 400 ? "sin_consulta" : "no_disponible", estado);
        }
        const longitud = respuesta.headers?.get("Content-Length");
        const huella = respuesta.headers?.get("X-Content-SHA256");
        const disposicion = /^attachment;\s*filename=(?:"([a-zA-Z0-9_-]+\.csv)"|([a-zA-Z0-9_-]+\.csv))$/u.exec(respuesta.headers?.get("Content-Disposition") || "");
        if (respuesta.headers?.get("Content-Type") !== MIME_EXPORTACION_SERVICIOS ||
            (disposicion?.[1] || disposicion?.[2]) !== nombreArchivoExportacionServicios ||
            respuesta.headers.get("X-Recibo-Ref") !== reciboRef ||
            !/^[1-9]\d{0,6}$/u.test(longitud || "") || Number(longitud) > MAXIMO_EXPORTACION_SERVICIOS ||
            !/^[0-9a-f]{64}$/u.test(huella || "")) throw error("respuesta_no_valida", estado);
        const bytes = await leerBytes(respuesta, Number(longitud));
        const digest = new Uint8Array(await cryptoImpl.subtle.digest("SHA-256", bytes));
        const comprobada = Array.from(digest, (valor) => valor.toString(16).padStart(2, "0")).join("");
        if (controlador.signal.aborted) throw error(signal?.aborted ? "operacion_abortada" : "no_disponible");
        if (comprobada !== huella) throw error("respuesta_no_valida", estado);
        return Object.freeze({ bytes, mime: MIME_EXPORTACION_SERVICIOS, nombre: nombreArchivoExportacionServicios, huella, reciboRef });
      } catch (causa) {
        if (signal?.aborted) throw error("operacion_abortada");
        if (causa instanceof ErrorExportacionServicios) throw causa;
        throw error("no_disponible");
      } finally {
        try { await respuesta?.body?.cancel?.(); } catch {}
        clearTimeout(temporizador); signal?.removeEventListener("abort", abortar);
      }
    },
  });
}
