import { registrarErrorCliente } from "./registro-errores.js?v=20261007-p7-http-v1";

const MAXIMO_BYTES = 256 * 1024;
const MAXIMO_MS = 15_000;
const MAXIMO_BYTES_ERROR = 8 * 1024;
const EN_VUELO = new WeakMap();

export class ErrorConsultaJSON extends Error {
  constructor(codigo, estado = 0) {
    super(codigo);
    this.name = "ErrorConsultaJSON";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function error(codigo, estado = 0) { return new ErrorConsultaJSON(codigo, estado); }

function rutaInterna(ruta) {
  if (typeof ruta !== "string" || ruta.length > 2048 || !/^\/api\/(vec|interna)\//u.test(ruta)
    || /[\\#\u0000-\u001f\u007f]/u.test(ruta)) throw new TypeError("ruta_interna_invalida");
  const path = ruta.split("?", 1)[0];
  if (path.includes("%") || path.includes("//") || path.split("/").some((parte) => parte === "." || parte === "..")) {
    throw new TypeError("ruta_interna_invalida");
  }
  return ruta;
}

function codigoEstado(estado) {
  if (estado === 401 || estado === 403) return "sin_permiso";
  if (estado === 404) return "no_encontrado";
  if (estado === 409) return "conflicto";
  return "no_disponible";
}

function cancelarCuerpo(respuesta) {
  try { void respuesta.body?.cancel?.()?.catch?.(() => {}); } catch { /* El estado HTTP ya determina el resultado. */ }
}

function esperar(ms, signal) {
  return new Promise((resolve, reject) => {
    if (signal.aborted) { reject(error("cancelado")); return; }
    const temporizador = setTimeout(terminar, ms);
    function abortar() { clearTimeout(temporizador); signal.removeEventListener("abort", abortar); reject(error("cancelado")); }
    function terminar() { signal.removeEventListener("abort", abortar); resolve(); }
    signal.addEventListener("abort", abortar, { once: true });
  });
}

function conCancelacion(promesa, signal) {
  if (signal.aborted) return Promise.reject(error("cancelado"));
  return new Promise((resolve, reject) => {
    const abortar = () => { signal.removeEventListener("abort", abortar); reject(error("cancelado")); };
    signal.addEventListener("abort", abortar, { once: true });
    promesa.then((valor) => { signal.removeEventListener("abort", abortar); resolve(valor); },
      (fallo) => { signal.removeEventListener("abort", abortar); reject(fallo); });
  });
}

async function leerJSON(respuesta, limiteBytes, signal) {
  const tipo = respuesta.headers?.get?.("content-type") ?? "";
  if (!/^application\/json(?:\s*;|\s*$)/iu.test(tipo)) throw error("respuesta_no_valida", respuesta.status);
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud != null && (!/^\d+$/u.test(longitud) || Number(longitud) > limiteBytes)) {
    throw error("respuesta_no_valida", respuesta.status);
  }
  if (!respuesta.body?.getReader) throw error("respuesta_no_valida", respuesta.status);
  const lector = respuesta.body.getReader();
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await conCancelacion(lector.read(), signal);
      if (done) break;
      total += value.byteLength;
      if (total > limiteBytes) throw error("respuesta_no_valida", respuesta.status);
      partes.push(value);
    }
  } finally {
    void lector.cancel().catch(() => {});
  }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw error("respuesta_no_valida", respuesta.status); }
}

// Código de negocio de un 409/422 ({"error":{"codigo":"…"}}), acotado; nunca se muestra tal cual.
async function codigoGobernado(respuesta, signal) {
  try {
    const codigo = (await leerJSON(respuesta, MAXIMO_BYTES_ERROR, signal))?.error?.codigo;
    return typeof codigo === "string" && /^[a-z_]{1,64}$/u.test(codigo) ? codigo : "";
  } catch { cancelarCuerpo(respuesta); return ""; }
}

async function ejecutar(ruta, metodo, cuerpoSerializado, fetchImpl, limiteBytes, plazoMs, reintentos, signal) {
  const controlador = new AbortController();
  const temporizador = setTimeout(() => controlador.abort(), plazoMs);
  const abortar = () => controlador.abort();
  signal?.addEventListener?.("abort", abortar, { once: true });
  if (signal?.aborted) controlador.abort();
  try {
    for (let intento = 0; ; intento++) {
      if (controlador.signal.aborted) throw error(signal?.aborted ? "cancelado" : "red");
      let respuesta;
      try {
        respuesta = await conCancelacion(Promise.resolve().then(() => fetchImpl(ruta, {
          method: metodo, mode: "same-origin", credentials: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
          headers: { Accept: "application/json", ...(cuerpoSerializado === undefined ? {} : { "Content-Type": "application/json" }) },
          ...(cuerpoSerializado === undefined ? {} : { body: cuerpoSerializado }),
        })), controlador.signal);
      } catch {
        if (controlador.signal.aborted) throw error(signal?.aborted ? "cancelado" : "red");
        if (metodo === "GET" && intento < reintentos) { await esperar(150 * (intento + 1), controlador.signal); continue; }
        throw error("red");
      }
      if (respuesta.redirected === true || (respuesta.url && new URL(respuesta.url, globalThis.location?.origin ?? "https://vec.invalid").origin !== (globalThis.location?.origin ?? "https://vec.invalid"))) {
        throw error("no_disponible", respuesta.status);
      }
      if (respuesta.status === 502 || respuesta.status === 503) {
        if (metodo === "GET" && intento < reintentos) { cancelarCuerpo(respuesta); await esperar(150 * (intento + 1), controlador.signal); continue; }
      }
      if (!respuesta.ok) {
        const fallo = error(codigoEstado(respuesta.status), respuesta.status);
        if (respuesta.status === 409 || respuesta.status === 422) fallo.codigoServidor = await codigoGobernado(respuesta, controlador.signal);
        else cancelarCuerpo(respuesta);
        throw fallo;
      }
      if (respuesta.status === 204) return null;
      return await leerJSON(respuesta, limiteBytes, controlador.signal);
    }
  } finally {
    clearTimeout(temporizador);
    signal?.removeEventListener?.("abort", abortar);
  }
}

function suscribir(entrada, signal, quitar) {
  if (!signal) { entrada.permanente = true; return entrada.promesa; }
  if (signal.aborted) return Promise.reject(error("cancelado"));
  entrada.activos++;
  return new Promise((resolve, reject) => {
    let terminado = false;
    const finalizar = () => {
      if (terminado) return false;
      terminado = true;
      signal.removeEventListener("abort", abortar);
      entrada.activos--;
      return true;
    };
    const abortar = () => {
      if (!finalizar()) return;
      if (!entrada.permanente && entrada.activos === 0) { quitar(); entrada.controlador.abort(); }
      reject(error("cancelado"));
    };
    signal.addEventListener("abort", abortar, { once: true });
    entrada.promesa.then((valor) => { if (finalizar()) resolve(valor); }, (fallo) => { if (finalizar()) reject(fallo); });
  });
}

/** Consulta JSON interna con límite de bytes, tiempo y reintentos solo de GET. */
export function consultarJSON(ruta, { metodo = "GET", cuerpo, signal, fetchImpl = globalThis.fetch,
  limiteBytes = MAXIMO_BYTES, plazoMs = MAXIMO_MS, reintentos = 2 } = {}) {
  rutaInterna(ruta);
  if (!["GET", "POST", "PUT", "PATCH", "DELETE"].includes(metodo) || (metodo === "GET" && cuerpo !== undefined)
    || typeof fetchImpl !== "function" || !Number.isSafeInteger(limiteBytes) || limiteBytes < 1 || limiteBytes > MAXIMO_BYTES
    || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > MAXIMO_MS
    || !Number.isSafeInteger(reintentos) || reintentos < 0 || reintentos > 2) throw new TypeError("opciones_http_invalidas");
  if (signal?.aborted) return Promise.reject(error("cancelado"));
  const cuerpoSerializado = cuerpo === undefined ? undefined : JSON.stringify(cuerpo);
  if (cuerpo !== undefined && typeof cuerpoSerializado !== "string") throw new TypeError("cuerpo_http_invalido");
  if (cuerpoSerializado !== undefined && new TextEncoder().encode(cuerpoSerializado).byteLength > MAXIMO_BYTES) throw new TypeError("cuerpo_http_demasiado_grande");
  if (metodo !== "GET") reintentos = 0;
  const clave = JSON.stringify([ruta, metodo, limiteBytes, plazoMs, reintentos]);
  let mapa = EN_VUELO.get(fetchImpl);
  if (!mapa) { mapa = new Map(); EN_VUELO.set(fetchImpl, mapa); }
  if (metodo === "GET" && mapa.has(clave)) return suscribir(mapa.get(clave), signal, () => mapa.delete(clave));
  const controlador = new AbortController();
  const entrada = { controlador, permanente: false, activos: 0, promesa: null };
  entrada.promesa = ejecutar(ruta, metodo, cuerpoSerializado, fetchImpl, limiteBytes, plazoMs, reintentos, controlador.signal)
    .catch((fallo) => {
      if (fallo instanceof ErrorConsultaJSON && !["cancelado", "sin_permiso", "no_encontrado", "conflicto"].includes(fallo.codigo)) registrarErrorCliente();
      throw fallo;
    });
  if (metodo === "GET") {
    mapa.set(clave, entrada);
    void entrada.promesa.then(() => { if (mapa.get(clave) === entrada) mapa.delete(clave); },
      () => { if (mapa.get(clave) === entrada) mapa.delete(clave); });
  }
  return suscribir(entrada, signal, () => mapa.delete(clave));
}
