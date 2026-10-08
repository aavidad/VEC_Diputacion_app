/** Montaje explícito de las secciones que publica el contexto vigente. */
export function crearMontajeSeccionesCentro({ documento, incorporaciones, cancelaciones }) {
  const contenedorI = documento.querySelector("#incorporaciones-centro");
  const contenedorC = documento.querySelector("#cancelaciones-centro");
  let retirarI = null;
  let retirarC = null;
  let bandeja = null;

  const ayuda = (seccion, disponible) => {
    for (const elemento of documento.querySelectorAll(`[data-i18n-ayuda-${seccion}]`)) {
      elemento.hidden = !disponible;
    }
  };
  function actualizar(capacidades) {
    const activaI = capacidades?.incorporaciones === true;
    const activaC = activaI && capacidades?.cancelaciones === true;
    ayuda("incorporacion", activaI);
    ayuda("cancelacion", activaC);
    if (!activaC && retirarC) { retirarC(); retirarC = null; }
    if (!activaI && retirarI) { retirarI(); retirarI = null; bandeja = null; }
    if (activaI && !retirarI) {
      bandeja = incorporaciones.crearClienteIncorporacionesCentro();
      incorporaciones.instalarAyudaIncorporacionesCentro(documento);
      retirarI = incorporaciones.montarIncorporacionesCentro({ contenedor: contenedorI, cliente: bandeja });
    }
    if (activaC && !retirarC) {
      cancelaciones.instalarAyudaCancelacionesCentro(documento);
      retirarC = cancelaciones.montarCancelacionesCentro({ contenedor: contenedorC, bandeja });
    }
  }
  actualizar();
  return Object.freeze({ actualizar, desmontar: () => actualizar() });
}
