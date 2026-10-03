/** Registro V2: el original se selecciona por su vínculo custodiado, sin enviar sus bytes. */
export const RUTA_REGISTRO_FIRMA_VEC = "/api/vec/contratacion-temporal/firmas-documento/registro-vec";
const ESQUEMA = "vec.contratacion-temporal.registro-firma-vec.v2";
const MAXIMO_PDF = 1 << 20;
const MAXIMO_RESPUESTA = 64 * 1024;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const DOCUMENTO = /^[a-z][a-z0-9_]{1,63}$/u;
const CLAVE = /^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const CODIGOS = new Set(["autenticacion_requerida", "acceso_denegado", "contenido_no_valido", "peticion_no_valida",
  "paso_no_pendiente", "cadena_rota", "conflicto", "firma_no_verificada", "verificacion_no_disponible",
  "servicio_no_disponible", "resultado_no_confiable", "recurso_no_encontrado"]);

function referenciaValida(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function documentoValido(valor) { return typeof valor === "string" && DOCUMENTO.test(valor); }
function huellaValida(valor) { return typeof valor === "string" && HUELLA.test(valor); }
function referenciaCustodiaValida(valor) { return typeof valor === "string" && /^ref:[0-9a-f]{64}$/u.test(valor); }

export class ErrorFirmaVec extends Error {
  constructor(codigo) { super(codigo); this.name = "ErrorFirmaVec"; this.codigo = codigo; }
}

function objeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor) && Object.getPrototypeOf(valor) === Object.prototype;
}

function campos(valor, requeridos, opcionales = []) {
  return objeto(valor) && requeridos.every((clave) => Object.hasOwn(valor, clave))
    && Object.keys(valor).every((clave) => requeridos.includes(clave) || opcionales.includes(clave));
}

function entero(valor) { return Number.isSafeInteger(valor) && valor > 0; }

function fechaRecibo(valor) {
  return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(valor)
    && !Number.isNaN(Date.parse(valor));
}

function pdfValido(bytes) {
  if (!(bytes instanceof Uint8Array) || bytes.length < 10 || bytes.length > MAXIMO_PDF) return false;
  const cabecera = new TextDecoder().decode(bytes.subarray(0, 5));
  const cola = new TextDecoder().decode(bytes.subarray(-1024));
  return cabecera === "%PDF-" && cola.includes("%%EOF");
}

function aBase64(bytes) {
  let binario = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binario += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binario);
}

/** Coteja la confirmación con los mismos bytes que se enviaron. */
export async function huellaPDFFirmado(bytes) {
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

async function leerJSON(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA)) {
    throw new ErrorFirmaVec("resultado_no_confiable");
  }
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
    throw new ErrorFirmaVec("resultado_no_confiable");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new ErrorFirmaVec("resultado_no_confiable");
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { value, done } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_RESPUESTA) throw new ErrorFirmaVec("resultado_no_confiable");
      partes.push(value);
    }
  } finally { void lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total);
  let inicio = 0;
  for (const parte of partes) { bytes.set(parte, inicio); inicio += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorFirmaVec("resultado_no_confiable"); }
}

/** Ambas vías conservan las mismas pruebas de revisión; la procedencia externa es declarada. */
export function validarReciboFirmaV2(datos, solicitud, externo = false) {
  const claves = ["esquema", "recibo_ref", "firma_ref", "ya_registrada", "expediente_ref", "version_expediente",
    "documento", "paso_orden", "paso_ref", "secuencia", "registrada_en", "documento_custodiado",
    "verificacion_tecnica", "firma_eficaz", "material_root_sha256", "revision_pdf",
    ...(externo ? ["procedencia_portafirmas"] : [])];
  const custodia = datos?.documento_custodiado;
  const verificacion = datos?.verificacion_tecnica;
  const revision = datos?.revision_pdf;
  if (!campos(datos, claves) || datos.esquema !== (externo ? "vec.contratacion-temporal.registro-firma-externa.v2" : ESQUEMA) || datos.firma_eficaz !== false
    || datos.expediente_ref !== solicitud.expedienteRef || datos.version_expediente !== solicitud.version
    || datos.documento !== solicitud.documento || datos.paso_orden !== solicitud.pasoOrden
    || !referenciaValida(datos.recibo_ref) || !referenciaValida(datos.firma_ref) || !referenciaValida(datos.paso_ref)
    || !entero(datos.secuencia) || typeof datos.ya_registrada !== "boolean" || !fechaRecibo(datos.registrada_en)
    || !campos(custodia, ["expediente_ref", "documento_ref", "version", "huella_sha256"])
    || !referenciaCustodiaValida(custodia.expediente_ref) || !referenciaCustodiaValida(custodia.documento_ref)
    || !entero(custodia.version) || !huellaValida(custodia.huella_sha256)
    || !campos(verificacion, ["estado", "motivo", "politica", "revocacion", "sello_tiempo", "original_sha256", "firmado_sha256"])
    || verificacion.estado !== "valida" || verificacion.motivo !== "verificada"
    || typeof verificacion.politica !== "string" || verificacion.politica.length < 1 || verificacion.politica.length > 256
    || verificacion.revocacion !== "vigente" || !["no_presente", "valido", "no_comprobado"].includes(verificacion.sello_tiempo)
    || !huellaValida(verificacion.original_sha256) || !huellaValida(verificacion.firmado_sha256)
    || (solicitud.originalHuella !== undefined && verificacion.original_sha256 !== solicitud.originalHuella)
    || verificacion.firmado_sha256 !== custodia.huella_sha256
    || !huellaValida(datos.material_root_sha256)
    || !campos(revision, ["orden_firma", "entrada_sha256", "revision_sha256", "evidencia_sha256"])
    || revision.orden_firma !== solicitud.pasoOrden
    || (solicitud.revisionEntradaHuella !== undefined && revision.entrada_sha256 !== solicitud.revisionEntradaHuella)
    || !huellaValida(revision.entrada_sha256) || !huellaValida(revision.revision_sha256)
    || revision.revision_sha256 !== verificacion.firmado_sha256
    || (solicitud.pasoOrden === 1 && revision.entrada_sha256 !== verificacion.original_sha256)
    || !huellaValida(revision.evidencia_sha256)) return null;
  if (externo && (!campos(datos.procedencia_portafirmas, ["estado", "referencia_declarada", "fecha_declarada"])
    || datos.procedencia_portafirmas.estado !== "declarada_por_rrhh"
    || datos.procedencia_portafirmas.referencia_declarada !== solicitud.referenciaPortafirmas
    || datos.procedencia_portafirmas.fecha_declarada !== solicitud.fechaPortafirmas)) return null;
  return Object.freeze({ ...datos });
}

export function crearClienteFirmaVec({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async registrar(solicitud, { signal } = {}) {
      const { expedienteRef, version, documento, pasoOrden, originalRef, originalVersion, firmado, clave } = solicitud ?? {};
      if (!referenciaValida(expedienteRef) || !entero(version) || !documentoValido(documento)
        || !entero(pasoOrden) || pasoOrden > 16 || !referenciaValida(originalRef) || !entero(originalVersion)
        || !pdfValido(firmado) || typeof clave !== "string" || !CLAVE.test(clave)) throw new ErrorFirmaVec("contenido_no_valido");
      if (signal?.aborted) throw new ErrorFirmaVec("operacion_abortada");
      if (typeof fetchImpl !== "function") throw new ErrorFirmaVec("servicio_no_disponible");
      const bytesEnviados = firmado.slice();
      let huellaEnviada;
      try { huellaEnviada = await huellaPDFFirmado(bytesEnviados); }
      catch { throw new ErrorFirmaVec("servicio_no_disponible"); }
      if (signal?.aborted) throw new ErrorFirmaVec("operacion_abortada");
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_REGISTRO_FIRMA_VEC, {
          method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ expediente_ref: expedienteRef, version_expediente: version, documento, paso_orden: pasoOrden,
            original_ref: originalRef, original_version: originalVersion, firmado_base64: aBase64(bytesEnviados),
            clave_idempotencia: clave }),
          signal, mode: "same-origin", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
      } catch { throw new ErrorFirmaVec(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible"); }
      if (signal?.aborted) { void respuesta.body?.cancel?.().catch(() => {}); throw new ErrorFirmaVec("operacion_abortada"); }
      if (respuesta.redirected) throw new ErrorFirmaVec("resultado_no_confiable");
      if (respuesta.status === 401 || respuesta.status === 403) {
        void respuesta.body?.cancel?.().catch(() => {});
        throw new ErrorFirmaVec("acceso_denegado");
      }
      const envoltorio = await leerJSON(respuesta);
      if (respuesta.status >= 400) {
        const error = envoltorio?.error;
        throw new ErrorFirmaVec(campos(error, ["codigo", "clave_i18n", "correlacion_ref"])
          && CODIGOS.has(error.codigo) ? error.codigo : "servicio_no_disponible");
      }
      if (!campos(envoltorio, ["data"]) || ![200, 201].includes(respuesta.status)) throw new ErrorFirmaVec("resultado_no_confiable");
      const recibo = validarReciboFirmaV2(envoltorio.data, solicitud);
      if (!recibo || recibo.ya_registrada !== (respuesta.status === 200)
        || recibo.documento_custodiado.huella_sha256 !== huellaEnviada) throw new ErrorFirmaVec("resultado_no_confiable");
      return recibo;
    },
  });
}
