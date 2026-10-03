/** Carga local revocable: nunca conserva el resultado de una lectura retirada. */
export function crearCargaLocal({ leer, retirar, mostrar, anunciar }) {
  let turno = 0; let activa = true;
  const cerrar = () => { ++turno; retirar(); };
  return Object.freeze({
    cerrar,
    async abrir(archivo) {
      if (!activa) return;
      cerrar(); const actual = turno;
      if (!archivo) { anunciar('vacio'); return; }
      anunciar('cargando');
      try {
        const resultado = await leer(archivo);
        if (!activa || actual !== turno) return;
        mostrar(resultado); anunciar('cargado');
      } catch (error) {
        if (!activa || actual !== turno) return;
        cerrar(); anunciar(error.message === 'tamano' ? 'error_tamano' : 'error_formato', true);
      }
    },
    desmontar() { activa = false; cerrar(); },
  });
}
