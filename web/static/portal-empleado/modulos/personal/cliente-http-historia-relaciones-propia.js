export const RUTA_HISTORIA_RELACIONES_PROPIA = "/api/interna/personal/mi-ficha/relaciones/historia";
export const MAXIMO_REVISIONES_HISTORIA_RELACIONES = 200;
const MAXIMO_BYTES = 1024 * 1024;
const REFERENCIA = /^[a-z][a-z0-9_:-]{2,159}$/u;
const RELACION = /^rel_[A-Za-z0-9_-]{22,128}$/u;
const RECIBO = /^aud_v3_[0-9a-f]{32}$/u;
const ESTADOS = new Set(["vigente", "suspendida", "finalizada"]);
const COBERTURAS = new Set(["completa", "parcial", "no_acreditada"]);

export class ErrorHistoriaRelacionesPropia extends Error {
  constructor(codigo, estado = 0) { super(codigo); this.name = "ErrorHistoriaRelacionesPropia"; this.codigo = codigo; this.estado = estado; }
}
const error = (codigo, estado = 0) => new ErrorHistoriaRelacionesPropia(codigo, estado);
const objeto = (dato) => dato !== null && typeof dato === "object" && !Array.isArray(dato);
const claves = (dato, nombres) => objeto(dato) && Object.keys(dato).length === nombres.length && nombres.every((nombre) => Object.hasOwn(dato, nombre));
const texto = (dato, maximo = 300) => typeof dato === "string" && dato.length <= maximo && !/[\u0000-\u001f\u007f]/u.test(dato);

export function fechaHistoriaRelacionesValida(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor) || valor.startsWith("0000")) return false;
  const fecha = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}
function instante(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(valor) || valor.startsWith("0000")) return false;
  const fecha = new Date(valor);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString() === `${valor.slice(0, 23)}Z`;
}
export function intervaloHistoriaRelacionesValido(desde, hasta) {
  return fechaHistoriaRelacionesValida(desde) && fechaHistoriaRelacionesValida(hasta) && desde < hasta;
}
function precede(a, b) {
  if (a.traza.desde !== b.traza.desde) return a.traza.desde > b.traza.desde;
  if (a.traza.registrada_en !== b.traza.registrada_en) return a.traza.registrada_en > b.traza.registrada_en;
  if (a.relacion_ref !== b.relacion_ref) return a.relacion_ref < b.relacion_ref;
  return a.traza.version > b.traza.version;
}

/** La pertenencia y la autorización las acredita el servidor, nunca los prefijos del DTO. */
export function validarRespuestaHistoriaRelaciones(sobre, { efectosDesde, efectosHasta }) {
  const d = sobre?.data, h = d?.historia, c = h?.corte;
  if (!claves(sobre, ["data"]) || !claves(d, ["historia", "consultada_en", "recibo_ref"]) ||
      !claves(h, ["corte", "cobertura", "revisiones"]) || !claves(c, ["efectos_desde", "efectos_hasta", "conocido_en"]) ||
      !intervaloHistoriaRelacionesValido(c.efectos_desde, c.efectos_hasta) || c.efectos_desde !== efectosDesde || c.efectos_hasta !== efectosHasta ||
      !instante(c.conocido_en) || !instante(d.consultada_en) || c.conocido_en > d.consultada_en ||
      typeof d.recibo_ref !== "string" || !RECIBO.test(d.recibo_ref) || !COBERTURAS.has(h.cobertura) || !Array.isArray(h.revisiones)) throw error("respuesta_no_valida");
  if (h.revisiones.length > MAXIMO_REVISIONES_HISTORIA_RELACIONES) throw error("excede_limite");
  const vistos = new Set();
  const revisiones = h.revisiones.map((r, i) => {
    const t = r?.traza;
    if (!claves(r, ["relacion_ref", "estado", "regimen", "modalidad", "unidad", "puesto", "situacion", "traza"]) ||
        typeof r.relacion_ref !== "string" || !RELACION.test(r.relacion_ref) ||
        !ESTADOS.has(r.estado) || ![r.regimen, r.modalidad, r.unidad, r.puesto, r.situacion].every((v) => texto(v)) ||
        !claves(t, ["desde", ...(Object.hasOwn(t || {}, "hasta") ? ["hasta"] : []), "registrada_en", "version", "acto_ref", "fuente_ref", "fuente_version"]) ||
        !fechaHistoriaRelacionesValida(t.desde) || (Object.hasOwn(t, "hasta") && (!fechaHistoriaRelacionesValida(t.hasta) || t.hasta <= t.desde)) ||
        !instante(t.registrada_en) || t.registrada_en > c.conocido_en || !Number.isSafeInteger(t.version) || t.version < 1 ||
        !Number.isSafeInteger(t.fuente_version) || t.fuente_version < 1 || typeof t.acto_ref !== "string" || !REFERENCIA.test(t.acto_ref) ||
        typeof t.fuente_ref !== "string" || !REFERENCIA.test(t.fuente_ref) || t.desde >= c.efectos_hasta || (t.hasta && t.hasta <= c.efectos_desde)) throw error("respuesta_no_valida");
    const id = `${r.relacion_ref}|${t.version}`;
    if (vistos.has(id) || (i > 0 && !precede(h.revisiones[i - 1], r))) throw error("respuesta_no_valida");
    vistos.add(id);
    return Object.freeze({ ...r, traza: Object.freeze({ ...t }) });
  });
  return Object.freeze({ historia: Object.freeze({ corte: Object.freeze({ ...c }), cobertura: h.cobertura, revisiones: Object.freeze(revisiones) }), consultada_en: d.consultada_en, recibo_ref: d.recibo_ref });
}
async function leerJSON(respuesta, maximo) {
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw error("respuesta_no_valida", respuesta.status);
  const partes = []; let total = 0; let terminada = false;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) { terminada = true; break; }
      if (!(value instanceof Uint8Array) || (total += value.byteLength) > maximo) throw error("respuesta_no_valida", respuesta.status);
      partes.push(value);
    }
  } finally { if (!terminada) await lector.cancel().catch(() => {}); lector.releaseLock(); }
  const bytes = new Uint8Array(total); let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); } catch { throw error("respuesta_no_valida", respuesta.status); }
}

/** POST de lectura propio: no selecciona actor/empleado ni renueva otra consulta. */
export function crearClienteHistoriaRelacionesPropia({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw error("no_disponible");
  return Object.freeze({ async consultar({ efectosDesde, efectosHasta, signal } = {}) {
    if (!intervaloHistoriaRelacionesValido(efectosDesde, efectosHasta)) throw error("intervalo_invalido");
    if (signal?.aborted) throw error("operacion_abortada");
    const controlador = new AbortController(), abortar = () => controlador.abort();
    signal?.addEventListener("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, plazoMs); let respuesta;
    try {
      respuesta = await fetchImpl(RUTA_HISTORIA_RELACIONES_PROPIA, {
        method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ efectos_desde: efectosDesde, efectos_hasta: efectosHasta }), signal: controlador.signal,
      });
      const estado = respuesta?.status || 0;
      if (respuesta?.redirected || respuesta?.headers?.get("Content-Type") !== "application/json; charset=utf-8") throw error("respuesta_no_valida", estado);
      const limite = estado === 200 ? MAXIMO_BYTES : 4096;
      const declarada = respuesta.headers.get("Content-Length");
      if (declarada !== null && (!/^[1-9]\d*$/u.test(declarada) || Number(declarada) > limite)) throw error("respuesta_no_valida", estado);
      const sobre = await leerJSON(respuesta, limite);
      if (controlador.signal.aborted) throw error(signal?.aborted ? "operacion_abortada" : "no_disponible");
      if (estado === 200 && respuesta.ok === true) return validarRespuestaHistoriaRelaciones(sobre, { efectosDesde, efectosHasta });
      const codigos = { 400: "peticion_invalida", 401: "autenticacion_requerida", 403: "acceso_denegado", 404: "no_encontrada", 422: "excede_limite", 503: "no_disponible" };
      if (!Object.hasOwn(codigos, estado) || !claves(sobre, ["error"]) || sobre.error !== codigos[estado]) throw error("respuesta_no_valida", estado);
      throw error({ 400: "intervalo_invalido", 401: "sesion_caducada", 403: "denegado", 404: "no_configurado", 422: "excede_limite", 503: "no_disponible" }[estado], estado);
    } catch (causa) {
      if (signal?.aborted) throw error("operacion_abortada");
      if (causa instanceof ErrorHistoriaRelacionesPropia) throw causa;
      throw error("no_disponible");
    } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); try { await respuesta?.body?.cancel?.(); } catch {} }
  } });
}
