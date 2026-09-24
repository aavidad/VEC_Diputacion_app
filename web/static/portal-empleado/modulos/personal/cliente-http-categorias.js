import { RUTA_CATEGORIAS_PROFESIONALES, validarConsultaCategorias, validarPaginaCategorias } from "./contrato.js?v=20260920-personal-catalogo-v1";

const MAXIMO_RESPUESTA_BYTES = 256 * 1024;
const MAXIMO_FRAGMENTOS = 256;
const PLAZO_POR_DEFECTO_MS = 10_000;

export class ErrorClienteCategoriasPersonal extends Error {
  constructor(codigo, estado = 0) { super(`cliente de categorías profesionales: ${codigo}`); this.name = "ErrorClienteCategoriasPersonal"; this.codigo = codigo; this.estado = estado; Object.freeze(this); }
}
function registro(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor) && Object.getPrototypeOf(valor) === Object.prototype; }
function error(codigo, estado) { return new ErrorClienteCategoriasPersonal(codigo, estado); }
function validarSignal(signal) { if (signal === undefined) return undefined; if (!signal || typeof signal !== "object" || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function" || typeof signal.removeEventListener !== "function") throw error("signal_no_valida"); if (signal.aborted) throw error("operacion_abortada"); return signal; }
function validarPlazo(plazoMs) { if (!Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("plazo de categorías profesionales no válido"); return plazoMs; }

async function ejecutarConPlazo(ejecutor, externo, plazoMs) {
  const controlador = new AbortController(); let temporizador; let terminar; let rechazarPendiente; let motivo = "";
  const abortarExterno = () => { motivo = "abortada"; controlador.abort(); rechazarPendiente?.(error("operacion_abortada")); };
  if (externo?.aborted) throw error("operacion_abortada");
  externo?.addEventListener("abort", abortarExterno, { once: true });
  try {
    return await new Promise((resolver, rechazar) => {
      let cerrada = false;
      terminar = (resultado, valor) => { if (cerrada) return; cerrada = true; clearTimeout(temporizador); externo?.removeEventListener("abort", abortarExterno); resultado(valor); };
      rechazarPendiente = (valor) => terminar(rechazar, valor);
      temporizador = setTimeout(() => { motivo = "plazo"; controlador.abort(); rechazarPendiente(error("plazo_agotado")); }, plazoMs);
      Promise.resolve().then(() => ejecutor(controlador.signal)).then(
        (valor) => terminar(resolver, valor),
        (causa) => terminar(rechazar, motivo === "abortada" ? error("operacion_abortada") : motivo === "plazo" ? error("plazo_agotado") : causa instanceof ErrorClienteCategoriasPersonal ? causa : error("red_no_disponible")),
      );
    });
  } finally { clearTimeout(temporizador); externo?.removeEventListener("abort", abortarExterno); }
}

async function cancelarRespuesta(respuesta) {
  try { await respuesta?.body?.cancel?.("respuesta descartada"); } catch {}
}

async function leerJSONAcotado(respuesta, signal) {
  const declarada = respuesta.headers?.get?.("content-length");
  if (declarada !== null && declarada !== undefined && (!/^(?:0|[1-9][0-9]*)$/u.test(declarada) || Number(declarada) > MAXIMO_RESPUESTA_BYTES)) { await cancelarRespuesta(respuesta); throw error("respuesta_excesiva", respuesta.status); }
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") { await cancelarRespuesta(respuesta); throw error("respuesta_no_incremental", respuesta.status); }
  const lector = respuesta.body.getReader(); if (!lector || typeof lector.read !== "function" || typeof lector.cancel !== "function") { await cancelarRespuesta(respuesta); throw error("respuesta_no_incremental", respuesta.status); }
  const fragmentos = []; let total = 0; const cancelar = () => { Promise.resolve(lector.cancel("consulta cancelada")).catch(() => {}); };
  signal.addEventListener("abort", cancelar, { once: true });
  try {
    while (true) {
      if (signal.aborted) throw error("operacion_abortada");
      const lectura = await lector.read();
      if (!lectura || typeof lectura.done !== "boolean" || !lectura.done && (!(lectura.value instanceof Uint8Array) || lectura.value.byteLength === 0)) throw error("respuesta_incompatible", respuesta.status);
      if (lectura.done) break;
      total += lectura.value.byteLength; if (total > MAXIMO_RESPUESTA_BYTES || fragmentos.length >= MAXIMO_FRAGMENTOS) throw error("respuesta_excesiva", respuesta.status); fragmentos.push(lectura.value);
    }
  } catch (causa) { await Promise.resolve(lector.cancel("respuesta descartada")).catch(() => {}); throw causa; } finally { signal.removeEventListener("abort", cancelar); try { lector.releaseLock?.(); } catch {} }
  const bytes = new Uint8Array(total); let posicion = 0; fragmentos.forEach((fragmento) => { bytes.set(fragmento, posicion); posicion += fragmento.byteLength; });
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); } catch { throw error("json_no_valido", respuesta.status); }
}

export function crearClienteHTTPCategoriasPersonal({ fetchImpl = globalThis.fetch, plazoMs = PLAZO_POR_DEFECTO_MS } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch de categorías profesionales no disponible"); validarPlazo(plazoMs);
  return Object.freeze({
    async listarCategorias(entrada, opciones = {}) {
      const consulta = validarConsultaCategorias(entrada);
      if (!registro(opciones) || Object.keys(opciones).some((clave) => clave !== "signal")) throw error("opciones_no_validas");
      const externo = validarSignal(opciones.signal); const parametros = new URLSearchParams({ q: consulta.q, area: consulta.area, limit: String(consulta.limit), offset: String(consulta.offset) });
      const respuesta = await ejecutarConPlazo(async (signal) => {
        let resultado;
        try { resultado = await fetchImpl(`${RUTA_CATEGORIAS_PROFESIONALES}?${parametros.toString()}`, { method: "GET", credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); } catch { throw error("red_no_disponible"); }
        if (!resultado || resultado.redirected === true || resultado.status !== 200 || resultado.ok !== true) { await cancelarRespuesta(resultado); throw error("estado_no_valido", resultado?.status || 0); }
        const tipo = resultado.headers?.get?.("content-type"); if (typeof tipo !== "string" || !/^application\/json(?:\s*;\s*charset\s*=\s*utf-8)?$/iu.test(tipo)) { await cancelarRespuesta(resultado); throw error("tipo_respuesta_no_valido", resultado.status); }
        return leerJSONAcotado(resultado, signal);
      }, externo, plazoMs);
      try {
        if (!registro(respuesta) || Object.keys(respuesta).length !== 1 || !registro(respuesta.data) || Object.keys(respuesta.data).length !== 1 || !Object.hasOwn(respuesta.data, "categories")) throw new TypeError("sobre");
        return validarPaginaCategorias(respuesta.data.categories, consulta);
      } catch (causa) { if (causa instanceof ErrorClienteCategoriasPersonal) throw causa; throw error("sobre_no_valido", 200); }
    },
  });
}
