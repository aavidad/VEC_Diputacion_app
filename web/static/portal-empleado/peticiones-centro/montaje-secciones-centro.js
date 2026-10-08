/** Montaje explícito de las secciones que publica el contexto vigente. */
export function crearMontajeSeccionesCentro({ documento, incorporaciones, cancelaciones }) {
  const contenedorI = documento.querySelector("#incorporaciones-centro");
  const contenedorC = documento.querySelector("#cancelaciones-centro");
  let retirarI = null;
  let retirarC = null;
  let bandeja = null;
  let contextoAnterior;
  let actualizando = false;

  const ayuda = (seccion, disponible) => {
    for (const elemento of documento.querySelectorAll(`[data-i18n-ayuda-${seccion}]`)) {
      elemento.hidden = !disponible;
    }
  };
  function actualizar(contexto) {
    if (actualizando) return;
    actualizando = true;
    try {
      const nuevaLectura = contexto !== contextoAnterior;
      contextoAnterior = contexto;
      const activaI = contexto?.capacidades?.incorporaciones === true;
      const activaC = activaI && contexto?.capacidades?.cancelaciones === true;
      ayuda("incorporacion", activaI);
      ayuda("cancelacion", activaC);
      if ((nuevaLectura || !activaC) && retirarC) {
        const retirar = retirarC; retirarC = null; retirar();
      }
      if ((nuevaLectura || !activaI) && retirarI) {
        const retirar = retirarI; retirarI = null; bandeja = null; retirar();
      }
      if (activaI && !retirarI) {
        bandeja = incorporaciones.crearClienteIncorporacionesCentro();
        incorporaciones.instalarAyudaIncorporacionesCentro(documento);
        retirarI = incorporaciones.montarIncorporacionesCentro({ contenedor: contenedorI, cliente: bandeja });
      }
      if (activaC && !retirarC) {
        cancelaciones.instalarAyudaCancelacionesCentro(documento);
        retirarC = cancelaciones.montarCancelacionesCentro({ contenedor: contenedorC, bandeja });
      }
    } finally { actualizando = false; }
  }
  actualizar();
  return Object.freeze({ actualizar, desmontar: () => actualizar() });
}
