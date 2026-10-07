import { ErrorFichaConvocatoria, validarSelectorFicha, validarLecturaFicha } from "./contrato.js?v=20261001-s1-ficha-v1";

export const RUTA_FICHA_CONVOCATORIA = "/api/vec/seleccion/convocatorias/ficha";
const MAXIMO_BYTES = 4 * 1024 * 1024;
const fallo = (codigo, estado = 0) => new ErrorFichaConvocatoria(codigo, estado);

async function leerJSON(respuesta) {
  const declarado = respuesta.headers.get("content-length");
  if (declarado !== null && (!/^\d+$/u.test(declarado) || Number(declarado) > MAXIMO_BYTES)) throw fallo("respuesta_incompatible");
  if (!respuesta.body?.getReader) throw fallo("respuesta_incompatible");
  const lector = respuesta.body.getReader(), partes = []; let total = 0;
  try {
    while (true) {
      const { done, value } = await lector.read(); if (done) break;
      total += value.byteLength; if (total > MAXIMO_BYTES) throw fallo("respuesta_incompatible"); partes.push(value);
    }
  } catch (error) { await lector.cancel().catch(() => {}); throw error; }
  finally { lector.releaseLock(); }
  const bytes = new Uint8Array(total); let i = 0;
  for (const parte of partes) { bytes.set(parte, i); i += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw fallo("respuesta_incompatible"); }
}

/** Ruta fija interna. Actor y correlación nunca proceden de la web. */
export function crearLectorFichaHTTP({ fetchImpl = globalThis.fetch, plazoMs = 10000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30000) throw fallo("configuracion_invalida");
  return Object.freeze({
    async consultarExacta(entrada, { signal } = {}) {
      const selector = validarSelectorFicha(entrada);
      if (signal?.aborted) throw fallo("operacion_abortada");
      const controlador = new AbortController(), abortar = () => controlador.abort();
      signal?.addEventListener("abort", abortar, { once: true });
      const limite = setTimeout(abortar, plazoMs);
      try {
        const respuesta = await fetchImpl(RUTA_FICHA_CONVOCATORIA, {
          method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify(selector), mode: "same-origin", credentials: "same-origin", cache: "no-store", redirect: "error",
          referrerPolicy: "no-referrer", signal: controlador.signal,
        });
        if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible");
        if (!respuesta || respuesta.redirected) throw fallo("respuesta_incompatible");
        if (respuesta.status === 401 || respuesta.status === 403) throw fallo("acceso_denegado", respuesta.status);
        if (respuesta.status === 404) throw fallo("no_encontrada", 404);
        if (respuesta.status !== 200 || !respuesta.ok) throw fallo("servicio_no_disponible", respuesta.status);
        if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get("content-type") || "")) throw fallo("respuesta_incompatible");
        const cuerpo = await leerJSON(respuesta);
        if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible");
        return validarLecturaFicha(cuerpo, selector);
      } catch (error) {
        if (signal?.aborted) throw fallo("operacion_abortada");
        if (error instanceof ErrorFichaConvocatoria) throw error;
        throw fallo("servicio_no_disponible");
      } finally { clearTimeout(limite); signal?.removeEventListener("abort", abortar); }
    },
  });
}
