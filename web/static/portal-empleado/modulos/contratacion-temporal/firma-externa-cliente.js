/** Registro manual de un PDF firmado fuera de VEC. La referencia y la fecha
 * de Portafirmas son declaraciones de RRHH; el servidor verifica el PDF. */

export const RUTA_REGISTRO_FIRMA_EXTERNA = "/api/vec/contratacion-temporal/firmas-documento/registro-externo";
const ESQUEMA = "vec.contratacion-temporal.registro-firma-externa.v1";
const MAXIMO_PDF = 1 << 20;
const MAXIMO_RESPUESTA = 64 * 1024;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const DOCUMENTO = /^[a-z][a-z0-9_]{1,63}$/u;
const CLAVE = /^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const UTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;
const CODIGOS = new Set(["autenticacion_requerida", "acceso_denegado", "contenido_no_valido", "peticion_no_valida",
  "paso_no_pendiente", "cadena_rota", "conflicto", "firma_no_verificada", "verificacion_no_disponible",
  "servicio_no_disponible", "resultado_no_confiable", "recurso_no_encontrado"]);

function referenciaValida(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function documentoValido(valor) { return typeof valor === "string" && DOCUMENTO.test(valor); }
function huellaValida(valor) { return typeof valor === "string" && HUELLA.test(valor); }
function referenciaCustodiaValida(valor) { return typeof valor === "string" && /^ref:[0-9a-f]{64}$/u.test(valor); }

export class ErrorFirmaExterna extends Error {
  constructor(codigo) { super(codigo); this.name = "ErrorFirmaExterna"; this.codigo = codigo; }
}

function objeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor) && Object.getPrototypeOf(valor) === Object.prototype;
}

function campos(valor, requeridos, opcionales = []) {
  return objeto(valor) && requeridos.every((clave) => Object.hasOwn(valor, clave))
    && Object.keys(valor).every((clave) => requeridos.includes(clave) || opcionales.includes(clave));
}

function entero(valor) { return Number.isSafeInteger(valor) && valor > 0; }

function fechaCanonica(valor) {
  return typeof valor === "string" && UTC.test(valor) && !Number.isNaN(Date.parse(valor))
    && new Date(valor).toISOString().replace(/\.000Z$/u, "Z") === valor;
}

function fechaRecibo(valor) {
  return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(valor)
    && !Number.isNaN(Date.parse(valor));
}

function referenciaDeclarada(valor) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= 256 && valor.trim() === valor
    && !/[\u0000-\u001f\u007f]/u.test(valor);
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

async function leerJSON(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA)) {
    throw new ErrorFirmaExterna("resultado_no_confiable");
  }
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
    throw new ErrorFirmaExterna("resultado_no_confiable");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new ErrorFirmaExterna("resultado_no_confiable");
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { value, done } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_RESPUESTA) throw new ErrorFirmaExterna("resultado_no_confiable");
      partes.push(value);
    }
  } finally { void lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total);
  let inicio = 0;
  for (const parte of partes) { bytes.set(parte, inicio); inicio += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorFirmaExterna("resultado_no_confiable"); }
}

function validarRecibo(datos, solicitud) {
  const claves = ["esquema", "recibo_ref", "firma_ref", "ya_registrada", "expediente_ref", "version_expediente",
    "documento", "paso_orden", "paso_ref", "secuencia", "registrada_en", "documento_custodiado",
    "verificacion_tecnica", "procedencia_portafirmas", "firma_eficaz"];
  const custodia = datos?.documento_custodiado;
  const verificacion = datos?.verificacion_tecnica;
  const procedencia = datos?.procedencia_portafirmas;
  if (!campos(datos, claves) || datos.esquema !== ESQUEMA || datos.firma_eficaz !== false
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
    || verificacion.firmado_sha256 !== custodia.huella_sha256
    || !campos(procedencia, ["estado", "referencia_declarada", "fecha_declarada"])
    || procedencia.estado !== "declarada_por_rrhh"
    || procedencia.referencia_declarada !== solicitud.referenciaPortafirmas
    || procedencia.fecha_declarada !== solicitud.fechaPortafirmas) return null;
  return Object.freeze({ ...datos });
}

export function crearClienteFirmaExterna({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async registrar(solicitud, { signal } = {}) {
      const { expedienteRef, version, documento, pasoOrden, originalRef, originalVersion, firmado,
        referenciaPortafirmas, fechaPortafirmas, clave } = solicitud ?? {};
      if (!referenciaValida(expedienteRef) || !entero(version) || !documentoValido(documento)
        || !entero(pasoOrden) || pasoOrden > 16 || !referenciaValida(originalRef) || !entero(originalVersion)
        || !pdfValido(firmado) || !referenciaDeclarada(referenciaPortafirmas)
        || !fechaCanonica(fechaPortafirmas) || typeof clave !== "string" || !CLAVE.test(clave)) throw new ErrorFirmaExterna("contenido_no_valido");
      if (typeof fetchImpl !== "function") throw new ErrorFirmaExterna("servicio_no_disponible");
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_REGISTRO_FIRMA_EXTERNA, {
          method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ expediente_ref: expedienteRef, version_expediente: version, documento, paso_orden: pasoOrden,
            original_ref: originalRef, original_version: originalVersion, firmado_base64: aBase64(firmado),
            referencia_portafirmas_declarada: referenciaPortafirmas, fecha_portafirmas_declarada: fechaPortafirmas,
            clave_idempotencia: clave }),
          signal, mode: "same-origin", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
      } catch { throw new ErrorFirmaExterna(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible"); }
      if (respuesta.redirected) throw new ErrorFirmaExterna("resultado_no_confiable");
      if (respuesta.status === 401 || respuesta.status === 403) {
        void respuesta.body?.cancel?.().catch(() => {});
        throw new ErrorFirmaExterna("acceso_denegado");
      }
      const envoltorio = await leerJSON(respuesta);
      if (respuesta.status >= 400) {
        const error = envoltorio?.error;
        throw new ErrorFirmaExterna(campos(error, ["codigo", "clave_i18n", "correlacion_ref"])
          && CODIGOS.has(error.codigo) ? error.codigo : "servicio_no_disponible");
      }
      if (!campos(envoltorio, ["data"]) || ![200, 201].includes(respuesta.status)) throw new ErrorFirmaExterna("resultado_no_confiable");
      const recibo = validarRecibo(envoltorio.data, solicitud);
      if (!recibo || recibo.ya_registrada !== (respuesta.status === 200)) throw new ErrorFirmaExterna("resultado_no_confiable");
      return recibo;
    },
  });
}
