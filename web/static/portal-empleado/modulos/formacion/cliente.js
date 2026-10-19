import { validarEscenario } from './modelo.js?v=VEC_WEB_VERSION';

export function crearClienteEscenario({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async cargar({ signal } = {}) {
      const respuesta = await fetchImpl('./escenario.json?v=VEC_WEB_VERSION', {
        method: 'GET', credentials: 'omit', cache: 'no-store', redirect: 'error',
        referrerPolicy: 'no-referrer', headers: { Accept: 'application/json' }, signal,
      });
      if (!respuesta.ok) throw new Error('formacion.transporte');
      const bytes = new Uint8Array(await respuesta.arrayBuffer());
      if (bytes.length > 1024 * 1024) throw new Error('formacion.limite');
      const escenario = validarEscenario(JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(bytes)));
      return Object.freeze({ escenario, bytes });
    },
  });
}
