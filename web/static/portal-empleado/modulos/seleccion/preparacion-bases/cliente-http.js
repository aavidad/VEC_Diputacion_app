import { ErrorConsultaBases, validarSelector, leerConsulta } from './contrato-http.js?v=20261003-s2-consulta-v2';
import { MAXIMO_BYTES } from './modelo.js?v=20261003-s2-consulta-v1';

export const RUTA_CONSULTA_BASES = '/api/vec/seleccion/preparacion-bases/consultar';
const fallo = codigo => new ErrorConsultaBases(codigo);

async function leerBytes(respuesta) {
  const n = respuesta.headers?.get('content-length');
  if (n !== null && (!/^\d+$/u.test(n) || Number(n) > MAXIMO_BYTES)) throw fallo('respuesta_incompatible');
  if (!respuesta.body?.getReader) throw fallo('respuesta_incompatible');
  const lector = respuesta.body.getReader(), partes = []; let total = 0;
  try {
    while (true) {
      const { done, value } = await lector.read(); if (done) break;
      total += value.byteLength; if (total > MAXIMO_BYTES) throw fallo('respuesta_incompatible'); partes.push(value);
    }
  } catch (error) { await lector.cancel().catch(() => {}); throw error; }
  finally { lector.releaseLock(); }
  const bytes = new Uint8Array(total); let i = 0;
  for (const parte of partes) { bytes.set(parte, i); i += parte.byteLength; }
  return bytes;
}

/** Transporte nominal interno existente: la sesión y el ámbito los resuelve el servidor. */
export function crearLectorBasesHTTP({ fetchImpl = globalThis.fetch, plazoMs = 25000 } = {}) {
  if (typeof fetchImpl !== 'function' || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30000) throw fallo('configuracion_invalida');
  return Object.freeze({ async consultar(entrada, { signal } = {}) {
    const selector = validarSelector(entrada);
    if (signal?.aborted) throw fallo('operacion_abortada');
    const controlador = new AbortController(), abortar = () => controlador.abort();
    signal?.addEventListener('abort', abortar, { once: true }); const limite = setTimeout(abortar, plazoMs);
    try {
      const respuesta = await fetchImpl(RUTA_CONSULTA_BASES, {
        method: 'POST', headers: { Accept: 'application/json', 'Content-Type': 'application/json' }, body: JSON.stringify(selector),
        mode: 'same-origin', credentials: 'same-origin', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer', signal: controlador.signal,
      });
      if (controlador.signal.aborted) throw fallo('operacion_abortada');
      if (!respuesta || respuesta.redirected) throw fallo('respuesta_incompatible');
      if ([401, 403].includes(respuesta.status)) throw fallo('acceso_denegado');
      if (respuesta.status === 404) throw fallo('no_encontrada');
      if (respuesta.status === 409) throw fallo('version_en_conflicto');
      if (respuesta.status !== 200 || !respuesta.ok) throw fallo('servicio_no_disponible');
      if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get('content-type') || '')) throw fallo('respuesta_incompatible');
      const bytes = await leerBytes(respuesta);
      if (controlador.signal.aborted) throw fallo('operacion_abortada');
      return { bytes, dto: leerConsulta(bytes, selector) };
    } catch (error) {
      if (signal?.aborted) throw fallo('operacion_abortada');
      if (controlador.signal.aborted) throw fallo('servicio_no_disponible');
      if (error instanceof ErrorConsultaBases) throw error;
      throw fallo('servicio_no_disponible');
    } finally { clearTimeout(limite); signal?.removeEventListener('abort', abortar); }
  } });
}
