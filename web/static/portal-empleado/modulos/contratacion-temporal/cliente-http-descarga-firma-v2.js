/** Descarga nominal de un PDF firmado, cotejado con la historia R5 visible. */
export const RUTA_DESCARGA_FIRMA_V2 = "/api/vec/contratacion-temporal/firmas-documento/descargas-v2";
const ESQUEMA = "vec.contratacion-temporal.descarga-firma-r5.v2";
const MAXIMO_PDF = 1 << 20;
const MAXIMO_RESPUESTA = Math.ceil(MAXIMO_PDF / 3) * 4 + 16 * 1024;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const DOCUMENTO = /^[a-z][a-z0-9_]{1,63}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const BASE64 = /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/u;
const entero = (x) => Number.isSafeInteger(x) && x > 0;
const referencia = (x) => typeof x === "string" && REFERENCIA.test(x);
const objeto = (x) => x !== null && typeof x === "object" && Object.getPrototypeOf(x) === Object.prototype;
const exacto = (x, campos) => objeto(x) && Object.keys(x).length === campos.length
  && campos.every((campo) => Object.hasOwn(x, campo));

export class ErrorDescargaFirmaV2 extends Error {
  constructor(codigo) { super(codigo); this.name = "ErrorDescargaFirmaV2"; this.codigo = codigo; }
}

async function leerJSON(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")
    || (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA))) {
    throw new ErrorDescargaFirmaV2("resultado_no_confiable");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new ErrorDescargaFirmaV2("resultado_no_confiable");
  const partes = []; let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_RESPUESTA) throw new ErrorDescargaFirmaV2("resultado_no_confiable");
      partes.push(value);
    }
  } finally { void lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total); let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorDescargaFirmaV2("resultado_no_confiable"); }
}

async function validar(datos, solicitud, cryptoImpl) {
  const campos = ["esquema", "pdf_base64", "expediente_ref", "version_observada", "documento", "firma_ref",
    "recibo_ref", "documento_ref", "documento_version", "huella_sha256"];
  if (!exacto(datos, campos) || datos.esquema !== ESQUEMA
    || datos.expediente_ref !== solicitud.expedienteRef || datos.version_observada !== solicitud.version
    || datos.documento !== solicitud.documento || datos.firma_ref !== solicitud.firmaRef
    || !referencia(datos.recibo_ref) || datos.documento_ref !== solicitud.custodiaRef
    || datos.documento_version !== solicitud.custodiaVersion || datos.huella_sha256 !== solicitud.huellaEsperada
    || typeof datos.pdf_base64 !== "string" || datos.pdf_base64.length < 16
    || datos.pdf_base64.length > Math.ceil(MAXIMO_PDF / 3) * 4 || !BASE64.test(datos.pdf_base64)) {
    throw new ErrorDescargaFirmaV2("resultado_no_confiable");
  }
  let contenido;
  try {
    const binario = atob(datos.pdf_base64);
    if (btoa(binario) !== datos.pdf_base64) throw new Error();
    contenido = Uint8Array.from(binario, (letra) => letra.charCodeAt(0));
  } catch { throw new ErrorDescargaFirmaV2("resultado_no_confiable"); }
  if (contenido.length < 10 || contenido.length > MAXIMO_PDF
    || new TextDecoder().decode(contenido.subarray(0, 5)) !== "%PDF-"
    || !new TextDecoder().decode(contenido.subarray(-1024)).includes("%%EOF")
    || typeof cryptoImpl?.subtle?.digest !== "function") throw new ErrorDescargaFirmaV2("resultado_no_confiable");
  let hash;
  try {
    const digest = await cryptoImpl.subtle.digest("SHA-256", contenido);
    hash = [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  } catch { throw new ErrorDescargaFirmaV2("resultado_no_confiable"); }
  if (hash !== datos.huella_sha256) throw new ErrorDescargaFirmaV2("resultado_no_confiable");
  return Object.freeze({ contenido, tipo: "application/pdf", nombre: `${solicitud.documento.replaceAll("_", "-")}.pdf`,
    reciboRef: datos.recibo_ref, documentoRef: datos.documento_ref,
    documentoVersion: datos.documento_version, huellaSha256: hash });
}

export function crearClienteDescargaFirmaV2({ fetchImpl = globalThis.fetch, cryptoImpl = globalThis.crypto } = {}) {
  return Object.freeze({
    async descargar(solicitud, { signal } = {}) {
      const { expedienteRef, version, documento, firmaRef, custodiaRef, custodiaVersion, huellaEsperada } = solicitud ?? {};
      if (!referencia(expedienteRef) || !entero(version) || typeof documento !== "string" || !DOCUMENTO.test(documento)
        || !referencia(firmaRef) || !referencia(custodiaRef) || !entero(custodiaVersion)
        || typeof huellaEsperada !== "string" || !HUELLA.test(huellaEsperada)) {
        throw new ErrorDescargaFirmaV2("contenido_no_valido");
      }
      if (signal?.aborted) throw new ErrorDescargaFirmaV2("operacion_abortada");
      if (typeof fetchImpl !== "function") throw new ErrorDescargaFirmaV2("servicio_no_disponible");
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_DESCARGA_FIRMA_V2, {
          method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ expediente_ref: expedienteRef, version_observada: version, documento, firma_ref: firmaRef }),
          signal, mode: "same-origin", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
      } catch { throw new ErrorDescargaFirmaV2(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible"); }
      if (signal?.aborted) { void respuesta.body?.cancel?.().catch(() => {}); throw new ErrorDescargaFirmaV2("operacion_abortada"); }
      if (respuesta.redirected) throw new ErrorDescargaFirmaV2("resultado_no_confiable");
      if ([401, 403, 404].includes(respuesta.status)) {
        void respuesta.body?.cancel?.().catch(() => {}); throw new ErrorDescargaFirmaV2("denegado");
      }
      if (respuesta.status !== 200) {
        void respuesta.body?.cancel?.().catch(() => {});
        throw new ErrorDescargaFirmaV2(respuesta.status === 409 ? "conflicto" : "servicio_no_disponible");
      }
      const envoltorio = await leerJSON(respuesta);
      if (!exacto(envoltorio, ["data"])) throw new ErrorDescargaFirmaV2("resultado_no_confiable");
      const archivo = await validar(envoltorio.data, solicitud, cryptoImpl);
      if (signal?.aborted) throw new ErrorDescargaFirmaV2("operacion_abortada");
      return archivo;
    },
  });
}
