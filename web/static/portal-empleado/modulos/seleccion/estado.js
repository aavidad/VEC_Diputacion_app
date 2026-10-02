/** Una revisión de reglas invalida el resultado y cualquier respuesta todavía en vuelo. */
export function crearEstadoEnsayo({ cliente, publicar, validarResultado }) {
  let revision = 0; let controlador = null; let cerrado = false;
  function invalidar(estado = 'inicial') {
    revision += 1; controlador?.abort(); controlador = null;
    if (!cerrado) publicar({ estado, resultado: null });
  }
  return Object.freeze({
    invalidar,
    cancelar: () => invalidar('cancelado'),
    async simular(datos) {
      invalidar('cargando'); const actual = revision;
      const peticion = new AbortController(); controlador = peticion;
      try {
        const resultado = await cliente.simular(structuredClone(datos), { signal: peticion.signal });
        if (cerrado || actual !== revision || peticion.signal.aborted) return;
        validarResultado(resultado, datos);
        publicar({ estado: 'resultado', resultado });
      } catch (error) {
        if (cerrado || actual !== revision || peticion.signal.aborted) return;
        publicar({ estado: error?.codigo === 'denegado' ? 'denegado'
          : error?.codigo === 'validacion' ? 'validacion_servidor' : 'error', resultado: null });
      } finally { if (actual === revision) controlador = null; }
    },
    cerrar() { cerrado = true; invalidar(); },
  });
}
