const RUTA = "/api/vec/personal/solicitudes-rectificacion-dietas";
const MAXIMO_RESPUESTA = 512 * 1024;
const MAXIMO_CUERPO = 4 * 1024;
const encoder = new TextEncoder();
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v) &&
  (Object.getPrototypeOf(v) === Object.prototype || Object.getPrototypeOf(v) === null);
const ref = (v, prefijo) => typeof v === "string" && new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u").test(v);
const srd = (v) => typeof v === "string" && /^srd_[0-9a-f]{32}$/u.test(v);
const rrd = (v) => typeof v === "string" && /^rrd_[0-9a-f]{32}$/u.test(v);
const sha = (v) => typeof v === "string" && /^[0-9a-f]{64}$/u.test(v);
const fecha = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(v) &&
  Number.isFinite(Date.parse(`${v}T00:00:00Z`)) && new Date(`${v}T00:00:00Z`).toISOString().slice(0, 10) === v;
const instante = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(v) && Number.isFinite(Date.parse(v));
const texto = (v, min, max) => typeof v === "string" && v.trim() === v &&
  encoder.encode(v).byteLength >= min && encoder.encode(v).byteLength <= max && !/[\x00-\x1f\x7f]/u.test(v);
const campos = new Set(["centro_ref", "unidad_ref", "administrativo_persona_ref", "responsable_persona_ref"]);
const clavesExactas = (v, claves) => objeto(v) && Object.keys(v).length === claves.length && Object.keys(v).every((k) => claves.includes(k));

export class ErrorClienteRectificacionAdminDietas extends Error {
  constructor(codigo, estado = 0, resultadoIndeterminado = false) {
    super(`cliente administrativo de rectificación de Dietas: ${codigo}`);
    this.name = "ErrorClienteRectificacionAdminDietas"; this.codigo = codigo; this.estado = estado;
    this.resultadoIndeterminado = resultadoIndeterminado; Object.freeze(this);
  }
}
const fallo = (codigo, estado = 0, incierto = false) => new ErrorClienteRectificacionAdminDietas(codigo, estado, incierto);
function signalValida(signal) {
  if (signal === undefined) return undefined;
  if (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function") throw fallo("signal_no_valida");
  if (signal.aborted) throw fallo("operacion_abortada"); return signal;
}
function validarAsignacion(a) {
  return clavesExactas(a, ["centro_ref", "administrativo_persona_ref", "responsable_persona_ref", "grupo_dieta", "vigente_desde", "version", "asignacion_ref"]) &&
    texto(a.centro_ref, 1, 160) && ref(a.administrativo_persona_ref, "per_") && ref(a.responsable_persona_ref, "per_") &&
    [1, 2, 3].includes(a.grupo_dieta) && fecha(a.vigente_desde) && Number.isSafeInteger(a.version) && a.version > 0 &&
    ref(a.asignacion_ref, "ads_");
}
function validarSolicitud(s) {
  const claves = ["solicitud_ref", "estado", "persona_ref", "empleado_ref", "relacion_ref", "unidad_ref", "asignacion_ref", "version_origen", "fecha_referencia", "campos_a_revisar", "motivo_revision", "detalle_solicitado", "registrada_en", "asignacion_actual"];
  return clavesExactas(s, claves) && srd(s.solicitud_ref) && s.estado === "pendiente" && ref(s.persona_ref, "per_") &&
    ref(s.empleado_ref, "emp_") && ref(s.relacion_ref, "rel_") && texto(s.unidad_ref, 1, 256) && ref(s.asignacion_ref, "ads_") &&
    Number.isSafeInteger(s.version_origen) && s.version_origen > 0 && fecha(s.fecha_referencia) &&
    Array.isArray(s.campos_a_revisar) && s.campos_a_revisar.length >= 1 && s.campos_a_revisar.length <= 4 &&
    new Set(s.campos_a_revisar).size === s.campos_a_revisar.length && s.campos_a_revisar.every((c) => campos.has(c)) &&
    texto(s.motivo_revision, 3, 500) && texto(s.detalle_solicitado, 0, 500) && instante(s.registrada_en) &&
    validarAsignacion(s.asignacion_actual);
}
function validarLista(v) {
  const claves = ["recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ad3_ref", "consultada_en", "cardinalidad", "solicitudes"];
  if (!clavesExactas(v, claves) || !rrd(v.recibo_ref) || !texto(v.decision_ref, 1, 256) ||
      !texto(v.efecto_ref, 1, 256) || !sha(v.consumo_huella_sha256) || !texto(v.auditoria_ad3_ref, 1, 256) ||
      !instante(v.consultada_en) || !Number.isSafeInteger(v.cardinalidad) || v.cardinalidad < 0 || v.cardinalidad > 50 ||
      !Array.isArray(v.solicitudes) || v.solicitudes.length !== v.cardinalidad || !v.solicitudes.every(validarSolicitud) ||
      new Set(v.solicitudes.map((s) => s.solicitud_ref)).size !== v.solicitudes.length) throw fallo("respuesta_incompatible", 200);
  return Object.freeze({ ...v, solicitudes: Object.freeze(v.solicitudes.map((s) => Object.freeze({ ...s,
    campos_a_revisar: Object.freeze([...s.campos_a_revisar]), asignacion_actual: Object.freeze({ ...s.asignacion_actual }) }))) });
}
function validarDecision(entrada) {
  const confirmar = entrada?.decision === "confirmar";
  const claves = ["decision", "persona_ref", "empleado_ref", "relacion_ref", "unidad_ref", "asignacion_ref", "version_esperada", "fecha_referencia", "clave_idempotencia", "motivo_revision", ...(confirmar ? ["correccion"] : [])];
  if (!clavesExactas(entrada, claves) || !["confirmar", "rechazar"].includes(entrada.decision) ||
      !ref(entrada.persona_ref, "per_") || !ref(entrada.empleado_ref, "emp_") || !ref(entrada.relacion_ref, "rel_") ||
      !texto(entrada.unidad_ref, 1, 256) || !fecha(entrada.fecha_referencia) ||
      !/^[A-Za-z0-9:_-]{16,128}$/u.test(entrada.clave_idempotencia) || !texto(entrada.motivo_revision, 3, 500) ||
      /per_[A-Za-z0-9_-]{22,128}/u.test(entrada.motivo_revision)) throw new TypeError("decisión de rectificación no válida");
  if (confirmar) {
    const c = entrada.correccion;
    if (!ref(entrada.asignacion_ref, "ads_") || !Number.isSafeInteger(entrada.version_esperada) || entrada.version_esperada < 1 ||
        !clavesExactas(c, ["centro_ref", "administrativo_persona_ref", "responsable_persona_ref", "grupo_dieta", "vigente_desde"]) ||
        !texto(c.centro_ref, 1, 160) || !ref(c.administrativo_persona_ref, "per_") || !ref(c.responsable_persona_ref, "per_") ||
        ![1, 2, 3].includes(c.grupo_dieta) || !fecha(c.vigente_desde)) throw new TypeError("corrección de rectificación no válida");
  } else if (entrada.asignacion_ref !== "" || entrada.version_esperada !== 0) throw new TypeError("rechazo de rectificación no válido");
  return Object.freeze({ ...entrada, ...(confirmar ? { correccion: Object.freeze({ ...entrada.correccion }) } : {}) });
}
function validarRecibo(v, solicitudRef, entrada) {
  const claves = ["solicitud_ref", "recibo_ref", "estado", "registrada_en", "asignacion_ref", "version_origen", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ad3_ref"];
  if (!clavesExactas(v, claves) || v.solicitud_ref !== solicitudRef || !rrd(v.recibo_ref) ||
      ![entrada.decision === "confirmar" ? "confirmada" : "rechazada", "replay_confirmado"].includes(v.estado) ||
      !instante(v.registrada_en) || !ref(v.asignacion_ref, "ads_") || !Number.isSafeInteger(v.version_origen) || v.version_origen < 1 ||
      !texto(v.decision_ref, 1, 256) || !texto(v.efecto_ref, 1, 256) || !sha(v.consumo_huella_sha256) ||
      !texto(v.auditoria_ad3_ref, 1, 256) || (entrada.decision === "confirmar" &&
      (v.asignacion_ref !== entrada.asignacion_ref || v.version_origen !== entrada.version_esperada))) throw fallo("respuesta_incompatible", 200, true);
  return Object.freeze({ ...v });
}
async function cancelar(respuesta, lector) { try { await (lector?.cancel?.() ?? respuesta?.body?.cancel?.()); } catch {} }
async function leerJSON(respuesta, signal) {
  const estado = respuesta?.status || 0, longitud = respuesta?.headers?.get?.("content-length");
  if (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA)) { await cancelar(respuesta); throw fallo("respuesta_excesiva", estado); }
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta?.headers?.get?.("content-type") || "") || !respuesta?.body?.getReader) { await cancelar(respuesta); throw fallo("respuesta_incompatible", estado); }
  const lector = respuesta.body.getReader(), partes = []; let total = 0;
  const abortar = () => { Promise.resolve(lector.cancel()).catch(() => {}); }; signal?.addEventListener("abort", abortar, { once: true });
  try { while (true) {
    if (signal?.aborted) throw fallo("operacion_abortada", estado);
    const tramo = await lector.read(); if (tramo.done) break;
    if (!(tramo.value instanceof Uint8Array) || tramo.value.byteLength === 0 || (total += tramo.value.byteLength) > MAXIMO_RESPUESTA) throw fallo("respuesta_excesiva", estado);
    partes.push(tramo.value);
  }
    if (longitud !== null && longitud !== undefined && Number(longitud) !== total) throw fallo("respuesta_incompatible", estado);
    const bytes = new Uint8Array(total); let offset = 0; for (const p of partes) { bytes.set(p, offset); offset += p.byteLength; }
    try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); } catch { throw fallo("json_no_valido", estado); }
  } finally { signal?.removeEventListener("abort", abortar); try { lector.releaseLock?.(); } catch {} }
}
function codigoError(cuerpo, estado) {
  const bruto = typeof cuerpo?.error === "string" && cuerpo.error.startsWith("personal.error.") ? cuerpo.error.slice(15) : "";
  return ({ 400: ["peticion_invalida"], 401: ["autenticacion_requerida"], 403: ["acceso_denegado"], 404: ["no_encontrada"], 409: ["conflicto"], 503: ["no_disponible"] }[estado] || []).includes(bruto) ? bruto : "respuesta_rechazada";
}
async function ejecutar(fetchImpl, ruta, opciones, signal, escritura) {
  let respuesta;
  try { respuesta = await fetchImpl(ruta, { ...opciones, credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); }
  catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura && !signal?.aborted); }
  if (!respuesta || respuesta.redirected) { await cancelar(respuesta); throw fallo("respuesta_rechazada", respuesta?.status || 0, escritura); }
  const estado = respuesta.status || 0; let cuerpo;
  try { cuerpo = await leerJSON(respuesta, signal); }
  catch (e) { if (escritura && !signal?.aborted && (estado >= 500 || estado >= 200 && estado < 300)) throw fallo(e.codigo || "respuesta_incompatible", estado, true); throw e; }
  if (!(estado === 200 && respuesta.ok === true)) throw fallo(codigoError(cuerpo, estado), estado, escritura && estado >= 500);
  return cuerpo;
}

/** Consulta sólo solicitudes que Personal haya enumerado por competencia vigente. */
export function crearClienteRectificacionAdminDietasHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente administrativo de rectificación no disponible");
  return Object.freeze({
    async listar({ signal } = {}) {
      const cuerpo = await ejecutar(fetchImpl, `${RUTA}/competentes`, { method: "GET", headers: { Accept: "application/json" } }, signalValida(signal), false);
      return validarLista(cuerpo);
    },
    async decidir(solicitudRef, entrada, { signal } = {}) {
      if (!srd(solicitudRef)) throw new TypeError("referencia de solicitud no válida");
      const decision = validarDecision(entrada), cuerpo = JSON.stringify(decision);
      if (encoder.encode(cuerpo).byteLength > MAXIMO_CUERPO) throw new TypeError("decisión de rectificación demasiado grande");
      const resultado = await ejecutar(fetchImpl, `${RUTA}/${solicitudRef}`, { method: "PUT", headers: { Accept: "application/json", "Content-Type": "application/json; charset=utf-8" }, body: cuerpo }, signalValida(signal), true);
      return validarRecibo(resultado, solicitudRef, decision);
    },
  });
}
