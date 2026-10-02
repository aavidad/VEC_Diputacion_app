/** Transporte exclusivo del servidor local de ensayos, sin identidad ni efectos. */
export function crearClienteEnsayosLocal({ fetchImpl = globalThis.fetch } = {}) {
  async function enviar(ruta, opciones = {}) {
    const respuesta = await fetchImpl(ruta, { credentials: 'omit', redirect: 'error', referrerPolicy: 'no-referrer', cache: 'no-store', ...opciones, headers: { Accept: 'application/json', ...(opciones.body ? { 'Content-Type': 'application/json' } : {}) } });
    if (!respuesta.ok) { const error = new Error('provision.ensayo'); error.codigo = [400, 422].includes(respuesta.status) ? 'validacion' : [401, 403].includes(respuesta.status) ? 'denegado' : respuesta.status === 409 ? 'conflicto' : 'error'; throw error; }
    const bytes = await respuesta.text(); if (bytes.length > 1024 * 1024) throw new Error('provision.limite'); return JSON.parse(bytes);
  }
  return Object.freeze({
    listarCiclos: ({ signal } = {}) => enviar('/api/provision/v1/ciclos-locales', { signal }),
    simularCiclo: ({ ejemplo_ref, caso_ref }, { signal } = {}) => enviar('/api/provision/v1/ciclos-locales/simulaciones', { method: 'POST', signal, body: JSON.stringify({ ejemplo_ref, caso_ref }) }),
    listarAdjudicaciones: ({ signal } = {}) => enviar('/api/provision/v1/adjudicaciones-locales', { signal }),
    simularAdjudicacion: ({ ejemplo_ref, configuracion }, { signal } = {}) => enviar('/api/provision/v1/adjudicaciones-locales/simulaciones', { method: 'POST', signal, body: JSON.stringify({ ejemplo_ref, configuracion }) }),
  });
}
