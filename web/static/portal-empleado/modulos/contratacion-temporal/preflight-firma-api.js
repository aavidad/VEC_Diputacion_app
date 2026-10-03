/** Consulta nominal previa: no registra firmas ni crea originales. */
export const RUTA_PREFLIGHT_FIRMA = "/api/vec/contratacion-temporal/firma/preflight";
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const DOCUMENTO = /^[a-z][a-z0-9_]{1,63}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const VIAS = ["certificado_vec", "portafirmas_registro_rrhh"];
const MAXIMO_RESPUESTA = 16 * 1024;
const CAMPOS = ["esquema", "version_expediente", "documento", "catalogo_ref", "catalogo_huella", "paso_pendiente",
  "original_ref", "original_version", "vias_disponibles", "entrada_documento_ref", "entrada_documento_version", "entrada_documento_sha256"];

export class ErrorPreflightFirma extends Error {
  constructor(codigo) { super(codigo); this.name = "ErrorPreflightFirma"; this.codigo = codigo; }
}
function entero(v) { return Number.isSafeInteger(v) && v > 0; }
function referencia(v) { return typeof v === "string" && REFERENCIA.test(v); }
function exacto(v, claves) {
  return v !== null && typeof v === "object" && Object.getPrototypeOf(v) === Object.prototype
    && Object.keys(v).length === claves.length && claves.every((c) => Object.hasOwn(v, c));
}
export function validarPreflightFirma(datos, solicitud) {
  if (!exacto(datos, CAMPOS) || datos.esquema !== "vec.contratacion-temporal.preflight-firma.v2"
    || !solicitud || datos.version_expediente !== solicitud.version
    || !entero(datos.version_expediente) || datos.documento !== solicitud.documento
    || !DOCUMENTO.test(datos.documento) || !referencia(datos.catalogo_ref)
    || typeof datos.catalogo_huella !== "string" || !HUELLA.test(datos.catalogo_huella)
    || !Number.isSafeInteger(datos.paso_pendiente) || datos.paso_pendiente < 0 || datos.paso_pendiente > 2
    || datos.original_ref !== solicitud.originalRef || !referencia(datos.original_ref)
    || datos.original_version !== solicitud.originalVersion || !entero(datos.original_version)
    || !Array.isArray(datos.vias_disponibles) || datos.vias_disponibles.length > 2
    || new Set(datos.vias_disponibles).size !== datos.vias_disponibles.length
    || datos.vias_disponibles.some((v) => !VIAS.includes(v))
    || (datos.paso_pendiente === 0 && (datos.vias_disponibles.length !== 0
      || datos.entrada_documento_ref !== "" || datos.entrada_documento_version !== 0 || datos.entrada_documento_sha256 !== ""))
    || (datos.paso_pendiente > 0 && (!referencia(datos.entrada_documento_ref)
      || !entero(datos.entrada_documento_version) || !HUELLA.test(datos.entrada_documento_sha256)))
    || (datos.paso_pendiente === 1 && (datos.entrada_documento_ref !== datos.original_ref
      || datos.entrada_documento_version !== datos.original_version))
    || (datos.paso_pendiente > 1 && datos.entrada_documento_ref === datos.original_ref)) return null;
  return Object.freeze({ ...datos, vias_disponibles: Object.freeze([...datos.vias_disponibles]) });
}
async function leerJSON(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")
    || (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA))) {
    throw new ErrorPreflightFirma("resultado_no_confiable");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new ErrorPreflightFirma("resultado_no_confiable");
  const partes = []; let total = 0;
  try {
    for (;;) {
      const { value, done } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_RESPUESTA) throw new ErrorPreflightFirma("resultado_no_confiable");
      partes.push(value);
    }
  } finally { void lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total); let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorPreflightFirma("resultado_no_confiable"); }
}
export function crearClientePreflightFirma({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async consultar(solicitud, { signal } = {}) {
      const { expedienteRef, version, documento, originalRef, originalVersion } = solicitud ?? {};
      if (!referencia(expedienteRef) || !entero(version) || typeof documento !== "string" || !DOCUMENTO.test(documento)
        || !referencia(originalRef) || !entero(originalVersion)) throw new ErrorPreflightFirma("contenido_no_valido");
      if (signal?.aborted) throw new ErrorPreflightFirma("operacion_abortada");
      if (typeof fetchImpl !== "function") throw new ErrorPreflightFirma("servicio_no_disponible");
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_PREFLIGHT_FIRMA, {
          method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ expediente_ref: expedienteRef, version_observada: version, documento,
            original_ref: originalRef, original_version: originalVersion }), signal,
          mode: "same-origin", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
      } catch { throw new ErrorPreflightFirma(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible"); }
      if (signal?.aborted) { void respuesta.body?.cancel?.().catch(() => {}); throw new ErrorPreflightFirma("operacion_abortada"); }
      if (respuesta.redirected) throw new ErrorPreflightFirma("resultado_no_confiable");
      if ([401, 403, 404].includes(respuesta.status)) {
        void respuesta.body?.cancel?.().catch(() => {}); throw new ErrorPreflightFirma("acceso_denegado");
      }
      const envoltorio = await leerJSON(respuesta);
      if (respuesta.status !== 200) throw new ErrorPreflightFirma(respuesta.status === 409 ? "conflicto" : "servicio_no_disponible");
      const datos = exacto(envoltorio, ["data"]) ? validarPreflightFirma(envoltorio.data, solicitud) : null;
      if (!datos) throw new ErrorPreflightFirma("resultado_no_confiable");
      return datos;
    },
  });
}
