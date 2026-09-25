import { ErrorClienteSaldoCronos, leerJSONAcotado, validarConsultaSaldoCronos } from "./cliente-saldo-http.js";

/**
 * Cliente de movimientos, correcciones y permisos propios. El servidor
 * deriva la persona y el empleado de la sesión mTLS; el cliente sólo envía
 * periodo, año, fechas, horas, permiso y la clave de su operación.
 */
export const RUTAS_SOLICITUDES_CRONOS = Object.freeze({
  movimientos: "/api/interna/cronos/movimientos/propio",
  correcciones: "/api/interna/cronos/correcciones/propias",
  permisos: "/api/interna/cronos/permisos/propio",
  solicitudes: "/api/interna/cronos/permisos/solicitudes",
});

const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const ESTADOS_CORRECCION = new Set(["pendiente_responsable", "pendiente_rrhh", "denegada_responsable", "denegada_rrhh", "pendiente_aplicacion", "aplicada"]);
const ESTADOS_PERMISO = new Set(["solicitado", "pendiente_administracion", "concedido", "denegado", "cancelado"]);
const ERRORES_422 = new Set(["permiso_no_solicitable", "calendario_no_publicado", "fuera_de_limites", "solapado"]);

export class ErrorClienteSolicitudesCronos extends Error {
  constructor(codigo, estado = 0) {
    super(`cliente de solicitudes Cronos: ${codigo}`);
    this.name = "ErrorClienteSolicitudesCronos";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function fallo(codigo, estado = 0) { return new ErrorClienteSolicitudesCronos(codigo, estado); }
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
function texto(v, max = 160) { return typeof v === "string" && v.length > 0 && v.length <= max; }
function entero(v, min = 0) { return Number.isSafeInteger(v) && v >= min; }
function enteroONulo(v) { return v === null || entero(v, 1); }
function instante(v) { return typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v) && Number.isFinite(Date.parse(v)); }
function referencia(v, prefijo) { return typeof v === "string" && v.startsWith(prefijo) && /^[A-Za-z0-9:._-]{1,200}$/u.test(v); }
function unidad(v) { return v === "dia" || v === "hora"; }
function clave(v) { return typeof v === "string" && /^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u.test(v); }
function lista(v, max) { return Array.isArray(v) && v.length <= max; }
function incompatible() { return fallo("respuesta_incompatible", 200); }
/**
 * Circuito aplicado de una solicitud propia (lo añade Cronos 000010; antes
 * falta). Sólo «A» o «J-A»; «pendiente de asignar jefatura» sólo en lo
 * solicitado por jefatura; lo pendiente de RRHH tras la jefatura es «J-A».
 */
function circuitoSolicitudValido(s) {
  if (s.circuito !== undefined && s.circuito !== "A" && s.circuito !== "J-A") return false;
  if (s.pendiente_asignacion !== undefined && (s.pendiente_asignacion !== true || s.estado !== "solicitado" || s.circuito !== "J-A")) return false;
  return s.estado !== "pendiente_administracion" || s.circuito === undefined || s.circuito === "J-A";
}

export function validarMovimientosPropiosCronos(v, consulta) {
  if (!campos(v, ["periodo", "calendario", "marcajes_por_dia", "absentismos", "correcciones"])
    || !campos(v.periodo, ["tipo", "desde", "hasta"]) || v.periodo.tipo !== consulta.periodo
    || !fecha(v.periodo.desde) || !fecha(v.periodo.hasta) || v.periodo.desde > v.periodo.hasta
    || (consulta.periodo === "rango" && (v.periodo.desde !== consulta.desde || v.periodo.hasta !== consulta.hasta))
    || !campos(v.calendario, ["disponible", "dias"]) || typeof v.calendario.disponible !== "boolean" || !lista(v.calendario.dias, 367)
    || (!v.calendario.disponible && v.calendario.dias.length) || !lista(v.marcajes_por_dia, 367)
    || !lista(v.absentismos, 1000) || !lista(v.correcciones, 2000)) throw incompatible();
  const dentro = (f) => fecha(f) && f >= v.periodo.desde && f <= v.periodo.hasta;
  for (const d of v.calendario.dias) if (!campos(d, ["fecha", "tipo", "nombre"]) || !dentro(d.fecha) || !["festivo", "no_laborable"].includes(d.tipo) || !texto(d.nombre, 120)) throw incompatible();
  for (const d of v.marcajes_por_dia) if (!campos(d, ["fecha", "marcajes"]) || !dentro(d.fecha) || !entero(d.marcajes, 1)) throw incompatible();
  for (const a of v.absentismos) {
    if (!campos(a, ["solicitud_ref", "permiso_ref", "nombre", "desde", "hasta", "cantidad", "unidad", "pendiente_justificar"])
      || !referencia(a.solicitud_ref, "permiso:cronos:solicitud:") || !referencia(a.permiso_ref, "permiso:cronos:") || !texto(a.nombre, 120)
      || !fecha(a.desde) || !fecha(a.hasta) || a.desde > a.hasta || a.hasta < v.periodo.desde || a.desde > v.periodo.hasta
      || !entero(a.cantidad, 1) || !unidad(a.unidad) || typeof a.pendiente_justificar !== "boolean") throw incompatible();
  }
  for (const c of v.correcciones) {
    if (!campos(c, ["solicitud_ref", "fecha_civil", "hora_pretendida", "movimiento", "estado", "version", "solicitada_en"])
      || !referencia(c.solicitud_ref, "correccion:cronos:") || !dentro(c.fecha_civil) || !hora(c.hora_pretendida)
      || !MOVIMIENTOS.has(c.movimiento) || !ESTADOS_CORRECCION.has(c.estado) || !entero(c.version, 1) || !instante(c.solicitada_en)) throw incompatible();
  }
  return v;
}

export function validarPermisosPropiosCronos(v, anio) {
  if (!campos(v, ["anio", "permisos", "solicitudes"]) || !entero(v.anio, 2000) || (anio && v.anio !== anio)
    || !lista(v.permisos, 500) || !lista(v.solicitudes, 5000)) throw incompatible();
  for (const p of v.permisos) {
    if (!campos(p, ["permiso_ref", "version_ref", "nombre", "unidad", "computo", "circuito", "minimo", "maximo_solicitud", "maximo_mensual", "maximo_anual",
      "justificante_exigido", "solicitable", "sintetico", "solicitado", "concedido", "pendiente_justificar", "resta"], ["sin_conciliar"])
      || !referencia(p.permiso_ref, "permiso:cronos:") || !referencia(p.version_ref, "catalogo:cronos:") || !texto(p.nombre, 120) || !unidad(p.unidad)
      || !["laborables", "naturales"].includes(p.computo) || !["A", "J-A"].includes(p.circuito) || !entero(p.minimo, 1)
      || !enteroONulo(p.maximo_solicitud) || !enteroONulo(p.maximo_mensual) || !enteroONulo(p.maximo_anual)
      || ![p.justificante_exigido, p.solicitable, p.sintetico].every((b) => typeof b === "boolean")
      || !entero(p.solicitado) || !entero(p.concedido) || !entero(p.pendiente_justificar) || !(p.resta === null || entero(p.resta))
      || (Object.hasOwn(p, "sin_conciliar") && typeof p.sin_conciliar !== "boolean")) throw incompatible();
  }
  for (const s of v.solicitudes) {
    if (!campos(s, ["solicitud_ref", "catalogo_version_ref", "permiso_ref", "desde", "hasta", "cantidad", "unidad", "estado", "version", "pendiente_justificar", "solicitada_en"],
      ["hora_inicio", "hora_fin", "circuito", "pendiente_asignacion"])
      || !circuitoSolicitudValido(s)
      || !referencia(s.solicitud_ref, "permiso:cronos:solicitud:") || !referencia(s.permiso_ref, "permiso:cronos:") || !fecha(s.desde) || !fecha(s.hasta)
      || s.desde > s.hasta || Number(s.desde.slice(0, 4)) !== v.anio || !entero(s.cantidad, 1) || !unidad(s.unidad) || !ESTADOS_PERMISO.has(s.estado)
      || !entero(s.version, 1) || typeof s.pendiente_justificar !== "boolean" || !instante(s.solicitada_en)
      || (s.hora_inicio !== undefined && !hora(s.hora_inicio)) || (s.hora_fin !== undefined && !hora(s.hora_fin))) throw incompatible();
  }
  return v;
}

function validarReciboCorreccion(v, entrada) {
  const r = v?.recibo;
  if (!campos(v, ["recibo"]) || !campos(r, ["solicitud_ref", "actuacion_ref", "recibo_ref", "estado", "version", "instante_utc", "replay"])
    || r.solicitud_ref !== `correccion:cronos:${entrada.clave_operacion}` || !referencia(r.recibo_ref, "recibo:cronos:")
    || r.estado !== "pendiente_responsable" || r.version !== 1 || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

function validarReciboPermiso(v, entrada) {
  const r = v?.recibo;
  if (!campos(v, ["recibo"]) || !campos(r, ["solicitud_ref", "recibo_ref", "catalogo_version_ref", "version", "estado", "cantidad", "unidad", "instante_utc", "replay"])
    || r.solicitud_ref !== `permiso:cronos:solicitud:${entrada.clave_operacion}` || !referencia(r.recibo_ref, "recibo:cronos:")
    || r.estado !== "solicitado" || r.version !== 1 || !entero(r.cantidad, 1) || !unidad(r.unidad) || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

export function validarEntradaCorreccionCronos(e) {
  if (!campos(e, ["clave_operacion", "movimiento", "fecha_civil", "hora_pretendida"]) || !clave(e.clave_operacion)
    || !MOVIMIENTOS.has(e.movimiento) || !fecha(e.fecha_civil) || !hora(e.hora_pretendida)) throw new TypeError("solicitud de corrección no válida");
  return Object.freeze({ ...e });
}

export function validarEntradaPermisoCronos(e) {
  if (!campos(e, ["clave_operacion", "permiso_ref", "desde", "hasta"], ["hora_inicio", "hora_fin"]) || !clave(e.clave_operacion)
    || !referencia(e.permiso_ref, "permiso:cronos:") || !fecha(e.desde) || !fecha(e.hasta) || e.desde > e.hasta
    || e.desde.slice(0, 4) !== e.hasta.slice(0, 4) || (e.hora_inicio === undefined) !== (e.hora_fin === undefined)
    || (e.hora_inicio !== undefined && (!hora(e.hora_inicio) || !hora(e.hora_fin) || e.hora_fin <= e.hora_inicio || e.desde !== e.hasta))) throw new TypeError("solicitud de permiso no válida");
  return Object.freeze({ ...e });
}

export function crearClienteSolicitudesCronosHTTP({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de solicitudes Cronos no disponible");

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
        if (respuesta.status === 403) throw fallo(codigoServidor === "sin_empleado" ? "sin_empleado" : "acceso_denegado", 403);
        if (respuesta.status === 400) throw fallo("peticion_invalida", 400);
        if (respuesta.status === 409) throw fallo("conflicto", 409);
        if (respuesta.status === 422 && ERRORES_422.has(codigoServidor)) throw fallo(codigoServidor, 422);
        throw fallo("servicio_no_disponible", respuesta.status);
      }
      if (!json) throw fallo("tipo_respuesta_no_valido", respuesta.status);
      const valor = await leer();
      if (controlador.signal.aborted) throw cortado();
      return { estado: respuesta.status, valor };
    } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
  }

  return Object.freeze({
    async consultarMovimientos(entrada, { signal } = {}) {
      const consulta = validarConsultaSaldoCronos(entrada);
      const p = new URLSearchParams({ periodo: consulta.periodo });
      if (consulta.periodo === "rango") { p.set("desde", consulta.desde); p.set("hasta", consulta.hasta); }
      const { valor } = await llamar(`${RUTAS_SOLICITUDES_CRONOS.movimientos}?${p}`, { signal });
      return validarMovimientosPropiosCronos(valor, consulta);
    },
    async consultarPermisos({ anio } = {}, { signal } = {}) {
      if (anio !== undefined && (!Number.isSafeInteger(anio) || anio < 2000 || anio > 2100)) throw new TypeError("año de permisos no válido");
      const url = anio === undefined ? RUTAS_SOLICITUDES_CRONOS.permisos : `${RUTAS_SOLICITUDES_CRONOS.permisos}?${new URLSearchParams({ anio: String(anio) })}`;
      const { valor } = await llamar(url, { signal });
      return validarPermisosPropiosCronos(valor, anio);
    },
    async solicitarCorreccion(entrada, { signal } = {}) {
      const cuerpo = validarEntradaCorreccionCronos(entrada);
      const { valor } = await llamar(RUTAS_SOLICITUDES_CRONOS.correcciones, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] });
      return validarReciboCorreccion(valor, cuerpo);
    },
    async solicitarPermiso(entrada, { signal } = {}) {
      const cuerpo = validarEntradaPermisoCronos(entrada);
      const { valor } = await llamar(RUTAS_SOLICITUDES_CRONOS.solicitudes, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] });
      return validarReciboPermiso(valor, cuerpo);
    },
  });
}
