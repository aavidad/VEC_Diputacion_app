import { ErrorClienteSaldoCronos, leerJSONAcotado } from "./cliente-saldo-http.js";

/**
 * Cliente de la resolución de permisos (bandeja y resolución de jefatura o
 * RRHH) y de los avisos de resolución de la persona. El servidor deriva la
 * persona, su empleado y su competencia de la sesión mTLS y del circuito
 * publicado; el cliente sólo envía paso, solicitud, decisión, motivo,
 * versión vista y la clave de su operación.
 */
export const RUTAS_RESOLUCION_CRONOS = Object.freeze({
  bandeja: "/api/interna/cronos/permisos/bandeja",
  resoluciones: "/api/interna/cronos/permisos/resoluciones",
  avisos: "/api/interna/cronos/avisos/propio",
  archivos: "/api/interna/cronos/avisos/archivos",
});

export const PASOS_RESOLUCION_CRONOS = Object.freeze(["responsable", "administracion"]);
export const MAXIMO_MOTIVO_RESOLUCION_CRONOS = 500;

const ESTADOS_PENDIENTES = new Set(["solicitado", "pendiente_administracion"]);
const ESTADOS_FINALES = new Set(["concedido", "denegado"]);

export class ErrorClienteResolucionCronos extends Error {
  constructor(codigo, estado = 0) {
    super(`cliente de resolución Cronos: ${codigo}`);
    this.name = "ErrorClienteResolucionCronos";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function fallo(codigo, estado = 0) { return new ErrorClienteResolucionCronos(codigo, estado); }
function objeto(v) { return v !== null && typeof v === "object" && !Array.isArray(v); }
function campos(v, requeridos, opcionales = []) {
  return objeto(v) && requeridos.every((c) => Object.hasOwn(v, c)) && Object.keys(v).every((c) => requeridos.includes(c) || opcionales.includes(c));
}
function fecha(v) {
  if (typeof v !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(v)) return false;
  const [a, m, d] = v.split("-").map(Number); const f = new Date(Date.UTC(a, m - 1, d));
  return f.getUTCFullYear() === a && f.getUTCMonth() === m - 1 && f.getUTCDate() === d;
}
function hora(v) { return typeof v === "string" && /^(?:[01]\d|2[0-3]):[0-5]\d$/u.test(v); }
function texto(v, max = 160) { return typeof v === "string" && v.length > 0 && [...v].length <= max; }
function entero(v, min = 0) { return Number.isSafeInteger(v) && v >= min; }
function instante(v) { return typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v) && Number.isFinite(Date.parse(v)); }
function referencia(v, prefijo) { return typeof v === "string" && v.startsWith(prefijo) && /^[A-Za-z0-9:._-]{1,200}$/u.test(v); }
function unidad(v) { return v === "dia" || v === "hora"; }
function clave(v) { return typeof v === "string" && /^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u.test(v); }
function incompatible() { return fallo("respuesta_incompatible", 200); }
function tramo(v) {
  return (v.hora_inicio === undefined) === (v.hora_fin === undefined)
    && (v.hora_inicio === undefined || (hora(v.hora_inicio) && hora(v.hora_fin) && v.hora_inicio < v.hora_fin && v.desde === v.hasta));
}

/** Motivo admitido: una línea sin controles ni espacios extremos. */
export function motivoResolucionValido(motivo) {
  return typeof motivo === "string" && motivo === motivo.trim() && [...motivo].length <= MAXIMO_MOTIVO_RESOLUCION_CRONOS
    && !/\p{Cc}/u.test(motivo);
}

export function validarBandejaCronos(v, paso) {
  if (!campos(v, ["paso", "pendientes"]) || v.paso !== paso || !Array.isArray(v.pendientes) || v.pendientes.length > 500) throw incompatible();
  for (const s of v.pendientes) {
    if (!campos(s, ["solicitud_ref", "empleado_ref", "empleado_etiqueta", "permiso_ref", "nombre", "circuito", "justificante_exigido", "desde", "hasta",
      "cantidad", "unidad", "estado", "version", "solicitada_en"], ["hora_inicio", "hora_fin"])
      || !referencia(s.solicitud_ref, "permiso:cronos:solicitud:") || !referencia(s.empleado_ref, "emp_") || typeof s.empleado_etiqueta !== "string"
      || [...s.empleado_etiqueta].length > 120 || !referencia(s.permiso_ref, "permiso:cronos:") || !texto(s.nombre, 120)
      || !["A", "J-A"].includes(s.circuito) || typeof s.justificante_exigido !== "boolean" || !fecha(s.desde) || !fecha(s.hasta) || s.desde > s.hasta
      || !tramo(s) || !entero(s.cantidad, 1) || !unidad(s.unidad) || !ESTADOS_PENDIENTES.has(s.estado) || !entero(s.version, 1) || !instante(s.solicitada_en)) throw incompatible();
  }
  return v;
}

export function validarAvisosCronos(v) {
  if (!campos(v, ["avisos"]) || !Array.isArray(v.avisos) || v.avisos.length > 500) throw incompatible();
  for (const a of v.avisos) {
    if (!campos(a, ["aviso_ref", "solicitud_ref", "estado", "resuelto_en", "permiso_ref", "nombre", "desde", "hasta", "cantidad", "unidad", "archivado"],
      ["motivo", "hora_inicio", "hora_fin", "archivado_en"])
      || !referencia(a.aviso_ref, "aviso:cronos:") || !referencia(a.solicitud_ref, "permiso:cronos:solicitud:") || !ESTADOS_FINALES.has(a.estado)
      || !instante(a.resuelto_en) || !referencia(a.permiso_ref, "permiso:cronos:") || !texto(a.nombre, 120) || !fecha(a.desde) || !fecha(a.hasta)
      || a.desde > a.hasta || !tramo(a) || !entero(a.cantidad, 1) || !unidad(a.unidad) || typeof a.archivado !== "boolean"
      || a.archivado !== (a.archivado_en !== undefined) || (a.archivado_en !== undefined && !instante(a.archivado_en))
      || (a.motivo !== undefined && !texto(a.motivo, MAXIMO_MOTIVO_RESOLUCION_CRONOS)) || (a.estado === "denegado" && a.motivo === undefined)) throw incompatible();
  }
  return v;
}

export function validarEntradaResolucionCronos(e) {
  if (!campos(e, ["clave_operacion", "solicitud_ref", "paso", "decision", "version_esperada"], ["motivo"]) || !clave(e.clave_operacion)
    || !referencia(e.solicitud_ref, "permiso:cronos:solicitud:") || !PASOS_RESOLUCION_CRONOS.includes(e.paso) || !["aprobar", "denegar"].includes(e.decision)
    || !entero(e.version_esperada, 1) || e.version_esperada > 999 || (e.motivo !== undefined && !motivoResolucionValido(e.motivo))
    || (e.decision === "denegar" && !e.motivo)) throw new TypeError("resolución no válida");
  return Object.freeze({ clave_operacion: e.clave_operacion, solicitud_ref: e.solicitud_ref, paso: e.paso, decision: e.decision,
    version_esperada: String(e.version_esperada), ...(e.motivo ? { motivo: e.motivo } : {}) });
}

function estadoTras(paso, decision) {
  if (decision === "denegar") return "denegado";
  return paso === "responsable" ? "pendiente_administracion" : "concedido";
}

function validarReciboResolucion(v, entrada) {
  const r = v?.recibo;
  if (!campos(v, ["recibo"]) || !campos(r, ["resolucion_ref", "solicitud_ref", "recibo_ref", "estado", "version", "instante_utc", "replay"])
    || r.resolucion_ref !== `permiso:cronos:resolucion:${entrada.clave_operacion}` || r.solicitud_ref !== entrada.solicitud_ref
    || !referencia(r.recibo_ref, "recibo:cronos:") || r.estado !== estadoTras(entrada.paso, entrada.decision)
    || r.version !== Number(entrada.version_esperada) + 1 || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

function validarReciboArchivo(v, entrada) {
  const r = v?.recibo;
  if (!campos(v, ["recibo"]) || !campos(r, ["archivo_ref", "aviso_ref", "recibo_ref", "instante_utc", "replay"])
    || r.archivo_ref !== `aviso:cronos:archivo:${entrada.clave_operacion}` || r.aviso_ref !== entrada.aviso_ref
    || !referencia(r.recibo_ref, "recibo:cronos:") || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

export function crearClienteResolucionCronosHTTP({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de resolución Cronos no disponible");

  async function llamar(url, { metodo = "GET", cuerpo, signal, aceptados = [200] }) {
    if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal no válida");
    if (signal?.aborted) throw fallo("operacion_abortada");
    const controlador = new AbortController(); let agotado = false;
    const abortar = () => controlador.abort(); signal?.addEventListener("abort", abortar, { once: true });
    const temporizador = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
    const cortado = () => fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
    try {
      let respuesta;
      try {
        // Mismo origen: el navegador presenta el certificado mTLS de la sesión;
        // sin cookies propias, sin redirecciones y sin referer.
        respuesta = await fetchImpl(url, {
          method: metodo, headers: cuerpo ? { Accept: "application/json", "Content-Type": "application/json" } : { Accept: "application/json" },
          ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), credentials: "same-origin", mode: "same-origin",
          cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
        });
      } catch { throw fallo(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted) throw cortado();
      if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw fallo("respuesta_incompatible");
      const json = /^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "");
      const leer = async () => {
        try { return await leerJSONAcotado(respuesta); }
        catch (error) {
          if (controlador.signal.aborted) throw cortado();
          throw error instanceof ErrorClienteSaldoCronos ? fallo(error.codigo, error.estado) : fallo("respuesta_incompatible", respuesta.status);
        }
      };
      if (!aceptados.includes(respuesta.status)) {
        let codigoServidor = "";
        if (json) { try { codigoServidor = (await leer())?.error ?? ""; } catch { codigoServidor = ""; } }
        if (respuesta.status === 401) throw fallo("autenticacion_requerida", 401);
        if (respuesta.status === 403) throw fallo(["sin_empleado", "no_competente"].includes(codigoServidor) ? codigoServidor : "acceso_denegado", 403);
        if (respuesta.status === 400) throw fallo("peticion_invalida", 400);
        if (respuesta.status === 409) throw fallo(codigoServidor === "estado_cambiado" ? "estado_cambiado" : "conflicto", 409);
        throw fallo("servicio_no_disponible", respuesta.status);
      }
      if (!json) throw fallo("tipo_respuesta_no_valido", respuesta.status);
      const valor = await leer();
      if (controlador.signal.aborted) throw cortado();
      return { estado: respuesta.status, valor };
    } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
  }

  return Object.freeze({
    async consultarBandeja({ paso } = {}, { signal } = {}) {
      if (!PASOS_RESOLUCION_CRONOS.includes(paso)) throw new TypeError("paso de resolución no válido");
      const { valor } = await llamar(`${RUTAS_RESOLUCION_CRONOS.bandeja}?${new URLSearchParams({ paso })}`, { signal });
      return validarBandejaCronos(valor, paso);
    },
    async resolver(entrada, { signal } = {}) {
      const cuerpo = validarEntradaResolucionCronos(entrada);
      const { valor } = await llamar(RUTAS_RESOLUCION_CRONOS.resoluciones, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] });
      return validarReciboResolucion(valor, cuerpo);
    },
    async consultarAvisos({ signal } = {}) {
      const { valor } = await llamar(RUTAS_RESOLUCION_CRONOS.avisos, { signal });
      return validarAvisosCronos(valor);
    },
    async archivarAviso(entrada, { signal } = {}) {
      if (!campos(entrada, ["clave_operacion", "aviso_ref"]) || !clave(entrada.clave_operacion) || !referencia(entrada.aviso_ref, "aviso:cronos:")) throw new TypeError("archivo de aviso no válido");
      const cuerpo = Object.freeze({ clave_operacion: entrada.clave_operacion, aviso_ref: entrada.aviso_ref });
      const { valor } = await llamar(RUTAS_RESOLUCION_CRONOS.archivos, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] });
      return validarReciboArchivo(valor, cuerpo);
    },
  });
}
