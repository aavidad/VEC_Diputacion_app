import { leerJSONAcotado } from "./cliente-saldo-http.js";
import { IDIOMA_ACTUAL } from "../../../comun/idioma.js";

const MAX_PDF = 2 * 1024 * 1024;
const TIPOS = new Set(["saldo", "movimientos", "permisos"]);
const FORMATOS = new Set(["pdf", "csv"]);
const VISTAS = new Set(["presencia", "incidencias"]);
const ESTADOS = new Set(["entrada_registrada", "pausa_registrada", "salida_registrada", "indeterminado", "no_evaluado", "registro_incompleto", "registrado", "sin_registros"]);
const CAUSAS = new Set(["", "cobertura_incompleta", "sin_marcajes", "secuencia_ambigua"]);
const REF_EQUIPO = /^[A-Za-z0-9][A-Za-z0-9._:#-]{0,127}$/u;
const REF_EMPLEADO = /^emp_[A-Za-z0-9_-]{22,128}$/u;
const REF_RECIBO = /^informe:recibo:[0-9a-f-]{36}$/u;
const CURSOR = /^emp_[A-Za-z0-9_-]{22,128}\|\d{4}-\d{2}-\d{2}$/u;

export class ErrorClienteEquipoInformesCronos extends Error {
  constructor(codigo, estado = 0) { super(`cliente Cronos: ${codigo}`); this.name = "ErrorClienteEquipoInformesCronos"; this.codigo = codigo; this.estado = estado; }
}
const fallo = (codigo, estado = 0) => new ErrorClienteEquipoInformesCronos(codigo, estado);
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const campos = (v, req) => objeto(v) && req.every((k) => Object.hasOwn(v, k)) && Object.keys(v).every((k) => req.includes(k));
function fecha(v) {
  if (typeof v !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(v)) return false;
  const [a, m, d] = v.split("-").map(Number); const f = new Date(Date.UTC(a, m - 1, d));
  return f.getUTCFullYear() === a && f.getUTCMonth() === m - 1 && f.getUTCDate() === d;
}
function entradaConsulta(e) {
  if (!objeto(e) || !REF_EQUIPO.test(e.equipo) || !VISTAS.has(e.vista) || !fecha(e.desde) || !fecha(e.hasta) || e.desde > e.hasta
    || (Date.parse(`${e.hasta}T12:00:00Z`) - Date.parse(`${e.desde}T12:00:00Z`)) > 31 * 86400000
    || (e.cursor && !CURSOR.test(e.cursor))) throw new TypeError("consulta de equipo inválida");
  return { equipo: e.equipo, vista: e.vista, desde: e.desde, hasta: e.hasta, cursor: e.cursor || "" };
}
function validarResultado(v, e) {
  if (!campos(v, ["equipo_ref", "alcance", "desde", "hasta", "vista", "version_ref", "filas", "agregado", "siguiente_cursor"])
    || v.equipo_ref !== e.equipo || v.desde !== e.desde || v.hasta !== e.hasta || v.vista !== e.vista
    || !["jefatura", "rrhh"].includes(v.alcance)
    || typeof v.version_ref !== "string" || !Array.isArray(v.filas) || v.filas.length > 100
    || !campos(v.agregado, ["total", "indeterminado", "registrado", "incidencias"])
    || !["total", "indeterminado", "registrado", "incidencias"].every((k) => Number.isSafeInteger(v.agregado[k]) && v.agregado[k] >= 0)
    || v.agregado.total !== v.filas.length || typeof v.siguiente_cursor !== "string"
    || (v.siguiente_cursor && !CURSOR.test(v.siguiente_cursor))) throw fallo("respuesta_incompatible", 200);
  let anterior = e.cursor;
  for (const f of v.filas) {
    if (!campos(f, ["empleado_ref", "empleado_etiqueta", "fecha", "estado", "causa", "cobertura", "hechos_registrados"])
      || !REF_EMPLEADO.test(f.empleado_ref) || !fecha(f.fecha) || f.fecha < e.desde || f.fecha > e.hasta
      || typeof f.empleado_etiqueta !== "string" || !f.empleado_etiqueta.trim() || [...f.empleado_etiqueta].length > 120
      || `${f.empleado_ref}|${f.fecha}` <= anterior || !ESTADOS.has(f.estado)
      || !CAUSAS.has(f.causa)
      || !["complete", "incomplete", "not_available"].includes(f.cobertura)
      || !Number.isSafeInteger(f.hechos_registrados) || f.hechos_registrados < 0) throw fallo("respuesta_incompatible", 200);
    anterior = `${f.empleado_ref}|${f.fecha}`;
  }
  const conteos = { total: v.filas.length, indeterminado: v.filas.filter((f) => ["indeterminado", "no_evaluado"].includes(f.estado)).length,
    registrado: v.filas.filter((f) => ["entrada_registrada", "registrado"].includes(f.estado)).length,
    incidencias: v.filas.filter((f) => f.estado === "registro_incompleto").length };
  if (Object.keys(conteos).some((k) => conteos[k] !== v.agregado[k])) throw fallo("respuesta_incompatible", 200);
  if (v.siguiente_cursor && v.siguiente_cursor !== anterior) throw fallo("respuesta_incompatible", 200);
  return v;
}
async function leerBytesAcotados(respuesta, maximo) {
  const declarada = respuesta.headers?.get?.("content-length");
  if (declarada && (!/^\d+$/u.test(declarada) || Number(declarada) > maximo)) throw fallo("respuesta_excesiva", respuesta.status);
  if (!respuesta.body?.getReader) throw fallo("respuesta_incompatible", respuesta.status);
  const lector = respuesta.body.getReader(); const trozos = []; let total = 0;
  try {
    while (true) { const { value, done } = await lector.read(); if (done) break;
      if (!(value instanceof Uint8Array) || (total += value.byteLength) > maximo) throw fallo("respuesta_excesiva", respuesta.status);
      trozos.push(value);
    }
  } finally { lector.releaseLock?.(); }
  const bytes = new Uint8Array(total); let i = 0; for (const parte of trozos) { bytes.set(parte, i); i += parte.length; }
  return bytes;
}
function errorEstado(estado) { return estado === 401 ? "autenticacion_requerida" : estado === 403 ? "acceso_denegado" : estado === 400 ? "peticion_invalida" : "servicio_no_disponible"; }

export function crearClienteEquipoInformesCronosHTTP({ fetchImpl = globalThis.fetch, plazoMs = 15000, idiomaActual = () => IDIOMA_ACTUAL } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30000
    || typeof idiomaActual !== "function") throw new TypeError("cliente Cronos no disponible");
  async function llamar(url, { metodo = "GET", cuerpo, signal, descargar = false } = {}) {
    if (signal?.aborted) throw fallo("operacion_abortada");
    const controlador = new AbortController(); let agotado = false; const abortar = () => controlador.abort(); signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
    try {
      let r;
      const idioma = idiomaActual();
      if (typeof idioma !== "string" || !/^[a-z]{2,8}(?:-[A-Za-z0-9]{2,8})?$/u.test(idioma)) throw fallo("idioma_no_disponible");
      try { r = await fetchImpl(url, { method: metodo, ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
        headers: { Accept: descargar ? "application/pdf, text/csv" : "application/json", "Accept-Language": idioma,
          ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
        credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal }); }
      catch { throw fallo(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted || signal?.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "operacion_abortada");
      if (!r || r.redirected || !Number.isInteger(r.status)) throw fallo("respuesta_incompatible");
      if (r.status !== 200) throw fallo(errorEstado(r.status), r.status);
      if (descargar) {
        const tipo = (r.headers?.get?.("content-type") || "").toLowerCase();
        const recibo = r.headers?.get?.("x-vec-recibo-ref") || "";
        if (!REF_RECIBO.test(recibo) || !["application/pdf", "text/csv; charset=utf-8"].includes(tipo)) throw fallo("respuesta_incompatible", 200);
        const bytes = await leerBytesAcotados(r, MAX_PDF);
        if (!bytes.length || (tipo === "application/pdf" && new TextDecoder().decode(bytes.slice(0, 5)) !== "%PDF-")) throw fallo("respuesta_incompatible", 200);
        return { bytes, tipo, recibo };
      }
      if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(r.headers?.get?.("content-type") || "")) throw fallo("respuesta_incompatible", 200);
      try { return await leerJSONAcotado(r); } catch { throw fallo("respuesta_incompatible", 200); }
    } finally { clearTimeout(temporizador); signal?.removeEventListener?.("abort", abortar); }
  }
  return Object.freeze({
    async oferta({ equipo = "" } = {}, { signal } = {}) {
      if (equipo !== "" && !REF_EQUIPO.test(equipo)) throw new TypeError("oferta de informe inválida");
      const url = equipo ? `/api/interna/cronos/equipos/${encodeURIComponent(equipo)}/informes/oferta`
        : "/api/interna/cronos/informes/oferta";
      const r = await llamar(url, { signal });
      const claves = equipo ? ["equipo_ref", "alcance", "exportaciones"] : ["alcance", "exportaciones"];
      if (!campos(r, claves) || (equipo && r.equipo_ref !== equipo) ||
        (equipo ? !["jefatura", "rrhh"].includes(r.alcance) : r.alcance !== "empleado") ||
        !Array.isArray(r.exportaciones) || r.exportaciones.length > 6 ||
        new Set(r.exportaciones).size !== r.exportaciones.length ||
        r.exportaciones.some((v) => typeof v !== "string" ||
          !TIPOS.has(v.split(".")[0]) || !FORMATOS.has(v.split(".")[1]) || v.split(".").length !== 2)) {
        throw fallo("respuesta_incompatible", 200);
      }
      return Object.freeze({ alcance: r.alcance, exportaciones: Object.freeze([...r.exportaciones]) });
    },
    async consultar(e, { signal } = {}) {
      const q = entradaConsulta(e); const url = `/api/interna/cronos/equipos/${encodeURIComponent(q.equipo)}/${q.vista}`;
      const params = new URLSearchParams({ desde: q.desde, hasta: q.hasta, ...(q.cursor ? { cursor: q.cursor } : {}) });
      return validarResultado(await llamar(`${url}?${params}`, { signal }), q);
    },
    async exportar(e, { signal } = {}) {
      if (!objeto(e) || (e.equipo && !REF_EQUIPO.test(e.equipo)) || !TIPOS.has(e.tipo) || !FORMATOS.has(e.formato)
        || !fecha(e.desde) || !fecha(e.hasta) || e.desde > e.hasta || !/^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u.test(e.clave_operacion)) throw new TypeError("informe Cronos inválido");
      const prefijo = e.equipo ? `/api/interna/cronos/equipos/${encodeURIComponent(e.equipo)}/informes/` : "/api/interna/cronos/informes/";
      const resultado = await llamar(`${prefijo}${e.tipo}.${e.formato}`, { metodo: "POST", cuerpo: { desde: e.desde, hasta: e.hasta, clave_operacion: e.clave_operacion }, signal, descargar: true });
      if ((e.formato === "pdf" && resultado.tipo !== "application/pdf") || (e.formato === "csv" && resultado.tipo !== "text/csv; charset=utf-8")) throw fallo("respuesta_incompatible", 200);
      return resultado;
    },
  });
}
