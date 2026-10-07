import { referenciaConsultaValida, validarResultadoConsulta } from "./consulta-contrato.js";

export const RUTA_CONSULTA_MERITO_PROPIO = "/api/meritos/hecho-propio/consulta";
const MAX_BYTES = 64 * 1024;
const MAX_FRAGMENTOS = 1024;

export class ErrorConsultaMerito extends Error {
  constructor(codigo) {
    super(`meritos.consulta.${codigo}`);
    this.name = "ErrorConsultaMerito";
    this.codigo = codigo;
    Object.freeze(this);
  }
}
function fallo(codigo) { return new ErrorConsultaMerito(codigo); }
function opcionesValidas(entrada) {
  return entrada !== null && typeof entrada === "object" && !Array.isArray(entrada)
    && Object.keys(entrada).every((clave) => ["hechoRef", "signal"].includes(clave))
    && referenciaConsultaValida(entrada.hechoRef)
    && (entrada.signal === undefined || (typeof entrada.signal?.aborted === "boolean"
      && typeof entrada.signal.addEventListener === "function" && typeof entrada.signal.removeEventListener === "function"));
}

/** La lectura por fragmentos limita bytes y no conserva cuerpos de error. */
async function leerJSON(respuesta, signal) {
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw fallo("respuesta_invalida");
  const decoder = new TextDecoder("utf-8", { fatal: true });
  let bytes = 0; let fragmentos = 0; let texto = "";
  const abortar = () => { Promise.resolve(lector.cancel()).catch(() => {}); };
  signal.addEventListener("abort", abortar, { once: true });
  try {
    while (true) {
      if (signal.aborted) throw fallo("abortada");
      const { done, value } = await lector.read();
      if (signal.aborted) throw fallo("abortada");
      if (done) break;
      if (!(value instanceof Uint8Array) || ++fragmentos > MAX_FRAGMENTOS || (bytes += value.byteLength) > MAX_BYTES) throw fallo("respuesta_invalida");
      texto += decoder.decode(value, { stream: true });
    }
    texto += decoder.decode();
    return JSON.parse(texto);
  } catch (causa) {
    try { await lector.cancel(); } catch {}
    if (signal.aborted) throw fallo("abortada");
    if (causa instanceof ErrorConsultaMerito) throw causa;
    throw fallo("respuesta_invalida");
  } finally {
    signal.removeEventListener("abort", abortar);
    lector.releaseLock?.();
  }
}

/** El servidor resuelve identidad y permiso; el cliente solo envía un hecho conocido. */
export function crearLectorConsultaMeritoPropio(opciones = {}) {
  if (!opciones || typeof opciones !== "object" || Array.isArray(opciones)
    || Object.keys(opciones).some((clave) => !["fetchImpl", "plazoMs"].includes(clave))) throw new TypeError("meritos.consulta.lector_invalido");
  const { fetchImpl = globalThis.fetch, plazoMs = 10_000 } = opciones;
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("meritos.consulta.lector_invalido");
  return Object.freeze({
    async consultar(entrada) {
      if (!opcionesValidas(entrada)) throw fallo("peticion_invalida");
      const { hechoRef, signal } = entrada;
      if (signal?.aborted) throw fallo("abortada");
      const controlador = new AbortController();
      const abortar = () => controlador.abort();
      signal?.addEventListener("abort", abortar, { once: true });
      const timer = setTimeout(abortar, plazoMs);
      try {
        const respuesta = await fetchImpl(RUTA_CONSULTA_MERITO_PROPIO, {
          method: "POST", mode: "same-origin", credentials: "omit", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
          headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify({ hecho_ref: hechoRef }), signal: controlador.signal,
        });
        if (controlador.signal.aborted) throw fallo(signal?.aborted ? "abortada" : "error");
        if ([401, 403].includes(respuesta?.status)) {
          try { await respuesta.body?.cancel?.(); } catch {}
          throw fallo("denegada");
        }
        if (respuesta?.status !== 200 || respuesta.ok !== true || respuesta.redirected === true) {
          try { await respuesta?.body?.cancel?.(); } catch {}
          throw fallo("error");
        }
        const tipo = respuesta.headers?.get?.("content-type") ?? "";
        const longitud = respuesta.headers?.get?.("content-length");
        if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(tipo)
          || (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAX_BYTES))) {
          try { await respuesta.body?.cancel?.(); } catch {}
          throw fallo("respuesta_invalida");
        }
        return validarResultadoConsulta(await leerJSON(respuesta, controlador.signal), hechoRef);
      } catch (causa) {
        if (signal?.aborted) throw fallo("abortada");
        if (causa instanceof ErrorConsultaMerito) throw causa;
        throw fallo("error");
      } finally {
        clearTimeout(timer);
        signal?.removeEventListener("abort", abortar);
      }
    },
  });
}
