/** Adaptador del servidor LOCAL de ensayos. No autentica ni presenta solicitudes. */
export function crearClienteProvisionLocal({ fetchImpl = globalThis.fetch } = {}) {
  async function enviar(ruta, opciones) {
    const respuesta = await fetchImpl(ruta, { credentials: 'omit', redirect: 'error', referrerPolicy: 'no-referrer', cache: 'no-store', ...opciones, headers: { Accept: 'application/json', ...(opciones?.body ? { 'Content-Type': 'application/json' } : {}) } });
    if (!respuesta.ok) { const e = new Error('provision.transporte'); e.codigo = [401, 403].includes(respuesta.status) ? 'denegado' : respuesta.status === 409 ? 'conflicto' : [400, 422].includes(respuesta.status) ? 'validacion' : 'error'; throw e; }
    const bytes = await respuesta.text(); if (bytes.length > 1024 * 1024) throw new Error('provision.limite'); return JSON.parse(bytes);
  }
  return Object.freeze({
    listar: ({ signal } = {}) => enviar('/api/provision/v1/procesos-locales', { signal }),
    simular: (peticion, { signal } = {}) => enviar('/api/provision/v1/procesos-locales/simulaciones', { method: 'POST', signal, body: JSON.stringify({ ejemplo_ref: peticion.ejemplo_ref, configuracion: peticion.configuracion, preferencias: peticion.preferencias }) }),
  });
}
