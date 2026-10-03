const RUTA = '/api/seleccion/v1/';

/** Transporte del ensayo local: referencias, reglas y notas del ejemplo, sin credenciales. */
export function crearClienteSeleccion({ fetchImpl = globalThis.fetch } = {}) {
  async function enviar(recurso, { signal, body } = {}) {
    const respuesta = await fetchImpl(`${RUTA}${recurso}`, {
      method: body ? 'POST' : 'GET', credentials: 'omit', cache: 'no-store',
      redirect: 'error', referrerPolicy: 'no-referrer', signal,
      headers: { Accept: 'application/json', ...(body ? { 'Content-Type': 'application/json' } : {}) },
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
    if (!respuesta?.ok) {
      const error = new Error('seleccion.respuesta');
      error.codigo = [400, 422].includes(respuesta?.status) ? 'validacion'
        : [401, 403].includes(respuesta?.status) ? 'denegado' : 'error';
      throw error;
    }
    if (Number(respuesta.headers?.get?.('Content-Length')) > 1024 * 1024) throw new Error('seleccion.limite');
    const contenido = await respuesta.text();
    if (contenido.length > 1024 * 1024) throw new Error('seleccion.limite');
    return JSON.parse(contenido);
  }
  return Object.freeze({
    listar: ({ signal } = {}) => enviar('ensayos', { signal }),
    simular: ({ ejemplo_ref, configuracion, notas_prueba }, { signal } = {}) => enviar('simulaciones', {
      signal, body: { ejemplo_ref, configuracion,
        ...(notas_prueba ? { notas_prueba: notas_prueba.map(({ solicitud_ref, fase_ref, puntos_micropuntos }) => ({ solicitud_ref, fase_ref, puntos_micropuntos })) } : {}),
      },
    }),
  });
}
