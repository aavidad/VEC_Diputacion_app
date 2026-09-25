/** Lectura propia de saldo horario. El servidor deriva la persona de la sesión. */
export const RUTA_SALDO_PROPIO_CRONOS = "/api/interna/cronos/saldos/propio";

const PERIODOS = new Set(["hoy", "semana", "mes", "anio", "rango"]);
const MAXIMO_RESPUESTA_BYTES = 2 * 1024 * 1024;
const MAXIMO_DETALLE = 367;
const MAXIMO_MARCAJES_DIA = 10_000;

export class ErrorClienteSaldoCronos extends Error {
  constructor(codigo, estado = 0) {
    super(`cliente de saldo Cronos: ${codigo}`);
    this.name = "ErrorClienteSaldoCronos";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function fallo(codigo, estado = 0) { return new ErrorClienteSaldoCronos(codigo, estado); }
function objeto(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor); }
function campos(valor, requeridos, opcionales = []) {
  return objeto(valor) && requeridos.every((clave) => Object.hasOwn(valor, clave))
    && Object.keys(valor).every((clave) => requeridos.includes(clave) || opcionales.includes(clave));
}
function fechaCivil(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [anio, mes, dia] = valor.split("-").map(Number);
  const fecha = new Date(Date.UTC(anio, mes - 1, dia));
  return fecha.getUTCFullYear() === anio && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia;
}
function minutos(valor, admiteNulo = false) {
  return (admiteNulo && valor === null) || (Number.isSafeInteger(valor) && Math.abs(valor) <= 1_000_000);
}
function codigo(valor) { return typeof valor === "string" && /^[a-z][a-z0-9_]{0,63}$/u.test(valor); }
function instanteUTC(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(valor)) return false;
  const fecha = new Date(valor);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 19) === valor.slice(0, 19);
}

export function validarConsultaSaldoCronos(consulta) {
  if (!objeto(consulta) || !PERIODOS.has(consulta.periodo)) throw new TypeError("periodo de saldo no válido");
  if (consulta.periodo !== "rango") {
    if (Object.keys(consulta).length !== 1) throw new TypeError("fechas de saldo no permitidas");
    return Object.freeze({ periodo: consulta.periodo });
  }
  if (!campos(consulta, ["periodo", "desde", "hasta"]) || !fechaCivil(consulta.desde)
    || !fechaCivil(consulta.hasta) || consulta.desde > consulta.hasta) throw new TypeError("rango de saldo no válido");
  const dias = (Date.parse(`${consulta.hasta}T12:00:00Z`) - Date.parse(`${consulta.desde}T12:00:00Z`)) / 86_400_000 + 1;
  if (dias > MAXIMO_DETALLE) throw new TypeError("rango de saldo demasiado largo");
  return Object.freeze({ periodo: "rango", desde: consulta.desde, hasta: consulta.hasta });
}

export function validarResultadoSaldoCronos(valor, consulta) {
  if (!campos(valor, ["periodo", "resumen", "detalle"])
    || !campos(valor.periodo, ["tipo", "desde", "hasta"])
    || valor.periodo.tipo !== consulta.periodo || !fechaCivil(valor.periodo.desde)
    || !fechaCivil(valor.periodo.hasta) || valor.periodo.desde > valor.periodo.hasta
    || (consulta.periodo === "rango" && (valor.periodo.desde !== consulta.desde || valor.periodo.hasta !== consulta.hasta))
    || !campos(valor.resumen, ["previstos_minutos", "trabajados_minutos", "saldo_minutos", "estado"])
    || !minutos(valor.resumen.previstos_minutos, true)
    || !minutos(valor.resumen.trabajados_minutos)
    || !minutos(valor.resumen.saldo_minutos, true)
    || !codigo(valor.resumen.estado)
    || !Array.isArray(valor.detalle) || valor.detalle.length > MAXIMO_DETALLE) throw fallo("respuesta_incompatible", 200);
  let anterior = "";
  for (const dia of valor.detalle) {
    if (!campos(dia, ["fecha", "previstos_minutos", "trabajados_minutos", "pausas_minutos", "saldo_minutos", "estado", "marcajes"], ["turno_ref", "politica_version_ref"])
      || !fechaCivil(dia.fecha) || dia.fecha < valor.periodo.desde || dia.fecha > valor.periodo.hasta
      || dia.fecha <= anterior || !minutos(dia.previstos_minutos, true)
      || !minutos(dia.trabajados_minutos) || !minutos(dia.pausas_minutos) || !minutos(dia.saldo_minutos, true)
      || (dia.turno_ref !== undefined && (typeof dia.turno_ref !== "string" || dia.turno_ref.length > 128))
      || (dia.politica_version_ref !== undefined && (typeof dia.politica_version_ref !== "string" || dia.politica_version_ref.length > 128))
      || !codigo(dia.estado) || !Array.isArray(dia.marcajes) || dia.marcajes.length > MAXIMO_MARCAJES_DIA) throw fallo("respuesta_incompatible", 200);
    anterior = dia.fecha;
    for (const marcaje of dia.marcajes) {
      if (!campos(marcaje, ["instante_utc", "movimiento", "origen"])
        || !instanteUTC(marcaje.instante_utc) || !codigo(marcaje.movimiento)
        || (marcaje.origen !== null && marcaje.origen !== "terminal" && marcaje.origen !== "remoto")) throw fallo("respuesta_incompatible", 200);
    }
  }
  return valor;
}

/** Lectura acotada del cuerpo JSON; la comparten los clientes propios de Cronos. */
export async function leerJSONAcotado(respuesta) {
  const declarada = respuesta.headers?.get?.("content-length");
  if (declarada !== null && declarada !== undefined && (!/^(?:0|[1-9]\d*)$/u.test(declarada) || Number(declarada) > MAXIMO_RESPUESTA_BYTES)) throw fallo("respuesta_excesiva", respuesta.status);
  if (!respuesta.body?.getReader) throw fallo("respuesta_incompatible", respuesta.status);
  const lector = respuesta.body.getReader(); const partes = []; let total = 0;
  try {
    while (true) {
      const parte = await lector.read();
      if (parte.done) break;
      if (!(parte.value instanceof Uint8Array)) throw fallo("respuesta_incompatible", respuesta.status);
      total += parte.value.byteLength;
      if (total > MAXIMO_RESPUESTA_BYTES) throw fallo("respuesta_excesiva", respuesta.status);
      partes.push(parte.value);
    }
  } catch (error) { await lector.cancel?.().catch?.(() => {}); throw error; }
  finally { lector.releaseLock?.(); }
  const bytes = new Uint8Array(total); let offset = 0;
  for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw fallo("json_no_valido", respuesta.status); }
}

export function crearClienteSaldoCronosHTTP({ fetchImpl = globalThis.fetch, ruta = RUTA_SALDO_PROPIO_CRONOS, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || ruta !== RUTA_SALDO_PROPIO_CRONOS
    || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de saldo Cronos no disponible");
  return Object.freeze({
    async consultar(entrada, { signal } = {}) {
      const consulta = validarConsultaSaldoCronos(entrada);
      if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal de saldo no válida");
      if (signal?.aborted) throw fallo("operacion_abortada");
      const controlador = new AbortController(); let agotado = false;
      const abortar = () => controlador.abort(); signal?.addEventListener("abort", abortar, { once: true });
      const temporizador = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
      try {
        const parametros = new URLSearchParams({ periodo: consulta.periodo });
        if (consulta.periodo === "rango") { parametros.set("desde", consulta.desde); parametros.set("hasta", consulta.hasta); }
        let respuesta;
        try {
          respuesta = await fetchImpl(`${ruta}?${parametros}`, {
            method: "GET", headers: { Accept: "application/json" }, credentials: "same-origin",
            mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
            signal: controlador.signal,
          });
        } catch { throw fallo(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
        if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
        if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw fallo("respuesta_incompatible");
        if (respuesta.status === 401 || respuesta.status === 403) throw fallo("acceso_denegado", respuesta.status);
        if (respuesta.status !== 200 || respuesta.ok !== true) throw fallo("servicio_no_disponible", respuesta.status);
        if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "")) throw fallo("tipo_respuesta_no_valido", 200);
        let cuerpo;
        try { cuerpo = await leerJSONAcotado(respuesta); }
        catch (error) {
          if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
          throw error instanceof ErrorClienteSaldoCronos ? error : fallo("respuesta_incompatible", respuesta.status);
        }
        if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
        return validarResultadoSaldoCronos(cuerpo, consulta);
      } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
    },
  });
}
