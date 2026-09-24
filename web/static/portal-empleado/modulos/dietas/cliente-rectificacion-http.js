const RUTA = "/api/vec/personal/solicitudes-rectificacion-dietas";
const LIMITE_RESPUESTA = 64 * 1024;
const LIMITE_CUERPO = 4 * 1024;
const encoder = new TextEncoder();
const objeto = (valor) => valor !== null && typeof valor === "object" && !Array.isArray(valor)
  && (Object.getPrototypeOf(valor) === Object.prototype || Object.getPrototypeOf(valor) === null);
const referencia = (valor, prefijo) => typeof valor === "string" && new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u").test(valor);
const fecha = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor)
  && Number.isFinite(Date.parse(`${valor}T00:00:00Z`));
const texto = (valor, minimo, maximo) => typeof valor === "string" && valor.trim() === valor
  && encoder.encode(valor).byteLength >= minimo && encoder.encode(valor).byteLength <= maximo && !/[\x00-\x1f\x7f]/u.test(valor);
const CAMPOS = new Set(["centro_ref", "unidad_ref", "administrativo_persona_ref", "responsable_persona_ref"]);

export class ErrorClienteRectificacionDietas extends Error {
  constructor(codigo, estado = 0, resultadoIndeterminado = false) {
    super(`cliente de rectificación de Dietas: ${codigo}`);
    this.name = "ErrorClienteRectificacionDietas"; this.codigo = codigo; this.estado = estado;
    this.resultadoIndeterminado = resultadoIndeterminado; Object.freeze(this);
  }
}
const fallo = (codigo, estado = 0, incierto = false) => new ErrorClienteRectificacionDietas(codigo, estado, incierto);
function signalValida(signal) {
  if (signal === undefined) return undefined;
  if (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function") throw fallo("signal_no_valida");
  if (signal.aborted) throw fallo("operacion_abortada"); return signal;
}
function validarSolicitud(entrada) {
  const claves = ["relacion_ref", "unidad_ref", "asignacion_ref", "version_esperada", "fecha_referencia", "clave_idempotencia", "campos_a_revisar", "motivo_revision", "detalle_solicitado"];
  if (!objeto(entrada) || Object.keys(entrada).length !== claves.length || Object.keys(entrada).some((clave) => !claves.includes(clave))
    || !referencia(entrada.relacion_ref, "rel_") || !texto(entrada.unidad_ref, 1, 256) || !referencia(entrada.asignacion_ref, "ads_")
    || !Number.isSafeInteger(entrada.version_esperada) || entrada.version_esperada < 1 || !fecha(entrada.fecha_referencia)
    || !/^[A-Za-z0-9:_-]{16,128}$/u.test(entrada.clave_idempotencia) || !Array.isArray(entrada.campos_a_revisar)
    || entrada.campos_a_revisar.length < 1 || entrada.campos_a_revisar.length > 4 || new Set(entrada.campos_a_revisar).size !== entrada.campos_a_revisar.length
    || entrada.campos_a_revisar.some((campo) => !CAMPOS.has(campo)) || !texto(entrada.motivo_revision, 3, 500)
    || !texto(entrada.detalle_solicitado, 0, 500)) throw new TypeError("solicitud de rectificación no válida");
  return Object.freeze({ ...entrada, campos_a_revisar: Object.freeze([...entrada.campos_a_revisar].sort()) });
}
function validarResultado(valor) {
  const claves = ["solicitud_ref", "recibo_ref", "estado", "registrada_en", "asignacion_ref", "version_origen", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ad3_ref"];
  if (!objeto(valor) || Object.keys(valor).length !== claves.length || Object.keys(valor).some((clave) => !claves.includes(clave))
    || !/^srd_[0-9a-f]{32}$/u.test(valor.solicitud_ref) || !/^rrd_[0-9a-f]{32}$/u.test(valor.recibo_ref)
    || !["pendiente", "confirmada", "rechazada", "replay_confirmado"].includes(valor.estado)
    || typeof valor.registrada_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(valor.registrada_en) || !Number.isFinite(Date.parse(valor.registrada_en))
    || !referencia(valor.asignacion_ref, "ads_") || !Number.isSafeInteger(valor.version_origen) || valor.version_origen < 1
    || typeof valor.decision_ref !== "string" || typeof valor.efecto_ref !== "string" || !/^[0-9a-f]{64}$/u.test(valor.consumo_huella_sha256) || typeof valor.auditoria_ad3_ref !== "string") throw new TypeError("resultado de rectificación incompatible");
  return Object.freeze({ ...valor });
}
async function cancelar(respuesta, lector) { try { await (lector?.cancel?.() ?? respuesta?.body?.cancel?.()); } catch {} }
async function leerJSON(respuesta, signal) {
  const longitud = respuesta?.headers?.get?.("content-length"); const estado = respuesta?.status || 0;
  if (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > LIMITE_RESPUESTA)) { await cancelar(respuesta); throw fallo("respuesta_excesiva", estado); }
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta?.headers?.get?.("content-type") || "") || !respuesta?.body?.getReader) { await cancelar(respuesta); throw fallo("respuesta_incompatible", estado); }
  const lector = respuesta.body.getReader(); const partes = []; let total = 0;
  const abortar = () => { Promise.resolve(lector.cancel()).catch(() => {}); }; signal?.addEventListener("abort", abortar, { once: true });
  try {
    while (true) { if (signal?.aborted) throw fallo("operacion_abortada", estado); const tramo = await lector.read(); if (tramo.done) break;
      if (!(tramo.value instanceof Uint8Array) || tramo.value.byteLength === 0 || (total += tramo.value.byteLength) > LIMITE_RESPUESTA) throw fallo("respuesta_excesiva", estado); partes.push(tramo.value); }
    if (longitud !== null && longitud !== undefined && Number(longitud) !== total) throw fallo("respuesta_incompatible", estado);
    const bytes = new Uint8Array(total); let posicion = 0; for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
    try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); } catch { throw fallo("json_no_valido", estado); }
  } finally { signal?.removeEventListener("abort", abortar); try { lector.releaseLock?.(); } catch {} }
}
function codigoError(cuerpo, estado) {
  const codigo = typeof cuerpo?.error === "string" && cuerpo.error.startsWith("personal.error.") ? cuerpo.error.slice(15) : "";
  return ({ 400: ["peticion_invalida"], 401: ["autenticacion_requerida"], 403: ["acceso_denegado"], 404: ["no_encontrada"], 409: ["conflicto"], 503: ["no_disponible"] }[estado] || []).includes(codigo) ? codigo : "respuesta_rechazada";
}
async function ejecutar(fetchImpl, ruta, opciones, estados, signal, escritura) {
  let respuesta;
  try { respuesta = await fetchImpl(ruta, { ...opciones, credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); }
  catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura && !signal?.aborted); }
  if (!respuesta || respuesta.redirected) { await cancelar(respuesta); throw fallo("respuesta_rechazada", respuesta?.status || 0, escritura); }
  const estado = respuesta.status || 0; let cuerpo;
  try { cuerpo = await leerJSON(respuesta, signal); } catch (error) { if (escritura && !signal?.aborted && (estado >= 500 || (estado >= 200 && estado < 300))) throw fallo(error.codigo || "respuesta_incompatible", estado, true); throw error; }
  if (!estados.includes(estado) || respuesta.ok !== true) throw fallo(codigoError(cuerpo, estado), estado, escritura && estado >= 500);
  try { return validarResultado(cuerpo); } catch { throw fallo("respuesta_incompatible", estado, escritura); }
}

/** Puerto HTTP de rectificación: recibe solamente la asignación ya verificada por D7. */
export function crearClienteRectificacionDietasHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de rectificación no disponible");
  return Object.freeze({
    consultar: async ({ relacion_ref, unidad_ref, fecha_referencia }, { signal } = {}) => {
      if (!referencia(relacion_ref, "rel_") || !texto(unidad_ref, 1, 256) || !fecha(fecha_referencia)) throw new TypeError("consulta de rectificación no válida");
      const query = new URLSearchParams({ relacion_ref, unidad_ref, fecha_referencia });
      return ejecutar(fetchImpl, `${RUTA}?${query}`, { method: "GET", headers: { Accept: "application/json" } }, [200], signalValida(signal), false);
    },
    solicitar: async (entrada, { signal } = {}) => {
      const solicitud = validarSolicitud(entrada); const cuerpo = JSON.stringify(solicitud);
      if (encoder.encode(cuerpo).byteLength > LIMITE_CUERPO) throw new TypeError("solicitud de rectificación demasiado grande");
      const resultado = await ejecutar(fetchImpl, RUTA, { method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json; charset=utf-8" }, body: cuerpo }, [200, 201], signalValida(signal), true);
      if (resultado.asignacion_ref !== solicitud.asignacion_ref || resultado.version_origen !== solicitud.version_esperada) throw fallo("respuesta_incompatible", 0, true);
      return resultado;
    },
  });
}
